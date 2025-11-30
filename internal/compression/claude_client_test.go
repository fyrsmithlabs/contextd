package compression

import (
	"context"
	"strings"
	"testing"
)

// MockClaudeClient implements ClaudeClient for testing
type MockClaudeClient struct {
	SummarizeFn func(ctx context.Context, content string, targetRatio float64) (string, error)
}

func (m *MockClaudeClient) Summarize(ctx context.Context, content string, targetRatio float64) (string, error) {
	if m.SummarizeFn != nil {
		return m.SummarizeFn(ctx, content, targetRatio)
	}
	// Default mock behavior: simple compression
	lines := strings.Split(content, "\n")
	targetLines := int(float64(len(lines)) / targetRatio)
	if targetLines < 1 {
		targetLines = 1
	}
	if len(lines) <= targetLines {
		return content, nil
	}
	return strings.Join(lines[:targetLines], "\n"), nil
}

// TestNewClaudeClient tests client creation
func TestNewClaudeClient(t *testing.T) {
	tests := []struct {
		name    string
		apiKey  string
		baseURL string
		model   string
		wantErr bool
	}{
		{
			name:    "valid config",
			apiKey:  "sk-ant-test123",
			baseURL: "https://api.anthropic.com",
			model:   "claude-3-5-sonnet-20241022",
			wantErr: false,
		},
		{
			name:    "empty API key",
			apiKey:  "",
			baseURL: "https://api.anthropic.com",
			model:   "claude-3-5-sonnet-20241022",
			wantErr: true,
		},
		{
			name:    "default baseURL",
			apiKey:  "sk-ant-test123",
			baseURL: "",
			model:   "claude-3-5-sonnet-20241022",
			wantErr: false,
		},
		{
			name:    "default model",
			apiKey:  "sk-ant-test123",
			baseURL: "https://api.anthropic.com",
			model:   "",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClaudeClient(tt.apiKey, tt.baseURL, tt.model)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewClaudeClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && client == nil {
				t.Error("NewClaudeClient() returned nil client")
			}
			if !tt.wantErr {
				if client.apiKey == "" {
					t.Error("Client API key is empty")
				}
				if client.baseURL == "" {
					t.Error("Client baseURL is empty")
				}
				if client.model == "" {
					t.Error("Client model is empty")
				}
			}
		})
	}
}

// TestScrubSecrets tests secret redaction
func TestScrubSecrets(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		mustNOTContain []string // Patterns that should be redacted
		mustContain    []string // Patterns that should remain
	}{
		{
			name:           "OpenAI API key in quotes",
			input:          `API_KEY="sk-abc123def456ghi789jkl012mno345pqr678stu901vwx234"`,
			mustNOTContain: []string{"sk-abc123def456"},
			mustContain:    []string{"[REDACTED:API_KEY]"},
		},
		{
			name:           "Anthropic API key as env var",
			input:          `ANTHROPIC_API_KEY=sk-ant-api03-aBcDeFgHiJkLmNoPqRsTuVwXyZ1234567890aBcDeFgHiJkLmNoPqRsTuVwXyZ1234567890aBcDeFgHiJkL`,
			mustNOTContain: []string{"sk-ant-api03"},
			mustContain:    []string{"[REDACTED"}, // Accept any redaction label
		},
		{
			name:           "Bearer token",
			input:          `Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9`,
			mustNOTContain: []string{"eyJhbGciOiJIUzI1NiIs"},
			mustContain:    []string{"[REDACTED:BEARER_TOKEN]"},
		},
		{
			name:           "Password in config",
			input:          `password="my-secret-password-123"`,
			mustNOTContain: []string{"my-secret-password-123"},
			mustContain:    []string{"[REDACTED:PASSWORD]"},
		},
		{
			name:        "API key in JSON (no colon/equals, won't match)",
			input:       `{"api_key": "sk-proj-abc123xyz"}`,
			mustContain: []string{"sk-proj-abc123xyz"}, // This won't be redacted due to JSON format
		},
		{
			name:        "No secrets",
			input:       `This is normal text with no secrets. It should pass through unchanged.`,
			mustContain: []string{"This is normal text"},
		},
		{
			name:           "Multiple secrets",
			input:          `OPENAI_API_KEY=sk-abc123 and ANTHROPIC_API_KEY=sk-ant-def456abc`,
			mustNOTContain: []string{"sk-abc123", "sk-ant-def456"},
			mustContain:    []string{"[REDACTED"}, // Both should be redacted
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := scrubSecrets(tt.input)

			// Check that secrets are redacted
			for _, pattern := range tt.mustNOTContain {
				if strings.Contains(result, pattern) {
					t.Errorf("Secret not redacted: found %q in result: %s", pattern, result)
				}
			}

			// Check that expected patterns remain
			for _, pattern := range tt.mustContain {
				if !strings.Contains(result, pattern) {
					t.Errorf("Expected pattern not found: %q in result: %s", pattern, result)
				}
			}
		})
	}
}

// TestMockClaudeClient tests the mock client
func TestMockClaudeClient(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		targetRatio float64
		wantErr     bool
	}{
		{
			name: "simple compression",
			content: `Line 1
Line 2
Line 3
Line 4`,
			targetRatio: 2.0,
			wantErr:     false,
		},
		{
			name:        "single line",
			content:     "Single line content",
			targetRatio: 2.0,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockClaudeClient{}
			ctx := context.Background()

			result, err := mock.Summarize(ctx, tt.content, tt.targetRatio)
			if (err != nil) != tt.wantErr {
				t.Errorf("Summarize() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if result == "" {
					t.Error("Summarize() returned empty string")
				}
				if len(result) > len(tt.content) {
					t.Errorf("Summarize() result longer than input: %d > %d", len(result), len(tt.content))
				}
			}
		})
	}
}
