// Package mcp provides a simplified MCP server that calls internal packages directly.
//
// This implementation uses the MCP SDK (github.com/modelcontextprotocol/go-sdk/mcp)
// and calls internal services directly without gRPC overhead.
package mcp

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"

	"github.com/fyrsmithlabs/contextd/internal/checkpoint"
	"github.com/fyrsmithlabs/contextd/internal/conversation"
	"github.com/fyrsmithlabs/contextd/internal/folding"
	"github.com/fyrsmithlabs/contextd/internal/ignore"
	"github.com/fyrsmithlabs/contextd/internal/reasoningbank"
	"github.com/fyrsmithlabs/contextd/internal/remediation"
	"github.com/fyrsmithlabs/contextd/internal/repository"
	"github.com/fyrsmithlabs/contextd/internal/secrets"
	"github.com/fyrsmithlabs/contextd/internal/troubleshoot"
)

// Server is a simplified MCP server that calls internal packages directly.
type Server struct {
	mcp              *mcp.Server
	checkpointSvc    checkpoint.Service
	remediationSvc   remediation.Service
	repositorySvc    *repository.Service
	troubleshootSvc  *troubleshoot.Service
	reasoningbankSvc *reasoningbank.Service
	conversationSvc  conversation.ConversationService
	foldingSvc       *folding.BranchManager
	distiller        *reasoningbank.Distiller
	scrubber         secrets.Scrubber
	ignoreParser     *ignore.Parser
	toolRegistry     *ToolRegistry
	logger           *zap.Logger
}

// Config configures the MCP server.
type Config struct {
	// Name is the server implementation name (default: "contextd-v2")
	Name string

	// Version is the server version (default: "1.0.0")
	Version string

	// Logger for structured logging
	Logger *zap.Logger

	// IgnoreFiles is the list of ignore file names to parse from project root.
	// Default: [".gitignore", ".dockerignore", ".contextdignore"]
	IgnoreFiles []string

	// FallbackExcludes are used when no ignore files are found.
	// Default: [".git/**", "node_modules/**", "vendor/**", "__pycache__/**"]
	FallbackExcludes []string
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		Name:    "contextd-v2",
		Version: "1.0.0",
		Logger:  zap.NewNop(),
		IgnoreFiles: []string{
			".gitignore",
			".dockerignore",
			".contextdignore",
		},
		FallbackExcludes: []string{
			".git/**",
			"node_modules/**",
			"vendor/**",
			"__pycache__/**",
		},
	}
}

// NewServer creates a new MCP server with the given services.
func NewServer(
	cfg *Config,
	checkpointSvc checkpoint.Service,
	remediationSvc remediation.Service,
	repositorySvc *repository.Service,
	troubleshootSvc *troubleshoot.Service,
	reasoningbankSvc *reasoningbank.Service,
	foldingSvc *folding.BranchManager,
	distiller *reasoningbank.Distiller,
	scrubber secrets.Scrubber,
) (*Server, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}
	if checkpointSvc == nil {
		return nil, fmt.Errorf("checkpoint service is required")
	}
	if remediationSvc == nil {
		return nil, fmt.Errorf("remediation service is required")
	}
	if repositorySvc == nil {
		return nil, fmt.Errorf("repository service is required")
	}
	if troubleshootSvc == nil {
		return nil, fmt.Errorf("troubleshoot service is required")
	}
	if reasoningbankSvc == nil {
		return nil, fmt.Errorf("reasoningbank service is required")
	}
	// foldingSvc is optional - context folding is an optional feature
	if scrubber == nil {
		return nil, fmt.Errorf("scrubber is required")
	}

	// Create MCP server
	mcpServer := mcp.NewServer(
		&mcp.Implementation{
			Name:    cfg.Name,
			Version: cfg.Version,
		},
		nil,
	)

	// Create ignore parser for repository indexing
	ignoreParser := ignore.NewParser(cfg.IgnoreFiles, cfg.FallbackExcludes)

	s := &Server{
		mcp:              mcpServer,
		checkpointSvc:    checkpointSvc,
		remediationSvc:   remediationSvc,
		repositorySvc:    repositorySvc,
		troubleshootSvc:  troubleshootSvc,
		reasoningbankSvc: reasoningbankSvc,
		foldingSvc:       foldingSvc,
		distiller:        distiller,
		scrubber:         scrubber,
		ignoreParser:     ignoreParser,
		logger:           cfg.Logger,
	}

	// Register tools
	if err := s.registerTools(); err != nil {
		return nil, fmt.Errorf("failed to register tools: %w", err)
	}

	return s, nil
}

// SetConversationService sets the optional conversation service.
// Must be called before Run() to enable conversation tools.
func (s *Server) SetConversationService(svc conversation.ConversationService) {
	s.conversationSvc = svc
}

// Run starts the MCP server on the stdio transport.
func (s *Server) Run(ctx context.Context) error {
	s.logger.Info("starting MCP server on stdio transport")
	transport := &mcp.StdioTransport{}
	if err := s.mcp.Run(ctx, transport); err != nil {
		return fmt.Errorf("server run failed: %w", err)
	}
	return nil
}

// Close closes the server and all services.
func (s *Server) Close() error {
	s.logger.Info("closing MCP server and services")

	var errs []error

	if err := s.checkpointSvc.Close(); err != nil {
		errs = append(errs, fmt.Errorf("checkpoint service close: %w", err))
	}
	if err := s.remediationSvc.Close(); err != nil {
		errs = append(errs, fmt.Errorf("remediation service close: %w", err))
	}

	if len(errs) > 0 {
		return fmt.Errorf("close errors: %v", errs)
	}
	return nil
}
