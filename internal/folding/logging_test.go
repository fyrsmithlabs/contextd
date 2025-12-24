package folding

import (
	"context"
	"testing"
	"time"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func newTestLogger() (*Logger, *observer.ObservedLogs) {
	core, observed := observer.New(zapcore.DebugLevel)
	zapLogger := zap.New(core)
	return NewLogger(zapLogger), observed
}

func TestNewLogger(t *testing.T) {
	// Test with nil logger
	l := NewLogger(nil)
	if l == nil {
		t.Fatal("NewLogger(nil) returned nil")
	}

	// Test with real logger
	zapLogger := zap.NewNop()
	l = NewLogger(zapLogger)
	if l == nil {
		t.Fatal("NewLogger(zap.NewNop()) returned nil")
	}
}

func TestLogger_BranchCreated(t *testing.T) {
	l, logs := newTestLogger()
	ctx := context.Background()

	l.BranchCreated(ctx, "br_123", "sess_001", 2, 8192)

	entries := logs.All()
	if len(entries) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(entries))
	}

	entry := entries[0]
	if entry.Message != "branch created" {
		t.Errorf("message = %q, want %q", entry.Message, "branch created")
	}
	if entry.Level != zapcore.InfoLevel {
		t.Errorf("level = %v, want Info", entry.Level)
	}

	// Check fields
	fields := entry.ContextMap()
	if fields["branch_id"] != "br_123" {
		t.Errorf("branch_id = %v, want br_123", fields["branch_id"])
	}
	if fields["session_id"] != "sess_001" {
		t.Errorf("session_id = %v, want sess_001", fields["session_id"])
	}
	if fields["depth"].(int64) != 2 {
		t.Errorf("depth = %v, want 2", fields["depth"])
	}
	if fields["budget"].(int64) != 8192 {
		t.Errorf("budget = %v, want 8192", fields["budget"])
	}
}

func TestLogger_BranchReturned(t *testing.T) {
	l, logs := newTestLogger()
	ctx := context.Background()

	l.BranchReturned(ctx, "br_123", "sess_001", 1, 4096, 8192, 30*time.Second)

	entries := logs.All()
	if len(entries) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(entries))
	}

	entry := entries[0]
	if entry.Message != "branch returned" {
		t.Errorf("message = %q, want %q", entry.Message, "branch returned")
	}
	if entry.Level != zapcore.InfoLevel {
		t.Errorf("level = %v, want Info", entry.Level)
	}

	fields := entry.ContextMap()
	if fields["tokens_used"].(int64) != 4096 {
		t.Errorf("tokens_used = %v, want 4096", fields["tokens_used"])
	}
	if fields["budget_utilization"].(float64) != 0.5 {
		t.Errorf("budget_utilization = %v, want 0.5", fields["budget_utilization"])
	}
}

func TestLogger_BranchTimeout(t *testing.T) {
	l, logs := newTestLogger()
	ctx := context.Background()

	l.BranchTimeout(ctx, "br_123", "sess_001", 0, 1000, 8192, 300, 5*time.Minute)

	entries := logs.All()
	if len(entries) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(entries))
	}

	entry := entries[0]
	if entry.Message != "branch timeout" {
		t.Errorf("message = %q, want %q", entry.Message, "branch timeout")
	}
	if entry.Level != zapcore.WarnLevel {
		t.Errorf("level = %v, want Warn", entry.Level)
	}

	fields := entry.ContextMap()
	if fields["timeout_seconds"].(int64) != 300 {
		t.Errorf("timeout_seconds = %v, want 300", fields["timeout_seconds"])
	}
}

func TestLogger_BranchFailed(t *testing.T) {
	l, logs := newTestLogger()
	ctx := context.Background()

	l.BranchFailed(ctx, "br_123", "sess_001", 1, "budget exhausted", 8192, 8192, time.Minute)

	entries := logs.All()
	if len(entries) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(entries))
	}

	entry := entries[0]
	if entry.Message != "branch failed" {
		t.Errorf("message = %q, want %q", entry.Message, "branch failed")
	}
	if entry.Level != zapcore.WarnLevel {
		t.Errorf("level = %v, want Warn", entry.Level)
	}

	fields := entry.ContextMap()
	if fields["reason"] != "budget exhausted" {
		t.Errorf("reason = %v, want 'budget exhausted'", fields["reason"])
	}
}

func TestLogger_BudgetExhausted(t *testing.T) {
	l, logs := newTestLogger()
	ctx := context.Background()

	l.BudgetExhausted(ctx, "br_123", 8192, 8192)

	entries := logs.All()
	if len(entries) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(entries))
	}

	entry := entries[0]
	if entry.Message != "budget exhausted" {
		t.Errorf("message = %q, want %q", entry.Message, "budget exhausted")
	}
	if entry.Level != zapcore.WarnLevel {
		t.Errorf("level = %v, want Warn", entry.Level)
	}

	fields := entry.ContextMap()
	if fields["utilization"].(float64) != 1.0 {
		t.Errorf("utilization = %v, want 1.0", fields["utilization"])
	}
}

