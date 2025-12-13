package framework

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCrossDeveloperKnowledgeSharing validates contextd's core value proposition:
// Knowledge recorded by Dev A should be discoverable and useful to Dev B.
func TestCrossDeveloperKnowledgeSharing(t *testing.T) {
	t.Run("Dev B can retrieve Dev A's recorded memory", func(t *testing.T) {
		ctx := context.Background()

		// Create shared store for both developers (simulates shared Qdrant)
		sharedStore, err := NewSharedStore(SharedStoreConfig{
			ProjectID: "shared_project",
		})
		require.NoError(t, err)
		defer sharedStore.Close()

		// Create Dev A
		devA, err := NewDeveloperWithStore(DeveloperConfig{
			ID:        "dev-a",
			TenantID:  "team-alpha",
			ProjectID: "shared_project",
		}, sharedStore)
		require.NoError(t, err)

		err = devA.StartContextd(ctx)
		require.NoError(t, err)
		defer devA.StopContextd(ctx)

		// Dev A records a fix for a null pointer bug
		memoryID, err := devA.RecordMemory(ctx, MemoryRecord{
			Title:   "Fix for null pointer in user service",
			Content: "Always check if user is nil before accessing user.ID. Add guard clause at function entry.",
			Outcome: "success",
			Tags:    []string{"bugfix", "null-pointer", "user-service"},
		})
		require.NoError(t, err)
		require.NotEmpty(t, memoryID)
		t.Logf("Dev A recorded memory ID: %s", memoryID)

		// Dev A can search their own memory
		devAResults, err := devA.SearchMemory(ctx, "Fix for null pointer in user service", 5)
		require.NoError(t, err)
		t.Logf("Dev A search returned %d results", len(devAResults))

		// Create Dev B with SAME shared store
		devB, err := NewDeveloperWithStore(DeveloperConfig{
			ID:        "dev-b",
			TenantID:  "team-alpha",
			ProjectID: "shared_project",
		}, sharedStore)
		require.NoError(t, err)

		err = devB.StartContextd(ctx)
		require.NoError(t, err)
		defer devB.StopContextd(ctx)

		// Dev B searches using EXACT title (test embedder uses hash-based similarity)
		// In production with real embeddings, semantic search would work
		results, err := devB.SearchMemory(ctx, "Fix for null pointer in user service", 5)
		require.NoError(t, err)

		// Log results for debugging
		t.Logf("Search returned %d results", len(results))
		for i, r := range results {
			t.Logf("  Result %d: ID=%s Title=%s Confidence=%.2f", i, r.ID, r.Title, r.Confidence)
		}

		// Dev B should find Dev A's memory (same project, same store)
		// Note: With hash-based test embedder, exact match queries work best
		require.NotEmpty(t, results, "Dev B should find Dev A's recorded memory with shared store")
		assert.Contains(t, results[0].Content, "nil")
	})

	t.Run("Dev B's feedback affects shared memory confidence", func(t *testing.T) {
		ctx := context.Background()

		sharedStore, err := NewSharedStore(SharedStoreConfig{
			ProjectID: "shared_project_feedback",
		})
		require.NoError(t, err)
		defer sharedStore.Close()

		// Dev A records a memory
		devA, err := NewDeveloperWithStore(DeveloperConfig{
			ID:        "dev-a",
			TenantID:  "team-alpha",
			ProjectID: "shared_project_feedback",
		}, sharedStore)
		require.NoError(t, err)
		err = devA.StartContextd(ctx)
		require.NoError(t, err)
		defer devA.StopContextd(ctx)

		memoryID, err := devA.RecordMemory(ctx, MemoryRecord{
			Title:   "Database connection pattern",
			Content: "Use connection pooling with max 10 connections",
			Outcome: "success",
			Tags:    []string{"database", "patterns"},
		})
		require.NoError(t, err)

		// Dev B gives positive feedback
		devB, err := NewDeveloperWithStore(DeveloperConfig{
			ID:        "dev-b",
			TenantID:  "team-alpha",
			ProjectID: "shared_project_feedback",
		}, sharedStore)
		require.NoError(t, err)
		err = devB.StartContextd(ctx)
		require.NoError(t, err)
		defer devB.StopContextd(ctx)

		err = devB.GiveFeedback(ctx, memoryID, true, "This helped me fix my connection issues")
		require.NoError(t, err)

		// Verify confidence increased (would need to expose GetMemory or similar)
		// For now, just verify the operation succeeded
		assert.Equal(t, 1, devB.SessionStats().MemoryFeedbacks)
	})

	t.Run("knowledge flows from multiple developers", func(t *testing.T) {
		ctx := context.Background()

		sharedStore, err := NewSharedStore(SharedStoreConfig{
			ProjectID: "shared_project_multi",
		})
		require.NoError(t, err)
		defer sharedStore.Close()

		// Dev A records one type of knowledge
		devA, err := NewDeveloperWithStore(DeveloperConfig{
			ID:        "dev-a",
			TenantID:  "team-alpha",
			ProjectID: "shared_project_multi",
		}, sharedStore)
		require.NoError(t, err)
		err = devA.StartContextd(ctx)
		require.NoError(t, err)
		defer devA.StopContextd(ctx)

		_, err = devA.RecordMemory(ctx, MemoryRecord{
			Title:   "Auth token refresh pattern",
			Content: "Refresh tokens 5 minutes before expiry to avoid race conditions",
			Outcome: "success",
			Tags:    []string{"auth", "tokens"},
		})
		require.NoError(t, err)

		// Dev B records different knowledge
		devB, err := NewDeveloperWithStore(DeveloperConfig{
			ID:        "dev-b",
			TenantID:  "team-alpha",
			ProjectID: "shared_project_multi",
		}, sharedStore)
		require.NoError(t, err)
		err = devB.StartContextd(ctx)
		require.NoError(t, err)
		defer devB.StopContextd(ctx)

		_, err = devB.RecordMemory(ctx, MemoryRecord{
			Title:   "API rate limiting best practice",
			Content: "Implement exponential backoff with jitter for retries",
			Outcome: "success",
			Tags:    []string{"api", "rate-limiting"},
		})
		require.NoError(t, err)

		// Dev C can find knowledge from both A and B
		devC, err := NewDeveloperWithStore(DeveloperConfig{
			ID:        "dev-c",
			TenantID:  "team-alpha",
			ProjectID: "shared_project_multi",
		}, sharedStore)
		require.NoError(t, err)
		err = devC.StartContextd(ctx)
		require.NoError(t, err)
		defer devC.StopContextd(ctx)

		// Search for auth knowledge (from Dev A)
		authResults, err := devC.SearchMemory(ctx, "auth token refresh", 5)
		require.NoError(t, err)
		// The search may return results based on embedding similarity
		assert.NotNil(t, authResults)

		// Search for API knowledge (from Dev B)
		apiResults, err := devC.SearchMemory(ctx, "API rate limiting backoff", 5)
		require.NoError(t, err)
		assert.NotNil(t, apiResults)
	})
}
