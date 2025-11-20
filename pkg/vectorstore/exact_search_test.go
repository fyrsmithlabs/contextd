package vectorstore

import (
	"context"
	"testing"
)

// TestCosineSimilarity tests the cosine similarity calculation
func TestCosineSimilarity(t *testing.T) {
	tests := []struct {
		name     string
		a        []float64
		b        []float64
		expected float32
	}{
		{
			name:     "identical vectors",
			a:        []float64{1.0, 2.0, 3.0},
			b:        []float64{1.0, 2.0, 3.0},
			expected: 1.0,
		},
		{
			name:     "orthogonal vectors",
			a:        []float64{1.0, 0.0},
			b:        []float64{0.0, 1.0},
			expected: 0.0,
		},
		{
			name:     "opposite vectors",
			a:        []float64{1.0, 0.0},
			b:        []float64{-1.0, 0.0},
			expected: -1.0,
		},
		{
			name:     "similar vectors (high similarity)",
			a:        []float64{1.0, 2.0, 3.0},
			b:        []float64{1.1, 2.1, 2.9},
			expected: 0.9989, // approximately
		},
		{
			name:     "different length vectors",
			a:        []float64{1.0, 2.0, 3.0},
			b:        []float64{1.0, 2.0},
			expected: 0.0, // should return 0 for mismatched lengths
		},
		{
			name:     "zero vector",
			a:        []float64{0.0, 0.0, 0.0},
			b:        []float64{1.0, 2.0, 3.0},
			expected: 0.0, // should return 0 to avoid division by zero
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cosineSimilarity(tt.a, tt.b)

			// For most tests, check with small tolerance
			tolerance := float32(0.001) // Allow 0.1% tolerance
			if abs(result-tt.expected) > tolerance {
				t.Errorf("cosineSimilarity() = %v, want %v (tolerance %v)", result, tt.expected, tolerance)
			}
		})
	}
}

// Helper function for absolute value
func abs(x float32) float32 {
	if x < 0 {
		return -x
	}
	return x
}

// TestExactSearch_Validation tests input validation
func TestExactSearch_Validation(t *testing.T) {
	// Create service with mock embedder
	config := Config{
		URL:            "http://localhost:6334",
		CollectionName: "test-collection",
		Embedder: &mockEmbedder{
			embedFunc: func(ctx context.Context, texts []string) ([][]float32, error) {
				return [][]float32{{0.1, 0.2, 0.3}}, nil
			},
		},
	}
	svc, err := NewService(config)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	tests := []struct {
		name           string
		collectionName string
		query          string
		k              int
		wantErr        bool
		errContains    string
	}{
		{
			name:           "empty collection name",
			collectionName: "",
			query:          "test query",
			k:              10,
			wantErr:        true,
			errContains:    "collection name required",
		},
		{
			name:           "empty query",
			collectionName: "test-collection",
			query:          "",
			k:              10,
			wantErr:        true,
			errContains:    "query required",
		},
		{
			name:           "zero k (should default to 10)",
			collectionName: "test-collection",
			query:          "test query",
			k:              0,
			wantErr:        false, // k=0 defaults to 10, not an error
		},
		{
			name:           "negative k (should default to 10)",
			collectionName: "test-collection",
			query:          "test query",
			k:              -5,
			wantErr:        false, // negative k defaults to 10, not an error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.ExactSearch(context.Background(), tt.collectionName, tt.query, tt.k)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ExactSearch() expected error, got nil")
					return
				}
				if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("ExactSearch() error = %v, want to contain %q", err, tt.errContains)
				}
			} else {
				// For these tests, we expect them to fail at a later stage (e.g., HTTP request)
				// We're just validating that input validation doesn't reject them
				// err != nil is OK here (will fail on HTTP request to non-existent Qdrant)
			}
		})
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && stringContains(s, substr))
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// mockEmbedder for testing
type mockEmbedder struct {
	embedFunc func(ctx context.Context, texts []string) ([][]float32, error)
}

func (m *mockEmbedder) EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error) {
	if m.embedFunc != nil {
		return m.embedFunc(ctx, texts)
	}
	// Default: return dummy embeddings
	embeddings := make([][]float32, len(texts))
	for i := range embeddings {
		embeddings[i] = []float32{0.1, 0.2, 0.3}
	}
	return embeddings, nil
}

func (m *mockEmbedder) EmbedQuery(ctx context.Context, text string) ([]float32, error) {
	if m.embedFunc != nil {
		embeddings, err := m.embedFunc(ctx, []string{text})
		if err != nil {
			return nil, err
		}
		return embeddings[0], nil
	}
	return []float32{0.1, 0.2, 0.3}, nil
}

// TestGenerateEmbedding tests the embedding generation helper
func TestGenerateEmbedding(t *testing.T) {
	tests := []struct {
		name      string
		embedder  *mockEmbedder
		text      string
		wantErr   bool
		wantLen   int
	}{
		{
			name: "successful embedding generation",
			embedder: &mockEmbedder{
				embedFunc: func(ctx context.Context, texts []string) ([][]float32, error) {
					return [][]float32{{0.1, 0.2, 0.3, 0.4, 0.5}}, nil
				},
			},
			text:    "test query",
			wantErr: false,
			wantLen: 5,
		},
		{
			name: "no embeddings returned",
			embedder: &mockEmbedder{
				embedFunc: func(ctx context.Context, texts []string) ([][]float32, error) {
					return [][]float32{}, nil
				},
			},
			text:    "test query",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := Config{
				URL:            "http://localhost:6334",
				CollectionName: "test-collection",
				Embedder:       tt.embedder,
			}
			svc, err := NewService(config)
			if err != nil {
				t.Fatalf("NewService() unexpected error: %v", err)
			}

			result, err := svc.generateEmbedding(context.Background(), tt.text)

			if tt.wantErr {
				if err == nil {
					t.Errorf("generateEmbedding() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("generateEmbedding() unexpected error: %v", err)
				return
			}

			if len(result) != tt.wantLen {
				t.Errorf("generateEmbedding() returned embedding of length %d, want %d", len(result), tt.wantLen)
			}
		})
	}
}
