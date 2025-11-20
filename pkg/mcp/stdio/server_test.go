package stdio

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// TestNewServer tests server initialization
func TestNewServer(t *testing.T) {
	tests := []struct {
		name       string
		daemonURL  string
		wantErr    bool
		errMessage string
	}{
		{
			name:      "valid URL",
			daemonURL: "http://localhost:9090",
			wantErr:   false,
		},
		{
			name:       "empty URL",
			daemonURL:  "",
			wantErr:    true,
			errMessage: "daemonURL cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, err := NewServer(tt.daemonURL)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				if err != nil && tt.errMessage != "" && err.Error() != tt.errMessage {
					t.Errorf("Error message = %q, want %q", err.Error(), tt.errMessage)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if server == nil {
				t.Error("Server is nil")
				return
			}

			if server.mcpServer == nil {
				t.Error("mcpServer is nil")
			}

			if server.client == nil {
				t.Error("client is nil")
			}
		})
	}
}

// TestHandleCheckpointSave tests checkpoint_save tool handler with HTTP delegation
func TestHandleCheckpointSave(t *testing.T) {
	tests := []struct {
		name           string
		params         CheckpointSaveParams
		mockResponse   map[string]interface{}
		mockStatusCode int
		wantErr        bool
		wantID         string
	}{
		{
			name: "successful save",
			params: CheckpointSaveParams{
				Summary:     "Test checkpoint",
				ProjectPath: "/test/project",
				Content:     "Test content",
				Tags:        []string{"test"},
			},
			mockResponse: map[string]interface{}{
				"checkpoint_id": "test-id-123",
				"status":        "success",
			},
			mockStatusCode: http.StatusOK,
			wantErr:        false,
			wantID:         "test-id-123",
		},
		{
			name: "daemon returns error",
			params: CheckpointSaveParams{
				Summary:     "Test checkpoint",
				ProjectPath: "/test/project",
			},
			mockResponse:   map[string]interface{}{},
			mockStatusCode: http.StatusInternalServerError,
			wantErr:        true,
		},
		{
			name: "minimal params (no content/tags)",
			params: CheckpointSaveParams{
				Summary:     "Minimal checkpoint",
				ProjectPath: "/test/project",
			},
			mockResponse: map[string]interface{}{
				"checkpoint_id": "min-id-456",
			},
			mockStatusCode: http.StatusOK,
			wantErr:        false,
			wantID:         "min-id-456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock HTTP server
			mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request
				if r.Method != http.MethodPost {
					t.Errorf("Method = %s, want POST", r.Method)
				}

				if r.URL.Path != "/mcp/checkpoint/save" {
					t.Errorf("Path = %s, want /mcp/checkpoint/save", r.URL.Path)
				}

				// Decode request body
				var reqBody map[string]interface{}
				if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
					t.Errorf("Failed to decode request: %v", err)
				}

				// Verify required fields
				if summary, ok := reqBody["summary"].(string); !ok || summary != tt.params.Summary {
					t.Errorf("Summary = %v, want %s", reqBody["summary"], tt.params.Summary)
				}

				if projectPath, ok := reqBody["project_path"].(string); !ok || projectPath != tt.params.ProjectPath {
					t.Errorf("ProjectPath = %v, want %s", reqBody["project_path"], tt.params.ProjectPath)
				}

				// Send mock response
				w.WriteHeader(tt.mockStatusCode)
				if tt.mockStatusCode == http.StatusOK {
					json.NewEncoder(w).Encode(tt.mockResponse)
				} else {
					json.NewEncoder(w).Encode(map[string]string{"error": "internal error"})
				}
			}))
			defer mockServer.Close()

			// Create server with mock daemon URL
			server, err := NewServer(mockServer.URL)
			if err != nil {
				t.Fatalf("Failed to create server: %v", err)
			}

			// Call handler
			result, _, err := server.handleCheckpointSave(context.Background(), &mcpsdk.CallToolRequest{}, &tt.params)

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
			if result == nil {
				t.Fatal("Result is nil")
			}

			if len(result.Content) == 0 {
				t.Fatal("Result content is empty")
			}

			// Check that response contains checkpoint ID
			textContent, ok := result.Content[0].(*mcpsdk.TextContent)
			if !ok {
				t.Fatal("Content is not TextContent")
			}

			if tt.wantID != "" {
				if !containsString(textContent.Text, tt.wantID) {
					t.Errorf("Response text doesn't contain checkpoint ID %s: %s", tt.wantID, textContent.Text)
				}
			}
		})
	}
}

