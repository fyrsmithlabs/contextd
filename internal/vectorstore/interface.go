// Package vectorstore defines the interface for vector storage operations.
package vectorstore

import (
	"context"
	"errors"
)

// Sentinel errors for vector store operations.
var (
	// ErrCollectionNotFound is returned when a collection does not exist.
	ErrCollectionNotFound = errors.New("collection not found")

	// ErrCollectionExists is returned when attempting to create an existing collection.
	ErrCollectionExists = errors.New("collection already exists")

	// ErrInvalidConfig indicates invalid configuration.
	ErrInvalidConfig = errors.New("invalid configuration")

	// ErrEmptyDocuments indicates empty or nil documents.
	ErrEmptyDocuments = errors.New("empty or nil documents")

	// ErrConnectionFailed indicates gRPC connection issues.
	ErrConnectionFailed = errors.New("failed to connect to Qdrant")

	// ErrEmbeddingFailed indicates embedding generation failure.
	ErrEmbeddingFailed = errors.New("failed to generate embeddings")

	// ErrInvalidCollectionName indicates collection name validation failure.
	ErrInvalidCollectionName = errors.New("invalid collection name")
)

// CollectionInfo contains metadata about a vector collection.
type CollectionInfo struct {
	// Name is the collection name.
	Name string `json:"name"`

	// PointCount is the number of vectors in the collection.
	PointCount int `json:"point_count"`

	// VectorSize is the dimensionality of vectors in this collection.
	VectorSize int `json:"vector_size"`
}

// Embedder generates vector embeddings from text.
//
// Embeddings are dense numerical representations that capture semantic meaning,
// enabling similarity search. Implementations can use local models (TEI) or
// cloud APIs (OpenAI, Cohere).
type Embedder interface {
	// EmbedDocuments generates embeddings for multiple texts.
	// Returns a slice of embeddings (one per input text) or an error.
	EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error)

	// EmbedQuery generates an embedding for a single query.
	// Some models optimize differently for queries vs documents.
	EmbedQuery(ctx context.Context, text string) ([]float32, error)
}

