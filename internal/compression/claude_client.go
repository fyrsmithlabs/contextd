package compression

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"time"
)

// ClaudeClient defines the interface for Claude API interactions
// This enables testing with mocks
type ClaudeClient interface {
	// Summarize generates an abstractive summary using Claude API
	Summarize(ctx context.Context, content string, targetRatio float64) (string, error)
}

// HTTPClaudeClient implements ClaudeClient using the Anthropic API
type HTTPClaudeClient struct {
	apiKey     string
	baseURL    string
	model      string
	httpClient *http.Client
}

// ClaudeRequest represents the request format for Claude API
type ClaudeRequest struct {
	Model       string          `json:"model"`
	MaxTokens   int             `json:"max_tokens"`
	Messages    []ClaudeMessage `json:"messages"`
	System      string          `json:"system,omitempty"`
	Temperature float64         `json:"temperature"`
}

// ClaudeMessage represents a message in the conversation
type ClaudeMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ClaudeResponse represents the response from Claude API
type ClaudeResponse struct {
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

// ClaudeError represents an error response from Claude API
type ClaudeError struct {
	Type  string `json:"type"`
	Error struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

// NewClaudeClient creates a new Claude API client
func NewClaudeClient(apiKey, baseURL, model string) (*HTTPClaudeClient, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("API key is required")
	}
	if baseURL == "" {
		baseURL = "https://api.anthropic.com"
	}
	if model == "" {
		model = "claude-3-5-sonnet-20241022"
	}

	return &HTTPClaudeClient{
		apiKey:  apiKey,
		baseURL: baseURL,
		model:   model,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}, nil
}

// Summarize generates an abstractive summary using Claude API
func (c *HTTPClaudeClient) Summarize(ctx context.Context, content string, targetRatio float64) (string, error) {
	// Scrub secrets before sending to API
	scrubbedContent := scrubSecrets(content)

	// Calculate target length (approximation: 4 chars per token)
	targetLength := int(float64(len(scrubbedContent)) / targetRatio)
	if targetLength < 100 {
		targetLength = 100
	}

	// Build system prompt with compression instructions
	systemPrompt := fmt.Sprintf(`You are an expert at abstractive summarization. Your task is to compress the following content while preserving its semantic meaning and key information.

Target compression: %.1fx reduction (reduce to approximately %d characters)
Original length: %d characters

Requirements:
- Preserve the core semantic meaning
- Keep technical terms, code snippets, and key concepts
- Use concise language without losing context
- Maintain logical flow and structure
- If content contains code, preserve code blocks
- If content contains markdown, maintain markdown structure
- Remove redundancy and filler words

Generate ONLY the compressed summary, with no preamble or explanation.`, targetRatio, targetLength, len(scrubbedContent))

	// Build request
	req := ClaudeRequest{
		Model:       c.model,
		MaxTokens:   4096,
		Temperature: 0.3, // Lower temperature for consistent compression
		System:      systemPrompt,
		Messages: []ClaudeMessage{
			{
				Role:    "user",
				Content: scrubbedContent,
			},
		},
	}

	// Marshal request
	jsonData, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/v1/messages", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-API-Key", c.apiKey)
	httpReq.Header.Set("Anthropic-Version", "2023-06-01")

	// Send request
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		var errResp ClaudeError
		if err := json.Unmarshal(body, &errResp); err == nil {
			return "", fmt.Errorf("API error (%d): %s", resp.StatusCode, errResp.Error.Message)
		}
		return "", fmt.Errorf("API error (%d): %s", resp.StatusCode, string(body))
	}

	// Parse response
	var claudeResp ClaudeResponse
	if err := json.Unmarshal(body, &claudeResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	// Extract summary text
	if len(claudeResp.Content) == 0 {
		return "", fmt.Errorf("empty response from API")
	}

	summary := claudeResp.Content[0].Text
	if summary == "" {
		return "", fmt.Errorf("empty summary text")
	}

	return summary, nil
}

// scrubSecrets removes common secret patterns from content before sending to API
// This prevents accidental leakage of API keys, tokens, passwords, etc.
func scrubSecrets(content string) string {
	// Define secret patterns to scrub (order matters - more specific first)
	patterns := []struct {
		regex       *regexp.Regexp
		label       string
		replacement string
	}{
		// Environment variables with sensitive data (must be first to catch specific patterns)
		{
			regexp.MustCompile(`(OPENAI_API_KEY|ANTHROPIC_API_KEY|GITHUB_TOKEN|GITLAB_TOKEN|AWS_SECRET_ACCESS_KEY)\s*=\s*([^\s]+)`),
			"ENV_SECRET",
			"$1=[REDACTED:ENV_SECRET]",
		},
		// OpenAI API keys (sk- followed by 48 alphanumeric chars)
		{
			regexp.MustCompile(`sk-[a-zA-Z0-9]{48}`),
			"OPENAI_KEY",
			"[REDACTED:OPENAI_KEY]",
		},
		// Anthropic API keys (sk-ant- followed by many chars)
		{
			regexp.MustCompile(`sk-ant-[a-zA-Z0-9-]{80,}`),
			"ANTHROPIC_KEY",
			"[REDACTED:ANTHROPIC_KEY]",
		},
		// Generic API keys in various formats
		{
			regexp.MustCompile(`(?i)(api[_-]?key|apikey)\s*[:=]\s*["']?\s*([^"'\s]+)["']?`),
			"API_KEY",
			"$1=[REDACTED:API_KEY]",
		},
		// Bearer tokens
		{
			regexp.MustCompile(`(?i)bearer\s+[a-zA-Z0-9_\-\.=]+`),
			"BEARER_TOKEN",
			"[REDACTED:BEARER_TOKEN]",
		},
		// Tokens
		{
			regexp.MustCompile(`(?i)(token|auth[_-]?token)\s*[:=]\s*["']?\s*([^"'\s]+)["']?`),
			"TOKEN",
			"$1=[REDACTED:TOKEN]",
		},
		// Passwords
		{
			regexp.MustCompile(`(?i)(password|passwd|pwd)\s*[:=]\s*["']?\s*([^"'\s]+)["']?`),
			"PASSWORD",
			"$1=[REDACTED:PASSWORD]",
		},
	}

	result := content
	for _, p := range patterns {
		if p.replacement != "" {
			result = p.regex.ReplaceAllString(result, p.replacement)
		} else {
			result = p.regex.ReplaceAllString(result, "[REDACTED:"+p.label+"]")
		}
	}

	return result
}
