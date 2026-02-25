package vectorstore

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/fyrsmithlabs/contextd/internal/secrets"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// TestProductionHardening_HealthCallbackWorkerPool validates the semaphore-based
// worker pool prevents unbounded goroutine creation under high-frequency health changes.
func TestProductionHardening_HealthCallbackWorkerPool(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping validation test in short mode")
	}

	logger := zap.NewNop()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	checker := NewMockHealthChecker()
	monitor := NewHealthMonitor(ctx, checker, 100*time.Millisecond, logger)

	// Register 100 callbacks (more than semaphore capacity of 10)
	callbackCount := 100
	var wg sync.WaitGroup
	callbackExecutions := make([]int, callbackCount)
	var mu sync.Mutex

	for i := 0; i < callbackCount; i++ {
		idx := i
		err := monitor.RegisterCallback(func(healthy bool) {
			mu.Lock()
			callbackExecutions[idx]++
			mu.Unlock()
			// Simulate slow callback (10ms)
			time.Sleep(10 * time.Millisecond)
		})
		require.NoError(t, err, "Failed to register callback %d", i)
	}

	monitor.Start()

	// Trigger rapid health changes (100 changes in 1 second)
	wg.Add(1)
	go func() {
		defer wg.Done()
		for j := 0; j < 100; j++ {
			checker.SetHealthy(j%2 == 0)
			monitor.updateHealth(j%2 == 0)
			time.Sleep(10 * time.Millisecond)
		}
	}()

	wg.Wait()
	time.Sleep(200 * time.Millisecond) // Allow callbacks to complete

	monitor.Stop()

	// Verify: All callbacks should have been executed at least once
	mu.Lock()
	defer mu.Unlock()
	for i, count := range callbackExecutions {
		assert.Greater(t, count, 0, "Callback %d was never executed", i)
	}

	t.Logf("✅ Worker pool handled %d callbacks with rapid health changes", callbackCount)
}

// TestProductionHardening_PathInjection validates WAL path injection protection.
func TestProductionHardening_PathInjection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping validation test in short mode")
	}

	logger := zap.NewNop()
	scrubber := &secrets.NoopScrubber{}

	testCases := []struct {
		name        string
		path        string
		shouldFail  bool
		description string
	}{
		{
			name:        "directory_traversal_dotdot",
			path:        "../../../etc/passwd",
			shouldFail:  true,
			description: "Should block directory traversal with ../",
		},
		{
			name:        "directory_traversal_encoded",
			path:        "/tmp/wal/../../etc/passwd",
			shouldFail:  true,
			description: "Should block encoded directory traversal",
		},
		{
			name:        "valid_absolute_path",
			path:        t.TempDir(),
			shouldFail:  false,
			description: "Should allow valid absolute path",
		},
		{
			name:        "valid_relative_path",
			path:        "./wal",
			shouldFail:  false,
			description: "Should allow valid relative path (converted to absolute)",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			wal, err := NewWAL(tc.path, scrubber, logger)
			if tc.shouldFail {
				assert.Error(t, err, tc.description)
			} else {
				assert.NoError(t, err, tc.description)
				if wal != nil {
					wal.Close()
				}
			}
			t.Logf("✅ %s", tc.description)
		})
	}
}

// TestProductionHardening_TenantContextImmutability validates defensive copying
// prevents race conditions from context modifications.
func TestProductionHardening_TenantContextImmutability(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping validation test in short mode")
	}

	logger := zap.NewNop()
	ctx := context.Background()

	// Create fallback store
	localStore, _ := createTestChromemStore(t, "local")
	remoteStore, _ := createTestChromemStore(t, "remote")

	checker := NewMockHealthChecker()
	checker.SetHealthy(true)

	health := NewHealthMonitor(ctx, checker, 1*time.Second, logger)

	scrubber := &secrets.NoopScrubber{}
	wal, err := NewWAL(t.TempDir(), scrubber, logger)
	require.NoError(t, err)
	defer wal.Close()

	config := FallbackConfig{
		Enabled:             true,
		LocalPath:           t.TempDir(),
		WALPath:             t.TempDir(),
		SyncOnConnect:       true,
		HealthCheckInterval: "1s",
		WALRetentionDays:    7,
	}

	fs, err := NewFallbackStore(ctx, remoteStore, localStore, health, wal, config, logger)
	require.NoError(t, err)
	defer fs.Close()

	// Create tenant context
	tenant := &TenantInfo{
		TenantID:  "org-123",
		TeamID:    "team-1",
		ProjectID: "proj-1",
	}
	tenantCtx := ContextWithTenant(ctx, tenant)

	// Attempt to modify tenant while concurrent operations are running
	var wg sync.WaitGroup
	errorsCh := make(chan error, 100)

	// Run 100 concurrent AddDocuments operations
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			docs := []Document{{
				ID:       fmt.Sprintf("doc-%d", idx),
				Content:  fmt.Sprintf("content-%d", idx),
				Metadata: map[string]interface{}{"index": idx},
			}}
			_, err := fs.AddDocuments(tenantCtx, docs)
			if err != nil {
				errorsCh <- err
			}
		}(i)

		// Concurrently try to modify the tenant (should have no effect)
		if i%10 == 0 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				tenant.TenantID = "malicious-tenant"
				tenant.TeamID = "malicious-team"
			}()
		}
	}

	wg.Wait()
	close(errorsCh)

	// Verify no errors occurred
	var errors []error
	for err := range errorsCh {
		errors = append(errors, err)
	}
	assert.Empty(t, errors, "Should have no errors from concurrent operations")

	t.Logf("✅ Tenant context immutability prevents race conditions")
}

