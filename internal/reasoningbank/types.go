package reasoningbank

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// Common errors for ReasoningBank operations.
var (
	ErrMemoryNotFound      = errors.New("memory not found")
	ErrInvalidMemory       = errors.New("invalid memory")
	ErrEmptyTitle          = errors.New("memory title cannot be empty")
	ErrEmptyContent        = errors.New("memory content cannot be empty")
	ErrInvalidConfidence   = errors.New("confidence must be between 0.0 and 1.0")
	ErrInvalidOutcome      = errors.New("outcome must be 'success' or 'failure'")
	ErrEmptyProjectID      = errors.New("project ID cannot be empty")
)

// Outcome represents the result type of a memory.
type Outcome string

const (
	// OutcomeSuccess indicates a successful strategy or pattern.
	OutcomeSuccess Outcome = "success"

	// OutcomeFailure indicates an anti-pattern or failed approach.
	OutcomeFailure Outcome = "failure"
)

// Memory represents a cross-session memory in the ReasoningBank.
//
// Memories are distilled strategies learned from agent interactions.
// They can represent successful patterns (outcome="success") or
// anti-patterns to avoid (outcome="failure").
//
// Confidence is tracked and adjusted based on feedback signals:
//   - Explicit ratings from users
//   - Implicit success (memory helped solve a task)
//   - Code stability (solution didn't need rework)
type Memory struct {
	// ID is the unique memory identifier (UUID).
	ID string `json:"id"`

	// ProjectID identifies which project this memory belongs to.
	ProjectID string `json:"project_id"`

	// Title is a brief summary of the memory (e.g., "Go error handling with context").
	Title string `json:"title"`

	// Description provides additional context about when/why this memory is useful.
	Description string `json:"description,omitempty"`

	// Content is the main memory content (strategy, anti-pattern, code example).
	Content string `json:"content"`

	// Outcome indicates if this is a success pattern or failure anti-pattern.
	Outcome Outcome `json:"outcome"`

	// Confidence is a score from 0.0 to 1.0 indicating reliability.
	// Higher confidence memories are prioritized in search results.
	// Adjusted based on feedback and usage patterns.
	Confidence float64 `json:"confidence"`

	// UsageCount tracks how many times this memory has been retrieved.
	UsageCount int `json:"usage_count"`

	// Tags are labels for categorization (e.g., "go", "error-handling", "auth").
	Tags []string `json:"tags,omitempty"`

	// ConsolidationID links this memory to a consolidated memory it was merged into.
	// When a memory is consolidated with others, this field is set to the ID of the
	// resulting ConsolidatedMemory. The original memory is preserved for attribution.
	ConsolidationID *string `json:"consolidation_id,omitempty"`

	// CreatedAt is when the memory was created.
	CreatedAt time.Time `json:"created_at"`

	// UpdatedAt is when the memory was last modified.
	UpdatedAt time.Time `json:"updated_at"`
}

// NewMemory creates a new memory with a generated UUID and default values.
func NewMemory(projectID, title, content string, outcome Outcome, tags []string) (*Memory, error) {
	if projectID == "" {
		return nil, ErrEmptyProjectID
	}
	if title == "" {
		return nil, ErrEmptyTitle
	}
	if content == "" {
		return nil, ErrEmptyContent
	}
	if outcome != OutcomeSuccess && outcome != OutcomeFailure {
		return nil, ErrInvalidOutcome
	}

	now := time.Now()
	return &Memory{
		ID:         uuid.New().String(),
		ProjectID:  projectID,
		Title:      title,
		Content:    content,
		Outcome:    outcome,
		Confidence: 0.5, // Default confidence (neutral)
		UsageCount: 0,
		Tags:       tags,
		CreatedAt:  now,
		UpdatedAt:  now,
	}, nil
}

// Validate checks if the memory has valid fields.
func (m *Memory) Validate() error {
	if m.ID == "" {
		return errors.New("memory ID cannot be empty")
	}
	if _, err := uuid.Parse(m.ID); err != nil {
		return errors.New("invalid memory ID format")
	}
	if m.ProjectID == "" {
		return ErrEmptyProjectID
	}
	if m.Title == "" {
		return ErrEmptyTitle
	}
	if m.Content == "" {
		return ErrEmptyContent
	}
	if m.Outcome != OutcomeSuccess && m.Outcome != OutcomeFailure {
		return ErrInvalidOutcome
	}
	if m.Confidence < 0.0 || m.Confidence > 1.0 {
		return ErrInvalidConfidence
	}
	if m.UsageCount < 0 {
		return errors.New("usage count cannot be negative")
	}
	return nil
}

// AdjustConfidence updates the confidence based on feedback.
//
// For helpful feedback:
//   - Increases confidence by up to 0.1 (capped at 1.0)
//
// For unhelpful feedback:
//   - Decreases confidence by up to 0.15 (floored at 0.0)
func (m *Memory) AdjustConfidence(helpful bool) {
	if helpful {
		m.Confidence += 0.1
		if m.Confidence > 1.0 {
			m.Confidence = 1.0
		}
	} else {
		m.Confidence -= 0.15
		if m.Confidence < 0.0 {
			m.Confidence = 0.0
		}
	}
	m.UpdatedAt = time.Now()
}

// IncrementUsage increments the usage count and updates timestamp.
func (m *Memory) IncrementUsage() {
	m.UsageCount++
	m.UpdatedAt = time.Now()
}

// ConsolidationType represents the method used to create a consolidated memory.
type ConsolidationType string

const (
	// ConsolidationMerged indicates memories were merged into a single synthesized memory.
	ConsolidationMerged ConsolidationType = "merged"

	// ConsolidationDeduplicated indicates duplicate or near-duplicate memories were combined.
	ConsolidationDeduplicated ConsolidationType = "deduplicated"

	// ConsolidationSynthesized indicates memories were synthesized into higher-level knowledge.
	ConsolidationSynthesized ConsolidationType = "synthesized"
)

// ConsolidatedMemory represents a memory created by consolidating multiple source memories.
//
// ConsolidatedMemories are created by the Distiller when it detects similar or related
// memories that can be merged into more valuable synthesized knowledge. The original
// source memories are preserved with their ConsolidationID field pointing to this
// consolidated memory.
type ConsolidatedMemory struct {
	// Memory is the consolidated memory record.
	*Memory

	// SourceIDs contains the IDs of all source memories that were consolidated.
	SourceIDs []string `json:"source_ids"`

	// ConsolidationType indicates the method used for consolidation.
	ConsolidationType ConsolidationType `json:"consolidation_type"`

	// SourceAttribution provides context about how the source memories contributed.
	// This is a human-readable description generated by the LLM during synthesis.
	SourceAttribution string `json:"source_attribution,omitempty"`
}
