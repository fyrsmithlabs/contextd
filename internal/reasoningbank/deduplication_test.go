package reasoningbank

import (
	"context"
	"testing"

	"github.com/fyrsmithlabs/contextd/internal/project"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// TestService_Search_Deduplication tests that duplicate memories are filtered from search results.
// This addresses GitHub Issue #124: duplicate memories appearing due to race conditions during
// memory updates (delete→add pattern in Feedback/RecordOutcome).
func TestService_Search_Deduplication(t *testing.T) {
	ctx := context.Background()
	store := newMockStore()
	svc, _ := NewService(store, zap.NewNop(), WithDefaultTenant("test-tenant"))

	projectID := "project-dedup"

	// Create a memory
	memory, err := NewMemory(projectID, "Dedup Test Memory", "Content to deduplicate", OutcomeSuccess, []string{"test"})
	require.NoError(t, err)
	memory.Confidence = 0.85
	memory.State = MemoryStateActive

	// Record the memory normally
	err = svc.Record(ctx, memory)
	require.NoError(t, err)

	// Simulate duplicate by directly adding the same document to the mock store's collection
	// This simulates what happens during a race condition in Feedback/RecordOutcome
	collection, _ := project.GetCollectionName(projectID, project.CollectionMemories)
	doc := svc.memoryToDocument(memory, collection)

	// Add duplicate directly to store (simulating race condition)
	store.mu.Lock()
	store.collections[collection] = append(store.collections[collection], doc)
	store.mu.Unlock()

	// The store now has 2 copies of the same memory ID
	store.mu.RLock()
	docsInCollection := len(store.collections[collection])
	store.mu.RUnlock()
	require.Equal(t, 2, docsInCollection, "store should have 2 copies (simulating race condition)")

	// Search should only return 1 result (deduplication should filter the duplicate)
	results, err := svc.Search(ctx, projectID, "Dedup Test Memory", 10)
	require.NoError(t, err)
	assert.Len(t, results, 1, "search should return only 1 memory after deduplication")
	assert.Equal(t, memory.ID, results[0].ID, "returned memory should have correct ID")
}

// TestService_Search_Deduplication_MultipleMemories tests that deduplication works correctly
// when there are multiple different memories with some duplicates.
func TestService_Search_Deduplication_MultipleMemories(t *testing.T) {
	ctx := context.Background()
	store := newMockStore()
	svc, _ := NewService(store, zap.NewNop(), WithDefaultTenant("test-tenant"))

	projectID := "project-dedup-multi"

	// Create three distinct memories
	memory1, _ := NewMemory(projectID, "Memory One", "First unique content", OutcomeSuccess, []string{"test"})
	memory1.Confidence = 0.85
	memory1.State = MemoryStateActive

	memory2, _ := NewMemory(projectID, "Memory Two", "Second unique content", OutcomeSuccess, []string{"test"})
	memory2.Confidence = 0.85
	memory2.State = MemoryStateActive

	memory3, _ := NewMemory(projectID, "Memory Three", "Third unique content", OutcomeSuccess, []string{"test"})
	memory3.Confidence = 0.85
	memory3.State = MemoryStateActive

	// Record all memories
	_ = svc.Record(ctx, memory1)
	_ = svc.Record(ctx, memory2)
	_ = svc.Record(ctx, memory3)

	// Simulate duplicates by adding copies of memory1 and memory2
	collection, _ := project.GetCollectionName(projectID, project.CollectionMemories)
	doc1 := svc.memoryToDocument(memory1, collection)
	doc2 := svc.memoryToDocument(memory2, collection)

	store.mu.Lock()
	store.collections[collection] = append(store.collections[collection], doc1, doc1, doc2) // 2 extra copies of memory1, 1 extra of memory2
	store.mu.Unlock()

	// Store now has: memory1 (x3), memory2 (x2), memory3 (x1) = 6 documents total
	store.mu.RLock()
	docsInCollection := len(store.collections[collection])
	store.mu.RUnlock()
	require.Equal(t, 6, docsInCollection, "store should have 6 documents (3+2+1)")

	// Search should return only 3 unique memories
	results, err := svc.Search(ctx, projectID, "content", 10)
	require.NoError(t, err)
	assert.Len(t, results, 3, "search should return 3 unique memories after deduplication")

	// Verify all three unique IDs are present
	ids := make(map[string]bool)
	for _, m := range results {
		ids[m.ID] = true
	}
	assert.True(t, ids[memory1.ID], "memory1 should be in results")
	assert.True(t, ids[memory2.ID], "memory2 should be in results")
	assert.True(t, ids[memory3.ID], "memory3 should be in results")
}
