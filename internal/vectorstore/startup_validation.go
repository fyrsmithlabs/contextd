// Package vectorstore provides startup validation for metadata integrity.
package vectorstore

import (
	"context"
	"fmt"

	"go.uber.org/zap"
)

// StartupValidationConfig configures pre-flight health checks.
type StartupValidationConfig struct {
	// FailOnCorruption blocks startup if corrupt collections are detected.
	// Default: false (log warning but continue with graceful degradation)
	FailOnCorruption bool

	// FailOnDegraded blocks startup if any health issues are detected.
	// This is stricter than FailOnCorruption (includes empty collections, etc.)
	// Default: false
	FailOnDegraded bool
}

// StartupValidationResult contains the outcome of pre-flight checks.
type StartupValidationResult struct {
	Passed       bool
	Health       *MetadataHealth
	WarningCount int
	ErrorCount   int
	Messages     []string
}

// ValidateStartup performs pre-flight health checks before services start.
// Returns nil if validation passes, error if startup should be blocked.
func ValidateStartup(ctx context.Context, checker *MetadataHealthChecker, cfg *StartupValidationConfig, logger *zap.Logger) (*StartupValidationResult, error) {
	if checker == nil {
		logger.Debug("startup validation skipped: no health checker configured")
		return &StartupValidationResult{
			Passed:   true,
			Messages: []string{"validation skipped: no health checker"},
		}, nil
	}

	if cfg == nil {
		cfg = &StartupValidationConfig{}
	}

	logger.Info("running startup validation (pre-flight checks)")

	// Perform health check
	health, err := checker.Check(ctx)
	if err != nil {
		logger.Error("startup validation failed: health check error", zap.Error(err))
		return &StartupValidationResult{
			Passed:     false,
			ErrorCount: 1,
			Messages:   []string{fmt.Sprintf("health check failed: %v", err)},
		}, fmt.Errorf("startup validation failed: %w", err)
	}

	result := &StartupValidationResult{
		Passed: true,
		Health: health,
	}

	// Analyze results
	if health.CorruptCount > 0 {
		msg := fmt.Sprintf("CRITICAL: %d corrupt collection(s) detected - will be quarantined on load", health.CorruptCount)
		result.Messages = append(result.Messages, msg)
		result.ErrorCount++

		for _, hash := range health.Corrupt {
			logger.Warn("corrupt collection detected during startup validation",
				zap.String("collection_hash", hash),
				zap.String("action", "will quarantine on load"))
		}

		if cfg.FailOnCorruption {
			result.Passed = false
			logger.Error("startup blocked: corrupt collections detected and FailOnCorruption=true",
				zap.Int("corrupt_count", health.CorruptCount))
			return result, fmt.Errorf("startup blocked: %d corrupt collection(s) detected", health.CorruptCount)
		}
	}

	if len(health.Empty) > 0 {
		msg := fmt.Sprintf("WARNING: %d empty collection(s) detected", len(health.Empty))
		result.Messages = append(result.Messages, msg)
		result.WarningCount++

		logger.Info("empty collections detected during startup validation",
			zap.Int("count", len(health.Empty)))
	}

	if !health.IsHealthy() && cfg.FailOnDegraded {
		result.Passed = false
		logger.Error("startup blocked: degraded state and FailOnDegraded=true",
			zap.String("status", health.Status()))
		return result, fmt.Errorf("startup blocked: vectorstore in degraded state")
	}

	// Log summary
	if result.Passed {
		logger.Info("startup validation passed",
			zap.Int("total_collections", health.Total),
			zap.Int("healthy", health.HealthyCount),
			zap.Int("corrupt", health.CorruptCount),
			zap.Int("empty", len(health.Empty)),
			zap.Int("warnings", result.WarningCount),
			zap.Int("errors", result.ErrorCount),
			zap.String("status", health.Status()))
	}

	return result, nil
}
