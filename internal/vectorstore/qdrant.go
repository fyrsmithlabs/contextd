// Package vectorstore provides vector storage implementations.
package vectorstore

import (
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/qdrant/go-client/qdrant"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"google.golang.org/grpc"
	grpccodes "google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Tracer for OpenTelemetry instrumentation.
var tracer = otel.Tracer("contextd-v2.vectorstore.qdrant")

// collectionNamePattern validates collection names.
// Pattern: lowercase letters, numbers, underscores, 1-64 characters.
var collectionNamePattern = regexp.MustCompile(`^[a-z0-9_]{1,64}$`)

// QdrantConfig holds configuration for Qdrant gRPC client.
type QdrantConfig struct {
	// Host is the Qdrant server hostname or IP address.
	// Default: "localhost"
	Host string

	// Port is the Qdrant gRPC port (NOT HTTP REST port).
	// Default: 6334 (gRPC), not 6333 (HTTP)
	Port int

	// CollectionName is the default collection for operations.
	// Format: {scope}_{type} for multi-tenancy
	// Examples: org_memories, platform_memories, platform_contextd_memories
	CollectionName string

	// VectorSize is the dimensionality of embeddings.
	// Examples: 384 (BAAI/bge-small-en-v1.5), 768 (BERT), 1536 (OpenAI)
	// MUST match Embedder output dimensions.
	VectorSize uint64

	// Distance is the similarity metric for vector search.
	// Options: Cosine (default), Euclid, Dot
	Distance qdrant.Distance

	// UseTLS enables TLS encryption for gRPC connection.
	// Default: false (MVP), true (production)
	UseTLS bool

	// MaxRetries is the maximum number of retry attempts for transient failures.
	// Default: 3
	MaxRetries int

	// RetryBackoff is the initial backoff duration for retries.
	// Doubles on each retry (exponential backoff).
	// Default: 1 second
	RetryBackoff time.Duration

	// MaxMessageSize is the maximum gRPC message size in bytes.
	// Default: 50MB (to handle large documents)
	MaxMessageSize int

	// CircuitBreakerThreshold is the number of failures before opening circuit.
	// Default: 5
	CircuitBreakerThreshold int

	// Isolation is the tenant isolation mode.
	// Default: PayloadIsolation for fail-closed security.
	// Set at construction time; immutable afterward to prevent race conditions.
	Isolation IsolationMode
}

// Validate validates the configuration.
func (c QdrantConfig) Validate() error {
	if c.Host == "" {
		return fmt.Errorf("%w: host required", ErrInvalidConfig)
	}
	if c.Port <= 0 || c.Port > 65535 {
		return fmt.Errorf("%w: invalid port: %d", ErrInvalidConfig, c.Port)
	}
	if c.CollectionName == "" {
		return fmt.Errorf("%w: collection name required", ErrInvalidConfig)
	}
	if c.VectorSize == 0 {
		return fmt.Errorf("%w: vector size required", ErrInvalidConfig)
	}
	return nil
}

// ApplyDefaults sets default values for unset fields.
func (c *QdrantConfig) ApplyDefaults() {
	if c.MaxRetries == 0 {
		c.MaxRetries = 3
	}
	if c.RetryBackoff == 0 {
		c.RetryBackoff = time.Second
	}
	if c.MaxMessageSize == 0 {
		c.MaxMessageSize = 50 * 1024 * 1024 // 50MB
	}
	if c.CircuitBreakerThreshold == 0 {
		c.CircuitBreakerThreshold = 5
	}
	if c.Distance == 0 {
		c.Distance = qdrant.Distance_Cosine
	}
}

// ValidateCollectionName validates a collection name against security rules.
// Pattern: ^[a-z0-9_]{1,64}$
// Rejects: uppercase, special chars, path traversal, spaces.
func ValidateCollectionName(name string) error {
	if name == "" {
		return fmt.Errorf("%w: collection name cannot be empty", ErrInvalidCollectionName)
	}
	if !collectionNamePattern.MatchString(name) {
		return fmt.Errorf("%w: collection name must match pattern ^[a-z0-9_]{1,64}$, got %q", ErrInvalidCollectionName, name)
	}
	return nil
}

// IsTransientError checks if an error is transient (should retry).
// Returns true for network timeouts, temporary unavailability.
// Returns false for invalid config, not found, permission denied.
func IsTransientError(err error) bool {
	if err == nil {
		return false
	}

	st, ok := status.FromError(err)
	if !ok {
		return false
	}

	switch st.Code() {
	case grpccodes.Unavailable, grpccodes.DeadlineExceeded, grpccodes.Aborted, grpccodes.ResourceExhausted:
		return true
	case grpccodes.InvalidArgument, grpccodes.NotFound, grpccodes.PermissionDenied, grpccodes.Unauthenticated:
		return false
	default:
		return false
	}
}

// QdrantStore is a Store implementation using Qdrant's native gRPC client.
//
// This implementation bypasses Qdrant's actix-web HTTP layer, eliminating the 256kB
// payload limit that causes 413 errors during repository indexing.
//
// Key features:
//   - Native gRPC transport (port 6334)
//   - Binary protobuf encoding (no JSON size limits)
//   - Better performance than HTTP REST
//   - Full Qdrant feature access
//   - Collection-per-project isolation
//   - Tenant isolation via payload filtering
type QdrantStore struct {
	// client is the official Qdrant Go gRPC client
	client *qdrant.Client

	// embedder generates vector embeddings from text
	embedder Embedder

	// config holds the store configuration
	config QdrantConfig

	// isolation defines how tenant isolation is enforced
	isolation IsolationMode

	// collections is a cache of collection existence to avoid repeated checks
	// Key: collection name, Value: true if exists
	collections sync.Map

	// circuitBreaker tracks failures for circuit breaker pattern
	circuitBreaker struct {
		failures int
		lastFail time.Time
		mu       sync.Mutex
	}
}

// NewQdrantStore creates a new QdrantStore with the given configuration.
//
// The constructor performs the following steps:
//  1. Validates configuration
//  2. Creates Qdrant gRPC client
//  3. Performs health check
//  4. Returns ready-to-use store
//
// Returns an error if:
//   - Configuration is invalid
//   - Connection to Qdrant fails
//   - Health check fails
func NewQdrantStore(config QdrantConfig, embedder Embedder) (*QdrantStore, error) {
	// Apply defaults
	config.ApplyDefaults()

	// Validate config
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("validating config: %w", err)
	}

	// Validate collection name
	if err := ValidateCollectionName(config.CollectionName); err != nil {
		return nil, fmt.Errorf("validating collection name: %w", err)
	}

	// Warn if TLS is disabled (plaintext gRPC)
	if !config.UseTLS {
		fmt.Fprintf(os.Stderr, "WARNING: Qdrant gRPC using plaintext (TLS disabled). Insecure for production.\n")
	}

	// Create Qdrant client with gRPC options
	qdrantConfig := &qdrant.Config{
		Host:   config.Host,
		Port:   config.Port,
		UseTLS: config.UseTLS,
		GrpcOptions: []grpc.DialOption{
			grpc.WithDefaultCallOptions(
				grpc.MaxCallRecvMsgSize(config.MaxMessageSize),
				grpc.MaxCallSendMsgSize(config.MaxMessageSize),
			),
		},
	}

	client, err := qdrant.NewClient(qdrantConfig)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrConnectionFailed, err)
	}

	// Use isolation from config, defaulting to PayloadIsolation for fail-closed security
	isolation := config.Isolation
	if isolation == nil {
		isolation = NewPayloadIsolation()
	}

	store := &QdrantStore{
		client:    client,
		embedder:  embedder,
		config:    config,
		isolation: isolation,
	}

	// Health check
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := store.healthCheck(ctx); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("health check failed: %w", err)
	}

	return store, nil
}

