package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/labstack/echo/v4"
	"github.com/nats-io/nats.go"
	"go.uber.org/zap"

	"github.com/fyrsmithlabs/contextd/pkg/auth"
	"github.com/fyrsmithlabs/contextd/pkg/checkpoint"
	"github.com/fyrsmithlabs/contextd/pkg/config"
	"github.com/fyrsmithlabs/contextd/pkg/prefetch"
	"github.com/fyrsmithlabs/contextd/pkg/remediation"
	"github.com/fyrsmithlabs/contextd/pkg/vectorstore"
)

// VectorStoreInterface defines the collection management methods needed by MCP handlers.
//
// This interface is implemented by vectorstore.Service and allows for testing
// with mock implementations.
type VectorStoreInterface interface {
	CreateCollection(ctx context.Context, collectionName string, vectorSize int) error
	DeleteCollection(ctx context.Context, collectionName string) error
	ListCollections(ctx context.Context) ([]string, error)
	GetCollectionInfo(ctx context.Context, collectionName string) (*vectorstore.CollectionInfo, error)
}

// Server implements MCP protocol over HTTP with Echo router.
//
// The server provides:
//   - 9 MCP tool endpoints (checkpoint, remediation, skill, index, status)
//   - SSE streaming for long-running operations
//   - NATS-based operation tracking
//   - JSON-RPC 2.0 protocol compliance
//   - Pre-fetch engine integration (optional)
//
// Example usage:
//
//	mcpServer := mcp.NewServer(echo, operations, natsConn)
//	mcpServer.RegisterRoutes()
type Server struct {
	echo       *echo.Echo
	operations *OperationRegistry
	nats       *nats.Conn

	// Services
	checkpointService  *checkpoint.Service
	remediationService *remediation.Service
	vectorStore        VectorStoreInterface
	logger             *zap.Logger

	// Pre-fetch support (optional)
	prefetchEnabled   bool
	prefetchCache     *prefetch.Cache
	prefetchExecutor  *prefetch.Executor
	prefetchDetectors map[string]*prefetch.Detector // projectPath -> detector
	prefetchMu        sync.RWMutex
	prefetchLogger    *zap.Logger
}

// NewServer creates a new MCP server with Echo router and NATS connection.
//
// The server registers MCP endpoints under /mcp/* and SSE streaming
// under /mcp/sse/:operation_id.
//
// Services can be nil if not needed (handlers will return appropriate errors).
func NewServer(
	e *echo.Echo,
	operations *OperationRegistry,
	nc *nats.Conn,
	checkpointSvc *checkpoint.Service,
	remediationSvc *remediation.Service,
	vectorStoreSvc VectorStoreInterface,
	logger *zap.Logger,
) *Server {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Server{
		echo:               e,
		operations:         operations,
		nats:               nc,
		checkpointService:  checkpointSvc,
		remediationService: remediationSvc,
		vectorStore:        vectorStoreSvc,
		logger:             logger,
		prefetchLogger:     logger,
	}
}

