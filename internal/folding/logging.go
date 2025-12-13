// Package folding provides context-folding for LLM agent context management.
package folding

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// Logger wraps zap.Logger with folding-specific structured logging.
type Logger struct {
	logger *zap.Logger
}

// NewLogger creates a new Logger. If logger is nil, uses a no-op logger.
func NewLogger(logger *zap.Logger) *Logger {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Logger{logger: logger.Named("folding")}
}

// BranchCreated logs a branch creation event.
func (l *Logger) BranchCreated(ctx context.Context, branchID, sessionID string, depth, budget int) {
	if l == nil || l.logger == nil {
		return
	}
	fields := l.baseFields(ctx, branchID, sessionID, depth)
	fields = append(fields, zap.Int("budget", budget))
	l.logger.Info("branch created", fields...)
}

// BranchReturned logs a successful branch return.
func (l *Logger) BranchReturned(ctx context.Context, branchID, sessionID string, depth, tokensUsed, budget int, duration time.Duration) {
	if l == nil || l.logger == nil {
		return
	}
	fields := l.baseFields(ctx, branchID, sessionID, depth)
	fields = append(fields,
		zap.Int("tokens_used", tokensUsed),
		zap.Int("budget", budget),
		zap.Duration("duration", duration),
		zap.Float64("budget_utilization", budgetUtilization(tokensUsed, budget)),
	)
	l.logger.Info("branch returned", fields...)
}

// BranchTimeout logs a branch timeout.
func (l *Logger) BranchTimeout(ctx context.Context, branchID, sessionID string, depth, tokensUsed, budget int, timeoutSeconds int, duration time.Duration) {
	if l == nil || l.logger == nil {
		return
	}
	fields := l.baseFields(ctx, branchID, sessionID, depth)
	fields = append(fields,
		zap.Int("tokens_used", tokensUsed),
		zap.Int("budget", budget),
		zap.Int("timeout_seconds", timeoutSeconds),
		zap.Duration("duration", duration),
	)
	l.logger.Warn("branch timeout", fields...)
}

// BranchFailed logs a branch failure.
func (l *Logger) BranchFailed(ctx context.Context, branchID, sessionID string, depth int, reason string, tokensUsed, budget int, duration time.Duration) {
	if l == nil || l.logger == nil {
		return
	}
	fields := l.baseFields(ctx, branchID, sessionID, depth)
	fields = append(fields,
		zap.String("reason", reason),
		zap.Int("tokens_used", tokensUsed),
		zap.Int("budget", budget),
		zap.Duration("duration", duration),
	)
	l.logger.Warn("branch failed", fields...)
}

// BudgetExhausted logs a budget exhaustion event.
func (l *Logger) BudgetExhausted(ctx context.Context, branchID string, budgetUsed, budgetTotal int) {
	if l == nil || l.logger == nil {
		return
	}
	fields := []zap.Field{
		zap.String("branch_id", branchID),
		zap.Int("budget_used", budgetUsed),
		zap.Int("budget_total", budgetTotal),
		zap.Float64("utilization", budgetUtilization(budgetUsed, budgetTotal)),
	}
	fields = append(fields, l.traceFields(ctx)...)
	l.logger.Warn("budget exhausted", fields...)
}

// BudgetWarning logs a budget warning event (80% usage).
func (l *Logger) BudgetWarning(ctx context.Context, branchID string, budgetUsed, budgetTotal int, percentage float64) {
	if l == nil || l.logger == nil {
		return
	}
	fields := []zap.Field{
		zap.String("branch_id", branchID),
		zap.Int("budget_used", budgetUsed),
		zap.Int("budget_total", budgetTotal),
		zap.Float64("percentage", percentage),
	}
	fields = append(fields, l.traceFields(ctx)...)
	l.logger.Warn("budget warning threshold reached", fields...)
}

// SessionCleanup logs a session cleanup event.
func (l *Logger) SessionCleanup(ctx context.Context, sessionID string, branchCount int) {
	if l == nil || l.logger == nil {
		return
	}
	fields := []zap.Field{
		zap.String("session_id", sessionID),
		zap.Int("branch_count", branchCount),
	}
	fields = append(fields, l.traceFields(ctx)...)
	l.logger.Info("session cleanup", fields...)
}

// ForceReturn logs a force return event.
func (l *Logger) ForceReturn(ctx context.Context, branchID, sessionID string, depth int, reason string) {
	if l == nil || l.logger == nil {
		return
	}
	fields := l.baseFields(ctx, branchID, sessionID, depth)
	fields = append(fields, zap.String("reason", reason))
	l.logger.Warn("branch force-returned", fields...)
}

// Error logs an error with context.
func (l *Logger) Error(ctx context.Context, msg string, err error, fields ...zap.Field) {
	if l == nil || l.logger == nil {
		return
	}
	allFields := l.traceFields(ctx)
	allFields = append(allFields, zap.Error(err))
	allFields = append(allFields, fields...)
	l.logger.Error(msg, allFields...)
}

// Debug logs a debug message with context.
func (l *Logger) Debug(ctx context.Context, msg string, fields ...zap.Field) {
	if l == nil || l.logger == nil {
		return
	}
	allFields := l.traceFields(ctx)
	allFields = append(allFields, fields...)
	l.logger.Debug(msg, allFields...)
}

// baseFields returns common fields for branch events.
func (l *Logger) baseFields(ctx context.Context, branchID, sessionID string, depth int) []zap.Field {
	fields := []zap.Field{
		zap.String("branch_id", branchID),
		zap.String("session_id", sessionID),
		zap.Int("depth", depth),
	}
	return append(fields, l.traceFields(ctx)...)
}

// traceFields extracts trace context from the context.
func (l *Logger) traceFields(ctx context.Context) []zap.Field {
	span := trace.SpanFromContext(ctx)
	if !span.SpanContext().IsValid() {
		return nil
	}
	sc := span.SpanContext()
	fields := []zap.Field{
		zap.String("trace_id", sc.TraceID().String()),
		zap.String("span_id", sc.SpanID().String()),
	}
	if sc.IsSampled() {
		fields = append(fields, zap.Bool("trace_sampled", true))
	}
	return fields
}

// budgetUtilization calculates budget utilization ratio.
func budgetUtilization(used, total int) float64 {
	if total <= 0 {
		return 0
	}
	return float64(used) / float64(total)
}
