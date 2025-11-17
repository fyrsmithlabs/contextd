package monitor

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// MetricsClient queries VictoriaMetrics API
type MetricsClient struct {
	baseURL string
	client  *http.Client
}

// QueryResult represents the VictoriaMetrics query response
type QueryResult struct {
	Status string    `json:"status"`
	Data   QueryData `json:"data"`
}

// QueryData holds the query result data
type QueryData struct {
	ResultType string         `json:"resultType"`
	Result     []MetricResult `json:"result"`
}

// MetricResult represents a single metric result
type MetricResult struct {
	Metric map[string]string `json:"metric"`
	Value  [2]interface{}    `json:"value"`
}

// NewMetricsClient creates a new metrics client
func NewMetricsClient(baseURL string) *MetricsClient {
	return &MetricsClient{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 2 * time.Second,
		},
	}
}

// Query executes a PromQL query against VictoriaMetrics
func (c *MetricsClient) Query(ctx context.Context, query string) (QueryResult, error) {
	u, err := url.Parse(c.baseURL + "/api/v1/query")
	if err != nil {
		return QueryResult{}, fmt.Errorf("invalid base URL: %w", err)
	}

	q := u.Query()
	q.Set("query", query)
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return QueryResult{}, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return QueryResult{}, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return QueryResult{}, fmt.Errorf("unexpected status code %d", resp.StatusCode)
	}

	var result QueryResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return QueryResult{}, fmt.Errorf("failed to decode response: %w", err)
	}

	return result, nil
}

// QueryHTTPRate queries HTTP request rate
func (c *MetricsClient) QueryHTTPRate(ctx context.Context) (float64, error) {
	result, err := c.Query(ctx, "rate(http_server_request_duration_seconds_count[1m])")
	if err != nil {
		return 0, err
	}
	return extractFloatValue(result)
}

// QueryHTTPLatencyP95 queries HTTP P95 latency
func (c *MetricsClient) QueryHTTPLatencyP95(ctx context.Context) (float64, error) {
	result, err := c.Query(ctx, "histogram_quantile(0.95, rate(http_server_request_duration_seconds_bucket[1m]))")
	if err != nil {
		return 0, err
	}
	return extractFloatValue(result)
}

// QueryEmbeddingRate queries embedding operations rate
func (c *MetricsClient) QueryEmbeddingRate(ctx context.Context) (float64, error) {
	result, err := c.Query(ctx, "rate(contextd_embedding_operations_total[1m])")
	if err != nil {
		return 0, err
	}
	return extractFloatValue(result)
}

// QueryContextTokensUsed queries current context tokens used
func (c *MetricsClient) QueryContextTokensUsed(ctx context.Context) (float64, error) {
	result, err := c.Query(ctx, "contextd_context_tokens_used")
	if err != nil {
		return 0, err
	}
	return extractFloatValue(result)
}

// QueryContextUsagePercent queries current context usage percentage
func (c *MetricsClient) QueryContextUsagePercent(ctx context.Context) (float64, error) {
	result, err := c.Query(ctx, "contextd_context_usage_percent")
	if err != nil {
		return 0, err
	}
	return extractFloatValue(result)
}

// QueryContext70ThresholdHits queries 70% threshold hit count
func (c *MetricsClient) QueryContext70ThresholdHits(ctx context.Context) (float64, error) {
	result, err := c.Query(ctx, `rate(contextd_context_threshold_hit_total{threshold="70_percent"}[5m])`)
	if err != nil {
		return 0, err
	}
	return extractFloatValue(result)
}

// QueryContext90ThresholdHits queries 90% threshold hit count
func (c *MetricsClient) QueryContext90ThresholdHits(ctx context.Context) (float64, error) {
	result, err := c.Query(ctx, `rate(contextd_context_threshold_hit_total{threshold="90_percent"}[5m])`)
	if err != nil {
		return 0, err
	}
	return extractFloatValue(result)
}

// QueryAvgTokensSaved queries average tokens saved per checkpoint
func (c *MetricsClient) QueryAvgTokensSaved(ctx context.Context) (float64, error) {
	result, err := c.Query(ctx, "avg_over_time(contextd_checkpoint_tokens_saved[5m])")
	if err != nil {
		return 0, err
	}
	return extractFloatValue(result)
}

// QueryAvgReductionPct queries average checkpoint reduction percentage
func (c *MetricsClient) QueryAvgReductionPct(ctx context.Context) (float64, error) {
	result, err := c.Query(ctx, "avg_over_time(contextd_checkpoint_reduction_pct[5m])")
	if err != nil {
		return 0, err
	}
	return extractFloatValue(result)
}

// extractFloatValue extracts a float value from query result
func extractFloatValue(result QueryResult) (float64, error) {
	if len(result.Data.Result) == 0 {
		return 0, nil
	}

	valueStr, ok := result.Data.Result[0].Value[1].(string)
	if !ok {
		return 0, fmt.Errorf("value is not a string")
	}

	value, err := strconv.ParseFloat(valueStr, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse value: %w", err)
	}

	return value, nil
}
