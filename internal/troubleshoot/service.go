// Package troubleshoot provides AI-powered error diagnosis and pattern recognition.
//
// This package analyzes error messages using AI and known patterns to provide
// root cause analysis and remediation suggestions. Patterns are stored in a
// shared database for team-wide knowledge sharing.
package troubleshoot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"

	"github.com/fyrsmithlabs/contextd/internal/vectorstore"
)

var tracer = otel.Tracer("contextd/troubleshoot")

// VectorStore defines the interface for vector database operations.
type VectorStore interface {
	AddDocuments(ctx context.Context, docs []vectorstore.Document) error
	SearchWithFilters(ctx context.Context, query string, k int, filters map[string]interface{}) ([]vectorstore.SearchResult, error)
}

// AIClient defines the interface for AI text generation.
type AIClient interface {
	Generate(ctx context.Context, prompt string) (string, error)
}

// Service provides AI-powered error diagnosis.
type Service struct {
	store    VectorStore
	logger   *zap.Logger
	aiClient AIClient
	tracer   trace.Tracer
}

// NewService creates a new troubleshoot service.
//
// The service uses vector storage for pattern matching and an optional
// AI client for hypothesis generation. If aiClient is nil, the service
// will only use pattern-based diagnosis.
func NewService(store VectorStore, logger *zap.Logger, aiClient AIClient) (*Service, error) {
	if store == nil {
		return nil, errors.New("vector store is required for troubleshoot service")
	}
	if logger == nil {
		return nil, errors.New("logger is required for troubleshoot service")
	}
	return &Service{
		store:    store,
		logger:   logger,
		aiClient: aiClient,
		tracer:   tracer,
	}, nil
}

// SavePattern stores an error pattern for future matching.
//
// Patterns are stored in the shared database with embeddings for semantic
// search. The pattern is validated before storage.
func (s *Service) SavePattern(ctx context.Context, pattern *Pattern) error {
	ctx, span := s.tracer.Start(ctx, "Service.SavePattern")
	defer span.End()

	// Validate pattern
	if pattern == nil {
		return errors.New("pattern cannot be nil")
	}
	if err := pattern.Validate(); err != nil {
		return err
	}

	// Generate ID if not provided
	if pattern.ID == "" {
		pattern.ID = "pattern_" + uuid.New().String()
	}

	// Set timestamp
	if pattern.CreatedAt.IsZero() {
		pattern.CreatedAt = time.Now()
	}

	// Default confidence if not set
	if pattern.Confidence == 0.0 {
		pattern.Confidence = 0.5
	}

	// Create document for vector storage
	// Embed error type + description for semantic search
	embedContent := fmt.Sprintf("%s: %s", pattern.ErrorType, pattern.Description)

	doc := vectorstore.Document{
		ID:      pattern.ID,
		Content: embedContent,
		Metadata: map[string]interface{}{
			"error_type":  pattern.ErrorType,
			"description": pattern.Description,
			"solution":    pattern.Solution,
			"confidence":  pattern.Confidence,
			"frequency":   pattern.Frequency,
			"created_at":  pattern.CreatedAt.Format(time.RFC3339),
		},
	}

	// Store in vector database
	if err := s.store.AddDocuments(ctx, []vectorstore.Document{doc}); err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to store pattern: %w", err)
	}

	s.logger.Info("pattern saved",
		zap.String("id", pattern.ID),
		zap.String("error_type", pattern.ErrorType),
		zap.Float64("confidence", pattern.Confidence),
	)

	return nil
}

// GetPatterns retrieves all known error patterns.
//
// Patterns are sorted by frequency (most common first) and returned
// with their metadata.
func (s *Service) GetPatterns(ctx context.Context) ([]Pattern, error) {
	ctx, span := s.tracer.Start(ctx, "Service.GetPatterns")
	defer span.End()

	// Use empty query to retrieve all patterns
	// Filters ensure we only get troubleshooting patterns
	filters := map[string]interface{}{
		"must": []map[string]interface{}{
			{
				"key": "error_type",
				"match": map[string]interface{}{
					"any": []string{}, // Match any error_type (just checking field exists)
				},
			},
		},
	}

	// Search with high limit to get all patterns
	results, err := s.store.SearchWithFilters(ctx, "error", 1000, filters)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to retrieve patterns: %w", err)
	}

	// Convert results to patterns
	patterns := make([]Pattern, 0, len(results))
	for _, result := range results {
		pattern, err := resultToPattern(result)
		if err != nil {
			s.logger.Warn("failed to convert pattern",
				zap.Error(err),
				zap.String("result_id", result.ID),
			)
			continue
		}
		patterns = append(patterns, pattern)
	}

	return patterns, nil
}

