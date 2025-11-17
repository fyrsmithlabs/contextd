// Package embeddings provides embedding generation via langchaingo.
//
// This package wraps langchaingo's embedding functionality to generate
// vector embeddings for text content. It supports both local TEI
// (Text Embeddings Inference) servers and OpenAI's embedding API.
//
// Example usage with TEI:
//
//	config := embeddings.Config{
//	    BaseURL: "http://localhost:8080",
//	    Model:   "BAAI/bge-small-en-v1.5",
//	}
//	service, err := embeddings.NewService(config)
//	if err != nil {
//	    // Handle error
//	}
//	vectors, err := service.Embed(ctx, []string{"text1", "text2"})
//
// Example usage with OpenAI:
//
//	config := embeddings.Config{
//	    BaseURL: "https://api.openai.com/v1",
//	    Model:   "text-embedding-3-small",
//	    APIKey:  os.Getenv("OPENAI_API_KEY"),
//	}
//	service, err := embeddings.NewService(config)
package embeddings

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/llms/openai"
)

var (
	// ErrEmptyInput indicates empty or nil input texts
	ErrEmptyInput = errors.New("empty or nil input texts")

	// ErrInvalidConfig indicates invalid configuration
	ErrInvalidConfig = errors.New("invalid configuration")
)

// Config holds configuration for the embedding service.
type Config struct {
	// BaseURL is the base URL for the embedding API
	// For TEI: http://localhost:8080
	// For OpenAI: https://api.openai.com/v1
	BaseURL string

	// Model is the embedding model to use
	// For TEI: BAAI/bge-small-en-v1.5, Alibaba-NLP/gte-base-en-v1.5
	// For OpenAI: text-embedding-3-small, text-embedding-3-large
	Model string

	// APIKey is the API key (required for OpenAI, optional for TEI)
	APIKey string
}

// ConfigFromEnv creates a Config from environment variables.
//
// Environment variables:
//   - EMBEDDING_BASE_URL: Base URL (default: http://localhost:8080/v1)
//   - EMBEDDING_MODEL: Model name (default: BAAI/bge-small-en-v1.5)
//   - OPENAI_API_KEY: OpenAI API key (optional)
func ConfigFromEnv() Config {
	baseURL := os.Getenv("EMBEDDING_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080/v1"
	}

	model := os.Getenv("EMBEDDING_MODEL")
	if model == "" {
		model = "BAAI/bge-small-en-v1.5"
	}

	apiKey := os.Getenv("OPENAI_API_KEY")

	return Config{
		BaseURL: baseURL,
		Model:   model,
		APIKey:  apiKey,
	}
}

// Validate validates the configuration.
func (c Config) Validate() error {
	if c.BaseURL == "" {
		return fmt.Errorf("%w: base URL required", ErrInvalidConfig)
	}
	if c.Model == "" {
		return fmt.Errorf("%w: model required", ErrInvalidConfig)
	}
	return nil
}

// Service provides embedding generation functionality.
type Service struct {
	embedder *embeddings.EmbedderImpl
	config   Config
}

// NewService creates a new embedding service with the given configuration.
//
// The service uses langchaingo's embeddings abstraction to support
// multiple embedding providers (TEI, OpenAI, etc.).
//
// Returns an error if the configuration is invalid.
func NewService(config Config) (*Service, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("validating config: %w", err)
	}

	// Create OpenAI client with custom base URL
	// This works for both OpenAI API and TEI (OpenAI-compatible)
	apiKey := config.APIKey
	if apiKey == "" {
		// langchaingo requires a token, use placeholder for TEI
		apiKey = "placeholder"
	}

	opts := []openai.Option{
		openai.WithBaseURL(config.BaseURL),
		openai.WithModel(config.Model),
		openai.WithToken(apiKey),
	}

	llm, err := openai.New(opts...)
	if err != nil {
		return nil, fmt.Errorf("creating OpenAI client: %w", err)
	}

	embedder, err := embeddings.NewEmbedder(llm)
	if err != nil {
		return nil, fmt.Errorf("creating embedder: %w", err)
	}

	return &Service{
		embedder: embedder,
		config:   config,
	}, nil
}

// Embedder returns the underlying langchaingo Embedder.
//
// This allows the service to be used with other langchaingo components
// that require an Embedder interface (e.g., vector stores).
func (s *Service) Embedder() embeddings.Embedder {
	return s.embedder
}

// Embed generates embeddings for the given texts.
//
// It accepts a slice of text strings and returns a slice of float32
// vectors, one for each input text. All vectors have the same dimensions.
//
// The method validates input (non-empty), respects context cancellation,
// and handles errors from the underlying embedding provider.
//
// Returns ErrEmptyInput if texts is empty or nil.
func (s *Service) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	// Validate input
	if len(texts) == 0 {
		return nil, fmt.Errorf("%w: texts cannot be empty", ErrEmptyInput)
	}

	// Generate embeddings using langchaingo
	vectors, err := s.embedder.EmbedDocuments(ctx, texts)
	if err != nil {
		return nil, fmt.Errorf("embedding documents: %w", err)
	}

	return vectors, nil
}
