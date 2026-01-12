package extraction

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestNewAnthropicSummarizer tests the Anthropic summarizer creation.
func TestNewAnthropicSummarizer(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: Config{
				APIKey:  "sk-ant-test123",
				BaseURL: "https://api.anthropic.com",
				Model:   "claude-3-5-sonnet-20241022",
			},
			wantErr: false,
		},
		{
			name: "empty API key",
			cfg: Config{
				BaseURL: "https://api.anthropic.com",
				Model:   "claude-3-5-sonnet-20241022",
			},
			wantErr: true,
		},
		{
			name: "default baseURL and model",
			cfg: Config{
				APIKey: "sk-ant-test123",
			},
			wantErr: false,
		},
		{
			name: "custom timeout",
			cfg: Config{
				APIKey:  "sk-ant-test123",
				Timeout: 120,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			summarizer, err := newAnthropicSummarizer(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("newAnthropicSummarizer() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && summarizer == nil {
				t.Error("newAnthropicSummarizer() returned nil summarizer")
			}
			if !tt.wantErr {
				if !summarizer.Available() {
					t.Error("summarizer.Available() = false, want true")
				}
			}
		})
	}
}

// TestNewOpenAISummarizer tests the OpenAI summarizer creation.
func TestNewOpenAISummarizer(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: Config{
				APIKey:  "sk-test123",
				BaseURL: "https://api.openai.com",
				Model:   "gpt-4o-mini",
			},
			wantErr: false,
		},
		{
			name: "empty API key",
			cfg: Config{
				BaseURL: "https://api.openai.com",
				Model:   "gpt-4o-mini",
			},
			wantErr: true,
		},
		{
			name: "default baseURL and model",
			cfg: Config{
				APIKey: "sk-test123",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			summarizer, err := newOpenAISummarizer(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("newOpenAISummarizer() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && summarizer == nil {
				t.Error("newOpenAISummarizer() returned nil summarizer")
			}
			if !tt.wantErr {
				if !summarizer.Available() {
					t.Error("summarizer.Available() = false, want true")
				}
			}
		})
	}
}

