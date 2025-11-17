package prefetch

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCache(t *testing.T) {
	cache := NewCache(5*time.Minute, 100)
	require.NotNil(t, cache)
}

func TestCache_SetAndGet(t *testing.T) {
	cache := NewCache(5*time.Minute, 100)

	// Test data
	projectPath := "/tmp/test-project"
	results := []PreFetchResult{
		{
			Type:       "branch_diff",
			Data:       map[string]interface{}{"summary": "3 files changed"},
			Metadata:   map[string]string{"old_branch": "main", "new_branch": "feature"},
			Confidence: 1.0,
		},
	}

	// Set entry
	cache.Set(projectPath, results)

	// Get entry
	retrieved, ok := cache.Get(projectPath)
	assert.True(t, ok, "entry should exist")
	assert.Equal(t, projectPath, retrieved.ProjectPath)
	assert.Len(t, retrieved.Results, 1)
	assert.Equal(t, "branch_diff", retrieved.Results[0].Type)
	assert.Equal(t, 1.0, retrieved.Results[0].Confidence)
}

func TestCache_GetNonExistent(t *testing.T) {
	cache := NewCache(5*time.Minute, 100)

	_, ok := cache.Get("/nonexistent/path")
	assert.False(t, ok, "non-existent entry should return false")
}

func TestCache_ExpiredEntry(t *testing.T) {
	// Short TTL for testing
	cache := NewCache(100*time.Millisecond, 100)

	projectPath := "/tmp/test-project"
	results := []PreFetchResult{
		{Type: "branch_diff", Data: "test", Confidence: 1.0},
	}

	// Set entry
	cache.Set(projectPath, results)

	// Verify it exists
	_, ok := cache.Get(projectPath)
	assert.True(t, ok, "entry should exist immediately")

	// Wait for expiry
	time.Sleep(150 * time.Millisecond)

	// Verify it's expired
	_, ok = cache.Get(projectPath)
	assert.False(t, ok, "entry should be expired")
}

func TestCache_Delete(t *testing.T) {
	cache := NewCache(5*time.Minute, 100)

	projectPath := "/tmp/test-project"
	results := []PreFetchResult{
		{Type: "branch_diff", Data: "test", Confidence: 1.0},
	}

	// Set and verify
	cache.Set(projectPath, results)
	_, ok := cache.Get(projectPath)
	assert.True(t, ok)

	// Delete
	cache.Delete(projectPath)

	// Verify deleted
	_, ok = cache.Get(projectPath)
	assert.False(t, ok, "entry should be deleted")
}

func TestCache_Clear(t *testing.T) {
	cache := NewCache(5*time.Minute, 100)

	// Add multiple entries
	for i := 0; i < 5; i++ {
		projectPath := "/tmp/test-project-" + string(rune('0'+i))
		results := []PreFetchResult{
			{Type: "test", Data: i, Confidence: 1.0},
		}
		cache.Set(projectPath, results)
	}

	// Verify entries exist
	_, ok := cache.Get("/tmp/test-project-0")
	assert.True(t, ok)

	// Clear cache
	cache.Clear()

	// Verify all entries deleted
	for i := 0; i < 5; i++ {
		projectPath := "/tmp/test-project-" + string(rune('0'+i))
		_, ok := cache.Get(projectPath)
		assert.False(t, ok, "entry should be cleared")
	}
}

func TestCache_LRUEviction(t *testing.T) {
	// Small max size for testing LRU
	maxEntries := 3
	cache := NewCache(5*time.Minute, maxEntries)

	// Add entries up to max
	for i := 0; i < maxEntries; i++ {
		projectPath := "/tmp/project-" + string(rune('0'+i))
		results := []PreFetchResult{
			{Type: "test", Data: i, Confidence: 1.0},
		}
		cache.Set(projectPath, results)
	}

	// Verify all exist
	for i := 0; i < maxEntries; i++ {
		projectPath := "/tmp/project-" + string(rune('0'+i))
		_, ok := cache.Get(projectPath)
		assert.True(t, ok, "entry %d should exist", i)
	}

	// Add one more entry (should evict LRU)
	cache.Set("/tmp/project-new", []PreFetchResult{
		{Type: "test", Data: "new", Confidence: 1.0},
	})

	// The oldest entry (project-0) should be evicted
	_, ok := cache.Get("/tmp/project-0")
	assert.False(t, ok, "oldest entry should be evicted")

	// New entry should exist
	_, ok = cache.Get("/tmp/project-new")
	assert.True(t, ok, "new entry should exist")

	// Other entries should still exist
	_, ok = cache.Get("/tmp/project-1")
	assert.True(t, ok, "project-1 should still exist")
	_, ok = cache.Get("/tmp/project-2")
	assert.True(t, ok, "project-2 should still exist")
}

func TestCache_ConcurrentAccess(t *testing.T) {
	cache := NewCache(5*time.Minute, 100)

	var wg sync.WaitGroup
	numGoroutines := 50
	numOperations := 100

	// Concurrent writes
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				projectPath := "/tmp/project-" + string(rune('0'+id%10))
				results := []PreFetchResult{
					{Type: "test", Data: id*numOperations + j, Confidence: 1.0},
				}
				cache.Set(projectPath, results)
			}
		}(i)
	}

	// Concurrent reads
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				projectPath := "/tmp/project-" + string(rune('0'+id%10))
				cache.Get(projectPath)
			}
		}(i)
	}

	// Concurrent deletes
	for i := 0; i < numGoroutines/10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations/10; j++ {
				projectPath := "/tmp/project-" + string(rune('0'+id%10))
				cache.Delete(projectPath)
			}
		}(i)
	}

	wg.Wait()
	// If we reach here without deadlock, concurrent access works
}

func TestCache_CleanupExpiredEntries(t *testing.T) {
	// Short TTL and cleanup interval for testing
	cache := NewCache(100*time.Millisecond, 100)

	// Add entries
	for i := 0; i < 5; i++ {
		projectPath := "/tmp/project-" + string(rune('0'+i))
		results := []PreFetchResult{
			{Type: "test", Data: i, Confidence: 1.0},
		}
		cache.Set(projectPath, results)
	}

	// Wait for expiry
	time.Sleep(150 * time.Millisecond)

	// Trigger cleanup by calling Get (which checks expiry)
	for i := 0; i < 5; i++ {
		projectPath := "/tmp/project-" + string(rune('0'+i))
		_, ok := cache.Get(projectPath)
		assert.False(t, ok, "expired entry should not be returned")
	}
}

func TestCache_UpdateExistingEntry(t *testing.T) {
	cache := NewCache(5*time.Minute, 100)

	projectPath := "/tmp/test-project"

	// First set
	results1 := []PreFetchResult{
		{Type: "branch_diff", Data: "first", Confidence: 1.0},
	}
	cache.Set(projectPath, results1)

	// Get and verify first
	entry, ok := cache.Get(projectPath)
	assert.True(t, ok)
	assert.Equal(t, "first", entry.Results[0].Data)

	// Update
	results2 := []PreFetchResult{
		{Type: "branch_diff", Data: "updated", Confidence: 1.0},
	}
	cache.Set(projectPath, results2)

	// Get and verify update
	entry, ok = cache.Get(projectPath)
	assert.True(t, ok)
	assert.Equal(t, "updated", entry.Results[0].Data)
}
