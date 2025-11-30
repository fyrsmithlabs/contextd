package logging

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.uber.org/zap"
)

func TestContextFields_Trace(t *testing.T) {
	// Test with no span context (empty case)
	ctx := context.Background()
	fields := ContextFields(ctx)
	assert.Empty(t, fields)
}

func TestContextFields_OTELTracing(t *testing.T) {
	// Create real OTEL tracer with in-memory exporter
	exporter := tracetest.NewInMemoryExporter()
	provider := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
	)
	tracer := provider.Tracer("test")

	ctx, span := tracer.Start(context.Background(), "test-operation")
	defer span.End()

	fields := ContextFields(ctx)

	// Should have trace_id and span_id
	var hasTraceID, hasSpanID bool
	for _, f := range fields {
		if f.Key == "trace_id" {
			hasTraceID = true
			assert.NotEmpty(t, f.String, "trace_id should not be empty")
		}
		if f.Key == "span_id" {
			hasSpanID = true
			assert.NotEmpty(t, f.String, "span_id should not be empty")
		}
	}
	assert.True(t, hasTraceID, "trace_id field missing from context fields")
	assert.True(t, hasSpanID, "span_id field missing from context fields")
}

func TestContextFields_OTELSampling(t *testing.T) {
	// Test with sampled span (always sample)
	exporter := tracetest.NewInMemoryExporter()
	provider := trace.NewTracerProvider(
		trace.WithSampler(trace.AlwaysSample()),
		trace.WithBatcher(exporter),
	)
	tracer := provider.Tracer("test")

	ctx, span := tracer.Start(context.Background(), "sampled-operation")
	defer span.End()

	fields := ContextFields(ctx)

	// Should have trace_sampled=true
	assertBoolFieldExists(t, fields, "trace_sampled", true)
}

func TestContextFields_Tenant(t *testing.T) {
	tenant := &Tenant{
		OrgID:     "acme",
		TeamID:    "platform",
		ProjectID: "api",
	}
	ctx := context.WithValue(context.Background(), tenantCtxKey{}, tenant)

	fields := ContextFields(ctx)

	assert.Len(t, fields, 3)
	assertFieldExists(t, fields, "tenant.org", "acme")
	assertFieldExists(t, fields, "tenant.team", "platform")
	assertFieldExists(t, fields, "tenant.project", "api")
}

func TestContextFields_Session(t *testing.T) {
	ctx := context.WithValue(context.Background(), sessionCtxKey{}, "sess_123")

	fields := ContextFields(ctx)

	assert.Len(t, fields, 1)
	assertFieldExists(t, fields, "session.id", "sess_123")
}

func TestContextFields_Request(t *testing.T) {
	ctx := context.WithValue(context.Background(), requestCtxKey{}, "req_456")

	fields := ContextFields(ctx)

	assert.Len(t, fields, 1)
	assertFieldExists(t, fields, "request.id", "req_456")
}

func assertFieldExists(t *testing.T, fields []zap.Field, key, expected string) {
	t.Helper()
	for _, field := range fields {
		if field.Key == key && field.String == expected {
			return
		}
	}
	t.Errorf("field %q with value %q not found", key, expected)
}

func assertBoolFieldExists(t *testing.T, fields []zap.Field, key string, expected bool) {
	t.Helper()
	for _, field := range fields {
		if field.Key == key {
			// For boolean fields from zap.Bool(), check the Integer representation
			// zap internally stores bool as integer (1 for true, 0 for false)
			if expected && field.Integer == 1 {
				return
			} else if !expected && field.Integer == 0 {
				return
			}
		}
	}
	t.Errorf("bool field %q with value %v not found", key, expected)
}

func TestLogger_InContext(t *testing.T) {
	logger := &Logger{zap: zap.NewNop(), config: NewDefaultConfig()}
	ctx := WithLogger(context.Background(), logger)

	retrieved := FromContext(ctx)
	assert.Equal(t, logger, retrieved)
}

func TestLogger_FromContextMissing(t *testing.T) {
	ctx := context.Background()
	retrieved := FromContext(ctx)

	// Should return default logger (nop for test)
	assert.NotNil(t, retrieved)
}

// Validation tests

func TestWithTenant_Valid(t *testing.T) {
	tenant := &Tenant{
		OrgID:     "acme",
		TeamID:    "platform",
		ProjectID: "api-server",
	}

	ctx := WithTenant(context.Background(), tenant)
	retrieved := TenantFromContext(ctx)

	assert.Equal(t, tenant, retrieved)
}

func TestWithTenant_NilPanics(t *testing.T) {
	assert.PanicsWithValue(t, "logging: tenant cannot be nil", func() {
		WithTenant(context.Background(), nil)
	})
}

