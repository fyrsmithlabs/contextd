package vectorstore

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fyrsmithlabs/contextd/internal/secrets"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// TestStress_HealthCallbackConcurrency stress tests the health callback worker pool
// with extreme concurrent health changes and callback executions.
func TestStress_HealthCallbackConcurrency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	duration := getStressDuration()
	workers := getStressWorkers()

	logger := zap.NewNop()
	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()

	checker := NewMockHealthChecker()
	monitor := NewHealthMonitor(ctx, checker, 10*time.Millisecond, logger)

	var callbackExecutions atomic.Int64
	var callbackErrors atomic.Int64

	// Register many callbacks
	for i := 0; i < workers; i++ {
		err := monitor.RegisterCallback(func(healthy bool) {
			// Simulate varying callback durations
			delay := time.Duration(rand.Intn(50)) * time.Millisecond
			time.Sleep(delay)
			callbackExecutions.Add(1)
		})
		if err != nil {
			callbackErrors.Add(1)
		}
	}

	monitor.Start()

	// Rapid health flapping
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(5 * time.Millisecond)
		defer ticker.Stop()

		state := false
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				state = !state
				checker.SetHealthy(state)
				monitor.updateHealth(state)
			}
		}
	}()

	// Wait for test duration
	<-ctx.Done()
	wg.Wait()
	time.Sleep(200 * time.Millisecond) // Allow final callbacks to complete

	monitor.Stop()

	executions := callbackExecutions.Load()
	errors := callbackErrors.Load()

	t.Logf("✅ Stress test completed:")
	t.Logf("   Duration: %v", duration)
	t.Logf("   Workers: %d", workers)
	t.Logf("   Callback executions: %d", executions)
	t.Logf("   Callback errors: %d", errors)

	assert.Greater(t, executions, int64(0), "Should have executed callbacks")
	assert.Equal(t, int64(0), errors, "Should have no callback registration errors")
}

