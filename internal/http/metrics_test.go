package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.uber.org/zap"
)

func TestHTTPMetrics_MetricsMiddleware(t *testing.T) {
	reader := metric.NewManualReader()
	mp := metric.NewMeterProvider(metric.WithReader(reader))

	logger := zap.NewNop()
	m := &HTTPMetrics{
		meter:  mp.Meter(httpInstrumentationName),
		logger: logger,
	}
	m.init()

	// Create Echo instance with middleware
	e := echo.New()
	e.Use(m.MetricsMiddleware())
	e.GET("/test", func(c echo.Context) error {
		return c.String(http.StatusOK, "hello")
	})
	e.GET("/health", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})
	e.POST("/api/scrub", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	// Make test requests
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	req = httptest.NewRequest(http.MethodGet, "/health", nil)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	req = httptest.NewRequest(http.MethodPost, "/api/scrub", nil)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	// Collect metrics
	ctx := context.Background()
	var rm metricdata.ResourceMetrics
	if err := reader.Collect(ctx, &rm); err != nil {
		t.Fatalf("failed to collect metrics: %v", err)
	}

	// Check for expected metrics
	foundRequests := false
	foundDuration := false
	foundResponseSize := false

	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			switch m.Name {
			case "contextd.http.requests_total":
				foundRequests = true
				if sum, ok := m.Data.(metricdata.Sum[int64]); ok {
					total := int64(0)
					for _, dp := range sum.DataPoints {
						total += dp.Value
					}
					if total != 3 {
						t.Errorf("expected 3 requests, got %d", total)
					}
				}
			case "contextd.http.request_duration_seconds":
				foundDuration = true
				if hist, ok := m.Data.(metricdata.Histogram[float64]); ok {
					total := uint64(0)
					for _, dp := range hist.DataPoints {
						total += dp.Count
					}
					if total != 3 {
						t.Errorf("expected 3 duration recordings, got %d", total)
					}
				}
			case "contextd.http.response_size_bytes":
				foundResponseSize = true
			}
		}
	}

	if !foundRequests {
		t.Error("requests counter not found")
	}
	if !foundDuration {
		t.Error("duration histogram not found")
	}
	if !foundResponseSize {
		t.Error("response size histogram not found")
	}
}

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", "/"},
		{"/health", "/health"},
		{"/api/v1/scrub", "/api/v1/scrub"},
		{"/api/v1/status", "/api/v1/status"},
	}

	for _, tt := range tests {
		result := normalizePath(tt.input)
		if result != tt.expected {
			t.Errorf("normalizePath(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}
