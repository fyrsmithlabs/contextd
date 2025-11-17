// Package collections provides collection naming and management for contextd.
//
// Collection names follow the format: owner_<hash>/project_<hash>/<branch>
// This ensures proper isolation between owners and projects while supporting
// the delta collection model for feature branches.
//
// Example:
//
//	name, err := collections.GenerateName(
//	    "owner_2bd806c9",
//	    "project_abc123",
//	    "feature/v3-rebuild",
//	)
//	// Result: "owner_2bd806c9/project_abc123/feature_v3-rebuild"
package collections

import (
	"errors"
	"fmt"
	"strings"
)

var (
	// ErrInvalidOwnerID indicates empty or invalid owner ID
	ErrInvalidOwnerID = errors.New("invalid owner ID")

	// ErrInvalidProjectID indicates empty or invalid project ID
	ErrInvalidProjectID = errors.New("invalid project ID")

	// ErrInvalidBranch indicates empty or invalid branch name
	ErrInvalidBranch = errors.New("invalid branch name")

	// ErrInvalidCollectionName indicates malformed collection name
	ErrInvalidCollectionName = errors.New("invalid collection name format")
)

// GenerateName generates a collection name from owner ID, project ID, and branch.
//
// The collection name format is: owner_<hash>/project_<hash>/<branch>
// Branch names containing slashes are sanitized (/ replaced with _).
//
// Parameters:
//   - ownerID: Owner identifier (e.g., "owner_2bd806c9")
//   - projectID: Project identifier (e.g., "project_abc123")
//   - branch: Git branch name (e.g., "main", "feature/v3-rebuild")
//
// Returns:
//   - Collection name string
//   - Error if any parameter is empty
//
// Example:
//
//	name, err := GenerateName("owner_abc", "project_def", "feature/auth")
//	// Result: "owner_abc/project_def/feature_auth"
func GenerateName(ownerID, projectID, branch string) (string, error) {
	// Validate inputs
	if ownerID == "" {
		return "", fmt.Errorf("%w: owner ID required", ErrInvalidOwnerID)
	}
	if projectID == "" {
		return "", fmt.Errorf("%w: project ID required", ErrInvalidProjectID)
	}
	if branch == "" {
		return "", fmt.Errorf("%w: branch required", ErrInvalidBranch)
	}

	// Sanitize branch name (replace slashes with underscores)
	sanitizedBranch := SanitizeBranch(branch)

	// Generate collection name
	return fmt.Sprintf("%s/%s/%s", ownerID, projectID, sanitizedBranch), nil
}

// SanitizeBranch sanitizes a branch name for use in collection names.
//
// It replaces forward slashes with underscores to ensure the branch
// name doesn't interfere with the collection name structure.
//
// Example:
//
//	branch := SanitizeBranch("feature/auth")
//	// Result: "feature_auth"
func SanitizeBranch(branch string) string {
	return strings.ReplaceAll(branch, "/", "_")
}

// ParseCollectionName parses a collection name into its components.
//
// It extracts the owner ID, project ID, and branch from a collection name.
// The collection name must follow the format: owner_<hash>/project_<hash>/<branch>
//
// Returns:
//   - ownerID: Owner identifier
//   - projectID: Project identifier
//   - branch: Branch name (sanitized)
//   - error: If collection name is invalid
//
// Example:
//
//	owner, project, branch, err := ParseCollectionName("owner_abc/project_def/main")
//	// owner = "owner_abc", project = "project_def", branch = "main"
func ParseCollectionName(collectionName string) (string, string, string, error) {
	if collectionName == "" {
		return "", "", "", fmt.Errorf("%w: collection name required", ErrInvalidCollectionName)
	}

	parts := strings.Split(collectionName, "/")
	if len(parts) != 3 {
		return "", "", "", fmt.Errorf("%w: expected format owner_<hash>/project_<hash>/<branch>, got %d parts", ErrInvalidCollectionName, len(parts))
	}

	return parts[0], parts[1], parts[2], nil
}
