// Package vectorstore provides vector storage implementations.
package vectorstore

import (
	"context"
)

// TroubleshootAdapter adapts Store to implement troubleshoot.VectorStore interface.
type TroubleshootAdapter struct {
	store Store
}

// NewTroubleshootAdapter creates an adapter for troubleshoot service.
func NewTroubleshootAdapter(store Store) *TroubleshootAdapter {
	return &TroubleshootAdapter{store: store}
}

// AddDocuments adds documents to the vector store.
// Returns nil on success (discards the returned IDs since troubleshoot doesn't need them).
func (a *TroubleshootAdapter) AddDocuments(ctx context.Context, docs []Document) error {
	_, err := a.store.AddDocuments(ctx, docs)
	return err
}

// SearchWithFilters performs similarity search with metadata filters.
func (a *TroubleshootAdapter) SearchWithFilters(ctx context.Context, query string, k int, filters map[string]interface{}) ([]SearchResult, error) {
	return a.store.SearchWithFilters(ctx, query, k, filters)
}
