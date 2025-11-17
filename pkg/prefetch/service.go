package prefetch

import (
	"context"
	"fmt"
	"sync"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
)

var tracer = otel.Tracer("contextd/prefetch")

// Detector is a high-level service that coordinates git event detection,
// rule execution, and caching for a single project.
//
// It manages the lifecycle of GitEventDetector, RuleRegistry, Executor,
// and Cache components, providing a unified interface for pre-fetch operations.
type Detector struct {
	projectPath  string
	cache        *Cache
	executor     *Executor
	logger       *zap.Logger
	gitDetector  *GitEventDetector
	ruleRegistry *RuleRegistry
	stopChan     chan struct{}
	stopOnce     sync.Once
	metrics      *Metrics
}

// NewDetector creates a new Detector for the specified project path.
//
// Parameters:
//   - projectPath: Absolute path to the git repository
//   - cache: Shared cache instance for storing prefetch results
//   - executor: Rule executor for parallel rule execution
//   - logger: Structured logger
//
// Returns an error if the path is not a git repository or the watcher fails.
func NewDetector(projectPath string, cache *Cache, executor *Executor, logger *zap.Logger) (*Detector, error) {
	// Create git event detector
	gitDetector, err := NewGitEventDetector(projectPath)
	if err != nil {
		return nil, fmt.Errorf("creating git event detector: %w", err)
	}

	// Create rule registry with default rules
	ruleRegistry := NewRuleRegistry(projectPath)

	// Create metrics (reuse existing metrics if cache has them)
	metrics := NewMetrics()

	return &Detector{
		projectPath:  projectPath,
		cache:        cache,
		executor:     executor,
		logger:       logger,
		gitDetector:  gitDetector,
		ruleRegistry: ruleRegistry,
		stopChan:     make(chan struct{}),
		metrics:      metrics,
	}, nil
}

// Start begins watching for git events and processing pre-fetch rules.
//
// This method runs in the foreground and blocks until Stop() is called
// or the context is cancelled.
func (d *Detector) Start(ctx context.Context) {
	d.logger.Info("Prefetch detector started",
		zap.String("project", d.projectPath))

	// Start git event detector
	if err := d.gitDetector.Start(ctx); err != nil {
		d.logger.Error("Failed to start git event detector",
			zap.Error(err),
			zap.String("project", d.projectPath))
		return
	}

	// Process events
	eventChan := d.gitDetector.Events()

	for {
		select {
		case <-ctx.Done():
			d.logger.Info("Prefetch detector stopped (context cancelled)",
				zap.String("project", d.projectPath))
			return

		case <-d.stopChan:
			d.logger.Info("Prefetch detector stopped",
				zap.String("project", d.projectPath))
			return

		case event, ok := <-eventChan:
			if !ok {
				d.logger.Warn("Event channel closed",
					zap.String("project", d.projectPath))
				return
			}

			// Process event with tracing
			d.processEvent(ctx, event)
		}
	}
}

// Stop stops the detector and cleans up resources.
//
// This method is safe to call multiple times.
func (d *Detector) Stop() {
	d.stopOnce.Do(func() {
		close(d.stopChan)
		d.gitDetector.Stop()
	})
}

// Cache returns the cache instance used by this detector.
func (d *Detector) Cache() *Cache {
	return d.cache
}

// processEvent handles a single git event by executing rules and caching results.
func (d *Detector) processEvent(ctx context.Context, event GitEvent) {
	ctx, span := tracer.Start(ctx, "prefetch.detect_event")
	defer span.End()

	eventType := "branch_switch"
	if event.Type == EventTypeNewCommit {
		eventType = "new_commit"
	}

	span.SetAttributes(
		attribute.String("event.type", eventType),
		attribute.String("project.path", d.projectPath),
	)

	d.logger.Info("Git event detected",
		zap.String("type", eventType),
		zap.String("project", d.projectPath),
		zap.String("old_branch", event.OldBranch),
		zap.String("new_branch", event.NewBranch),
		zap.String("commit_hash", event.CommitHash))

	// Record git event metric
	d.metrics.RecordGitEvent(eventType)

	// Get rules for this event type
	rules := d.ruleRegistry.GetRulesForEvent(event.Type)
	if len(rules) == 0 {
		d.logger.Debug("No rules configured for event type",
			zap.String("event_type", eventType))
		return
	}

	// Execute rules
	results := d.executor.Execute(ctx, event, rules)

	d.logger.Debug("Rules executed",
		zap.String("project", d.projectPath),
		zap.Int("rule_count", len(rules)),
		zap.Int("result_count", len(results)))

	// Store results in cache
	if len(results) > 0 {
		d.cache.Set(d.projectPath, results)
		d.logger.Debug("Prefetch results cached",
			zap.String("project", d.projectPath),
			zap.Int("results", len(results)))
	}
}
