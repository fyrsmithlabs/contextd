package reasoningbank

import (
	"context"
	"math"
	"sort"
	"testing"
	"time"

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

// TestService_SearchWithScores tests that SearchWithScores returns relevance scores.
// This addresses GitHub Issue #125: flat 0.80 relevance scores due to returning
// confidence instead of search similarity.
func TestService_SearchWithScores(t *testing.T) {
	ctx := context.Background()
	store := newMockStore()
	svc, _ := NewService(store, zap.NewNop(), WithDefaultTenant("test-tenant"))

	projectID := "project-scores"

	// Create a memory (using lowercase words that won't match entity extraction)
	memory, err := NewMemory(projectID, "error handling strategies", "how to handle errors in go code", OutcomeSuccess, []string{"test"})
	require.NoError(t, err)
	memory.Confidence = 0.85
	memory.State = MemoryStateActive

	// Record the memory
	err = svc.Record(ctx, memory)
	require.NoError(t, err)

	// SearchWithScores should return both memory and relevance score
	// Use lowercase query to avoid entity extraction/boosting
	results, err := svc.SearchWithScores(ctx, projectID, "error handling", 10)
	require.NoError(t, err)
	require.Len(t, results, 1, "should return 1 result")

	// Check that relevance is the search similarity (0.9 from mock), not confidence (0.85)
	assert.Equal(t, memory.ID, results[0].Memory.ID, "memory ID should match")
	assert.Equal(t, 0.85, results[0].Memory.Confidence, "memory confidence should be 0.85")
	assert.InDelta(t, 0.9, results[0].Relevance, 0.01, "relevance should be ~0.9 (mock search score)")
}

// TestService_SearchWithScores_RelevanceVsConfidence verifies that relevance and
// confidence are independent values.
func TestService_SearchWithScores_RelevanceVsConfidence(t *testing.T) {
	ctx := context.Background()
	store := newMockStore()
	svc, _ := NewService(store, zap.NewNop(), WithDefaultTenant("test-tenant"))

	projectID := "project-rel-conf"

	// Create memories with different confidence levels
	highConf, _ := NewMemory(projectID, "High Confidence", "Very reliable pattern", OutcomeSuccess, []string{})
	highConf.Confidence = 0.95
	highConf.State = MemoryStateActive

	lowConf, _ := NewMemory(projectID, "Low Confidence", "Uncertain pattern", OutcomeSuccess, []string{})
	lowConf.Confidence = 0.72 // Just above MinConfidence (0.7)
	lowConf.State = MemoryStateActive

	_ = svc.Record(ctx, highConf)
	_ = svc.Record(ctx, lowConf)

	results, err := svc.SearchWithScores(ctx, projectID, "pattern", 10)
	require.NoError(t, err)
	require.Len(t, results, 2, "should return both memories")

	// Both should have same relevance (0.9 from mock) but different confidences
	for _, r := range results {
		// Relevance is from search, not confidence
		assert.InDelta(t, 0.9, r.Relevance, 0.01, "relevance should be search score")
		// Confidence should match original values
		if r.Memory.ID == highConf.ID {
			assert.Equal(t, 0.95, r.Memory.Confidence)
		} else if r.Memory.ID == lowConf.ID {
			assert.Equal(t, 0.72, r.Memory.Confidence)
		}
		// They should be different
		assert.False(t, math.Abs(r.Relevance-r.Memory.Confidence) < 0.01,
			"relevance and confidence should be independent values")
	}
}

// TestService_ExtractQueryEntities tests named entity extraction from queries.
// This addresses GitHub Issue #126: query understanding with entity extraction.
func TestService_ExtractQueryEntities(t *testing.T) {
	svc, _ := NewService(newMockStore(), zap.NewNop(), WithDefaultTenant("test-tenant"))

	tests := []struct {
		name     string
		query    string
		expected []string
	}{
		{
			name:     "single name",
			query:    "What is Caroline's identity?",
			expected: []string{"Caroline"},
		},
		{
			name:     "multiple names",
			query:    "Tell me about John and Alice",
			expected: []string{"John", "Alice"},
		},
		{
			name:     "no entities",
			query:    "how do i handle errors?",
			expected: nil,
		},
		{
			name:     "duplicate names deduplicated",
			query:    "Did Alice meet Alice yesterday?",
			expected: []string{"Alice"},
		},
		{
			name:     "name at start of sentence",
			query:    "Bob went to the store",
			expected: []string{"Bob"},
		},
		{
			name:     "possessive name",
			query:    "Where is David's book?",
			expected: []string{"David"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entities := svc.extractQueryEntities(tt.query)
			if tt.expected == nil {
				assert.Nil(t, entities)
			} else {
				// Sort both slices for comparison since order may vary
				sort.Strings(entities)
				sort.Strings(tt.expected)
				assert.Equal(t, tt.expected, entities)
			}
		})
	}
}

// TestService_MemoryContainsEntity tests entity matching in memories.
func TestService_MemoryContainsEntity(t *testing.T) {
	svc, _ := NewService(newMockStore(), zap.NewNop(), WithDefaultTenant("test-tenant"))

	memory := &Memory{
		Title:       "Caroline's identity discussion",
		Content:     "Caroline mentioned she is a transgender woman",
		Description: "From conversation with Caroline",
	}

	tests := []struct {
		name     string
		entities []string
		expected bool
	}{
		{
			name:     "entity in title",
			entities: []string{"Caroline"},
			expected: true,
		},
		{
			name:     "entity not present",
			entities: []string{"Alice"},
			expected: false,
		},
		{
			name:     "case insensitive match",
			entities: []string{"CAROLINE"},
			expected: true,
		},
		{
			name:     "empty entities",
			entities: []string{},
			expected: false,
		},
		{
			name:     "multiple entities one matches",
			entities: []string{"Alice", "Caroline", "Bob"},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := svc.memoryContainsEntity(memory, tt.entities)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestService_Search_EntityBoost tests that memories mentioning query entities
// get boosted relevance scores.
func TestService_Search_EntityBoost(t *testing.T) {
	ctx := context.Background()
	store := newMockStore()
	svc, _ := NewService(store, zap.NewNop(), WithDefaultTenant("test-tenant"))

	projectID := "project-entity-boost"

	// Create two memories: one mentions Caroline, one doesn't
	carolineMemory, _ := NewMemory(projectID, "About Caroline", "Caroline is a software engineer who loves Go", OutcomeSuccess, []string{})
	carolineMemory.Confidence = 0.85
	carolineMemory.State = MemoryStateActive

	genericMemory, _ := NewMemory(projectID, "General info", "Software engineering best practices", OutcomeSuccess, []string{})
	genericMemory.Confidence = 0.85
	genericMemory.State = MemoryStateActive

	_ = svc.Record(ctx, carolineMemory)
	_ = svc.Record(ctx, genericMemory)

	// Search with Caroline's name - entity boost should apply
	results, err := svc.SearchWithScores(ctx, projectID, "What is Caroline's background?", 10)
	require.NoError(t, err)
	require.Len(t, results, 2)

	// The memory mentioning Caroline should have higher relevance (boosted by 1.3x)
	var carolineResult, genericResult ScoredMemory
	for _, r := range results {
		if r.Memory.ID == carolineMemory.ID {
			carolineResult = r
		} else {
			genericResult = r
		}
	}

	// Caroline memory should be boosted (0.9 * 1.3 = 1.17)
	// Generic memory stays at 0.9
	assert.Greater(t, carolineResult.Relevance, genericResult.Relevance,
		"memory mentioning Caroline should have higher relevance")
	assert.InDelta(t, 0.9*1.3, carolineResult.Relevance, 0.01,
		"Caroline memory should be boosted by 1.3x")
	assert.InDelta(t, 0.9, genericResult.Relevance, 0.01,
		"generic memory should not be boosted")

	// Caroline memory should be ranked first
	assert.Equal(t, carolineMemory.ID, results[0].Memory.ID,
		"Caroline memory should be ranked first due to entity boost")
}

// TestService_IsTemporalQuery tests detection of time-sensitive queries.
func TestService_IsTemporalQuery(t *testing.T) {
	svc, _ := NewService(newMockStore(), zap.NewNop(), WithDefaultTenant("test-tenant"))

	tests := []struct {
		name     string
		query    string
		expected bool
	}{
		{"recent keyword", "what did i do recently?", true},
		{"yesterday keyword", "what happened yesterday with the bug?", true},
		{"last keyword", "show me the last error fix", true},
		{"past week", "errors from the past week", true},
		{"no temporal", "how do i handle errors?", false},
		{"case insensitive", "RECENTLY fixed bugs", true},
		{"earlier keyword", "the earlier approach to caching", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := svc.isTemporalQuery(tt.query)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestService_GetTemporalMultiplier tests age-based scoring multipliers.
func TestService_GetTemporalMultiplier(t *testing.T) {
	svc, _ := NewService(newMockStore(), zap.NewNop(), WithDefaultTenant("test-tenant"))

	now := time.Now()

	tests := []struct {
		name       string
		age        time.Duration
		expected   float32
		comparison string
	}{
		{"1 day old (recent)", 1 * 24 * time.Hour, 1.25, "boost"},
		{"5 days old (recent)", 5 * 24 * time.Hour, 1.25, "boost"},
		{"14 days old (medium)", 14 * 24 * time.Hour, 1.0, "neutral"},
		{"45 days old (old)", 45 * 24 * time.Hour, 0.8, "penalty"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			memory := &Memory{
				UpdatedAt: now.Add(-tt.age),
			}
			result := svc.getTemporalMultiplier(memory)
			assert.InDelta(t, tt.expected, result, 0.01, tt.comparison)
		})
	}
}

// TestService_Search_TemporalBoost tests that recent memories get boosted
// for temporal queries.
func TestService_Search_TemporalBoost(t *testing.T) {
	ctx := context.Background()
	store := newMockStore()
	svc, _ := NewService(store, zap.NewNop(), WithDefaultTenant("test-tenant"))

	projectID := "project-temporal"

	// Create two memories - we'll set timestamps via metadata directly
	// since Record() always sets UpdatedAt = now
	recentMemory, _ := NewMemory(projectID, "recent fix", "fixed the bug using retry logic", OutcomeSuccess, []string{})
	recentMemory.Confidence = 0.85
	recentMemory.State = MemoryStateActive

	oldMemory, _ := NewMemory(projectID, "old approach", "used a different approach for the same issue", OutcomeSuccess, []string{})
	oldMemory.Confidence = 0.85
	oldMemory.State = MemoryStateActive

	// Record both memories
	_ = svc.Record(ctx, recentMemory)
	_ = svc.Record(ctx, oldMemory)

	// Manually update timestamps in the mock store to simulate age differences
	// Record() always sets UpdatedAt = now, so we override via metadata
	collection, _ := project.GetCollectionName(projectID, project.CollectionMemories)
	store.mu.Lock()
	for i := range store.collections[collection] {
		if store.collections[collection][i].ID == recentMemory.ID {
			store.collections[collection][i].Metadata["updated_at"] = time.Now().Add(-2 * 24 * time.Hour).Unix() // 2 days ago
		}
		if store.collections[collection][i].ID == oldMemory.ID {
			store.collections[collection][i].Metadata["updated_at"] = time.Now().Add(-60 * 24 * time.Hour).Unix() // 60 days ago
		}
	}
	store.mu.Unlock()

	// Search with temporal query - recent memory should be boosted
	results, err := svc.SearchWithScores(ctx, projectID, "what did i recently fix?", 10)
	require.NoError(t, err)
	require.Len(t, results, 2)

	// Find results by ID
	var recentResult, oldResult ScoredMemory
	for _, r := range results {
		if r.Memory.ID == recentMemory.ID {
			recentResult = r
		} else {
			oldResult = r
		}
	}

	// Recent memory should have higher relevance (0.9 * 1.25 = 1.125)
	// Old memory should be penalized (0.9 * 0.8 = 0.72)
	assert.Greater(t, recentResult.Relevance, oldResult.Relevance,
		"recent memory should have higher relevance for temporal query")

	// Recent memory should be ranked first
	assert.Equal(t, recentMemory.ID, results[0].Memory.ID,
		"recent memory should be ranked first for temporal query")
}
