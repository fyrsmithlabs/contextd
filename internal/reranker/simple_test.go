package reranker

import (
	"context"
	"testing"
)

func TestSimpleRerankerRerank(t *testing.T) {
	tests := []struct {
		name      string
		query     string
		docs      []Document
		topK      int
		wantCount int
		wantIDs   []string // Expected first N IDs
	}{
		{
			name:  "empty documents",
			query: "test query",
			docs:  []Document{},
			topK:  10,
			wantCount: 0,
		},
		{
			name:  "single document",
			query: "authentication error",
			docs: []Document{
				{ID: "doc1", Content: "authentication failed due to invalid token", Score: 0.9},
			},
			topK:      10,
			wantCount: 1,
			wantIDs:   []string{"doc1"},
		},
		{
			name:  "multiple documents with term overlap",
			query: "authentication token retry",
			docs: []Document{
				{ID: "doc1", Content: "use retry with exponential backoff for authentication", Score: 0.8},
				{ID: "doc2", Content: "invalid request parameter", Score: 0.9},
				{ID: "doc3", Content: "token refresh and authentication handling", Score: 0.85},
			},
			topK:      10,
			wantCount: 3,
			// doc3 and doc1 have high overlap with query, doc2 has none
			wantIDs: []string{"doc3", "doc1", "doc2"},
		},
		{
			name:  "topK limits results",
			query: "error handling",
			docs: []Document{
				{ID: "doc1", Content: "error handling patterns", Score: 0.9},
				{ID: "doc2", Content: "error recovery strategies", Score: 0.85},
				{ID: "doc3", Content: "error logging and monitoring", Score: 0.8},
				{ID: "doc4", Content: "error codes reference", Score: 0.75},
			},
			topK:      2,
			wantCount: 2,
		},
		{
			name:  "zero topK defaults to all documents",
			query: "test",
			docs: []Document{
				{ID: "a", Content: "test data", Score: 0.8},
				{ID: "b", Content: "another test", Score: 0.7},
			},
			topK:      0,
			wantCount: 2,
		},
		{
			name:      "empty query tokens",
			query:     "   ",
			docs: []Document{
				{ID: "doc1", Content: "some content", Score: 0.9},
			},
			topK:      10,
			wantCount: 1,
		},
		{
			name:  "combining original score with overlap",
			query: "database optimization",
			docs: []Document{
				// High original score, no overlap
				{ID: "high_score", Content: "irrelevant content about something else", Score: 0.95},
				// Lower original score, high overlap
				{ID: "high_overlap", Content: "database and optimization techniques", Score: 0.6},
			},
			topK:      10,
			wantCount: 2,
			// high_overlap should rank first due to term overlap despite lower original score
			// Combined: 0.5*0.95 + 0.5*0.0 = 0.475 vs 0.5*0.6 + 0.5*1.0 = 0.8
			wantIDs: []string{"high_overlap", "high_score"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reranker := NewSimpleReranker()
			defer reranker.Close()

			ctx := context.Background()
			results, err := reranker.Rerank(ctx, tt.query, tt.docs, tt.topK)

			if err != nil {
				t.Fatalf("Rerank() error = %v, want nil", err)
			}

			if len(results) != tt.wantCount {
				t.Errorf("Rerank() got %d results, want %d", len(results), tt.wantCount)
			}

			if tt.wantIDs != nil {
				for i, wantID := range tt.wantIDs {
					if i >= len(results) {
						t.Errorf("Rerank() got %d results, want at least %d", len(results), len(tt.wantIDs))
						break
					}
					if results[i].ID != wantID {
						t.Errorf("Rerank() position %d got ID %q, want %q", i, results[i].ID, wantID)
					}
				}
			}

			// Verify results are sorted by score descending
			for i := 1; i < len(results); i++ {
				if results[i-1].RerankerScore < results[i].RerankerScore {
					t.Errorf("Rerank() results not sorted: position %d (%.3f) < position %d (%.3f)",
						i-1, results[i-1].RerankerScore, i, results[i].RerankerScore)
				}
			}
		})
	}
}