// Store is the interface for vector storage operations.
//
// This interface is transport-agnostic - implementations can use HTTP REST,
// gRPC, or any other protocol. The interface focuses on contextd's specific
// needs for document storage, search, and collection management.
//
// Collection Naming Convention:
//   - Organization: org_{type} (e.g., org_memories)
//   - Team: {team}_{type} (e.g., platform_memories)
//   - Project: {team}_{project}_{type} (e.g., platform_contextd_memories)
//
// Tenant Isolation:
//
// Stores support two isolation modes. The preferred pattern is to set isolation
// via config at construction time (e.g., ChromemConfig.Isolation) for thread-safety:
//
//   - PayloadIsolation: Single collection per type with metadata-based filtering.
//     All documents include tenant_id, team_id, project_id in metadata.
//     Queries automatically filter by tenant context from ctx.
//     Requires: TenantInfo in context (see ContextWithTenant).
//     Security: Fail-closed - missing tenant context returns ErrMissingTenant.
//
//   - FilesystemIsolation: Database-per-project isolation (legacy).
//     Uses StoreProvider to create separate stores per tenant/project path.
//     Physical filesystem isolation provides security boundary.
//
// When using PayloadIsolation, callers MUST provide tenant context:
//
//	ctx = vectorstore.ContextWithTenant(ctx, &vectorstore.TenantInfo{
//	    TenantID:  "org-123",
//	    TeamID:    "team-1",    // optional
//	    ProjectID: "proj-1",    // optional
//	})
//	results, err := store.Search(ctx, query, k)
//
// Implementations:
//   - ChromemStore: Embedded chromem-go (default)
//   - QdrantStore: External Qdrant gRPC client
type Store interface {
	// AddDocuments adds documents to the vector store.
	//
	// Documents are embedded and stored with their metadata. The document ID
	// is used as the unique identifier in the vector store.
	//
	// If Document.Collection is specified, the document is added to that collection.
	// Otherwise, the implementation's default collection is used.
	//
	// Returns the IDs of added documents and an error if the operation fails.
	AddDocuments(ctx context.Context, docs []Document) ([]string, error)

	// Search performs similarity search in the default collection.
	//
	// It searches for documents similar to the query and returns up to k results
	// ordered by similarity score (highest first).
	//
	// Returns search results with scores and metadata, or an error if search fails.
	Search(ctx context.Context, query string, k int) ([]SearchResult, error)

	// SearchWithFilters performs similarity search with metadata filters.
	//
	// Filters are applied to document metadata (e.g., {"owner": "alice"}).
	// Only documents matching ALL filter conditions are returned.
	//
	// Returns filtered search results or an error if search fails.
	SearchWithFilters(ctx context.Context, query string, k int, filters map[string]interface{}) ([]SearchResult, error)

	// SearchInCollection performs similarity search in a specific collection.
	//
	// This supports the hierarchical collection architecture by allowing searches
	// in scope-specific collections (e.g., "org_memories", "platform_contextd_memories").
	//
	// Returns filtered search results from the specified collection, or an error.
	SearchInCollection(ctx context.Context, collectionName string, query string, k int, filters map[string]interface{}) ([]SearchResult, error)

	// DeleteDocuments deletes documents by their IDs from the default collection.
	//
	// Returns an error if deletion fails.
	DeleteDocuments(ctx context.Context, ids []string) error

	// DeleteDocumentsFromCollection deletes documents by their IDs from a specific collection.
	//
	// Returns an error if deletion fails.
	DeleteDocumentsFromCollection(ctx context.Context, collectionName string, ids []string) error

	// CreateCollection creates a new collection with the specified configuration.
	//
	// Collections are namespaces for documents (e.g., project-specific collections).
	// The vectorSize parameter specifies the dimensionality of embeddings.
	//
	// Returns an error if collection creation fails or collection already exists.
	CreateCollection(ctx context.Context, collectionName string, vectorSize int) error

	// DeleteCollection deletes a collection and all its documents.
	//
	// This is a destructive operation that cannot be undone.
	//
	// Returns an error if deletion fails or collection doesn't exist.
	DeleteCollection(ctx context.Context, collectionName string) error

	// CollectionExists checks if a collection exists.
	//
	// Returns true if the collection exists, false otherwise.
	// Returns an error only if the check operation itself fails.
	CollectionExists(ctx context.Context, collectionName string) (bool, error)

	// ListCollections returns a list of all collection names.
	//
	// Returns collection names or an error if listing fails.
	ListCollections(ctx context.Context) ([]string, error)

	// GetCollectionInfo returns metadata about a collection.
	//
	// Returns collection info including point count and vector size.
	// Returns ErrCollectionNotFound if the collection doesn't exist.
	GetCollectionInfo(ctx context.Context, collectionName string) (*CollectionInfo, error)

	// ExactSearch performs brute-force similarity search without using HNSW index.
	//
	// This is a fallback for small datasets (<10 vectors) where HNSW index
	// may not be built. It performs exact cosine similarity on all vectors.
	//
	// Returns search results ordered by similarity score (highest first).
	ExactSearch(ctx context.Context, collectionName string, query string, k int) ([]SearchResult, error)

	// SetIsolationMode sets the tenant isolation mode for this store.
	//
	// DEPRECATED: Prefer setting isolation via config at construction time
	// (e.g., ChromemConfig.Isolation) for thread-safety. This method exists
	// for backward compatibility but should only be called once before any
	// operations. Calling SetIsolationMode concurrently with operations may
	// cause race conditions.
	//
	// Use NewPayloadIsolation() for multi-tenant payload filtering,
	// NewFilesystemIsolation() for database-per-project isolation,
	// or NewNoIsolation() for testing only.
	//
	// Default is PayloadIsolation for fail-closed security.
	SetIsolationMode(mode IsolationMode)

	// IsolationMode returns the current isolation mode.
	IsolationMode() IsolationMode

	// Close closes the vector store connection and releases resources.
	Close() error
}
