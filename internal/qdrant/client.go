package qdrant

import (
	"context"
)

// Client provides a unified interface to Qdrant vector database.
// This is a stub interface for contextd-v2 port - full implementation pending.
type Client interface {
	// Collection operations
	CreateCollection(ctx context.Context, name string, vectorSize uint64) error
	DeleteCollection(ctx context.Context, name string) error
	CollectionExists(ctx context.Context, name string) (bool, error)
	ListCollections(ctx context.Context) ([]string, error)

	// Point operations
	Upsert(ctx context.Context, collection string, points []*Point) error
	Search(ctx context.Context, collection string, vector []float32, limit uint64, filter *Filter) ([]*ScoredPoint, error)
	Get(ctx context.Context, collection string, ids []string) ([]*Point, error)
	Delete(ctx context.Context, collection string, ids []string) error

	// Health
	Health(ctx context.Context) error

	// Close closes the client connection
	Close() error
}

// Point represents a vector point in Qdrant.
type Point struct {
	ID      string
	Vector  []float32
	Payload map[string]interface{}
}

// ScoredPoint represents a search result with score.
type ScoredPoint struct {
	Point
	Score float32
}

// Filter represents a filter for search operations.
type Filter struct {
	Must    []Condition
	Should  []Condition
	MustNot []Condition
}

// Condition represents a filter condition.
type Condition struct {
	Field string
	Match interface{}
	Range *RangeCondition
}

// RangeCondition represents a range filter.
type RangeCondition struct {
	Gte *float64
	Lte *float64
	Gt  *float64
	Lt  *float64
}