func TestTokenize(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "simple text",
			input: "error handling retry",
			want:  []string{"error", "handling", "retry"},
		},
		{
			name:  "stopwords filtered",
			input: "the error handling and retry",
			want:  []string{"error", "handling", "retry"},
		},
		{
			name:  "punctuation removed",
			input: "error, handling; retry!",
			want:  []string{"error", "handling", "retry"},
		},
		{
			name:  "short tokens filtered",
			input: "a an to error handling",
			want:  []string{"error", "handling"},
		},
		{
			name:  "case normalization",
			input: "ERROR Handling RETRY",
			want:  []string{"error", "handling", "retry"},
		},
		{
			name:  "empty string",
			input: "",
			want:  []string{},
		},
		{
			name:  "only stopwords",
			input: "the a an and or but",
			want:  []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tokenize(tt.input)
			if len(got) != len(tt.want) {
				t.Errorf("tokenize() got %d tokens, want %d: %v vs %v", len(got), len(tt.want), got, tt.want)
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("tokenize() token %d got %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestCalculateTermOverlap(t *testing.T) {
	tests := []struct {
		name         string
		queryTokens  []string
		docTokens    []string
		wantApprox   float32 // Approximate overlap percentage
		wantTolerance float32
	}{
		{
			name:        "perfect overlap",
			queryTokens: []string{"error", "handling", "retry"},
			docTokens:   []string{"error", "handling", "retry"},
			wantApprox:  1.0,
			wantTolerance: 0.01,
		},
		{
			name:        "partial overlap",
			queryTokens: []string{"error", "handling", "retry"},
			docTokens:   []string{"error", "handling"},
			wantApprox:  0.67,
			wantTolerance: 0.01,
		},
		{
			name:        "no overlap",
			queryTokens: []string{"error", "handling"},
			docTokens:   []string{"success", "recovery"},
			wantApprox:  0.0,
			wantTolerance: 0.01,
		},
		{
			name:        "empty query",
			queryTokens: []string{},
			docTokens:   []string{"error", "handling"},
			wantApprox:  0.0,
			wantTolerance: 0.01,
		},
		{
			name:        "empty document",
			queryTokens: []string{"error", "handling"},
			docTokens:   []string{},
			wantApprox:  0.0,
			wantTolerance: 0.01,
		},
		{
			name:        "single token",
			queryTokens: []string{"error"},
			docTokens:   []string{"error"},
			wantApprox:  1.0,
			wantTolerance: 0.01,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateTermOverlap(tt.queryTokens, tt.docTokens)
			diff := got - tt.wantApprox
			if diff < 0 {
				diff = -diff
			}
			if diff > tt.wantTolerance {
				t.Errorf("calculateTermOverlap() got %.3f, want ~%.3f (tolerance: %.3f)", got, tt.wantApprox, tt.wantTolerance)
			}
		})
	}
}

func TestIsStopword(t *testing.T) {
	tests := []struct {
		token string
		want  bool
	}{
		{"the", true},
		{"error", false},
		{"and", true},
		{"handling", false},
		{"in", true},
		{"database", false},
	}

	for _, tt := range tests {
		t.Run(tt.token, func(t *testing.T) {
			got := isStopword(tt.token)
			if got != tt.want {
				t.Errorf("isStopword(%q) got %v, want %v", tt.token, got, tt.want)
			}
		})
	}
}

func TestSimpleRerankerClose(t *testing.T) {
	reranker := NewSimpleReranker()
	err := reranker.Close()
	if err != nil {
		t.Errorf("Close() error = %v, want nil", err)
	}
}

func BenchmarkSimpleRerankerRerank(b *testing.B) {
	reranker := NewSimpleReranker()
	defer reranker.Close()

	query := "authentication token retry error handling database optimization"
	docs := make([]Document, 100)
	for i := 0; i < len(docs); i++ {
		docs[i] = Document{
			ID:      "doc" + string(rune(i)),
			Content: "error handling with retry logic and authentication token management",
			Score:   0.8,
		}
	}

	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = reranker.Rerank(ctx, query, docs, 10)
	}
}
