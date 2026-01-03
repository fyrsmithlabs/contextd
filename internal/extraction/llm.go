package extraction

import (
	"context"
	"fmt"
)

// anthropicSummarizer implements Summarizer using Anthropic's Claude API.
// Currently a stub - full implementation requires langchain-go or direct API.
type anthropicSummarizer struct {
	model   string
	apiKey  string
	baseURL string
}

// newAnthropicSummarizer creates a new Anthropic summarizer.
func newAnthropicSummarizer(cfg Config) (Summarizer, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("anthropic API key required")
	}

	return &anthropicSummarizer{
		model:   cfg.Model,
		apiKey:  cfg.APIKey,
		baseURL: cfg.BaseURL,
	}, nil
}

// Summarize refines a decision candidate using Claude.
func (a *anthropicSummarizer) Summarize(ctx context.Context, candidate DecisionCandidate) (Decision, error) {
	// TODO: Implement actual API call using langchain-go or anthropic SDK
	// For now, return a basic decision
	return Decision{
		Summary:    extractFirstSentence(candidate.Content),
		Confidence: candidate.Confidence,
	}, nil
}

// Available returns true if the summarizer is configured.
func (a *anthropicSummarizer) Available() bool {
	return a.apiKey != ""
}

// openAISummarizer implements Summarizer using OpenAI's API.
// Currently a stub - full implementation requires langchain-go or direct API.
type openAISummarizer struct {
	model   string
	apiKey  string
	baseURL string
}

// newOpenAISummarizer creates a new OpenAI summarizer.
func newOpenAISummarizer(cfg Config) (Summarizer, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("openai API key required")
	}

	return &openAISummarizer{
		model:   cfg.Model,
		apiKey:  cfg.APIKey,
		baseURL: cfg.BaseURL,
	}, nil
}

// Summarize refines a decision candidate using GPT.
func (o *openAISummarizer) Summarize(ctx context.Context, candidate DecisionCandidate) (Decision, error) {
	// TODO: Implement actual API call using langchain-go or openai SDK
	// For now, return a basic decision
	return Decision{
		Summary:    extractFirstSentence(candidate.Content),
		Confidence: candidate.Confidence,
	}, nil
}

// Available returns true if the summarizer is configured.
func (o *openAISummarizer) Available() bool {
	return o.apiKey != ""
}

// Ensure interfaces are implemented.
var _ Summarizer = (*anthropicSummarizer)(nil)
var _ Summarizer = (*openAISummarizer)(nil)