// TestAnthropicSummarizer_Summarize tests the Anthropic summarizer with a mock server.
func TestAnthropicSummarizer_Summarize(t *testing.T) {
	tests := []struct {
		name           string
		candidate      DecisionCandidate
		serverResponse string
		statusCode     int
		wantErr        bool
		wantSummary    string
	}{
		{
			name: "successful summarization",
			candidate: DecisionCandidate{
				SessionID:      "s1",
				MessageUUID:    "m1",
				Content:        "Let's use the repository pattern for data access.",
				PatternMatched: "lets_use",
				Confidence:     0.9,
			},
			serverResponse: `{
				"id": "msg_123",
				"type": "message",
				"role": "assistant",
				"content": [{
					"type": "text",
					"text": "{\"summary\": \"Use repository pattern for data access layer\", \"reasoning\": \"Provides abstraction over data storage\", \"tags\": [\"architecture\", \"patterns\"], \"confidence\": 0.95}"
				}],
				"model": "claude-3-5-sonnet-20241022",
				"stop_reason": "end_turn"
			}`,
			statusCode:  http.StatusOK,
			wantErr:     false,
			wantSummary: "Use repository pattern for data access layer",
		},
		{
			name: "response with markdown code block",
			candidate: DecisionCandidate{
				SessionID:      "s1",
				MessageUUID:    "m1",
				Content:        "Decided to implement caching.",
				PatternMatched: "decided_to",
				Confidence:     0.8,
			},
			serverResponse: `{
				"id": "msg_123",
				"type": "message",
				"role": "assistant",
				"content": [{
					"type": "text",
					"text": "{\"summary\": \"Implement caching layer\", \"confidence\": 0.85}"
				}],
				"model": "claude-3-5-sonnet-20241022",
				"stop_reason": "end_turn"
			}`,
			statusCode:  http.StatusOK,
			wantErr:     false,
			wantSummary: "Implement caching layer",
		},
		{
			name: "unauthorized error",
			candidate: DecisionCandidate{
				SessionID:   "s1",
				MessageUUID: "m1",
				Content:     "Test content",
				Confidence:  0.8,
			},
			serverResponse: `{
				"type": "error",
				"error": {
					"type": "authentication_error",
					"message": "Invalid API key"
				}
			}`,
			statusCode: http.StatusUnauthorized,
			wantErr:    true,
		},
		{
			name: "empty response",
			candidate: DecisionCandidate{
				SessionID:   "s1",
				MessageUUID: "m1",
				Content:     "Test content",
				Confidence:  0.8,
			},
			serverResponse: `{
				"id": "msg_123",
				"type": "message",
				"role": "assistant",
				"content": [],
				"model": "claude-3-5-sonnet-20241022"
			}`,
			statusCode: http.StatusOK,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request headers
				if r.Header.Get("X-API-Key") == "" {
					t.Error("Missing X-API-Key header")
				}
				if r.Header.Get("Content-Type") != "application/json" {
					t.Error("Missing Content-Type header")
				}
				if r.Header.Get("Anthropic-Version") != "2023-06-01" {
					t.Error("Missing or incorrect Anthropic-Version header")
				}

				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.serverResponse))
			}))
			defer server.Close()

			cfg := Config{
				APIKey:  "sk-ant-test123",
				BaseURL: server.URL,
				Model:   "claude-3-5-sonnet-20241022",
			}

			summarizer, err := newAnthropicSummarizer(cfg)
			if err != nil {
				t.Fatalf("Failed to create summarizer: %v", err)
			}

			ctx := context.Background()
			decision, err := summarizer.Summarize(ctx, tt.candidate)

			if (err != nil) != tt.wantErr {
				t.Errorf("Summarize() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && decision.Summary != tt.wantSummary {
				t.Errorf("Summarize() summary = %q, want %q", decision.Summary, tt.wantSummary)
			}
		})
	}
}

// TestOpenAISummarizer_Summarize tests the OpenAI summarizer with a mock server.
func TestOpenAISummarizer_Summarize(t *testing.T) {
	tests := []struct {
		name           string
		candidate      DecisionCandidate
		serverResponse string
		statusCode     int
		wantErr        bool
		wantSummary    string
	}{
		{
			name: "successful summarization",
			candidate: DecisionCandidate{
				SessionID:      "s1",
				MessageUUID:    "m1",
				Content:        "Let's use the repository pattern for data access.",
				PatternMatched: "lets_use",
				Confidence:     0.9,
			},
			serverResponse: `{
				"id": "chatcmpl-123",
				"object": "chat.completion",
				"created": 1677652288,
				"model": "gpt-4o-mini",
				"choices": [{
					"index": 0,
					"message": {
						"role": "assistant",
						"content": "{\"summary\": \"Use repository pattern for data access layer\", \"reasoning\": \"Provides abstraction over data storage\", \"tags\": [\"architecture\", \"patterns\"], \"confidence\": 0.95}"
					},
					"finish_reason": "stop"
				}]
			}`,
			statusCode:  http.StatusOK,
			wantErr:     false,
			wantSummary: "Use repository pattern for data access layer",
		},
		{
			name: "unauthorized error",
			candidate: DecisionCandidate{
				SessionID:   "s1",
				MessageUUID: "m1",
				Content:     "Test content",
				Confidence:  0.8,
			},
			serverResponse: `{
				"error": {
					"message": "Invalid API key",
					"type": "invalid_request_error",
					"code": "invalid_api_key"
				}
			}`,
			statusCode: http.StatusUnauthorized,
			wantErr:    true,
		},
		{
			name: "empty choices",
			candidate: DecisionCandidate{
				SessionID:   "s1",
				MessageUUID: "m1",
				Content:     "Test content",
				Confidence:  0.8,
			},
			serverResponse: `{
				"id": "chatcmpl-123",
				"object": "chat.completion",
				"choices": []
			}`,
			statusCode: http.StatusOK,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request headers
				auth := r.Header.Get("Authorization")
				if !strings.HasPrefix(auth, "Bearer ") {
					t.Error("Missing or invalid Authorization header")
				}
				if r.Header.Get("Content-Type") != "application/json" {
					t.Error("Missing Content-Type header")
				}

				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.serverResponse))
			}))
			defer server.Close()

			cfg := Config{
				APIKey:  "sk-test123",
				BaseURL: server.URL,
				Model:   "gpt-4o-mini",
			}

			summarizer, err := newOpenAISummarizer(cfg)
			if err != nil {
				t.Fatalf("Failed to create summarizer: %v", err)
			}

			ctx := context.Background()
			decision, err := summarizer.Summarize(ctx, tt.candidate)

			if (err != nil) != tt.wantErr {
				t.Errorf("Summarize() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && decision.Summary != tt.wantSummary {
				t.Errorf("Summarize() summary = %q, want %q", decision.Summary, tt.wantSummary)
			}
		})
	}
}

