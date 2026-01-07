package framework

import (
	"context"
	"testing"

	"github.com/fyrsmithlabs/contextd/internal/reasoningbank"
	"github.com/fyrsmithlabs/contextd/internal/vectorstore"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// TestDebug_SemanticSearch tests the basic flow of semantic search.
func TestDebug_SemanticSearch(t *testing.T) {
	ctx := context.Background()
	logger, _ := zap.NewDevelopment()

	// Create chromem store with our embedder
	embedder := newTestEmbedder(384) // Use the existing deterministic embedder
	store, err := vectorstore.NewChromemStore(vectorstore.ChromemConfig{
		Path: "", // In-memory
	}, embedder, logger)
	require.NoError(t, err)
	defer store.Close()

	store.SetIsolationMode(vectorstore.NewNoIsolation())

	// Create service with tenant
	svc, err := reasoningbank.NewService(store, logger, reasoningbank.WithDefaultTenant("test-tenant"))
	require.NoError(t, err)

	projectID := "debug-project"

	// Record a memory
	t.Log("Recording memory...")
	memory, err := reasoningbank.NewMemory(
		projectID,
		"Test memory title",
		"Test memory content about database connections",
		reasoningbank.OutcomeSuccess,
		[]string{"test"},
	)
	require.NoError(t, err)
	t.Logf("Memory ID: %s, Confidence: %.2f", memory.ID, memory.Confidence)

	err = svc.Record(ctx, memory)
	require.NoError(t, err)
	t.Log("Memory recorded successfully")

	// Check collection info
	collections, err := store.ListCollections(ctx)
	require.NoError(t, err)
	t.Logf("Collections: %v", collections)

	for _, coll := range collections {
		info, err := store.GetCollectionInfo(ctx, coll)
		if err != nil {
			t.Logf("Collection %s: error getting info: %v", coll, err)
		} else {
			t.Logf("Collection %s: %d documents", coll, info.PointCount)
		}
	}

	// Search directly on store
	t.Log("Searching on store directly...")
	for _, coll := range collections {
		results, err := store.SearchInCollection(ctx, coll, "database connections", 10, nil)
		if err != nil {
			t.Logf("Search in %s error: %v", coll, err)
		} else {
			t.Logf("Search in %s: %d results", coll, len(results))
			for i, r := range results {
				t.Logf("  %d: ID=%s Score=%.4f Confidence=%.2f", i, r.ID, r.Score, r.Metadata["confidence"])
			}
		}
	}

	// Search for the specific collection directly to debug
	collName := "debug_project_memories"
	t.Logf("Searching collection %s directly...", collName)
	directResults, err := store.SearchInCollection(ctx, collName, "database connections", 10, nil)
	require.NoError(t, err)
	t.Logf("Direct search returned %d results", len(directResults))
	for i, r := range directResults {
		confVal := r.Metadata["confidence"]
		t.Logf("  %d: ID=%s Score=%.4f ConfType=%T ConfVal=%v", i, r.ID, r.Score, confVal, confVal)
	}

	// Search via service
	t.Log("Searching via service...")
	results, err := svc.Search(ctx, projectID, "database connections", 10)
	require.NoError(t, err)
	t.Logf("Service search returned %d results", len(results))
	for i, r := range results {
		t.Logf("  %d: ID=%s Title=%s Confidence=%.2f", i, r.ID, r.Title, r.Confidence)
	}
}
