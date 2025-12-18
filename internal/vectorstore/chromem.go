// Package vectorstore provides vector storage implementations.
package vectorstore

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	chromem "github.com/philippgille/chromem-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.uber.org/zap"
)

// timeNow is a variable for testing purposes (allows mocking time).
var timeNow = time.Now

// chromemTracer for OpenTelemetry instrumentation.
var chromemTracer = otel.Tracer("contextd.vectorstore.chromem")

// ChromemConfig holds configuration for chromem-go embedded vector database.
type ChromemConfig struct {
	// Path is the directory for persistent storage.
	// Default: "~/.config/contextd/vectorstore"
	Path string

	// Compress enables gzip compression for stored data.
	// Note: This defaults to false (Go zero value). Set explicitly if compression is desired.
	Compress bool

	// DefaultCollection is the default collection name.
	// Default: "contextd_default"
	DefaultCollection string

	// VectorSize is the expected embedding dimension.
	// Must match the embedder's output dimension.
	// Default: 384 (for FastEmbed bge-small-en-v1.5)
	VectorSize int
}

// ApplyDefaults sets default values for unset fields.
func (c *ChromemConfig) ApplyDefaults() {
	if c.Path == "" {
		c.Path = "~/.config/contextd/vectorstore"
	}
	if c.DefaultCollection == "" {
		c.DefaultCollection = "contextd_default"
	}
	if c.VectorSize == 0 {
		c.VectorSize = 384
	}
}

// Validate validates the configuration.
func (c *ChromemConfig) Validate() error {
	if c.VectorSize <= 0 {
		return fmt.Errorf("%w: vector size must be positive", ErrInvalidConfig)
	}
	return nil
}

// ChromemStore implements the Store interface using chromem-go.
//
// chromem-go is an embeddable vector database with zero third-party dependencies.
// It provides in-memory storage with optional persistence to gob files.
//
// Key features:
//   - Pure Go, no CGO required
//   - No external database service needed
//   - Fast similarity search (1000 docs in 0.3ms)
//   - Automatic persistence to disk
//   - Tenant isolation via payload filtering or filesystem isolation
type ChromemStore struct {
	db        *chromem.DB
	embedder  Embedder
	config    ChromemConfig
	logger    *zap.Logger
	isolation IsolationMode

	// collections tracks which collections have been created
	collections sync.Map
}

// NewChromemStore creates a new ChromemStore with the given configuration.
func NewChromemStore(config ChromemConfig, embedder Embedder, logger *zap.Logger) (*ChromemStore, error) {
	// Validate required dependencies
	if embedder == nil {
		return nil, fmt.Errorf("%w: embedder is required", ErrInvalidConfig)
	}
	if logger == nil {
		logger = zap.NewNop() // Use no-op logger if none provided
	}

	config.ApplyDefaults()

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("validating config: %w", err)
	}

	// Expand path
	expandedPath, err := expandChromemPath(config.Path)
	if err != nil {
		return nil, fmt.Errorf("expanding path: %w", err)
	}

	// Ensure directory exists
	if err := os.MkdirAll(expandedPath, 0755); err != nil {
		return nil, fmt.Errorf("creating directory %s: %w", expandedPath, err)
	}

	// Create persistent DB
	db, err := chromem.NewPersistentDB(expandedPath, config.Compress)
	if err != nil {
		return nil, fmt.Errorf("creating chromem DB: %w", err)
	}

	store := &ChromemStore{
		db:        db,
		embedder:  embedder,
		config:    config,
		logger:    logger,
		isolation: NewPayloadIsolation(), // Default to payload isolation for fail-closed security
	}

	logger.Info("ChromemStore initialized",
		zap.String("path", expandedPath),
		zap.Bool("compress", config.Compress),
		zap.Int("vector_size", config.VectorSize),
		zap.String("default_collection", config.DefaultCollection),
	)

	return store, nil
}

// expandChromemPath expands ~ to home directory.
func expandChromemPath(path string) (string, error) {
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, path[1:]), nil
	}
	return path, nil
}

// SetIsolationMode sets the tenant isolation mode for this store.
// Use NewPayloadIsolation() for multi-tenant payload filtering,
// or NewNoIsolation() for testing (default).
func (s *ChromemStore) SetIsolationMode(mode IsolationMode) {
	s.isolation = mode
}

// IsolationMode returns the current isolation mode.
func (s *ChromemStore) IsolationMode() IsolationMode {
	return s.isolation
}

// createEmbeddingFunc creates a chromem.EmbeddingFunc from our Embedder interface.
func (s *ChromemStore) createEmbeddingFunc() chromem.EmbeddingFunc {
	return func(ctx context.Context, text string) ([]float32, error) {
		return s.embedder.EmbedQuery(ctx, text)
	}
}

