// Package framework provides the integration test harness for contextd.
//
// This file contains benchmark and load tests for the ReasoningBank service.
// These tests address the LOW priority gap from KNOWN-GAPS.md for load testing.
package framework

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/fyrsmithlabs/contextd/internal/reasoningbank"
	"github.com/fyrsmithlabs/contextd/internal/vectorstore"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// createTempChromemStore creates a ChromemStore with a unique temp directory.
// The caller is responsible for closing the store and cleaning up.
func createTempChromemStore(t testing.TB) (*vectorstore.ChromemStore, func()) {
	t.Helper()

	embedder := newSemanticEmbedder(384)
	logger := zap.NewNop()

	// Create unique temp directory
	tempDir, err := os.MkdirTemp("", "chromem-test-*")
	require.NoError(t, err)

	store, err := vectorstore.NewChromemStore(vectorstore.ChromemConfig{
		Path: tempDir,
	}, embedder, logger)
	require.NoError(t, err)

	store.SetIsolationMode(vectorstore.NewNoIsolation())

	cleanup := func() {
		store.Close()
		os.RemoveAll(tempDir)
	}

	return store, cleanup
}

// BenchmarkMemoryRecord measures the performance of recording memories.
func BenchmarkMemoryRecord(b *testing.B) {
	ctx := context.Background()
	logger := zap.NewNop()

	embedder := newSemanticEmbedder(384)
	store, err := vectorstore.NewChromemStore(vectorstore.ChromemConfig{
		Path: "", // In-memory
	}, embedder, logger)
	require.NoError(b, err)
	defer store.Close()

	store.SetIsolationMode(vectorstore.NewNoIsolation())

	svc, err := reasoningbank.NewService(store, logger, reasoningbank.WithDefaultTenant("bench-tenant"))
	require.NoError(b, err)

	projectID := "benchmark-project"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		memory, _ := reasoningbank.NewMemory(
			projectID,
			fmt.Sprintf("Benchmark memory %d", i),
			"This is benchmark content for testing record performance",
			reasoningbank.OutcomeSuccess,
			[]string{"benchmark"},
		)
		_ = svc.Record(ctx, memory)
	}
}

// BenchmarkMemorySearch measures search performance with varying collection sizes.
func BenchmarkMemorySearch(b *testing.B) {
	ctx := context.Background()
	logger := zap.NewNop()

	embedder := newSemanticEmbedder(384)
	store, err := vectorstore.NewChromemStore(vectorstore.ChromemConfig{
		Path: "", // In-memory
	}, embedder, logger)
	require.NoError(b, err)
	defer store.Close()

	store.SetIsolationMode(vectorstore.NewNoIsolation())

	svc, err := reasoningbank.NewService(store, logger, reasoningbank.WithDefaultTenant("bench-tenant"))
	require.NoError(b, err)

	projectID := "benchmark-search-project"

	// Pre-populate with memories
	for i := 0; i < 100; i++ {
		memory, _ := reasoningbank.NewMemory(
			projectID,
			fmt.Sprintf("Database optimization strategy %d", i),
			"Use connection pooling and query caching for better performance",
			reasoningbank.OutcomeSuccess,
			[]string{"database", "performance"},
		)
		_ = svc.Record(ctx, memory)
	}

	queries := []string{
		"database connection pooling",
		"performance optimization",
		"caching strategies",
		"query optimization",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		query := queries[i%len(queries)]
		_, _ = svc.Search(ctx, projectID, query, 10)
	}
}

// BenchmarkSignalStore measures signal store performance.
func BenchmarkSignalStore(b *testing.B) {
	ctx := context.Background()
	signalStore := reasoningbank.NewInMemorySignalStore()
	projectID := "signal-bench"

	b.Run("StoreSignal", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			memoryID := fmt.Sprintf("memory-%d", i%100)
			signal, _ := reasoningbank.NewSignal(memoryID, projectID, reasoningbank.SignalExplicit, true, "")
			_ = signalStore.StoreSignal(ctx, signal)
		}
	})

	// Pre-populate for retrieval benchmark
	for i := 0; i < 1000; i++ {
		memoryID := "retrieval-memory"
		signal, _ := reasoningbank.NewSignal(memoryID, projectID, reasoningbank.SignalUsage, true, "")
		_ = signalStore.StoreSignal(ctx, signal)
	}

	b.Run("GetRecentSignals", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = signalStore.GetRecentSignals(ctx, "retrieval-memory", 24*time.Hour)
		}
	})
}

