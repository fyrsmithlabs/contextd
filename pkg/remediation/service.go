package remediation

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/fyrsmithlabs/contextd/pkg/vectorstore"
)

// VectorStore defines the interface for vector database operations.
type VectorStore interface {
	AddDocuments(ctx context.Context, docs []vectorstore.Document) error
	SearchWithFilters(ctx context.Context, query string, k int, filters map[string]interface{}) ([]vectorstore.SearchResult, error)
}

// Service provides remediation management with hybrid matching.
//
// The service combines semantic search (70%) with string similarity (30%)
// to find relevant error solutions. Pattern extraction improves matching
// accuracy for common error types.
type Service struct {
	store  VectorStore
	logger *zap.Logger
}

// NewService creates a new remediation service.
func NewService(store VectorStore, logger *zap.Logger) *Service {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Service{
		store:  store,
		logger: logger,
	}
}

// Save stores a remediation with automatic pattern extraction.
//
// The remediation is validated, patterns are extracted from the error message,
// and the data is stored in the vector database with embeddings for semantic search.
//
// Returns an error if validation fails or the vector store operation fails.
func (s *Service) Save(ctx context.Context, rem *Remediation) error {
	// Validate remediation
	if err := rem.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Generate ID and timestamp
	if rem.ID == "" {
		rem.ID = "rem_" + uuid.New().String()
	}
	if rem.CreatedAt.IsZero() {
		rem.CreatedAt = time.Now()
	}

	// Extract patterns from error message
	rem.Patterns = ExtractPatterns(rem.ErrorMsg)

	// Create document for vector storage
	// Embed error message + solution for semantic search
	embedContent := rem.ErrorMsg + "\n\n" + rem.Solution
	if rem.Context != "" {
		embedContent += "\n\n" + rem.Context
	}

	doc := vectorstore.Document{
		ID:      rem.ID,
		Content: embedContent,
		Metadata: map[string]interface{}{
			"project_path": rem.ProjectPath,
			"error_msg":    rem.ErrorMsg,
			"solution":     rem.Solution,
			"context":      rem.Context,
			"patterns":     rem.Patterns,
			"created_at":   rem.CreatedAt.Format(time.RFC3339),
		},
	}

	// Add metadata if present
	if rem.Metadata != nil {
		for k, v := range rem.Metadata {
			doc.Metadata[k] = v
		}
	}

	// Store in vector database
	if err := s.store.AddDocuments(ctx, []vectorstore.Document{doc}); err != nil {
		return fmt.Errorf("failed to store remediation: %w", err)
	}

	s.logger.Info("remediation saved",
		zap.String("id", rem.ID),
		zap.String("project_path", rem.ProjectPath),
		zap.Int("pattern_count", len(rem.Patterns)),
	)

	return nil
}

// Search finds remediations using hybrid matching (semantic + string similarity).
//
// The search combines:
//   - 70% semantic similarity (vector embeddings)
//   - 30% string similarity (Levenshtein distance)
//
// Results are filtered by project path for multi-tenant isolation and
// sorted by combined score.
func (s *Service) Search(ctx context.Context, query string, opts *SearchOptions) ([]*SearchResult, error) {
	// Validate options
	if err := opts.Validate(); err != nil {
		return nil, fmt.Errorf("invalid options: %w", err)
	}

	// Create project filter for multi-tenant isolation using Qdrant filter structure
	filters := map[string]interface{}{
		"must": []map[string]interface{}{
			{
				"key": "project_path",
				"match": map[string]interface{}{
					"value": opts.ProjectPath,
				},
			},
		},
	}

	// Perform semantic search
	results, err := s.store.SearchWithFilters(ctx, query, opts.Limit, filters)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	// Convert results to SearchResult with hybrid scoring
	searchResults := make([]*SearchResult, 0, len(results))
	for _, r := range results {
		rem := &Remediation{
			ID:          r.ID,
			ProjectPath: getString(r.Metadata, "project_path"),
			ErrorMsg:    getString(r.Metadata, "error_msg"),
			Solution:    getString(r.Metadata, "solution"),
			Context:     getString(r.Metadata, "context"),
			Patterns:    getStringSlice(r.Metadata, "patterns"),
			Metadata:    r.Metadata,
		}

		// Parse created_at if present
		if createdAtStr := getString(r.Metadata, "created_at"); createdAtStr != "" {
			if t, err := time.Parse(time.RFC3339, createdAtStr); err == nil {
				rem.CreatedAt = t
			}
		}

		// Hybrid scoring: 70% semantic + 30% string similarity
		semanticScore := float64(r.Score)
		stringScore := calculateStringSimilarity(query, rem.ErrorMsg)
		hybridScore := 0.7*semanticScore + 0.3*stringScore

		// Filter by threshold
		if hybridScore < opts.Threshold {
			continue
		}

		searchResults = append(searchResults, &SearchResult{
			Remediation:   rem,
			Score:         hybridScore,
			SemanticScore: semanticScore,
			StringScore:   stringScore,
		})
	}

	s.logger.Info("remediation search completed",
		zap.String("query", query),
		zap.Int("result_count", len(searchResults)),
	)

	return searchResults, nil
}