// TestHandleCheckpointSearch tests checkpoint_search tool handler
func TestHandleCheckpointSearch(t *testing.T) {
	tests := []struct {
		name           string
		params         CheckpointSearchParams
		mockResponse   map[string]interface{}
		mockStatusCode int
		wantErr        bool
		wantCount      int
	}{
		{
			name: "successful search with results",
			params: CheckpointSearchParams{
				Query:       "authentication",
				ProjectPath: "/test/project",
				Limit:       5,
			},
			mockResponse: map[string]interface{}{
				"results": []interface{}{
					map[string]interface{}{
						"summary": "Auth implementation",
						"score":   0.95,
					},
					map[string]interface{}{
						"summary": "JWT tokens",
						"score":   0.87,
					},
				},
			},
			mockStatusCode: http.StatusOK,
			wantErr:        false,
			wantCount:      2,
		},
		{
			name: "no results",
			params: CheckpointSearchParams{
				Query:       "nonexistent",
				ProjectPath: "/test/project",
				Limit:       10,
			},
			mockResponse: map[string]interface{}{
				"results": []interface{}{},
			},
			mockStatusCode: http.StatusOK,
			wantErr:        false,
			wantCount:      0,
		},
		{
			name: "default limit",
			params: CheckpointSearchParams{
				Query:       "test",
				ProjectPath: "/test/project",
				Limit:       0, // Should default to 10
			},
			mockResponse: map[string]interface{}{
				"results": []interface{}{},
			},
			mockStatusCode: http.StatusOK,
			wantErr:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock HTTP server
			mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request
				if r.Method != http.MethodPost {
					t.Errorf("Method = %s, want POST", r.Method)
				}

				if r.URL.Path != "/mcp/checkpoint/search" {
					t.Errorf("Path = %s, want /mcp/checkpoint/search", r.URL.Path)
				}

				// Decode request body
				var reqBody map[string]interface{}
				if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
					t.Errorf("Failed to decode request: %v", err)
				}

				// Verify limit is set (default 10 if not specified)
				expectedLimit := tt.params.Limit
				if expectedLimit == 0 {
					expectedLimit = 10
				}
				if limit, ok := reqBody["limit"].(float64); !ok || int(limit) != expectedLimit {
					t.Errorf("Limit = %v, want %d", reqBody["limit"], expectedLimit)
				}

				// Send mock response
				w.WriteHeader(tt.mockStatusCode)
				json.NewEncoder(w).Encode(tt.mockResponse)
			}))
			defer mockServer.Close()

			// Create server with mock daemon URL
			server, err := NewServer(mockServer.URL)
			if err != nil {
				t.Fatalf("Failed to create server: %v", err)
			}

			// Call handler
			result, _, err := server.handleCheckpointSearch(context.Background(), &mcpsdk.CallToolRequest{}, &tt.params)

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
			if result == nil {
				t.Fatal("Result is nil")
			}

			if len(result.Content) == 0 {
				t.Fatal("Result content is empty")
			}

			// Check content
			textContent, ok := result.Content[0].(*mcpsdk.TextContent)
			if !ok {
				t.Fatal("Content is not TextContent")
			}

			// Verify result count in response text
			if tt.wantCount == 0 {
				if !containsString(textContent.Text, "No checkpoints found") {
					t.Errorf("Expected 'No checkpoints found' message, got: %s", textContent.Text)
				}
			} else {
				if !containsString(textContent.Text, "Found") {
					t.Errorf("Expected 'Found' message, got: %s", textContent.Text)
				}
			}
		})
	}
}

// TestHandleStatus tests status tool handler
func TestHandleStatus(t *testing.T) {
	tests := []struct {
		name           string
		mockResponse   map[string]interface{}
		mockStatusCode int
		wantErr        bool
	}{
		{
			name: "healthy status",
			mockResponse: map[string]interface{}{
				"status":  "healthy",
				"version": "1.0.0",
			},
			mockStatusCode: http.StatusOK,
			wantErr:        false,
		},
		{
			name:           "daemon unreachable",
			mockResponse:   map[string]interface{}{},
			mockStatusCode: http.StatusServiceUnavailable,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock HTTP server
			mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request
				if r.Method != http.MethodGet {
					t.Errorf("Method = %s, want GET", r.Method)
				}

				if r.URL.Path != "/health" {
					t.Errorf("Path = %s, want /health", r.URL.Path)
				}

				// Send mock response
				w.WriteHeader(tt.mockStatusCode)
				json.NewEncoder(w).Encode(tt.mockResponse)
			}))
			defer mockServer.Close()

			// Create server with mock daemon URL
			server, err := NewServer(mockServer.URL)
			if err != nil {
				t.Fatalf("Failed to create server: %v", err)
			}

			// Call handler
			result, _, err := server.handleStatus(context.Background(), &mcpsdk.CallToolRequest{}, &StatusParams{})

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
			if result == nil {
				t.Fatal("Result is nil")
			}

			if len(result.Content) == 0 {
				t.Fatal("Result content is empty")
			}

			// Check content contains status info
			textContent, ok := result.Content[0].(*mcpsdk.TextContent)
			if !ok {
				t.Fatal("Content is not TextContent")
			}

			if !containsString(textContent.Text, "healthy") {
				t.Errorf("Expected 'healthy' in response, got: %s", textContent.Text)
			}
		})
	}
}

// Helper function to check if string contains substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
