// internal/mcp/handlers/memory.go
package handlers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/fyrsmithlabs/contextd/internal/reasoningbank"
)

// MemoryConsolidateInput is the input for memory_consolidate tool.
type MemoryConsolidateInput struct {
	ProjectID           string  `json:"project_id"`
	SimilarityThreshold float64 `json:"similarity_threshold,omitempty"`
	DryRun              bool    `json:"dry_run,omitempty"`
	MaxClusters         int     `json:"max_clusters,omitempty"`
}

// MemoryConsolidateOutput is the output for memory_consolidate tool.
type MemoryConsolidateOutput struct {
	CreatedMemories  []string `json:"created_memories"`
	ArchivedMemories []string `json:"archived_memories"`
	SkippedCount     int      `json:"skipped_count"`
	TotalProcessed   int      `json:"total_processed"`
	DurationSeconds  float64  `json:"duration_seconds"`
}

// MemoryHandler handles memory-related tools.
type MemoryHandler struct {
	distiller *reasoningbank.Distiller
}

// NewMemoryHandler creates a new memory handler.
func NewMemoryHandler(distiller *reasoningbank.Distiller) *MemoryHandler {
	return &MemoryHandler{distiller: distiller}
}

// Consolidate handles the memory_consolidate tool.
// It consolidates similar memories to reduce redundancy and improve knowledge quality.
func (h *MemoryHandler) Consolidate(ctx context.Context, input json.RawMessage) (interface{}, error) {
	var req MemoryConsolidateInput
	if err := json.Unmarshal(input, &req); err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}

	// Validate input
	if req.ProjectID == "" {
		return nil, fmt.Errorf("project_id is required")
	}

	// Check if distiller is available
	if h.distiller == nil {
		return nil, fmt.Errorf("memory consolidation not available: distiller not configured")
	}

	// Apply default similarity threshold if not specified
	threshold := req.SimilarityThreshold
	if threshold == 0 {
		threshold = 0.8 // Default as specified in spec
	}

	// Build consolidation options
	opts := reasoningbank.ConsolidationOptions{
		SimilarityThreshold: threshold,
		DryRun:              req.DryRun,
		MaxClustersPerRun:   req.MaxClusters,
	}

	// Execute consolidation
	result, err := h.distiller.Consolidate(ctx, req.ProjectID, opts)
	if err != nil {
		return nil, fmt.Errorf("consolidation failed: %w", err)
	}

	// Convert duration to seconds
	durationSeconds := result.Duration.Seconds()

	// Build output
	output := MemoryConsolidateOutput{
		CreatedMemories:  result.CreatedMemories,
		ArchivedMemories: result.ArchivedMemories,
		SkippedCount:     result.SkippedCount,
		TotalProcessed:   result.TotalProcessed,
		DurationSeconds:  durationSeconds,
	}

	return output, nil
}
