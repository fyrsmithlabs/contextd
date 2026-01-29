package reasoningbank

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

// Common errors for fact extraction operations.
var (
	ErrEmptyFactText        = errors.New("fact text cannot be empty")
	ErrInvalidFactSubject   = errors.New("fact subject cannot be empty")
	ErrInvalidFactPredicate = errors.New("fact predicate cannot be empty")
	ErrInvalidFactObject    = errors.New("fact object cannot be empty")
)

// Fact represents a structured triple extracted from text (subject-predicate-object).
//
// Facts capture relationships between entities extracted from memory content or
// conversations. Each fact has:
// - Subject: The entity performing or being described by the action/property
// - Predicate: The relationship/action between subject and object
// - Object: The entity being acted upon or the property value
//
// Facts are used to build a knowledge graph of learned information over time.
type Fact struct {
	// ID is the unique fact identifier (UUID).
	ID string `json:"id"`

	// Subject is the entity performing or being described by the action/property.
	// Example: "I", "Claude", "user", "the system"
	Subject string `json:"subject"`

	// Predicate is the relationship or action between subject and object.
	// Example: "attended", "learned", "considering", "implemented"
	Predicate string `json:"predicate"`

	// Object is the entity being acted upon or the property value.
	// Example: "meeting X", "Go error handling", "architecture review"
	Object string `json:"object"`

	// Timestamp is when this fact was extracted or when it occurred.
	// Supports temporal reference resolution (e.g., "yesterday" -> absolute date).
	Timestamp time.Time `json:"timestamp"`

	// Confidence is a score from 0.0 to 1.0 indicating extraction reliability.
	// Higher values indicate higher confidence in the extraction.
	// Example: 1.0 for explicit statements, 0.7 for implicit inferences.
	Confidence float64 `json:"confidence"`

	// Provenance is the original source text from which this fact was extracted.
	// Preserved for verification and traceability.
	Provenance string `json:"provenance"`

	// SourceID is the ID of the memory or message from which this fact was extracted.
	// Links facts back to their source for context and attribution.
	SourceID string `json:"source_id"`

	// ProjectID identifies which project this fact belongs to.
	ProjectID string `json:"project_id"`

	// CreatedAt is when the fact was extracted.
	CreatedAt time.Time `json:"created_at"`
}

// NewFact creates a new fact with validation and generated UUID.
func NewFact(subject, predicate, object string, timestamp time.Time, confidence float64, provenance, sourceID, projectID string) (*Fact, error) {
	if subject == "" {
		return nil, ErrInvalidFactSubject
	}
	if predicate == "" {
		return nil, ErrInvalidFactPredicate
	}
	if object == "" {
		return nil, ErrInvalidFactObject
	}
	if confidence < 0.0 || confidence > 1.0 {
		return nil, errors.New("confidence must be between 0.0 and 1.0")
	}
	if projectID == "" {
		return nil, ErrEmptyProjectID
	}

	return &Fact{
		ID:         uuid.New().String(),
		Subject:    subject,
		Predicate:  predicate,
		Object:     object,
		Timestamp:  timestamp,
		Confidence: confidence,
		Provenance: provenance,
		SourceID:   sourceID,
		ProjectID:  projectID,
		CreatedAt:  time.Now(),
	}, nil
}

// Validate checks if the fact has valid fields.
func (f *Fact) Validate() error {
	if f.ID == "" {
		return errors.New("fact ID cannot be empty")
	}
	if _, err := uuid.Parse(f.ID); err != nil {
		return errors.New("invalid fact ID format")
	}
	if f.Subject == "" {
		return ErrInvalidFactSubject
	}
	if f.Predicate == "" {
		return ErrInvalidFactPredicate
	}
	if f.Object == "" {
		return ErrInvalidFactObject
	}
	if f.Confidence < 0.0 || f.Confidence > 1.0 {
		return errors.New("confidence must be between 0.0 and 1.0")
	}
	if f.ProjectID == "" {
		return ErrEmptyProjectID
	}
	if f.Timestamp.IsZero() {
		return errors.New("timestamp cannot be zero")
	}
	return nil
}

// FactExtractor defines the interface for extracting facts from text.
//
// Implementations extract structured triples (subject-predicate-object) from
// unstructured text, with support for temporal reference resolution.
type FactExtractor interface {
	// Extract parses text and returns structured facts.
	//
	// Supports:
	//   - Subject-verb-object patterns: "I went to X" -> (I, attended, X)
	//   - Temporal references: "yesterday", "last week" -> resolved to absolute dates
	//   - Implicit relations: "I'm thinking about X" -> (I, considering, X)
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeouts
	//   - text: Source text to extract facts from
	//   - referenceDate: Base date for resolving temporal references
	//
	// Returns:
	//   - Slice of extracted facts with confidence scores
	//   - Error if extraction fails
	Extract(ctx context.Context, text string, referenceDate time.Time) ([]Fact, error)
}