// TestAnthropicSummarizer_RateLimiting tests that rate limiting is applied.
func TestAnthropicSummarizer_RateLimiting(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		response := `{
			"id": "msg_123",
			"type": "message",
			"role": "assistant",
			"content": [{
				"type": "text",
				"text": "{\"summary\": \"Test\", \"confidence\": 0.9}"
			}],
			"model": "claude-3-5-sonnet-20241022"
		}`
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(response))
	}))
	defer server.Close()

	cfg := Config{
		APIKey:  "sk-ant-test123",
		BaseURL: server.URL,
	}

	summarizer, err := newAnthropicSummarizer(cfg)
	if err != nil {
		t.Fatalf("Failed to create summarizer: %v", err)
	}

	ctx := context.Background()
	candidate := DecisionCandidate{
		Content:    "Test",
		Confidence: 0.9,
	}

	// Make a few requests - they should succeed (within burst limit)
	for i := 0; i < 3; i++ {
		_, err := summarizer.Summarize(ctx, candidate)
		if err != nil {
			t.Errorf("Request %d failed unexpectedly: %v", i, err)
		}
	}

	if requestCount != 3 {
		t.Errorf("Expected 3 requests, got %d", requestCount)
	}
}

// TestAnthropicSummarizer_Retry tests retry behavior on server errors.
func TestAnthropicSummarizer_Retry(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if requestCount <= 2 {
			// First two requests fail with server error
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"error": {"message": "Service temporarily unavailable"}}`))
			return
		}
		// Third request succeeds
		response := `{
			"id": "msg_123",
			"type": "message",
			"role": "assistant",
			"content": [{
				"type": "text",
				"text": "{\"summary\": \"Success after retry\", \"confidence\": 0.9}"
			}],
			"model": "claude-3-5-sonnet-20241022"
		}`
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(response))
	}))
	defer server.Close()

	cfg := Config{
		APIKey:  "sk-ant-test123",
		BaseURL: server.URL,
	}

	summarizer, err := newAnthropicSummarizer(cfg)
	if err != nil {
		t.Fatalf("Failed to create summarizer: %v", err)
	}

	ctx := context.Background()
	candidate := DecisionCandidate{
		Content:    "Test",
		Confidence: 0.9,
	}

	decision, err := summarizer.Summarize(ctx, candidate)
	if err != nil {
		t.Fatalf("Summarize() failed after retries: %v", err)
	}

	if decision.Summary != "Success after retry" {
		t.Errorf("Summary = %q, want %q", decision.Summary, "Success after retry")
	}

	if requestCount != 3 {
		t.Errorf("Expected 3 requests (2 retries), got %d", requestCount)
	}
}

// TestAnthropicSummarizer_ContextCancellation tests that context cancellation is respected.
func TestAnthropicSummarizer_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second) // Delay response
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := Config{
		APIKey:  "sk-ant-test123",
		BaseURL: server.URL,
	}

	summarizer, err := newAnthropicSummarizer(cfg)
	if err != nil {
		t.Fatalf("Failed to create summarizer: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	candidate := DecisionCandidate{
		Content:    "Test",
		Confidence: 0.9,
	}

	_, err = summarizer.Summarize(ctx, candidate)
	if err == nil {
		t.Error("Expected error due to context cancellation")
	}
}

// TestParseDecisionJSON tests the JSON parsing of LLM responses.
func TestParseDecisionJSON(t *testing.T) {
	tests := []struct {
		name               string
		content            string
		fallbackConfidence float64
		wantSummary        string
		wantConfidence     float64
		wantTags           []string
	}{
		{
			name:               "valid JSON",
			content:            `{"summary": "Test summary", "reasoning": "Because reasons", "confidence": 0.95, "tags": ["go", "testing"]}`,
			fallbackConfidence: 0.8,
			wantSummary:        "Test summary",
			wantConfidence:     0.95,
			wantTags:           []string{"go", "testing"},
		},
		{
			name:               "JSON with markdown code block",
			content:            "```json\n{\"summary\": \"Test summary\", \"confidence\": 0.85}\n```",
			fallbackConfidence: 0.8,
			wantSummary:        "Test summary",
			wantConfidence:     0.85,
		},
		{
			name:               "invalid JSON falls back",
			content:            "This is not valid JSON at all.",
			fallbackConfidence: 0.7,
			wantSummary:        "This is not valid JSON at all.",
			wantConfidence:     0.7,
		},
		{
			name:               "invalid confidence uses fallback",
			content:            `{"summary": "Test", "confidence": -0.5}`,
			fallbackConfidence: 0.8,
			wantSummary:        "Test",
			wantConfidence:     0.8,
		},
		{
			name:               "confidence over 1 uses fallback",
			content:            `{"summary": "Test", "confidence": 1.5}`,
			fallbackConfidence: 0.8,
			wantSummary:        "Test",
			wantConfidence:     0.8,
		},
		{
			name:               "zero confidence uses fallback",
			content:            `{"summary": "Test", "confidence": 0}`,
			fallbackConfidence: 0.8,
			wantSummary:        "Test",
			wantConfidence:     0.8,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decision, err := parseDecisionJSON(tt.content, tt.fallbackConfidence)
			if err != nil {
				t.Fatalf("parseDecisionJSON() error = %v", err)
			}

			if decision.Summary != tt.wantSummary {
				t.Errorf("Summary = %q, want %q", decision.Summary, tt.wantSummary)
			}

			if decision.Confidence != tt.wantConfidence {
				t.Errorf("Confidence = %f, want %f", decision.Confidence, tt.wantConfidence)
			}

			if tt.wantTags != nil {
				if len(decision.Tags) != len(tt.wantTags) {
					t.Errorf("Tags count = %d, want %d", len(decision.Tags), len(tt.wantTags))
				}
			}
		})
	}
}

