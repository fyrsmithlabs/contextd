package checkpoint

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"

	"github.com/fyrsmithlabs/contextd/internal/tenant"
	"github.com/fyrsmithlabs/contextd/internal/vectorstore"
)

const instrumentationName = "github.com/fyrsmithlabs/contextd/internal/checkpoint"

// Service provides checkpoint management operations.
type Service interface {
	// Save creates a new checkpoint.
	Save(ctx context.Context, req *SaveRequest) (*Checkpoint, error)

	// List retrieves checkpoints for a session or project.
	List(ctx context.Context, req *ListRequest) ([]*Checkpoint, error)

	// Resume restores a checkpoint at the specified level.
	Resume(ctx context.Context, req *ResumeRequest) (*ResumeResponse, error)

	// Get retrieves a checkpoint by ID.
	Get(ctx context.Context, tenantID, checkpointID string) (*Checkpoint, error)

	// Delete removes a checkpoint.
	Delete(ctx context.Context, tenantID, checkpointID string) error

	// Close closes the service.
	Close() error
}

// Config configures the checkpoint service.
type Config struct {
	// VectorSize is the dimension of embedding vectors (default: 1536)
	VectorSize uint64

	// MaxCheckpointsPerSession limits checkpoints per session (default: 10)
	MaxCheckpointsPerSession int

	// AutoCheckpointThresholds are context % levels for auto-checkpoint.
	AutoCheckpointThresholds []float64
}

// DefaultServiceConfig returns sensible defaults.
func DefaultServiceConfig() *Config {
	return &Config{
		VectorSize:               1536,
		MaxCheckpointsPerSession: 10,
		AutoCheckpointThresholds: []float64{0.25, 0.5, 0.75, 0.9},
	}
}

// service implements the Service interface.
type service struct {
	config *Config
	store  vectorstore.Store
	logger *zap.Logger
	router tenant.CollectionRouter

	// Telemetry
	tracer        trace.Tracer
	meter         metric.Meter
	saveCounter   metric.Int64Counter
	resumeCounter metric.Int64Counter

	mu     sync.RWMutex
	closed bool
}

// NewService creates a new checkpoint service.
func NewService(cfg *Config, store vectorstore.Store, logger *zap.Logger) (Service, error) {
	if cfg == nil {
		cfg = DefaultServiceConfig()
	}
	if store == nil {
		return nil, errors.New("vector store is required")
	}
	if logger == nil {
		return nil, errors.New("logger is required for checkpoint service")
	}

	s := &service{
		config: cfg,
		store:  store,
		logger: logger,
		router: tenant.NewRouter(false),
		tracer: otel.Tracer(instrumentationName),
		meter:  otel.Meter(instrumentationName),
	}

	s.initMetrics()

	return s, nil
}

// initMetrics initializes OpenTelemetry metrics.
func (s *service) initMetrics() {
	var err error

	s.saveCounter, err = s.meter.Int64Counter(
		"contextd.checkpoint.saves_total",
		metric.WithDescription("Total number of checkpoints saved"),
		metric.WithUnit("{save}"),
	)
	if err != nil {
		s.logger.Warn("failed to create save counter", zap.Error(err))
	}

	s.resumeCounter, err = s.meter.Int64Counter(
		"contextd.checkpoint.resumes_total",
		metric.WithDescription("Total number of checkpoint resumes"),
		metric.WithUnit("{resume}"),
	)
	if err != nil {
		s.logger.Warn("failed to create resume counter", zap.Error(err))
	}
}

// collectionName returns the collection name for checkpoints (project-level).
// Per spec, checkpoints are stored at project level: {team}_{project}_checkpoints
// NOTE: For now using orgcheckpoints at org level until teamID is added to checkpoint schema
func (s *service) collectionName(tenantID string) (string, error) {
	// TODO: Add TeamID to Checkpoint struct, then use ScopeProject
	// For now, use org-level collection for backwards compatibility
	return s.router.GetCollectionName(tenant.ScopeOrg, tenant.CollectionCheckpoints, tenantID, "", "")
}

