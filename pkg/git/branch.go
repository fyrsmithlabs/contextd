// Package git provides Git repository utilities for contextd.
//
// This package includes functions for detecting the current Git branch,
// identifying main branches, and handling Git worktrees. It's designed
// to support contextd's delta collection model where feature branches
// store only changed files.
package git

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var (
	// ErrNotGitRepo indicates the directory is not a Git repository
	ErrNotGitRepo = errors.New("not a git repository")

	// ErrHeadNotFound indicates the .git/HEAD file is missing
	ErrHeadNotFound = errors.New("HEAD file not found")
)

// DetectBranch detects the current Git branch from a project directory.
//
// It reads the .git/HEAD file to determine the branch name. If the HEAD
// is detached (not pointing to a branch), it returns "detached".
//
// Returns:
//   - Branch name (e.g., "main", "feature/v3-rebuild")
//   - "detached" if HEAD is detached
//   - Error if not a Git repo or HEAD file is unreadable
//
// Example:
//
//	branch, err := DetectBranch("/path/to/project")
//	if err != nil {
//	    // Handle error
//	}
//	if branch == "main" {
//	    // Use main collection
//	} else {
//	    // Use delta collection
//	}
func DetectBranch(projectPath string) (string, error) {
	// Check if .git directory exists
	gitDir := filepath.Join(projectPath, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		return "", fmt.Errorf("%w: %s", ErrNotGitRepo, projectPath)
	}

	// Read HEAD file
	headFile := filepath.Join(gitDir, "HEAD")
	content, err := os.ReadFile(headFile)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("%w: %s", ErrHeadNotFound, headFile)
		}
		return "", fmt.Errorf("reading HEAD file: %w", err)
	}

	// Parse HEAD content
	head := strings.TrimSpace(string(content))

	// Empty HEAD file indicates detached state
	if head == "" {
		return "detached", nil
	}

	// Check if HEAD points to a branch (ref: refs/heads/<branch>)
	if strings.HasPrefix(head, "ref: refs/heads/") {
		branch := strings.TrimPrefix(head, "ref: refs/heads/")
		return branch, nil
	}

	// If HEAD contains a commit hash (detached HEAD)
	return "detached", nil
}

// IsMainBranch checks if the given branch name is a main branch.
//
// Main branches are typically "main" or "master". These branches
// use full collections in contextd's delta model, while feature
// branches use delta collections that only store changed files.
//
// Returns:
//   - true if branch is "main" or "master"
//   - false otherwise
//
// Example:
//
//	if IsMainBranch(branch) {
//	    collectionName = "project_123/main"
//	} else {
//	    collectionName = fmt.Sprintf("project_123/%s", branch)
//	}
func IsMainBranch(branch string) bool {
	return branch == "main" || branch == "master"
}