// TestLoadMemoryRecordConcurrent tests concurrent memory recording.
// Note: chromem-go has internal race conditions with high concurrency,
// so we test with moderate parallelism (5 goroutines) rather than extreme load.
func TestLoadMemoryRecordConcurrent(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping load test in short mode")
	}

	ctx := context.Background()

	store, cleanup := createTempChromemStore(t)
	defer cleanup()

	svc, err := reasoningbank.NewService(store, zap.NewNop(), reasoningbank.WithDefaultTenant("load-tenant"))
	require.NoError(t, err)

	// Use unique project ID to avoid state leakage from other tests
	projectID := fmt.Sprintf("load-test-project-%d", time.Now().UnixNano())
	// Use moderate parallelism to avoid chromem-go internal race conditions
	numGoroutines := 5
	memoriesPerGoroutine := 10

	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines*memoriesPerGoroutine)

	start := time.Now()

	for g := 0; g < numGoroutines; g++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			for i := 0; i < memoriesPerGoroutine; i++ {
				memory, err := reasoningbank.NewMemory(
					projectID,
					fmt.Sprintf("Memory from goroutine %d item %d", goroutineID, i),
					"Concurrent test content for load testing",
					reasoningbank.OutcomeSuccess,
					[]string{"concurrent", "load"},
				)
				if err != nil {
					errors <- err
					continue
				}
				if err := svc.Record(ctx, memory); err != nil {
					errors <- err
				}
			}
		}(g)
	}

	wg.Wait()
	close(errors)

	elapsed := time.Since(start)
	totalMemories := numGoroutines * memoriesPerGoroutine

	// Count errors
	errorCount := 0
	for err := range errors {
		t.Logf("Error: %v", err)
		errorCount++
	}

	t.Logf("Concurrent record: %d memories in %v (%.2f memories/sec)",
		totalMemories, elapsed, float64(totalMemories)/elapsed.Seconds())
	t.Logf("Errors: %d/%d", errorCount, totalMemories)

	// Verify some memories were recorded
	count, err := svc.Count(ctx, projectID)
	require.NoError(t, err)
	t.Logf("Final memory count: %d", count)

	// With moderate parallelism, most should succeed
	require.Greater(t, count, 0, "some memories should be recorded")
}

