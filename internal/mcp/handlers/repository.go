package handlers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/fyrsmithlabs/contextd/internal/repository"
)

// RepositoryHandler wraps repository service for MCP tool interface.
type RepositoryHandler struct {
	service *repository.Service
}

// NewRepositoryHandler creates a new repository handler.
func NewRepositoryHandler(service *repository.Service) *RepositoryHandler {
	return &RepositoryHandler{
		service: service,
	}
}

// RepositoryIndexInput represents input for repository_index tool.
type RepositoryIndexInput struct {
	Path            string   `json:"path"`
	IncludePatterns []string `json:"include_patterns,omitempty"`
	ExcludePatterns []string `json:"exclude_patterns,omitempty"`
	MaxFileSize     int64    `json:"max_file_size,omitempty"`
}

// Index handles repository_index MCP tool call.
func (h *RepositoryHandler) Index(ctx context.Context, input json.RawMessage) (interface{}, error) {
	var req RepositoryIndexInput
	if err := json.Unmarshal(input, &req); err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}

	// Convert to service options
	opts := repository.IndexOptions{
		IncludePatterns: req.IncludePatterns,
		ExcludePatterns: req.ExcludePatterns,
		MaxFileSize:     req.MaxFileSize,
	}

	result, err := h.service.IndexRepository(ctx, req.Path, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to index repository: %w", err)
	}

	return map[string]interface{}{
		"path":             result.Path,
		"files_indexed":    result.FilesIndexed,
		"include_patterns": result.IncludePatterns,
		"exclude_patterns": result.ExcludePatterns,
		"max_file_size":    result.MaxFileSize,
		"indexed_at":       result.IndexedAt,
	}, nil
}
