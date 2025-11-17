package monitor

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// RED PHASE: Test 1 - NewMetricsClient constructor
func TestNewMetricsClient(t *testing.T) {
	client := NewMetricsClient("http://localhost:8428")
	assert.NotNil(t, client)
	assert.Equal(t, "http://localhost:8428", client.baseURL)
	assert.NotNil(t, client.client)
}

// RED PHASE: Test 2 - Query basic functionality
func TestMetricsClient_Query_Success(t *testing.T) {
	// Mock VictoriaMetrics server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/query", r.URL.Path)
		assert.Equal(t, "up", r.URL.Query().Get("query"))

		response := QueryResult{
			Status: "success",
			Data: QueryData{
				ResultType: "vector",
				Result: []MetricResult{
					{
						Metric: map[string]string{"job": "contextd"},
						Value:  [2]interface{}{float64(1699564800), "1"},
					},
				},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewMetricsClient(server.URL)
	ctx := context.Background()

	result, err := client.Query(ctx, "up")
	require.NoError(t, err)
	assert.Equal(t, "success", result.Status)
	assert.Equal(t, "vector", result.Data.ResultType)
	assert.Len(t, result.Data.Result, 1)
	assert.Equal(t, "contextd", result.Data.Result[0].Metric["job"])
	assert.Equal(t, "1", result.Data.Result[0].Value[1])
}

// RED PHASE: Test 3 - Query with timeout
func TestMetricsClient_Query_Timeout(t *testing.T) {
	// Server that delays response beyond timeout
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(3 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewMetricsClient(server.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := client.Query(ctx, "up")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "context deadline exceeded")
}

// RED PHASE: Test 4 - Query with HTTP error
func TestMetricsClient_Query_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	client := NewMetricsClient(server.URL)
	ctx := context.Background()

	_, err := client.Query(ctx, "up")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "status code 500")
}

// RED PHASE: Test 5 - Query with malformed JSON
func TestMetricsClient_Query_MalformedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("{invalid json"))
	}))
	defer server.Close()

	client := NewMetricsClient(server.URL)
	ctx := context.Background()

	_, err := client.Query(ctx, "up")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to decode response")
}

// RED PHASE: Test 6 - QueryHTTPRate helper
func TestMetricsClient_QueryHTTPRate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Query().Get("query"), "rate(http_server_request_duration_seconds_count[1m])")

		response := QueryResult{
			Status: "success",
			Data: QueryData{
				ResultType: "vector",
				Result: []MetricResult{
					{
						Metric: map[string]string{},
						Value:  [2]interface{}{float64(1699564800), "45.7"},
					},
				},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewMetricsClient(server.URL)
	ctx := context.Background()

	rate, err := client.QueryHTTPRate(ctx)
	require.NoError(t, err)
	assert.InDelta(t, 45.7, rate, 0.01)
}

// RED PHASE: Test 7 - QueryHTTPRate with no data
func TestMetricsClient_QueryHTTPRate_NoData(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := QueryResult{
			Status: "success",
			Data: QueryData{
				ResultType: "vector",
				Result:     []MetricResult{},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewMetricsClient(server.URL)
	ctx := context.Background()

	rate, err := client.QueryHTTPRate(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0.0, rate)
}

// RED PHASE: Test 8 - QueryHTTPLatencyP95 helper
func TestMetricsClient_QueryHTTPLatencyP95(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Query().Get("query"), "histogram_quantile")

		response := QueryResult{
			Status: "success",
			Data: QueryData{
				ResultType: "vector",
				Result: []MetricResult{
					{
						Metric: map[string]string{},
						Value:  [2]interface{}{float64(1699564800), "0.0123"},
					},
				},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewMetricsClient(server.URL)
	ctx := context.Background()

	latency, err := client.QueryHTTPLatencyP95(ctx)
	require.NoError(t, err)
	assert.InDelta(t, 0.0123, latency, 0.0001)
}

// RED PHASE: Test 9 - QueryEmbeddingRate helper
func TestMetricsClient_QueryEmbeddingRate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Query().Get("query"), "rate(contextd_embedding_operations_total[1m])")

		response := QueryResult{
			Status: "success",
			Data: QueryData{
				ResultType: "vector",
				Result: []MetricResult{
					{
						Metric: map[string]string{},
						Value:  [2]interface{}{float64(1699564800), "120"},
					},
				},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewMetricsClient(server.URL)
	ctx := context.Background()

	rate, err := client.QueryEmbeddingRate(ctx)
	require.NoError(t, err)
	assert.InDelta(t, 120.0, rate, 0.01)
}

// RED PHASE: Test 10 - Query with empty result set
func TestMetricsClient_Query_EmptyResult(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := QueryResult{
			Status: "success",
			Data: QueryData{
				ResultType: "vector",
				Result:     []MetricResult{},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewMetricsClient(server.URL)
	ctx := context.Background()

	result, err := client.Query(ctx, "up")
	require.NoError(t, err)
	assert.Equal(t, "success", result.Status)
	assert.Empty(t, result.Data.Result)
}
