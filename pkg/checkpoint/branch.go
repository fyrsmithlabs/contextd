package checkpoint

import (
	"github.com/go-git/go-git/v5"
)

// detectGitBranch auto-detects the current git branch from a project path.
//
// Returns the branch name if the path is a git repository, or empty string
// if not a git repo or detection fails.
//
// This function is used to automatically populate the Branch field when
// saving checkpoints.
func detectGitBranch(projectPath string) (string, error) {
	// Open git repository
	repo, err := git.PlainOpen(projectPath)
	if err != nil {
		// Not a git repository or can't open - return empty, not an error
		return "", nil
	}

	// Get HEAD reference
	head, err := repo.Head()
	if err != nil {
		// Can't get HEAD (detached HEAD, bare repo, etc.) - return empty
		return "", nil
	}

	// Extract branch name from reference
	// head.Name() returns refs/heads/branch-name format
	if head.Name().IsBranch() {
		return head.Name().Short(), nil
	}

	// Not on a branch (detached HEAD) - return empty
	return "", nil
}
