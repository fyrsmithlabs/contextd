package embeddings

import (
	"context"
	"os"
	"testing"
)

func TestNewFastEmbedProvider(t *testing.T) {
	// Skip in short mode as this downloads models
	if testing.Short() {
		t.Skip("skipping FastEmbed test in short mode")
	}

	// Skip if ONNX runtime not available
	if _, err := os.Stat("/usr/lib/libonnxruntime.so"); os.IsNotExist(err) {
		if os.Getenv("ONNX_PATH") == "" {
			t.Skip("ONNX runtime not available, skipping FastEmbed test")
		}
	}

	tests := []struct {
		name      string
		cfg       FastEmbedConfig
		wantDim   int
		wantError bool
	}{
		{
			name: "default model",
			cfg: FastEmbedConfig{
				Model: "BAAI/bge-small-en-v1.5",
			},
			wantDim:   384,
			wantError: false,
		},
		{
			name: "fastembed model name",
			cfg: FastEmbedConfig{
				Model: "fast-bge-small-en-v1.5",
			},
			wantDim:   384,
			wantError: false,
		},
		{
			name: "base model",
			cfg: FastEmbedConfig{
				Model: "BAAI/bge-base-en-v1.5",
			},
			wantDim:   768,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewFastEmbedProvider(tt.cfg)
			if tt.wantError {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("NewFastEmbedProvider() error = %v", err)
			}
			defer provider.Close()

			if provider.Dimension() != tt.wantDim {
				t.Errorf("Dimension() = %d, want %d", provider.Dimension(), tt.wantDim)
			}
		})
	}
}

func TestFastEmbedProvider_EmbedDocuments(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping FastEmbed test in short mode")
	}

	if _, err := os.Stat("/usr/lib/libonnxruntime.so"); os.IsNotExist(err) {
		if os.Getenv("ONNX_PATH") == "" {
			t.Skip("ONNX runtime not available")
		}
	}

	provider, err := NewFastEmbedProvider(FastEmbedConfig{
		Model: "BAAI/bge-small-en-v1.5",
	})
	if err != nil {
		t.Fatalf("NewFastEmbedProvider() error = %v", err)
	}
	defer provider.Close()

	ctx := context.Background()

	t.Run("single document", func(t *testing.T) {
		embeddings, err := provider.EmbedDocuments(ctx, []string{"Hello world"})
		if err != nil {
			t.Fatalf("EmbedDocuments() error = %v", err)
		}
		if len(embeddings) != 1 {
			t.Errorf("expected 1 embedding, got %d", len(embeddings))
		}
		if len(embeddings[0]) != 384 {
			t.Errorf("expected 384 dimensions, got %d", len(embeddings[0]))
		}
	})

	t.Run("multiple documents", func(t *testing.T) {
		texts := []string{"Hello world", "Test document", "Another text"}
		embeddings, err := provider.EmbedDocuments(ctx, texts)
		if err != nil {
			t.Fatalf("EmbedDocuments() error = %v", err)
		}
		if len(embeddings) != 3 {
			t.Errorf("expected 3 embeddings, got %d", len(embeddings))
		}
	})

	t.Run("empty input", func(t *testing.T) {
		_, err := provider.EmbedDocuments(ctx, []string{})
		if err == nil {
			t.Error("expected error for empty input")
		}
	})
}

func TestFastEmbedProvider_EmbedQuery(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping FastEmbed test in short mode")
	}

	if _, err := os.Stat("/usr/lib/libonnxruntime.so"); os.IsNotExist(err) {
		if os.Getenv("ONNX_PATH") == "" {
			t.Skip("ONNX runtime not available")
		}
	}

	provider, err := NewFastEmbedProvider(FastEmbedConfig{
		Model: "BAAI/bge-small-en-v1.5",
	})
	if err != nil {
		t.Fatalf("NewFastEmbedProvider() error = %v", err)
	}
	defer provider.Close()

	ctx := context.Background()

	t.Run("valid query", func(t *testing.T) {
		embedding, err := provider.EmbedQuery(ctx, "test query")
		if err != nil {
			t.Fatalf("EmbedQuery() error = %v", err)
		}
		if len(embedding) != 384 {
			t.Errorf("expected 384 dimensions, got %d", len(embedding))
		}
	})

	t.Run("empty query", func(t *testing.T) {
		_, err := provider.EmbedQuery(ctx, "")
		if err == nil {
			t.Error("expected error for empty query")
		}
	})
}

func TestModelMapping(t *testing.T) {
	tests := []struct {
		name        string
		modelName   string
		wantDim     int
		shouldExist bool
	}{
		{"BAAI format", "BAAI/bge-small-en-v1.5", 384, true},
		{"fastembed format", "fast-bge-small-en-v1.5", 384, true},
		{"base model", "BAAI/bge-base-en-v1.5", 768, true},
		{"MiniLM", "sentence-transformers/all-MiniLM-L6-v2", 384, true},
		{"unknown", "unknown-model", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model, ok := modelMapping[tt.modelName]
			if tt.shouldExist {
				if !ok {
					t.Errorf("model %q should be in mapping", tt.modelName)
					return
				}
				dim := modelDimensions[model]
				if dim != tt.wantDim {
					t.Errorf("dimension = %d, want %d", dim, tt.wantDim)
				}
			} else {
				if ok {
					t.Errorf("model %q should not be in mapping", tt.modelName)
				}
			}
		})
	}
}