// Diagnose analyzes an error message and provides diagnosis.
//
// The diagnosis process:
// 1. Search known patterns for similar errors
// 2. If high-confidence match (>0.8) found, return pattern-based diagnosis
// 3. Otherwise, query AI for hypothesis generation
// 4. Combine pattern matches with AI hypotheses
// 5. Generate recommendations
func (s *Service) Diagnose(ctx context.Context, errorMsg, errorContext string) (*Diagnosis, error) {
	ctx, span := s.tracer.Start(ctx, "Service.Diagnose")
	defer span.End()

	// Validate input
	if errorMsg == "" {
		return nil, ErrEmptyErrorMessage
	}

	// 1. Search known patterns
	patterns, err := s.searchPatterns(ctx, errorMsg)
	if err != nil {
		span.RecordError(err)
		s.logger.Warn("pattern search failed",
			zap.Error(err),
			zap.String("error_msg", errorMsg),
		)
		// Continue with AI diagnosis even if pattern search fails
	}

	// 2. Check for high-confidence pattern match
	if len(patterns) > 0 && patterns[0].Confidence > 0.8 {
		diagnosis := s.buildDiagnosisFromPattern(patterns[0], patterns)
		s.logger.Info("high-confidence pattern match",
			zap.String("pattern_id", patterns[0].ID),
			zap.Float64("confidence", patterns[0].Confidence),
		)
		return diagnosis, nil
	}

	// 3. Query AI for hypothesis generation (if available)
	hypotheses := []Hypothesis{}  // Initialize as empty slice, not nil (for JSON encoding)
	recommendations := []string{} // Initialize as empty slice, not nil (for JSON encoding)
	var aiRootCause string

	if s.aiClient != nil {
		aiResponse, err := s.generateHypotheses(ctx, errorMsg, errorContext, patterns)
		if err != nil {
			span.RecordError(err)
			s.logger.Warn("AI hypothesis generation failed",
				zap.Error(err),
				zap.String("error_msg", errorMsg),
			)
			// Fallback to pattern-based diagnosis
			if len(patterns) > 0 {
				return s.buildDiagnosisFromPattern(patterns[0], patterns), nil
			}
			return nil, fmt.Errorf("failed to generate diagnosis: %w", err)
		}
		hypotheses = aiResponse.Hypotheses
		aiRootCause = aiResponse.RootCause
		recommendations = aiResponse.Recommendations
	}

	// 4. Build comprehensive diagnosis
	diagnosis := &Diagnosis{
		ErrorMessage:    errorMsg,
		RootCause:       aiRootCause,
		Hypotheses:      hypotheses,
		Recommendations: recommendations,
		RelatedPatterns: patterns,
		Confidence:      calculateConfidence(patterns, hypotheses),
	}

	// Add pattern-based recommendations if available
	if len(patterns) > 0 {
		diagnosis.Recommendations = append(diagnosis.Recommendations, patterns[0].Solution)
	}

	s.logger.Info("diagnosis generated",
		zap.String("error_msg", errorMsg),
		zap.Int("pattern_count", len(patterns)),
		zap.Int("hypothesis_count", len(hypotheses)),
		zap.Float64("confidence", diagnosis.Confidence),
	)

	return diagnosis, nil
}

// searchPatterns finds similar error patterns using semantic search.
func (s *Service) searchPatterns(ctx context.Context, errorMsg string) ([]Pattern, error) {
	ctx, span := s.tracer.Start(ctx, "searchPatterns")
	defer span.End()

	// Search for similar patterns
	filters := map[string]interface{}{
		"must": []map[string]interface{}{
			{
				"key": "error_type",
				"match": map[string]interface{}{
					"any": []string{}, // Match any error_type
				},
			},
		},
	}

	results, err := s.store.SearchWithFilters(ctx, errorMsg, 5, filters)
	if err != nil {
		return nil, fmt.Errorf("pattern search failed: %w", err)
	}

	// Convert results to patterns
	patterns := make([]Pattern, 0, len(results))
	for _, result := range results {
		pattern, err := resultToPattern(result)
		if err != nil {
			s.logger.Warn("failed to convert pattern",
				zap.Error(err),
				zap.String("result_id", result.ID),
			)
			continue
		}
		// Use search score as confidence for matched patterns
		pattern.Confidence = float64(result.Score)
		patterns = append(patterns, pattern)
	}

	return patterns, nil
}

// aiDiagnosisResponse represents the AI's diagnosis response.
type aiDiagnosisResponse struct {
	RootCause       string       `json:"root_cause"`
	Hypotheses      []Hypothesis `json:"hypotheses"`
	Recommendations []string     `json:"recommendations"`
}

