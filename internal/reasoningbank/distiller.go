package reasoningbank

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"go.uber.org/zap"
)

// SessionOutcome represents the overall outcome of a session.
type SessionOutcome string

const (
	// SessionSuccess indicates the session achieved its goal.
	SessionSuccess SessionOutcome = "success"

	// SessionFailure indicates the session did not achieve its goal.
	SessionFailure SessionOutcome = "failure"

	// SessionPartial indicates partial success or mixed results.
	SessionPartial SessionOutcome = "partial"
)

// SessionSummary contains distilled information from a completed session.
type SessionSummary struct {
	// SessionID uniquely identifies the session.
	SessionID string

	// ProjectID identifies the project this session belongs to.
	ProjectID string

	// Outcome is the overall session result.
	Outcome SessionOutcome

	// Task is a brief description of what the session was trying to accomplish.
	Task string

	// Approach is the strategy or method used (extracted from session).
	Approach string

	// Result describes what happened (success details or failure reasons).
	Result string

	// Tags are labels for categorization (language, domain, problem type).
	Tags []string

	// Duration is how long the session lasted.
	Duration time.Duration

	// CompletedAt is when the session ended.
	CompletedAt time.Time
}

// Distiller extracts learnings from completed sessions and creates memories.
//
// FR-006: Distillation pipeline for async memory extraction
// FR-009: Outcome differentiation (success vs failure)
type Distiller struct {
	service *Service
	logger  *zap.Logger
}

// NewDistiller creates a new session distiller.
func NewDistiller(service *Service, logger *zap.Logger) (*Distiller, error) {
	if service == nil {
		return nil, fmt.Errorf("service cannot be nil")
	}
	if logger == nil {
		return nil, fmt.Errorf("logger cannot be nil")
	}

	return &Distiller{
		service: service,
		logger:  logger,
	}, nil
}

// DistillSession extracts learnings from a completed session and creates memories.
//
// This is called asynchronously after a session ends, so it should not block.
//
// Success patterns (outcome="success") become positive memories.
// Failure patterns (outcome="failure") become anti-pattern warnings.
//
// Initial confidence is set to DistilledConfidence (0.6) since distilled
// memories are less reliable than explicit captures (0.8).
func (d *Distiller) DistillSession(ctx context.Context, summary SessionSummary) error {
	if summary.ProjectID == "" {
		return fmt.Errorf("project ID cannot be empty")
	}
	if summary.SessionID == "" {
		return fmt.Errorf("session ID cannot be empty")
	}

	d.logger.Info("distilling session",
		zap.String("session_id", summary.SessionID),
		zap.String("project_id", summary.ProjectID),
		zap.String("outcome", string(summary.Outcome)))

	// Extract memories based on outcome
	var memories []*Memory
	var err error

	switch summary.Outcome {
	case SessionSuccess:
		memories, err = d.extractSuccessPatterns(summary)
	case SessionFailure:
		memories, err = d.extractFailurePatterns(summary)
	case SessionPartial:
		// For partial outcomes, extract both success and failure patterns
		successMems, err1 := d.extractSuccessPatterns(summary)
		failureMems, err2 := d.extractFailurePatterns(summary)
		if err1 != nil {
			d.logger.Warn("error extracting success patterns from partial session",
				zap.Error(err1))
		}
		if err2 != nil {
			d.logger.Warn("error extracting failure patterns from partial session",
				zap.Error(err2))
		}
		memories = append(successMems, failureMems...)
	default:
		return fmt.Errorf("unknown session outcome: %s", summary.Outcome)
	}

	if err != nil {
		return fmt.Errorf("extracting patterns: %w", err)
	}

	// Record extracted memories
	for _, memory := range memories {
		if err := d.service.Record(ctx, memory); err != nil {
			d.logger.Error("failed to record distilled memory",
				zap.String("session_id", summary.SessionID),
				zap.String("memory_title", memory.Title),
				zap.Error(err))
			// Continue with other memories even if one fails
		} else {
			d.logger.Info("distilled memory recorded",
				zap.String("session_id", summary.SessionID),
				zap.String("memory_id", memory.ID),
				zap.String("title", memory.Title))
		}
	}

	d.logger.Info("session distillation completed",
		zap.String("session_id", summary.SessionID),
		zap.Int("memories_extracted", len(memories)))

	return nil
}

// extractSuccessPatterns creates memories from successful sessions.
//
// Success patterns become positive guidance for future sessions.
func (d *Distiller) extractSuccessPatterns(summary SessionSummary) ([]*Memory, error) {
	// Create a success pattern memory
	title := d.generateTitle(summary.Task, "Success")
	content := d.formatSuccessContent(summary)

	memory, err := NewMemory(
		summary.ProjectID,
		title,
		content,
		OutcomeSuccess,
		summary.Tags,
	)
	if err != nil {
		return nil, fmt.Errorf("creating success memory: %w", err)
	}

	// Set distilled confidence
	memory.Confidence = DistilledConfidence

	// Add session metadata to description
	memory.Description = fmt.Sprintf("Learned from session %s (duration: %s)",
		summary.SessionID,
		summary.Duration.Round(time.Second))

	return []*Memory{memory}, nil
}

