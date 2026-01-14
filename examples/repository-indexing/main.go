// Package main demonstrates repository indexing and semantic code search in contextd.
//
// This example shows how to:
// 1. Index a code repository for semantic search
// 2. Search indexed code using natural language queries
// 3. Use include/exclude patterns for selective indexing
// 4. Fall back to grep when semantic search doesn't find results
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/fyrsmithlabs/contextd/internal/embeddings"
	"github.com/fyrsmithlabs/contextd/internal/repository"
	"github.com/fyrsmithlabs/contextd/internal/vectorstore"
	"go.uber.org/zap"
)

const (
	tenantID = "demo-user"
)

func main() {
	fmt.Println("Repository Indexing Example - Semantic Code Search")
	fmt.Println("===================================================\n")

	// Initialize components
	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatal(err)
	}
	defer logger.Sync()

	ctx := context.Background()

	// Create embeddings provider
	embedder, err := embeddings.NewProvider(embeddings.ProviderConfig{
		Provider: "fastembed",
		Model:    "BAAI/bge-small-en-v1.5",
		CacheDir: "/tmp/fastembed-cache",
	})
	if err != nil {
		log.Fatalf("Failed to create embedder: %v", err)
	}

	// Create in-memory vector store for demo
	store, err := vectorstore.NewChromemStore(vectorstore.ChromemConfig{
		Path:              "", // Empty path = in-memory
		DefaultCollection: "codebase",
		VectorSize:        384,
	}, embedder, logger)
	if err != nil {
		log.Fatalf("Failed to create vector store: %v", err)
	}

	// Create repository service
	service := repository.NewService(store)

	// Get the path to index (use current project as example)
	repoPath := getRepositoryPath()

	// Run the repository indexing demo
	if err := runIndexingDemo(ctx, service, repoPath); err != nil {
		log.Fatalf("Repository indexing demo failed: %v", err)
	}

	fmt.Println("\n✓ Repository indexed and searchable!")
}

// getRepositoryPath determines which repository to index for the demo.
// Defaults to current directory, or you can set REPO_PATH env variable.
func getRepositoryPath() string {
	if path := os.Getenv("REPO_PATH"); path != "" {
		return path
	}

	// Try to find the contextd project root (go up from examples/repository-indexing)
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get current directory: %v", err)
	}

	// Go up two directories to reach project root
	projectRoot := filepath.Join(cwd, "..", "..")
	absPath, err := filepath.Abs(projectRoot)
	if err != nil {
		log.Fatalf("Failed to resolve project root: %v", err)
	}

	return absPath
}

// runIndexingDemo demonstrates repository indexing and semantic search.
func runIndexingDemo(ctx context.Context, service *repository.Service, repoPath string) error {
	fmt.Printf("Repository Path: %s\n\n", repoPath)

	// Step 1: Index the repository with selective patterns
	fmt.Println("Step 1: Indexing repository...")
	fmt.Println("This may take a moment for large repositories.\n")

	opts := repository.IndexOptions{
		TenantID: tenantID,
		// Index only Go source files and docs
		IncludePatterns: []string{"*.go", "*.md"},
		// Exclude test files, vendor, and build artifacts
		ExcludePatterns: []string{
			"*_test.go",
			"vendor/**",
			".git/**",
			"node_modules/**",
			"dist/**",
			"build/**",
		},
		MaxFileSize: 1024 * 1024, // 1MB max file size
	}

	result, err := service.IndexRepository(ctx, repoPath, opts)
	if err != nil {
		return fmt.Errorf("indexing repository: %w", err)
	}

	fmt.Printf("✓ Indexed %d files from branch '%s'\n", result.FilesIndexed, result.Branch)
	fmt.Printf("  Collection: %s\n", result.CollectionName)
	fmt.Printf("  Include patterns: %v\n", result.IncludePatterns)
	fmt.Printf("  Exclude patterns: %v\n", result.ExcludePatterns)
	fmt.Printf("  Max file size: %d bytes\n\n", result.MaxFileSize)

	// Step 2: Perform semantic search queries
	fmt.Println("Step 2: Performing semantic searches...\n")

	// Query 1: Find vector store implementation
	query1 := "vector database implementation with embeddings"
	if err := searchAndDisplay(ctx, service, repoPath, query1); err != nil {
		return err
	}

	// Query 2: Find error handling patterns
	query2 := "error wrapping and handling patterns"
	if err := searchAndDisplay(ctx, service, repoPath, query2); err != nil {
		return err
	}

	// Query 3: Find MCP tool handlers
	query3 := "MCP tool registration and handlers"
	if err := searchAndDisplay(ctx, service, repoPath, query3); err != nil {
		return err
	}

	// Step 3: Demonstrate grep fallback
	fmt.Println("\nStep 3: Demonstrating grep fallback for exact matches...\n")

	grepPattern := `func.*Index.*Repository`
	grepOpts := repository.GrepOptions{
		ProjectPath:     repoPath,
		IncludePatterns: []string{"*.go"},
		ExcludePatterns: opts.ExcludePatterns,
		CaseSensitive:   false,
	}

	grepResults, err := service.Grep(ctx, grepPattern, grepOpts)
	if err != nil {
		return fmt.Errorf("grep search: %w", err)
	}

	fmt.Printf("Grep pattern: %s\n", grepPattern)
	fmt.Printf("Found %d matches:\n", len(grepResults))
	displayCount := 3
	if len(grepResults) < displayCount {
		displayCount = len(grepResults)
	}
	for i := 0; i < displayCount; i++ {
		result := grepResults[i]
		fmt.Printf("\n  File: %s:%d\n", result.FilePath, result.LineNumber)
		fmt.Printf("  Code: %s\n", truncate(result.Content, 80))
	}
	if len(grepResults) > displayCount {
		fmt.Printf("  ... and %d more matches\n", len(grepResults)-displayCount)
	}

	return nil
}

// searchAndDisplay performs a semantic search and displays results.
func searchAndDisplay(ctx context.Context, service *repository.Service, repoPath, query string) error {
	fmt.Printf("Query: \"%s\"\n", query)

	searchOpts := repository.SearchOptions{
		ProjectPath: repoPath,
		TenantID:    tenantID,
		Limit:       5,
	}

	results, err := service.Search(ctx, query, searchOpts)
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}

	if len(results) == 0 {
		fmt.Println("No results found.\n")
		return nil
	}

	fmt.Printf("Found %d results:\n", len(results))
	for i, result := range results {
		if i >= 3 {
			break // Show top 3 results
		}
		fmt.Printf("\n  %d. %s (score: %.3f)\n", i+1, result.FilePath, result.Score)
		fmt.Printf("     Branch: %s\n", result.Branch)
		// Show first few lines of content
		preview := truncate(result.Content, 150)
		fmt.Printf("     Preview: %s...\n", preview)
	}
	if len(results) > 3 {
		fmt.Printf("  ... and %d more results\n", len(results)-3)
	}
	fmt.Println()

	return nil
}

// truncate truncates a string to maxLen characters.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
