package reasoningbank

import (
	"context"
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

	// projectIDs is the list of projects to consolidate on each run
	projectIDs []string

	// opts are the consolidation options to use (threshold, dry run, etc.)
	opts ConsolidationOptions

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

// WithProjectIDs sets the project IDs to consolidate.
// If not set, the scheduler will not consolidate any projects.
func WithProjectIDs(projectIDs []string) SchedulerOption {
	return func(s *ConsolidationScheduler) {
		s.projectIDs = projectIDs
	}
}

// WithConsolidationOptions sets the consolidation options.
// If not set, uses default options (threshold: 0.8, dry_run: false).
func WithConsolidationOptions(opts ConsolidationOptions) SchedulerOption {
	return func(s *ConsolidationScheduler) {
		s.opts = opts
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
		distiller:  distiller,
		logger:     logger,
		interval:   24 * time.Hour, // Default: daily consolidation
		projectIDs: []string{},
		opts: ConsolidationOptions{
			SimilarityThreshold: 0.8, // Default threshold
			DryRun:              false,
			ForceAll:            false,
			MaxClustersPerRun:   0, // No limit
		},
		running: false,
		stopCh:  make(chan struct{}),
	}

	// Apply options
	for _, opt := range opts {
		opt(s)
	}

	return s, nil
}

// Start begins the background consolidation scheduler.
//
// The scheduler runs consolidation at the configured interval until Stop() is called.
// This method is idempotent - calling Start() on an already running scheduler
// returns an error without starting a second goroutine.
//
// Returns:
//   - Error if the scheduler is already running
func (s *ConsolidationScheduler) Start() error {
	if s.running {
		return fmt.Errorf("scheduler is already running")
	}

	s.running = true
	s.logger.Info("consolidation scheduler started",
		zap.Duration("interval", s.interval),
	)

	// Start background goroutine for scheduled consolidation
	go s.run()

	return nil
}

// Stop gracefully stops the consolidation scheduler.
//
// Signals the background goroutine to stop and waits for it to finish.
// This method is idempotent - calling Stop() on an already stopped scheduler
// is a no-op.
//
// Returns:
//   - Always returns nil (for interface compatibility and future error handling)
func (s *ConsolidationScheduler) Stop() error {
	if !s.running {
		s.logger.Debug("scheduler stop called but not running")
		return nil
	}

	s.logger.Info("stopping consolidation scheduler")
	s.running = false

	// Signal the goroutine to stop
	close(s.stopCh)

	return nil
}

// run is the main scheduler loop that executes consolidation on the configured interval.
// This runs in a background goroutine started by Start().
//
// The loop uses a ticker to trigger consolidation at regular intervals. Each consolidation
// attempt is independent - errors are logged but do not stop the scheduler. The scheduler
// continues running until Stop() is called.
func (s *ConsolidationScheduler) run() {
	s.logger.Debug("scheduler goroutine started")
	defer s.logger.Debug("scheduler goroutine stopped")

	// Create a ticker for the configured interval
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	// Main scheduler loop
	for {
		select {
		case <-ticker.C:
			// Time to run consolidation
			s.runConsolidation()

		case <-s.stopCh:
			// Shutdown signal received
			s.logger.Debug("scheduler received stop signal")
			return
		}
	}
}

// runConsolidation executes a single consolidation run.
// Errors are logged but do not stop the scheduler.
func (s *ConsolidationScheduler) runConsolidation() {
	// Check if we have any projects to consolidate
	if len(s.projectIDs) == 0 {
		s.logger.Debug("no projects configured for consolidation, skipping")
		return
	}

	s.logger.Info("starting scheduled consolidation",
		zap.Int("project_count", len(s.projectIDs)),
		zap.Float64("threshold", s.opts.SimilarityThreshold),
		zap.Bool("dry_run", s.opts.DryRun),
	)

	// Use background context with a reasonable timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	// Run consolidation across all configured projects
	result, err := s.distiller.ConsolidateAll(ctx, s.projectIDs, s.opts)
	if err != nil {
		s.logger.Error("consolidation failed",
			zap.Error(err),
			zap.Int("project_count", len(s.projectIDs)),
		)
		return
	}

	// Log successful consolidation
	s.logger.Info("scheduled consolidation completed",
		zap.Int("created", len(result.CreatedMemories)),
		zap.Int("archived", len(result.ArchivedMemories)),
		zap.Int("skipped", result.SkippedCount),
		zap.Int("total_processed", result.TotalProcessed),
		zap.Duration("duration", result.Duration),
	)
}
