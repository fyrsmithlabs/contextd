// Package workflows provides Temporal workflow definitions for contextd automation.
//
// This file contains shared types used across multiple workflows and activities.
package workflows

import (
	"fmt"

	"github.com/fyrsmithlabs/contextd/internal/config"
)

// Common result types

// PostCommentResult represents the result of posting a GitHub comment.
type PostCommentResult struct {
	URL string // URL of the posted or updated comment
}

// FileChange represents a changed file in a pull request.
type FileChange struct {
	Path      string // File path relative to repository root
	Status    string // "added", "modified", "removed", "renamed"
	Additions int    // Number of lines added
	Deletions int    // Number of lines deleted
}

// Version Validation types

// VersionValidationConfig configures the version validation workflow.
type VersionValidationConfig struct {
	Owner       string        // GitHub repository owner
	Repo        string        // GitHub repository name
	PRNumber    int           // Pull request number
	HeadSHA     string        // PR commit SHA
	GitHubToken config.Secret // GitHub API token for activities
}

// Validate checks that all required fields are set.
func (c *VersionValidationConfig) Validate() error {
	if c.Owner == "" {
		return fmt.Errorf("Owner is required")
	}
	if c.Repo == "" {
		return fmt.Errorf("Repo is required")
	}
	if c.PRNumber <= 0 {
		return fmt.Errorf("PRNumber must be positive")
	}
	if c.HeadSHA == "" {
		return fmt.Errorf("HeadSHA is required")
	}
	if !c.GitHubToken.IsSet() {
		return fmt.Errorf("GitHubToken is required")
	}
	return nil
}

// VersionValidationResult contains validation results.
type VersionValidationResult struct {
	VersionMatches bool     // Whether VERSION file matches plugin.json
	VersionFile    string   // Version from VERSION file
	PluginVersion  string   // Version from plugin.json
	CommentPosted  bool     // Whether we posted a comment
	CommentURL     string   // URL of posted comment
	Errors         []string // Any errors encountered
}

// FetchFileContentInput defines parameters for fetching file content from GitHub.
type FetchFileContentInput struct {
	Owner       string        // Repository owner
	Repo        string        // Repository name
	Path        string        // File path (restricted to allowed paths)
	Ref         string        // Commit SHA or branch name
	GitHubToken config.Secret // GitHub API token
}

// PostVersionCommentInput defines parameters for posting version-related comments.
type PostVersionCommentInput struct {
	Owner         string        // Repository owner
	Repo          string        // Repository name
	PRNumber      int           // Pull request number
	VersionFile   string        // Version from VERSION file
	PluginVersion string        // Version from plugin.json
	GitHubToken   config.Secret // GitHub API token
}

// Plugin Update Validation types

// PluginUpdateValidationConfig configures the plugin validation workflow.
type PluginUpdateValidationConfig struct {
	Owner              string        // GitHub repository owner
	Repo               string        // GitHub repository name
	PRNumber           int           // Pull request number
	BaseBranch         string        // Base branch (usually "main")
	HeadBranch         string        // PR branch
	HeadSHA            string        // PR commit SHA
	GitHubToken        config.Secret // GitHub API token for activities
	UseAgentValidation bool          // Whether to use AI agent for documentation validation
}

// PluginUpdateValidationResult contains validation results.
type PluginUpdateValidationResult struct {
	CodeFilesChanged      []string                       // Files that affect plugin behavior
	PluginFilesChanged    []string                       // Files in .claude-plugin/
	NeedsUpdate           bool                           // Whether plugin needs updating
	SchemaValid           bool                           // Whether schemas are valid JSON
	AgentValidation       *DocumentationValidationResult // Agent validation results (if enabled)
	AgentValidationRan    bool                           // Whether agent validation was executed
	CommentPosted         bool                           // Whether we posted a comment
	CommentURL            string                         // URL of posted comment
	Errors                []string                       // Any errors encountered
}

// CategorizedFiles contains files categorized by type.
type CategorizedFiles struct {
	CodeFiles   []string // Files that affect plugin behavior
	PluginFiles []string // Files in .claude-plugin/
	OtherFiles  []string // Other files (tests, docs, etc.)
}

// FetchPRFilesInput defines parameters for fetching PR file changes.
type FetchPRFilesInput struct {
	Owner       string        // Repository owner
	Repo        string        // Repository name
	PRNumber    int           // Pull request number
	GitHubToken config.Secret // GitHub API token
}

// CategorizeFilesInput defines parameters for file categorization.
type CategorizeFilesInput struct {
	Files []FileChange // List of files to categorize
}

// ValidateSchemasInput defines parameters for schema validation.
type ValidateSchemasInput struct {
	Owner       string        // Repository owner
	Repo        string        // Repository name
	HeadSHA     string        // Commit SHA to validate
	FilePath    string        // Path to schema file
	GitHubToken config.Secret // GitHub API token
}

// SchemaValidationResult represents schema validation results.
type SchemaValidationResult struct {
	Valid  bool     // Whether the schema is valid
	Errors []string // Validation errors, if any
}

// PostCommentInput defines parameters for posting comments.
type PostCommentInput struct {
	Owner           string                         // Repository owner
	Repo            string                         // Repository name
	PRNumber        int                            // Pull request number
	CodeFiles       []string                       // List of code files changed
	PluginFiles     []string                       // List of plugin files changed
	GitHubToken     config.Secret                  // GitHub API token
	AgentValidation *DocumentationValidationResult // Optional agent validation results
}

// Documentation Validation types

// DocumentationValidationInput defines parameters for ValidateDocumentationActivity.
type DocumentationValidationInput struct {
	Owner       string   // Repository owner
	Repo        string   // Repository name
	PRNumber    int      // Pull request number
	HeadSHA     string   // Commit SHA
	CodeFiles   []string // List of code files changed
	PluginFiles []string // List of plugin files changed
}

// DocumentationValidationResult represents validation findings.
type DocumentationValidationResult struct {
	Valid          bool              `json:"valid"`                    // Whether documentation is valid
	Summary        string            `json:"summary"`                  // Summary of validation
	FilesReviewed  int               `json:"files_reviewed,omitempty"` // Number of files reviewed
	CriticalIssues []ValidationIssue `json:"critical_issues"`          // Critical issues found
	HighIssues     []ValidationIssue `json:"high_issues"`              // High priority issues
	MediumIssues   []ValidationIssue `json:"medium_issues"`            // Medium priority issues
	LowIssues      []ValidationIssue `json:"low_issues"`               // Low priority issues
}

// ValidationIssue represents a single validation finding.
type ValidationIssue struct {
	File     string `json:"file"`                // File path
	Line     int    `json:"line,omitempty"`      // Line number (if applicable)
	Severity string `json:"severity,omitempty"`  // Issue severity
	Issue    string `json:"issue"`               // Description of the issue
	Current  string `json:"current,omitempty"`   // Current value/state
	ShouldBe string `json:"should_be,omitempty"` // Expected value/state
	Fix      string `json:"fix"`                 // How to fix the issue
	Impact   string `json:"impact,omitempty"`    // Impact description
}

// GitHub Client types

// GitHubClientConfig holds GitHub client configuration.
type GitHubClientConfig struct {
	Token config.Secret // GitHub API token
}

// Note: RetryConfig is defined in github_retry.go
// Note: WorkflowError is defined in errors.go
