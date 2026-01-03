package extraction

import (
	"context"
	"testing"
)

func TestNewDecisionExtractor(t *testing.T) {
	tests := []struct {
		name    string
		cfg     ExtractionConfig
		wantNil bool
		wantErr bool
	}{
		{
			name:    "disabled returns NoOp",
			cfg:     ExtractionConfig{Enabled: false},
			wantNil: false,
			wantErr: false,
		},
		{
			name:    "enabled returns Heuristic",
			cfg:     DefaultConfig(),
			wantNil: false,
			wantErr: false,
		},
		{
			name:    "empty config returns Heuristic",
			cfg:     ExtractionConfig{Enabled: true},
			wantNil: false,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewDecisionExtractor(tt.cfg)

			if (err != nil) != tt.wantErr {
				t.Errorf("NewDecisionExtractor() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if (got == nil) != tt.wantNil {
				t.Errorf("NewDecisionExtractor() = %v, wantNil %v", got, tt.wantNil)
			}
		})
	}
}

func TestNewSummarizer(t *testing.T) {
	tests := []struct {
		name    string
		cfg     ExtractionConfig
		wantErr bool
	}{
		{
			name:    "disabled returns NoOp",
			cfg:     ExtractionConfig{Enabled: false},
			wantErr: false,
		},
		{
			name:    "heuristic provider returns NoOp",
			cfg:     ExtractionConfig{Enabled: true, Provider: "heuristic"},
			wantErr: false,
		},
		{
			name: "anthropic without config errors",
			cfg: ExtractionConfig{
				Enabled:  true,
				Provider: "anthropic",
			},
			wantErr: true,
		},
		{
			name: "openai without config errors",
			cfg: ExtractionConfig{
				Enabled:  true,
				Provider: "openai",
			},
			wantErr: true,
		},
		{
			name: "unknown provider errors",
			cfg: ExtractionConfig{
				Enabled:   true,
				Provider:  "unknown",
				Providers: map[string]Config{"unknown": {}},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewSummarizer(tt.cfg)

			if (err != nil) != tt.wantErr {
				t.Errorf("NewSummarizer() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && got == nil {
				t.Error("NewSummarizer() returned nil without error")
			}
		})
	}
}

func TestNoOpExtractor_Extract(t *testing.T) {
	extractor := &NoOpExtractor{}

	messages := []RawMessage{
		{SessionID: "s1", UUID: "m1", Role: "assistant", Content: "Let's use this approach."},
	}

	candidates, err := extractor.Extract(messages)
	if err != nil {
		t.Fatalf("Extract() error = %v", err)
	}

	if len(candidates) != 0 {
		t.Errorf("Extract() got %d candidates, want 0", len(candidates))
	}
}

func TestNoOpSummarizer_Summarize(t *testing.T) {
	summarizer := &NoOpSummarizer{}

	candidate := DecisionCandidate{
		SessionID:   "s1",
		MessageUUID: "m1",
		Content:     "This is the first sentence. This is the second sentence.",
		Confidence:  0.8,
	}

	decision, err := summarizer.Summarize(context.Background(), candidate)
	if err != nil {
		t.Fatalf("Summarize() error = %v", err)
	}

	// Should extract first sentence
	if decision.Summary != "This is the first sentence." {
		t.Errorf("Summary = %q, want %q", decision.Summary, "This is the first sentence.")
	}

	// Confidence should be preserved
	if decision.Confidence != 0.8 {
		t.Errorf("Confidence = %f, want 0.8", decision.Confidence)
	}
}

func TestNoOpSummarizer_Available(t *testing.T) {
	summarizer := &NoOpSummarizer{}

	if summarizer.Available() {
		t.Error("Available() = true, want false")
	}
}

func TestExtractFirstSentence(t *testing.T) {
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
			content: string(make([]byte, 300)), // 300 characters
			want:    string(make([]byte, 200)) + "...",
		},
		{
			name:    "no punctuation short",
			content: "Short content",
			want:    "Short content",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractFirstSentence(tt.content)
			if got != tt.want {
				t.Errorf("extractFirstSentence() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if !cfg.Enabled {
		t.Error("DefaultConfig().Enabled = false, want true")
	}

	if cfg.Provider != "heuristic" {
		t.Errorf("DefaultConfig().Provider = %q, want 'heuristic'", cfg.Provider)
	}

	if len(cfg.Patterns) == 0 {
		t.Error("DefaultConfig().Patterns is empty")
	}

	if cfg.ConfidenceThreshold <= 0 {
		t.Error("DefaultConfig().ConfidenceThreshold <= 0")
	}

	if cfg.LLMRefineThreshold <= 0 {
		t.Error("DefaultConfig().LLMRefineThreshold <= 0")
	}
}
