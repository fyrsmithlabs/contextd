//go:build integration
// +build integration

package monitor

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMetricsClient_Integration tests against real VictoriaMetrics
// Run with: go test -tags=integration ./internal/monitor/...
func TestMetricsClient_Integration(t *testing.T) {
	// Skip if VictoriaMetrics not available
	vmURL := "http://localhost:8428"
	client := NewMetricsClient(vmURL)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Test basic query
	t.Run("basic_query", func(t *testing.T) {
		result, err := client.Query(ctx, "up")
		require.NoError(t, err, "VictoriaMetrics should be reachable at %s", vmURL)
		assert.NotNil(t, result)
		t.Logf("Query result: %+v", result)
	})

	// Test HTTP rate query
	t.Run("http_rate", func(t *testing.T) {
		rate, err := client.QueryHTTPRate(ctx)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, rate, 0.0, "Rate should be non-negative")
		t.Logf("HTTP rate: %.2f req/min", rate)
	})

	// Test HTTP latency query
	t.Run("http_latency_p95", func(t *testing.T) {
		latency, err := client.QueryHTTPLatencyP95(ctx)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, latency, 0.0, "Latency should be non-negative")
		t.Logf("HTTP P95 latency: %.4fs", latency)
	})

	// Test embedding rate query
	t.Run("embedding_rate", func(t *testing.T) {
		rate, err := client.QueryEmbeddingRate(ctx)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, rate, 0.0, "Embedding rate should be non-negative")
		t.Logf("Embedding rate: %.2f ops/min", rate)
	})

	// Test context monitoring metrics
	t.Run("context_tokens_used", func(t *testing.T) {
		tokens, err := client.QueryContextTokensUsed(ctx)
		// Graceful handling if metric doesn't exist yet
		if err == nil {
			assert.GreaterOrEqual(t, tokens, 0.0, "Tokens should be non-negative")
			t.Logf("Context tokens used: %.0f", tokens)
		} else {
			t.Logf("Context tokens metric not available yet: %v", err)
		}
	})

	t.Run("context_usage_percent", func(t *testing.T) {
		percent, err := client.QueryContextUsagePercent(ctx)
		if err == nil {
			assert.GreaterOrEqual(t, percent, 0.0)
			assert.LessOrEqual(t, percent, 100.0, "Percentage should be 0-100")
			t.Logf("Context usage: %.1f%%", percent)
		} else {
			t.Logf("Context usage metric not available yet: %v", err)
		}
	})

	t.Run("threshold_hits", func(t *testing.T) {
		hits70, err := client.QueryContext70ThresholdHits(ctx)
		if err == nil {
			assert.GreaterOrEqual(t, hits70, 0.0)
			t.Logf("70%% threshold hits: %.1f", hits70)
		}

		hits90, err := client.QueryContext90ThresholdHits(ctx)
		if err == nil {
			assert.GreaterOrEqual(t, hits90, 0.0)
			t.Logf("90%% threshold hits: %.1f", hits90)
		}
	})

	t.Run("checkpoint_effectiveness", func(t *testing.T) {
		avgSaved, err := client.QueryAvgTokensSaved(ctx)
		if err == nil {
			assert.GreaterOrEqual(t, avgSaved, 0.0)
			t.Logf("Avg tokens saved: %.0f", avgSaved)
		}

		avgReduction, err := client.QueryAvgReductionPct(ctx)
		if err == nil {
			assert.GreaterOrEqual(t, avgReduction, 0.0)
			assert.LessOrEqual(t, avgReduction, 100.0)
			t.Logf("Avg reduction: %.1f%%", avgReduction)
		}
	})
}

// TestMonitorModel_Integration tests the full dashboard model with real VictoriaMetrics
func TestMonitorModel_Integration(t *testing.T) {
	vmURL := "http://localhost:8428"
	model := NewModel(vmURL, 5*time.Second)

	// Initialize model
	cmd := model.Init()
	require.NotNil(t, cmd, "Init should return command")

	// Simulate fetching metrics
	fetchCmd := fetchMetrics(vmURL)
	msg := fetchCmd()

	// Should either get metrics or error
	switch msg := msg.(type) {
	case metricsMsg:
		t.Logf("Received metrics: HTTP rate=%.2f, latency=%.4fs, embedding rate=%.2f",
			msg.HTTPRate, msg.HTTPLatencyP95, msg.EmbeddingRate)
		assert.GreaterOrEqual(t, msg.HTTPRate, 0.0)
		assert.GreaterOrEqual(t, msg.HTTPLatencyP95, 0.0)
		assert.GreaterOrEqual(t, msg.EmbeddingRate, 0.0)

	case errMsg:
		t.Logf("Error fetching metrics (expected if contextd not instrumented): %v", msg)

	default:
		t.Fatalf("Unexpected message type: %T", msg)
	}
}