func TestWithTenant_EmptyFieldsPanics(t *testing.T) {
	tests := []struct {
		name   string
		tenant *Tenant
		want   string
	}{
		{
			name:   "empty OrgID",
			tenant: &Tenant{OrgID: "", TeamID: "platform", ProjectID: "api"},
			want:   "logging: tenant.OrgID cannot be empty",
		},
		{
			name:   "empty TeamID",
			tenant: &Tenant{OrgID: "acme", TeamID: "", ProjectID: "api"},
			want:   "logging: tenant.TeamID cannot be empty",
		},
		{
			name:   "empty ProjectID",
			tenant: &Tenant{OrgID: "acme", TeamID: "platform", ProjectID: ""},
			want:   "logging: tenant.ProjectID cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.PanicsWithValue(t, tt.want, func() {
				WithTenant(context.Background(), tt.tenant)
			})
		})
	}
}

func TestWithTenant_InvalidCharactersPanics(t *testing.T) {
	tests := []struct {
		name   string
		tenant *Tenant
	}{
		{
			name:   "OrgID with spaces",
			tenant: &Tenant{OrgID: "acme corp", TeamID: "platform", ProjectID: "api"},
		},
		{
			name:   "TeamID with special chars",
			tenant: &Tenant{OrgID: "acme", TeamID: "platform@dev", ProjectID: "api"},
		},
		{
			name:   "ProjectID with slash",
			tenant: &Tenant{OrgID: "acme", TeamID: "platform", ProjectID: "api/v1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Panics(t, func() {
				WithTenant(context.Background(), tt.tenant)
			})
		})
	}
}

func TestWithTenant_TooLongPanics(t *testing.T) {
	longString := string(make([]byte, 65)) // 65 chars, max is 64
	for i := range longString {
		longString = longString[:i] + "a" + longString[i+1:]
	}

	tenant := &Tenant{
		OrgID:     longString,
		TeamID:    "platform",
		ProjectID: "api",
	}

	assert.Panics(t, func() {
		WithTenant(context.Background(), tenant)
	})
}

func TestWithSessionID_Valid(t *testing.T) {
	tests := []struct {
		name      string
		sessionID string
	}{
		{"simple", "sess_123"},
		{"with hyphens", "sess-abc-123"},
		{"with underscores", "sess_abc_123"},
		{"alphanumeric", "sessABC123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := WithSessionID(context.Background(), tt.sessionID)
			retrieved := SessionIDFromContext(ctx)
			assert.Equal(t, tt.sessionID, retrieved)
		})
	}
}

func TestWithSessionID_EmptyPanics(t *testing.T) {
	assert.PanicsWithValue(t, "logging: sessionID cannot be empty", func() {
		WithSessionID(context.Background(), "")
	})
}

func TestWithSessionID_InvalidCharactersPanics(t *testing.T) {
	tests := []struct {
		name      string
		sessionID string
	}{
		{"with spaces", "sess 123"},
		{"with slash", "sess/123"},
		{"with special chars", "sess@123"},
		{"with dots", "sess.123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Panics(t, func() {
				WithSessionID(context.Background(), tt.sessionID)
			})
		})
	}
}

func TestWithSessionID_TooLongPanics(t *testing.T) {
	longID := string(make([]byte, 129)) // 129 chars, max is 128
	for i := range longID {
		longID = longID[:i] + "a" + longID[i+1:]
	}

	assert.Panics(t, func() {
		WithSessionID(context.Background(), longID)
	})
}

func TestWithRequestID_Valid(t *testing.T) {
	tests := []struct {
		name      string
		requestID string
	}{
		{"simple", "req_456"},
		{"with hyphens", "req-abc-456"},
		{"with underscores", "req_abc_456"},
		{"alphanumeric", "reqABC456"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := WithRequestID(context.Background(), tt.requestID)
			retrieved := RequestIDFromContext(ctx)
			assert.Equal(t, tt.requestID, retrieved)
		})
	}
}

func TestWithRequestID_EmptyPanics(t *testing.T) {
	assert.PanicsWithValue(t, "logging: requestID cannot be empty", func() {
		WithRequestID(context.Background(), "")
	})
}

func TestWithRequestID_InvalidCharactersPanics(t *testing.T) {
	tests := []struct {
		name      string
		requestID string
	}{
		{"with spaces", "req 456"},
		{"with slash", "req/456"},
		{"with special chars", "req@456"},
		{"with dots", "req.456"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Panics(t, func() {
				WithRequestID(context.Background(), tt.requestID)
			})
		})
	}
}

func TestWithRequestID_TooLongPanics(t *testing.T) {
	longID := string(make([]byte, 129)) // 129 chars, max is 128
	for i := range longID {
		longID = longID[:i] + "a" + longID[i+1:]
	}

	assert.Panics(t, func() {
		WithRequestID(context.Background(), longID)
	})
}
