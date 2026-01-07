package reasoningbank

import (
	"fmt"
	"time"

	"go.uber.org/zap"
)

// ConsolidationScheduler manages automatic scheduled memory consolidation.
//
// The scheduler runs consolidation periodically in the background for configured
// projects. It provides lifecycle management (Start/Stop) with graceful shutdown
// and ensures consolidation runs on a predictable schedule.
type ConsolidationScheduler struct {
	// interval is the time between consolidation runs (e.g., 24 hours for daily consolidation)
	interval time.Duration

	// distiller performs the actual memory consolidation
	distiller *Distiller

	// running tracks whether the scheduler is currently running
	running bool

	// stopCh is used to signal the scheduler to stop
	stopCh chan struct{}

	// logger for structured logging
	logger *zap.Logger
}

// SchedulerOption configures a ConsolidationScheduler.
type SchedulerOption func(*ConsolidationScheduler)

// WithInterval sets the consolidation interval.
// If not set, defaults to 24 hours.
func WithInterval(interval time.Duration) SchedulerOption {
	return func(s *ConsolidationScheduler) {
		s.interval = interval
	}
}

// NewConsolidationScheduler creates a new consolidation scheduler.
//
// The scheduler does not start automatically - call Start() to begin
// scheduled consolidation runs.
//
// Parameters:
//   - distiller: The distiller to use for consolidation
//   - logger: Logger for structured logging
//   - opts: Optional configuration options
//
// Returns:
//   - A new scheduler instance
//   - Error if distiller or logger is nil
func NewConsolidationScheduler(distiller *Distiller, logger *zap.Logger, opts ...SchedulerOption) (*ConsolidationScheduler, error) {
	if distiller == nil {
		return nil, fmt.Errorf("distiller cannot be nil")
	}
	if logger == nil {
		return nil, fmt.Errorf("logger cannot be nil")
	}

	s := &ConsolidationScheduler{
		distiller: distiller,
		logger:    logger,
		interval:  24 * time.Hour, // Default: daily consolidation
		running:   false,
		stopCh:    make(chan struct{}),
	}

	// Apply options
	for _, opt := range opts {
		opt(s)
	}

	return s, nil
}