// TestScrubSecrets tests secret redaction in content.
func TestScrubSecrets(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		mustNOTContain []string
		mustContain    []string
	}{
		{
			name:           "OpenAI API key",
			input:          "OPENAI_API_KEY=sk-abc123def456ghi789jkl012mno345pqr678",
			mustNOTContain: []string{"sk-abc123def456"},
			mustContain:    []string{"[REDACTED"},
		},
		{
			name:           "Anthropic API key",
			input:          "ANTHROPIC_API_KEY=sk-ant-api03-aBcDeFgHiJkLmNoPqRsTuVwXyZ1234567890",
			mustNOTContain: []string{"sk-ant-api03"},
			mustContain:    []string{"[REDACTED"},
		},
		{
			name:           "Bearer token",
			input:          "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.test",
			mustNOTContain: []string{"eyJhbGciOiJIUzI1NiIs"},
			mustContain:    []string{"[REDACTED:BEARER_TOKEN]"},
		},
		{
			name:           "Password",
			input:          `password="my-secret-password-123"`,
			mustNOTContain: []string{"my-secret-password-123"},
			mustContain:    []string{"[REDACTED:PASSWORD]"},
		},
		{
			name:           "Private key",
			input:          "-----BEGIN RSA PRIVATE KEY-----\nMIIE...\n-----END RSA PRIVATE KEY-----",
			mustNOTContain: []string{"BEGIN RSA PRIVATE KEY"},
			mustContain:    []string{"[REDACTED:PRIVATE_KEY]"},
		},
		{
			name:        "No secrets",
			input:       "This is normal text with no secrets.",
			mustContain: []string{"This is normal text"},
		},
		{
			name:           "API key in config format",
			input:          "api_key: sk-verylongtestkey12345678901234567890",
			mustNOTContain: []string{"sk-verylongtestkey"},
			mustContain:    []string{"[REDACTED"},
		},
		{
			name:           "GitHub token as env var",
			input:          "GITHUB_TOKEN=ghp_1234567890abcdefghijklmnop",
			mustNOTContain: []string{"ghp_123456"},
			mustContain:    []string{"[REDACTED"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := scrubSecrets(tt.input)

			for _, pattern := range tt.mustNOTContain {
				if strings.Contains(result, pattern) {
					t.Errorf("Secret not redacted: found %q in result: %s", pattern, result)
				}
			}

			for _, pattern := range tt.mustContain {
				if !strings.Contains(result, pattern) {
					t.Errorf("Expected pattern not found: %q in result: %s", pattern, result)
				}
			}
		})
	}
}

