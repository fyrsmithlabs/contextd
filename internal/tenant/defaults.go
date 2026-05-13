package tenant

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"

	"github.com/fyrsmithlabs/contextd/internal/sanitize"
)

// GetDefaultTenantID returns the default tenant ID for single-tenant mode.
// Uses git user.name from global config if available, falls back to OS username.
func GetDefaultTenantID() string {
	return GetTenantIDForPath("")
}

// GetDefaultProjectID returns a best-effort project ID derived from the
// current working directory. Returns "" when no project can be derived
// (e.g. unreadable CWD); callers should treat that as fail-closed.
//
// This is the floor of contextd's isolation model: a solo developer running
// without explicit configuration still gets per-repo separation by virtue of
// the working directory's basename.
func GetDefaultProjectID() string {
	cwd, err := os.Getwd()
	if err != nil || cwd == "" {
		return ""
	}
	return GetProjectIDForPath(cwd)
}

// GetProjectIDForPath returns a sanitized project identifier derived from a
// filesystem path. Returns "" if the path is empty or cannot be reduced to a
// valid identifier.
func GetProjectIDForPath(path string) string {
	if path == "" {
		return ""
	}
	base := filepath.Base(filepath.Clean(path))
	if base == "" || base == "." || base == string(filepath.Separator) {
		return ""
	}
	id := sanitize.Identifier(strings.ToLower(base))
	if id == "" || id == "default" {
		// sanitize.Identifier returns "default" when no valid characters remain,
		// which is not actually descriptive of the project — treat as unresolved.
		return ""
	}
	return id
}

// DefaultsForPath returns a (tenantID, projectID) pair using the given path as
// a hint. Empty path falls back to CWD-based derivation. This is the canonical
// entry point used by vectorstore to resolve missing tenant context.
func DefaultsForPath(path string) (tenantID, projectID string) {
	tenantID = GetTenantIDForPath(path)
	if path == "" {
		projectID = GetDefaultProjectID()
	} else {
		projectID = GetProjectIDForPath(path)
	}
	return tenantID, projectID
}

// GetTenantIDForPath returns a tenant ID, preferring GitHub username from repo remote.
// Priority: GitHub username from remote → git user.name → $USER → "local"
func GetTenantIDForPath(repoPath string) string {
	// Try to get GitHub username from repo remote
	if repoPath != "" {
		if username := getGitHubUsernameFromRepo(repoPath); username != "" {
			return sanitize.Identifier(strings.ToLower(username))
		}
	}

	// Try git global config user.name
	cfg, err := config.LoadConfig(config.GlobalScope)
	if err == nil && cfg.User.Name != "" {
		name := strings.ToLower(cfg.User.Name)
		name = strings.ReplaceAll(name, " ", "_")
		return sanitize.Identifier(name)
	}

	// Fall back to OS username
	if user := os.Getenv("USER"); user != "" {
		return sanitize.Identifier(strings.ToLower(user))
	}

	return "local"
}

// getGitHubUsernameFromRepo extracts the GitHub username from the origin remote URL.
func getGitHubUsernameFromRepo(repoPath string) string {
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return ""
	}

	remote, err := repo.Remote("origin")
	if err != nil {
		return ""
	}

	urls := remote.Config().URLs
	if len(urls) == 0 {
		return ""
	}

	return parseGitHubUsername(urls[0])
}

// parseGitHubUsername extracts the username from a GitHub URL.
// Supports: git@github.com:user/repo.git, https://github.com/user/repo.git
func parseGitHubUsername(url string) string {
	// SSH format: git@github.com:user/repo.git
	sshPattern := regexp.MustCompile(`git@github\.com:([^/]+)/`)
	if matches := sshPattern.FindStringSubmatch(url); len(matches) > 1 {
		return matches[1]
	}

	// HTTPS format: https://github.com/user/repo.git
	httpsPattern := regexp.MustCompile(`github\.com/([^/]+)/`)
	if matches := httpsPattern.FindStringSubmatch(url); len(matches) > 1 {
		return matches[1]
	}

	return ""
}
