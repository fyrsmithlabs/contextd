// internal/logging/context.go
package logging

import (
	"context"
	"fmt"
	"regexp"
	"unicode/utf8"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// ContextFields extracts correlation data from context.
func ContextFields(ctx context.Context) []zap.Field {
	fields := make([]zap.Field, 0, 8)

	// Trace correlation (from OpenTelemetry)
	if span := trace.SpanFromContext(ctx); span.SpanContext().IsValid() {
		sc := span.SpanContext()
		fields = append(fields,
			zap.String("trace_id", sc.TraceID().String()),
			zap.String("span_id", sc.SpanID().String()),
		)
		if sc.IsSampled() {
			fields = append(fields, zap.Bool("trace_sampled", true))
		}
	}

	// Tenant context
	if tenant := TenantFromContext(ctx); tenant != nil {
		fields = append(fields,
			zap.String("tenant.org", tenant.OrgID),
			zap.String("tenant.team", tenant.TeamID),
			zap.String("tenant.project", tenant.ProjectID),
		)
	}

	// Session context
	if sessionID := SessionIDFromContext(ctx); sessionID != "" {
		fields = append(fields, zap.String("session.id", sessionID))
	}

	// Request ID
	if requestID := RequestIDFromContext(ctx); requestID != "" {
		fields = append(fields, zap.String("request.id", requestID))
	}

	return fields
}

// Context key types
type tenantCtxKey struct{}
type sessionCtxKey struct{}
type requestCtxKey struct{}

// Tenant represents multi-tenant context.
type Tenant struct {
	OrgID     string
	TeamID    string
	ProjectID string
}

// Validation constants
const (
	maxTenantFieldLen = 64
	maxIDLen          = 128
)

var (
	// tenantFieldPattern allows alphanumeric, hyphen, underscore
	tenantFieldPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	// idPattern allows alphanumeric, hyphen, underscore with optional prefix
	idPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
)

// validateTenantField validates a tenant field (org, team, project ID).
func validateTenantField(field, name string) error {
	if field == "" {
		return fmt.Errorf("%s cannot be empty", name)
	}
	if !utf8.ValidString(field) {
		return fmt.Errorf("%s contains invalid UTF-8", name)
	}
	if len(field) > maxTenantFieldLen {
		return fmt.Errorf("%s exceeds max length %d", name, maxTenantFieldLen)
	}
	if !tenantFieldPattern.MatchString(field) {
		return fmt.Errorf("%s contains invalid characters (must be alphanumeric, hyphen, underscore)", name)
	}
	return nil
}

// validateID validates a session or request ID.
func validateID(id, name string) error {
	if id == "" {
		return fmt.Errorf("%s cannot be empty", name)
	}
	if !utf8.ValidString(id) {
		return fmt.Errorf("%s contains invalid UTF-8", name)
	}
	if len(id) > maxIDLen {
		return fmt.Errorf("%s exceeds max length %d", name, maxIDLen)
	}
	if !idPattern.MatchString(id) {
		return fmt.Errorf("%s contains invalid characters (must be alphanumeric, hyphen, underscore)", name)
	}
	return nil
}

// TenantFromContext extracts tenant from context.
func TenantFromContext(ctx context.Context) *Tenant {
	if t, ok := ctx.Value(tenantCtxKey{}).(*Tenant); ok {
		return t
	}
	return nil
}

// WithTenant adds tenant to context.
// Panics if tenant is nil or contains invalid field values.
func WithTenant(ctx context.Context, tenant *Tenant) context.Context {
	if tenant == nil {
		panic("logging: tenant cannot be nil")
	}
	// Validate all tenant fields
	if err := validateTenantField(tenant.OrgID, "tenant.OrgID"); err != nil {
		panic(fmt.Sprintf("logging: %v", err))
	}
	if err := validateTenantField(tenant.TeamID, "tenant.TeamID"); err != nil {
		panic(fmt.Sprintf("logging: %v", err))
	}
	if err := validateTenantField(tenant.ProjectID, "tenant.ProjectID"); err != nil {
		panic(fmt.Sprintf("logging: %v", err))
	}
	return context.WithValue(ctx, tenantCtxKey{}, tenant)
}

// SessionIDFromContext extracts session ID from context.
func SessionIDFromContext(ctx context.Context) string {
	if s, ok := ctx.Value(sessionCtxKey{}).(string); ok {
		return s
	}
	return ""
}

// WithSessionID adds session ID to context.
// Panics if sessionID is empty or contains invalid characters.
func WithSessionID(ctx context.Context, sessionID string) context.Context {
	if err := validateID(sessionID, "sessionID"); err != nil {
		panic(fmt.Sprintf("logging: %v", err))
	}
	return context.WithValue(ctx, sessionCtxKey{}, sessionID)
}

// RequestIDFromContext extracts request ID from context.
func RequestIDFromContext(ctx context.Context) string {
	if r, ok := ctx.Value(requestCtxKey{}).(string); ok {
		return r
	}
	return ""
}

// WithRequestID adds request ID to context.
// Panics if requestID is empty or contains invalid characters.
func WithRequestID(ctx context.Context, requestID string) context.Context {
	if err := validateID(requestID, "requestID"); err != nil {
		panic(fmt.Sprintf("logging: %v", err))
	}
	return context.WithValue(ctx, requestCtxKey{}, requestID)
}

// loggerCtxKey is the context key for Logger.
type loggerCtxKey struct{}

// WithLogger stores logger in context.
func WithLogger(ctx context.Context, logger *Logger) context.Context {
	return context.WithValue(ctx, loggerCtxKey{}, logger)
}

// FromContext retrieves logger from context.
// Returns a default nop logger if not found.
func FromContext(ctx context.Context) *Logger {
	if l, ok := ctx.Value(loggerCtxKey{}).(*Logger); ok {
		return l
	}
	return &Logger{zap: zap.NewNop(), config: NewDefaultConfig()}
}