// Save creates a new checkpoint.
func (s *service) Save(ctx context.Context, req *SaveRequest) (*Checkpoint, error) {
	ctx, span := s.tracer.Start(ctx, "checkpoint.save")
	defer span.End()

	span.SetAttributes(
		attribute.String("tenant_id", req.TenantID),
		attribute.String("session_id", req.SessionID),
		attribute.Bool("auto_created", req.AutoCreated),
	)

	s.mu.RLock()
	if s.closed {
		s.mu.RUnlock()
		return nil, errors.New("service is closed")
	}
	s.mu.RUnlock()

	// Create checkpoint
	cp := &Checkpoint{
		ID:          uuid.New().String(),
		SessionID:   req.SessionID,
		TenantID:    req.TenantID,
		ProjectPath: req.ProjectPath,
		Name:        req.Name,
		Description: req.Description,
		Summary:     req.Summary,
		Context:     req.Context,
		FullState:   req.FullState,
		TokenCount:  req.TokenCount,
		Threshold:   req.Threshold,
		AutoCreated: req.AutoCreated,
		Metadata:    req.Metadata,
		CreatedAt:   time.Now(),
	}

	// Get collection name
	collection, err := s.collectionName(req.TenantID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("failed to get collection name: %w", err)
	}

	// Ensure collection exists
	exists, err := s.store.CollectionExists(ctx, collection)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to check collection: %w", err)
	}
	if !exists {
		// Use store's configured vector size (0 = use default from embedder)
		if err := s.store.CreateCollection(ctx, collection, 0); err != nil {
			span.RecordError(err)
			return nil, fmt.Errorf("failed to create collection: %w", err)
		}
	}

	// Convert checkpoint to document for storage
	doc := s.checkpointToDocument(cp, collection)

	// Store in vector store
	if _, err := s.store.AddDocuments(ctx, []vectorstore.Document{doc}); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("failed to save checkpoint: %w", err)
	}

	// Record metrics
	if s.saveCounter != nil {
		s.saveCounter.Add(ctx, 1, metric.WithAttributes(
			attribute.Bool("auto_created", req.AutoCreated),
		))
	}

	s.logger.Info("saved checkpoint",
		zap.String("id", cp.ID),
		zap.String("session_id", cp.SessionID),
		zap.Bool("auto_created", cp.AutoCreated),
	)

	span.SetAttributes(attribute.String("checkpoint_id", cp.ID))
	return cp, nil
}

// List retrieves checkpoints for a session or project.
func (s *service) List(ctx context.Context, req *ListRequest) ([]*Checkpoint, error) {
	ctx, span := s.tracer.Start(ctx, "checkpoint.list")
	defer span.End()

	span.SetAttributes(
		attribute.String("tenant_id", req.TenantID),
		attribute.String("session_id", req.SessionID),
		attribute.String("project_path", req.ProjectPath),
		attribute.Int("limit", req.Limit),
	)

	s.mu.RLock()
	if s.closed {
		s.mu.RUnlock()
		return nil, errors.New("service is closed")
	}
	s.mu.RUnlock()

	collection, err := s.collectionName(req.TenantID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("failed to get collection name: %w", err)
	}

	// Check if collection exists
	exists, err := s.store.CollectionExists(ctx, collection)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to check collection: %w", err)
	}
	if !exists {
		return []*Checkpoint{}, nil
	}

	// Build filter
	filters := make(map[string]interface{})
	if req.SessionID != "" {
		filters["session_id"] = req.SessionID
	}
	if req.ProjectPath != "" {
		filters["project_path"] = req.ProjectPath
	}
	if req.AutoOnly {
		filters["auto_created"] = true
	}

	limit := req.Limit
	if limit == 0 {
		limit = 20
	}

	// Search with a generic query to get checkpoints
	// Use "checkpoint" as a neutral search term since we filter by metadata
	results, err := s.store.SearchInCollection(ctx, collection, "checkpoint", limit, filters)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("failed to list checkpoints: %w", err)
	}

	checkpoints := make([]*Checkpoint, 0, len(results))
	for _, r := range results {
		cp := s.resultToCheckpoint(r)
		if cp != nil {
			checkpoints = append(checkpoints, cp)
		}
	}

	span.SetAttributes(attribute.Int("result_count", len(checkpoints)))
	return checkpoints, nil
}

