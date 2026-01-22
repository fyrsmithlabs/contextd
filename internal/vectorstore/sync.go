// Package vectorstore provides vector storage implementations.
package vectorstore

import (
	"context"
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

const (
	circuitClosed   uint32 = 0
	circuitOpen     uint32 = 1
	circuitHalfOpen uint32 = 2
)

// CircuitBreaker protects against repeated sync failures.
type CircuitBreaker struct {
	failures    atomic.Int32
	threshold   int32         // Default: 5
	resetAfter  time.Duration // Default: 5m
	state       atomic.Uint32 // 0=closed, 1=open, 2=half-open
	lastFailure atomic.Int64  // Unix nano timestamp
}

// NewCircuitBreaker creates a new circuit breaker.
func NewCircuitBreaker(threshold int32, resetAfter time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		threshold:  threshold,
		resetAfter: resetAfter,
	}
}

// Allow returns true if the operation is allowed.
func (cb *CircuitBreaker) Allow() bool {
	for {
		state := cb.state.Load()
		switch state {
		case circuitOpen:
			lastFail := time.Unix(0, cb.lastFailure.Load())
			if time.Since(lastFail) > cb.resetAfter {
				// CAS: only one goroutine transitions to half-open
				if cb.state.CompareAndSwap(circuitOpen, circuitHalfOpen) {
					return true // This goroutine gets the test request
				}
				continue // Another goroutine won, retry
			}
			return false
		case circuitHalfOpen:
			return false // Only one request allowed in half-open
		default: // circuitClosed
			return true
		}
	}
}

// RecordSuccess records a successful operation.
func (cb *CircuitBreaker) RecordSuccess() {
	cb.failures.Store(0)
	cb.state.Store(circuitClosed)
}

// RecordFailure records a failed operation.
func (cb *CircuitBreaker) RecordFailure() {
	// Atomic increment + CAS loop to prevent TOCTOU race
	for {
		currentFailures := cb.failures.Load()

		// Prevent overflow
		if currentFailures == math.MaxInt32 {
			return
		}

		newFailures := currentFailures + 1

		// Try to increment atomically
		if !cb.failures.CompareAndSwap(currentFailures, newFailures) {
			continue // Another goroutine incremented, retry
		}

		// Check threshold with the value we successfully stored
		if newFailures >= cb.threshold {
			// CAS to open state (only one goroutine wins)
			if cb.state.CompareAndSwap(circuitClosed, circuitOpen) ||
				cb.state.CompareAndSwap(circuitHalfOpen, circuitOpen) {
				cb.lastFailure.Store(time.Now().UnixNano())
			}
		}
		return
	}
}

// State returns the current circuit state.
func (cb *CircuitBreaker) State() string {
	state := cb.state.Load()
	switch state {
	case circuitClosed:
		return "closed"
	case circuitOpen:
		return "open"
	case circuitHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// SyncManager manages background synchronization from local to remote store.
type SyncManager struct {
	wal      *WAL
	local    Store
	remote   Store
	health   *HealthMonitor
	syncCh   chan struct{}     // Bounded channel for backpressure
	ctx      context.Context   // For graceful shutdown
	cancel   context.CancelFunc
	wg       sync.WaitGroup    // Wait for goroutines on shutdown
	cb       *CircuitBreaker
	logger   *zap.Logger
}

// NewSyncManager creates a SyncManager with bounded channels and shutdown support.
func NewSyncManager(ctx context.Context, wal *WAL, local, remote Store, health *HealthMonitor, logger *zap.Logger) *SyncManager {
	ctx, cancel := context.WithCancel(ctx)
	return &SyncManager{
		wal:    wal,
		local:  local,
		remote: remote,
		health: health,
		syncCh: make(chan struct{}, 100), // Bounded: backpressure after 100 pending
		ctx:    ctx,
		cancel: cancel,
		cb:     NewCircuitBreaker(5, 5*time.Minute),
		logger: logger,
	}
}

// Start begins background sync goroutine.
func (s *SyncManager) Start() {
	// Register callback for health changes
	s.health.RegisterCallback(func(healthy bool) {
		if healthy {
			s.logger.Info("sync: remote became healthy, triggering sync")
			s.TriggerSync()
		}
	})

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.runSyncLoop()
	}()
}

// runSyncLoop is the main sync loop.
func (s *SyncManager) runSyncLoop() {
	for {
		select {
		case <-s.ctx.Done():
			s.logger.Info("sync: shutdown requested")
			return
		case <-s.syncCh:
			s.performSync()
		}
	}
}

// TriggerSync requests a sync operation (non-blocking).
func (s *SyncManager) TriggerSync() {
	select {
	case s.syncCh <- struct{}{}:
		// Successfully queued
	default:
		// Channel full, backpressure - log and skip
		s.logger.Warn("sync: sync queue full, skipping trigger")
	}
}

// performSync performs the actual sync operation.
func (s *SyncManager) performSync() {
	// Check health first
	if !s.health.IsHealthy() {
		s.logger.Debug("sync: skipping sync, remote unhealthy")
		return
	}

	// Check circuit breaker
	if !s.cb.Allow() {
		s.logger.Debug("sync: circuit breaker open, skipping sync")
		return
	}

	// Get pending entries
	pending := s.wal.PendingEntries()
	if len(pending) == 0 {
		s.logger.Debug("sync: no pending entries")
		return
	}

	s.logger.Info("sync: starting sync",
		zap.Int("pending_entries", len(pending)))

	startTime := time.Now()
	synced := 0
	failed := 0

	// Process each entry with FIFO order
	for _, entry := range pending {
		if err := s.syncEntry(entry); err != nil {
			s.logger.Warn("sync: entry sync failed",
				zap.String("entry_id", entry.ID),
				zap.Error(err))
			failed++
			s.cb.RecordFailure()

			// Record the failure
			if err := s.wal.RecordSyncAttempt(entry.ID, err); err != nil {
				s.logger.Error("sync: failed to record sync attempt",
					zap.String("entry_id", entry.ID),
					zap.Error(err))
			}
		} else {
			synced++
			s.cb.RecordSuccess()

			// Mark as synced in WAL
			if err := s.wal.MarkSynced(entry.ID); err != nil {
				s.logger.Error("sync: failed to mark entry as synced",
					zap.String("entry_id", entry.ID),
					zap.Error(err))
			}
		}
	}

	duration := time.Since(startTime)
	s.logger.Info("sync: completed",
		zap.Int("synced", synced),
		zap.Int("failed", failed),
		zap.Duration("duration", duration))
}

// syncEntry syncs a single WAL entry to the remote store.
func (s *SyncManager) syncEntry(entry WALEntry) error {
	ctx := s.ctx

	switch entry.Operation {
	case "add":
		// Upsert to remote (local wins)
		_, err := s.remote.AddDocuments(ctx, entry.Docs)
		return err

	case "delete":
		// Delete from remote
		return s.remote.DeleteDocuments(ctx, entry.IDs)

	default:
		return fmt.Errorf("unknown operation: %s", entry.Operation)
	}
}

// Stop gracefully shuts down the sync manager.
func (s *SyncManager) Stop() error {
	s.logger.Info("sync: stopping")
	s.cancel()
	s.wg.Wait() // Wait for goroutine to finish
	s.logger.Info("sync: stopped")
	return nil
}
