package vectorstore

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestHealthMonitor_IsHealthy(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()

	checker := NewMockHealthChecker()
	checker.SetHealthy(true)

	hm := NewHealthMonitor(ctx, checker, 30*time.Second, logger)
	defer hm.Stop()

	assert.True(t, hm.IsHealthy())
}

func TestHealthMonitor_RegisterCallback(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()

	checker := NewMockHealthChecker()
	checker.SetHealthy(true)

	hm := NewHealthMonitor(ctx, checker, 30*time.Second, logger)
	defer hm.Stop()

	called := false
	hm.RegisterCallback(func(healthy bool) {
		called = true
	})

	// Manually trigger callback
	hm.updateHealth(false)

	// Give callback goroutine time to execute
	time.Sleep(10 * time.Millisecond)

	assert.True(t, called)
}

func TestHealthMonitor_LastCheck(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()

	checker := NewMockHealthChecker()
	checker.SetHealthy(true)

	hm := NewHealthMonitor(ctx, checker, 30*time.Second, logger)
	defer hm.Stop()

	lastCheck := hm.LastCheck()
	assert.False(t, lastCheck.IsZero())
}

func TestHealthMonitor_Stop(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()

	checker := NewMockHealthChecker()
	checker.SetHealthy(true)

	hm := NewHealthMonitor(ctx, checker, 30*time.Second, logger)

	// Should not panic
	hm.Stop()
}

func TestMockHealthChecker_IsHealthy(t *testing.T) {
	checker := NewMockHealthChecker()
	assert.False(t, checker.IsHealthy(context.Background()))

	checker.SetHealthy(true)
	assert.True(t, checker.IsHealthy(context.Background()))

	checker.SetHealthy(false)
	assert.False(t, checker.IsHealthy(context.Background()))
}

func TestMockHealthChecker_WatchState(t *testing.T) {
	checker := NewMockHealthChecker()

	// Should not error (noop implementation)
	err := checker.WatchState(context.Background(), func(healthy bool) {})
	assert.NoError(t, err)
}
