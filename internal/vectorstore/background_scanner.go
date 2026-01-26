// Package vectorstore provides periodic background health scanning.
package vectorstore

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"
)

// BackgroundScannerConfig configures periodic health scanning.
type BackgroundScannerConfig struct {
	// Interval between health scans. Default: 5 minutes.
	Interval time.Duration

	// OnDegraded is called when degraded state is detected.
	// Receives the health result for alerting/notification.
	OnDegraded func(health *MetadataHealth)

	// OnRecovered is called when system recovers from degraded state.
	OnRecovered func(health *MetadataHealth)

	// OnError is called when health check fails.
	OnError func(err error)
}

// BackgroundScanner performs periodic health checks in the background.
type BackgroundScanner struct {
	checker    *MetadataHealthChecker
	config     *BackgroundScannerConfig
	logger     *zap.Logger

	mu         sync.RWMutex
	lastHealth *MetadataHealth
	lastError  error
	running    bool

	stopCh     chan struct{}
	doneCh     chan struct{}
}

// NewBackgroundScanner creates a new background health scanner.
func NewBackgroundScanner(checker *MetadataHealthChecker, config *BackgroundScannerConfig, logger *zap.Logger) *BackgroundScanner {
	if config == nil {
		config = &BackgroundScannerConfig{}
	}
	if config.Interval <= 0 {
		config.Interval = 5 * time.Minute
	}

	return &BackgroundScanner{
		checker: checker,
		config:  config,
		logger:  logger,
		stopCh:  make(chan struct{}),
		doneCh:  make(chan struct{}),
	}
}

// Start begins periodic health scanning in the background.
// Returns immediately; scanning happens in a goroutine.
func (s *BackgroundScanner) Start(ctx context.Context) {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true
	s.mu.Unlock()

	s.logger.Info("starting background health scanner",
		zap.Duration("interval", s.config.Interval))

	go s.run(ctx)
}

// Stop halts the background scanner and waits for it to finish.
func (s *BackgroundScanner) Stop() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	s.mu.Unlock()

	s.logger.Info("stopping background health scanner")
	close(s.stopCh)
	<-s.doneCh

	s.mu.Lock()
	s.running = false
	s.mu.Unlock()
}

// LastHealth returns the most recent health check result.
func (s *BackgroundScanner) LastHealth() *MetadataHealth {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastHealth
}

// LastError returns the most recent health check error (if any).
func (s *BackgroundScanner) LastError() error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastError
}

// IsRunning returns true if the scanner is actively running.
func (s *BackgroundScanner) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

func (s *BackgroundScanner) run(ctx context.Context) {
	defer close(s.doneCh)

	// Run initial scan immediately
	s.scan(ctx)

	ticker := time.NewTicker(s.config.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("background scanner stopped: context canceled")
			return
		case <-s.stopCh:
			s.logger.Info("background scanner stopped: stop requested")
			return
		case <-ticker.C:
			s.scan(ctx)
		}
	}
}

func (s *BackgroundScanner) scan(ctx context.Context) {
	s.logger.Debug("running background health scan")

	health, err := s.checker.Check(ctx)

	s.mu.Lock()
	previousHealth := s.lastHealth
	s.lastHealth = health
	s.lastError = err
	s.mu.Unlock()

	if err != nil {
		s.logger.Error("background health scan failed", zap.Error(err))
		if s.config.OnError != nil {
			s.config.OnError(err)
		}
		return
	}

	// Check for state transitions
	wasHealthy := previousHealth == nil || previousHealth.IsHealthy()
	isHealthy := health.IsHealthy()

	if wasHealthy && !isHealthy {
		// Transition: healthy -> degraded
		s.logger.Warn("vectorstore entered degraded state",
			zap.Int("corrupt_count", health.CorruptCount),
			zap.Strings("corrupt_hashes", health.Corrupt))

		if s.config.OnDegraded != nil {
			s.config.OnDegraded(health)
		}
	} else if !wasHealthy && isHealthy {
		// Transition: degraded -> healthy
		s.logger.Info("vectorstore recovered to healthy state",
			zap.Int("healthy_count", health.HealthyCount))

		if s.config.OnRecovered != nil {
			s.config.OnRecovered(health)
		}
	}

	s.logger.Debug("background health scan completed",
		zap.String("status", health.Status()),
		zap.Int("total", health.Total),
		zap.Int("healthy", health.HealthyCount),
		zap.Int("corrupt", health.CorruptCount))
}
