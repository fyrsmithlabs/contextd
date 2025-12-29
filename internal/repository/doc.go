// Package repository provides repository indexing functionality for semantic code search.
//
// The package walks file trees, filters files based on patterns and size limits,
// and creates searchable checkpoints for each indexed file. This enables semantic
// search over codebases using vector embeddings.
//
// # Security
//
// The package implements defense-in-depth security:
//   - Path traversal prevention via filepath.Clean()
//   - File size limits (1MB default, 10MB maximum)
//   - Glob pattern validation
//   - Multi-tenant isolation via project-scoped checkpoints
//   - Binary file detection (skips invalid UTF-8)
//
// # Usage
//
// Basic indexing example:
//
//	svc := repository.NewService(checkpointService)
//	opts := repository.IndexOptions{
//	    IncludePatterns: []string{"*.go", "*.md"},
//	    ExcludePatterns: []string{"vendor/**", "*_test.go"},
//	    MaxFileSize:     1024 * 1024, // 1MB
//	}
//	result, err := svc.IndexRepository(ctx, "/path/to/repo", opts)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Indexed %d files\n", result.FilesIndexed)
//
// # Pattern Matching
//
// Include patterns specify which files to index. If empty, all files are included
// (subject to exclude patterns). Exclude patterns take precedence over include patterns.
//
// Patterns use Go's filepath.Match syntax:
//   - "*.go" matches all Go files in the current directory
//   - "*_test.go" matches all test files
//   - "vendor/**" matches the vendor directory recursively (custom handling)
//
// # Performance
//
// Current implementation uses sequential file walking with one checkpoint per file.
// Future optimizations planned:
//   - Batch embedding generation (10x speedup)
//   - Parallel processing with worker pools (20x speedup)
//   - Incremental indexing (skip unchanged files)
package repository
