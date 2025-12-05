package repository

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"

	"github.com/fyrsmithlabs/contextd/internal/tenant"
	"github.com/fyrsmithlabs/contextd/internal/vectorstore"
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

// Store defines the interface for vector store operations.
// This allows the repository service to store and search indexed files.
type Store interface {
	// AddDocuments adds documents to the vector store.
	// Documents with Collection field set will be stored in that collection.
	AddDocuments(ctx context.Context, docs []vectorstore.Document) ([]string, error)

	// SearchInCollection performs semantic search in a specific collection.
	SearchInCollection(ctx context.Context, collectionName string, query string, k int, filters map[string]interface{}) ([]vectorstore.SearchResult, error)
}

// Service provides repository indexing functionality.
//
// It walks file trees, filters files based on patterns and size limits,
// and stores them in a dedicated _codebase collection with branch awareness.
type Service struct {
	store Store
}

// NewService creates a new repository indexing service.
func NewService(store Store) *Service {
	return &Service{
		store: store,
	}
}

// SearchOptions configures repository search behavior.
type SearchOptions struct {
	ProjectPath string // Required: project path to search within
	TenantID    string // Required: tenant identifier
	Branch      string // Optional: filter by branch (empty = all branches)
	Limit       int    // Max results (default: 10)
}

// RepoSearchResult from repository search.
type RepoSearchResult struct {
	FilePath string                 `json:"file_path"`
	Content  string                 `json:"content"`
	Score    float32                `json:"score"`
	Branch   string                 `json:"branch"`
	Metadata map[string]interface{} `json:"metadata"`
}

// Search performs semantic search over indexed repository files.
func (s *Service) Search(ctx context.Context, query string, opts SearchOptions) ([]RepoSearchResult, error) {
	if s.store == nil {
		return nil, fmt.Errorf("store not configured")
	}
	if query == "" {
		return nil, fmt.Errorf("query cannot be empty")
	}
	if opts.ProjectPath == "" {
		return nil, fmt.Errorf("project_path is required")
	}
	if opts.TenantID == "" {
		return nil, fmt.Errorf("tenant_id is required")
	}

	limit := opts.Limit
	if limit <= 0 {
		limit = 10
	}

	// Build collection name for codebase
	// Format: {tenant}_{project}_codebase (matches spec)
	projectName := sanitizeProjectName(filepath.Base(opts.ProjectPath))
	collectionName := fmt.Sprintf("%s_%s_codebase", opts.TenantID, projectName)

	// Build filters
	filters := make(map[string]interface{})
	if opts.Branch != "" {
		filters["branch"] = opts.Branch
	}

	results, err := s.store.SearchInCollection(ctx, collectionName, query, limit, filters)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	// Convert to repository search results
	repoResults := make([]RepoSearchResult, 0, len(results))
	for _, r := range results {
		branch := ""
		if b, ok := r.Metadata["branch"].(string); ok {
			branch = b
		}
		filePath := ""
		if fp, ok := r.Metadata["file_path"].(string); ok {
			filePath = fp
		}

		repoResults = append(repoResults, RepoSearchResult{
			FilePath: filePath,
			Content:  r.Content,
			Score:    r.Score,
			Branch:   branch,
			Metadata: r.Metadata,
		})
	}

	return repoResults, nil
}

// IndexRepository indexes all files in a repository matching the given options.
//
// Files are stored in a dedicated {tenant}_{project}_codebase collection,
// with branch metadata for filtering.
//
// Security: The path is cleaned and validated to prevent path traversal attacks.
// Multi-tenant isolation is maintained through project-specific collections.
//
// Returns IndexResult with statistics, or an error if indexing fails.
func (s *Service) IndexRepository(ctx context.Context, path string, opts IndexOptions) (*IndexResult, error) {
	if s.store == nil {
		return nil, fmt.Errorf("store not configured")
	}

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

	// Determine tenant ID
	tenantID := opts.TenantID
	if tenantID == "" {
		tenantID = tenant.GetTenantIDForPath(cleanPath)
	}

	// Detect branch (auto-detect if not specified)
	branch := opts.Branch
	if branch == "" {
		branch = detectGitBranch(cleanPath)
	}

	// Build collection name: {tenant}_{project}_codebase
	projectName := sanitizeProjectName(filepath.Base(cleanPath))
	collectionName := fmt.Sprintf("%s_%s_codebase", tenantID, projectName)

	// Collect documents to index
	var docs []vectorstore.Document

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
		if !utf8.Valid(content) {
			return nil
		}

		// Create document for vector store
		doc := vectorstore.Document{
			Content:    string(content),
			Collection: collectionName,
			Metadata: map[string]interface{}{
				"file_path":    relPath,
				"file_size":    info.Size(),
				"extension":    filepath.Ext(relPath),
				"branch":       branch,
				"project_path": cleanPath,
				"tenant_id":    tenantID,
				"indexed_at":   time.Now().UTC().Format(time.RFC3339),
			},
		}

		docs = append(docs, doc)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("walking file tree: %w", err)
	}

	// Add all documents to vector store
	if len(docs) > 0 {
		if _, err := s.store.AddDocuments(ctx, docs); err != nil {
			return nil, fmt.Errorf("storing documents: %w", err)
		}
	}

	// Return result
	return &IndexResult{
		Path:            cleanPath,
		Branch:          branch,
		CollectionName:  collectionName,
		FilesIndexed:    len(docs),
		IncludePatterns: opts.IncludePatterns,
		ExcludePatterns: opts.ExcludePatterns,
		MaxFileSize:     opts.MaxFileSize,
		IndexedAt:       time.Now(),
	}, nil
}

// detectGitBranch detects the current git branch for a path.
// Returns "unknown" if not a git repository or detection fails.
func detectGitBranch(path string) string {
	repo, err := git.PlainOpen(path)
	if err != nil {
		// Try parent directories (path might be inside repo)
		for parent := filepath.Dir(path); parent != "/" && parent != "."; parent = filepath.Dir(parent) {
			repo, err = git.PlainOpen(parent)
			if err == nil {
				break
			}
		}
		if err != nil {
			return "unknown"
		}
	}

	head, err := repo.Head()
	if err != nil {
		return "unknown"
	}

	// Get branch name from reference
	if head.Name().IsBranch() {
		return head.Name().Short()
	}

	// Detached HEAD - try to find branch name
	if head.Type() == plumbing.HashReference {
		// Return short hash for detached HEAD
		return head.Hash().String()[:8]
	}

	return "unknown"
}

// sanitizeProjectName converts a project name to a valid collection name component.
// Converts to lowercase, replaces invalid chars with underscores.
func sanitizeProjectName(name string) string {
	name = strings.ToLower(name)
	var result strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
			result.WriteRune(r)
		} else {
			result.WriteRune('_')
		}
	}
	// Trim leading/trailing underscores and collapse multiple underscores
	s := result.String()
	for strings.Contains(s, "__") {
		s = strings.ReplaceAll(s, "__", "_")
	}
	s = strings.Trim(s, "_")
	if s == "" {
		s = "project"
	}
	return s
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