// Resume restores a checkpoint at the specified level.
func (s *service) Resume(ctx context.Context, req *ResumeRequest) (*ResumeResponse, error) {
	ctx, span := s.tracer.Start(ctx, "checkpoint.resume")
	defer span.End()

	span.SetAttributes(
		attribute.String("tenant_id", req.TenantID),
		attribute.String("checkpoint_id", req.CheckpointID),
		attribute.String("level", string(req.Level)),
	)

	s.mu.RLock()
	if s.closed {
		s.mu.RUnlock()
		return nil, errors.New("service is closed")
	}
	s.mu.RUnlock()

	// Get the checkpoint
	cp, err := s.Get(ctx, req.TenantID, req.CheckpointID)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}

	// Determine content based on level
	var content string
	var tokenCount int32

	switch req.Level {
	case ResumeSummary:
		content = cp.Summary
		tokenCount = estimateTokens(content)
	case ResumeContext:
		content = cp.Summary + "\n\n---\n\n" + cp.Context
		tokenCount = estimateTokens(content)
	case ResumeFull:
		content = cp.FullState
		tokenCount = cp.TokenCount
	default:
		content = cp.Summary
		tokenCount = estimateTokens(content)
	}

	// Record metrics
	if s.resumeCounter != nil {
		s.resumeCounter.Add(ctx, 1, metric.WithAttributes(
			attribute.String("level", string(req.Level)),
		))
	}

	s.logger.Info("resumed checkpoint",
		zap.String("id", cp.ID),
		zap.String("level", string(req.Level)),
		zap.Int32("token_count", tokenCount),
	)

	return &ResumeResponse{
		Checkpoint: cp,
		Content:    content,
		TokenCount: tokenCount,
	}, nil
}

// Get retrieves a checkpoint by ID.
func (s *service) Get(ctx context.Context, tenantID, checkpointID string) (*Checkpoint, error) {
	ctx, span := s.tracer.Start(ctx, "checkpoint.get")
	defer span.End()

	span.SetAttributes(
		attribute.String("tenant_id", tenantID),
		attribute.String("checkpoint_id", checkpointID),
	)

	s.mu.RLock()
	if s.closed {
		s.mu.RUnlock()
		return nil, errors.New("service is closed")
	}
	s.mu.RUnlock()

	collection, err := s.collectionName(tenantID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("failed to get collection name: %w", err)
	}

	// Check if collection exists
	exists, err := s.store.CollectionExists(ctx, collection)
	if err != nil || !exists {
		return nil, fmt.Errorf("checkpoint not found: %s", checkpointID)
	}

	// Search by ID using filter
	filters := map[string]interface{}{
		"id": checkpointID,
	}

	// Use a dummy query since we're filtering by ID
	results, err := s.store.SearchInCollection(ctx, collection, "checkpoint", 1, filters)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to get checkpoint: %w", err)
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("checkpoint not found: %s", checkpointID)
	}

	cp := s.resultToCheckpoint(results[0])
	if cp == nil {
		return nil, fmt.Errorf("invalid checkpoint data: %s", checkpointID)
	}

	return cp, nil
}

// Delete removes a checkpoint.
func (s *service) Delete(ctx context.Context, tenantID, checkpointID string) error {
	ctx, span := s.tracer.Start(ctx, "checkpoint.delete")
	defer span.End()

	span.SetAttributes(
		attribute.String("tenant_id", tenantID),
		attribute.String("checkpoint_id", checkpointID),
	)

	s.mu.RLock()
	if s.closed {
		s.mu.RUnlock()
		return errors.New("service is closed")
	}
	s.mu.RUnlock()

	collection, err := s.collectionName(tenantID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("failed to get collection name: %w", err)
	}

	if err := s.store.DeleteDocumentsFromCollection(ctx, collection, []string{checkpointID}); err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to delete checkpoint: %w", err)
	}

	s.logger.Info("deleted checkpoint", zap.String("id", checkpointID))
	return nil
}

// Close closes the service.
func (s *service) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}
	s.closed = true
	return nil
}

// Helper functions

// checkpointToDocument converts a Checkpoint to a vectorstore Document.
func (s *service) checkpointToDocument(cp *Checkpoint, collectionName string) vectorstore.Document {
	// Combine name and summary for embedding
	content := fmt.Sprintf("%s\n\n%s", cp.Name, cp.Summary)

	metadata := map[string]interface{}{
		"id":           cp.ID,
		"session_id":   cp.SessionID,
		"tenant_id":    cp.TenantID,
		"project_path": cp.ProjectPath,
		"name":         cp.Name,
		"description":  cp.Description,
		"summary":      cp.Summary,
		"context":      cp.Context,
		"full_state":   cp.FullState,
		"token_count":  int64(cp.TokenCount),
		"threshold":    cp.Threshold,
		"auto_created": cp.AutoCreated,
		"created_at":   cp.CreatedAt.Unix(),
	}

	// Add metadata
	for k, v := range cp.Metadata {
		metadata["meta_"+k] = v
	}

	return vectorstore.Document{
		ID:         cp.ID,
		Content:    content,
		Metadata:   metadata,
		Collection: collectionName,
	}
}