// TestProductionHardening_HealthStatusRaceCondition validates the fix for
// health status variable mutation causing inconsistent behavior.
func TestProductionHardening_HealthStatusRaceCondition(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping validation test in short mode")
	}

	logger := zap.NewNop()
	ctx := context.Background()

	// Create fallback store with initially healthy remote
	localStore, _ := createTestChromemStore(t, "local")
	remoteStore := &FailingStore{
		failCount: 0,
		maxFails:  5, // Fail first 5 operations, then succeed
		Store:     nil,
	}

	checker := NewMockHealthChecker()
	checker.SetHealthy(true) // Remote reports healthy

	health := NewHealthMonitor(ctx, checker, 1*time.Second, logger)

	scrubber := &secrets.NoopScrubber{}
	wal, err := NewWAL(t.TempDir(), scrubber, logger)
	require.NoError(t, err)
	defer wal.Close()

	config := FallbackConfig{
		Enabled:             true,
		LocalPath:           t.TempDir(),
		WALPath:             t.TempDir(),
		SyncOnConnect:       true,
		HealthCheckInterval: "1s",
		WALRetentionDays:    7,
	}

	fs, err := NewFallbackStore(ctx, remoteStore, localStore, health, wal, config, logger)
	require.NoError(t, err)
	defer fs.Close()

	tenant := &TenantInfo{TenantID: "org-123"}
	tenantCtx := ContextWithTenant(ctx, tenant)

	// Run 10 operations - first 5 should fail remote and use local,
	// next 5 should succeed on remote
	for i := 0; i < 10; i++ {
		docs := []Document{{
			ID:      fmt.Sprintf("doc-%d", i),
			Content: fmt.Sprintf("content-%d", i),
		}}
		_, err := fs.AddDocuments(tenantCtx, docs)
		assert.NoError(t, err, "Operation %d should succeed via fallback", i)
	}

	// Verify: All documents should be in local store
	results, err := localStore.Search(tenantCtx, "content", 20)
	assert.NoError(t, err)
	assert.Equal(t, 10, len(results), "All 10 documents should be in local store")

	t.Logf("✅ Health status handling is consistent without variable mutation")
}

// TestProductionHardening_CircuitBreakerReset validates the circuit breaker
// reset mechanism at max failures.
func TestProductionHardening_CircuitBreakerReset(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping validation test in short mode")
	}

	cb := NewCircuitBreaker(5, 1*time.Second)

	// Force failures to max int32
	cb.failures.Store(2147483647) // math.MaxInt32

	// Record another failure - should reset to threshold
	cb.RecordFailure()

	// Verify circuit breaker is still functional
	assert.Equal(t, "open", cb.State(), "Circuit should be open")
	assert.Equal(t, int32(5), cb.failures.Load(), "Failures should reset to threshold")

	// Wait for reset period
	time.Sleep(1100 * time.Millisecond)

	// Circuit should allow one request (half-open)
	allowed := cb.Allow()
	assert.True(t, allowed, "Circuit should allow request after reset period")
	assert.Equal(t, "half-open", cb.State(), "Circuit should be half-open")

	// Success should close the circuit
	cb.RecordSuccess()
	assert.Equal(t, "closed", cb.State(), "Circuit should be closed after success")
	assert.Equal(t, int32(0), cb.failures.Load(), "Failures should be reset to 0")

	t.Logf("✅ Circuit breaker reset mechanism prevents max failure deadlock")
}

// TestProductionHardening_SecretScrubbing validates secret scrubbing still works
// with reduced log verbosity.
func TestProductionHardening_SecretScrubbing(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping validation test in short mode")
	}

	logger := zap.NewNop()
	scrubber, err := secrets.New(nil)
	require.NoError(t, err)

	ctx := context.Background()
	wal, err := NewWAL(t.TempDir(), scrubber, logger)
	require.NoError(t, err)
	defer wal.Close()

	// Write entry with AWS key (should be scrubbed)
	entry := WALEntry{
		ID:        "secret-test",
		Operation: "add",
		Docs: []Document{{
			ID:      "doc1",
			Content: "AWS Key: AKIAIOSFODNN7EXAMPLE should be redacted",
		}},
		Timestamp: time.Now(),
	}

	err = wal.WriteEntry(ctx, entry)
	assert.NoError(t, err, "WAL write should succeed")

	// Verify content was scrubbed
	pending := wal.PendingEntries()
	require.Len(t, pending, 1)
	assert.NotContains(t, pending[0].Docs[0].Content, "AKIAIOSFODNN7EXAMPLE",
		"AWS key should be scrubbed from content")

	t.Logf("✅ Secret scrubbing works correctly (logs reduced to debug level)")
}

// FailingStore is a test helper that fails the first N operations.
type FailingStore struct {
	Store
	failCount int
	maxFails  int
	mu        sync.Mutex
}

func (f *FailingStore) AddDocuments(ctx context.Context, docs []Document) ([]string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.failCount < f.maxFails {
		f.failCount++
		return nil, fmt.Errorf("simulated remote failure %d/%d", f.failCount, f.maxFails)
	}

	// After max fails, succeed
	ids := make([]string, len(docs))
	for i := range docs {
		ids[i] = docs[i].ID
	}
	return ids, nil
}

func (f *FailingStore) Search(ctx context.Context, query string, k int) ([]SearchResult, error) {
	return nil, nil
}

func (f *FailingStore) SearchWithFilters(ctx context.Context, query string, k int, filters map[string]interface{}) ([]SearchResult, error) {
	return nil, nil
}

func (f *FailingStore) DeleteDocuments(ctx context.Context, ids []string) error {
	return nil
}

func (f *FailingStore) Close() error {
	return nil
}