// generateHypotheses uses AI to generate diagnostic hypotheses.
func (s *Service) generateHypotheses(ctx context.Context, errorMsg, errorContext string, patterns []Pattern) (*aiDiagnosisResponse, error) {
	ctx, span := s.tracer.Start(ctx, "generateHypotheses")
	defer span.End()

	// Build prompt for AI
	prompt := buildDiagnosticPrompt(errorMsg, errorContext, patterns)

	// Call AI
	responseText, err := s.aiClient.Generate(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("AI generation failed: %w", err)
	}

	// Parse AI response
	var response aiDiagnosisResponse
	if err := json.Unmarshal([]byte(responseText), &response); err != nil {
		return nil, fmt.Errorf("failed to parse AI response: %w", err)
	}

	return &response, nil
}

// buildDiagnosisFromPattern creates diagnosis from a pattern match.
func (s *Service) buildDiagnosisFromPattern(primary Pattern, related []Pattern) *Diagnosis {
	return &Diagnosis{
		ErrorMessage: primary.Description,
		RootCause:    primary.ErrorType,
		Hypotheses: []Hypothesis{
			{
				Description: primary.Description,
				Likelihood:  primary.Confidence,
				Evidence:    fmt.Sprintf("Pattern matched %d times previously", primary.Frequency),
			},
		},
		Recommendations: []string{primary.Solution},
		RelatedPatterns: related,
		Confidence:      primary.Confidence,
	}
}

// buildDiagnosticPrompt creates the AI prompt for diagnosis.
func buildDiagnosticPrompt(errorMsg, errorContext string, patterns []Pattern) string {
	var sb strings.Builder

	sb.WriteString("You are an expert software engineer diagnosing an error.\n\n")
	sb.WriteString(fmt.Sprintf("Error message: %s\n\n", errorMsg))

	if errorContext != "" {
		sb.WriteString(fmt.Sprintf("Context: %s\n\n", errorContext))
	}

	if len(patterns) > 0 {
		sb.WriteString("Similar known patterns:\n")
		for i, p := range patterns {
			if i >= 3 {
				break // Limit to top 3 patterns
			}
			sb.WriteString(fmt.Sprintf("- %s: %s (solution: %s)\n", p.ErrorType, p.Description, p.Solution))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("Provide a JSON response with:\n")
	sb.WriteString("1. root_cause: Brief description of the likely root cause\n")
	sb.WriteString("2. hypotheses: Array of possible causes with description, likelihood (0-1), and evidence\n")
	sb.WriteString("3. recommendations: Array of actionable steps to fix the error\n\n")
	sb.WriteString("Format: {\"root_cause\": \"...\", \"hypotheses\": [...], \"recommendations\": [...]}")

	return sb.String()
}

// calculateConfidence computes overall diagnosis confidence.
func calculateConfidence(patterns []Pattern, hypotheses []Hypothesis) float64 {
	if len(patterns) == 0 && len(hypotheses) == 0 {
		return 0.0
	}

	var total float64
	var count int

	// Factor in pattern confidence
	if len(patterns) > 0 {
		total += patterns[0].Confidence
		count++
	}

	// Factor in hypothesis likelihood
	for _, h := range hypotheses {
		total += h.Likelihood
		count++
	}

	if count == 0 {
		return 0.0
	}

	return total / float64(count)
}

// resultToPattern converts a search result to a Pattern.
func resultToPattern(result vectorstore.SearchResult) (Pattern, error) {
	pattern := Pattern{
		ID: result.ID,
	}

	// Extract required fields
	errorType, ok := result.Metadata["error_type"].(string)
	if !ok {
		return pattern, errors.New("missing or invalid error_type")
	}
	pattern.ErrorType = errorType

	description, ok := result.Metadata["description"].(string)
	if !ok {
		return pattern, errors.New("missing or invalid description")
	}
	pattern.Description = description

	solution, ok := result.Metadata["solution"].(string)
	if !ok {
		return pattern, errors.New("missing or invalid solution")
	}
	pattern.Solution = solution

	// Extract optional fields
	if confidence, ok := result.Metadata["confidence"].(float64); ok {
		pattern.Confidence = confidence
	}

	if frequency, ok := result.Metadata["frequency"].(int); ok {
		pattern.Frequency = frequency
	} else if frequency, ok := result.Metadata["frequency"].(float64); ok {
		pattern.Frequency = int(frequency)
	}

	// Parse timestamp
	if createdAtStr, ok := result.Metadata["created_at"].(string); ok {
		if t, err := time.Parse(time.RFC3339, createdAtStr); err == nil {
			pattern.CreatedAt = t
		}
	}

	return pattern, nil
}