// Close closes the Qdrant gRPC connection.
func (s *QdrantStore) Close() error {
	if s.client != nil {
		return s.client.Close()
	}
	return nil
}

// SetIsolationMode sets the tenant isolation mode for this store.
//
// DEPRECATED: Prefer setting isolation via config at construction time
// (e.g., QdrantConfig.Isolation) for thread-safety. This method exists
// for backward compatibility but should only be called once before any
// operations. Calling SetIsolationMode concurrently with operations may
// cause race conditions.
//
// Use NewPayloadIsolation() for multi-tenant payload filtering,
// NewFilesystemIsolation() for database-per-project isolation,
// or NewNoIsolation() for testing only.
//
// Default is PayloadIsolation for fail-closed security.
func (s *QdrantStore) SetIsolationMode(mode IsolationMode) {
	s.isolation = mode
}

// IsolationMode returns the current isolation mode.
func (s *QdrantStore) IsolationMode() IsolationMode {
	return s.isolation
}

// healthCheck performs a health check on the Qdrant connection.
func (s *QdrantStore) healthCheck(ctx context.Context) error {
	ctx, span := tracer.Start(ctx, "QdrantStore.HealthCheck")
	defer span.End()

	_, err := s.client.HealthCheck(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("health check failed: %w", err)
	}

	span.SetStatus(codes.Ok, "healthy")
	return nil
}