// resultToCheckpoint converts a vectorstore SearchResult to a Checkpoint.
func (s *service) resultToCheckpoint(result vectorstore.SearchResult) *Checkpoint {
	if result.Metadata == nil {
		return nil
	}

	cp := &Checkpoint{
		Metadata: make(map[string]string),
	}

	// Extract ID
	if id, ok := result.Metadata["id"].(string); ok {
		cp.ID = id
	} else {
		cp.ID = result.ID
	}

	if v, ok := result.Metadata["session_id"].(string); ok {
		cp.SessionID = v
	}
	if v, ok := result.Metadata["tenant_id"].(string); ok {
		cp.TenantID = v
	}
	if v, ok := result.Metadata["project_path"].(string); ok {
		cp.ProjectPath = v
	}
	if v, ok := result.Metadata["name"].(string); ok {
		cp.Name = v
	}
	if v, ok := result.Metadata["description"].(string); ok {
		cp.Description = v
	}
	if v, ok := result.Metadata["summary"].(string); ok {
		cp.Summary = v
	}
	if v, ok := result.Metadata["context"].(string); ok {
		cp.Context = v
	}
	if v, ok := result.Metadata["full_state"].(string); ok {
		cp.FullState = v
	}
	if v, ok := result.Metadata["token_count"].(int64); ok {
		cp.TokenCount = int32(v)
	} else if v, ok := result.Metadata["token_count"].(float64); ok {
		cp.TokenCount = int32(v)
	}
	if v, ok := result.Metadata["threshold"].(float64); ok {
		cp.Threshold = v
	}
	if v, ok := result.Metadata["auto_created"].(bool); ok {
		cp.AutoCreated = v
	} else if v, ok := result.Metadata["auto_created"].(string); ok {
		cp.AutoCreated = v == "true"
	}
	if v, ok := result.Metadata["created_at"].(int64); ok {
		cp.CreatedAt = time.Unix(v, 0)
	} else if v, ok := result.Metadata["created_at"].(float64); ok {
		cp.CreatedAt = time.Unix(int64(v), 0)
	}

	// Extract metadata
	for k, v := range result.Metadata {
		if len(k) > 5 && k[:5] == "meta_" {
			if str, ok := v.(string); ok {
				cp.Metadata[k[5:]] = str
			}
		}
	}

	return cp
}

// checkpointToPayload converts a Checkpoint to a map for storage (used by tests).
func checkpointToPayload(cp *Checkpoint) map[string]interface{} {
	payload := map[string]interface{}{
		"session_id":   cp.SessionID,
		"tenant_id":    cp.TenantID,
		"project_path": cp.ProjectPath,
		"name":         cp.Name,
		"description":  cp.Description,
		"summary":      cp.Summary,
		"context":      cp.Context,
		"full_state":   cp.FullState,
		"token_count":  int64(cp.TokenCount),
		"threshold":    cp.Threshold,
		"auto_created": cp.AutoCreated,
		"created_at":   cp.CreatedAt.Unix(),
	}

	// Add metadata
	for k, v := range cp.Metadata {
		payload["meta_"+k] = v
	}

	return payload
}

// payloadToCheckpoint converts a map to a Checkpoint (used by tests).
func payloadToCheckpoint(payload map[string]interface{}) *Checkpoint {
	if payload == nil {
		return nil
	}

	cp := &Checkpoint{
		Metadata: make(map[string]string),
	}

	if v, ok := payload["session_id"].(string); ok {
		cp.SessionID = v
	}
	if v, ok := payload["tenant_id"].(string); ok {
		cp.TenantID = v
	}
	if v, ok := payload["project_path"].(string); ok {
		cp.ProjectPath = v
	}
	if v, ok := payload["name"].(string); ok {
		cp.Name = v
	}
	if v, ok := payload["description"].(string); ok {
		cp.Description = v
	}
	if v, ok := payload["summary"].(string); ok {
		cp.Summary = v
	}
	if v, ok := payload["context"].(string); ok {
		cp.Context = v
	}
	if v, ok := payload["full_state"].(string); ok {
		cp.FullState = v
	}
	if v, ok := payload["token_count"].(int64); ok {
		cp.TokenCount = int32(v)
	}
	if v, ok := payload["threshold"].(float64); ok {
		cp.Threshold = v
	}
	if v, ok := payload["auto_created"].(bool); ok {
		cp.AutoCreated = v
	}
	if v, ok := payload["created_at"].(int64); ok {
		cp.CreatedAt = time.Unix(v, 0)
	}

	// Extract metadata
	for k, v := range payload {
		if len(k) > 5 && k[:5] == "meta_" {
			if str, ok := v.(string); ok {
				cp.Metadata[k[5:]] = str
			}
		}
	}

	return cp
}

func estimateTokens(text string) int32 {
	// Simple estimate: ~4 chars per token
	return int32(len(text) / 4)
}