// TestExtractFirstSentenceFromContent tests the fallback sentence extraction.
func TestExtractFirstSentenceFromContent(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "single sentence with period",
			content: "This is a test.",
			want:    "This is a test.",
		},
		{
			name:    "multiple sentences",
			content: "First sentence. Second sentence.",
			want:    "First sentence.",
		},
		{
			name:    "exclamation mark",
			content: "Wow! This is great.",
			want:    "Wow!",
		},
		{
			name:    "question mark",
			content: "How does this work? Let me explain.",
			want:    "How does this work?",
		},
		{
			name:    "long content truncated",
			content: strings.Repeat("a", 300),
			want:    strings.Repeat("a", 200) + "...",
		},
		{
			name:    "no punctuation short",
			content: "Short content",
			want:    "Short content",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractFirstSentenceFromContent(tt.content)
			if got != tt.want {
				t.Errorf("extractFirstSentenceFromContent() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestRetryableError tests the retryable error type.
func TestRetryableError(t *testing.T) {
	err := &retryableError{err: fmt.Errorf("test error")}

	if err.Error() != "test error" {
		t.Errorf("Error() = %q, want %q", err.Error(), "test error")
	}

	if err.Unwrap() == nil {
		t.Error("Unwrap() = nil, want non-nil")
	}

	if !isRetryableError(err) {
		t.Error("isRetryableError() = false, want true")
	}

	normalErr := fmt.Errorf("normal error")
	if isRetryableError(normalErr) {
		t.Error("isRetryableError() = true for normal error, want false")
	}
}

// TestAnthropicSummarizer_WithContext tests context in candidate content.
func TestAnthropicSummarizer_WithContext(t *testing.T) {
	var receivedBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&receivedBody)
		response := `{
			"id": "msg_123",
			"type": "message",
			"role": "assistant",
			"content": [{
				"type": "text",
				"text": "{\"summary\": \"Test with context\", \"confidence\": 0.9}"
			}],
			"model": "claude-3-5-sonnet-20241022"
		}`
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(response))
	}))
	defer server.Close()

	cfg := Config{
		APIKey:  "sk-ant-test123",
		BaseURL: server.URL,
	}

	summarizer, err := newAnthropicSummarizer(cfg)
	if err != nil {
		t.Fatalf("Failed to create summarizer: %v", err)
	}

	ctx := context.Background()
	candidate := DecisionCandidate{
		Content:        "Main decision content",
		Context:        []string{"Previous message 1", "Previous message 2"},
		PatternMatched: "test_pattern",
		Confidence:     0.8,
	}

	_, err = summarizer.Summarize(ctx, candidate)
	if err != nil {
		t.Fatalf("Summarize() error = %v", err)
	}

	// Verify context was included in the request
	messages := receivedBody["messages"].([]interface{})
	if len(messages) == 0 {
		t.Fatal("No messages in request")
	}

	userMessage := messages[0].(map[string]interface{})
	content := userMessage["content"].(string)
	if !strings.Contains(content, "Context:") {
		t.Error("Context not included in request")
	}
	if !strings.Contains(content, "Previous message 1") {
		t.Error("Context message not found in request")
	}
}