// retryOperation retries an operation with exponential backoff.
func (s *QdrantStore) retryOperation(ctx context.Context, operationName string, operation func() error) error {
	backoff := s.config.RetryBackoff

	for attempt := 0; attempt <= s.config.MaxRetries; attempt++ {
		err := operation()
		if err == nil {
			s.resetCircuitBreaker()
			return nil
		}

		// Check circuit breaker
		if s.isCircuitOpen() {
			return fmt.Errorf("%s: circuit breaker open", operationName)
		}

		// Check if error is transient
		if !IsTransientError(err) {
			return fmt.Errorf("%s failed (permanent): %w", operationName, err)
		}

		// Record failure for circuit breaker
		s.recordFailure()

		// Last attempt, return error
		if attempt == s.config.MaxRetries {
			return fmt.Errorf("%s failed after %d retries: %w", operationName, s.config.MaxRetries, err)
		}

		// Wait before retry (exponential backoff)
		select {
		case <-ctx.Done():
			return fmt.Errorf("%s canceled: %w", operationName, ctx.Err())
		case <-time.After(backoff):
			backoff *= 2
		}
	}
	return nil
}

func (s *QdrantStore) recordFailure() {
	s.circuitBreaker.mu.Lock()
	defer s.circuitBreaker.mu.Unlock()
	s.circuitBreaker.failures++
	s.circuitBreaker.lastFail = time.Now()
}

func (s *QdrantStore) resetCircuitBreaker() {
	s.circuitBreaker.mu.Lock()
	defer s.circuitBreaker.mu.Unlock()
	s.circuitBreaker.failures = 0
}

func (s *QdrantStore) isCircuitOpen() bool {
	s.circuitBreaker.mu.Lock()
	defer s.circuitBreaker.mu.Unlock()

	// Circuit is open if too many failures recently
	if s.circuitBreaker.failures >= s.config.CircuitBreakerThreshold {
		// Allow retry after 30 seconds
		if time.Since(s.circuitBreaker.lastFail) > 30*time.Second {
			s.circuitBreaker.failures = 0
			return false
		}
		return true
	}
	return false
}

