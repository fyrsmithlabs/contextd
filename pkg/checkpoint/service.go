package checkpoint

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/fyrsmithlabs/contextd/pkg/vectorstore"
)

// VectorStore defines the interface for vector storage operations.
//
// This interface allows for testing and decouples the service from
// the specific vector store implementation.
type VectorStore interface {
	AddDocuments(ctx context.Context, docs []vectorstore.Document) error
	SearchWithFilters(ctx context.Context, query string, k int, filters map[string]interface{}) ([]vectorstore.SearchResult, error)
}

// Service provides checkpoint management with vector storage and semantic search.
//
// The service uses a vector store for persistence and enables semantic search
// across checkpoints using embeddings. All operations are scoped to project
// paths for multi-tenant isolation.
type Service struct {
	vectorStore VectorStore
	logger      *zap.Logger
}

// NewService creates a new checkpoint service.
//
// The vector store must be configured with an embedder for automatic
// embedding generation. The service will use the vectorstore's collection
// naming scheme for multi-tenant isolation.
func NewService(vs VectorStore, logger *zap.Logger) *Service {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Service{
		vectorStore: vs,
		logger:      logger,
	}
}

// Save stores a checkpoint with automatic embedding generation.
//
// The checkpoint is validated before saving. A unique ID is generated if not
// provided. Timestamps are set automatically. The checkpoint content and
// summary are embedded for semantic search.
//
// Returns the checkpoint ID on success, or an error if validation or storage fails.
func (s *Service) Save(ctx context.Context, cp *Checkpoint) error {
	// Validate checkpoint
	if err := cp.Validate(); err != nil {
		return fmt.Errorf("validating checkpoint: %w", err)
	}

	// Generate ID if not provided
	if cp.ID == "" {
		cp.ID = uuid.New().String()
	}

	// Set timestamps
	now := time.Now()
	if cp.CreatedAt.IsZero() {
		cp.CreatedAt = now
	}
	cp.UpdatedAt = now

	// Build content for embedding (summary + content)
	embedContent := cp.Summary
	if cp.Content != "" {
		embedContent = fmt.Sprintf("%s\n\n%s", cp.Summary, cp.Content)
	}

	// Prepare metadata for vector store
	metadata := map[string]interface{}{
		"id":           cp.ID,
		"project_path": cp.ProjectPath,
		"project_hash": projectHash(cp.ProjectPath),
		"summary":      cp.Summary,
		"created_at":   cp.CreatedAt.Format(time.RFC3339),
		"updated_at":   cp.UpdatedAt.Format(time.RFC3339),
	}

	// Add user metadata
	for k, v := range cp.Metadata {
		metadata[k] = v
	}

	// Add tags
	if len(cp.Tags) > 0 {
		metadata["tags"] = cp.Tags
	}

	// Add full content to metadata (not embedded, but stored for retrieval)
	if cp.Content != "" {
		metadata["content"] = cp.Content
	}

	// Create vector store document
	doc := vectorstore.Document{
		ID:       cp.ID,
		Content:  embedContent,
		Metadata: metadata,
	}

	// Store in vector database (embedding generated automatically)
	if err := s.vectorStore.AddDocuments(ctx, []vectorstore.Document{doc}); err != nil {
		s.logger.Error("Failed to store checkpoint",
			zap.Error(err),
			zap.String("checkpoint_id", cp.ID),
			zap.String("project", cp.ProjectPath))
		return fmt.Errorf("storing checkpoint: %w", err)
	}

	s.logger.Info("Checkpoint saved",
		zap.String("checkpoint_id", cp.ID),
		zap.String("project", cp.ProjectPath),
		zap.String("summary", cp.Summary))

	return nil
}