func TestLogger_BudgetWarning(t *testing.T) {
	l, logs := newTestLogger()
	ctx := context.Background()

	l.BudgetWarning(ctx, "br_123", 6554, 8192, 0.8)

	entries := logs.All()
	if len(entries) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(entries))
	}

	entry := entries[0]
	if entry.Message != "budget warning threshold reached" {
		t.Errorf("message = %q, want %q", entry.Message, "budget warning threshold reached")
	}
	if entry.Level != zapcore.WarnLevel {
		t.Errorf("level = %v, want Warn", entry.Level)
	}
}

func TestLogger_SessionCleanup(t *testing.T) {
	l, logs := newTestLogger()
	ctx := context.Background()

	l.SessionCleanup(ctx, "sess_001", 3)

	entries := logs.All()
	if len(entries) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(entries))
	}

	entry := entries[0]
	if entry.Message != "session cleanup" {
		t.Errorf("message = %q, want %q", entry.Message, "session cleanup")
	}
	if entry.Level != zapcore.InfoLevel {
		t.Errorf("level = %v, want Info", entry.Level)
	}

	fields := entry.ContextMap()
	if fields["branch_count"].(int64) != 3 {
		t.Errorf("branch_count = %v, want 3", fields["branch_count"])
	}
}

func TestLogger_ForceReturn(t *testing.T) {
	l, logs := newTestLogger()
	ctx := context.Background()

	l.ForceReturn(ctx, "br_123", "sess_001", 2, "parent returning")

	entries := logs.All()
	if len(entries) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(entries))
	}

	entry := entries[0]
	if entry.Message != "branch force-returned" {
		t.Errorf("message = %q, want %q", entry.Message, "branch force-returned")
	}
	if entry.Level != zapcore.WarnLevel {
		t.Errorf("level = %v, want Warn", entry.Level)
	}
}

func TestLogger_Error(t *testing.T) {
	l, logs := newTestLogger()
	ctx := context.Background()

	l.Error(ctx, "operation failed", ErrBranchNotFound, zap.String("branch_id", "br_123"))

	entries := logs.All()
	if len(entries) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(entries))
	}

	entry := entries[0]
	if entry.Message != "operation failed" {
		t.Errorf("message = %q, want %q", entry.Message, "operation failed")
	}
	if entry.Level != zapcore.ErrorLevel {
		t.Errorf("level = %v, want Error", entry.Level)
	}
}

func TestLogger_Debug(t *testing.T) {
	l, logs := newTestLogger()
	ctx := context.Background()

	l.Debug(ctx, "debug info", zap.String("key", "value"))

	entries := logs.All()
	if len(entries) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(entries))
	}

	entry := entries[0]
	if entry.Message != "debug info" {
		t.Errorf("message = %q, want %q", entry.Message, "debug info")
	}
	if entry.Level != zapcore.DebugLevel {
		t.Errorf("level = %v, want Debug", entry.Level)
	}
}

func TestLogger_NilSafe(t *testing.T) {
	var l *Logger
	ctx := context.Background()

	// Should not panic with nil receiver
	l.BranchCreated(ctx, "br_123", "sess_001", 0, 8192)
	l.BranchReturned(ctx, "br_123", "sess_001", 0, 1000, 8192, time.Second)
	l.BranchTimeout(ctx, "br_123", "sess_001", 0, 1000, 8192, 300, time.Minute)
	l.BranchFailed(ctx, "br_123", "sess_001", 0, "test", 1000, 8192, time.Second)
	l.BudgetExhausted(ctx, "br_123", 8192, 8192)
	l.BudgetWarning(ctx, "br_123", 6500, 8192, 0.8)
	l.SessionCleanup(ctx, "sess_001", 3)
	l.ForceReturn(ctx, "br_123", "sess_001", 0, "test")
	l.Error(ctx, "test", ErrBranchNotFound)
	l.Debug(ctx, "test")
}

func TestLogger_WithTraceContext(t *testing.T) {
	l, logs := newTestLogger()

	// Set up trace context
	spanRecorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(spanRecorder))
	defer func() { _ = tp.Shutdown(context.Background()) }()

	ctx := context.Background()
	tracer := tp.Tracer(InstrumentationName)
	ctx, span := tracer.Start(ctx, "test")
	defer span.End()

	l.BranchCreated(ctx, "br_123", "sess_001", 0, 8192)

	entries := logs.All()
	if len(entries) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(entries))
	}

	// Should have trace fields
	entry := entries[0]
	fields := entry.ContextMap()
	if _, ok := fields["trace_id"]; !ok {
		t.Error("expected trace_id field")
	}
	if _, ok := fields["span_id"]; !ok {
		t.Error("expected span_id field")
	}
}

func TestBudgetUtilization(t *testing.T) {
	tests := []struct {
		used, total int
		expected    float64
	}{
		{0, 0, 0},
		{0, 100, 0},
		{50, 100, 0.5},
		{100, 100, 1.0},
		{100, 0, 0}, // Edge case: zero total
		{-1, 100, 0}, // Edge case: negative would give negative result
	}

	for _, tt := range tests {
		result := budgetUtilization(tt.used, tt.total)
		if tt.total <= 0 && result != 0 {
			t.Errorf("budgetUtilization(%d, %d) = %f, want 0 for zero/negative total", tt.used, tt.total, result)
		} else if tt.total > 0 {
			expected := float64(tt.used) / float64(tt.total)
			if result != expected {
				t.Errorf("budgetUtilization(%d, %d) = %f, want %f", tt.used, tt.total, result, expected)
			}
		}
	}
}
