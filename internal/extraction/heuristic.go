package extraction

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

// HeuristicExtractor implements DecisionExtractor using pattern matching.
type HeuristicExtractor struct {
	patterns            []*compiledPattern
	confidenceThreshold float64
	llmRefineThreshold  float64
	contextWindow       int
}

// compiledPattern holds a pre-compiled regex pattern.
type compiledPattern struct {
	Pattern
	regex *regexp.Regexp
}

// NewHeuristicExtractor creates a new heuristic decision extractor.
func NewHeuristicExtractor(cfg ExtractionConfig) (*HeuristicExtractor, error) {
	patterns := cfg.Patterns
	if len(patterns) == 0 {
		patterns = DefaultPatterns()
	}

	compiled := make([]*compiledPattern, 0, len(patterns))
	for _, p := range patterns {
		re, err := regexp.Compile(p.Regex)
		if err != nil {
			// Skip invalid patterns
			continue
		}
		compiled = append(compiled, &compiledPattern{
			Pattern: p,
			regex:   re,
		})
	}

	confidenceThreshold := cfg.ConfidenceThreshold
	if confidenceThreshold == 0 {
		confidenceThreshold = 0.5
	}

	llmRefineThreshold := cfg.LLMRefineThreshold
	if llmRefineThreshold == 0 {
		llmRefineThreshold = 0.8
	}

	contextWindow := cfg.ContextWindowMessages
	if contextWindow == 0 {
		contextWindow = 3
	}

	return &HeuristicExtractor{
		patterns:            compiled,
		confidenceThreshold: confidenceThreshold,
		llmRefineThreshold:  llmRefineThreshold,
		contextWindow:       contextWindow,
	}, nil
}

// Extract finds decision candidates in messages using pattern matching.
func (h *HeuristicExtractor) Extract(messages []RawMessage) ([]DecisionCandidate, error) {
	var candidates []DecisionCandidate

	for i, msg := range messages {
		// Only check assistant messages for decisions
		if msg.Role != "assistant" {
			continue
		}

		// Check all patterns against the content
		match := h.findBestMatch(msg.Content)
		if match == nil {
			continue
		}

		// Skip if confidence is below threshold
		if match.Weight < h.confidenceThreshold {
			continue
		}

		// Build context from surrounding messages
		context := h.buildContext(messages, i)

		candidates = append(candidates, DecisionCandidate{
			SessionID:      msg.SessionID,
			MessageUUID:    msg.UUID,
			Content:        msg.Content,
			Context:        context,
			PatternMatched: match.Name,
			Confidence:     match.Weight,
			NeedsLLMRefine: match.Weight < h.llmRefineThreshold,
		})
	}

	return candidates, nil
}

// findBestMatch finds the pattern with highest weight that matches the content.
func (h *HeuristicExtractor) findBestMatch(content string) *compiledPattern {
	var best *compiledPattern
	var bestWeight float64

	for _, p := range h.patterns {
		if p.regex.MatchString(content) {
			if p.Weight > bestWeight {
				best = p
				bestWeight = p.Weight
			}
		}
	}

	return best
}

// buildContext builds context from surrounding messages.
func (h *HeuristicExtractor) buildContext(messages []RawMessage, idx int) []string {
	var context []string

	// Add messages before
	start := idx - h.contextWindow
	if start < 0 {
		start = 0
	}

	for i := start; i < idx; i++ {
		context = append(context, formatContextMessage(messages[i]))
	}

	return context
}

// formatContextMessage formats a message for context.
func formatContextMessage(msg RawMessage) string {
	role := capitalizeFirst(msg.Role)
	content := truncateToRunes(msg.Content, 200)
	return role + ": " + content
}

// truncateToRunes truncates a string to a maximum number of runes, preserving UTF-8 validity.
func truncateToRunes(s string, maxRunes int) string {
	runes := []rune(s)
	if len(runes) <= maxRunes {
		return s
	}
	return string(runes[:maxRunes]) + "..."
}

// capitalizeFirst capitalizes the first letter of a string, preserving UTF-8 validity.
func capitalizeFirst(s string) string {
	if s == "" {
		return s
	}
	r, size := utf8.DecodeRuneInString(s)
	if r == utf8.RuneError {
		return s
	}
	return strings.ToUpper(string(r)) + s[size:]
}

// Ensure HeuristicExtractor implements DecisionExtractor.
var _ DecisionExtractor = (*HeuristicExtractor)(nil)
