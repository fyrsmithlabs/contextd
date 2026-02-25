// Package extraction provides decision detection and tag extraction
// from Claude Code conversation messages. It supports both heuristic
// (pattern-based) and LLM-based extraction methods.
package extraction

import (
	"context"
)

// Pattern represents a decision detection pattern.
type Pattern struct {
	Name   string  `json:"name"`
	Regex  string  `json:"regex"`
	Weight float64 `json:"weight"`
}

// DecisionCandidate represents a potential decision found in messages.
type DecisionCandidate struct {
	SessionID      string   `json:"session_id"`
	MessageUUID    string   `json:"message_uuid"`
	Content        string   `json:"content"`
	Context        []string `json:"context,omitempty"` // Surrounding messages
	PatternMatched string   `json:"pattern_matched"`
	Confidence     float64  `json:"confidence"`
	NeedsLLMRefine bool     `json:"needs_llm_refine"`
}

// Decision represents a refined, structured decision extracted from conversation.
type Decision struct {
	Summary      string   `json:"summary"`
	Alternatives []string `json:"alternatives,omitempty"`
	Reasoning    string   `json:"reasoning,omitempty"`
	Tags         []string `json:"tags,omitempty"`
	Confidence   float64  `json:"confidence"`
}

// RawMessage is the interface expected from conversation.RawMessage.
// We define it here to avoid circular imports.
type RawMessage struct {
	SessionID string `json:"session_id"`
	UUID      string `json:"uuid"`
	Role      string `json:"role"`
	Content   string `json:"content"`
}

// DecisionExtractor extracts decision candidates from messages.
type DecisionExtractor interface {
	// Extract finds decision candidates in messages.
	Extract(messages []RawMessage) ([]DecisionCandidate, error)
}

// Summarizer refines decision candidates using LLM or other methods.
type Summarizer interface {
	// Summarize refines a decision candidate into a structured decision.
	Summarize(ctx context.Context, candidate DecisionCandidate) (Decision, error)

	// Available returns true if the summarizer is configured and ready.
	Available() bool
}

// TagExtractor extracts tags from content based on rules.
type TagExtractor interface {
	// ExtractTags returns tags found in the content.
	ExtractTags(content string) []string

	// ExtractTagsFromFiles returns tags based on file paths.
	ExtractTagsFromFiles(paths []string) []string
}

// ExtractionConfig holds configuration for extraction operations.
type ExtractionConfig struct {
	Enabled   bool              `json:"enabled"`
	Provider  string            `json:"provider"` // "disabled", "heuristic", "anthropic", "openai"
	Providers map[string]Config `json:"providers,omitempty"`

	// Heuristic configuration
	Patterns              []Pattern `json:"patterns,omitempty"`
	ConfidenceThreshold   float64   `json:"confidence_threshold"`
	LLMRefineThreshold    float64   `json:"llm_refine_threshold"`
	ContextWindowMessages int       `json:"context_window_messages"`
}

// Config holds provider-specific configuration.
type Config struct {
	Model     string `json:"model,omitempty"`
	APIKey    string `json:"api_key,omitempty"`
	BaseURL   string `json:"base_url,omitempty"`
	MaxTokens int    `json:"max_tokens,omitempty"`
	Timeout   int    `json:"timeout,omitempty"`
}

// DefaultConfig returns a default extraction configuration.
func DefaultConfig() ExtractionConfig {
	return ExtractionConfig{
		Enabled:               true,
		Provider:              "heuristic",
		ConfidenceThreshold:   0.5,
		LLMRefineThreshold:    0.8,
		ContextWindowMessages: 3,
		Patterns:              DefaultPatterns(),
	}
}

// DefaultPatterns returns the default decision detection patterns.
func DefaultPatterns() []Pattern {
	return []Pattern{
		// Explicit decisions
		{Name: "lets_use", Regex: `(?i)let's (go with|use|choose|pick)`, Weight: 0.9},
		{Name: "decided_to", Regex: `(?i)decided to`, Weight: 0.9},
		{Name: "approach_is", Regex: `(?i)the approach (is|will be)`, Weight: 0.8},
		{Name: "choosing_over", Regex: `(?i)choosing .+ over`, Weight: 0.9},

		// Architectural
		{Name: "architecture", Regex: `(?i)architecture.*(should|will)`, Weight: 0.7},
		{Name: "pattern_for", Regex: `(?i)pattern for this`, Weight: 0.7},

		// Anti-patterns
		{Name: "dont_because", Regex: `(?i)don't (do|use).*because`, Weight: 0.8},
		{Name: "avoid_because", Regex: `(?i)avoid.*because`, Weight: 0.8},
		{Name: "failed_approach", Regex: `(?i)this (broke|failed)`, Weight: 0.7},

		// Explicit capture
		{Name: "remember_this", Regex: `(?i)remember (this|that)`, Weight: 1.0},
		{Name: "note_future", Regex: `(?i)note for (future|later)`, Weight: 1.0},
	}
}