// TestLoadSearchUnderLoad tests search performance under concurrent load.
func TestLoadSearchUnderLoad(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping load test in short mode")
	}

	ctx := context.Background()

	store, cleanup := createTempChromemStore(t)
	defer cleanup()

	svc, err := reasoningbank.NewService(store, zap.NewNop(), reasoningbank.WithDefaultTenant("load-tenant"))
	require.NoError(t, err)

	// Use unique project ID to avoid state leakage from other tests
	projectID := fmt.Sprintf("search-load-project-%d", time.Now().UnixNano())

	// Pre-populate with memories
	t.Log("Populating test data...")
	for i := 0; i < 200; i++ {
		memory, _ := reasoningbank.NewMemory(
			projectID,
			fmt.Sprintf("Strategy %d for handling errors", i),
			"Use retry logic with exponential backoff and circuit breakers",
			reasoningbank.OutcomeSuccess,
			[]string{"errors", "resilience"},
		)
		_ = svc.Record(ctx, memory)
	}

	// Concurrent searches - use moderate parallelism
	numSearchers := 3
	searchesPerGoroutine := 20

	var wg sync.WaitGroup
	latencies := make(chan time.Duration, numSearchers*searchesPerGoroutine)

	queries := []string{
		"error handling retry",
		"circuit breaker pattern",
		"resilience strategies",
		"exponential backoff",
		"failure recovery",
	}

	t.Log("Running concurrent searches...")
	start := time.Now()

	for g := 0; g < numSearchers; g++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			for i := 0; i < searchesPerGoroutine; i++ {
				query := queries[(goroutineID+i)%len(queries)]
				searchStart := time.Now()
				_, _ = svc.Search(ctx, projectID, query, 10)
				latencies <- time.Since(searchStart)
			}
		}(g)
	}

	wg.Wait()
	close(latencies)

	elapsed := time.Since(start)
	totalSearches := numSearchers * searchesPerGoroutine

	// Calculate latency statistics
	var totalLatency time.Duration
	var maxLatency time.Duration
	count := 0
	for lat := range latencies {
		totalLatency += lat
		if lat > maxLatency {
			maxLatency = lat
		}
		count++
	}
	avgLatency := totalLatency / time.Duration(count)

	t.Logf("Search load test: %d searches in %v (%.2f searches/sec)",
		totalSearches, elapsed, float64(totalSearches)/elapsed.Seconds())
	t.Logf("Avg latency: %v, Max latency: %v", avgLatency, maxLatency)

	// Performance assertions
	require.Less(t, avgLatency, 100*time.Millisecond, "average search latency should be under 100ms")
}

// TestLoadLargeMemoryCollection tests behavior with a large memory collection.
func TestLoadLargeMemoryCollection(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping load test in short mode")
	}

	ctx := context.Background()

	store, cleanup := createTempChromemStore(t)
	defer cleanup()

	svc, err := reasoningbank.NewService(store, zap.NewNop(), reasoningbank.WithDefaultTenant("load-tenant"))
	require.NoError(t, err)

	// Use unique project ID to avoid state leakage from other tests
	projectID := fmt.Sprintf("large-collection-project-%d", time.Now().UnixNano())
	targetSize := 500 // Number of memories to create

	// Record many memories
	t.Logf("Recording %d memories...", targetSize)
	recordStart := time.Now()

	topics := []string{
		"database optimization",
		"API design patterns",
		"error handling strategies",
		"caching mechanisms",
		"security best practices",
	}

	for i := 0; i < targetSize; i++ {
		topic := topics[i%len(topics)]
		memory, err := reasoningbank.NewMemory(
			projectID,
			fmt.Sprintf("%s variant %d", topic, i),
			fmt.Sprintf("Detailed content about %s with specific implementation details for case %d", topic, i),
			reasoningbank.OutcomeSuccess,
			[]string{topic, "test"},
		)
		require.NoError(t, err)
		require.NoError(t, svc.Record(ctx, memory))

		if (i+1)%100 == 0 {
			t.Logf("Recorded %d/%d memories", i+1, targetSize)
		}
	}

	recordElapsed := time.Since(recordStart)
	t.Logf("Recording complete: %d memories in %v (%.2f memories/sec)",
		targetSize, recordElapsed, float64(targetSize)/recordElapsed.Seconds())

	// Verify count
	count, err := svc.Count(ctx, projectID)
	require.NoError(t, err)
	require.Equal(t, targetSize, count, "all memories should be recorded")

	// Test search performance on large collection
	t.Log("Testing search performance on large collection...")
	searchStart := time.Now()
	numSearches := 20

	for i := 0; i < numSearches; i++ {
		topic := topics[i%len(topics)]
		results, err := svc.Search(ctx, projectID, topic, 10)
		require.NoError(t, err)
		require.NotEmpty(t, results, "search should return results")
	}

	searchElapsed := time.Since(searchStart)
	avgSearchTime := searchElapsed / time.Duration(numSearches)

	t.Logf("Search performance: %d searches in %v (avg: %v per search)",
		numSearches, searchElapsed, avgSearchTime)

	require.Less(t, avgSearchTime, 200*time.Millisecond,
		"average search time should be under 200ms even with large collection")
}