// Search finds semantically similar checkpoints using vector similarity.
//
// The search query is embedded and compared against all checkpoints in the
// project. Results are filtered by minimum score and limited to the specified
// number of results.
//
// Returns checkpoints ordered by similarity score (highest first).
func (s *Service) Search(ctx context.Context, query string, opts *SearchOptions) ([]*SearchResult, error) {
	if query == "" {
		return nil, errors.New("query cannot be empty")
	}

	// Validate options
	if opts == nil {
		opts = &SearchOptions{}
	}
	if err := opts.Validate(); err != nil {
		return nil, fmt.Errorf("invalid search options: %w", err)
	}

	// Build filters for project-scoped search using Qdrant filter structure
	filters := map[string]interface{}{
		"must": []map[string]interface{}{
			{
				"key": "project_hash",
				"match": map[string]interface{}{
					"value": projectHash(opts.ProjectPath),
				},
			},
		},
	}

	// TODO: Add tag filtering when needed
	// Tag filtering will be implemented in memory after retrieval
	_ = opts.Tags // Avoid unused variable warning

	// Execute semantic search
	results, err := s.vectorStore.SearchWithFilters(ctx, query, opts.Limit, filters)
	if err != nil {
		s.logger.Error("Checkpoint search failed",
			zap.Error(err),
			zap.String("project", opts.ProjectPath),
			zap.String("query", query))
		return nil, fmt.Errorf("searching checkpoints: %w", err)
	}

	// Convert results to checkpoints
	searchResults := make([]*SearchResult, 0, len(results))
	for _, result := range results {
		// Filter by minimum score
		if result.Score < opts.MinScore {
			continue
		}

		checkpoint, err := resultToCheckpoint(result)
		if err != nil {
			s.logger.Warn("Failed to convert search result",
				zap.Error(err),
				zap.String("result_id", result.ID))
			continue
		}

		// Filter by tags if specified (in-memory filtering)
		if len(opts.Tags) > 0 && !hasAnyTag(checkpoint.Tags, opts.Tags) {
			continue
		}

		searchResults = append(searchResults, &SearchResult{
			Checkpoint: checkpoint,
			Score:      result.Score,
		})
	}

	s.logger.Debug("Checkpoint search completed",
		zap.String("project", opts.ProjectPath),
		zap.Int("results", len(searchResults)),
		zap.String("query", query))

	return searchResults, nil
}

// List retrieves recent checkpoints with pagination.
//
// Checkpoints are returned in reverse chronological order (newest first).
// Results are scoped to the specified project path for multi-tenant isolation.
//
// Note: This is a placeholder implementation. Full implementation would require
// a list/scan method on the vector store interface.
func (s *Service) List(ctx context.Context, opts *ListOptions) ([]*Checkpoint, error) {
	// Validate options
	if opts == nil {
		opts = &ListOptions{}
	}
	if err := opts.Validate(); err != nil {
		return nil, fmt.Errorf("invalid list options: %w", err)
	}

	// For 0.9.0-rc-1, we implement List as a search with empty query
	// This leverages the vector store's ability to return documents
	// The empty query will match all documents in the collection
	searchOpts := &SearchOptions{
		ProjectPath: opts.ProjectPath,
		Limit:       opts.Limit,
		MinScore:    0.0, // Accept all scores for listing
		Tags:        opts.Tags,
	}

	// Use a generic query that will match most checkpoints
	results, err := s.Search(ctx, "checkpoint", searchOpts)
	if err != nil {
		return nil, fmt.Errorf("listing checkpoints: %w", err)
	}

	// Convert to checkpoint list (discard scores)
	checkpoints := make([]*Checkpoint, len(results))
	for i, result := range results {
		checkpoints[i] = result.Checkpoint
	}

	s.logger.Debug("Checkpoint list completed",
		zap.String("project", opts.ProjectPath),
		zap.Int("count", len(checkpoints)))

	return checkpoints, nil
}

