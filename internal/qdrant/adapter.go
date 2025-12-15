// Package qdrant provides Qdrant vector database client implementations.
package qdrant

import (
	"context"

	"github.com/fyrsmithlabs/contextd/internal/vectorstore"
)

// RemediationAdapter adapts GRPCClient to implement vectorstore.QdrantClient interface.
type RemediationAdapter struct {
	client *GRPCClient
}

// NewRemediationAdapter creates an adapter for remediation service.
func NewRemediationAdapter(client *GRPCClient) *RemediationAdapter {
	return &RemediationAdapter{client: client}
}

// CreateCollection creates a new collection with the specified configuration.
func (a *RemediationAdapter) CreateCollection(ctx context.Context, name string, vectorSize uint64) error {
	return a.client.CreateCollection(ctx, name, vectorSize)
}

// DeleteCollection deletes a collection and all its documents.
func (a *RemediationAdapter) DeleteCollection(ctx context.Context, name string) error {
	return a.client.DeleteCollection(ctx, name)
}

// CollectionExists checks if a collection exists.
func (a *RemediationAdapter) CollectionExists(ctx context.Context, name string) (bool, error) {
	return a.client.CollectionExists(ctx, name)
}

// ListCollections returns a list of all collection names.
func (a *RemediationAdapter) ListCollections(ctx context.Context) ([]string, error) {
	return a.client.ListCollections(ctx)
}

// Upsert inserts or updates points in a collection.
func (a *RemediationAdapter) Upsert(ctx context.Context, collection string, points []*vectorstore.QdrantPoint) error {
	// Convert vectorstore points to qdrant points
	qdrantPoints := make([]*Point, len(points))
	for i, p := range points {
		qdrantPoints[i] = &Point{
			ID:      p.ID,
			Vector:  p.Vector,
			Payload: p.Payload,
		}
	}
	return a.client.Upsert(ctx, collection, qdrantPoints)
}

// Search performs similarity search in a collection.
func (a *RemediationAdapter) Search(ctx context.Context, collection string, vector []float32, limit uint64, filter *vectorstore.QdrantFilter) ([]*vectorstore.QdrantScoredPoint, error) {
	// Convert filter
	var qdrantFilter *Filter
	if filter != nil {
		qdrantFilter = &Filter{}
		for _, c := range filter.Must {
			cond := Condition{Field: c.Field, Match: c.Match}
			if c.Range != nil {
				cond.Range = &RangeCondition{
					Gte: c.Range.Gte,
					Lte: c.Range.Lte,
					Gt:  c.Range.Gt,
					Lt:  c.Range.Lt,
				}
			}
			qdrantFilter.Must = append(qdrantFilter.Must, cond)
		}
		for _, c := range filter.Should {
			cond := Condition{Field: c.Field, Match: c.Match}
			if c.Range != nil {
				cond.Range = &RangeCondition{
					Gte: c.Range.Gte,
					Lte: c.Range.Lte,
					Gt:  c.Range.Gt,
					Lt:  c.Range.Lt,
				}
			}
			qdrantFilter.Should = append(qdrantFilter.Should, cond)
		}
		for _, c := range filter.MustNot {
			cond := Condition{Field: c.Field, Match: c.Match}
			if c.Range != nil {
				cond.Range = &RangeCondition{
					Gte: c.Range.Gte,
					Lte: c.Range.Lte,
					Gt:  c.Range.Gt,
					Lt:  c.Range.Lt,
				}
			}
			qdrantFilter.MustNot = append(qdrantFilter.MustNot, cond)
		}
	}

	results, err := a.client.Search(ctx, collection, vector, limit, qdrantFilter)
	if err != nil {
		return nil, err
	}

	// Convert results
	adapterResults := make([]*vectorstore.QdrantScoredPoint, len(results))
	for i, r := range results {
		adapterResults[i] = &vectorstore.QdrantScoredPoint{
			QdrantPoint: vectorstore.QdrantPoint{
				ID:      r.ID,
				Vector:  r.Vector,
				Payload: r.Payload,
			},
			Score: r.Score,
		}
	}
	return adapterResults, nil
}

// Get retrieves points by their IDs.
func (a *RemediationAdapter) Get(ctx context.Context, collection string, ids []string) ([]*vectorstore.QdrantPoint, error) {
	results, err := a.client.Get(ctx, collection, ids)
	if err != nil {
		return nil, err
	}

	// Convert results
	adapterPoints := make([]*vectorstore.QdrantPoint, len(results))
	for i, r := range results {
		adapterPoints[i] = &vectorstore.QdrantPoint{
			ID:      r.ID,
			Vector:  r.Vector,
			Payload: r.Payload,
		}
	}
	return adapterPoints, nil
}

// Delete removes points from a collection.
func (a *RemediationAdapter) Delete(ctx context.Context, collection string, ids []string) error {
	return a.client.Delete(ctx, collection, ids)
}

// Health performs a health check on the Qdrant connection.
func (a *RemediationAdapter) Health(ctx context.Context) error {
	return a.client.Health(ctx)
}

// Close closes the client connection.
func (a *RemediationAdapter) Close() error {
	return a.client.Close()
}

// Ensure RemediationAdapter implements vectorstore.QdrantClient interface.
var _ vectorstore.QdrantClient = (*RemediationAdapter)(nil)
