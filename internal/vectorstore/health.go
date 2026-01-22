// Package vectorstore provides vector storage implementations.
package vectorstore

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
)

// HealthChecker interface for dependency injection and testability.
type HealthChecker interface {
	// IsHealthy returns true if the remote store is healthy.
	IsHealthy(ctx context.Context) bool

	// WatchState watches for connectivity state changes.
	// The callback is invoked whenever health status changes.
	WatchState(ctx context.Context, callback func(healthy bool)) error
}

// GRPCHealthChecker implements HealthChecker for Qdrant gRPC connections.
type GRPCHealthChecker struct {
	conn   *grpc.ClientConn
	logger *zap.Logger
}

// NewGRPCHealthChecker creates a new gRPC health checker.
func NewGRPCHealthChecker(conn *grpc.ClientConn, logger *zap.Logger) *GRPCHealthChecker {
	return &GRPCHealthChecker{
		conn:   conn,
		logger: logger,
	}
}

// IsHealthy returns true if the gRPC connection is in Ready state.
func (g *GRPCHealthChecker) IsHealthy(ctx context.Context) bool {
	if g.conn == nil {
		return false
	}
	state := g.conn.GetState()
	return state == connectivity.Ready
}

// WatchState watches for gRPC connectivity state changes.
func (g *GRPCHealthChecker) WatchState(ctx context.Context, callback func(healthy bool)) error {
	if g.conn == nil {
		return nil
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			currentState := g.conn.GetState()
			g.logger.Debug("gRPC state change",
				zap.String("state", currentState.String()))

			// Notify on state change
			healthy := currentState == connectivity.Ready
			callback(healthy)

			// Wait for next state change
			if !g.conn.WaitForStateChange(ctx, currentState) {
				return
			}
		}
	}()

	return nil
}

// MockHealthChecker for testing.
type MockHealthChecker struct {
	healthy atomic.Bool
}

// NewMockHealthChecker creates a new mock health checker.
func NewMockHealthChecker() *MockHealthChecker {
	return &MockHealthChecker{}
}

// IsHealthy returns the mock health status.
func (m *MockHealthChecker) IsHealthy(ctx context.Context) bool {
	return m.healthy.Load()
}

// SetHealthy sets the mock health status and does not trigger callbacks.
func (m *MockHealthChecker) SetHealthy(healthy bool) {
	m.healthy.Store(healthy)
}

// WatchState does nothing for mock (no state changes to watch).
func (m *MockHealthChecker) WatchState(ctx context.Context, callback func(healthy bool)) error {
	return nil
}

// HealthMonitor monitors remote store connectivity.
type HealthMonitor struct {
	checker       HealthChecker     // Interface for DI (gRPC, HTTP, mock)
	healthy       atomic.Bool       // Current health status
	lastCheck     atomic.Value      // time.Time
	checkInterval time.Duration     // Configurable via FallbackConfig
	mu            sync.RWMutex      // Protects callbacks slice
	callbacks     []func(bool)      // Callbacks to notify on health change
	ctx           context.Context   // For graceful shutdown
	cancel        context.CancelFunc
	logger        *zap.Logger
}

// NewHealthMonitor creates a new health monitor.
func NewHealthMonitor(ctx context.Context, checker HealthChecker, checkInterval time.Duration, logger *zap.Logger) *HealthMonitor {
	ctx, cancel := context.WithCancel(ctx)
	hm := &HealthMonitor{
		checker:       checker,
		checkInterval: checkInterval,
		callbacks:     make([]func(bool), 0),
		ctx:           ctx,
		cancel:        cancel,
		logger:        logger,
	}

	// Initialize with current health status
	hm.healthy.Store(checker.IsHealthy(ctx))
	hm.lastCheck.Store(time.Now())

	return hm
}

// Start begins health monitoring.
func (hm *HealthMonitor) Start() {
	// Watch for state changes (primary detection)
	hm.checker.WatchState(hm.ctx, func(healthy bool) {
		hm.updateHealth(healthy)
	})

	// Periodic ping (fallback detection)
	go hm.runPeriodicCheck()
}

// runPeriodicCheck performs periodic health checks.
func (hm *HealthMonitor) runPeriodicCheck() {
	ticker := time.NewTicker(hm.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-hm.ctx.Done():
			return
		case <-ticker.C:
			healthy := hm.checker.IsHealthy(hm.ctx)
			hm.updateHealth(healthy)
		}
	}
}

// updateHealth updates health status and notifies callbacks if changed.
func (hm *HealthMonitor) updateHealth(healthy bool) {
	oldHealth := hm.healthy.Load()
	hm.healthy.Store(healthy)
	hm.lastCheck.Store(time.Now())

	// Only notify if health status changed
	if oldHealth != healthy {
		hm.logger.Info("health status changed",
			zap.Bool("healthy", healthy),
			zap.Bool("previous", oldHealth))
		hm.notifyCallbacks(healthy)
	}
}

// IsHealthy returns the current health status.
func (hm *HealthMonitor) IsHealthy() bool {
	return hm.healthy.Load()
}

// LastCheck returns the time of the last health check.
func (hm *HealthMonitor) LastCheck() time.Time {
	v := hm.lastCheck.Load()
	if v == nil {
		return time.Time{}
	}
	return v.(time.Time)
}

// RegisterCallback adds a callback with mutex protection.
// Returns an error if the callback is nil.
func (hm *HealthMonitor) RegisterCallback(cb func(bool)) error {
	if cb == nil {
		return fmt.Errorf("health: callback cannot be nil")
	}

	hm.mu.Lock()
	defer hm.mu.Unlock()
	hm.callbacks = append(hm.callbacks, cb)
	return nil
}

// notifyCallbacks fires all callbacks under read lock (allows concurrent reads).
// Copy-before-fire pattern prevents holding lock during callbacks.
func (hm *HealthMonitor) notifyCallbacks(healthy bool) {
	hm.mu.RLock()
	callbacks := make([]func(bool), len(hm.callbacks))
	copy(callbacks, hm.callbacks)
	hm.mu.RUnlock()

	for _, cb := range callbacks {
		// Call in separate goroutine to prevent blocking
		go func(callback func(bool)) {
			defer func() {
				if r := recover(); r != nil {
					hm.logger.Error("health callback panic",
						zap.Any("panic", r))
				}
			}()

			// Create timeout context for callback (5 seconds)
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			// Run callback with timeout protection
			done := make(chan struct{})
			go func() {
				callback(healthy)
				close(done)
			}()

			select {
			case <-done:
				// Callback completed successfully
			case <-ctx.Done():
				hm.logger.Warn("health callback timeout",
					zap.Duration("timeout", 5*time.Second))
			}
		}(cb)
	}
}

// Stop gracefully shuts down the health monitor.
func (hm *HealthMonitor) Stop() {
	hm.cancel()
}