// AddDocuments adds documents to the vector store.
// If isolation mode is set, tenant metadata is automatically injected.
func (s *QdrantStore) AddDocuments(ctx context.Context, docs []Document) ([]string, error) {
	ctx, span := tracer.Start(ctx, "QdrantStore.AddDocuments")
	defer span.End()

	span.SetAttributes(
		attribute.Int("document_count", len(docs)),
		attribute.String("collection", s.config.CollectionName),
	)

	if len(docs) == 0 {
		return nil, fmt.Errorf("documents cannot be empty")
	}

	// Inject tenant metadata if isolation mode requires it
	if s.isolation != nil {
		if err := s.isolation.InjectMetadata(ctx, docs); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, fmt.Errorf("injecting tenant metadata: %w", err)
		}
	}

	// Generate embeddings
	texts := make([]string, len(docs))
	for i, doc := range docs {
		texts[i] = doc.Content
	}

	var embeddings [][]float32
	if s.embedder != nil {
		embs, err := s.embedder.EmbedDocuments(ctx, texts)
		if err != nil {
			span.RecordError(err)
			return nil, fmt.Errorf("%w: %v", ErrEmbeddingFailed, err)
		}
		embeddings = embs
	} else {
		// For testing without embedder, create zero vectors
		embeddings = make([][]float32, len(docs))
		for i := range embeddings {
			embeddings[i] = make([]float32, s.config.VectorSize)
		}
	}

	// Convert to Qdrant points
	points := make([]*qdrant.PointStruct, len(docs))
	ids := make([]string, len(docs))

	for i, doc := range docs {
		pointID := doc.ID
		if pointID == "" {
			pointID = fmt.Sprintf("doc_%d_%d", time.Now().UnixNano(), i)
		}
		ids[i] = pointID

		// Build payload from metadata
		payload := make(map[string]*qdrant.Value)
		payload["content"] = &qdrant.Value{Kind: &qdrant.Value_StringValue{StringValue: doc.Content}}
		payload["id"] = &qdrant.Value{Kind: &qdrant.Value_StringValue{StringValue: pointID}}

		for k, v := range doc.Metadata {
			switch val := v.(type) {
			case string:
				payload[k] = &qdrant.Value{Kind: &qdrant.Value_StringValue{StringValue: val}}
			case int:
				payload[k] = &qdrant.Value{Kind: &qdrant.Value_IntegerValue{IntegerValue: int64(val)}}
			case int64:
				payload[k] = &qdrant.Value{Kind: &qdrant.Value_IntegerValue{IntegerValue: val}}
			case float64:
				payload[k] = &qdrant.Value{Kind: &qdrant.Value_DoubleValue{DoubleValue: val}}
			case bool:
				payload[k] = &qdrant.Value{Kind: &qdrant.Value_BoolValue{BoolValue: val}}
			}
		}

		// Generate Qdrant point ID - use doc ID if valid UUID, otherwise generate new
		// The original document ID is preserved in payload["id"] for retrieval
		var qdrantPointID *qdrant.PointId
		if _, err := uuid.Parse(pointID); err == nil {
			qdrantPointID = qdrant.NewIDUUID(pointID)
		} else {
			qdrantPointID = qdrant.NewIDUUID(uuid.New().String())
		}

		points[i] = &qdrant.PointStruct{
			Id:      qdrantPointID,
			Vectors: qdrant.NewVectors(embeddings[i]...),
			Payload: payload,
		}
	}

	// Determine collection name - use doc.Collection if specified, otherwise default
	collectionName := s.config.CollectionName
	if len(docs) > 0 && docs[0].Collection != "" {
		collectionName = docs[0].Collection
	}

	// Ensure collection exists (auto-create for project-specific collections)
	if collectionName != s.config.CollectionName {
		exists, err := s.CollectionExists(ctx, collectionName)
		if err != nil {
			span.RecordError(err)
			return nil, fmt.Errorf("checking collection %s: %w", collectionName, err)
		}
		if !exists {
			if err := s.CreateCollection(ctx, collectionName, int(s.config.VectorSize)); err != nil {
				span.RecordError(err)
				return nil, fmt.Errorf("creating collection %s: %w", collectionName, err)
			}
		}
	}

	// Upsert to Qdrant
	err := s.retryOperation(ctx, "upsert", func() error {
		_, err := s.client.Upsert(ctx, &qdrant.UpsertPoints{
			CollectionName: collectionName,
			Points:         points,
		})
		return err
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("upserting points to collection %s: %w", collectionName, err)
	}

	span.SetAttributes(attribute.Int("points_added", len(ids)))
	span.SetStatus(codes.Ok, "success")
	return ids, nil
}

// Search performs similarity search in the default collection.
func (s *QdrantStore) Search(ctx context.Context, query string, k int) ([]SearchResult, error) {
	return s.SearchInCollection(ctx, s.config.CollectionName, query, k, nil)
}

// SearchWithFilters performs similarity search with metadata filters.
func (s *QdrantStore) SearchWithFilters(ctx context.Context, query string, k int, filters map[string]interface{}) ([]SearchResult, error) {
	return s.SearchInCollection(ctx, s.config.CollectionName, query, k, filters)
}

