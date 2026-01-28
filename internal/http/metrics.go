// Package http provides HTTP API with metrics instrumentation.
package http

import (
	"time"

	"github.com/labstack/echo/v4"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.uber.org/zap"
)

const httpInstrumentationName = "github.com/fyrsmithlabs/contextd/internal/http"

// HTTPMetrics holds all HTTP-related metrics.
type HTTPMetrics struct {
	meter          metric.Meter
	logger         *zap.Logger
	requestsTotal  metric.Int64Counter
	requestDur     metric.Float64Histogram
	responseSize   metric.Int64Histogram
	activeRequests metric.Int64UpDownCounter
}

// NewHTTPMetrics creates a new HTTPMetrics instance.
func NewHTTPMetrics(logger *zap.Logger) *HTTPMetrics {
	if logger == nil {
		logger = zap.NewNop()
	}

	m := &HTTPMetrics{
		meter:  otel.Meter(httpInstrumentationName),
		logger: logger,
	}
	m.init()
	return m
}

func (m *HTTPMetrics) init() {
	var err error

	// Total requests by endpoint, method, and status
	m.requestsTotal, err = m.meter.Int64Counter(
		"contextd.http.requests_total",
		metric.WithDescription("Total HTTP requests labeled by method (GET, POST), endpoint (/api/v1/scrub, etc.), and status code. Use rate() for request throughput."),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		m.logger.Warn("failed to create requests counter", zap.Error(err))
	}

	// Request duration histogram
	m.requestDur, err = m.meter.Float64Histogram(
		"contextd.http.request_duration_seconds",
		metric.WithDescription("HTTP request duration in seconds, labeled by method, endpoint, and status. Use histogram_quantile for P50/P95/P99 latency."),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0),
	)
	if err != nil {
		m.logger.Warn("failed to create duration histogram", zap.Error(err))
	}

	// Response size histogram
	m.responseSize, err = m.meter.Int64Histogram(
		"contextd.http.response_size_bytes",
		metric.WithDescription("HTTP response body size in bytes, labeled by method, endpoint, and status. Large responses may indicate inefficient payloads."),
		metric.WithUnit("By"),
		metric.WithExplicitBucketBoundaries(100, 500, 1000, 5000, 10000, 50000, 100000, 500000),
	)
	if err != nil {
		m.logger.Warn("failed to create response size histogram", zap.Error(err))
	}

	// Active requests gauge
	m.activeRequests, err = m.meter.Int64UpDownCounter(
		"contextd.http.active_requests",
		metric.WithDescription("Number of currently active HTTP requests"),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		m.logger.Warn("failed to create active requests gauge", zap.Error(err))
	}
}

// MetricsMiddleware returns an Echo middleware that records HTTP metrics.
func (m *HTTPMetrics) MetricsMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()
			req := c.Request()
			path := c.Path()
			method := req.Method

			// Increment active requests
			if m.activeRequests != nil {
				m.activeRequests.Add(req.Context(), 1)
			}

			// Process request
			err := next(c)

			// Record metrics after request completes
			duration := time.Since(start)
			status := c.Response().Status
			size := c.Response().Size

			// Normalize path to avoid cardinality explosion
			// Replace path parameters with placeholders
			normalizedPath := normalizePath(path)

			attrs := []attribute.KeyValue{
				attribute.String("method", method),
				attribute.String("endpoint", normalizedPath),
				attribute.Int("status", status),
			}

			ctx := req.Context()

			// Record request count
			if m.requestsTotal != nil {
				m.requestsTotal.Add(ctx, 1, metric.WithAttributes(attrs...))
			}

			// Record duration
			if m.requestDur != nil {
				m.requestDur.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))
			}

			// Record response size
			if m.responseSize != nil {
				m.responseSize.Record(ctx, size, metric.WithAttributes(attrs...))
			}

			// Decrement active requests
			if m.activeRequests != nil {
				m.activeRequests.Add(ctx, -1)
			}

			return err
		}
	}
}

// normalizePath replaces dynamic path segments with placeholders to prevent
// metric cardinality explosion.
//
// Current behavior: Returns path as-is because contextd uses only fixed routes:
//   - /api/v1/scrub
//   - /api/v1/threshold
//   - /api/v1/status
//
// Future expansion guide:
// If parameterized routes are added, implement normalization like:
//
//	func normalizePath(path string) string {
//	    // Replace UUID segments: /api/v1/projects/abc-123 -> /api/v1/projects/{id}
//	    uuidRegex := regexp.MustCompile(`/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`)
//	    path = uuidRegex.ReplaceAllString(path, "/{id}")
//
//	    // Replace numeric IDs: /api/v1/items/42 -> /api/v1/items/{id}
//	    numericRegex := regexp.MustCompile(`/\d+`)
//	    path = numericRegex.ReplaceAllString(path, "/{id}")
//
//	    return path
//	}
//
// Why this matters: Without normalization, each unique path becomes a metric label,
// causing cardinality explosion (e.g., 1M unique UUIDs = 1M time series).
func normalizePath(path string) string {
	if path == "" {
		return "/"
	}
	return path
}
