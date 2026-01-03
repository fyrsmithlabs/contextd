package extraction

import (
	"testing"
)

func TestHeuristicExtractor_Extract(t *testing.T) {
	cfg := DefaultConfig()
	extractor, err := NewHeuristicExtractor(cfg)
	if err != nil {
		t.Fatalf("NewHeuristicExtractor() error = %v", err)
	}

	tests := []struct {
		name     string
		messages []RawMessage
		wantLen  int
		wantPattern string
	}{
		{
			name: "explicit decision - lets use",
			messages: []RawMessage{
				{SessionID: "s1", UUID: "m1", Role: "user", Content: "How should we implement caching?"},
				{SessionID: "s1", UUID: "m2", Role: "assistant", Content: "Let's use Redis for this since it's already in our stack."},
			},
			wantLen:     1,
			wantPattern: "lets_use",
		},
		{
			name: "explicit decision - decided to",
			messages: []RawMessage{
				{SessionID: "s1", UUID: "m1", Role: "assistant", Content: "After reviewing the options, I decided to use a factory pattern here."},
			},
			wantLen:     1,
			wantPattern: "decided_to",
		},
		{
			name: "explicit capture - remember this",
			messages: []RawMessage{
				{SessionID: "s1", UUID: "m1", Role: "assistant", Content: "Remember this: always validate input at the boundary."},
			},
			wantLen:     1,
			wantPattern: "remember_this",
		},
		{
			name: "anti-pattern - don't because",
			messages: []RawMessage{
				{SessionID: "s1", UUID: "m1", Role: "assistant", Content: "Don't use global state because it makes testing impossible."},
			},
			wantLen:     1,
			wantPattern: "dont_because",
		},
		{
			name: "no decision pattern",
			messages: []RawMessage{
				{SessionID: "s1", UUID: "m1", Role: "assistant", Content: "Here's the code to implement the feature."},
			},
			wantLen: 0,
		},
		{
			name: "user message ignored",
			messages: []RawMessage{
				{SessionID: "s1", UUID: "m1", Role: "user", Content: "Let's use Redis for caching."},
			},
			wantLen: 0, // User messages are not checked for decisions
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			candidates, err := extractor.Extract(tt.messages)
			if err != nil {
				t.Fatalf("Extract() error = %v", err)
			}

			if len(candidates) != tt.wantLen {
				t.Errorf("Extract() got %d candidates, want %d", len(candidates), tt.wantLen)
				return
			}

			if tt.wantLen > 0 && tt.wantPattern != "" {
				if candidates[0].PatternMatched != tt.wantPattern {
					t.Errorf("PatternMatched = %q, want %q", candidates[0].PatternMatched, tt.wantPattern)
				}
			}
		})
	}
}

func TestHeuristicExtractor_BestMatch(t *testing.T) {
	cfg := DefaultConfig()
	extractor, err := NewHeuristicExtractor(cfg)
	if err != nil {
		t.Fatalf("NewHeuristicExtractor() error = %v", err)
	}

	// Message with multiple patterns - should pick highest weight
	messages := []RawMessage{
		{
			SessionID: "s1",
			UUID:      "m1",
			Role:      "assistant",
			Content:   "Remember this: let's use Redis for caching.", // Both patterns match
		},
	}

	candidates, err := extractor.Extract(messages)
	if err != nil {
		t.Fatalf("Extract() error = %v", err)
	}

	if len(candidates) != 1 {
		t.Fatalf("Extract() got %d candidates, want 1", len(candidates))
	}

	// "remember_this" has weight 1.0, "lets_use" has weight 0.9
	if candidates[0].PatternMatched != "remember_this" {
		t.Errorf("PatternMatched = %q, want 'remember_this' (highest weight)", candidates[0].PatternMatched)
	}
}

func TestHeuristicExtractor_Context(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ContextWindowMessages = 2
	extractor, err := NewHeuristicExtractor(cfg)
	if err != nil {
		t.Fatalf("NewHeuristicExtractor() error = %v", err)
	}

	messages := []RawMessage{
		{SessionID: "s1", UUID: "m1", Role: "user", Content: "First message"},
		{SessionID: "s1", UUID: "m2", Role: "assistant", Content: "Second message"},
		{SessionID: "s1", UUID: "m3", Role: "user", Content: "Third message"},
		{SessionID: "s1", UUID: "m4", Role: "assistant", Content: "Let's use this approach."}, // Decision
	}

	candidates, err := extractor.Extract(messages)
	if err != nil {
		t.Fatalf("Extract() error = %v", err)
	}

	if len(candidates) != 1 {
		t.Fatalf("Extract() got %d candidates, want 1", len(candidates))
	}

	// Should have context from 2 preceding messages
	if len(candidates[0].Context) != 2 {
		t.Errorf("Context has %d messages, want 2", len(candidates[0].Context))
	}
}

func TestHeuristicExtractor_ConfidenceThreshold(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ConfidenceThreshold = 0.85 // Higher threshold

	extractor, err := NewHeuristicExtractor(cfg)
	if err != nil {
		t.Fatalf("NewHeuristicExtractor() error = %v", err)
	}

	// Message with pattern weight 0.7 (below threshold)
	messages := []RawMessage{
		{SessionID: "s1", UUID: "m1", Role: "assistant", Content: "The architecture should be modular."},
	}

	candidates, err := extractor.Extract(messages)
	if err != nil {
		t.Fatalf("Extract() error = %v", err)
	}

	// Should be filtered out by threshold
	if len(candidates) != 0 {
		t.Errorf("Extract() got %d candidates, want 0 (filtered by threshold)", len(candidates))
	}
}

func TestHeuristicExtractor_NeedsLLMRefine(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ConfidenceThreshold = 0.5
	cfg.LLMRefineThreshold = 0.85

	extractor, err := NewHeuristicExtractor(cfg)
	if err != nil {
		t.Fatalf("NewHeuristicExtractor() error = %v", err)
	}

	messages := []RawMessage{
		// Weight 0.7 - below LLM threshold, should need refinement
		{SessionID: "s1", UUID: "m1", Role: "assistant", Content: "This pattern for this use case works well."},
		// Weight 0.9 - above LLM threshold, should not need refinement
		{SessionID: "s1", UUID: "m2", Role: "assistant", Content: "Let's use this approach."},
	}

	candidates, err := extractor.Extract(messages)
	if err != nil {
		t.Fatalf("Extract() error = %v", err)
	}

	if len(candidates) != 2 {
		t.Fatalf("Extract() got %d candidates, want 2", len(candidates))
	}

	// First candidate should need LLM refinement (weight 0.7 < 0.85)
	if !candidates[0].NeedsLLMRefine {
		t.Error("candidates[0].NeedsLLMRefine = false, want true")
	}

	// Second candidate should not need LLM refinement (weight 0.9 >= 0.85)
	if candidates[1].NeedsLLMRefine {
		t.Error("candidates[1].NeedsLLMRefine = true, want false")
	}
}

func TestDefaultPatterns(t *testing.T) {
	patterns := DefaultPatterns()

	if len(patterns) == 0 {
		t.Error("DefaultPatterns() returned empty slice")
	}

	// Check some expected patterns exist
	names := make(map[string]bool)
	for _, p := range patterns {
		names[p.Name] = true
	}

	expectedNames := []string{"lets_use", "decided_to", "remember_this", "dont_because"}
	for _, name := range expectedNames {
		if !names[name] {
			t.Errorf("DefaultPatterns() missing pattern %q", name)
		}
	}
}
