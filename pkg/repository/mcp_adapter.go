package repository

import (
	"context"

	"github.com/fyrsmithlabs/contextd/pkg/mcp"
)

// MCPAdapter adapts repository.Service to MCP server interface requirements.
//
// This adapter bridges the repository service with the MCP server's
// type definitions, allowing loose coupling between packages.
type MCPAdapter struct {
	service *Service
}

// NewMCPAdapter creates an adapter for MCP integration.
func NewMCPAdapter(service *Service) *MCPAdapter {
	return &MCPAdapter{service: service}
}

// IndexRepository implements the MCP RepositoryService interface.
//
// It converts MCP types to repository types, calls the underlying service,
// and converts the result back to MCP types.
func (a *MCPAdapter) IndexRepository(ctx context.Context, path string, opts mcp.RepositoryIndexOptions) (*mcp.RepositoryIndexResult, error) {
	// Convert MCP options to repository options
	repoOpts := IndexOptions{
		IncludePatterns: opts.IncludePatterns,
		ExcludePatterns: opts.ExcludePatterns,
		MaxFileSize:     opts.MaxFileSize,
	}

	// Call underlying service
	result, err := a.service.IndexRepository(ctx, path, repoOpts)
	if err != nil {
		return nil, err
	}

	// Convert repository result to MCP result
	return &mcp.RepositoryIndexResult{
		Path:            result.Path,
		FilesIndexed:    result.FilesIndexed,
		IncludePatterns: result.IncludePatterns,
		ExcludePatterns: result.ExcludePatterns,
		MaxFileSize:     result.MaxFileSize,
	}, nil
}
