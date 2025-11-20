package vectorstore

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"sort"
)

// ExactSearch performs brute-force cosine similarity search for small datasets.
//
// This is used as a fallback when Qdrant's HNSW index isn't available (<10 vectors).
// It retrieves all points from the collection, computes cosine similarity manually,
// and returns the top-k most similar results.
//
// Performance: <10ms for collections with <10 vectors (acceptable for small datasets).
//
// BUG-2025-11-20-005: Qdrant requires ≥10 vectors for HNSW index to work.
// Collections with <10 vectors return 0 results even if data exists.
// This method provides exact search fallback to fix that issue.
func (s *Service) ExactSearch(ctx context.Context, collectionName string, query string, k int) ([]SearchResult, error) {
	if collectionName == "" {
		return nil, fmt.Errorf("%w: collection name required", ErrInvalidConfig)
	}
	if query == "" {
		return nil, fmt.Errorf("%w: query required", ErrInvalidConfig)
	}
	if k <= 0 {
		k = 10
	}

	// Step 1: Generate embedding for the query
	queryEmbedding, err := s.generateEmbedding(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("generating query embedding: %w", err)
	}

	// Step 2: Retrieve all points from the collection using Qdrant scroll API
	points, err := s.scrollAllPoints(ctx, collectionName)
	if err != nil {
		return nil, fmt.Errorf("scrolling collection points: %w", err)
	}

	// Step 3: Compute cosine similarity for each point
	type scoredPoint struct {
		point SearchResult
		score float32
	}
	scoredPoints := make([]scoredPoint, 0, len(points))

	for _, point := range points {
		// Extract vector from point
		vector, ok := point["vector"].([]interface{})
		if !ok {
			continue // Skip points without vectors
		}

		// Convert vector to []float64
		pointVector := make([]float64, len(vector))
		for i, v := range vector {
			if f, ok := v.(float64); ok {
				pointVector[i] = f
			}
		}

		// Compute cosine similarity
		similarity := cosineSimilarity(queryEmbedding, pointVector)

		// Extract metadata
		id, _ := point["id"].(string)
		payload, _ := point["payload"].(map[string]interface{})

		// Create search result
		result := SearchResult{
			ID:       id,
			Score:    similarity,
			Metadata: payload,
		}

		// Extract content from payload if available
		if content, ok := payload["content"].(string); ok {
			result.Content = content
		}

		scoredPoints = append(scoredPoints, scoredPoint{
			point: result,
			score: similarity,
		})
	}

	// Step 4: Sort by similarity score (descending)
	sort.Slice(scoredPoints, func(i, j int) bool {
		return scoredPoints[i].score > scoredPoints[j].score
	})

	// Step 5: Return top-k results
	limit := k
	if len(scoredPoints) < limit {
		limit = len(scoredPoints)
	}

	results := make([]SearchResult, 0, limit)
	for i := 0; i < limit; i++ {
		results = append(results, scoredPoints[i].point)
	}

	return results, nil
}

// scrollAllPoints retrieves all points from a collection using Qdrant's scroll API.
//
// Reference: https://qdrant.tech/documentation/concepts/points/#scroll-points
func (s *Service) scrollAllPoints(ctx context.Context, collectionName string) ([]map[string]interface{}, error) {
	url := fmt.Sprintf("%s/collections/%s/points/scroll", s.config.URL, collectionName)

	// Request all points with payload and vector
	reqBody := map[string]interface{}{
		"with_payload": true,
		"with_vector":  true,
		"limit":        100, // Qdrant scroll limit
	}

	reqBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, io.NopCloser(bytes.NewReader(reqBytes)))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Parse response
	var result struct {
		Result struct {
			Points []map[string]interface{} `json:"points"`
		} `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return result.Result.Points, nil
}

// cosineSimilarity computes cosine similarity between two vectors.
//
// Formula: cos(θ) = (A · B) / (||A|| * ||B||)
// Returns value in range [-1, 1] where 1 = identical, 0 = orthogonal, -1 = opposite
func cosineSimilarity(a, b []float64) float32 {
	if len(a) != len(b) {
		return 0
	}

	// Compute dot product
	var dotProduct float64
	for i := range a {
		dotProduct += a[i] * b[i]
	}

	// Compute magnitudes
	var magA, magB float64
	for i := range a {
		magA += a[i] * a[i]
		magB += b[i] * b[i]
	}

	magA = math.Sqrt(magA)
	magB = math.Sqrt(magB)

	// Avoid division by zero
	if magA == 0 || magB == 0 {
		return 0
	}

	return float32(dotProduct / (magA * magB))
}

// generateEmbedding generates an embedding vector for the given text.
func (s *Service) generateEmbedding(ctx context.Context, text string) ([]float64, error) {
	if s.config.Embedder == nil {
		return nil, fmt.Errorf("%w: embedder not configured", ErrInvalidConfig)
	}

	// Use the configured embedder
	embeddings, err := s.config.Embedder.EmbedDocuments(ctx, []string{text})
	if err != nil {
		return nil, fmt.Errorf("embedding text: %w", err)
	}

	if len(embeddings) == 0 {
		return nil, fmt.Errorf("no embeddings generated")
	}

	// Convert []float32 to []float64
	vector := make([]float64, len(embeddings[0]))
	for i, v := range embeddings[0] {
		vector[i] = float64(v)
	}

	return vector, nil
}
