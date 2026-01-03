package extraction

import (
	"context"
	"fmt"
)

// NewDecisionExtractor creates a decision extractor based on configuration.
func NewDecisionExtractor(cfg ExtractionConfig) (DecisionExtractor, error) {
	if !cfg.Enabled {
		return &NoOpExtractor{}, nil
	}

	// For now, only heuristic extraction is supported
	return NewHeuristicExtractor(cfg)
}

// NewSummarizer creates a summarizer based on configuration.
func NewSummarizer(cfg ExtractionConfig) (Summarizer, error) {
	if !cfg.Enabled || cfg.Provider == "disabled" || cfg.Provider == "heuristic" {
		return &NoOpSummarizer{}, nil
	}

	providerCfg, ok := cfg.Providers[cfg.Provider]
	if !ok {
		return nil, fmt.Errorf("provider %q not configured", cfg.Provider)
	}

	switch cfg.Provider {
	case "anthropic":
		return newAnthropicSummarizer(providerCfg)
	case "openai":
		return newOpenAISummarizer(providerCfg)
	default:
		return nil, fmt.Errorf("unknown provider: %s", cfg.Provider)
	}
}

// NoOpExtractor is a no-op implementation of DecisionExtractor.
type NoOpExtractor struct{}

// Extract returns an empty slice.
func (n *NoOpExtractor) Extract(messages []RawMessage) ([]DecisionCandidate, error) {
	return []DecisionCandidate{}, nil
}

// NoOpSummarizer is a no-op implementation of Summarizer.
type NoOpSummarizer struct{}

// Summarize returns the candidate as-is without LLM refinement.
func (n *NoOpSummarizer) Summarize(ctx context.Context, candidate DecisionCandidate) (Decision, error) {
	// Create a basic decision from the candidate without LLM
	return Decision{
		Summary:    extractFirstSentence(candidate.Content),
		Confidence: candidate.Confidence,
	}, nil
}

// Available returns false for NoOpSummarizer.
func (n *NoOpSummarizer) Available() bool {
	return false
}

// extractFirstSentence extracts the first sentence as a summary.
func extractFirstSentence(content string) string {
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

// Ensure interfaces are implemented.
var _ DecisionExtractor = (*NoOpExtractor)(nil)
var _ Summarizer = (*NoOpSummarizer)(nil)