// extractFailurePatterns creates anti-pattern memories from failed sessions.
//
// Failure patterns become warnings about approaches to avoid.
func (d *Distiller) extractFailurePatterns(summary SessionSummary) ([]*Memory, error) {
	// Create an anti-pattern memory
	title := d.generateTitle(summary.Task, "Anti-pattern")
	content := d.formatFailureContent(summary)

	memory, err := NewMemory(
		summary.ProjectID,
		title,
		content,
		OutcomeFailure,
		summary.Tags,
	)
	if err != nil {
		return nil, fmt.Errorf("creating failure memory: %w", err)
	}

	// Set distilled confidence (slightly lower for failures since they're harder to generalize)
	memory.Confidence = DistilledConfidence - 0.1
	if memory.Confidence < 0.0 {
		memory.Confidence = 0.0
	}

	// Add session metadata to description
	memory.Description = fmt.Sprintf("Anti-pattern learned from session %s (duration: %s)",
		summary.SessionID,
		summary.Duration.Round(time.Second))

	return []*Memory{memory}, nil
}

// generateTitle creates a concise title for a memory.
func (d *Distiller) generateTitle(task string, outcome string) string {
	// Truncate task if too long
	maxTaskLen := 50
	if len(task) > maxTaskLen {
		task = task[:maxTaskLen] + "..."
	}

	// Capitalize first letter
	if len(task) > 0 {
		task = strings.ToUpper(task[:1]) + task[1:]
	}

	return fmt.Sprintf("%s: %s", outcome, task)
}

// formatSuccessContent formats a success pattern into memory content.
func (d *Distiller) formatSuccessContent(summary SessionSummary) string {
	var b strings.Builder

	b.WriteString("## Task\n")
	b.WriteString(summary.Task)
	b.WriteString("\n\n")

	b.WriteString("## Successful Approach\n")
	b.WriteString(summary.Approach)
	b.WriteString("\n\n")

	b.WriteString("## Result\n")
	b.WriteString(summary.Result)
	b.WriteString("\n\n")

	if len(summary.Tags) > 0 {
		b.WriteString("## Tags\n")
		b.WriteString(strings.Join(summary.Tags, ", "))
		b.WriteString("\n\n")
	}

	b.WriteString("## When to Use\n")
	b.WriteString("Apply this approach when facing similar tasks involving: ")
	b.WriteString(strings.Join(summary.Tags, ", "))
	b.WriteString(".\n")

	return b.String()
}

// formatFailureContent formats a failure pattern into memory content.
func (d *Distiller) formatFailureContent(summary SessionSummary) string {
	var b strings.Builder

	b.WriteString("## Task\n")
	b.WriteString(summary.Task)
	b.WriteString("\n\n")

	b.WriteString("## Failed Approach (Avoid This)\n")
	b.WriteString(summary.Approach)
	b.WriteString("\n\n")

	b.WriteString("## What Went Wrong\n")
	b.WriteString(summary.Result)
	b.WriteString("\n\n")

	if len(summary.Tags) > 0 {
		b.WriteString("## Tags\n")
		b.WriteString(strings.Join(summary.Tags, ", "))
		b.WriteString("\n\n")
	}

	b.WriteString("## Warning\n")
	b.WriteString("Avoid this approach when working with: ")
	b.WriteString(strings.Join(summary.Tags, ", "))
	b.WriteString(". Look for alternative strategies instead.\n")

	return b.String()
}

// CosineSimilarity computes the cosine similarity between two embedding vectors.
//
// Cosine similarity measures the cosine of the angle between two vectors,
// producing a value between -1 and 1:
//   - 1.0: vectors point in the same direction (identical)
//   - 0.0: vectors are orthogonal (unrelated)
//   - -1.0: vectors point in opposite directions (opposite)
//
// For embedding vectors, similarity is typically in the range [0, 1] since
// embeddings generally have positive components.
//
// Formula: cos(θ) = (A · B) / (||A|| * ||B||)
//
// Returns 0.0 for invalid inputs (empty vectors, zero-magnitude vectors,
// or vectors of different lengths).
func CosineSimilarity(vec1, vec2 []float32) float64 {
	// Validate inputs
	if len(vec1) == 0 || len(vec2) == 0 {
		return 0.0
	}
	if len(vec1) != len(vec2) {
		return 0.0
	}

	// Compute dot product and magnitudes
	var dotProduct float64
	var magnitude1 float64
	var magnitude2 float64

	for i := 0; i < len(vec1); i++ {
		v1 := float64(vec1[i])
		v2 := float64(vec2[i])
		dotProduct += v1 * v2
		magnitude1 += v1 * v1
		magnitude2 += v2 * v2
	}

	// Check for zero-magnitude vectors
	if magnitude1 == 0.0 || magnitude2 == 0.0 {
		return 0.0
	}

	// Compute cosine similarity
	// Use sqrt for magnitudes: ||A|| = sqrt(A · A)
	magnitude1 = math.Sqrt(magnitude1)
	magnitude2 = math.Sqrt(magnitude2)

	return dotProduct / (magnitude1 * magnitude2)
}