// RegisterRoutes registers all MCP tool endpoints with authentication middleware.
//
// This method should be called after server creation to set up routing.
//
// Authentication:
//   - ALL /mcp/* endpoints require authentication (owner-based auth from system username)
//   - Public endpoints (/health, /metrics) do NOT have auth middleware
//
// Registered protected endpoints:
//   - POST /mcp/checkpoint/save
//   - POST /mcp/checkpoint/search
//   - POST /mcp/checkpoint/list
//   - POST /mcp/remediation/save
//   - POST /mcp/remediation/search
//   - POST /mcp/skill/save
//   - POST /mcp/skill/search
//   - POST /mcp/collection/create
//   - POST /mcp/collection/delete
//   - POST /mcp/collection/list
//   - POST /mcp/index/repository
//   - POST /mcp/status
//   - GET  /mcp/sse/:operation_id
//   - GET  /mcp/tools/list
//   - GET  /mcp/resources/list
//   - POST /mcp/resources/read
func (s *Server) RegisterRoutes() {
	// Create MCP group with authentication middleware
	mcp := s.echo.Group("/mcp", auth.OwnerAuthMiddleware())

	// Checkpoint endpoints (authenticated)
	mcp.POST("/checkpoint/save", s.handleCheckpointSave)
	mcp.POST("/checkpoint/search", s.handleCheckpointSearch)
	mcp.POST("/checkpoint/list", s.handleCheckpointList)

	// Remediation endpoints (authenticated)
	mcp.POST("/remediation/save", s.handleRemediationSave)
	mcp.POST("/remediation/search", s.handleRemediationSearch)

	// Skill endpoints (authenticated)
	mcp.POST("/skill/save", s.handleSkillSave)
	mcp.POST("/skill/search", s.handleSkillSearch)

	// Collection endpoints (authenticated)
	mcp.POST("/collection/create", s.handleCollectionCreate)
	mcp.POST("/collection/delete", s.handleCollectionDelete)
	mcp.POST("/collection/list", s.handleCollectionList)

	// Index endpoint (authenticated)
	mcp.POST("/index/repository", s.handleIndexRepository)

	// Status endpoint (authenticated)
	mcp.POST("/status", s.handleStatus)

	// SSE streaming endpoint (authenticated)
	mcp.GET("/sse/:operation_id", func(c echo.Context) error {
		return HandleSSE(c, s.operations, s.nats)
	})

	// MCP protocol discovery endpoints (authenticated)
	mcp.GET("/tools/list", s.handleToolsList)
	mcp.GET("/resources/list", s.handleResourcesList)
	mcp.POST("/resources/read", s.handleResourceRead)
}

// handleCheckpointSave handles POST /mcp/checkpoint/save.
//
// This is a long-running operation that:
//  1. Validates request parameters
//  2. Creates NATS operation
//  3. Starts async worker to save checkpoint
//  4. Returns operation_id immediately
//
// The client can monitor progress via SSE streaming.
func (s *Server) handleCheckpointSave(c echo.Context) error {
	// Parse JSON-RPC request
	var req JSONRPCRequest
	if err := c.Bind(&req); err != nil {
		return JSONRPCErrorWithContext(c, "", ParseError, err)
	}

	// Parse tool-specific params
	var params struct {
		Content     string            `json:"content"`
		ProjectPath string            `json:"project_path"`
		Metadata    map[string]string `json:"metadata"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return JSONRPCErrorWithContext(c, req.ID, InvalidParams, err)
	}

	// Validate params
	if params.Content == "" {
		return JSONRPCErrorWithContext(c, req.ID, InvalidParams, fmt.Errorf("content is required"))
	}
	if params.ProjectPath == "" {
		return JSONRPCErrorWithContext(c, req.ID, InvalidParams, fmt.Errorf("project_path is required"))
	}

	// Extract authenticated owner ID
	ownerID, err := ExtractOwnerID(c)
	if err != nil {
		return JSONRPCErrorWithContext(c, req.ID, AuthError, err)
	}

	// Create operation
	ctx := context.WithValue(c.Request().Context(), ownerIDKey, ownerID)
	ctx = context.WithValue(ctx, traceIDKey, c.Response().Header().Get("X-Request-ID"))
	opID := s.operations.Create(ctx, "checkpoint_save", params)

	// Start async worker (in production, this would call actual checkpoint service)
	go s.doCheckpointSave(ctx, opID, params)

	// Return operation_id immediately
	return JSONRPCSuccess(c, req.ID, map[string]string{
		"operation_id": opID,
		"status":       "pending",
	})
}

// doCheckpointSave performs the actual checkpoint save operation.
func (s *Server) doCheckpointSave(ctx context.Context, opID string, params struct {
	Content     string            `json:"content"`
	ProjectPath string            `json:"project_path"`
	Metadata    map[string]string `json:"metadata"`
}) {
	if err := s.operations.Started(opID); err != nil {
		_ = s.operations.Error(opID, InternalError, fmt.Errorf("failed to start: %w", err))
		return
	}

	// Check service availability
	if s.checkpointService == nil {
		_ = s.operations.Error(opID, InternalError, fmt.Errorf("checkpoint service not available"))
		return
	}

	// Create checkpoint from params
	cp := &checkpoint.Checkpoint{
		ProjectPath: params.ProjectPath,
		Summary:     extractSummary(params.Metadata),
		Content:     params.Content,
		Tags:        extractTags(params.Metadata),
		Metadata:    convertMetadata(params.Metadata),
	}

	// Save via service
	if err := s.checkpointService.Save(ctx, cp); err != nil {
		_ = s.operations.Error(opID, InternalError, err)
		return
	}

	// Complete
	_ = s.operations.Complete(opID, map[string]interface{}{
		"checkpoint_id": cp.ID,
	})
}

// extractSummary gets summary from metadata or uses default.
func extractSummary(metadata map[string]string) string {
	if summary, ok := metadata["summary"]; ok && summary != "" {
		return summary
	}
	return "Checkpoint saved"
}

// extractTags extracts tags from metadata.
func extractTags(metadata map[string]string) []string {
	if tagsStr, ok := metadata["tags"]; ok && tagsStr != "" {
		return strings.Split(tagsStr, ",")
	}
	return nil
}

// convertMetadata converts map[string]string to map[string]interface{}.
func convertMetadata(metadata map[string]string) map[string]interface{} {
	result := make(map[string]interface{}, len(metadata))
	for k, v := range metadata {
		result[k] = v
	}
	return result
}

// handleCheckpointSearch searches for checkpoints with optional prefetch injection.
func (s *Server) handleCheckpointSearch(c echo.Context) error {
	var req JSONRPCRequest
	if err := c.Bind(&req); err != nil {
		return JSONRPCErrorWithContext(c, "", ParseError, err)
	}

	var params struct {
		ProjectPath string `json:"project_path"`
		Query       string `json:"query"`
		Limit       int    `json:"limit"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return JSONRPCErrorWithContext(c, req.ID, InvalidParams, err)
	}

	// Check service availability
	if s.checkpointService == nil {
		return JSONRPCErrorWithContext(c, req.ID, InternalError, fmt.Errorf("checkpoint service not available"))
	}

	if params.Limit == 0 {
		params.Limit = 10
	}

	// Get prefetch results FIRST (if enabled)
	var prefetchData []prefetch.PreFetchResult
	if s.prefetchEnabled && params.ProjectPath != "" {
		prefetchData = s.GetPrefetchResults(params.ProjectPath)
	}

	// Execute search via service
	ctx := c.Request().Context()
	results, err := s.checkpointService.Search(ctx, params.Query, &checkpoint.SearchOptions{
		ProjectPath: params.ProjectPath,
		Limit:       params.Limit,
	})
	if err != nil {
		return JSONRPCErrorWithContext(c, req.ID, InternalError, err)
	}

	// Build response
	response := map[string]interface{}{
		"checkpoints": results,
	}
	if len(prefetchData) > 0 {
		response["prefetch"] = prefetchData
	}

	return JSONRPCSuccess(c, req.ID, response)
}

