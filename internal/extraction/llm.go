package extraction

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"golang.org/x/time/rate"
)

// Default configuration values.
const (
	defaultAnthropicBaseURL = "https://api.anthropic.com"
	defaultAnthropicModel   = "claude-3-5-sonnet-20241022"
	defaultOpenAIBaseURL    = "https://api.openai.com"
	defaultOpenAIModel      = "gpt-4o-mini"
	defaultMaxTokens        = 1024
	defaultTimeout          = 60 * time.Second
	defaultMaxRetries       = 3
	defaultBaseBackoff      = 1 * time.Second
)

// Rate limiter defaults: 50 requests per minute for both APIs.
const (
	defaultRateLimit = 50.0 / 60.0 // ~0.83 requests per second
	defaultBurst     = 5           // Allow bursts of up to 5 requests
)

// anthropicSummarizer implements Summarizer using Anthropic's Claude API.
type anthropicSummarizer struct {
	model      string
	apiKey     string `json:"-"` // Never serialize API keys
	baseURL    string
	httpClient *http.Client
	limiter    *rate.Limiter
	maxRetries int
}

// newAnthropicSummarizer creates a new Anthropic summarizer.
func newAnthropicSummarizer(cfg Config) (Summarizer, error) {
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

	return &anthropicSummarizer{
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

// anthropicRequest represents the request format for Claude API.
type anthropicRequest struct {
	Model       string             `json:"model"`
	MaxTokens   int                `json:"max_tokens"`
	Messages    []anthropicMessage `json:"messages"`
	System      string             `json:"system,omitempty"`
	Temperature float64            `json:"temperature"`
}

// anthropicMessage represents a message in the Claude conversation.
type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// anthropicResponse represents the response from Claude API.
type anthropicResponse struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Role    string `json:"role"`
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Model      string `json:"model"`
	StopReason string `json:"stop_reason"`
	Usage      struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

// anthropicError represents an error response from Claude API.
type anthropicError struct {
	Type  string `json:"type"`
	Error struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

// summarizePrompt is the system prompt for decision summarization.
const summarizePrompt = `You are an expert at analyzing and summarizing decisions made in software development conversations.

Your task is to extract and refine a decision from the provided content. The decision should be:
1. Clear and actionable
2. Free of unnecessary context
3. Focused on the "what" and "why" of the decision

Respond with a JSON object containing:
- "summary": A clear, concise summary of the decision (1-2 sentences)
- "reasoning": Why this decision was made (optional, if evident from context)
- "alternatives": Any alternatives that were considered and rejected (optional, as array)
- "tags": Relevant tags for categorization (optional, as array of strings like "architecture", "testing", "performance", etc.)
- "confidence": Your confidence in this being a significant decision (0.0 to 1.0)

Respond ONLY with the JSON object, no additional text.`

// Summarize refines a decision candidate using Claude.
func (a *anthropicSummarizer) Summarize(ctx context.Context, candidate DecisionCandidate) (Decision, error) {
	// Wait for rate limiter
	if err := a.limiter.Wait(ctx); err != nil {
		return Decision{}, fmt.Errorf("rate limiter error: %w", err)
	}

	// Scrub secrets from content before sending to API
	scrubbedContent := scrubSecrets(candidate.Content)
	contextStr := ""
	if len(candidate.Context) > 0 {
		scrubbedContext := make([]string, len(candidate.Context))
		for i, c := range candidate.Context {
			scrubbedContext[i] = scrubSecrets(c)
		}
		contextStr = "\n\nContext:\n" + strings.Join(scrubbedContext, "\n---\n")
	}

	userContent := fmt.Sprintf("Pattern matched: %s\nConfidence: %.2f\n\nDecision content:\n%s%s",
		candidate.PatternMatched, candidate.Confidence, scrubbedContent, contextStr)

	maxTokens := defaultMaxTokens

	req := anthropicRequest{
		Model:       a.model,
		MaxTokens:   maxTokens,
		Temperature: 0.3, // Low temperature for consistent extraction
		System:      summarizePrompt,
		Messages: []anthropicMessage{
			{
				Role:    "user",
				Content: userContent,
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
				return Decision{}, ctx.Err()
			}
		}

		decision, err := a.doRequest(ctx, req, candidate.Confidence)
		if err == nil {
			return decision, nil
		}

		lastErr = err
		// Check if error is retryable
		if !isRetryableError(err) {
			return Decision{}, err
		}
	}

	return Decision{}, fmt.Errorf("max retries exceeded: %w", lastErr)
}

// doRequest performs the actual HTTP request to the Claude API.
func (a *anthropicSummarizer) doRequest(ctx context.Context, req anthropicRequest, fallbackConfidence float64) (Decision, error) {
	jsonData, err := json.Marshal(req)
	if err != nil {
		return Decision{}, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", a.baseURL+"/v1/messages", bytes.NewBuffer(jsonData))
	if err != nil {
		return Decision{}, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-API-Key", a.apiKey)
	httpReq.Header.Set("Anthropic-Version", "2023-06-01")

	resp, err := a.httpClient.Do(httpReq)
	if err != nil {
		return Decision{}, &retryableError{err: fmt.Errorf("API request failed: %w", err)}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Decision{}, fmt.Errorf("failed to read response: %w", err)
	}

	// Handle rate limiting
	if resp.StatusCode == http.StatusTooManyRequests {
		return Decision{}, &retryableError{err: fmt.Errorf("rate limited (429)")}
	}

	// Handle server errors (retryable)
	if resp.StatusCode >= 500 {
		return Decision{}, &retryableError{err: fmt.Errorf("server error (%d): %s", resp.StatusCode, string(body))}
	}

	if resp.StatusCode != http.StatusOK {
		var errResp anthropicError
		if err := json.Unmarshal(body, &errResp); err == nil {
			return Decision{}, fmt.Errorf("API error (%d): %s", resp.StatusCode, errResp.Error.Message)
		}
		return Decision{}, fmt.Errorf("API error (%d): %s", resp.StatusCode, string(body))
	}

	var claudeResp anthropicResponse
	if err := json.Unmarshal(body, &claudeResp); err != nil {
		return Decision{}, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(claudeResp.Content) == 0 {
		return Decision{}, fmt.Errorf("empty response from API")
	}

	return parseDecisionJSON(claudeResp.Content[0].Text, fallbackConfidence)
}

// Available returns true if the summarizer is configured.
func (a *anthropicSummarizer) Available() bool {
	return a.apiKey != ""
}

// openAISummarizer implements Summarizer using OpenAI's API.
type openAISummarizer struct {
	model      string
	apiKey     string `json:"-"` // Never serialize API keys
	baseURL    string
	httpClient *http.Client
	limiter    *rate.Limiter
	maxRetries int
}

// newOpenAISummarizer creates a new OpenAI summarizer.
func newOpenAISummarizer(cfg Config) (Summarizer, error) {
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

	return &openAISummarizer{
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

// openAIRequest represents the request format for OpenAI Chat API.
type openAIRequest struct {
	Model       string          `json:"model"`
	Messages    []openAIMessage `json:"messages"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	Temperature float64         `json:"temperature"`
}

// openAIMessage represents a message in the OpenAI conversation.
type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// openAIResponse represents the response from OpenAI Chat API.
type openAIResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// openAIError represents an error response from OpenAI API.
type openAIError struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Param   string `json:"param"`
		Code    string `json:"code"`
	} `json:"error"`
}

// Summarize refines a decision candidate using GPT.
func (o *openAISummarizer) Summarize(ctx context.Context, candidate DecisionCandidate) (Decision, error) {
	// Wait for rate limiter
	if err := o.limiter.Wait(ctx); err != nil {
		return Decision{}, fmt.Errorf("rate limiter error: %w", err)
	}

	// Scrub secrets from content before sending to API
	scrubbedContent := scrubSecrets(candidate.Content)
	contextStr := ""
	if len(candidate.Context) > 0 {
		scrubbedContext := make([]string, len(candidate.Context))
		for i, c := range candidate.Context {
			scrubbedContext[i] = scrubSecrets(c)
		}
		contextStr = "\n\nContext:\n" + strings.Join(scrubbedContext, "\n---\n")
	}

	userContent := fmt.Sprintf("Pattern matched: %s\nConfidence: %.2f\n\nDecision content:\n%s%s",
		candidate.PatternMatched, candidate.Confidence, scrubbedContent, contextStr)

	maxTokens := defaultMaxTokens

	req := openAIRequest{
		Model:       o.model,
		MaxTokens:   maxTokens,
		Temperature: 0.3, // Low temperature for consistent extraction
		Messages: []openAIMessage{
			{
				Role:    "system",
				Content: summarizePrompt,
			},
			{
				Role:    "user",
				Content: userContent,
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
				return Decision{}, ctx.Err()
			}
		}

		decision, err := o.doRequest(ctx, req, candidate.Confidence)
		if err == nil {
			return decision, nil
		}

		lastErr = err
		// Check if error is retryable
		if !isRetryableError(err) {
			return Decision{}, err
		}
	}

	return Decision{}, fmt.Errorf("max retries exceeded: %w", lastErr)
}

// doRequest performs the actual HTTP request to the OpenAI API.
func (o *openAISummarizer) doRequest(ctx context.Context, req openAIRequest, fallbackConfidence float64) (Decision, error) {
	jsonData, err := json.Marshal(req)
	if err != nil {
		return Decision{}, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", o.baseURL+"/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return Decision{}, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+o.apiKey)

	resp, err := o.httpClient.Do(httpReq)
	if err != nil {
		return Decision{}, &retryableError{err: fmt.Errorf("API request failed: %w", err)}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Decision{}, fmt.Errorf("failed to read response: %w", err)
	}

	// Handle rate limiting
	if resp.StatusCode == http.StatusTooManyRequests {
		return Decision{}, &retryableError{err: fmt.Errorf("rate limited (429)")}
	}

	// Handle server errors (retryable)
	if resp.StatusCode >= 500 {
		return Decision{}, &retryableError{err: fmt.Errorf("server error (%d): %s", resp.StatusCode, string(body))}
	}

	if resp.StatusCode != http.StatusOK {
		var errResp openAIError
		if err := json.Unmarshal(body, &errResp); err == nil {
			return Decision{}, fmt.Errorf("API error (%d): %s", resp.StatusCode, errResp.Error.Message)
		}
		return Decision{}, fmt.Errorf("API error (%d): %s", resp.StatusCode, string(body))
	}

	var openAIResp openAIResponse
	if err := json.Unmarshal(body, &openAIResp); err != nil {
		return Decision{}, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(openAIResp.Choices) == 0 {
		return Decision{}, fmt.Errorf("empty response from API")
	}

	return parseDecisionJSON(openAIResp.Choices[0].Message.Content, fallbackConfidence)
}

// Available returns true if the summarizer is configured.
func (o *openAISummarizer) Available() bool {
	return o.apiKey != ""
}

// decisionResponse represents the expected JSON response from LLMs.
type decisionResponse struct {
	Summary      string   `json:"summary"`
	Reasoning    string   `json:"reasoning,omitempty"`
	Alternatives []string `json:"alternatives,omitempty"`
	Tags         []string `json:"tags,omitempty"`
	Confidence   float64  `json:"confidence"`
}

// parseDecisionJSON parses the LLM response into a Decision.
func parseDecisionJSON(content string, fallbackConfidence float64) (Decision, error) {
	// Clean up the response - sometimes LLMs wrap JSON in markdown code blocks
	content = strings.TrimSpace(content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	var resp decisionResponse
	if err := json.Unmarshal([]byte(content), &resp); err != nil {
		// If JSON parsing fails, return a basic decision with the raw content
		return Decision{
			Summary:    extractFirstSentenceFromContent(content),
			Confidence: fallbackConfidence,
		}, nil
	}

	// Validate confidence is in valid range
	confidence := resp.Confidence
	if confidence <= 0 || confidence > 1.0 {
		confidence = fallbackConfidence
	}

	return Decision{
		Summary:      resp.Summary,
		Reasoning:    resp.Reasoning,
		Alternatives: resp.Alternatives,
		Tags:         resp.Tags,
		Confidence:   confidence,
	}, nil
}

// extractFirstSentenceFromContent extracts the first sentence as a fallback summary.
func extractFirstSentenceFromContent(content string) string {
	// Find first period, exclamation, or question mark
	for i, r := range content {
		if r == '.' || r == '!' || r == '?' {
			if i < len(content)-1 {
				return content[:i+1]
			}
		}
		// Limit to first 200 chars
		if i >= 200 {
			return content[:200] + "..."
		}
	}
	if len(content) > 200 {
		return content[:200] + "..."
	}
	return content
}

// retryableError wraps an error to indicate it can be retried.
type retryableError struct {
	err error
}

func (e *retryableError) Error() string {
	return e.err.Error()
}

func (e *retryableError) Unwrap() error {
	return e.err
}

// isRetryableError checks if an error should be retried.
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}
	// Check if it's a retryableError type
	if _, isRetryable := err.(*retryableError); isRetryable {
		return true
	}
	// Also check unwrapped error
	for e := err; e != nil; {
		if _, ok := e.(*retryableError); ok {
			return true
		}
		if unwrapper, ok := e.(interface{ Unwrap() error }); ok {
			e = unwrapper.Unwrap()
		} else {
			break
		}
	}
	return false
}

// scrubSecrets removes common secret patterns from content before sending to API.
// This prevents accidental leakage of API keys, tokens, passwords, etc.
func scrubSecrets(content string) string {
	// Define secret patterns to scrub (order matters - more specific first)
	patterns := []struct {
		regex       *regexp.Regexp
		replacement string
	}{
		// Environment variables with sensitive data (must be first to catch specific patterns)
		{
			regexp.MustCompile(`(OPENAI_API_KEY|ANTHROPIC_API_KEY|GITHUB_TOKEN|GITLAB_TOKEN|AWS_SECRET_ACCESS_KEY)\s*=\s*([^\s]+)`),
			"$1=[REDACTED:ENV_SECRET]",
		},
		// OpenAI API keys (sk- followed by 48+ alphanumeric chars)
		{
			regexp.MustCompile(`sk-[a-zA-Z0-9]{20,}`),
			"[REDACTED:OPENAI_KEY]",
		},
		// Anthropic API keys (sk-ant- followed by many chars)
		{
			regexp.MustCompile(`sk-ant-[a-zA-Z0-9-]{20,}`),
			"[REDACTED:ANTHROPIC_KEY]",
		},
		// Generic API keys in various formats
		{
			regexp.MustCompile(`(?i)(api[_-]?key|apikey)\s*[:=]\s*["']?\s*([^"'\s]{8,})["']?`),
			"$1=[REDACTED:API_KEY]",
		},
		// Bearer tokens
		{
			regexp.MustCompile(`(?i)bearer\s+[a-zA-Z0-9_\-\.=]{20,}`),
			"[REDACTED:BEARER_TOKEN]",
		},
		// Tokens
		{
			regexp.MustCompile(`(?i)(token|auth[_-]?token)\s*[:=]\s*["']?\s*([^"'\s]{8,})["']?`),
			"$1=[REDACTED:TOKEN]",
		},
		// Passwords
		{
			regexp.MustCompile(`(?i)(password|passwd|pwd)\s*[:=]\s*["']?\s*([^"'\s]{4,})["']?`),
			"$1=[REDACTED:PASSWORD]",
		},
		// Private keys
		{
			regexp.MustCompile(`(?i)-----BEGIN (RSA |EC |DSA |OPENSSH )?PRIVATE KEY-----[\s\S]*?-----END (RSA |EC |DSA |OPENSSH )?PRIVATE KEY-----`),
			"[REDACTED:PRIVATE_KEY]",
		},
	}

	result := content
	for _, p := range patterns {
		result = p.regex.ReplaceAllString(result, p.replacement)
	}

	return result
}

// Ensure interfaces are implemented.
var _ Summarizer = (*anthropicSummarizer)(nil)
var _ Summarizer = (*openAISummarizer)(nil)