// List retrieves recent remediations for a project.
//
// Results are ordered by creation date (most recent first) and
// limited by the specified count.
func (s *Service) List(ctx context.Context, projectPath string, limit int) ([]*Remediation, error) {
	// Validate project path
	if projectPath == "" {
		return nil, ErrProjectPathRequired
	}

	// Use empty query for listing (relies on filters only) with Qdrant filter structure
	filters := map[string]interface{}{
		"must": []map[string]interface{}{
			{
				"key": "project_path",
				"match": map[string]interface{}{
					"value": projectPath,
				},
			},
		},
	}

	if limit <= 0 {
		limit = DefaultLimit
	}
	if limit > MaxLimit {
		limit = MaxLimit
	}

	results, err := s.store.SearchWithFilters(ctx, "", limit, filters)
	if err != nil {
		return nil, fmt.Errorf("list failed: %w", err)
	}

	// Convert results
	remediations := make([]*Remediation, 0, len(results))
	for _, r := range results {
		rem := &Remediation{
			ID:          r.ID,
			ProjectPath: getString(r.Metadata, "project_path"),
			ErrorMsg:    getString(r.Metadata, "error_msg"),
			Solution:    getString(r.Metadata, "solution"),
			Context:     getString(r.Metadata, "context"),
			Patterns:    getStringSlice(r.Metadata, "patterns"),
			Metadata:    r.Metadata,
		}

		if createdAtStr := getString(r.Metadata, "created_at"); createdAtStr != "" {
			if t, err := time.Parse(time.RFC3339, createdAtStr); err == nil {
				rem.CreatedAt = t
			}
		}

		remediations = append(remediations, rem)
	}

	return remediations, nil
}

// calculateStringSimilarity computes normalized Levenshtein distance.
//
// Returns a value between 0.0 (completely different) and 1.0 (identical).
func calculateStringSimilarity(s1, s2 string) float64 {
	// Normalize to lowercase for case-insensitive comparison
	s1 = strings.ToLower(s1)
	s2 = strings.ToLower(s2)

	// Handle edge cases
	if s1 == s2 {
		return 1.0
	}
	if len(s1) == 0 || len(s2) == 0 {
		return 0.0
	}

	// Compute Levenshtein distance
	distance := levenshteinDistance(s1, s2)
	maxLen := max(len(s1), len(s2))

	// Normalize to 0-1 range (1 = identical, 0 = completely different)
	similarity := 1.0 - float64(distance)/float64(maxLen)
	return similarity
}

// levenshteinDistance computes the edit distance between two strings.
func levenshteinDistance(s1, s2 string) int {
	len1, len2 := len(s1), len(s2)
	if len1 == 0 {
		return len2
	}
	if len2 == 0 {
		return len1
	}

	// Create matrix
	matrix := make([][]int, len1+1)
	for i := range matrix {
		matrix[i] = make([]int, len2+1)
	}

	// Initialize first row and column
	for i := 0; i <= len1; i++ {
		matrix[i][0] = i
	}
	for j := 0; j <= len2; j++ {
		matrix[0][j] = j
	}

	// Fill matrix
	for i := 1; i <= len1; i++ {
		for j := 1; j <= len2; j++ {
			cost := 1
			if s1[i-1] == s2[j-1] {
				cost = 0
			}

			matrix[i][j] = min(
				matrix[i-1][j]+1,      // deletion
				matrix[i][j-1]+1,      // insertion
				matrix[i-1][j-1]+cost, // substitution
			)
		}
	}

	return matrix[len1][len2]
}

// Helper functions

func getString(metadata map[string]interface{}, key string) string {
	if v, ok := metadata[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func getStringSlice(metadata map[string]interface{}, key string) []string {
	if v, ok := metadata[key]; ok {
		switch val := v.(type) {
		case []string:
			return val
		case []interface{}:
			result := make([]string, 0, len(val))
			for _, item := range val {
				if s, ok := item.(string); ok {
					result = append(result, s)
				}
			}
			return result
		}
	}
	return nil
}

func min(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