// getOrCreateCollection gets or creates a collection with the embedding function.
func (s *ChromemStore) getOrCreateCollection(ctx context.Context, name string) (*chromem.Collection, error) {
	// Validate collection name
	if err := ValidateCollectionName(name); err != nil {
		return nil, err
	}

	collection, err := s.db.GetOrCreateCollection(name, nil, s.createEmbeddingFunc())
	if err != nil {
		return nil, fmt.Errorf("getting/creating collection %s: %w", name, err)
	}

	s.collections.Store(name, true)
	return collection, nil
}

// AddDocuments adds documents to the vector store.
// If isolation mode is set, tenant metadata is automatically injected.
func (s *ChromemStore) AddDocuments(ctx context.Context, docs []Document) ([]string, error) {
	ctx, span := chromemTracer.Start(ctx, "ChromemStore.AddDocuments")
	defer span.End()

	span.SetAttributes(attribute.Int("document_count", len(docs)))

	if len(docs) == 0 {
		return nil, ErrEmptyDocuments
	}

	// Inject tenant metadata if isolation mode requires it
	if s.isolation != nil {
		if err := s.isolation.InjectMetadata(ctx, docs); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, fmt.Errorf("injecting tenant metadata: %w", err)
		}
	}

	// Determine collection name from first document
	collectionName := s.config.DefaultCollection
	if docs[0].Collection != "" {
		collectionName = docs[0].Collection
	}

	// Validate all documents target the same collection
	for i, doc := range docs {
		if doc.Collection != "" && doc.Collection != collectionName {
			return nil, fmt.Errorf("document at index %d has collection %q but batch targets %q - all documents must target the same collection",
				i, doc.Collection, collectionName)
		}
	}

	span.SetAttributes(attribute.String("collection", collectionName))

	collection, err := s.getOrCreateCollection(ctx, collectionName)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}

	// Convert documents to chromem format
	chromemDocs := make([]chromem.Document, len(docs))
	ids := make([]string, len(docs))
	texts := make([]string, len(docs))

	for i, doc := range docs {
		ids[i] = doc.ID
		if ids[i] == "" {
			// Generate unique ID using timestamp + index to avoid collisions
			ids[i] = fmt.Sprintf("doc_%d_%d", timeNow().UnixNano(), i)
			s.logger.Warn("auto-generated document ID - caller should provide explicit IDs",
				zap.String("generated_id", ids[i]),
				zap.Int("index", i),
			)
		}
		texts[i] = doc.Content
	}

	// Generate embeddings in batch
	embeddings, err := s.embedder.EmbedDocuments(ctx, texts)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("%w: %v", ErrEmbeddingFailed, err)
	}

	for i, doc := range docs {
		chromemDocs[i] = chromem.Document{
			ID:        ids[i],
			Content:   doc.Content,
			Metadata:  convertMetadataToString(doc.Metadata),
			Embedding: embeddings[i],
		}
	}

	// Add documents (concurrency of 1 since we already have embeddings)
	if err := collection.AddDocuments(ctx, chromemDocs, 1); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("adding documents: %w", err)
	}

	span.SetAttributes(attribute.Int("documents_added", len(ids)))
	span.SetStatus(codes.Ok, "success")

	s.logger.Debug("added documents to chromem",
		zap.String("collection", collectionName),
		zap.Int("count", len(docs)),
	)

	return ids, nil
}

// Search performs similarity search in the default collection.
func (s *ChromemStore) Search(ctx context.Context, query string, k int) ([]SearchResult, error) {
	return s.SearchInCollection(ctx, s.config.DefaultCollection, query, k, nil)
}

// SearchWithFilters performs similarity search with metadata filters.
func (s *ChromemStore) SearchWithFilters(ctx context.Context, query string, k int, filters map[string]interface{}) ([]SearchResult, error) {
	return s.SearchInCollection(ctx, s.config.DefaultCollection, query, k, filters)
}

