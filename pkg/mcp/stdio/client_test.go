package stdio

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestNewDaemonClient tests daemon client initialization
func TestNewDaemonClient(t *testing.T) {
	baseURL := "http://localhost:9090"
	client := NewDaemonClient(baseURL)

	if client == nil {
		t.Fatal("Client is nil")
	}

	if client.baseURL != baseURL {
		t.Errorf("baseURL = %s, want %s", client.baseURL, baseURL)
	}

	if client.httpClient == nil {
		t.Error("httpClient is nil")
	}

	if client.httpClient.Timeout != 30*time.Second {
		t.Errorf("Timeout = %v, want 30s", client.httpClient.Timeout)
	}
}

// TestDaemonClient_Post tests HTTP POST requests
func TestDaemonClient_Post(t *testing.T) {
	tests := []struct {
		name           string
		path           string
		request        map[string]interface{}
		mockResponse   map[string]interface{}
		mockStatusCode int
		wantErr        bool
	}{
		{
			name: "successful POST",
			path: "/test/endpoint",
			request: map[string]interface{}{
				"key": "value",
			},
			mockResponse: map[string]interface{}{
				"result": "success",
			},
			mockStatusCode: http.StatusOK,
			wantErr:        false,
		},
		{
			name: "server error",
			path: "/test/endpoint",
			request: map[string]interface{}{
				"key": "value",
			},
			mockResponse:   map[string]interface{}{},
			mockStatusCode: http.StatusInternalServerError,
			wantErr:        true,
		},
		{
			name: "not found",
			path: "/nonexistent",
			request: map[string]interface{}{
				"key": "value",
			},
			mockResponse:   map[string]interface{}{},
			mockStatusCode: http.StatusNotFound,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock server
			mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify method
				if r.Method != http.MethodPost {
					t.Errorf("Method = %s, want POST", r.Method)
				}

				// Verify path
				if r.URL.Path != tt.path {
					t.Errorf("Path = %s, want %s", r.URL.Path, tt.path)
				}

				// Verify content type
				contentType := r.Header.Get("Content-Type")
				if contentType != "application/json" {
					t.Errorf("Content-Type = %s, want application/json", contentType)
				}

				// Decode request body
				var reqBody map[string]interface{}
				if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
					t.Errorf("Failed to decode request: %v", err)
				}

				// Verify request matches
				if !mapsEqual(reqBody, tt.request) {
					t.Errorf("Request body = %v, want %v", reqBody, tt.request)
				}

				// Send response
				w.WriteHeader(tt.mockStatusCode)
				json.NewEncoder(w).Encode(tt.mockResponse)
			}))
			defer mockServer.Close()

			// Create client
			client := NewDaemonClient(mockServer.URL)

			// Make POST request
			var result map[string]interface{}
			err := client.Post(context.Background(), tt.path, tt.request, &result)

			// Check error
			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// Verify result
			if !mapsEqual(result, tt.mockResponse) {
				t.Errorf("Result = %v, want %v", result, tt.mockResponse)
			}
		})
	}
}

// TestDaemonClient_Get tests HTTP GET requests
func TestDaemonClient_Get(t *testing.T) {
	tests := []struct {
		name           string
		path           string
		mockResponse   map[string]interface{}
		mockStatusCode int
		wantErr        bool
	}{
		{
			name: "successful GET",
			path: "/health",
			mockResponse: map[string]interface{}{
				"status":  "healthy",
				"version": "1.0.0",
			},
			mockStatusCode: http.StatusOK,
			wantErr:        false,
		},
		{
			name:           "server error",
			path:           "/health",
			mockResponse:   map[string]interface{}{},
			mockStatusCode: http.StatusInternalServerError,
			wantErr:        true,
		},
		{
			name:           "unauthorized",
			path:           "/secure",
			mockResponse:   map[string]interface{}{},
			mockStatusCode: http.StatusUnauthorized,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock server
			mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify method
				if r.Method != http.MethodGet {
					t.Errorf("Method = %s, want GET", r.Method)
				}

				// Verify path
				if r.URL.Path != tt.path {
					t.Errorf("Path = %s, want %s", r.URL.Path, tt.path)
				}

				// Send response
				w.WriteHeader(tt.mockStatusCode)
				json.NewEncoder(w).Encode(tt.mockResponse)
			}))
			defer mockServer.Close()

			// Create client
			client := NewDaemonClient(mockServer.URL)

			// Make GET request
			var result map[string]interface{}
			err := client.Get(context.Background(), tt.path, &result)

			// Check error
			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// Verify result
			if !mapsEqual(result, tt.mockResponse) {
				t.Errorf("Result = %v, want %v", result, tt.mockResponse)
			}
		})
	}
}

// TestDaemonClient_ContextCancellation tests context cancellation handling
func TestDaemonClient_ContextCancellation(t *testing.T) {
	// Create mock server that delays response
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"result": "success"})
	}))
	defer mockServer.Close()

	// Create client
	client := NewDaemonClient(mockServer.URL)

	// Create context that will be canceled
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Make request with canceled context
	var result map[string]interface{}
	err := client.Post(ctx, "/test", map[string]string{"key": "value"}, &result)

	// Should get context canceled error
	if err == nil {
		t.Error("Expected context cancellation error, got nil")
	}
}

// TestDaemonClient_Timeout tests request timeout handling
func TestDaemonClient_Timeout(t *testing.T) {
	// Create mock server that never responds
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(60 * time.Second) // Longer than client timeout
	}))
	defer mockServer.Close()

	// Create client with short timeout
	client := NewDaemonClient(mockServer.URL)
	client.httpClient.Timeout = 10 * time.Millisecond

	// Make request
	var result map[string]interface{}
	err := client.Post(context.Background(), "/test", map[string]string{"key": "value"}, &result)

	// Should get timeout error
	if err == nil {
		t.Error("Expected timeout error, got nil")
	}
}

// Helper function to compare maps
func mapsEqual(a, b map[string]interface{}) bool {
	if len(a) != len(b) {
		return false
	}

	for k, v := range a {
		if bv, ok := b[k]; !ok || v != bv {
			return false
		}
	}

	return true
}
