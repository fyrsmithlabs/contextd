package handlers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/fyrsmithlabs/contextd/internal/checkpoint"
	"github.com/fyrsmithlabs/contextd/internal/remediation"
	"github.com/fyrsmithlabs/contextd/internal/repository"
	"github.com/fyrsmithlabs/contextd/internal/troubleshoot"
)

// ToolHandler is the interface for MCP tool handlers.
type ToolHandler func(ctx context.Context, input json.RawMessage) (interface{}, error)

// Registry manages all MCP tool handlers.
type Registry struct {
	handlers map[string]ToolHandler
}

// NewRegistry creates a new handler registry.
func NewRegistry(
	checkpointSvc checkpoint.Service,
	remediationSvc remediation.Service,
	repositorySvc *repository.Service,
	troubleshootSvc *troubleshoot.Service,
) *Registry {
	// Create handlers
	checkpointHandler := NewCheckpointHandler(checkpointSvc)
	remediationHandler := NewRemediationHandler(remediationSvc)
	repositoryHandler := NewRepositoryHandler(repositorySvc)
	troubleshootHandler := NewTroubleshootHandler(troubleshootSvc)

	// Register tool handlers
	handlers := map[string]ToolHandler{
		// Checkpoint tools
		"checkpoint_save":   checkpointHandler.Save,
		"checkpoint_list":   checkpointHandler.List,
		"checkpoint_resume": checkpointHandler.Resume,

		// Remediation tools
		"remediation_search": remediationHandler.Search,
		"remediation_record": remediationHandler.Record,

		// Repository tools
		"repository_index": repositoryHandler.Index,

		// Troubleshoot tools
		"troubleshoot":           troubleshootHandler.Diagnose,
		"troubleshoot_pattern":   troubleshootHandler.SavePattern,
		"troubleshoot_patterns":  troubleshootHandler.GetPatterns,
	}

	return &Registry{
		handlers: handlers,
	}
}

// GetHandler returns the handler for a given tool name.
func (r *Registry) GetHandler(toolName string) (ToolHandler, error) {
	handler, ok := r.handlers[toolName]
	if !ok {
		return nil, fmt.Errorf("unknown tool: %s", toolName)
	}
	return handler, nil
}

// ListTools returns all available tool names.
func (r *Registry) ListTools() []string {
	tools := make([]string, 0, len(r.handlers))
	for name := range r.handlers {
		tools = append(tools, name)
	}
	return tools
}

// Call invokes a tool handler by name.
func (r *Registry) Call(ctx context.Context, toolName string, input json.RawMessage) (interface{}, error) {
	handler, err := r.GetHandler(toolName)
	if err != nil {
		return nil, err
	}
	return handler(ctx, input)
}