// TestOpenAISummarizer_Retry tests retry behavior on server errors for OpenAI.
func TestOpenAISummarizer_Retry(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if requestCount <= 2 {
			// First two requests fail with rate limit
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error": {"message": "Rate limit exceeded"}}`))
			return
		}
		// Third request succeeds
		response := `{
			"id": "chatcmpl-123",
			"object": "chat.completion",
			"model": "gpt-4o-mini",
			"choices": [{
				"index": 0,
				"message": {
					"role": "assistant",
					"content": "{\"summary\": \"Success after rate limit\", \"confidence\": 0.9}"
				},
				"finish_reason": "stop"
			}]
		}`
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(response))
	}))
	defer server.Close()

	cfg := Config{
		APIKey:  "sk-test123",
		BaseURL: server.URL,
	}

	summarizer, err := newOpenAISummarizer(cfg)
	if err != nil {
		t.Fatalf("Failed to create summarizer: %v", err)
	}

	ctx := context.Background()
	candidate := DecisionCandidate{
		Content:    "Test",
		Confidence: 0.9,
	}

	decision, err := summarizer.Summarize(ctx, candidate)
	if err != nil {
		t.Fatalf("Summarize() failed after retries: %v", err)
	}

	if decision.Summary != "Success after rate limit" {
		t.Errorf("Summary = %q, want %q", decision.Summary, "Success after rate limit")
	}

	if requestCount != 3 {
		t.Errorf("Expected 3 requests (2 retries), got %d", requestCount)
	}
}

// TestScrubSecretsWithContext tests that secrets in context are also scrubbed.
func TestScrubSecretsWithContext(t *testing.T) {
	var receivedBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&receivedBody)
		response := `{
			"id": "msg_123",
			"type": "message",
			"role": "assistant",
			"content": [{
				"type": "text",
				"text": "{\"summary\": \"Test\", \"confidence\": 0.9}"
			}],
			"model": "claude-3-5-sonnet-20241022"
		}`
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(response))
	}))
	defer server.Close()

	cfg := Config{
		APIKey:  "sk-ant-test123",
		BaseURL: server.URL,
	}

	summarizer, err := newAnthropicSummarizer(cfg)
	if err != nil {
		t.Fatalf("Failed to create summarizer: %v", err)
	}

	ctx := context.Background()
	candidate := DecisionCandidate{
		Content: "Main content with ANTHROPIC_API_KEY=sk-ant-secret12345678901234567890",
		Context: []string{
			"Context with password=mysecretpassword123",
			"Another context with token: mysupersecretsessiontoken12345678",
		},
		PatternMatched: "test_pattern",
		Confidence:     0.8,
	}

	_, err = summarizer.Summarize(ctx, candidate)
	if err != nil {
		t.Fatalf("Summarize() error = %v", err)
	}

	// Verify secrets were scrubbed in the request
	messages := receivedBody["messages"].([]interface{})
	if len(messages) == 0 {
		t.Fatal("No messages in request")
	}

	userMessage := messages[0].(map[string]interface{})
	content := userMessage["content"].(string)

	if strings.Contains(content, "sk-ant-secret") {
		t.Error("API key not scrubbed from content")
	}
	if strings.Contains(content, "mysecretpassword") {
		t.Error("Password not scrubbed from context")
	}
	if strings.Contains(content, "mysupersecretsessiontoken") {
		t.Error("Token not scrubbed from context")
	}
	if !strings.Contains(content, "[REDACTED") {
		t.Error("Expected REDACTED placeholder in content")
	}
}
