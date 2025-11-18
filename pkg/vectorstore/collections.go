package vectorstore

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

var (
	// ErrCollectionExists indicates the collection already exists
	ErrCollectionExists = errors.New("collection already exists")

	// ErrCollectionNotFound indicates the collection doesn't exist
	ErrCollectionNotFound = errors.New("collection not found")

	// ErrInvalidVectorSize indicates invalid vector dimensions
	ErrInvalidVectorSize = errors.New("invalid vector size: must be positive")
)

// CollectionInfo holds metadata about a collection.
type CollectionInfo struct {
	Name       string `json:"name"`
	VectorSize int    `json:"vector_size"`
	PointCount int    `json:"point_count"`
}

// CreateCollection creates a new collection in Qdrant.
//
// The collection is created with the specified vector dimensions.
// If the collection already exists, returns ErrCollectionExists.
//
// Parameters:
//   - ctx: Context for cancellation
//   - collectionName: Name of the collection to create
//   - vectorSize: Dimension of vectors (e.g., 384 for BGE-small, 1536 for OpenAI)
//
// Returns:
//   - Error if creation fails or collection already exists
func (s *Service) CreateCollection(ctx context.Context, collectionName string, vectorSize int) error {
	// Validate inputs
	if collectionName == "" {
		return fmt.Errorf("%w: collection name required", ErrInvalidConfig)
	}
	if vectorSize <= 0 {
		return fmt.Errorf("%w: got %d", ErrInvalidVectorSize, vectorSize)
	}

	// Check if collection already exists
	exists, err := s.CollectionExists(ctx, collectionName)
	if err != nil {
		return fmt.Errorf("checking collection existence: %w", err)
	}
	if exists {
		return fmt.Errorf("%w: %s", ErrCollectionExists, collectionName)
	}

	// Create collection via Qdrant HTTP API
	// Ref: https://qdrant.github.io/qdrant/redoc/index.html#tag/collections/operation/create_collection
	url := fmt.Sprintf("%s/collections/%s", s.config.URL, collectionName)

	// Prepare request body
	body := map[string]interface{}{
		"vectors": map[string]interface{}{
			"size":     vectorSize,
			"distance": "Cosine",
		},
	}
	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshaling request body: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(bodyJSON))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Execute request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	// Check response
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

// DeleteCollection deletes a collection from Qdrant.
//
// If the collection doesn't exist, returns ErrCollectionNotFound.
//
// Parameters:
//   - ctx: Context for cancellation
//   - collectionName: Name of the collection to delete
//
// Returns:
//   - Error if deletion fails or collection doesn't exist
func (s *Service) DeleteCollection(ctx context.Context, collectionName string) error {
	// Validate input
	if collectionName == "" {
		return fmt.Errorf("%w: collection name required", ErrInvalidConfig)
	}

	// Check if collection exists
	exists, err := s.CollectionExists(ctx, collectionName)
	if err != nil {
		return fmt.Errorf("checking collection existence: %w", err)
	}
	if !exists {
		return fmt.Errorf("%w: %s", ErrCollectionNotFound, collectionName)
	}

	// Delete collection via Qdrant HTTP API
	// Ref: https://qdrant.github.io/qdrant/redoc/index.html#tag/collections/operation/delete_collection
	url := fmt.Sprintf("%s/collections/%s", s.config.URL, collectionName)

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	// Check response
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

// ListCollections lists all collections in Qdrant.
//
// Returns:
//   - Slice of collection names
//   - Error if listing fails
func (s *Service) ListCollections(ctx context.Context) ([]string, error) {
	// List collections via Qdrant HTTP API
	// Ref: https://qdrant.github.io/qdrant/redoc/index.html#tag/collections/operation/get_collections
	url := fmt.Sprintf("%s/collections", s.config.URL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	// Check response
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Parse response
	var result struct {
		Result struct {
			Collections []struct {
				Name string `json:"name"`
			} `json:"collections"`
		} `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	// Extract collection names
	collections := make([]string, len(result.Result.Collections))
	for i, col := range result.Result.Collections {
		collections[i] = col.Name
	}

	return collections, nil
}

// CollectionExists checks if a collection exists in Qdrant.
//
// Parameters:
//   - ctx: Context for cancellation
//   - collectionName: Name of the collection to check
//
// Returns:
//   - true if collection exists, false otherwise
//   - Error if check fails
func (s *Service) CollectionExists(ctx context.Context, collectionName string) (bool, error) {
	if collectionName == "" {
		return false, fmt.Errorf("%w: collection name required", ErrInvalidConfig)
	}

	// Check if collection exists by attempting to get collection info
	// This is done via the Qdrant HTTP API
	url := fmt.Sprintf("%s/collections/%s", s.config.URL, collectionName)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false, fmt.Errorf("creating request: %w", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Errorf("checking collection: %w", err)
	}
	defer resp.Body.Close()

	// 200 OK = collection exists
	// 404 Not Found = collection doesn't exist
	// Other errors = actual error
	switch resp.StatusCode {
	case http.StatusOK:
		return true, nil
	case http.StatusNotFound:
		return false, nil
	default:
		return false, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
}

// GetCollectionInfo retrieves metadata about a collection.
//
// Parameters:
//   - ctx: Context for cancellation
//   - collectionName: Name of the collection
//
// Returns:
//   - Collection metadata (name, vector size, point count)
//   - Error if collection doesn't exist or retrieval fails
func (s *Service) GetCollectionInfo(ctx context.Context, collectionName string) (*CollectionInfo, error) {
	if collectionName == "" {
		return nil, fmt.Errorf("%w: collection name required", ErrInvalidConfig)
	}

	// Check if collection exists
	exists, err := s.CollectionExists(ctx, collectionName)
	if err != nil {
		return nil, fmt.Errorf("checking collection existence: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrCollectionNotFound, collectionName)
	}

	// Get collection info via Qdrant HTTP API
	// Ref: https://qdrant.github.io/qdrant/redoc/index.html#tag/collections/operation/get_collection
	url := fmt.Sprintf("%s/collections/%s", s.config.URL, collectionName)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	// Check response
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Parse response
	var result struct {
		Result struct {
			Config struct {
				Params struct {
					Vectors struct {
						Size int `json:"size"`
					} `json:"vectors"`
				} `json:"params"`
			} `json:"config"`
			PointsCount int `json:"points_count"`
		} `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	info := &CollectionInfo{
		Name:       collectionName,
		VectorSize: result.Result.Config.Params.Vectors.Size,
		PointCount: result.Result.PointsCount,
	}

	return info, nil
}
