// Package embeddings provides embedding generation via TEI.
package embeddings

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/fyrsmithlabs/contextd/internal/vectorstore"
)

var (
	// ErrEmptyInput indicates empty or nil input texts
	ErrEmptyInput = errors.New("empty or nil input texts")

	// ErrInvalidConfig indicates invalid configuration
	ErrInvalidConfig = errors.New("invalid configuration")

	// ErrEmbeddingFailed indicates embedding generation failure
	ErrEmbeddingFailed = errors.New("embedding generation failed")
)

// Config holds configuration for the embedding service.
type Config struct {
	// BaseURL is the base URL for the embedding API
	BaseURL string

	// Model is the embedding model to use
	Model string

	// APIKey is the API key (optional for TEI)
	APIKey string
}

// ConfigFromEnv creates a Config from environment variables.
func ConfigFromEnv() Config {
	baseURL := os.Getenv("EMBEDDING_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
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
	return nil
}

// Service provides embedding generation functionality.
type Service struct {
	config Config
	client *http.Client
}

// NewService creates a new embedding service with the given configuration.
func NewService(config Config) (*Service, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("validating config: %w", err)
	}

	return &Service{
		config: config,
		client: &http.Client{},
	}, nil
}

// teiRequest is the request body for TEI embed endpoint.
type teiRequest struct {
	Inputs   interface{} `json:"inputs"`
	Truncate bool        `json:"truncate"`
}

// Embedder returns an Embedder interface implementation.
func (s *Service) Embedder() vectorstore.Embedder {
	return s
}

// EmbedDocuments generates embeddings for multiple texts.
func (s *Service) EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, fmt.Errorf("%w: texts cannot be empty", ErrEmptyInput)
	}

	req := teiRequest{
		Inputs:   texts,
		Truncate: true,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", s.config.BaseURL+"/embed", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrEmbeddingFailed, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("%w: status %d: %s", ErrEmbeddingFailed, resp.StatusCode, string(respBody))
	}

	var vectors [][]float32
	if err := json.NewDecoder(resp.Body).Decode(&vectors); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return vectors, nil
}

// EmbedQuery generates an embedding for a single query.
func (s *Service) EmbedQuery(ctx context.Context, text string) ([]float32, error) {
	if text == "" {
		return nil, fmt.Errorf("%w: text cannot be empty", ErrEmptyInput)
	}

	req := teiRequest{
		Inputs:   text,
		Truncate: true,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", s.config.BaseURL+"/embed", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrEmbeddingFailed, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("%w: status %d: %s", ErrEmbeddingFailed, resp.StatusCode, string(respBody))
	}

	var vectors [][]float32
	if err := json.NewDecoder(resp.Body).Decode(&vectors); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	if len(vectors) == 0 {
		return nil, fmt.Errorf("%w: empty response", ErrEmbeddingFailed)
	}

	return vectors[0], nil
}

// Embed generates embeddings for the given texts (legacy method).
func (s *Service) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	return s.EmbedDocuments(ctx, texts)
}
