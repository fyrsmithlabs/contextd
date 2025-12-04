// Package vectorstore provides vector storage implementations.
package vectorstore

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	chromago "github.com/amikos-tech/chroma-go"
	"github.com/amikos-tech/chroma-go/types"
	"go.uber.org/zap"
)

// ChromaStore implements the Store interface using Chroma.
type ChromaStore struct {
	client *chromago.Client
	config ChromaConfig
	logger *zap.Logger
}

// ChromaConfig holds configuration for Chroma vector database.
type ChromaConfig struct {
	Path      string // SQLite database path
	Model     string // Embedding model name
	Dimension int    // Embedding dimension
	Distance  string // Distance metric: "cosine", "l2", "ip"
}

// modelDimensions maps Chroma models to their expected dimensions.
var modelDimensions = map[string]int{
	"sentence-transformers/all-MiniLM-L6-v2":     384,
	"sentence-transformers/all-mpnet-base-v2":    768,
	"sentence-transformers/all-roberta-large-v1": 1024,
}

// validateConfig validates Chroma configuration.
func validateConfig(config ChromaConfig) error {
	// Check model is supported
	expectedDim, exists := modelDimensions[config.Model]
	if !exists {
		return fmt.Errorf("unsupported model: %s", config.Model)
	}

	// Check dimension matches model
	if config.Dimension != expectedDim {
		return fmt.Errorf("dimension mismatch: model %s requires %d dimensions, got %d",
			config.Model, expectedDim, config.Dimension)
	}

	// Check distance metric
	switch config.Distance {
	case "cosine", "l2", "ip":
		// Valid
	default:
		return fmt.Errorf("unsupported distance metric: %s (must be cosine, l2, or ip)", config.Distance)
	}

	return nil
}

// NewChromaStore creates a new ChromaStore with the given configuration.
func NewChromaStore(config ChromaConfig, logger *zap.Logger) (*ChromaStore, error) {
	// Validate configuration
	if err := validateConfig(config); err != nil {
		return nil, fmt.Errorf("invalid chroma config: %w", err)
	}

	// Expand path
	expandedPath, err := expandPath(config.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to expand path %s: %w", config.Path, err)
	}

	// Create Chroma client with path-based storage
	client, err := chromago.NewClient(chromago.WithPath(expandedPath))
	if err != nil {
		return nil, fmt.Errorf("failed to create chroma client: %w", err)
	}

	store := &ChromaStore{
		client: client,
		config: config,
		logger: logger,
	}

	logger.Info("ChromaStore initialized",
		zap.String("path", expandedPath),
		zap.String("model", config.Model),
		zap.Int("dimension", config.Dimension),
		zap.String("distance", config.Distance),
	)

	return store, nil
}

// expandPath expands ~ to home directory.
func expandPath(path string) (string, error) {
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, path[1:]), nil
	}
	return path, nil
}

// AddDocuments adds documents to the vector store.
func (s *ChromaStore) AddDocuments(ctx context.Context, docs []Document) ([]string, error) {
	if len(docs) == 0 {
		return nil, ErrEmptyDocuments
	}

	collectionName := "default" // Use default collection for now
	collection, err := s.client.GetOrCreateCollection(ctx, collectionName, nil, true, s.config.Model, types.DistanceFunction(s.config.Distance))
	if err != nil {
		return nil, fmt.Errorf("failed to get/create collection: %w", err)
	}

	var ids []string
	var documents []string
	var metadatas []map[string]interface{}

	for _, doc := range docs {
		ids = append(ids, doc.ID)
		documents = append(documents, doc.Content)

		// Convert metadata
		metadata := make(map[string]interface{})
		for k, v := range doc.Metadata {
			metadata[k] = v
		}
		metadatas = append(metadatas, metadata)
	}

	_, err = collection.Add(ctx, nil, metadatas, documents, ids)
	if err != nil {
		return nil, fmt.Errorf("failed to add documents: %w", err)
	}

	s.logger.Debug("added documents to chroma",
		zap.String("collection", collectionName),
		zap.Int("count", len(docs)),
	)

	return ids, nil
}

// Search performs similarity search.
func (s *ChromaStore) Search(ctx context.Context, query string, k int) ([]SearchResult, error) {
	return s.searchInCollection(ctx, "default", query, k, nil)
}

// SearchWithFilters performs similarity search with metadata filters.
func (s *ChromaStore) SearchWithFilters(ctx context.Context, query string, k int, filters map[string]interface{}) ([]SearchResult, error) {
	return s.searchInCollection(ctx, "default", query, k, filters)
}

// SearchInCollection performs similarity search in a specific collection.
func (s *ChromaStore) SearchInCollection(ctx context.Context, collectionName string, query string, k int, filters map[string]interface{}) ([]SearchResult, error) {
	return s.searchInCollection(ctx, collectionName, query, k, filters)
}

