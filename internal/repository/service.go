package repository

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"

	"github.com/fyrsmithlabs/contextd/internal/sanitize"
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
	store  Store                     // Legacy single-store mode
	stores vectorstore.StoreProvider // Database-per-project isolation mode
}

// NewService creates a new repository indexing service.
func NewService(store Store) *Service {
	return &Service{
		store: store,
	}
}

// NewServiceWithStoreProvider creates a repository service using StoreProvider
// for database-per-project isolation.
//
// With StoreProvider, each project gets its own chromem.DB instance,
// and the collection name is simplified to just "codebase".
func NewServiceWithStoreProvider(stores vectorstore.StoreProvider) *Service {
	return &Service{
		stores: stores,
	}
}

// getStore returns the appropriate store and collection name for a project path.
//
// With StoreProvider: returns project-scoped store with simple "codebase" collection.
// With legacy Store: returns shared store with "{tenant}_{project}_codebase" collection.
//
// Returns (store, collectionName, tenantID, error).
func (s *Service) getStore(ctx context.Context, projectPath, tenantID string) (Store, string, string, error) {
	// Determine tenant ID if not provided
	if tenantID == "" {
		tenantID = tenant.GetTenantIDForPath(projectPath)
	}

	// Extract project name from path
	projectName := filepath.Base(projectPath)

	// Prefer StoreProvider for database-per-project isolation
	if s.stores != nil {
		// Get project-scoped store (tenant, team="", project)
		// Team is empty for direct project path
		store, err := s.stores.GetProjectStore(ctx, tenantID, "", projectName)
		if err != nil {
			return nil, "", "", fmt.Errorf("getting project store: %w", err)
		}
		// With StoreProvider, use simple collection name (database is already project-scoped)
		return store, "codebase", tenantID, nil
	}

	// Fallback to legacy store
	if s.store == nil {
		return nil, "", "", fmt.Errorf("store not configured")
	}

	// Build full collection name for shared store
	collectionName := sanitize.CollectionName(tenantID, projectName, "codebase")

	return s.store, collectionName, tenantID, nil
}

// SearchOptions configures repository search behavior.
type SearchOptions struct {
	CollectionName string // Preferred: direct collection name from repository_index
	ProjectPath    string // Required if CollectionName not provided
	TenantID       string // Required if CollectionName not provided
	Branch         string // Optional: filter by branch (empty = all branches)
	Limit          int    // Max results (default: 10)
}

// RepoSearchResult from repository search.
type RepoSearchResult struct {
	FilePath string                 `json:"file_path"`
	Content  string                 `json:"content"`
	Score    float32                `json:"score"`
	Branch   string                 `json:"branch"`
	Metadata map[string]interface{} `json:"metadata"`
}

// GrepOptions configures repository grep behavior.
type GrepOptions struct {
	ProjectPath     string
	IncludePatterns []string
	ExcludePatterns []string
	CaseSensitive   bool
}

// GrepResult from repository grep.
type GrepResult struct {
	FilePath   string `json:"file_path"`
	Content    string `json:"content"`
	LineNumber int    `json:"line_number"`
}

