package embeddings

import (
	"os"
	"testing"
)

func TestNewProvider(t *testing.T) {
	tests := []struct {
		name      string
		cfg       ProviderConfig
		wantError bool
		skip      bool
		skipMsg   string
	}{
		{
			name: "tei provider with valid config",
			cfg: ProviderConfig{
				Provider: "tei",
				BaseURL:  "http://localhost:8080",
				Model:    "BAAI/bge-small-en-v1.5",
			},
			wantError: false,
		},
		{
			name: "tei provider without base URL",
			cfg: ProviderConfig{
				Provider: "tei",
				Model:    "BAAI/bge-small-en-v1.5",
			},
			wantError: true,
		},
		{
			name: "unknown provider",
			cfg: ProviderConfig{
				Provider: "unknown",
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skip {
				t.Skip(tt.skipMsg)
			}

			provider, err := NewProvider(tt.cfg)
			if tt.wantError {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("NewProvider() error = %v", err)
			}
			if provider != nil {
				provider.Close()
			}
		})
	}
}

func TestNewProvider_FastEmbed(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping FastEmbed test in short mode")
	}

	if _, err := os.Stat("/usr/lib/libonnxruntime.so"); os.IsNotExist(err) {
		if os.Getenv("ONNX_PATH") == "" {
			t.Skip("ONNX runtime not available")
		}
	}

	cfg := ProviderConfig{
		Provider: "fastembed",
		Model:    "BAAI/bge-small-en-v1.5",
	}

	provider, err := NewProvider(cfg)
	if err != nil {
		t.Fatalf("NewProvider() error = %v", err)
	}
	defer provider.Close()

	if provider.Dimension() != 384 {
		t.Errorf("Dimension() = %d, want 384", provider.Dimension())
	}
}

func TestNewProvider_DefaultToFastEmbed(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping FastEmbed test in short mode")
	}

	if _, err := os.Stat("/usr/lib/libonnxruntime.so"); os.IsNotExist(err) {
		if os.Getenv("ONNX_PATH") == "" {
			t.Skip("ONNX runtime not available")
		}
	}

	// Empty provider should default to fastembed
	cfg := ProviderConfig{
		Provider: "",
		Model:    "BAAI/bge-small-en-v1.5",
	}

	provider, err := NewProvider(cfg)
	if err != nil {
		t.Fatalf("NewProvider() error = %v", err)
	}
	defer provider.Close()

	if provider.Dimension() != 384 {
		t.Errorf("Dimension() = %d, want 384", provider.Dimension())
	}
}

func TestTEIProvider_Dimension(t *testing.T) {
	tests := []struct {
		name    string
		model   string
		wantDim int
	}{
		{"small model", "BAAI/bge-small-en-v1.5", 384},
		{"base model", "BAAI/bge-base-en-v1.5", 768},
		{"mini model", "sentence-transformers/all-MiniLM-L6-v2", 384},
		{"unknown defaults to 384", "unknown-model", 384},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := ProviderConfig{
				Provider: "tei",
				BaseURL:  "http://localhost:8080",
				Model:    tt.model,
			}

			provider, err := NewProvider(cfg)
			if err != nil {
				t.Fatalf("NewProvider() error = %v", err)
			}
			defer provider.Close()

			if provider.Dimension() != tt.wantDim {
				t.Errorf("Dimension() = %d, want %d", provider.Dimension(), tt.wantDim)
			}
		})
	}
}

func TestNewProvider_InvalidModel(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping FastEmbed test in short mode")
	}

	cfg := ProviderConfig{
		Provider: "fastembed",
		Model:    "nonexistent-model",
	}

	_, err := NewProvider(cfg)
	if err == nil {
		t.Error("expected error for invalid model")
	}
}
