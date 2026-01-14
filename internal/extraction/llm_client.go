package extraction

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/fyrsmithlabs/contextd/internal/reasoningbank"
	"golang.org/x/time/rate"
)

// anthropicLLMClient implements reasoningbank.LLMClient using Anthropic's Claude API.
// This client is used for memory consolidation in the reasoning bank, providing
// LLM-powered synthesis of similar memories into consolidated knowledge.
type anthropicLLMClient struct {
	model      string
	apiKey     string `json:"-"` // Never serialize API keys
	baseURL    string
	httpClient *http.Client
	limiter    *rate.Limiter
	maxRetries int
}

// newAnthropicLLMClient creates a new Anthropic LLM client for memory consolidation.
func newAnthropicLLMClient(cfg Config) (reasoningbank.LLMClient, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("anthropic API key required")
	}

	model := cfg.Model
	if model == "" {
		model = defaultAnthropicModel
	}

	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = defaultAnthropicBaseURL
	}

	timeout := defaultTimeout
	if cfg.Timeout > 0 {
		timeout = time.Duration(cfg.Timeout) * time.Second
	}

	return &anthropicLLMClient{
		model:   model,
		apiKey:  cfg.APIKey,
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		limiter:    rate.NewLimiter(rate.Limit(defaultRateLimit), defaultBurst),
		maxRetries: defaultMaxRetries,
	}, nil
}

// Complete generates a completion from the given prompt.
//
// This method implements the reasoningbank.LLMClient interface. It sends the
// prompt to the Anthropic Claude API and returns the generated text response.
//
// The method handles:
//   - Rate limiting to avoid API quota issues
//   - Context cancellation and deadlines
//   - Retries with exponential backoff for transient errors
//   - Error handling for various API failure modes
//
// Returns the generated text or an error if the request fails.
func (a *anthropicLLMClient) Complete(ctx context.Context, prompt string) (string, error) {
	// Wait for rate limiter
	if err := a.limiter.Wait(ctx); err != nil {
		return "", fmt.Errorf("rate limiter error: %w", err)
	}

	// Build request
	req := anthropicRequest{
		Model:       a.model,
		MaxTokens:   4096, // Higher token limit for memory consolidation
		Temperature: 0.3,  // Low temperature for consistent, factual outputs
		Messages: []anthropicMessage{
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}

	// Make request with retries
	var lastErr error
	for attempt := 0; attempt <= a.maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff
			backoff := defaultBaseBackoff * time.Duration(1<<(attempt-1))
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return "", ctx.Err()
			}
		}

		response, err := a.doRequest(ctx, req)
		if err == nil {
			return response, nil
		}

		lastErr = err
		// Check if error is retryable
		if !isRetryableError(err) {
			return "", err
		}
	}

	return "", fmt.Errorf("max retries exceeded: %w", lastErr)
}

// doRequest performs the actual HTTP request to the Claude API.
func (a *anthropicLLMClient) doRequest(ctx context.Context, req anthropicRequest) (string, error) {
	jsonData, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", a.baseURL+"/v1/messages", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-API-Key", a.apiKey)
	httpReq.Header.Set("Anthropic-Version", "2023-06-01")

	resp, err := a.httpClient.Do(httpReq)
	if err != nil {
		return "", &retryableError{err: fmt.Errorf("API request failed: %w", err)}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	// Handle rate limiting
	if resp.StatusCode == http.StatusTooManyRequests {
		return "", &retryableError{err: fmt.Errorf("rate limited (429)")}
	}

	// Handle server errors (retryable)
	if resp.StatusCode >= 500 {
		return "", &retryableError{err: fmt.Errorf("server error (%d): %s", resp.StatusCode, string(body))}
	}

	if resp.StatusCode != http.StatusOK {
		var errResp anthropicError
		if err := json.Unmarshal(body, &errResp); err == nil {
			return "", fmt.Errorf("API error (%d): %s", resp.StatusCode, errResp.Error.Message)
		}
		return "", fmt.Errorf("API error (%d): %s", resp.StatusCode, string(body))
	}

	var claudeResp anthropicResponse
	if err := json.Unmarshal(body, &claudeResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if len(claudeResp.Content) == 0 {
		return "", fmt.Errorf("empty response from API")
	}

	return claudeResp.Content[0].Text, nil
}

// openAILLMClient implements reasoningbank.LLMClient using OpenAI's API.
// This client is used for memory consolidation in the reasoning bank, providing
// LLM-powered synthesis of similar memories into consolidated knowledge.
type openAILLMClient struct {
	model      string
	apiKey     string `json:"-"` // Never serialize API keys
	baseURL    string
	httpClient *http.Client
	limiter    *rate.Limiter
	maxRetries int
}

// newOpenAILLMClient creates a new OpenAI LLM client for memory consolidation.
func newOpenAILLMClient(cfg Config) (reasoningbank.LLMClient, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("openai API key required")
	}

	model := cfg.Model
	if model == "" {
		model = defaultOpenAIModel
	}

	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = defaultOpenAIBaseURL
	}

	timeout := defaultTimeout
	if cfg.Timeout > 0 {
		timeout = time.Duration(cfg.Timeout) * time.Second
	}

	return &openAILLMClient{
		model:   model,
		apiKey:  cfg.APIKey,
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		limiter:    rate.NewLimiter(rate.Limit(defaultRateLimit), defaultBurst),
		maxRetries: defaultMaxRetries,
	}, nil
}