// TestStress_FallbackConcurrentOperations stress tests the fallback store with
// concurrent reads and writes under changing health conditions.
func TestStress_FallbackConcurrentOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	duration := getStressDuration()
	workers := getStressWorkers()

	logger := zap.NewNop()
	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()

	// Create stores
	localStore, _ := createTestChromemStore(t, "local")
	remoteStore, _ := createTestChromemStore(t, "remote")

	checker := NewMockHealthChecker()
	checker.SetHealthy(true)

	health := NewHealthMonitor(ctx, checker, 100*time.Millisecond, logger)

	scrubber := &secrets.NoopScrubber{}
	wal, err := NewWAL(t.TempDir(), scrubber, logger)
	require.NoError(t, err)
	defer wal.Close()

	config := FallbackConfig{
		Enabled:             true,
		LocalPath:           t.TempDir(),
		WALPath:             t.TempDir(),
		SyncOnConnect:       true,
		HealthCheckInterval: "100ms",
		WALRetentionDays:    7,
	}

	fs, err := NewFallbackStore(ctx, remoteStore, localStore, health, wal, config, logger)
	require.NoError(t, err)
	defer fs.Close()

	tenant := &TenantInfo{TenantID: "stress-test"}
	tenantCtx := ContextWithTenant(ctx, tenant)

	var writeOps atomic.Int64
	var readOps atomic.Int64
	var deleteOps atomic.Int64
	var errors atomic.Int64

	// Start health flapping
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(50 * time.Millisecond)
		defer ticker.Stop()

		state := true
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				state = !state
				checker.SetHealthy(state)
			}
		}
	}()

	// Concurrent writers
	for i := 0; i < workers/3; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			docCounter := 0
			for {
				select {
				case <-ctx.Done():
					return
				default:
					docs := []Document{{
						ID:      fmt.Sprintf("worker-%d-doc-%d", workerID, docCounter),
						Content: fmt.Sprintf("stress test content %d", docCounter),
					}}
					_, err := fs.AddDocuments(tenantCtx, docs)
					if err != nil {
						errors.Add(1)
					} else {
						writeOps.Add(1)
					}
					docCounter++
					time.Sleep(time.Duration(rand.Intn(10)) * time.Millisecond)
				}
			}
		}(i)
	}

	// Concurrent readers
	for i := 0; i < workers/3; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
					_, err := fs.Search(tenantCtx, "stress test", 10)
					if err != nil {
						errors.Add(1)
					} else {
						readOps.Add(1)
					}
					time.Sleep(time.Duration(rand.Intn(5)) * time.Millisecond)
				}
			}
		}(i)
	}

	// Concurrent deleters (occasional)
	for i := 0; i < workers/3; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
					// Delete random documents
					id := fmt.Sprintf("worker-%d-doc-%d", rand.Intn(workers/3), rand.Intn(100))
					err := fs.DeleteDocuments(tenantCtx, []string{id})
					if err != nil {
						// Ignore not found errors
						if err.Error() != "not found" {
							errors.Add(1)
						}
					} else {
						deleteOps.Add(1)
					}
					time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)
				}
			}
		}(i)
	}

	// Wait for test duration
	<-ctx.Done()
	cancel()
	wg.Wait()

	writes := writeOps.Load()
	reads := readOps.Load()
	deletes := deleteOps.Load()
	errorCount := errors.Load()

	t.Logf("✅ Stress test completed:")
	t.Logf("   Duration: %v", duration)
	t.Logf("   Concurrent workers: %d", workers)
	t.Logf("   Write operations: %d", writes)
	t.Logf("   Read operations: %d", reads)
	t.Logf("   Delete operations: %d", deletes)
	t.Logf("   Total operations: %d", writes+reads+deletes)
	t.Logf("   Errors: %d (%.2f%%)", errorCount, float64(errorCount)/float64(writes+reads+deletes)*100)

	assert.Greater(t, writes, int64(0), "Should have performed write operations")
	assert.Greater(t, reads, int64(0), "Should have performed read operations")
}

// TestStress_CircuitBreakerUnderLoad stress tests the circuit breaker with
// rapid failure and recovery cycles.
func TestStress_CircuitBreakerUnderLoad(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	duration := getStressDuration()

	const resetAfter = 100 * time.Millisecond
	cb := NewCircuitBreaker(10, resetAfter)

	var successOps atomic.Int64
	var failureOps atomic.Int64
	var blockedOps atomic.Int64

	// The test is split into two deterministic phases so it exercises both the
	// "open and block" behavior AND the "recover through half-open to closed"
	// behavior. A single mixed phase can never produce a success because the
	// breaker opens after `threshold` failures and starves every caller, so we
	// drive the lifecycle explicitly.
	//
	// failPhase drives the underlying operation to fail; recoverPhase drives it
	// to succeed. We give each phase roughly half of the configured stress
	// duration so the test scales with CONTEXTD_STRESS_DURATION.
	half := duration / 2
	if half < 200*time.Millisecond {
		half = 200 * time.Millisecond
	}

	// --- Phase 1: failure load. The breaker should open and block callers. ---
	failCtx, failCancel := context.WithTimeout(context.Background(), half)
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-failCtx.Done():
					return
				default:
					if cb.Allow() {
						cb.RecordFailure()
						failureOps.Add(1)
					} else {
						blockedOps.Add(1)
					}
					time.Sleep(time.Duration(rand.Intn(5)) * time.Millisecond)
				}
			}
		}()
	}
	wg.Wait()
	failCancel()

	// After sustained failures the breaker must be open (or half-open, if a
	// probe slipped through right at the boundary) and must have blocked work.
	require.NotEqual(t, "closed", cb.State(), "breaker should not be closed after sustained failures")

	// --- Recovery window: let the reset timeout elapse with no new failures. ---
	// No RecordFailure happens here, so lastFailure stays put and the breaker
	// becomes eligible to transition to half-open on the next Allow().
	time.Sleep(2 * resetAfter)

	// --- Phase 2: recovery load. The underlying operation now succeeds, so the
	// breaker should probe via half-open and transition back to closed. ---
	recoverCtx, recoverCancel := context.WithTimeout(context.Background(), half)
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-recoverCtx.Done():
					return
				default:
					if cb.Allow() {
						cb.RecordSuccess()
						successOps.Add(1)
					} else {
						blockedOps.Add(1)
					}
					time.Sleep(time.Duration(rand.Intn(5)) * time.Millisecond)
				}
			}
		}()
	}
	wg.Wait()
	recoverCancel()

	successes := successOps.Load()
	failures := failureOps.Load()
	blocked := blockedOps.Load()

	t.Logf("✅ Circuit breaker stress test completed:")
	t.Logf("   Duration: %v", duration)
	t.Logf("   Success operations: %d", successes)
	t.Logf("   Failed operations: %d", failures)
	t.Logf("   Blocked operations: %d", blocked)
	t.Logf("   Final state: %s", cb.State())

	assert.Greater(t, successes, int64(0), "Breaker should recover and allow successful operations")
	assert.Greater(t, failures, int64(0), "Should have failed operations during the failure phase")
	assert.Greater(t, blocked, int64(0), "Circuit should have blocked some operations while open")
	assert.Equal(t, "closed", cb.State(), "Breaker should be closed after the recovery phase")
}

