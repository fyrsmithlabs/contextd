package workflows

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/go-github/v57/github"
)

// FetchFileContentActivity fetches the content of a single file from GitHub.
func FetchFileContentActivity(ctx context.Context, input FetchFileContentInput) (string, error) {
	// Validate file path to prevent path traversal attacks
	allowedPaths := map[string]bool{
		"VERSION":                    true,
		".claude-plugin/plugin.json": true,
	}

	if !allowedPaths[input.Path] {
		return "", fmt.Errorf("invalid file path: %s (only VERSION and .claude-plugin/plugin.json are allowed)", input.Path)
	}

	// Additional safety checks
	if strings.Contains(input.Path, "..") {
		return "", fmt.Errorf("path traversal detected in path: %s", input.Path)
	}
	if strings.HasPrefix(input.Path, "/") {
		return "", fmt.Errorf("absolute paths not allowed: %s", input.Path)
	}

	// Create GitHub client
	client, err := NewGitHubClient(ctx, input.GitHubToken)
	if err != nil {
		return "", fmt.Errorf("failed to create GitHub client: %w", err)
	}

	// Fetch file content
	fileContent, _, _, err := client.Repositories.GetContents(ctx, input.Owner, input.Repo, input.Path, &github.RepositoryContentGetOptions{
		Ref: input.Ref,
	})
	if err != nil {
		return "", fmt.Errorf("failed to get file content: %w", err)
	}

	// Decode content
	content, err := fileContent.GetContent()
	if err != nil {
		return "", fmt.Errorf("failed to decode file content: %w", err)
	}

	return content, nil
}

// PostVersionMismatchCommentActivity posts a version mismatch comment to the PR.
func PostVersionMismatchCommentActivity(ctx context.Context, input PostVersionCommentInput) (*PostCommentResult, error) {
	client, err := NewGitHubClient(ctx, input.GitHubToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create GitHub client: %w", err)
	}

	// Build comment body
	body := buildVersionMismatchComment(input.VersionFile, input.PluginVersion)

	// Check if we already posted a comment (with pagination)
	opts := &github.IssueListCommentsOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}
	var allComments []*github.IssueComment
	for {
		comments, resp, err := client.Issues.ListComments(ctx, input.Owner, input.Repo, input.PRNumber, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to list comments: %w", err)
		}
		allComments = append(allComments, comments...)
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	// Look for existing version mismatch comment using the marker
	var existingComment *github.IssueComment
	for _, comment := range allComments {
		if strings.Contains(comment.GetBody(), versionValidationCommentMarker) {
			existingComment = comment
			break
		}
	}

	var commentURL string
	if existingComment != nil {
		// Update existing comment
		updated, _, err := client.Issues.EditComment(ctx, input.Owner, input.Repo, existingComment.GetID(), &github.IssueComment{
			Body: &body,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to update comment: %w", err)
		}
		commentURL = updated.GetHTMLURL()
	} else {
		// Create new comment
		created, _, err := client.Issues.CreateComment(ctx, input.Owner, input.Repo, input.PRNumber, &github.IssueComment{
			Body: &body,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create comment: %w", err)
		}
		commentURL = created.GetHTMLURL()
	}

	return &PostCommentResult{URL: commentURL}, nil
}

// RemoveVersionMismatchCommentActivity removes the version mismatch comment if it exists.
// This is called when versions match to clean up any previous mismatch comments.
func RemoveVersionMismatchCommentActivity(ctx context.Context, input PostVersionCommentInput) error {
	client, err := NewGitHubClient(ctx, input.GitHubToken)
	if err != nil {
		return fmt.Errorf("failed to create GitHub client: %w", err)
	}

	// Find the existing comment (with pagination)
	opts := &github.IssueListCommentsOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}
	var allComments []*github.IssueComment
	for {
		comments, resp, err := client.Issues.ListComments(ctx, input.Owner, input.Repo, input.PRNumber, opts)
		if err != nil {
			return fmt.Errorf("failed to list comments: %w", err)
		}
		allComments = append(allComments, comments...)
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	// Look for existing version mismatch comment using the marker
	for _, comment := range allComments {
		if strings.Contains(comment.GetBody(), versionValidationCommentMarker) {
			// Delete the comment
			_, err := client.Issues.DeleteComment(ctx, input.Owner, input.Repo, comment.GetID())
			if err != nil {
				return fmt.Errorf("failed to delete comment: %w", err)
			}
			return nil
		}
	}

	// No comment found, nothing to do
	return nil
}

// Comment marker for identifying bot comments
const versionValidationCommentMarker = "<!-- contextd-version-validation-bot -->"

// buildVersionMismatchComment builds the comment body for version mismatches.
func buildVersionMismatchComment(versionFile string, pluginVersion string) string {
	var b strings.Builder

	// Add HTML comment marker for reliable identification
	b.WriteString(versionValidationCommentMarker + "\n")
	b.WriteString("## ⚠️ Version Mismatch Detected\n\n")
	b.WriteString("The version in the `VERSION` file does not match the version in `.claude-plugin/plugin.json`.\n\n")

	b.WriteString("### Version Details\n\n")
	b.WriteString("| Location | Version |\n")
	b.WriteString("|----------|--------|\n")
	b.WriteString(fmt.Sprintf("| `VERSION` file | `%s` |\n", versionFile))
	b.WriteString(fmt.Sprintf("| `.claude-plugin/plugin.json` | `%s` |\n", pluginVersion))
	b.WriteString("\n")

	b.WriteString("### How to Fix\n\n")
	b.WriteString("Run the version sync script to automatically update all version-dependent files:\n\n")
	b.WriteString("```bash\n")
	b.WriteString("./scripts/sync-version.sh\n")
	b.WriteString("git add VERSION .claude-plugin/plugin.json\n")
	b.WriteString("\n")
	b.WriteString("# If you're the only one working on this PR:\n")
	b.WriteString("git commit --amend --no-edit\n")
	b.WriteString("git push -f\n")
	b.WriteString("\n")
	b.WriteString("# If others are collaborating on this PR:\n")
	b.WriteString("git commit -m \"chore: sync version files\"\n")
	b.WriteString("git push\n")
	b.WriteString("```\n\n")

	b.WriteString("### Why This Matters\n\n")
	b.WriteString("- The `VERSION` file is the **single source of truth** for all version information\n")
	b.WriteString("- Version mismatches can cause confusion about which version is installed\n")
	b.WriteString("- The sync script ensures consistency across all version-dependent files\n\n")

	b.WriteString("See [VERSIONING.md](https://github.com/fyrsmithlabs/contextd/blob/main/VERSIONING.md) for details on the version management workflow.\n\n")

	b.WriteString("---\n\n")
	b.WriteString("*This check is automated via Temporal workflows. Once you fix the versions and push, this comment will be automatically removed.*\n")

	return b.String()
}
