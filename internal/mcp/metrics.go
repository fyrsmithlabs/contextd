// Package mcp provides MCP server with metrics instrumentation.
package mcp

import (
	"context"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.uber.org/zap"
)

const instrumentationName = "github.com/fyrsmithlabs/contextd/internal/mcp"

// Metrics holds all MCP-related metrics.
type Metrics struct {
	meter          metric.Meter
	logger         *zap.Logger
	invocations    metric.Int64Counter
	duration       metric.Float64Histogram
	errors         metric.Int64Counter
	activeRequests metric.Int64UpDownCounter
}

// NewMetrics creates a new Metrics instance.
func NewMetrics(logger *zap.Logger) *Metrics {
	m := &Metrics{
		meter:  otel.Meter(instrumentationName),
		logger: logger,
	}
	m.init()
	return m
}

func (m *Metrics) init() {
	var err error

	// Total tool invocations by tool name
	m.invocations, err = m.meter.Int64Counter(
		"contextd.mcp.tool.invocations_total",
		metric.WithDescription("Total number of MCP tool invocations"),
		metric.WithUnit("{invocation}"),
	)
	if err != nil {
		m.logger.Warn("failed to create invocations counter", zap.Error(err))
	}

	// Tool execution duration histogram
	m.duration, err = m.meter.Float64Histogram(
		"contextd.mcp.tool.duration_seconds",
		metric.WithDescription("Duration of MCP tool invocations"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0),
	)
	if err != nil {
		m.logger.Warn("failed to create duration histogram", zap.Error(err))
	}

	// Error count by tool and reason
	m.errors, err = m.meter.Int64Counter(
		"contextd.mcp.tool.errors_total",
		metric.WithDescription("Total number of MCP tool errors"),
		metric.WithUnit("{error}"),
	)
	if err != nil {
		m.logger.Warn("failed to create errors counter", zap.Error(err))
	}

	// Active concurrent requests gauge
	m.activeRequests, err = m.meter.Int64UpDownCounter(
		"contextd.mcp.tool.active_requests",
		metric.WithDescription("Number of currently active MCP tool requests"),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		m.logger.Warn("failed to create active requests gauge", zap.Error(err))
	}
}

// RecordInvocation records a tool invocation metric.
func (m *Metrics) RecordInvocation(ctx context.Context, toolName string, duration time.Duration, err error) {
	attrs := []attribute.KeyValue{
		attribute.String("tool", toolName),
	}

	// Record invocation count
	if m.invocations != nil {
		m.invocations.Add(ctx, 1, metric.WithAttributes(attrs...))
	}

	// Record duration
	if m.duration != nil {
		m.duration.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))
	}

	// Record error if present
	if err != nil && m.errors != nil {
		errorAttrs := append(attrs, attribute.String("reason", categorizeError(err)))
		m.errors.Add(ctx, 1, metric.WithAttributes(errorAttrs...))
	}
}

// IncrementActive increments the active requests counter.
func (m *Metrics) IncrementActive(ctx context.Context, toolName string) {
	if m.activeRequests != nil {
		m.activeRequests.Add(ctx, 1, metric.WithAttributes(
			attribute.String("tool", toolName),
		))
	}
}

// DecrementActive decrements the active requests counter.
func (m *Metrics) DecrementActive(ctx context.Context, toolName string) {
	if m.activeRequests != nil {
		m.activeRequests.Add(ctx, -1, metric.WithAttributes(
			attribute.String("tool", toolName),
		))
	}
}

// categorizeError categorizes an error into a reason string.
func categorizeError(err error) string {
	if err == nil {
		return ""
	}

	errStr := strings.ToLower(err.Error())

	switch {
	case strings.Contains(errStr, "tenant"):
		return "tenant_error"
	case strings.Contains(errStr, "validation") || strings.Contains(errStr, "invalid"):
		return "validation_error"
	case strings.Contains(errStr, "not found"):
		return "not_found"
	case strings.Contains(errStr, "timeout"):
		return "timeout"
	case strings.Contains(errStr, "permission") || strings.Contains(errStr, "unauthorized"):
		return "auth_error"
	case strings.Contains(errStr, "vectorstore") || strings.Contains(errStr, "embedding"):
		return "storage_error"
	default:
		return "internal_error"
	}
}
