package prefetch

import (
	"context"
	"errors"
	"sync"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
)

// Executor executes pre-fetch rules in parallel with timeout protection.
type Executor struct {
	maxParallel int
	metrics     *Metrics
	logger      *zap.Logger
}

// NewExecutor creates a new rule executor.
//
// Parameters:
//   - maxParallel: Maximum number of rules to execute in parallel
func NewExecutor(maxParallel int) *Executor {
	return &Executor{
		maxParallel: maxParallel,
	}
}

// SetMetrics sets the metrics tracker for this executor.
// This is optional and should be called after executor creation if metrics are desired.
func (e *Executor) SetMetrics(m *Metrics) {
	e.metrics = m
}

// SetLogger sets the logger for this executor.
// This is optional and should be called after executor creation if logging is desired.
func (e *Executor) SetLogger(l *zap.Logger) {
	e.logger = l
}

// Execute runs all rules for an event in parallel.
//
// Rules are executed concurrently up to maxParallel. Failed rules are logged
// but don't block successful rules. Timeout handling is delegated to individual
// rules via the context.
//
// Returns:
//   - results: Successfully executed rule results
func (e *Executor) Execute(ctx context.Context, event GitEvent, rules []Rule) []PreFetchResult {
	if len(rules) == 0 {
		return nil
	}

	// Create channel for results
	resultsChan := make(chan *PreFetchResult, len(rules))

	// Use semaphore to limit parallelism
	sem := make(chan struct{}, e.maxParallel)

	// WaitGroup to wait for all rules to complete
	var wg sync.WaitGroup

	// Execute rules in parallel
	for _, rule := range rules {
		wg.Add(1)
		go func(r Rule) {
			defer wg.Done()

			// Acquire semaphore
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				return
			}

			// Execute rule with tracing and timing
			ruleCtx, ruleSpan := tracer.Start(ctx, "prefetch.execute_rule")
			ruleSpan.SetAttributes(
				attribute.String("rule.name", r.Name()),
				attribute.String("event.type", eventTypeString(event.Type)),
			)

			start := time.Now()
			result, err := r.Execute(ruleCtx, event)
			duration := time.Since(start)

			if err != nil {
				// Check if it's a timeout
				if errors.Is(err, context.DeadlineExceeded) {
					if e.logger != nil {
						e.logger.Warn("Rule execution timeout",
							zap.String("rule", r.Name()),
							zap.Duration("duration", duration))
					}
					if e.metrics != nil {
						e.metrics.RecordRuleTimeout(r.Name())
					}
				} else {
					if e.logger != nil {
						e.logger.Error("Rule execution failed",
							zap.String("rule", r.Name()),
							zap.Error(err))
					}
				}
				ruleSpan.End()
				return
			}

			// Record successful execution
			if e.logger != nil {
				e.logger.Debug("Rule executed",
					zap.String("rule", r.Name()),
					zap.Duration("duration", duration),
					zap.Int("results", 1))
			}
			if e.metrics != nil {
				e.metrics.RecordRuleExecution(r.Name(), duration.Seconds())
			}

			ruleSpan.SetAttributes(attribute.Int("results.count", 1))
			ruleSpan.End()

			// Send result
			select {
			case resultsChan <- result:
			case <-ctx.Done():
				return
			}
		}(rule)
	}

	// Wait for all rules to complete (in a goroutine)
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	// Collect results
	var results []PreFetchResult
	for result := range resultsChan {
		if result != nil {
			results = append(results, *result)
		}
	}

	return results
}

// eventTypeString converts EventType to string for logging/tracing.
func eventTypeString(t EventType) string {
	switch t {
	case EventTypeBranchSwitch:
		return "branch_switch"
	case EventTypeNewCommit:
		return "new_commit"
	default:
		return "unknown"
	}
}
