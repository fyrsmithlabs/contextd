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
	SearchInCollection(ctx context.Context, collectionName string, query string, k int, filters map[string]interface{}) ([]vectorstore.SearchResult, error)
	GetCollectionInfo(ctx context.Context, collectionName string) (*vectorstore.CollectionInfo, error)
	ExactSearch(ctx context.Context, collectionName string, query string, k int) ([]vectorstore.SearchResult, error)
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

	// Auto-detect git branch if not provided
	if cp.Branch == "" {
		if branch, err := detectGitBranch(cp.ProjectPath); err == nil && branch != "" {
			cp.Branch = branch
			s.logger.Debug("Auto-detected git branch",
				zap.String("project", cp.ProjectPath),
				zap.String("branch", branch))
		}
	}

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

	// Add branch
	if cp.Branch != "" {
		metadata["branch"] = cp.Branch
	}

	// Add full content to metadata (not embedded, but stored for retrieval)
	if cp.Content != "" {
		metadata["content"] = cp.Content
	}

	// Compute project-specific collection name (fixes BUG-2025-11-20-004 ROOT CAUSE #2)
	collectionName := fmt.Sprintf("project_%s__checkpoints", projectHash(cp.ProjectPath))

	// Create vector store document
	doc := vectorstore.Document{
		ID:         cp.ID,
		Content:    embedContent,
		Metadata:   metadata,
		Collection: collectionName, // Use project-specific collection
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

	// Compute project-specific collection name (fixes BUG-2025-11-20-004 ROOT CAUSE #2)
	collectionName := fmt.Sprintf("project_%s__checkpoints", projectHash(opts.ProjectPath))

	// NOTE: No filters needed - collection-based isolation provides project scoping
	// The project-specific collection name already ensures we only search within the project
	// Filters will be used later for tag filtering if needed
	var filters map[string]interface{}

	// TODO: Add tag filtering when needed
	// Tag filtering will be implemented in memory after retrieval
	_ = opts.Tags // Avoid unused variable warning

	// BUG-2025-11-20-005: Check collection size for small dataset fix
	// Qdrant requires ≥10 vectors for HNSW index to work
	// For <10 vectors, use exact search fallback (brute force cosine similarity)
	var results []vectorstore.SearchResult
	var err error

	collectionInfo, err := s.vectorStore.GetCollectionInfo(ctx, collectionName)
	if err != nil {
		s.logger.Warn("Failed to get collection info, falling back to standard search",
			zap.Error(err),
			zap.String("collection", collectionName))
		// If we can't get collection info, try standard search anyway
		results, err = s.vectorStore.SearchInCollection(ctx, collectionName, query, opts.Limit, filters)
	} else if collectionInfo.PointCount < 10 {
		// Small dataset: Use exact search fallback
		s.logger.Debug("Using exact search fallback for small dataset",
			zap.String("collection", collectionName),
			zap.Int("point_count", collectionInfo.PointCount))

		// Use exact search (brute force cosine similarity) for <10 vectors
		// This is slower but works without HNSW index
		results, err = s.vectorStore.ExactSearch(ctx, collectionName, query, opts.Limit)
	} else {
		// Normal dataset: Use HNSW indexed search
		results, err = s.vectorStore.SearchInCollection(ctx, collectionName, query, opts.Limit, filters)
	}

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

		// Filter by branch if specified (in-memory filtering)
		if opts.Branch != "" && checkpoint.Branch != opts.Branch {
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

	// Compute project-specific collection name (fixes BUG-2025-11-20-004 ROOT CAUSE #2)
	collectionName := fmt.Sprintf("project_%s__checkpoints", projectHash(projectPath))

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

	results, err := s.vectorStore.SearchInCollection(ctx, collectionName, id, 1, filters)
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
	return hex.EncodeToString(hash[:])[:8] // Use first 8 chars to match existing convention
}

// getOrCreateCollectionName resolves collection name with collision detection.
//
// Algorithm:
// 1. Generate base hash: project_<hash>__<type>
// 2. Check if collection exists
// 3. If exists, verify project_path matches (check sample document)
// 4. If collision (different project), append _01, _02, etc.
// 5. Limit: 100 collision attempts
//
// Returns collection name or error if collision limit exceeded.
func (s *Service) getOrCreateCollectionName(ctx context.Context, projectPath, collectionType string) (string, error) {
	baseHash := projectHash(projectPath)
	maxAttempts := 100

	for attempt := 0; attempt < maxAttempts; attempt++ {
		var collectionName string
		if attempt == 0 {
			collectionName = fmt.Sprintf("project_%s__%s", baseHash, collectionType)
		} else {
			collectionName = fmt.Sprintf("project_%s_%02d__%s", baseHash, attempt, collectionType)
		}

		// Check if collection exists
		info, err := s.vectorStore.GetCollectionInfo(ctx, collectionName)
		if err != nil {
			// Collection doesn't exist - we can use this name
			if errors.Is(err, vectorstore.ErrCollectionNotFound) {
				s.logger.Debug("Collection available",
					zap.String("collection", collectionName),
					zap.String("project", projectPath),
					zap.Int("attempt", attempt))
				return collectionName, nil
			}
			// Other error - return it
			return "", fmt.Errorf("checking collection info: %w", err)
		}

		// Collection exists - verify it's for the same project
		if info.PointCount == 0 {
			// Empty collection - assume it's ours (edge case: abandoned collection)
			s.logger.Warn("Found empty collection, assuming ownership",
				zap.String("collection", collectionName),
				zap.String("project", projectPath))
			return collectionName, nil
		}

		// Query a sample document to check project_path
		// Use SearchInCollection with limit=1 to get any document
		results, err := s.vectorStore.SearchInCollection(ctx, collectionName, projectPath, 1, nil)
		if err != nil {
			// If search fails, try next collision suffix
			s.logger.Debug("Failed to query collection for verification",
				zap.Error(err),
				zap.String("collection", collectionName))
			continue
		}

		if len(results) > 0 {
			// Check if the document belongs to this project
			if storedPath, ok := results[0].Metadata["project_path"].(string); ok {
				if storedPath == projectPath {
					// Same project - reuse this collection
					s.logger.Debug("Reusing existing collection",
						zap.String("collection", collectionName),
						zap.String("project", projectPath))
					return collectionName, nil
				}
			}
		}

		// Collision detected - try next suffix
		s.logger.Info("Hash collision detected, trying next suffix",
			zap.String("collection", collectionName),
			zap.String("project", projectPath),
			zap.Int("attempt", attempt))
	}

	// Exceeded collision limit
	return "", fmt.Errorf("exceeded collision limit (%d attempts) for project %s", maxAttempts, projectPath)
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

	// Extract branch
	if branch, ok := result.Metadata["branch"].(string); ok {
		cp.Branch = branch
	}

	// Copy all metadata (excluding system fields)
	for k, v := range result.Metadata {
		if k != "id" && k != "project_path" && k != "project_hash" && k != "summary" &&
			k != "created_at" && k != "updated_at" && k != "content" && k != "tags" && k != "branch" {
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