// TestStress_WALConcurrentWrites stress tests the WAL with concurrent writes
// from multiple goroutines.
func TestStress_WALConcurrentWrites(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	duration := getStressDuration()
	workers := getStressWorkers()

	logger := zap.NewNop()
	scrubber := &secrets.NoopScrubber{}
	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()

	wal, err := NewWAL(t.TempDir(), scrubber, logger)
	require.NoError(t, err)
	defer wal.Close()

	var writeOps atomic.Int64
	var errors atomic.Int64

	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			counter := 0
			for {
				select {
				case <-ctx.Done():
					return
				default:
					entry := WALEntry{
						ID:        fmt.Sprintf("worker-%d-entry-%d", workerID, counter),
						Operation: "add",
						Docs: []Document{{
							ID:      fmt.Sprintf("doc-%d-%d", workerID, counter),
							Content: fmt.Sprintf("content %d", counter),
						}},
						Timestamp: time.Now(),
					}
					err := wal.WriteEntry(ctx, entry)
					if err != nil {
						errors.Add(1)
					} else {
						writeOps.Add(1)
					}
					counter++
					time.Sleep(time.Duration(rand.Intn(5)) * time.Millisecond)
				}
			}
		}(i)
	}

	<-ctx.Done()
	cancel()
	wg.Wait()

	writes := writeOps.Load()
	errorCount := errors.Load()
	pending := len(wal.PendingEntries())

	t.Logf("✅ WAL stress test completed:")
	t.Logf("   Duration: %v", duration)
	t.Logf("   Workers: %d", workers)
	t.Logf("   Write operations: %d", writes)
	t.Logf("   Errors: %d", errorCount)
	t.Logf("   Pending entries: %d", pending)

	assert.Greater(t, writes, int64(0), "Should have performed write operations")
	assert.Equal(t, int64(pending), writes-errorCount, "Pending count should match successful writes")
}

// Helper functions

func getStressDuration() time.Duration {
	if durationStr := os.Getenv("STRESS_TEST_DURATION"); durationStr != "" {
		if d, err := time.ParseDuration(durationStr); err == nil {
			return d
		}
	}
	return 30 * time.Second // Default
}

func getStressWorkers() int {
	if workersStr := os.Getenv("STRESS_TEST_WORKERS"); workersStr != "" {
		var workers int
		if _, err := fmt.Sscanf(workersStr, "%d", &workers); err == nil {
			return workers
		}
	}
	return 100 // Default
}
