package skills

import (
	"errors"
	"fmt"
	"time"
)

var (
	// ErrInvalidSkill indicates validation failure
	ErrInvalidSkill = errors.New("invalid skill")

	// ErrSkillNotFound indicates skill was not found
	ErrSkillNotFound = errors.New("skill not found")
)

// Skill represents a reusable workflow template.
type Skill struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Content     string                 `json:"content"`
	Tags        []string               `json:"tags,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// Validate validates the skill fields.
func (s *Skill) Validate() error {
	if s.Name == "" {
		return fmt.Errorf("%w: name is required", ErrInvalidSkill)
	}
	if len(s.Name) > 200 {
		return fmt.Errorf("%w: name must be <= 200 characters", ErrInvalidSkill)
	}
	if s.Description == "" {
		return fmt.Errorf("%w: description is required", ErrInvalidSkill)
	}
	if len(s.Description) > 2000 {
		return fmt.Errorf("%w: description must be <= 2000 characters", ErrInvalidSkill)
	}
	if s.Content == "" {
		return fmt.Errorf("%w: content is required", ErrInvalidSkill)
	}
	if len(s.Content) > 50000 {
		return fmt.Errorf("%w: content must be <= 50000 characters", ErrInvalidSkill)
	}
	return nil
}

// SearchResult represents a skill search result with score.
type SearchResult struct {
	Skill *Skill  `json:"skill"`
	Score float32 `json:"score"`
}