// Get retrieves a specific checkpoint by ID.
//
// The project path must match the checkpoint's project for security.
// Returns ErrNotFound if the checkpoint doesn't exist.
func (s *Service) Get(ctx context.Context, projectPath, id string) (*Checkpoint, error) {
	if projectPath == "" {
		return nil, ErrProjectPathRequired
	}
	if id == "" {
		return nil, errors.New("checkpoint ID is required")
	}

	// Search by ID with project filter using Qdrant filter structure
	// This is a workaround since vectorstore doesn't have a Get method
	filters := map[string]interface{}{
		"must": []map[string]interface{}{
			{
				"key": "project_hash",
				"match": map[string]interface{}{
					"value": projectHash(projectPath),
				},
			},
			{
				"key": "id",
				"match": map[string]interface{}{
					"value": id,
				},
			},
		},
	}

	results, err := s.vectorStore.SearchWithFilters(ctx, id, 1, filters)
	if err != nil {
		return nil, fmt.Errorf("getting checkpoint: %w", err)
	}

	if len(results) == 0 {
		return nil, errors.New("checkpoint not found")
	}

	checkpoint, err := resultToCheckpoint(results[0])
	if err != nil {
		return nil, fmt.Errorf("converting checkpoint: %w", err)
	}

	s.logger.Debug("Checkpoint retrieved",
		zap.String("checkpoint_id", id),
		zap.String("project", projectPath))

	return checkpoint, nil
}

// Helper functions

// projectHash generates a SHA256 hash of the project path for collection naming.
func projectHash(projectPath string) string {
	hash := sha256.Sum256([]byte(projectPath))
	return hex.EncodeToString(hash[:])[:16] // Use first 16 chars for brevity
}

// resultToCheckpoint converts a vector store search result to a Checkpoint.
func resultToCheckpoint(result vectorstore.SearchResult) (*Checkpoint, error) {
	cp := &Checkpoint{
		Metadata: make(map[string]interface{}),
	}

	// Extract required fields
	id, ok := result.Metadata["id"].(string)
	if !ok {
		return nil, errors.New("missing or invalid ID in result")
	}
	cp.ID = id

	projectPath, ok := result.Metadata["project_path"].(string)
	if !ok {
		return nil, errors.New("missing or invalid project_path in result")
	}
	cp.ProjectPath = projectPath

	summary, ok := result.Metadata["summary"].(string)
	if !ok {
		return nil, errors.New("missing or invalid summary in result")
	}
	cp.Summary = summary

	// Parse timestamps
	if createdStr, ok := result.Metadata["created_at"].(string); ok {
		if t, err := time.Parse(time.RFC3339, createdStr); err == nil {
			cp.CreatedAt = t
		}
	}
	if updatedStr, ok := result.Metadata["updated_at"].(string); ok {
		if t, err := time.Parse(time.RFC3339, updatedStr); err == nil {
			cp.UpdatedAt = t
		}
	}

	// Extract optional content
	if content, ok := result.Metadata["content"].(string); ok {
		cp.Content = content
	}

	// Extract tags
	if tags, ok := result.Metadata["tags"].([]interface{}); ok {
		cp.Tags = make([]string, 0, len(tags))
		for _, tag := range tags {
			if tagStr, ok := tag.(string); ok {
				cp.Tags = append(cp.Tags, tagStr)
			}
		}
	}

	// Copy all metadata (excluding system fields)
	for k, v := range result.Metadata {
		if k != "id" && k != "project_path" && k != "project_hash" && k != "summary" &&
			k != "created_at" && k != "updated_at" && k != "content" && k != "tags" {
			cp.Metadata[k] = v
		}
	}

	return cp, nil
}

// hasAnyTag checks if the checkpoint has any of the specified tags.
func hasAnyTag(checkpointTags, searchTags []string) bool {
	tagSet := make(map[string]bool)
	for _, tag := range checkpointTags {
		tagSet[tag] = true
	}
	for _, tag := range searchTags {
		if tagSet[tag] {
			return true
		}
	}
	return false
}
