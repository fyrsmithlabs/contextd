package tenant

import (
	"os"
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