// handleCheckpointList lists recent checkpoints.
func (s *Server) handleCheckpointList(c echo.Context) error {
	var req JSONRPCRequest
	if err := c.Bind(&req); err != nil {
		return JSONRPCErrorWithContext(c, "", ParseError, err)
	}

	var params struct {
		ProjectPath string `json:"project_path"`
		Limit       int    `json:"limit"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return JSONRPCErrorWithContext(c, req.ID, InvalidParams, err)
	}

	// Check service availability
	if s.checkpointService == nil {
		return JSONRPCErrorWithContext(c, req.ID, InternalError, fmt.Errorf("checkpoint service not available"))
	}

	if params.Limit == 0 {
		params.Limit = 20
	}

	ctx := c.Request().Context()
	results, err := s.checkpointService.List(ctx, &checkpoint.ListOptions{
		ProjectPath: params.ProjectPath,
		Limit:       params.Limit,
	})
	if err != nil {
		return JSONRPCErrorWithContext(c, req.ID, InternalError, err)
	}

	return JSONRPCSuccess(c, req.ID, map[string]interface{}{
		"checkpoints": results,
	})
}

// handleRemediationSave saves a new remediation.
func (s *Server) handleRemediationSave(c echo.Context) error {
	var req JSONRPCRequest
	if err := c.Bind(&req); err != nil {
		return JSONRPCErrorWithContext(c, "", ParseError, err)
	}

	var params struct {
		ProjectPath string                 `json:"project_path"`
		ErrorMsg    string                 `json:"error_msg"`
		Solution    string                 `json:"solution"`
		Context     string                 `json:"context"`
		Metadata    map[string]interface{} `json:"metadata"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return JSONRPCErrorWithContext(c, req.ID, InvalidParams, err)
	}

	// Check service availability
	if s.remediationService == nil {
		return JSONRPCErrorWithContext(c, req.ID, InternalError, fmt.Errorf("remediation service not available"))
	}

	// Create remediation
	rem := &remediation.Remediation{
		ProjectPath: params.ProjectPath,
		ErrorMsg:    params.ErrorMsg,
		Solution:    params.Solution,
		Context:     params.Context,
		Metadata:    params.Metadata,
	}

	// Save via service
	ctx := c.Request().Context()
	if err := s.remediationService.Save(ctx, rem); err != nil {
		return JSONRPCErrorWithContext(c, req.ID, InternalError, err)
	}

	return JSONRPCSuccess(c, req.ID, map[string]interface{}{
		"remediation_id": rem.ID,
	})
}

