package remediation

import (
	"time"
)

// ErrorCategory represents the category of error.
type ErrorCategory string

const (
	// ErrorCompile represents compile-time errors.
	ErrorCompile ErrorCategory = "compile"
	// ErrorRuntime represents runtime errors.
	ErrorRuntime ErrorCategory = "runtime"
	// ErrorTest represents test failures.
	ErrorTest ErrorCategory = "test"
	// ErrorLint represents linter errors.
	ErrorLint ErrorCategory = "lint"
	// ErrorSecurity represents security vulnerabilities.
	ErrorSecurity ErrorCategory = "security"
	// ErrorPerformance represents performance issues.
	ErrorPerformance ErrorCategory = "performance"
	// ErrorOther represents other errors.
	ErrorOther ErrorCategory = "other"
)

// Scope represents the scope of a remediation.
type Scope string

const (
	// ScopeProject is project-level scope.
	ScopeProject Scope = "project"
	// ScopeTeam is team-level scope.
	ScopeTeam Scope = "team"
	// ScopeOrg is organization-level scope.
	ScopeOrg Scope = "org"
)

// Remediation represents a stored error fix pattern.
type Remediation struct {
	// ID is the unique identifier for this remediation.
	ID string `json:"id"`

	// Title is a short title for the remediation.
	Title string `json:"title"`

	// Problem is a description of the error or issue.
	Problem string `json:"problem"`

	// Symptoms are observable symptoms of this error.
	Symptoms []string `json:"symptoms"`

	// RootCause is the underlying cause of the error.
	RootCause string `json:"root_cause"`

	// Solution is the fix that resolved the error.
	Solution string `json:"solution"`

	// CodeDiff is an optional code diff showing the fix.
	CodeDiff string `json:"code_diff,omitempty"`

	// AffectedFiles are files that were changed.
	AffectedFiles []string `json:"affected_files,omitempty"`

	// Category is the error category.
	Category ErrorCategory `json:"category"`

	// Confidence is the current confidence score (0.0 - 1.0).
	Confidence float64 `json:"confidence"`

	// UsageCount is how many times this remediation has been retrieved.
	UsageCount int64 `json:"usage_count"`

	// Tags are labels for categorization and filtering.
	Tags []string `json:"tags"`

	// Scope determines visibility (project, team, org).
	Scope Scope `json:"scope"`

	// TenantID is the organization this remediation belongs to.
	TenantID string `json:"tenant_id"`

	// TeamID is the team this remediation belongs to (for team scope).
	TeamID string `json:"team_id,omitempty"`

	// ProjectPath is the project this remediation belongs to (for project scope).
	ProjectPath string `json:"project_path,omitempty"`

	// SessionID is the session this remediation was extracted from.
	SessionID string `json:"session_id,omitempty"`

	// CreatedAt is when this remediation was created.
	CreatedAt time.Time `json:"created_at"`

	// UpdatedAt is when this remediation was last updated.
	UpdatedAt time.Time `json:"updated_at"`

	// Vector is the embedding vector for semantic search.
	Vector []float32 `json:"-"`
}

// ScoredRemediation is a remediation with a similarity score.
type ScoredRemediation struct {
	Remediation
	Score float64 `json:"score"`
}

// SearchRequest represents parameters for remediation search.
type SearchRequest struct {
	// Query is the error message or description to search for.
	Query string

	// Vector is an optional pre-computed embedding vector.
	Vector []float32

	// Limit is the maximum number of results to return.
	Limit int

	// MinConfidence is the minimum confidence threshold.
	MinConfidence float64

	// Category filters by error category (optional).
	Category ErrorCategory

	// Scope filters by scope (optional, searches all if empty).
	Scope Scope

	// TenantID is required for multi-tenant isolation.
	TenantID string

	// TeamID filters to a specific team (optional).
	TeamID string

	// ProjectPath filters to a specific project (optional).
	ProjectPath string

	// Tags filters by tags (optional, any match).
	Tags []string

	// IncludeHierarchy includes parent scopes in search.
	// If searching project scope, also searches team and org.
	IncludeHierarchy bool
}

// RecordRequest represents parameters for recording a remediation.
type RecordRequest struct {
	Title         string
	Problem       string
	Symptoms      []string
	RootCause     string
	Solution      string
	CodeDiff      string
	AffectedFiles []string
	Category      ErrorCategory
	Tags          []string
	Scope         Scope
	TenantID      string
	TeamID        string
	ProjectPath   string
	SessionID     string
	Confidence    float64 // Initial confidence (default: 0.5)
}

// FeedbackRequest represents parameters for providing feedback on a remediation.
type FeedbackRequest struct {
	RemediationID string
	TenantID      string
	Rating        FeedbackRating
	SessionID     string
	Comment       string
}

// FeedbackRating represents a feedback rating.
type FeedbackRating string

const (
	// RatingHelpful indicates the remediation was helpful.
	RatingHelpful FeedbackRating = "helpful"
	// RatingNotHelpful indicates the remediation was not helpful.
	RatingNotHelpful FeedbackRating = "not_helpful"
	// RatingOutdated indicates the remediation is outdated.
	RatingOutdated FeedbackRating = "outdated"
)
