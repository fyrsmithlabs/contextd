package checkpoint

import (
	"errors"
	"fmt"
	"path/filepath"
	"time"
)

// Checkpoint represents a saved session state with semantic search support.
//
// Checkpoints store work-in-progress context with automatic vector embeddings
// for semantic search. Each checkpoint is isolated to a project path for
// multi-tenant security.
type Checkpoint struct {
	// ID is the unique identifier for this checkpoint (UUID)
	ID string `json:"id"`

	// ProjectPath is the absolute path to the project (required)
	ProjectPath string `json:"project_path"`

	// Summary is a brief description of the checkpoint (required, max 500 chars)
	Summary string `json:"summary"`

	// Content is the full checkpoint data (optional, max 100KB)
	Content string `json:"content"`

	// Metadata contains additional key-value pairs for filtering and context
	Metadata map[string]interface{} `json:"metadata"`

	// Tags categorize checkpoints for organization
	Tags []string `json:"tags"`

	// Branch is the git branch name (optional, auto-detected from project path)
	Branch string `json:"branch,omitempty"`

	// CreatedAt is when this checkpoint was created
	CreatedAt time.Time `json:"created_at"`

	// UpdatedAt is when this checkpoint was last modified
	UpdatedAt time.Time `json:"updated_at"`
}

// SearchOptions configures checkpoint search behavior.
type SearchOptions struct {
	// ProjectPath limits search to a specific project (required for multi-tenant isolation)
	ProjectPath string

	// Limit is the maximum number of results to return (default: 10, max: 100)
	Limit int

	// MinScore is the minimum similarity score (0.0-1.0, default: 0.7)
	MinScore float32

	// Tags filters results to checkpoints with any of these tags
	Tags []string

	// Branch filters results to checkpoints from a specific git branch (optional)
	Branch string
}

// ListOptions configures checkpoint listing behavior.
type ListOptions struct {
	// ProjectPath limits listing to a specific project (required)
	ProjectPath string

	// Limit is the maximum number of results (default: 20, max: 100)
	Limit int

	// Offset for pagination (default: 0)
	Offset int

	// Tags filters results to checkpoints with any of these tags
	Tags []string
}

// SearchResult represents a checkpoint with similarity score.
type SearchResult struct {
	Checkpoint *Checkpoint
	Score      float32
}

// Common validation errors.
var (
	ErrInvalidCheckpoint   = errors.New("invalid checkpoint")
	ErrProjectPathRequired = errors.New("project_path is required")
	ErrProjectPathNotAbs   = errors.New("project_path must be absolute")
	ErrSummaryRequired     = errors.New("summary is required")
	ErrSummaryTooLong      = errors.New("summary exceeds 500 characters")
	ErrContentTooLarge     = errors.New("content exceeds 100KB")
	ErrInvalidLimit        = errors.New("limit must be between 1 and 100")
	ErrInvalidMinScore     = errors.New("min_score must be between 0.0 and 1.0")
)

// Validation constants.
const (
	MaxSummaryLength = 500
	MaxContentSize   = 100 * 1024 // 100KB
	DefaultLimit     = 10
	MaxLimit         = 100
	DefaultMinScore  = 0.7
)

// Validate checks if the checkpoint is valid for creation/update.
//
// Returns ErrInvalidCheckpoint with specific details if validation fails.
func (c *Checkpoint) Validate() error {
	// Project path validation
	if c.ProjectPath == "" {
		return fmt.Errorf("%w: %v", ErrInvalidCheckpoint, ErrProjectPathRequired)
	}

	// Security: Ensure absolute path (prevents directory traversal)
	clean := filepath.Clean(c.ProjectPath)
	if !filepath.IsAbs(clean) {
		return fmt.Errorf("%w: %v", ErrInvalidCheckpoint, ErrProjectPathNotAbs)
	}
	c.ProjectPath = clean // Normalize

	// Summary validation
	if c.Summary == "" {
		return fmt.Errorf("%w: %v", ErrInvalidCheckpoint, ErrSummaryRequired)
	}
	if len(c.Summary) > MaxSummaryLength {
		return fmt.Errorf("%w: %v", ErrInvalidCheckpoint, ErrSummaryTooLong)
	}

	// Content validation (optional but enforce size limit)
	if len(c.Content) > MaxContentSize {
		return fmt.Errorf("%w: %v", ErrInvalidCheckpoint, ErrContentTooLarge)
	}

	return nil
}

// Validate checks if search options are valid.
func (opts *SearchOptions) Validate() error {
	// Project path required for multi-tenant isolation
	if opts.ProjectPath == "" {
		return ErrProjectPathRequired
	}
	if !filepath.IsAbs(opts.ProjectPath) {
		return ErrProjectPathNotAbs
	}

	// Apply defaults
	if opts.Limit == 0 {
		opts.Limit = DefaultLimit
	}
	if opts.MinScore == 0 {
		opts.MinScore = DefaultMinScore
	}

	// Validate limits
	if opts.Limit < 1 || opts.Limit > MaxLimit {
		return ErrInvalidLimit
	}
	if opts.MinScore < 0.0 || opts.MinScore > 1.0 {
		return ErrInvalidMinScore
	}

	return nil
}

// Validate checks if list options are valid.
func (opts *ListOptions) Validate() error {
	// Project path required for multi-tenant isolation
	if opts.ProjectPath == "" {
		return ErrProjectPathRequired
	}
	if !filepath.IsAbs(opts.ProjectPath) {
		return ErrProjectPathNotAbs
	}

	// Apply defaults
	if opts.Limit == 0 {
		opts.Limit = 20
	}

	// Validate limits
	if opts.Limit < 1 || opts.Limit > MaxLimit {
		return ErrInvalidLimit
	}
	if opts.Offset < 0 {
		return errors.New("offset must be non-negative")
	}

	return nil
}