// Complete generates a completion from the given prompt.
//
// This method implements the reasoningbank.LLMClient interface. It sends the
// prompt to the OpenAI API and returns the generated text response.
//
// The method handles:
//   - Rate limiting to avoid API quota issues
//   - Context cancellation and deadlines
//   - Retries with exponential backoff for transient errors
//   - Error handling for various API failure modes
//
// Returns the generated text or an error if the request fails.
func (o *openAILLMClient) Complete(ctx context.Context, prompt string) (string, error) {
	// Wait for rate limiter
	if err := o.limiter.Wait(ctx); err != nil {
		return "", fmt.Errorf("rate limiter error: %w", err)
	}

	// Build request
	req := openAIRequest{
		Model:       o.model,
		MaxTokens:   4096, // Higher token limit for memory consolidation
		Temperature: 0.3,  // Low temperature for consistent, factual outputs
		Messages: []openAIMessage{
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}

	// Make request with retries
	var lastErr error
	for attempt := 0; attempt <= o.maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff
			backoff := defaultBaseBackoff * time.Duration(1<<(attempt-1))
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return "", ctx.Err()
			}
		}

		response, err := o.doRequest(ctx, req)
		if err == nil {
			return response, nil
		}

		lastErr = err
		// Check if error is retryable
		if !isRetryableError(err) {
			return "", err
		}
	}

	return "", fmt.Errorf("max retries exceeded: %w", lastErr)
}

// doRequest performs the actual HTTP request to the OpenAI API.
func (o *openAILLMClient) doRequest(ctx context.Context, req openAIRequest) (string, error) {
	jsonData, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", o.baseURL+"/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+o.apiKey)

	resp, err := o.httpClient.Do(httpReq)
	if err != nil {
		return "", &retryableError{err: fmt.Errorf("API request failed: %w", err)}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	// Handle rate limiting
	if resp.StatusCode == http.StatusTooManyRequests {
		return "", &retryableError{err: fmt.Errorf("rate limited (429)")}
	}

	// Handle server errors (retryable)
	if resp.StatusCode >= 500 {
		return "", &retryableError{err: fmt.Errorf("server error (%d): %s", resp.StatusCode, string(body))}
	}

	if resp.StatusCode != http.StatusOK {
		var errResp openAIError
		if err := json.Unmarshal(body, &errResp); err == nil {
			return "", fmt.Errorf("API error (%d): %s", resp.StatusCode, errResp.Error.Message)
		}
		return "", fmt.Errorf("API error (%d): %s", resp.StatusCode, string(body))
	}

	var openAIResp openAIResponse
	if err := json.Unmarshal(body, &openAIResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if len(openAIResp.Choices) == 0 {
		return "", fmt.Errorf("empty response from API")
	}

	return openAIResp.Choices[0].Message.Content, nil
}

// Ensure interfaces are implemented at compile time.
var _ reasoningbank.LLMClient = (*anthropicLLMClient)(nil)
var _ reasoningbank.LLMClient = (*openAILLMClient)(nil)