// SearchInCollection performs similarity search in a specific collection.
// If isolation mode is set, tenant filters are automatically injected.
func (s *QdrantStore) SearchInCollection(ctx context.Context, collectionName string, query string, k int, filters map[string]interface{}) ([]SearchResult, error) {
	ctx, span := tracer.Start(ctx, "QdrantStore.SearchInCollection")
	defer span.End()

	span.SetAttributes(
		attribute.String("collection", collectionName),
		attribute.Int("k", k),
	)

	// Validate collection name
	if err := ValidateCollectionName(collectionName); err != nil {
		return nil, err
	}

	// Validate k parameter (security: prevent resource exhaustion)
	if k <= 0 {
		return nil, fmt.Errorf("k must be positive, got %d", k)
	}
	const maxK = 10000 // reasonable upper bound to prevent DoS
	if k > maxK {
		k = maxK // cap at maximum
	}

	// Validate query (security: prevent DoS via oversized queries)
	if query == "" {
		return nil, fmt.Errorf("query cannot be empty")
	}
	const maxQueryLength = 10000 // characters
	if len(query) > maxQueryLength {
		return nil, fmt.Errorf("query exceeds maximum length of %d characters", maxQueryLength)
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

	// Generate embedding for query
	var queryVector []float32
	if s.embedder != nil {
		vectors, err := s.embedder.EmbedDocuments(ctx, []string{query})
		if err != nil {
			span.RecordError(err)
			return nil, fmt.Errorf("%w: %v", ErrEmbeddingFailed, err)
		}
		queryVector = vectors[0]
	} else {
		queryVector = make([]float32, s.config.VectorSize)
	}

	// Build filter if provided
	var filter *qdrant.Filter
	if len(filters) > 0 {
		conditions := make([]*qdrant.Condition, 0, len(filters))
		for key, value := range filters {
			switch v := value.(type) {
			case string:
				conditions = append(conditions, &qdrant.Condition{
					ConditionOneOf: &qdrant.Condition_Field{
						Field: &qdrant.FieldCondition{
							Key: key,
							Match: &qdrant.Match{
								MatchValue: &qdrant.Match_Keyword{Keyword: v},
							},
						},
					},
				})
			}
		}
		if len(conditions) > 0 {
			filter = &qdrant.Filter{Must: conditions}
		}
	}

	// Search
	var results []*qdrant.ScoredPoint
	err := s.retryOperation(ctx, "search", func() error {
		res, err := s.client.Query(ctx, &qdrant.QueryPoints{
			CollectionName: collectionName,
			Query:          qdrant.NewQuery(queryVector...),
			Limit:          qdrant.PtrOf(uint64(k)),
			WithPayload:    qdrant.NewWithPayload(true),
			Filter:         filter,
		})
		if err != nil {
			return err
		}
		results = res
		return nil
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("searching collection %s: %w", collectionName, err)
	}

	// Convert to SearchResult
	searchResults := make([]SearchResult, len(results))
	for i, point := range results {
		result := SearchResult{
			Score: point.Score,
		}

		// Extract metadata from payload
		if point.Payload != nil {
			result.Metadata = make(map[string]interface{})
			for k, v := range point.Payload {
				switch val := v.Kind.(type) {
				case *qdrant.Value_StringValue:
					// Always add to metadata for consistent access
					result.Metadata[k] = val.StringValue
					// Also set dedicated fields for commonly accessed values
					if k == "content" {
						result.Content = val.StringValue
					} else if k == "id" {
						result.ID = val.StringValue
					}
				case *qdrant.Value_IntegerValue:
					result.Metadata[k] = val.IntegerValue
				case *qdrant.Value_DoubleValue:
					result.Metadata[k] = val.DoubleValue
				case *qdrant.Value_BoolValue:
					result.Metadata[k] = val.BoolValue
				}
			}
		}

		searchResults[i] = result
	}

	span.SetAttributes(attribute.Int("results_count", len(searchResults)))
	span.SetStatus(codes.Ok, "success")
	return searchResults, nil
}

// DeleteDocuments deletes documents by their IDs from the default collection.
func (s *QdrantStore) DeleteDocuments(ctx context.Context, ids []string) error {
	return s.DeleteDocumentsFromCollection(ctx, s.config.CollectionName, ids)
}

// DeleteDocumentsFromCollection deletes documents by their IDs from a specific collection.
func (s *QdrantStore) DeleteDocumentsFromCollection(ctx context.Context, collectionName string, ids []string) error {
	ctx, span := tracer.Start(ctx, "QdrantStore.DeleteDocumentsFromCollection")
	defer span.End()

	span.SetAttributes(
		attribute.Int("id_count", len(ids)),
		attribute.String("collection", collectionName),
	)

	if len(ids) == 0 {
		return nil
	}

	// Delete by filter matching document IDs
	err := s.retryOperation(ctx, "delete", func() error {
		_, err := s.client.Delete(ctx, &qdrant.DeletePoints{
			CollectionName: collectionName,
			Points: &qdrant.PointsSelector{
				PointsSelectorOneOf: &qdrant.PointsSelector_Filter{
					Filter: &qdrant.Filter{
						Must: []*qdrant.Condition{
							{
								ConditionOneOf: &qdrant.Condition_Field{
									Field: &qdrant.FieldCondition{
										Key: "id",
										Match: &qdrant.Match{
											MatchValue: &qdrant.Match_Keywords{
												Keywords: &qdrant.RepeatedStrings{Strings: ids},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		})
		return err
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("delete failed (permanent): %w", err)
	}

	span.SetStatus(codes.Ok, "success")
	return nil
}

// CreateCollection creates a new collection with the specified configuration.
func (s *QdrantStore) CreateCollection(ctx context.Context, collectionName string, vectorSize int) error {
	ctx, span := tracer.Start(ctx, "QdrantStore.CreateCollection")
	defer span.End()

	span.SetAttributes(
		attribute.String("collection", collectionName),
		attribute.Int("vector_size", vectorSize),
	)

	// Validate collection name
	if err := ValidateCollectionName(collectionName); err != nil {
		return err
	}

	err := s.retryOperation(ctx, "create_collection", func() error {
		return s.client.CreateCollection(ctx, &qdrant.CreateCollection{
			CollectionName: collectionName,
			VectorsConfig: qdrant.NewVectorsConfig(&qdrant.VectorParams{
				Size:     uint64(vectorSize),
				Distance: s.config.Distance,
			}),
		})
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("creating collection %s: %w", collectionName, err)
	}

	// Cache collection existence
	s.collections.Store(collectionName, true)

	span.SetStatus(codes.Ok, "success")
	return nil
}

// DeleteCollection deletes a collection and all its documents.
func (s *QdrantStore) DeleteCollection(ctx context.Context, collectionName string) error {
	ctx, span := tracer.Start(ctx, "QdrantStore.DeleteCollection")
	defer span.End()

	span.SetAttributes(attribute.String("collection", collectionName))

	// Validate collection name
	if err := ValidateCollectionName(collectionName); err != nil {
		return err
	}

	err := s.retryOperation(ctx, "delete_collection", func() error {
		return s.client.DeleteCollection(ctx, collectionName)
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("deleting collection %s: %w", collectionName, err)
	}

	// Remove from cache
	s.collections.Delete(collectionName)

	span.SetStatus(codes.Ok, "success")
	return nil
}

// CollectionExists checks if a collection exists.
func (s *QdrantStore) CollectionExists(ctx context.Context, collectionName string) (bool, error) {
	ctx, span := tracer.Start(ctx, "QdrantStore.CollectionExists")
	defer span.End()

	span.SetAttributes(attribute.String("collection", collectionName))

	// Validate collection name
	if err := ValidateCollectionName(collectionName); err != nil {
		return false, err
	}

	// Check cache first
	if _, ok := s.collections.Load(collectionName); ok {
		return true, nil
	}

	var exists bool
	err := s.retryOperation(ctx, "collection_exists", func() error {
		info, err := s.client.GetCollectionInfo(ctx, collectionName)
		if err != nil {
			st, ok := status.FromError(err)
			if ok && st.Code() == grpccodes.NotFound {
				exists = false
				return nil
			}
			return err
		}
		exists = info != nil
		return nil
	})
	if err != nil {
		span.RecordError(err)
		return false, fmt.Errorf("checking collection %s: %w", collectionName, err)
	}

	if exists {
		s.collections.Store(collectionName, true)
	}

	span.SetStatus(codes.Ok, "success")
	return exists, nil
}

// ListCollections returns a list of all collection names.
func (s *QdrantStore) ListCollections(ctx context.Context) ([]string, error) {
	ctx, span := tracer.Start(ctx, "QdrantStore.ListCollections")
	defer span.End()

	var collections []string
	err := s.retryOperation(ctx, "list_collections", func() error {
		result, err := s.client.ListCollections(ctx)
		if err != nil {
			return err
		}
		collections = result
		return nil
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("listing collections: %w", err)
	}

	span.SetAttributes(attribute.Int("collection_count", len(collections)))
	span.SetStatus(codes.Ok, "success")
	return collections, nil
}

// GetCollectionInfo returns metadata about a collection.
func (s *QdrantStore) GetCollectionInfo(ctx context.Context, collectionName string) (*CollectionInfo, error) {
	ctx, span := tracer.Start(ctx, "QdrantStore.GetCollectionInfo")
	defer span.End()

	span.SetAttributes(attribute.String("collection", collectionName))

	// Validate collection name
	if err := ValidateCollectionName(collectionName); err != nil {
		return nil, err
	}

	var info *CollectionInfo
	err := s.retryOperation(ctx, "get_collection_info", func() error {
		collInfo, err := s.client.GetCollectionInfo(ctx, collectionName)
		if err != nil {
			st, ok := status.FromError(err)
			if ok && st.Code() == grpccodes.NotFound {
				return ErrCollectionNotFound
			}
			return err
		}
		pointCount := 0
		if collInfo.PointsCount != nil {
			pointCount = int(*collInfo.PointsCount)
		}
		info = &CollectionInfo{
			Name:       collectionName,
			PointCount: pointCount,
			VectorSize: int(s.config.VectorSize),
		}
		return nil
	})
	if err != nil {
		span.RecordError(err)
		if errors.Is(err, ErrCollectionNotFound) {
			span.SetStatus(codes.Error, "collection not found")
			return nil, ErrCollectionNotFound
		}
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("getting collection info for %s: %w", collectionName, err)
	}

	span.SetAttributes(attribute.Int("point_count", info.PointCount))
	span.SetStatus(codes.Ok, "success")
	return info, nil
}

// ExactSearch performs brute-force similarity search without using HNSW index.
// This is a fallback for small datasets (<10 vectors) where HNSW index may not be built.
func (s *QdrantStore) ExactSearch(ctx context.Context, collectionName string, query string, k int) ([]SearchResult, error) {
	ctx, span := tracer.Start(ctx, "QdrantStore.ExactSearch")
	defer span.End()

	span.SetAttributes(
		attribute.String("collection", collectionName),
		attribute.Int("k", k),
		attribute.Bool("exact", true),
	)

	// Validate collection name
	if err := ValidateCollectionName(collectionName); err != nil {
		return nil, err
	}

	// Generate query embedding
	queryVector, err := s.embedder.EmbedQuery(ctx, query)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "embedding failed")
		return nil, fmt.Errorf("%w: %v", ErrEmbeddingFailed, err)
	}

	// Search with exact=true to disable HNSW and use brute-force
	var results []*qdrant.ScoredPoint
	err = s.retryOperation(ctx, "exact_search", func() error {
		res, err := s.client.Query(ctx, &qdrant.QueryPoints{
			CollectionName: collectionName,
			Query:          qdrant.NewQuery(queryVector...),
			Limit:          qdrant.PtrOf(uint64(k)),
			WithPayload:    qdrant.NewWithPayload(true),
			Params: &qdrant.SearchParams{
				Exact: qdrant.PtrOf(true), // Disable HNSW, use brute-force
			},
		})
		if err != nil {
			return err
		}
		results = res
		return nil
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("exact search in collection %s: %w", collectionName, err)
	}

	// Convert to SearchResult
	searchResults := make([]SearchResult, len(results))
	for i, point := range results {
		result := SearchResult{
			Score: point.Score,
		}

		// Extract metadata from payload
		if point.Payload != nil {
			result.Metadata = make(map[string]interface{})
			for k, v := range point.Payload {
				switch val := v.Kind.(type) {
				case *qdrant.Value_StringValue:
					// Always add to metadata for consistent access
					result.Metadata[k] = val.StringValue
					// Also set dedicated fields for commonly accessed values
					if k == "content" {
						result.Content = val.StringValue
					} else if k == "id" {
						result.ID = val.StringValue
					}
				case *qdrant.Value_IntegerValue:
					result.Metadata[k] = val.IntegerValue
				case *qdrant.Value_DoubleValue:
					result.Metadata[k] = val.DoubleValue
				case *qdrant.Value_BoolValue:
					result.Metadata[k] = val.BoolValue
				}
			}
		}

		searchResults[i] = result
	}

	span.SetAttributes(attribute.Int("results_count", len(searchResults)))
	span.SetStatus(codes.Ok, "success")
	return searchResults, nil
}

// Ensure QdrantStore implements Store interface.
var _ Store = (*QdrantStore)(nil)
