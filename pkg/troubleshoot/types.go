package troubleshoot

import (
	"errors"
	"time"
)

var (
	// ErrEmptyErrorMessage indicates error message is required
	ErrEmptyErrorMessage = errors.New("error message cannot be empty")

	// ErrInvalidConfidence indicates confidence score out of range
	ErrInvalidConfidence = errors.New("confidence must be between 0.0 and 1.0")
)

// Diagnosis represents AI-powered error analysis.
type Diagnosis struct {
	ErrorMessage    string       `json:"error_message"`
	RootCause       string       `json:"root_cause"`
	Hypotheses      []Hypothesis `json:"hypotheses"`
	Recommendations []string     `json:"recommendations"`
	RelatedPatterns []Pattern    `json:"related_patterns"`
	Confidence      float64      `json:"confidence"`
}

// Hypothesis represents a possible cause of the error.
type Hypothesis struct {
	Description string  `json:"description"`
	Likelihood  float64 `json:"likelihood"`
	Evidence    string  `json:"evidence"`
}

// Validate validates a hypothesis.
func (h *Hypothesis) Validate() error {
	if h.Description == "" {
		return errors.New("hypothesis description cannot be empty")
	}
	if h.Likelihood < 0.0 || h.Likelihood > 1.0 {
		return errors.New("likelihood must be between 0.0 and 1.0")
	}
	return nil
}

// Pattern represents a known error pattern with solution.
type Pattern struct {
	ID          string    `json:"id"`
	ErrorType   string    `json:"error_type"`
	Description string    `json:"description"`
	Solution    string    `json:"solution"`
	Frequency   int       `json:"frequency"`
	Confidence  float64   `json:"confidence"`
	CreatedAt   time.Time `json:"created_at"`
}

// Validate validates a pattern.
func (p *Pattern) Validate() error {
	if p.ErrorType == "" {
		return errors.New("error type is required")
	}
	if p.Description == "" {
		return errors.New("description is required")
	}
	if p.Solution == "" {
		return errors.New("solution is required")
	}
	if p.Confidence < 0.0 || p.Confidence > 1.0 {
		return ErrInvalidConfidence
	}
	return nil
}