// SearchInCollection performs similarity search in a specific collection.
// If isolation mode is set, tenant filters are automatically injected.
func (s *ChromemStore) SearchInCollection(ctx context.Context, collectionName string, query string, k int, filters map[string]interface{}) ([]SearchResult, error) {
	ctx, span := chromemTracer.Start(ctx, "ChromemStore.SearchInCollection")
	defer span.End()

	span.SetAttributes(
		attribute.String("collection", collectionName),
		attribute.Int("k", k),
	)

	// Validate inputs
	if err := ValidateCollectionName(collectionName); err != nil {
		return nil, err
	}

	if k <= 0 {
		return nil, fmt.Errorf("k must be positive, got %d", k)
	}

	if query == "" {
		return nil, fmt.Errorf("query cannot be empty")
	}

	// Inject tenant filters if isolation mode requires it
	if s.isolation != nil {
		var err error
		filters, err = s.isolation.InjectFilter(ctx, filters)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, fmt.Errorf("injecting tenant filter: %w", err)
		}
	}

	collection := s.db.GetCollection(collectionName, s.createEmbeddingFunc())
	if collection == nil {
		span.SetStatus(codes.Error, "collection not found")
		return nil, ErrCollectionNotFound
	}

	// Cap k at collection size (chromem requires nResults <= doc count)
	docCount := collection.Count()
	if docCount == 0 {
		return []SearchResult{}, nil
	}
	if k > docCount {
		k = docCount
	}

	// Convert filters to string map
	whereFilter := convertMetadataToString(filters)

	// Query collection
	results, err := collection.Query(ctx, query, k, whereFilter, nil)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("querying collection %s: %w", collectionName, err)
	}

	// Convert results
	searchResults := make([]SearchResult, len(results))
	for i, r := range results {
		searchResults[i] = SearchResult{
			ID:       r.ID,
			Content:  r.Content,
			Score:    r.Similarity,
			Metadata: convertMetadataFromString(r.Metadata),
		}
	}

	span.SetAttributes(attribute.Int("results_count", len(searchResults)))
	span.SetStatus(codes.Ok, "success")

	s.logger.Debug("searched chromem collection",
		zap.String("collection", collectionName),
		zap.Int("k", k),
		zap.Int("results", len(searchResults)),
	)

	return searchResults, nil
}

// DeleteDocuments deletes documents by their IDs from the default collection.
func (s *ChromemStore) DeleteDocuments(ctx context.Context, ids []string) error {
	return s.DeleteDocumentsFromCollection(ctx, s.config.DefaultCollection, ids)
}

// DeleteDocumentsFromCollection deletes documents by their IDs from a specific collection.
func (s *ChromemStore) DeleteDocumentsFromCollection(ctx context.Context, collectionName string, ids []string) error {
	ctx, span := chromemTracer.Start(ctx, "ChromemStore.DeleteDocumentsFromCollection")
	defer span.End()

	span.SetAttributes(
		attribute.String("collection", collectionName),
		attribute.Int("id_count", len(ids)),
	)

	if len(ids) == 0 {
		return nil
	}

	if err := ValidateCollectionName(collectionName); err != nil {
		return err
	}

	collection := s.db.GetCollection(collectionName, s.createEmbeddingFunc())
	if collection == nil {
		span.SetStatus(codes.Error, "collection not found")
		return ErrCollectionNotFound
	}

	// Delete each document, collecting failures
	var failures []string
	for _, id := range ids {
		if err := collection.Delete(ctx, nil, nil, id); err != nil {
			span.RecordError(err)
			s.logger.Error("failed to delete document",
				zap.String("collection", collectionName),
				zap.String("id", id),
				zap.Error(err),
			)
			failures = append(failures, id)
		}
	}

	if len(failures) > 0 {
		span.SetStatus(codes.Error, "partial deletion failure")
		return fmt.Errorf("failed to delete %d of %d documents: %v", len(failures), len(ids), failures)
	}

	span.SetStatus(codes.Ok, "success")

	s.logger.Debug("deleted documents from chromem",
		zap.String("collection", collectionName),
		zap.Int("count", len(ids)),
	)

	return nil
}

// CreateCollection creates a new collection with the specified configuration.
func (s *ChromemStore) CreateCollection(ctx context.Context, collectionName string, vectorSize int) error {
	ctx, span := chromemTracer.Start(ctx, "ChromemStore.CreateCollection")
	defer span.End()

	span.SetAttributes(
		attribute.String("collection", collectionName),
		attribute.Int("vector_size", vectorSize),
	)

	if err := ValidateCollectionName(collectionName); err != nil {
		return err
	}

	// Accept 0 as "use configured default"
	if vectorSize == 0 {
		vectorSize = s.config.VectorSize
	}

	// Check vector size matches
	if vectorSize != s.config.VectorSize {
		return fmt.Errorf("vector size %d does not match configured size %d", vectorSize, s.config.VectorSize)
	}

	// Check if collection already exists (chromem-go's CreateCollection is idempotent)
	// IMPORTANT: Must pass embedding function, not nil, because chromem-go sets
	// the default OpenAI embedder when nil is passed for persisted collections
	if existing := s.db.GetCollection(collectionName, s.createEmbeddingFunc()); existing != nil {
		return ErrCollectionExists
	}

	_, err := s.db.CreateCollection(collectionName, nil, s.createEmbeddingFunc())
	if err != nil {
		// Double-check in case of race condition
		if strings.Contains(err.Error(), "already exists") {
			return ErrCollectionExists
		}
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("creating collection %s: %w", collectionName, err)
	}

	s.collections.Store(collectionName, true)
	span.SetStatus(codes.Ok, "success")

	s.logger.Info("created chromem collection",
		zap.String("collection", collectionName),
		zap.Int("vector_size", vectorSize),
	)

	return nil
}

