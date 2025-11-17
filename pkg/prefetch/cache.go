// Package prefetch provides git-centric pre-fetching engine for contextd.
//
// This package implements deterministic pre-fetching based on git events
// (branch switches, new commits) to eliminate wasteful round trips and
// reduce context token usage.
//
// Example usage:
//
//	cache := prefetch.NewCache(5*time.Minute, 100)
//	cache.Set("/path/to/project", results)
//	entry, ok := cache.Get("/path/to/project")
package prefetch

import (
	"sync"
	"time"
)

// PreFetchResult represents a single pre-fetch result from a rule.
type PreFetchResult struct {
	// Type identifies the rule that produced this result
	// (e.g., "branch_diff", "related_files", "recent_commit")
	Type string

	// Data contains the rule-specific result data
	Data interface{}

	// Metadata contains additional context (branch names, timestamps, etc.)
	Metadata map[string]string

	// Confidence score (always 1.0 for deterministic rules)
	Confidence float64
}

// CacheEntry represents a cached set of pre-fetch results for a project.
type CacheEntry struct {
	// ProjectPath is the absolute path to the project
	ProjectPath string

	// Results are the pre-fetched data from all executed rules
	Results []PreFetchResult

	// ExpiresAt is when this entry should be evicted
	ExpiresAt time.Time

	// CreatedAt is when this entry was created
	CreatedAt time.Time

	// lastAccessed tracks LRU eviction (internal use only)
	lastAccessed time.Time
}

// Cache provides thread-safe in-memory caching with TTL and LRU eviction.
type Cache struct {
	mu         sync.RWMutex
	entries    map[string]*CacheEntry
	ttl        time.Duration
	maxEntries int
	metrics    *Metrics // Optional metrics tracking
}

// NewCache creates a new cache with the specified TTL and maximum entries.
//
// Parameters:
//   - ttl: Time-to-live for cache entries
//   - maxEntries: Maximum number of entries (LRU eviction when exceeded)
//
// Returns a new Cache instance.
func NewCache(ttl time.Duration, maxEntries int) *Cache {
	return &Cache{
		entries:    make(map[string]*CacheEntry),
		ttl:        ttl,
		maxEntries: maxEntries,
		metrics:    nil, // Metrics are optional, set via SetMetrics
	}
}

// SetMetrics sets the metrics tracker for this cache.
// This is optional and should be called after cache creation if metrics are desired.
func (c *Cache) SetMetrics(m *Metrics) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.metrics = m
}

// Set stores pre-fetch results for a project.
//
// If an entry already exists for this project, it will be replaced.
// If the cache is at maximum capacity, the least recently used entry
// will be evicted.
//
// This operation is thread-safe.
func (c *Cache) Set(projectPath string, results []PreFetchResult) {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()

	// Check if we need to evict an entry (LRU)
	if len(c.entries) >= c.maxEntries {
		// Find entry that hasn't been updated/created (not just accessed)
		if _, exists := c.entries[projectPath]; !exists {
			c.evictLRU()
		}
	}

	// Create new entry
	entry := &CacheEntry{
		ProjectPath:  projectPath,
		Results:      results,
		ExpiresAt:    now.Add(c.ttl),
		CreatedAt:    now,
		lastAccessed: now,
	}

	c.entries[projectPath] = entry

	// Update cache size metric
	if c.metrics != nil {
		c.metrics.SetCacheSize(len(c.entries))
	}
}

// Get retrieves pre-fetch results for a project.
//
// Returns:
//   - entry: The cache entry if it exists and is not expired
//   - ok: true if entry exists and is valid, false otherwise
//
// Expired entries are automatically removed from the cache.
// This operation is thread-safe.
func (c *Cache) Get(projectPath string) (*CacheEntry, bool) {
	c.mu.RLock()
	entry, exists := c.entries[projectPath]
	metrics := c.metrics
	c.mu.RUnlock()

	if !exists {
		// Record cache miss
		if metrics != nil {
			metrics.RecordCacheMiss()
		}
		return nil, false
	}

	// Check expiry
	if time.Now().After(entry.ExpiresAt) {
		// Entry expired, remove it
		c.mu.Lock()
		delete(c.entries, projectPath)
		if c.metrics != nil {
			c.metrics.SetCacheSize(len(c.entries))
		}
		c.mu.Unlock()

		// Record cache miss (expired = miss)
		if metrics != nil {
			metrics.RecordCacheMiss()
		}
		return nil, false
	}

	// Update last accessed time (for LRU)
	c.mu.Lock()
	entry.lastAccessed = time.Now()
	c.mu.Unlock()

	// Record cache hit with estimated token savings
	// Rough estimate: 100 tokens per result
	if metrics != nil {
		estimatedTokens := len(entry.Results) * 100
		metrics.RecordCacheHit(estimatedTokens)
	}

	return entry, true
}

// Delete removes an entry from the cache.
//
// This operation is thread-safe and is a no-op if the entry doesn't exist.
func (c *Cache) Delete(projectPath string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, projectPath)

	// Update cache size metric
	if c.metrics != nil {
		c.metrics.SetCacheSize(len(c.entries))
	}
}

// Clear removes all entries from the cache.
//
// This operation is thread-safe.
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = make(map[string]*CacheEntry)

	// Update cache size metric
	if c.metrics != nil {
		c.metrics.SetCacheSize(0)
	}
}

// evictLRU removes the least recently used entry from the cache.
//
// This is called internally when the cache is at maximum capacity.
// Caller must hold the write lock.
func (c *Cache) evictLRU() {
	var oldestPath string
	var oldestTime time.Time

	// Find entry with oldest access time
	first := true
	for path, entry := range c.entries {
		if first || entry.lastAccessed.Before(oldestTime) {
			oldestPath = path
			oldestTime = entry.lastAccessed
			first = false
		}
	}

	// Delete oldest entry
	if oldestPath != "" {
		delete(c.entries, oldestPath)
	}
}
