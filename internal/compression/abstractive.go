// Package compression provides text compression algorithms for context optimization.
//
// This package implements extractive, abstractive, and hybrid compression
// techniques to reduce token usage while preserving semantic meaning.
package compression

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/fyrsmithlabs/contextd/internal/vectorstore"
)

const (
	// Anthropic API endpoint
	anthropicAPIURL = "https://api.anthropic.com/v1/messages"
	// Claude model for summarization
	claudeModel = "claude-3-haiku-20240307" // Fast and cost-effective
	// Anthropic API version
	anthropicVersion = "2023-06-01"
	// Max tokens for response
	maxTokens = 4096
)

// anthropicRequest represents an Anthropic API request
type anthropicRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	Messages  []anthropicMessage `json:"messages"`
}

// anthropicMessage represents a message in the Anthropic API format
type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// anthropicResponse represents an Anthropic API response
type anthropicResponse struct {
	ID      string             `json:"id"`
	Type    string             `json:"type"`
	Role    string             `json:"role"`
	Content []anthropicContent `json:"content"`
	Model   string             `json:"model"`
	Error   *anthropicError    `json:"error,omitempty"`
}

// anthropicContent represents content in the response
type anthropicContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// anthropicError represents an error from the API
type anthropicError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// AbstractiveCompressor implements abstractive summarization using Claude API
type AbstractiveCompressor struct {
	config Config
	client *http.Client
}

// NewAbstractiveCompressor creates a new abstractive compressor
func NewAbstractiveCompressor(config Config) *AbstractiveCompressor {
	return &AbstractiveCompressor{
		config: config,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Compress implements the Compressor interface using abstractive summarization via Claude API
func (c *AbstractiveCompressor) Compress(ctx context.Context, content string, algorithm Algorithm, targetRatio float64) (*Result, error) {
	start := time.Now()

	// Validate API key is configured
	if c.config.AnthropicAPIKey == "" {
		return nil, fmt.Errorf("Anthropic API key not configured for abstractive compression")
	}

	// For very short content, return as-is
	if len(content) < 100 {
		return &Result{
			Content:        content,
			ProcessingTime: time.Since(start),
			QualityScore:   1.0,
			Metadata: vectorstore.CompressionMetadata{
				Level:            vectorstore.CompressionLevelSummary,
				Algorithm:        string(algorithm),
				OriginalSize:     len(content),
				CompressedSize:   len(content),
				CompressionRatio: 1.0,
				CompressedAt:     &start,
			},
		}, nil
	}

	// Create compression prompt
	targetReduction := int((1.0 - 1.0/targetRatio) * 100) // Convert ratio to percentage
	prompt := fmt.Sprintf(`Compress the following text to approximately %d%% of its original length while preserving all key information and semantic meaning. Focus on:
1. Removing redundant information
2. Consolidating similar ideas
3. Using concise language
4. Maintaining factual accuracy

Text to compress:
%s

Provide only the compressed version without any explanations or meta-commentary.`, targetReduction, content)

	// Call Claude API
	compressedContent, err := c.callClaudeAPI(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("Claude API call failed: %w", err)
	}

	// Calculate metrics
	originalSize := len(content)
	compressedSize := len(compressedContent)
	compressionRatio := float64(originalSize) / float64(compressedSize)
	if compressedSize == 0 {
		compressionRatio = 1.0
	}

	// Quality score based on compression achievement vs target
	ratioAchievement := math.Min(compressionRatio/targetRatio, 1.0)
	// Penalize if we compressed too much (might lose information)
	if compressionRatio > targetRatio*1.2 {
		ratioAchievement *= 0.9
	}
	qualityScore := 0.3 + (ratioAchievement * 0.6) // Range 0.3-0.9

	return &Result{
		Content:        compressedContent,
		ProcessingTime: time.Since(start),
		QualityScore:   qualityScore,
		Metadata: vectorstore.CompressionMetadata{
			Level:            vectorstore.CompressionLevelSummary,
			Algorithm:        string(algorithm),
			OriginalSize:     originalSize,
			CompressedSize:   compressedSize,
			CompressionRatio: compressionRatio,
			CompressedAt:     &start,
		},
	}, nil
}

// callClaudeAPI makes a request to the Anthropic Claude API
func (c *AbstractiveCompressor) callClaudeAPI(ctx context.Context, prompt string) (string, error) {
	// Prepare request
	reqBody := anthropicRequest{
		Model:     claudeModel,
		MaxTokens: maxTokens,
		Messages: []anthropicMessage{
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", anthropicAPIURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.config.AnthropicAPIKey)
	req.Header.Set("anthropic-version", anthropicVersion)

	// Make request
	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	// Check for HTTP errors
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var apiResp anthropicResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	// Check for API errors
	if apiResp.Error != nil {
		return "", fmt.Errorf("API error: %s - %s", apiResp.Error.Type, apiResp.Error.Message)
	}

	// Extract text from response
	if len(apiResp.Content) == 0 {
		return "", fmt.Errorf("API returned no content")
	}

	return strings.TrimSpace(apiResp.Content[0].Text), nil
}

// GetCapabilities returns the capabilities of this compressor
func (c *AbstractiveCompressor) GetCapabilities(ctx context.Context) Capabilities {
	return Capabilities{
		SupportedAlgorithms: []Algorithm{AlgorithmAbstractive},
		MaxContentLength:    50000, // 50KB (API token limits)
		SupportsTargetRatio: true,
		QualityScoreRange: struct {
			Min float64
			Max float64
		}{
			Min: 0.3,
			Max: 0.9,
		},
	}
}
