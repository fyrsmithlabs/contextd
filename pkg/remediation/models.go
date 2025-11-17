package remediation

import (
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// Remediation represents an error solution with semantic search support.
//
// Remediations store error messages and their solutions with automatic
// pattern extraction and hybrid matching (semantic + string similarity).
type Remediation struct {
	// ID is the unique identifier for this remediation (UUID)
	ID string `json:"id"`

	// ProjectPath is the absolute path to the project (required)
	ProjectPath string `json:"project_path"`

	// ErrorMsg is the error message (required, max 1000 chars)
	ErrorMsg string `json:"error_msg"`

	// Solution is the fix or workaround (required, max 5000 chars)
	Solution string `json:"solution"`

	// Context contains additional details about the error (optional, max 10KB)
	Context string `json:"context"`

	// Patterns are auto-extracted error patterns for matching
	Patterns []string `json:"patterns"`

	// Metadata contains additional key-value pairs for filtering
	Metadata map[string]interface{} `json:"metadata"`

	// CreatedAt is when this remediation was created
	CreatedAt time.Time `json:"created_at"`
}

// SearchOptions configures remediation search behavior.
type SearchOptions struct {
	// ProjectPath limits search to a specific project (required for multi-tenant isolation)
	ProjectPath string

	// Limit is the maximum number of results to return (default: 5, max: 50)
	Limit int

	// Threshold is the hybrid matching threshold (0.0-1.0, default: 0.6)
	// Score = 0.7 * semantic_similarity + 0.3 * string_similarity
	Threshold float64
}

// SearchResult represents a remediation with similarity score.
type SearchResult struct {
	Remediation   *Remediation
	Score         float64 // Hybrid score (0.0-1.0)
	SemanticScore float64 // Embedding similarity
	StringScore   float64 // Levenshtein similarity
}

// Common validation errors.
var (
	ErrInvalidRemediation  = errors.New("invalid remediation")
	ErrProjectPathRequired = errors.New("project_path is required")
	ErrProjectPathNotAbs   = errors.New("project_path must be absolute")
	ErrErrorMsgRequired    = errors.New("error_msg is required")
	ErrErrorMsgTooLong     = errors.New("error_msg exceeds 1000 characters")
	ErrSolutionRequired    = errors.New("solution is required")
	ErrSolutionTooLong     = errors.New("solution exceeds 5000 characters")
	ErrContextTooLarge     = errors.New("context exceeds 10KB")
	ErrInvalidLimit        = errors.New("limit must be between 1 and 50")
	ErrInvalidThreshold    = errors.New("threshold must be between 0.0 and 1.0")
)

// Validation constants.
const (
	MaxErrorMsgLength = 1000
	MaxSolutionLength = 5000
	MaxContextSize    = 10 * 1024 // 10KB
	DefaultLimit      = 5
	MaxLimit          = 50
	DefaultThreshold  = 0.6
)

// Validate checks if the remediation is valid for creation/update.
//
// Returns ErrInvalidRemediation with specific details if validation fails.
func (r *Remediation) Validate() error {
	// Project path validation
	if r.ProjectPath == "" {
		return fmt.Errorf("%w: %v", ErrInvalidRemediation, ErrProjectPathRequired)
	}

	// Security: Ensure absolute path (prevents directory traversal)
	clean := filepath.Clean(r.ProjectPath)
	if !filepath.IsAbs(clean) {
		return fmt.Errorf("%w: %v", ErrInvalidRemediation, ErrProjectPathNotAbs)
	}
	r.ProjectPath = clean // Normalize

	// Error message validation
	if r.ErrorMsg == "" {
		return fmt.Errorf("%w: %v", ErrInvalidRemediation, ErrErrorMsgRequired)
	}
	if len(r.ErrorMsg) > MaxErrorMsgLength {
		return fmt.Errorf("%w: %v", ErrInvalidRemediation, ErrErrorMsgTooLong)
	}

	// Solution validation
	if r.Solution == "" {
		return fmt.Errorf("%w: %v", ErrInvalidRemediation, ErrSolutionRequired)
	}
	if len(r.Solution) > MaxSolutionLength {
		return fmt.Errorf("%w: %v", ErrInvalidRemediation, ErrSolutionTooLong)
	}

	// Context validation (optional but enforce size limit)
	if len(r.Context) > MaxContextSize {
		return fmt.Errorf("%w: %v", ErrInvalidRemediation, ErrContextTooLarge)
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
	if opts.Threshold == 0 {
		opts.Threshold = DefaultThreshold
	}

	// Validate limits
	if opts.Limit < 1 || opts.Limit > MaxLimit {
		return ErrInvalidLimit
	}
	if opts.Threshold < 0.0 || opts.Threshold > 1.0 {
		return ErrInvalidThreshold
	}

	return nil
}

// ExtractPatterns identifies common error patterns for matching.
//
// Patterns extracted:
//   - File path errors (*.go:*, *.py:*, etc.)
//   - Connection errors (connection refused, etc.)
//   - File system errors (file not found, permission denied)
//   - Timeout errors
//
// Normalization applied:
//   - File paths: /path/to/file.go:123 → *.go:*
//   - Port numbers: port 8080 → port *
func ExtractPatterns(errorMsg string) []string {
	var patterns []string
	normalized := strings.ToLower(errorMsg)

	// Pattern 1: File path errors
	filePathRegex := regexp.MustCompile(`/[^\s]+\.(go|py|js|ts|java|rb|rs|cpp|c|h):\d+`)
	if filePathRegex.MatchString(errorMsg) {
		patterns = append(patterns, "file_path_error")
	}

	// Pattern 2: Common error messages
	commonErrors := map[string]string{
		"connection refused":        "connection refused",
		"no such file or directory": "file not found",
		"permission denied":         "permission denied",
		"timeout":                   "timeout",
		"deadline exceeded":         "timeout",
		"unauthorized":              "unauthorized",
		"forbidden":                 "forbidden",
	}

	for keyword, pattern := range commonErrors {
		if strings.Contains(normalized, keyword) {
			// Avoid duplicates
			found := false
			for _, p := range patterns {
				if p == pattern {
					found = true
					break
				}
			}
			if !found {
				patterns = append(patterns, pattern)
			}
		}
	}

	return patterns
}