// searchInCollection is the internal search implementation.
func (s *ChromaStore) searchInCollection(ctx context.Context, collectionName string, query string, k int, filters map[string]interface{}) ([]SearchResult, error) {
	collection, err := s.client.GetCollection(ctx, collectionName)
	if err != nil {
		return nil, fmt.Errorf("failed to get collection %s: %w", collectionName, err)
	}

	// Convert filters to Chroma format
	var where map[string]interface{}
	if filters != nil {
		where = filters
	}

	results, err := collection.Query(ctx, []string{query}, int32(k), where, nil, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to query collection: %w", err)
	}

	var searchResults []SearchResult
	if len(results) > 0 && len(results[0].Documents) > 0 {
		for i, doc := range results[0].Documents {
			var metadata map[string]interface{}
			if i < len(results[0].Metadatas) {
				metadata = results[0].Metadatas[i]
			}

			var score float32
			if i < len(results[0].Distances) {
				score = float32(results[0].Distances[i])
			}

			var id string
			if i < len(results[0].Ids) {
				id = results[0].Ids[i]
			}

			searchResults = append(searchResults, SearchResult{
				Document: Document{
					ID:       id,
					Content:  doc,
					Metadata: metadata,
				},
				Score: score,
			})
		}
	}

	s.logger.Debug("searched chroma collection",
		zap.String("collection", collectionName),
		zap.String("query", query),
		zap.Int("limit", k),
		zap.Int("results", len(searchResults)),
	)

	return searchResults, nil
}

// DeleteDocuments deletes documents by their IDs.
func (s *ChromaStore) DeleteDocuments(ctx context.Context, ids []string) error {
	return s.deleteDocumentsFromCollection(ctx, "default", ids)
}

// DeleteDocumentsFromCollection deletes documents by their IDs from a specific collection.
func (s *ChromaStore) DeleteDocumentsFromCollection(ctx context.Context, collectionName string, ids []string) error {
	return s.deleteDocumentsFromCollection(ctx, collectionName, ids)
}

// deleteDocumentsFromCollection is the internal delete implementation.
func (s *ChromaStore) deleteDocumentsFromCollection(ctx context.Context, collectionName string, ids []string) error {
	collection, err := s.client.GetCollection(ctx, collectionName)
	if err != nil {
		return fmt.Errorf("failed to get collection %s: %w", collectionName, err)
	}

	_, err = collection.Delete(ctx, ids, nil, nil)
	if err != nil {
		return fmt.Errorf("failed to delete documents: %w", err)
	}

	s.logger.Debug("deleted documents from chroma",
		zap.String("collection", collectionName),
		zap.Int("count", len(ids)),
	)

	return nil
}

// CreateCollection creates a new collection.
func (s *ChromaStore) CreateCollection(ctx context.Context, collectionName string, vectorSize int) error {
	// Validate vector size matches our embedding dimension
	if vectorSize != s.config.Dimension {
		return fmt.Errorf("vector size %d does not match embedding dimension %d", vectorSize, s.config.Dimension)
	}

	_, err := s.client.GetOrCreateCollection(ctx, collectionName, nil, true, s.config.Model, types.DistanceFunction(s.config.Distance))
	if err != nil {
		return fmt.Errorf("failed to create collection: %w", err)
	}

	s.logger.Info("created chroma collection",
		zap.String("collection", collectionName),
		zap.Int("vector_size", vectorSize),
	)

	return nil
}

// DeleteCollection deletes a collection.
func (s *ChromaStore) DeleteCollection(ctx context.Context, collectionName string) error {
	_, err := s.client.DeleteCollection(ctx, collectionName)
	if err != nil {
		return fmt.Errorf("failed to delete collection: %w", err)
	}

	s.logger.Info("deleted chroma collection",
		zap.String("collection", collectionName),
	)

	return nil
}

// CollectionExists checks if a collection exists.
func (s *ChromaStore) CollectionExists(ctx context.Context, collectionName string) (bool, error) {
	_, err := s.client.GetCollection(ctx, collectionName)
	if err != nil {
		// Chroma returns an error if collection doesn't exist
		return false, nil
	}
	return true, nil
}

// ListCollections returns all collection names.
func (s *ChromaStore) ListCollections(ctx context.Context) ([]string, error) {
	collections, err := s.client.ListCollections(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list collections: %w", err)
	}

	names := make([]string, len(collections))
	for i, col := range collections {
		names[i] = col.Name
	}

	return names, nil
}

// GetCollectionInfo returns information about a collection.
func (s *ChromaStore) GetCollectionInfo(ctx context.Context, collectionName string) (*CollectionInfo, error) {
	collection, err := s.client.GetCollection(ctx, collectionName)
	if err != nil {
		return nil, ErrCollectionNotFound
	}

	// Get document count (approximate)
	count, err := collection.Count(ctx)
	if err != nil {
		s.logger.Warn("failed to get collection count",
			zap.String("collection", collectionName),
			zap.Error(err),
		)
		count = 0
	}

	return &CollectionInfo{
		Name:       collectionName,
		PointCount: int(count),
		VectorSize: s.config.Dimension,
	}, nil
}

// ExactSearch performs exact similarity search.
func (s *ChromaStore) ExactSearch(ctx context.Context, collectionName string, query string, k int) ([]SearchResult, error) {
	// Chroma doesn't have a separate exact search, so use regular search
	return s.SearchInCollection(ctx, collectionName, query, k, nil)
}

// Close closes the Chroma client.
func (s *ChromaStore) Close() error {
	// Chroma client doesn't have a close method
	s.logger.Info("chroma store closed")
	return nil
}
