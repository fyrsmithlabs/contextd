package repository

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/fyrsmithlabs/contextd/internal/checkpoint"
	"github.com/fyrsmithlabs/contextd/internal/tenant"
)

// defaultSkipDirs are directories that should always be skipped during indexing.
// These typically contain generated code, dependencies, or version control data.
var defaultSkipDirs = map[string]bool{
	".git":         true,
	".svn":         true,
	".hg":          true,
	"node_modules": true,
	"vendor":       true,
	".venv":        true,
	"venv":         true,
	"__pycache__":  true,
	".idea":        true,
	".vscode":      true,
	".cache":       true,
	"dist":         true,
	"build":        true,
	".next":        true,
	"target":       true, // Rust/Java build output
}

// CheckpointService defines the interface for checkpoint operations.
//
// This allows the repository service to create checkpoints without
// depending on the concrete checkpoint.Service implementation.
type CheckpointService interface {
	Save(ctx context.Context, req *checkpoint.SaveRequest) (*checkpoint.Checkpoint, error)
}

// Service provides repository indexing functionality.
//
// It walks file trees, filters files based on patterns and size limits,
// and creates searchable checkpoints for each indexed file.
type Service struct {
	checkpointService CheckpointService
}

// NewService creates a new repository indexing service.
func NewService(checkpointSvc CheckpointService) *Service {
	return &Service{
		checkpointService: checkpointSvc,
	}
}

// IndexRepository indexes all files in a repository matching the given options.
//
// The function walks the file tree at path, filters files according to include/exclude
// patterns and size limits, and creates a checkpoint for each indexed file.
//
// Security: The path is cleaned and validated to prevent path traversal attacks.
// Multi-tenant isolation is maintained through project-specific checkpoints.
//
// Returns IndexResult with statistics, or an error if indexing fails.
func (s *Service) IndexRepository(ctx context.Context, path string, opts IndexOptions) (*IndexResult, error) {
	// Validate and clean path
	cleanPath, err := validatePath(path)
	if err != nil {
		return nil, fmt.Errorf("invalid path: %w", err)
	}

	// Set defaults
	if opts.MaxFileSize == 0 {
		opts.MaxFileSize = 1024 * 1024 // 1MB default
	}
	if opts.MaxFileSize > 10*1024*1024 {
		return nil, fmt.Errorf("max_file_size cannot exceed 10MB")
	}

	// Validate patterns
	if err := validatePatterns(opts.IncludePatterns); err != nil {
		return nil, fmt.Errorf("invalid include pattern: %w", err)
	}
	if err := validatePatterns(opts.ExcludePatterns); err != nil {
		return nil, fmt.Errorf("invalid exclude pattern: %w", err)
	}

	// Track indexed files
	filesIndexed := 0

	// Walk file tree
	err = filepath.Walk(cleanPath, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories that should not be indexed
		if info.IsDir() {
			dirName := filepath.Base(filePath)
			if defaultSkipDirs[dirName] {
				return filepath.SkipDir
			}
			return nil
		}

		// Check context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Get relative path for pattern matching
		relPath, err := filepath.Rel(cleanPath, filePath)
		if err != nil {
			return fmt.Errorf("computing relative path: %w", err)
		}

		// Apply filters
		if !shouldIncludeFile(relPath, info, opts) {
			return nil
		}

		// Read file content
		content, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("reading file %s: %w", filePath, err)
		}

		// Skip binary files (invalid UTF-8)
		// gRPC requires valid UTF-8 strings, so we skip files that contain invalid UTF-8
		if !utf8.Valid(content) {
			// Skip binary files silently
			return nil
		}

		// Determine tenant ID (use GitHub username from repo if not specified)
		tenantID := opts.TenantID
		if tenantID == "" {
			tenantID = tenant.GetTenantIDForPath(cleanPath)
		}

		// Create checkpoint for file
		// Note: In contextd-v2, checkpoint.Save() expects a SaveRequest instead of Checkpoint
		req := &checkpoint.SaveRequest{
			TenantID:    tenantID,
			ProjectPath: cleanPath,
			Summary:     fmt.Sprintf("Indexed: %s", relPath),
			FullState:   string(content),
			Metadata: map[string]string{
				"file_path": relPath,
				"file_size": fmt.Sprintf("%d", info.Size()),
				"indexed":   "true",
				"extension": filepath.Ext(relPath),
			},
		}

		// Save checkpoint
		if _, err := s.checkpointService.Save(ctx, req); err != nil {
			return fmt.Errorf("saving checkpoint for %s: %w", relPath, err)
		}

		filesIndexed++
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("walking file tree: %w", err)
	}

	// Return result
	return &IndexResult{
		Path:            cleanPath,
		FilesIndexed:    filesIndexed,
		IncludePatterns: opts.IncludePatterns,
		ExcludePatterns: opts.ExcludePatterns,
		MaxFileSize:     opts.MaxFileSize,
		IndexedAt:       time.Now(),
	}, nil
}

// validatePath validates and cleans a file path.
func validatePath(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("path cannot be empty")
	}

	// Clean path (removes . and .. components)
	cleanPath := filepath.Clean(path)

	// Check if path exists
	info, err := os.Stat(cleanPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("path does not exist: %s", cleanPath)
		}
		return "", fmt.Errorf("stat path: %w", err)
	}

	// Must be a directory
	if !info.IsDir() {
		return "", fmt.Errorf("path must be a directory: %s", cleanPath)
	}

	return cleanPath, nil
}

// validatePatterns validates glob patterns.
func validatePatterns(patterns []string) error {
	for _, pattern := range patterns {
		if _, err := filepath.Match(pattern, "test"); err != nil {
			return fmt.Errorf("invalid pattern %q: %w", pattern, err)
		}
	}
	return nil
}

// shouldIncludeFile determines if a file should be indexed.
func shouldIncludeFile(relPath string, info os.FileInfo, opts IndexOptions) bool {
	basename := filepath.Base(relPath)

	// Check file size limit
	if info.Size() > opts.MaxFileSize {
		return false
	}

	// Check exclude patterns (takes precedence)
	for _, pattern := range opts.ExcludePatterns {
		// Match against basename
		if matched, _ := filepath.Match(pattern, basename); matched {
			return false
		}
		// Match against relative path
		if matched, _ := filepath.Match(pattern, relPath); matched {
			return false
		}
		// Match directory components for patterns like "vendor/**"
		if strings.Contains(pattern, "**") {
			prefix := strings.TrimSuffix(pattern, "/**")
			if strings.HasPrefix(relPath, prefix+string(filepath.Separator)) {
				return false
			}
		}
	}

	// Check include patterns (if specified)
	if len(opts.IncludePatterns) > 0 {
		included := false
		for _, pattern := range opts.IncludePatterns {
			// Match against basename
			if matched, _ := filepath.Match(pattern, basename); matched {
				included = true
				break
			}
			// Match against relative path
			if matched, _ := filepath.Match(pattern, relPath); matched {
				included = true
				break
			}
		}
		if !included {
			return false
		}
	}

	return true
}