// handleRemediationSearch searches for similar error remediations with hybrid matching.
func (s *Server) handleRemediationSearch(c echo.Context) error {
	var req JSONRPCRequest
	if err := c.Bind(&req); err != nil {
		return JSONRPCErrorWithContext(c, "", ParseError, err)
	}

	var params struct {
		ProjectPath string `json:"project_path"`
		ErrorMsg    string `json:"error_msg"`
		Limit       int    `json:"limit"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return JSONRPCErrorWithContext(c, req.ID, InvalidParams, err)
	}

	// Check service availability
	if s.remediationService == nil {
		return JSONRPCErrorWithContext(c, req.ID, InternalError, fmt.Errorf("remediation service not available"))
	}

	if params.Limit == 0 {
		params.Limit = 5
	}

	// Get prefetch results (if enabled)
	var prefetchData []prefetch.PreFetchResult
	if s.prefetchEnabled && params.ProjectPath != "" {
		prefetchData = s.GetPrefetchResults(params.ProjectPath)
	}

	// Search via service
	ctx := c.Request().Context()
	results, err := s.remediationService.Search(ctx, params.ErrorMsg, &remediation.SearchOptions{
		ProjectPath: params.ProjectPath,
		Limit:       params.Limit,
	})
	if err != nil {
		return JSONRPCErrorWithContext(c, req.ID, InternalError, err)
	}

	response := map[string]interface{}{
		"remediations": results,
	}
	if len(prefetchData) > 0 {
		response["prefetch"] = prefetchData
	}

	return JSONRPCSuccess(c, req.ID, response)
}

func (s *Server) handleSkillSave(c echo.Context) error {
	return JSONRPCSuccess(c, "req-def", map[string]string{
		"skill_id": "skill-placeholder",
	})
}

func (s *Server) handleSkillSearch(c echo.Context) error {
	return JSONRPCSuccess(c, "req-ghi", map[string]interface{}{
		"results": []map[string]string{},
	})
}

func (s *Server) handleIndexRepository(c echo.Context) error {
	// This would be a long-running operation like checkpoint_save
	return JSONRPCSuccess(c, "req-jkl", map[string]string{
		"operation_id": "op-index-placeholder",
		"status":       "pending",
	})
}

func (s *Server) handleStatus(c echo.Context) error {
	return JSONRPCSuccess(c, "req-mno", map[string]interface{}{
		"status":  "healthy",
		"service": "contextd",
		"version": "0.9.0-rc-1",
	})
}

// InitializePrefetch initializes the pre-fetch engine support.
//
// This method should be called after server creation and before starting
// the server if pre-fetch functionality is desired.
//
// Parameters:
//   - cfg: Pre-fetch configuration (if nil or Enabled=false, pre-fetch is disabled)
//   - logger: Structured logger for prefetch operations
//
// Returns an error if initialization fails.
func (s *Server) InitializePrefetch(cfg *config.PreFetchConfig, logger *zap.Logger) error {
	if cfg == nil || !cfg.Enabled {
		s.prefetchEnabled = false
		return nil
	}

	// Create cache
	s.prefetchCache = prefetch.NewCache(cfg.CacheTTL, cfg.CacheMaxEntries)

	// Set metrics on cache
	metrics := prefetch.NewMetrics()
	s.prefetchCache.SetMetrics(metrics)

	// Create executor
	s.prefetchExecutor = prefetch.NewExecutor(3) // max 3 rules in parallel
	s.prefetchExecutor.SetMetrics(metrics)
	s.prefetchExecutor.SetLogger(logger)

	// Initialize detector map
	s.prefetchDetectors = make(map[string]*prefetch.Detector)
	s.prefetchLogger = logger
	s.prefetchEnabled = true

	logger.Info("Prefetch engine initialized",
		zap.Duration("cache_ttl", cfg.CacheTTL),
		zap.Int("cache_max_entries", cfg.CacheMaxEntries))

	return nil
}

// StartPrefetchDetector starts a pre-fetch detector for a project.
//
// This method should be called when a project is indexed to enable
// automatic pre-fetching for that project.
//
// Parameters:
//   - ctx: Context for cancellation
//   - projectPath: Absolute path to the project
//
// Returns an error if the detector fails to start.
func (s *Server) StartPrefetchDetector(ctx context.Context, projectPath string) error {
	if !s.prefetchEnabled {
		return nil // Pre-fetch disabled, no-op
	}

	s.prefetchMu.Lock()
	defer s.prefetchMu.Unlock()

	// Check if detector already exists
	if _, exists := s.prefetchDetectors[projectPath]; exists {
		s.prefetchLogger.Debug("Prefetch detector already running",
			zap.String("project", projectPath))
		return nil
	}

	// Create detector
	detector, err := prefetch.NewDetector(projectPath, s.prefetchCache, s.prefetchExecutor, s.prefetchLogger)
	if err != nil {
		s.prefetchLogger.Warn("Failed to create prefetch detector",
			zap.Error(err),
			zap.String("project", projectPath))
		return nil // Don't fail if prefetch fails
	}

	// Store detector
	s.prefetchDetectors[projectPath] = detector

	// Start detector in background
	go detector.Start(ctx)

	s.prefetchLogger.Info("Prefetch detector started",
		zap.String("project", projectPath))

	return nil
}

// StopPrefetchDetector stops the pre-fetch detector for a project.
//
// This method should be called when a project is removed or unindexed.
//
// Parameters:
//   - projectPath: Absolute path to the project
func (s *Server) StopPrefetchDetector(projectPath string) {
	if !s.prefetchEnabled {
		return
	}

	s.prefetchMu.Lock()
	detector, exists := s.prefetchDetectors[projectPath]
	if exists {
		delete(s.prefetchDetectors, projectPath)
	}
	s.prefetchMu.Unlock()

	if detector != nil {
		detector.Stop()
		s.prefetchLogger.Info("Prefetch detector stopped",
			zap.String("project", projectPath))
	}
}

// GetPrefetchResults retrieves pre-fetched results for a project.
//
// This method should be called from search handlers to inject
// prefetched data into responses.
//
// Parameters:
//   - projectPath: Absolute path to the project
//
// Returns:
//   - results: Pre-fetched results, or empty slice if cache miss
func (s *Server) GetPrefetchResults(projectPath string) []prefetch.PreFetchResult {
	if !s.prefetchEnabled || s.prefetchCache == nil {
		return nil
	}

	entry, ok := s.prefetchCache.Get(projectPath)
	if !ok {
		return nil
	}

	return entry.Results
}

// Shutdown gracefully shuts down the server and all prefetch detectors.
//
// This method stops all running detectors before shutting down the server.
func (s *Server) Shutdown(ctx context.Context) error {
	if s.prefetchEnabled {
		// Stop all detectors
		s.prefetchMu.Lock()
		detectors := make([]*prefetch.Detector, 0, len(s.prefetchDetectors))
		for _, d := range s.prefetchDetectors {
			detectors = append(detectors, d)
		}
		s.prefetchDetectors = make(map[string]*prefetch.Detector)
		s.prefetchMu.Unlock()

		// Stop each detector
		for _, d := range detectors {
			d.Stop()
		}

		if s.prefetchLogger != nil {
			s.prefetchLogger.Info("All prefetch detectors stopped")
		}
	}

	// TODO: Add other shutdown logic here (e.g., close NATS connection)

	return nil
}
