package remediation

import "context"

// QdrantClient provides a unified interface to Qdrant vector database.
// This interface will be implemented by the vectorstore package.
// TODO: Move this to internal/vectorstore once that package is ported.
type QdrantClient interface {
	// Collection operations
	CreateCollection(ctx context.Context, name string, vectorSize uint64) error
	DeleteCollection(ctx context.Context, name string) error
	CollectionExists(ctx context.Context, name string) (bool, error)
	ListCollections(ctx context.Context) ([]string, error)

	// Point operations
	Upsert(ctx context.Context, collection string, points []*QdrantPoint) error
	Search(ctx context.Context, collection string, vector []float32, limit uint64, filter *QdrantFilter) ([]*QdrantScoredPoint, error)
	Get(ctx context.Context, collection string, ids []string) ([]*QdrantPoint, error)
	Delete(ctx context.Context, collection string, ids []string) error

	// Health
	Health(ctx context.Context) error

	// Close closes the client connection
	Close() error
}

// QdrantPoint represents a vector point in Qdrant.
type QdrantPoint struct {
	ID      string
	Vector  []float32
	Payload map[string]interface{}
}

// QdrantScoredPoint represents a search result with score.
type QdrantScoredPoint struct {
	QdrantPoint
	Score float32
}

// QdrantFilter represents a filter for Qdrant queries.
type QdrantFilter struct {
	Must    []QdrantCondition
	Should  []QdrantCondition
	MustNot []QdrantCondition
}

// QdrantCondition represents a single filter condition.
type QdrantCondition struct {
	Field string
	Match interface{}
	Range *QdrantRangeCondition
}

// QdrantRangeCondition represents a range filter.
type QdrantRangeCondition struct {
	Gte *float64
	Lte *float64
	Gt  *float64
	Lt  *float64
}