// Grep performs a regex search over repository files.
func (s *Service) Grep(ctx context.Context, pattern string, opts GrepOptions) ([]GrepResult, error) {
	// Validate and clean path
	cleanPath, err := validatePath(opts.ProjectPath)
	if err != nil {
		return nil, fmt.Errorf("invalid path: %w", err)
	}

	// Compile regex
	var re *regexp.Regexp
	if opts.CaseSensitive {
		re, err = regexp.Compile(pattern)
	} else {
		re, err = regexp.Compile("(?i)" + pattern)
	}
	if err != nil {
		return nil, fmt.Errorf("invalid pattern: %w", err)
	}

	// Validate patterns
	if err := validatePatterns(opts.IncludePatterns); err != nil {
		return nil, fmt.Errorf("invalid include pattern: %w", err)
	}
	if err := validatePatterns(opts.ExcludePatterns); err != nil {
		return nil, fmt.Errorf("invalid exclude pattern: %w", err)
	}

	var results []GrepResult

	// Reuse logic from IndexRepository by creating equivalent IndexOptions
	indexOpts := IndexOptions{
		IncludePatterns: opts.IncludePatterns,
		ExcludePatterns: opts.ExcludePatterns,
		MaxFileSize:     1024 * 1024, // Default 1MB limit for grep too
	}

	// Walk file tree
	err = filepath.Walk(cleanPath, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories that should not be indexed (and thus not grepped)
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

		// Get relative path
		relPath, err := filepath.Rel(cleanPath, filePath)
		if err != nil {
			return fmt.Errorf("computing relative path: %w", err)
		}

		// Apply filters
		if !shouldIncludeFile(relPath, info, indexOpts) {
			return nil
		}

		// Read file
		file, openErr := os.Open(filePath)
		if openErr != nil {
			// Skip unreadable files
			return nil
		}

		// Scan lines
		scanner := bufio.NewScanner(file)
		lineNum := 0
		for scanner.Scan() {
			lineNum++
			line := scanner.Text()

			// Skip binary checks for simplicity, but we should probably respect utf8
			if !utf8.ValidString(line) {
				continue
			}

			if re.MatchString(line) {
				results = append(results, GrepResult{
					FilePath:   relPath,
					Content:    strings.TrimSpace(line),
					LineNumber: lineNum,
				})
			}
		}

		// Check scanner error (e.g., line too long for buffer)
		scanErr := scanner.Err()

		// Close file explicitly to avoid holding handles during the rest of the walk
		// (defer in a walk callback holds all handles until walk completes)
		file.Close()

		if scanErr != nil {
			return scanErr
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("walking file tree: %w", err)
	}

	return results, nil
}

// Search performs semantic search over indexed repository files.
func (s *Service) Search(ctx context.Context, query string, opts SearchOptions) ([]RepoSearchResult, error) {
	if query == "" {
		return nil, fmt.Errorf("query cannot be empty")
	}

	limit := opts.Limit
	if limit <= 0 {
		limit = 10
	}

	// Determine store and collection name
	var store Store
	var collectionName string
	var err error

	if opts.CollectionName != "" {
		// Use provided collection name directly (legacy behavior)
		collectionName = opts.CollectionName
		if s.store == nil {
			return nil, fmt.Errorf("store not configured")
		}
		store = s.store

		// Inject tenant context even when using CollectionName
		// Require ProjectPath to derive tenant info (fail-closed)
		if opts.ProjectPath == "" {
			return nil, fmt.Errorf("project_path is required for tenant context")
		}
		tenantID := opts.TenantID
		if tenantID == "" {
			tenantID = tenant.GetTenantIDForPath(opts.ProjectPath)
		}
		projectName := filepath.Base(opts.ProjectPath)
		ctx = vectorstore.ContextWithTenant(ctx, &vectorstore.TenantInfo{
			TenantID:  sanitize.Identifier(tenantID),
			ProjectID: sanitize.Identifier(projectName),
		})
	} else {
		// Use getStore() to determine appropriate store and collection
		if opts.ProjectPath == "" {
			return nil, fmt.Errorf("project_path is required when collection_name not provided")
		}
		var tenantID string
		store, collectionName, tenantID, err = s.getStore(ctx, opts.ProjectPath, opts.TenantID)
		if err != nil {
			return nil, fmt.Errorf("getting store: %w", err)
		}

		// Inject tenant context for payload-based isolation
		projectName := filepath.Base(opts.ProjectPath)
		ctx = vectorstore.ContextWithTenant(ctx, &vectorstore.TenantInfo{
			TenantID:  sanitize.Identifier(tenantID),
			ProjectID: sanitize.Identifier(projectName),
		})
	}

	// Build filters
	filters := make(map[string]interface{})
	if opts.Branch != "" {
		filters["branch"] = opts.Branch
	}

	results, err := store.SearchInCollection(ctx, collectionName, query, limit, filters)
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

	// Detect branch (auto-detect if not specified)
	branch := opts.Branch
	if branch == "" {
		branch = detectGitBranch(cleanPath)
	}

	// Get store and collection name using getStore()
	store, collectionName, tenantID, err := s.getStore(ctx, cleanPath, opts.TenantID)
	if err != nil {
		return nil, err
	}

	// Sanitize tenant ID for metadata consistency (store what we use for lookups)
	sanitizedTenant := sanitize.Identifier(tenantID)

	// Inject tenant context for payload-based isolation
	projectName := filepath.Base(cleanPath)
	ctx = vectorstore.ContextWithTenant(ctx, &vectorstore.TenantInfo{
		TenantID:  sanitizedTenant,
		ProjectID: sanitize.Identifier(projectName),
	})

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

		// Skip empty files (embedding layer rejects empty content)
		contentStr := strings.TrimSpace(string(content))
		if contentStr == "" {
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
				"tenant_id":    sanitizedTenant, // Use sanitized for consistency with collection name
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
		if _, err := store.AddDocuments(ctx, docs); err != nil {
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
