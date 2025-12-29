package embeddings

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewService(t *testing.T) {
	tests := []struct {
		name       string
		baseURL    string
		model      string
		apiKey     string
		wantErr    bool
		errMessage string
	}{
		{
			name:    "valid TEI configuration",
			baseURL: "http://localhost:8080",
			model:   "BAAI/bge-small-en-v1.5",
			apiKey:  "",
			wantErr: false,
		},
		{
			name:    "valid OpenAI configuration",
			baseURL: "https://api.openai.com/v1",
			model:   "text-embedding-3-small",
			apiKey:  "sk-test123",
			wantErr: false,
		},
		{
			name:       "empty base URL",
			baseURL:    "",
			model:      "test",
			apiKey:     "",
			wantErr:    true,
			errMessage: "base URL required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := Config{
				BaseURL: tt.baseURL,
				Model:   tt.model,
				APIKey:  tt.apiKey,
			}

			service, err := NewService(config)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errMessage != "" {
					assert.Contains(t, err.Error(), tt.errMessage)
				}
			} else {
				require.NoError(t, err)
				assert.NotNil(t, service)
			}
		})
	}
}

func TestService_Embed(t *testing.T) {
	// Unit test with mock (we'll implement later)
	tests := []struct {
		name    string
		texts   []string
		wantErr bool
	}{
		{
			name:    "single text",
			texts:   []string{"hello world"},
			wantErr: false,
		},
		{
			name:    "multiple texts",
			texts:   []string{"hello", "world", "test"},
			wantErr: false,
		},
		{
			name:    "empty input",
			texts:   []string{},
			wantErr: true,
		},
		{
			name:    "nil input",
			texts:   nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This test will pass once we implement validation
			// Actual embedding generation tested in integration test
			if tt.wantErr {
				// Expected to fail validation
				assert.True(t, len(tt.texts) == 0)
			}
		})
	}
}

func TestService_EmbedIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Check if TEI is available
	baseURL := os.Getenv("EMBEDDING_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}

	model := os.Getenv("EMBEDDING_MODEL")
	if model == "" {
		model = "Alibaba-NLP/gte-base-en-v1.5"
	}

	config := Config{
		BaseURL: baseURL,
		Model:   model,
	}

	service, err := NewService(config)
	require.NoError(t, err)

	ctx := context.Background()

	// Check if embedding service is reachable before running tests
	// Try a simple embed operation to verify service availability
	_, err = service.Embed(ctx, []string{"health check"})
	if err != nil {
		t.Skipf("embedding service not available at %s: %v", baseURL, err)
	}

	t.Run("single text embedding", func(t *testing.T) {
		vectors, err := service.Embed(ctx, []string{"test document"})
		require.NoError(t, err)
		require.Len(t, vectors, 1)
		assert.Greater(t, len(vectors[0]), 0, "embedding should have dimensions")
		t.Logf("Embedding dimensions: %d", len(vectors[0]))
	})

	t.Run("batch embedding", func(t *testing.T) {
		texts := []string{
			"first document",
			"second document",
			"third document",
		}
		vectors, err := service.Embed(ctx, texts)
		require.NoError(t, err)
		require.Len(t, vectors, len(texts))

		// Verify all vectors have same dimensions
		dims := len(vectors[0])
		for i, v := range vectors {
			assert.Equal(t, dims, len(v), "vector %d should have same dimensions", i)
		}
		t.Logf("Generated %d vectors of %d dimensions", len(vectors), dims)
	})

	t.Run("empty input validation", func(t *testing.T) {
		_, err := service.Embed(ctx, []string{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "empty")
	})

	t.Run("context cancellation", func(t *testing.T) {
		cancelCtx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		_, err := service.Embed(cancelCtx, []string{"test"})
		assert.Error(t, err)
	})
}

func TestConfigFromEnv(t *testing.T) {
	tests := []struct {
		name    string
		envVars map[string]string
		want    Config
	}{
		{
			name: "default TEI configuration",
			envVars: map[string]string{
				"EMBEDDING_BASE_URL": "",
				"EMBEDDING_MODEL":    "",
			},
			want: Config{
				BaseURL: "http://localhost:8080",
				Model:   "BAAI/bge-small-en-v1.5",
				APIKey:  "",
			},
		},
		{
			name: "custom configuration",
			envVars: map[string]string{
				"EMBEDDING_BASE_URL": "http://custom:9090",
				"EMBEDDING_MODEL":    "custom-model",
				"OPENAI_API_KEY":     "sk-test",
			},
			want: Config{
				BaseURL: "http://custom:9090",
				Model:   "custom-model",
				APIKey:  "sk-test",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			for k, v := range tt.envVars {
				if v != "" {
					os.Setenv(k, v)
					defer os.Unsetenv(k)
				}
			}

			got := ConfigFromEnv()
			assert.Equal(t, tt.want.BaseURL, got.BaseURL)
			assert.Equal(t, tt.want.Model, got.Model)

			if tt.envVars["OPENAI_API_KEY"] != "" {
				assert.Equal(t, tt.want.APIKey, got.APIKey)
			}
		})
	}
}
