package handlers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/fyrsmithlabs/contextd/internal/troubleshoot"
)

// TroubleshootHandler wraps troubleshoot service for MCP tool interface.
type TroubleshootHandler struct {
	service *troubleshoot.Service
}

// NewTroubleshootHandler creates a new troubleshoot handler.
func NewTroubleshootHandler(service *troubleshoot.Service) *TroubleshootHandler {
	return &TroubleshootHandler{
		service: service,
	}
}

// TroubleshootInput represents input for troubleshoot tool.
type TroubleshootInput struct {
	ErrorMessage string `json:"error_message"`
	ErrorContext string `json:"error_context,omitempty"`
}

// Diagnose handles troubleshoot MCP tool call.
func (h *TroubleshootHandler) Diagnose(ctx context.Context, input json.RawMessage) (interface{}, error) {
	var req TroubleshootInput
	if err := json.Unmarshal(input, &req); err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}

	diagnosis, err := h.service.Diagnose(ctx, req.ErrorMessage, req.ErrorContext)
	if err != nil {
		return nil, fmt.Errorf("failed to diagnose error: %w", err)
	}

	// Convert hypotheses
	hypotheses := make([]map[string]interface{}, 0, len(diagnosis.Hypotheses))
	for _, h := range diagnosis.Hypotheses {
		hypotheses = append(hypotheses, map[string]interface{}{
			"description": h.Description,
			"likelihood":  h.Likelihood,
			"evidence":    h.Evidence,
		})
	}

	// Convert related patterns
	patterns := make([]map[string]interface{}, 0, len(diagnosis.RelatedPatterns))
	for _, p := range diagnosis.RelatedPatterns {
		patterns = append(patterns, map[string]interface{}{
			"id":          p.ID,
			"error_type":  p.ErrorType,
			"description": p.Description,
			"solution":    p.Solution,
			"frequency":   p.Frequency,
			"confidence":  p.Confidence,
			"created_at":  p.CreatedAt,
		})
	}

	return map[string]interface{}{
		"error_message":    diagnosis.ErrorMessage,
		"root_cause":       diagnosis.RootCause,
		"hypotheses":       hypotheses,
		"recommendations":  diagnosis.Recommendations,
		"related_patterns": patterns,
		"confidence":       diagnosis.Confidence,
	}, nil
}

// PatternSaveInput represents input for saving an error pattern.
type PatternSaveInput struct {
	ErrorType   string  `json:"error_type"`
	Description string  `json:"description"`
	Solution    string  `json:"solution"`
	Frequency   int     `json:"frequency,omitempty"`
	Confidence  float64 `json:"confidence,omitempty"`
}

// SavePattern handles pattern save MCP tool call.
func (h *TroubleshootHandler) SavePattern(ctx context.Context, input json.RawMessage) (interface{}, error) {
	var req PatternSaveInput
	if err := json.Unmarshal(input, &req); err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}

	pattern := &troubleshoot.Pattern{
		ErrorType:   req.ErrorType,
		Description: req.Description,
		Solution:    req.Solution,
		Frequency:   req.Frequency,
		Confidence:  req.Confidence,
	}

	if err := h.service.SavePattern(ctx, pattern); err != nil {
		return nil, fmt.Errorf("failed to save pattern: %w", err)
	}

	return map[string]interface{}{
		"id":         pattern.ID,
		"error_type": pattern.ErrorType,
		"confidence": pattern.Confidence,
		"created_at": pattern.CreatedAt,
	}, nil
}

// GetPatterns handles pattern retrieval MCP tool call.
func (h *TroubleshootHandler) GetPatterns(ctx context.Context, input json.RawMessage) (interface{}, error) {
	patterns, err := h.service.GetPatterns(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get patterns: %w", err)
	}

	// Convert patterns
	results := make([]map[string]interface{}, 0, len(patterns))
	for _, p := range patterns {
		results = append(results, map[string]interface{}{
			"id":          p.ID,
			"error_type":  p.ErrorType,
			"description": p.Description,
			"solution":    p.Solution,
			"frequency":   p.Frequency,
			"confidence":  p.Confidence,
			"created_at":  p.CreatedAt,
		})
	}

	return map[string]interface{}{
		"patterns": results,
		"count":    len(results),
	}, nil
}