// DeleteCollection deletes a collection and all its documents.
func (s *ChromemStore) DeleteCollection(ctx context.Context, collectionName string) error {
	ctx, span := chromemTracer.Start(ctx, "ChromemStore.DeleteCollection")
	defer span.End()

	span.SetAttributes(attribute.String("collection", collectionName))

	if err := ValidateCollectionName(collectionName); err != nil {
		return err
	}

	if err := s.db.DeleteCollection(collectionName); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("deleting collection %s: %w", collectionName, err)
	}

	s.collections.Delete(collectionName)
	span.SetStatus(codes.Ok, "success")

	s.logger.Info("deleted chromem collection",
		zap.String("collection", collectionName),
	)

	return nil
}

// CollectionExists checks if a collection exists.
func (s *ChromemStore) CollectionExists(ctx context.Context, collectionName string) (bool, error) {
	ctx, span := chromemTracer.Start(ctx, "ChromemStore.CollectionExists")
	defer span.End()

	span.SetAttributes(attribute.String("collection", collectionName))

	if err := ValidateCollectionName(collectionName); err != nil {
		return false, err
	}

	// Must pass embedding function to avoid chromem-go setting OpenAI default
	collection := s.db.GetCollection(collectionName, s.createEmbeddingFunc())
	exists := collection != nil

	span.SetStatus(codes.Ok, "success")
	return exists, nil
}

// ListCollections returns a list of all collection names.
func (s *ChromemStore) ListCollections(ctx context.Context) ([]string, error) {
	ctx, span := chromemTracer.Start(ctx, "ChromemStore.ListCollections")
	defer span.End()

	collectionsMap := s.db.ListCollections()
	names := make([]string, 0, len(collectionsMap))
	for name := range collectionsMap {
		names = append(names, name)
	}

	span.SetAttributes(attribute.Int("collection_count", len(names)))
	span.SetStatus(codes.Ok, "success")

	return names, nil
}

// GetCollectionInfo returns metadata about a collection.
func (s *ChromemStore) GetCollectionInfo(ctx context.Context, collectionName string) (*CollectionInfo, error) {
	ctx, span := chromemTracer.Start(ctx, "ChromemStore.GetCollectionInfo")
	defer span.End()

	span.SetAttributes(attribute.String("collection", collectionName))

	if err := ValidateCollectionName(collectionName); err != nil {
		return nil, err
	}

	// Must pass embedding function to avoid chromem-go setting OpenAI default
	collection := s.db.GetCollection(collectionName, s.createEmbeddingFunc())
	if collection == nil {
		span.SetStatus(codes.Error, "collection not found")
		return nil, ErrCollectionNotFound
	}

	// chromem doesn't expose document count directly, so we count via query
	// Return a rough estimate by getting count from internal tracking
	info := &CollectionInfo{
		Name:       collectionName,
		PointCount: collection.Count(),
		VectorSize: s.config.VectorSize,
	}

	span.SetAttributes(attribute.Int("point_count", info.PointCount))
	span.SetStatus(codes.Ok, "success")

	return info, nil
}

// ExactSearch performs brute-force similarity search.
// Note: chromem-go always uses exact search (no HNSW), so this is the same as regular search.
func (s *ChromemStore) ExactSearch(ctx context.Context, collectionName string, query string, k int) ([]SearchResult, error) {
	return s.SearchInCollection(ctx, collectionName, query, k, nil)
}

// Close closes the ChromemStore.
// Note: chromem-go handles persistence automatically, no explicit close needed.
func (s *ChromemStore) Close() error {
	s.logger.Info("chromem store closed")
	return nil
}

// convertMetadataToString converts map[string]interface{} to map[string]string.
func convertMetadataToString(metadata map[string]interface{}) map[string]string {
	if metadata == nil {
		return nil
	}

	result := make(map[string]string, len(metadata))
	for k, v := range metadata {
		switch val := v.(type) {
		case string:
			result[k] = val
		case int:
			result[k] = fmt.Sprintf("%d", val)
		case int64:
			result[k] = fmt.Sprintf("%d", val)
		case float64:
			result[k] = fmt.Sprintf("%f", val)
		case bool:
			result[k] = fmt.Sprintf("%t", val)
		default:
			result[k] = fmt.Sprintf("%v", val)
		}
	}
	return result
}

// convertMetadataFromString converts map[string]string back to map[string]interface{}.
func convertMetadataFromString(metadata map[string]string) map[string]interface{} {
	if metadata == nil {
		return nil
	}

	result := make(map[string]interface{}, len(metadata))
	for k, v := range metadata {
		result[k] = v
	}
	return result
}

// Ensure ChromemStore implements Store interface.
var _ Store = (*ChromemStore)(nil)
