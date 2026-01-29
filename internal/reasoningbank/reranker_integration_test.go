package reasoningbank

import (
	"context"
	"testing"

	"github.com/fyrsmithlabs/contextd/internal/reranker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// TestService_WithReranker tests that the reranker integrates correctly with the Service.
func TestService_WithReranker(t *testing.T) {
	// Create in-memory mock vectorstore
	store := newMockStore()

	logger := zap.NewNop()
	defaultTenant := "test-tenant"

	// Create service WITHOUT reranker
	serviceNoReranker, err := NewService(store, logger,
		WithSignalStore(NewInMemorySignalStore()),
		WithDefaultTenant(defaultTenant))
	require.NoError(t, err)

	// Create service WITH reranker
	rerankerInstance := reranker.NewSimpleReranker()
	serviceWithReranker, err := NewService(store, logger,
		WithSignalStore(NewInMemorySignalStore()),
		WithDefaultTenant(defaultTenant),
		WithReranker(rerankerInstance))
	require.NoError(t, err)

	ctx := context.Background()
	projectID := "reranker-test"

	// Record some test memories
	testCases := []struct {
		title   string
		content string
	}{
		{
			title:   "Authentication error handling",
			content: "When handling authentication errors, use retry with exponential backoff. Check token validity before making requests.",
		},
		{
			title:   "Database optimization techniques",
			content: "Optimize database queries by adding appropriate indexes. Monitor query performance and use caching for frequently accessed data.",
		},
		{
			title:   "Error recovery patterns",
			content: "Implement error recovery patterns including circuit breakers and fallback mechanisms. Authentication errors should trigger token refresh.",
		},
		{
			title:   "Network timeouts",
			content: "Handle network timeouts gracefully. Implement exponential backoff and retry logic for transient failures.",
		},
	}

	for i, tc := range testCases {
		mem, err := NewMemory(projectID, tc.title, tc.content, OutcomeSuccess, []string{"testing"})
		require.NoError(t, err)
		mem.Confidence = 0.8
		err = serviceWithReranker.Record(ctx, mem)
		require.NoError(t, err, "Failed to record memory %d", i)
	}

	t.Run("reranker improves term overlap ranking", func(t *testing.T) {
		// Query with specific terms that should trigger reranking
		query := "authentication retry"

		// Search WITHOUT reranker
		resultsNoReranker, err := serviceNoReranker.Search(ctx, projectID, query, 10)
		require.NoError(t, err)

		// Search WITH reranker
		resultsWithReranker, err := serviceWithReranker.Search(ctx, projectID, query, 10)
		require.NoError(t, err)

		// Both should return results
		assert.NotEmpty(t, resultsNoReranker, "Results without reranker should not be empty")
		assert.NotEmpty(t, resultsWithReranker, "Results with reranker should not be empty")

		// With reranker, the first result should be the one with highest term overlap
		// The memory with "Authentication error handling" has the highest overlap with "authentication retry"
		firstResult := resultsWithReranker[0]
		assert.Contains(t, firstResult.Title, "Authentication", "First result should mention Authentication")
		assert.Contains(t, firstResult.Content, "retry", "First result should mention retry")
	})

	t.Run("reranker handles low confidence filtering", func(t *testing.T) {
		// Search with query - both should work without errors
		resultsNoReranker, err := serviceNoReranker.Search(ctx, projectID, "optimization", 10)
		require.NoError(t, err)

		resultsWithReranker, err := serviceWithReranker.Search(ctx, projectID, "optimization", 10)
		require.NoError(t, err)

		// Both should return the same result count (both use same confidence filtering)
		assert.Equal(t, len(resultsNoReranker), len(resultsWithReranker), "Result counts should match")

		// Results should be non-empty
		assert.NotEmpty(t, resultsNoReranker)
		assert.NotEmpty(t, resultsWithReranker)
	})

	t.Run("reranker respects result limits", func(t *testing.T) {
		query := "error"
		limit := 2

		resultsWithReranker, err := serviceWithReranker.Search(ctx, projectID, query, limit)
		require.NoError(t, err)

		assert.LessOrEqual(t, len(resultsWithReranker), limit, "Results should respect limit")
	})
}

// BenchmarkService_SearchWithoutReranker benchmarks search without reranking.
func BenchmarkService_SearchWithoutReranker(b *testing.B) {
	store := newMockStore()

	logger := zap.NewNop()
	svc, err := NewService(store, logger,
		WithSignalStore(NewInMemorySignalStore()),
		WithDefaultTenant("bench-tenant"))
	if err != nil {
		b.Fatalf("Failed to create service: %v", err)
	}

	ctx := context.Background()
	projectID := "bench-project"

	// Record test memories
	for i := 0; i < 50; i++ {
		mem := &Memory{
			Title:      "Error handling pattern " + string(rune(i)),
			Content:    "This is error handling pattern for error recovery and retry logic with exponential backoff",
			Outcome:    OutcomeSuccess,
			Confidence: 0.8,
		}
		_ = svc.Record(ctx, mem)
	}

	query := "error retry backoff"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = svc.Search(ctx, projectID, query, 10)
	}
}

// BenchmarkService_SearchWithReranker benchmarks search with reranking.
func BenchmarkService_SearchWithReranker(b *testing.B) {
	store := newMockStore()

	logger := zap.NewNop()
	rerankerInstance := reranker.NewSimpleReranker()
	svc, err := NewService(store, logger,
		WithSignalStore(NewInMemorySignalStore()),
		WithDefaultTenant("bench-tenant"),
		WithReranker(rerankerInstance))
	if err != nil {
		b.Fatalf("Failed to create service: %v", err)
	}

	ctx := context.Background()
	projectID := "bench-project"

	// Record test memories
	for i := 0; i < 50; i++ {
		mem := &Memory{
			Title:      "Error handling pattern " + string(rune(i)),
			Content:    "This is error handling pattern for error recovery and retry logic with exponential backoff",
			Outcome:    OutcomeSuccess,
			Confidence: 0.8,
		}
		_ = svc.Record(ctx, mem)
	}

	query := "error retry backoff"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = svc.Search(ctx, projectID, query, 10)
	}
}
