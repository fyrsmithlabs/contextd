// Package sanitize provides shared identifier sanitization and input validation.
package sanitize

import (
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// Validation errors for security checks.
var (
	// ErrPathTraversal indicates a path contains directory traversal sequences.
	ErrPathTraversal = errors.New("path contains directory traversal")

	// ErrAbsolutePath indicates an absolute path was provided where relative was expected.
	ErrAbsolutePath = errors.New("absolute path not allowed")

	// ErrInvalidTenantID indicates the tenant ID format is invalid.
	ErrInvalidTenantID = errors.New("invalid tenant ID format")

	// ErrInvalidTeamID indicates the team ID format is invalid.
	ErrInvalidTeamID = errors.New("invalid team ID format")

	// ErrInvalidProjectID indicates the project ID format is invalid.
	ErrInvalidProjectID = errors.New("invalid project ID format")

	// ErrInvalidPattern indicates a glob/regex pattern is dangerous.
	ErrInvalidPattern = errors.New("invalid or dangerous pattern")

	// ErrEmptyPath indicates an empty path was provided.
	ErrEmptyPath = errors.New("path cannot be empty")
)

// identifierPattern matches valid sanitized identifiers: lowercase alphanumeric with underscores.
// Max 64 chars to match collection name constraints.
var identifierPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_]{0,62}[a-z0-9]?$`)

// dangerousPatternChars are characters that could cause ReDoS or shell injection in patterns.
var dangerousPatternChars = regexp.MustCompile(`[;\|\$\x60\\<>&\(\)\{\}]|\.{3,}|\*{3,}`)

// ValidatePath checks a path for security issues:
//   - No directory traversal (..)
//   - Resolves to absolute path and validates it stays within expected root
//   - Returns the cleaned, absolute path or an error
//
// If allowedRoot is empty, only traversal checks are performed.
// If allowedRoot is provided, the path must resolve within that directory.
func ValidatePath(path, allowedRoot string) (string, error) {
	if path == "" {
		return "", ErrEmptyPath
	}

	// Check for obvious traversal patterns before any processing
	if strings.Contains(path, "..") {
		return "", fmt.Errorf("%w: contains '..'", ErrPathTraversal)
	}

	// Clean the path to normalize it
	cleanPath := filepath.Clean(path)

	// Re-check after cleaning (handles edge cases like "foo/../..")
	if strings.Contains(cleanPath, "..") {
		return "", fmt.Errorf("%w: resolves to traversal", ErrPathTraversal)
	}

	// If path is not absolute, make it absolute for consistent validation
	absPath := cleanPath
	if !filepath.IsAbs(cleanPath) {
		var err error
		absPath, err = filepath.Abs(cleanPath)
		if err != nil {
			return "", fmt.Errorf("failed to resolve path: %w", err)
		}
	}

	// Final traversal check on absolute path
	if strings.Contains(absPath, "..") {
		return "", fmt.Errorf("%w: absolute path contains traversal", ErrPathTraversal)
	}

	// If allowed root is specified, ensure path is within it
	if allowedRoot != "" {
		absRoot, err := filepath.Abs(allowedRoot)
		if err != nil {
			return "", fmt.Errorf("failed to resolve allowed root: %w", err)
		}

		// Ensure the path starts with the allowed root
		// Use filepath.Rel to check relationship
		rel, err := filepath.Rel(absRoot, absPath)
		if err != nil {
			return "", fmt.Errorf("%w: path outside allowed root", ErrPathTraversal)
		}

		// If relative path starts with "..", it's outside the root
		if strings.HasPrefix(rel, "..") {
			return "", fmt.Errorf("%w: path escapes allowed root", ErrPathTraversal)
		}
	}

	return absPath, nil
}

// ValidateProjectPath validates a project path for MCP tool use.
// Returns the validated absolute path.
func ValidateProjectPath(path string) (string, error) {
	if path == "" {
		return "", ErrEmptyPath
	}

	// Use ValidatePath without a root constraint
	// MCP tools operate on user-specified paths, so we just prevent traversal
	return ValidatePath(path, "")
}

// SafeBasename returns the base name of a path after validation.
// This is a secure replacement for filepath.Base() on untrusted input.
func SafeBasename(path string) (string, error) {
	// Validate the path first
	cleanPath, err := ValidateProjectPath(path)
	if err != nil {
		return "", err
	}

	// Now safe to use filepath.Base
	base := filepath.Base(cleanPath)

	// Ensure base is not empty or a root indicator
	if base == "" || base == "." || base == "/" || base == string(filepath.Separator) {
		return "", fmt.Errorf("%w: invalid path base", ErrPathTraversal)
	}

	return base, nil
}

// ValidateTenantID checks that a tenant ID conforms to expected format.
// Tenant IDs should be lowercase alphanumeric with underscores, 1-64 chars.
func ValidateTenantID(id string) error {
	if id == "" {
		return fmt.Errorf("%w: empty", ErrInvalidTenantID)
	}

	// Check for path traversal characters
	if strings.ContainsAny(id, "/\\..") {
		return fmt.Errorf("%w: contains path characters", ErrInvalidTenantID)
	}

	// Validate format
	if !identifierPattern.MatchString(id) {
		return fmt.Errorf("%w: must be lowercase alphanumeric with underscores (1-64 chars)", ErrInvalidTenantID)
	}

	return nil
}

// ValidateTeamID checks that a team ID conforms to expected format.
// Team IDs follow the same rules as tenant IDs.
func ValidateTeamID(id string) error {
	if id == "" {
		// Empty team ID is allowed (optional field)
		return nil
	}

	// Check for path traversal characters
	if strings.ContainsAny(id, "/\\..") {
		return fmt.Errorf("%w: contains path characters", ErrInvalidTeamID)
	}

	// Validate format
	if !identifierPattern.MatchString(id) {
		return fmt.Errorf("%w: must be lowercase alphanumeric with underscores (1-64 chars)", ErrInvalidTeamID)
	}

	return nil
}

// ValidateProjectID checks that a project ID conforms to expected format.
// Project IDs follow the same rules as tenant IDs.
func ValidateProjectID(id string) error {
	if id == "" {
		// Empty project ID is allowed (optional field)
		return nil
	}

	// Check for path traversal characters
	if strings.ContainsAny(id, "/\\..") {
		return fmt.Errorf("%w: contains path characters", ErrInvalidProjectID)
	}

	// Validate format
	if !identifierPattern.MatchString(id) {
		return fmt.Errorf("%w: must be lowercase alphanumeric with underscores (1-64 chars)", ErrInvalidProjectID)
	}

	return nil
}

// ValidateGlobPattern checks a glob pattern for dangerous constructs.
// Returns nil if the pattern is safe, or an error describing the issue.
func ValidateGlobPattern(pattern string) error {
	if pattern == "" {
		return nil // Empty pattern is allowed
	}

	// Check for dangerous characters that could cause issues
	if dangerousPatternChars.MatchString(pattern) {
		return fmt.Errorf("%w: contains dangerous characters", ErrInvalidPattern)
	}

	// Check for path traversal in patterns
	if strings.Contains(pattern, "..") {
		return fmt.Errorf("%w: contains path traversal", ErrInvalidPattern)
	}

	// Validate the pattern compiles (catches malformed patterns)
	_, err := filepath.Match(pattern, "test")
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidPattern, err)
	}

	return nil
}

// ValidateGlobPatterns validates a slice of glob patterns.
func ValidateGlobPatterns(patterns []string) error {
	for i, p := range patterns {
		if err := ValidateGlobPattern(p); err != nil {
			return fmt.Errorf("pattern[%d] %q: %w", i, p, err)
		}
	}
	return nil
}

// ValidateRequiredID validates an identifier that must be non-empty.
// Use in authorization contexts where empty IDs could bypass access controls.
func ValidateRequiredID(id, fieldName string) error {
	if id == "" {
		return fmt.Errorf("%s is required and cannot be empty", fieldName)
	}

	// Check for path traversal characters
	if strings.ContainsAny(id, "/\\..") {
		return fmt.Errorf("invalid %s: contains path characters", fieldName)
	}

	// Validate format
	if !identifierPattern.MatchString(id) {
		return fmt.Errorf("invalid %s: must be lowercase alphanumeric with underscores (1-64 chars)", fieldName)
	}

	return nil
}

// SanitizeAndValidateTenantID sanitizes a tenant ID and validates the result.
// This is the recommended way to process user-provided tenant IDs.
func SanitizeAndValidateTenantID(id string) (string, error) {
	// First sanitize
	sanitized := Identifier(id)

	// Then validate the result
	if err := ValidateTenantID(sanitized); err != nil {
		return "", err
	}

	return sanitized, nil
}
