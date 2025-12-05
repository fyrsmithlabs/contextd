package repository

import "time"

// IndexOptions configures repository indexing behavior.
type IndexOptions struct {
	// TenantID is the tenant identifier for multi-tenant isolation.
	// If empty, uses default from git user.name or OS username.
	TenantID string

	// Branch is the git branch to associate with indexed files.
	// If empty, auto-detects current branch from repository.
	Branch string

	// IncludePatterns are glob patterns for files to include (e.g., ["*.md", "*.go"]).
	// If empty, all files are included (subject to exclude patterns and size limit).
	IncludePatterns []string

	// ExcludePatterns are glob patterns for files to exclude (e.g., ["*.log", "node_modules/**"]).
	// Takes precedence over include patterns.
	ExcludePatterns []string

	// MaxFileSize is the maximum file size in bytes to index.
	// Default: 1MB (1048576), Maximum: 10MB (10485760).
	MaxFileSize int64
}

// IndexResult contains the results of a repository indexing operation.
type IndexResult struct {
	// Path is the repository path that was indexed.
	Path string

	// Branch is the git branch that was indexed.
	Branch string

	// CollectionName is the Qdrant collection where files were stored.
	CollectionName string

	// FilesIndexed is the number of files successfully indexed.
	FilesIndexed int

	// IncludePatterns used during indexing.
	IncludePatterns []string

	// ExcludePatterns used during indexing.
	ExcludePatterns []string

	// MaxFileSize applied during indexing.
	MaxFileSize int64

	// IndexedAt is the timestamp when indexing completed.
	IndexedAt time.Time
}
