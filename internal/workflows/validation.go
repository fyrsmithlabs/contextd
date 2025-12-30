package workflows

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// Validation errors
var (
	// ErrInvalidInput indicates workflow input validation failed.
	ErrInvalidInput = errors.New("invalid workflow input")

	// ErrEmptyField indicates a required field is empty.
	ErrEmptyField = errors.New("required field is empty")

	// ErrInvalidGitHubIdentifier indicates an invalid GitHub owner/repo name.
	ErrInvalidGitHubIdentifier = errors.New("invalid GitHub identifier")

	// ErrInvalidPRNumber indicates an invalid PR number.
	ErrInvalidPRNumber = errors.New("invalid PR number")

	// ErrPathTraversal indicates a path contains traversal attempts.
	ErrPathTraversal = errors.New("path traversal detected")

	// ErrMissingToken indicates GitHubToken is missing.
	ErrMissingToken = errors.New("GitHub token is required")
)

// Validation patterns
var (
	// gitHubIdentifierPattern matches valid GitHub owner/repo names.
	// GitHub allows alphanumeric, hyphen, underscore. Max 39 chars for username, 100 for repo.
	// See: https://github.com/dead-claudia/github-limits
	gitHubIdentifierPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9-]{0,38}$`)
	gitHubRepoPattern       = regexp.MustCompile(`^[a-zA-Z0-9_.-]{1,100}$`)
)

// ValidateComprehensive performs comprehensive validation with GitHub identifier patterns,
// path traversal checks, and other security validations.
func (c *VersionValidationConfig) ValidateComprehensive() error {
	// Validate Owner (GitHub username)
	if c.Owner == "" {
		return fmt.Errorf("%w: Owner", ErrEmptyField)
	}
	if !gitHubIdentifierPattern.MatchString(c.Owner) {
		return fmt.Errorf("%w: Owner must be valid GitHub username (alphanumeric, hyphen, 1-39 chars): %s", ErrInvalidGitHubIdentifier, c.Owner)
	}

	// Validate Repo (GitHub repository name)
	if c.Repo == "" {
		return fmt.Errorf("%w: Repo", ErrEmptyField)
	}
	if !gitHubRepoPattern.MatchString(c.Repo) {
		return fmt.Errorf("%w: Repo must be valid GitHub repository name (alphanumeric, dot, hyphen, underscore, 1-100 chars): %s", ErrInvalidGitHubIdentifier, c.Repo)
	}

	// Validate PRNumber
	if c.PRNumber <= 0 {
		return fmt.Errorf("%w: PRNumber must be positive, got %d", ErrInvalidPRNumber, c.PRNumber)
	}

	// Validate HeadSHA (git commit SHA - 40 hex chars or 7+ for short SHA)
	if c.HeadSHA == "" {
		return fmt.Errorf("%w: HeadSHA", ErrEmptyField)
	}
	if !isValidGitSHA(c.HeadSHA) {
		return fmt.Errorf("%w: HeadSHA must be valid git SHA (7-40 hex chars): %s", ErrInvalidInput, c.HeadSHA)
	}

	// Validate GitHubToken
	if !c.GitHubToken.IsSet() {
		return fmt.Errorf("%w: GitHubToken", ErrMissingToken)
	}

	return nil
}

// ValidateComprehensive performs comprehensive validation with GitHub identifier patterns,
// path traversal checks, and other security validations.
func (c *PluginUpdateValidationConfig) ValidateComprehensive() error {
	// Validate Owner
	if c.Owner == "" {
		return fmt.Errorf("%w: Owner", ErrEmptyField)
	}
	if !gitHubIdentifierPattern.MatchString(c.Owner) {
		return fmt.Errorf("%w: Owner must be valid GitHub username: %s", ErrInvalidGitHubIdentifier, c.Owner)
	}

	// Validate Repo
	if c.Repo == "" {
		return fmt.Errorf("%w: Repo", ErrEmptyField)
	}
	if !gitHubRepoPattern.MatchString(c.Repo) {
		return fmt.Errorf("%w: Repo must be valid GitHub repository name: %s", ErrInvalidGitHubIdentifier, c.Repo)
	}

	// Validate PRNumber
	if c.PRNumber <= 0 {
		return fmt.Errorf("%w: PRNumber must be positive, got %d", ErrInvalidPRNumber, c.PRNumber)
	}

	// Validate BaseBranch (branch names can contain /, but no path traversal)
	if c.BaseBranch == "" {
		return fmt.Errorf("%w: BaseBranch", ErrEmptyField)
	}
	if err := validateBranchName(c.BaseBranch); err != nil {
		return fmt.Errorf("BaseBranch: %w", err)
	}

	// Validate HeadBranch
	if c.HeadBranch == "" {
		return fmt.Errorf("%w: HeadBranch", ErrEmptyField)
	}
	if err := validateBranchName(c.HeadBranch); err != nil {
		return fmt.Errorf("HeadBranch: %w", err)
	}

	// Validate HeadSHA
	if c.HeadSHA == "" {
		return fmt.Errorf("%w: HeadSHA", ErrEmptyField)
	}
	if !isValidGitSHA(c.HeadSHA) {
		return fmt.Errorf("%w: HeadSHA must be valid git SHA: %s", ErrInvalidInput, c.HeadSHA)
	}

	// Validate GitHubToken
	if !c.GitHubToken.IsSet() {
		return fmt.Errorf("%w: GitHubToken", ErrMissingToken)
	}

	return nil
}

// ValidateComprehensive performs comprehensive validation with GitHub identifier patterns,
// path traversal checks, and other security validations.
func (i *FetchFileContentInput) ValidateComprehensive() error {
	// Validate Owner
	if i.Owner == "" {
		return fmt.Errorf("%w: Owner", ErrEmptyField)
	}
	if !gitHubIdentifierPattern.MatchString(i.Owner) {
		return fmt.Errorf("%w: Owner must be valid GitHub username: %s", ErrInvalidGitHubIdentifier, i.Owner)
	}

	// Validate Repo
	if i.Repo == "" {
		return fmt.Errorf("%w: Repo", ErrEmptyField)
	}
	if !gitHubRepoPattern.MatchString(i.Repo) {
		return fmt.Errorf("%w: Repo must be valid GitHub repository name: %s", ErrInvalidGitHubIdentifier, i.Repo)
	}

	// Validate Path (file path in repo)
	if i.Path == "" {
		return fmt.Errorf("%w: Path", ErrEmptyField)
	}
	if err := validateFilePath(i.Path); err != nil {
		return fmt.Errorf("Path: %w", err)
	}

	// Validate Ref (branch or SHA)
	if i.Ref == "" {
		return fmt.Errorf("%w: Ref", ErrEmptyField)
	}
	// Ref can be branch name or SHA, validate both
	if !isValidGitSHA(i.Ref) && validateBranchName(i.Ref) != nil {
		return fmt.Errorf("%w: Ref must be valid branch name or git SHA: %s", ErrInvalidInput, i.Ref)
	}

	// Validate GitHubToken
	if !i.GitHubToken.IsSet() {
		return fmt.Errorf("%w: GitHubToken", ErrMissingToken)
	}

	return nil
}

// ValidateComprehensive performs comprehensive validation with GitHub identifier patterns,
// path traversal checks, and other security validations.
func (i *PostVersionCommentInput) ValidateComprehensive() error {
	// Validate Owner
	if i.Owner == "" {
		return fmt.Errorf("%w: Owner", ErrEmptyField)
	}
	if !gitHubIdentifierPattern.MatchString(i.Owner) {
		return fmt.Errorf("%w: Owner must be valid GitHub username: %s", ErrInvalidGitHubIdentifier, i.Owner)
	}

	// Validate Repo
	if i.Repo == "" {
		return fmt.Errorf("%w: Repo", ErrEmptyField)
	}
	if !gitHubRepoPattern.MatchString(i.Repo) {
		return fmt.Errorf("%w: Repo must be valid GitHub repository name: %s", ErrInvalidGitHubIdentifier, i.Repo)
	}

	// Validate PRNumber
	if i.PRNumber <= 0 {
		return fmt.Errorf("%w: PRNumber must be positive, got %d", ErrInvalidPRNumber, i.PRNumber)
	}

	// VersionFile and PluginVersion can be any string (including empty for removal activities)

	// Validate GitHubToken
	if !i.GitHubToken.IsSet() {
		return fmt.Errorf("%w: GitHubToken", ErrMissingToken)
	}

	return nil
}

// ValidateComprehensive performs comprehensive validation with GitHub identifier patterns,
// path traversal checks, and other security validations.
func (i *FetchPRFilesInput) ValidateComprehensive() error {
	// Validate Owner
	if i.Owner == "" {
		return fmt.Errorf("%w: Owner", ErrEmptyField)
	}
	if !gitHubIdentifierPattern.MatchString(i.Owner) {
		return fmt.Errorf("%w: Owner must be valid GitHub username: %s", ErrInvalidGitHubIdentifier, i.Owner)
	}

	// Validate Repo
	if i.Repo == "" {
		return fmt.Errorf("%w: Repo", ErrEmptyField)
	}
	if !gitHubRepoPattern.MatchString(i.Repo) {
		return fmt.Errorf("%w: Repo must be valid GitHub repository name: %s", ErrInvalidGitHubIdentifier, i.Repo)
	}

	// Validate PRNumber
	if i.PRNumber <= 0 {
		return fmt.Errorf("%w: PRNumber must be positive, got %d", ErrInvalidPRNumber, i.PRNumber)
	}

	// Validate GitHubToken
	if !i.GitHubToken.IsSet() {
		return fmt.Errorf("%w: GitHubToken", ErrMissingToken)
	}

	return nil
}

// Helper functions

// validateFilePath validates a file path to prevent traversal attacks.
// Follows the same pattern as internal/workflows/version_validation_activities.go
func validateFilePath(path string) error {
	// Reject empty paths
	if path == "" {
		return fmt.Errorf("%w: path is empty", ErrInvalidInput)
	}

	// Check for path traversal sequences BEFORE any cleaning
	// This prevents attacks like: ../../etc/passwd, ./../secret, etc/../../root
	if strings.Contains(path, "..") {
		return fmt.Errorf("%w: path contains '..' sequence: %s", ErrPathTraversal, path)
	}

	// Reject absolute paths (files should be relative to repo root)
	if strings.HasPrefix(path, "/") {
		return fmt.Errorf("%w: absolute paths not allowed: %s", ErrInvalidInput, path)
	}

	// Additional safety: reject paths starting with ./ (normalized form should not start with dot)
	if strings.HasPrefix(path, "./") {
		return fmt.Errorf("%w: path should not start with './': %s", ErrInvalidInput, path)
	}

	return nil
}

// validateBranchName validates a git branch name.
// Git branch naming rules:
// - No path traversal (..)
// - No double dots (..)
// - No spaces
// - Cannot start/end with /
// - Cannot contain consecutive slashes
func validateBranchName(branch string) error {
	if branch == "" {
		return fmt.Errorf("%w: branch name is empty", ErrInvalidInput)
	}

	// Check for path traversal
	if strings.Contains(branch, "..") {
		return fmt.Errorf("%w: branch name contains '..' sequence: %s", ErrPathTraversal, branch)
	}

	// Check for invalid characters
	if strings.Contains(branch, " ") {
		return fmt.Errorf("%w: branch name contains spaces: %s", ErrInvalidInput, branch)
	}

	// Check for leading/trailing slashes
	if strings.HasPrefix(branch, "/") || strings.HasSuffix(branch, "/") {
		return fmt.Errorf("%w: branch name cannot start or end with '/': %s", ErrInvalidInput, branch)
	}

	// Check for consecutive slashes
	if strings.Contains(branch, "//") {
		return fmt.Errorf("%w: branch name contains consecutive slashes: %s", ErrInvalidInput, branch)
	}

	// Check for other git-forbidden sequences
	forbidden := []string{"~", "^", ":", "?", "*", "[", "\\", "@{"}
	for _, seq := range forbidden {
		if strings.Contains(branch, seq) {
			return fmt.Errorf("%w: branch name contains forbidden sequence '%s': %s", ErrInvalidInput, seq, branch)
		}
	}

	return nil
}

// isValidGitSHA checks if a string is a valid git SHA (full or short).
func isValidGitSHA(sha string) bool {
	// Git SHAs are hex strings, typically 40 chars (full) or 7+ chars (short)
	if len(sha) < 7 || len(sha) > 40 {
		return false
	}

	// Check if all characters are hex
	for _, c := range sha {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}

	return true
}
