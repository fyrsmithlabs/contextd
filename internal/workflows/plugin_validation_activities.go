package workflows

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/go-github/v57/github"
)

// FetchPRFilesActivity fetches the list of files changed in a PR.
func FetchPRFilesActivity(ctx context.Context, input FetchPRFilesInput) ([]FileChange, error) {
	// Create GitHub client
	client, err := NewGitHubClient(ctx, input.GitHubToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create GitHub client: %w", err)
	}

	// Fetch PR files
	opts := &github.ListOptions{PerPage: 100}
	var allFiles []*github.CommitFile
	for {
		files, resp, err := client.PullRequests.ListFiles(ctx, input.Owner, input.Repo, input.PRNumber, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to list PR files: %w", err)
		}
		allFiles = append(allFiles, files...)
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	// Convert to FileChange
	result := make([]FileChange, 0, len(allFiles))
	for _, f := range allFiles {
		result = append(result, FileChange{
			Path:      f.GetFilename(),
			Status:    f.GetStatus(),
			Additions: f.GetAdditions(),
			Deletions: f.GetDeletions(),
		})
	}

	return result, nil
}

// CategorizeFilesActivity categorizes files by whether they affect the plugin.
func CategorizeFilesActivity(ctx context.Context, input CategorizeFilesInput) (*CategorizedFiles, error) {
	result := &CategorizedFiles{
		CodeFiles:   make([]string, 0),
		PluginFiles: make([]string, 0),
		OtherFiles:  make([]string, 0),
	}

	// Patterns that indicate code files affecting plugin behavior
	codePatterns := []*regexp.Regexp{
		regexp.MustCompile(`^internal/mcp/tools\.go$`),
		regexp.MustCompile(`^internal/mcp/handlers/.*\.go$`),
		regexp.MustCompile(`^internal/.*/service\.go$`),
		regexp.MustCompile(`^internal/config/(types|config)\.go$`),
	}

	// Plugin files pattern
	pluginPattern := regexp.MustCompile(`^\.claude-plugin/`)

	for _, file := range input.Files {
		path := file.Path

		// Check if it's a plugin file
		if pluginPattern.MatchString(path) {
			result.PluginFiles = append(result.PluginFiles, path)
			continue
		}

		// Check if it's a code file that affects plugin
		isCodeFile := false
		for _, pattern := range codePatterns {
			if pattern.MatchString(path) {
				isCodeFile = true
				break
			}
		}

		if isCodeFile {
			result.CodeFiles = append(result.CodeFiles, path)
		} else {
			result.OtherFiles = append(result.OtherFiles, path)
		}
	}

	return result, nil
}

// ValidatePluginSchemasActivity validates JSON schemas in plugin files.
func ValidatePluginSchemasActivity(ctx context.Context, input ValidateSchemasInput) (*SchemaValidationResult, error) {
	result := &SchemaValidationResult{
		Valid:  true,
		Errors: make([]string, 0),
	}

	// Create GitHub client
	client, err := NewGitHubClient(ctx, input.GitHubToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create GitHub client: %w", err)
	}

	// Fetch file content at HEAD SHA
	fileContent, _, _, err := client.Repositories.GetContents(ctx, input.Owner, input.Repo, input.FilePath, &github.RepositoryContentGetOptions{
		Ref: input.HeadSHA,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get file content: %w", err)
	}

	// Decode content
	content, err := fileContent.GetContent()
	if err != nil {
		return nil, fmt.Errorf("failed to decode file content: %w", err)
	}

	// Validate JSON
	var jsonData interface{}
	if err := json.Unmarshal([]byte(content), &jsonData); err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("Invalid JSON in %s: %v", input.FilePath, err))
		return result, nil // Early return - don't process invalid JSON
	}

	// Additional validation: check for required fields in MCP tools schema
	if input.FilePath == ".claude-plugin/schemas/contextd-mcp-tools.schema.json" {
		schemaMap, ok := jsonData.(map[string]interface{})
		if !ok {
			result.Valid = false
			result.Errors = append(result.Errors, "Schema is not a JSON object")
		} else {
			// Check for required top-level fields
			requiredFields := []string{"tools"}
			for _, field := range requiredFields {
				if _, exists := schemaMap[field]; !exists {
					result.Valid = false
					result.Errors = append(result.Errors, fmt.Sprintf("Missing required field: %s", field))
				}
			}
		}
	}

	return result, nil
}

// PostReminderCommentActivity posts a reminder comment to the PR.
func PostReminderCommentActivity(ctx context.Context, input PostCommentInput) (*PostCommentResult, error) {
	client, err := NewGitHubClient(ctx, input.GitHubToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create GitHub client: %w", err)
	}

	// Build comment body
	body := buildReminderComment(input.CodeFiles)

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

	// Look for existing bot comment
	var existingComment *github.IssueComment
	for _, comment := range allComments {
		if strings.Contains(comment.GetBody(), "⚠️ Claude Plugin Update Reminder") {
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

// PostSuccessCommentActivity posts a success message to the PR.
func PostSuccessCommentActivity(ctx context.Context, input PostCommentInput) (*PostCommentResult, error) {
	client, err := NewGitHubClient(ctx, input.GitHubToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create GitHub client: %w", err)
	}

	// Build success message
	var b strings.Builder
	b.WriteString("## ✅ Claude Plugin Updated\n\n")
	b.WriteString("Great! This PR includes updates to the Claude plugin alongside code changes.\n\n")
	b.WriteString("Plugin schemas have been validated and are correct.\n")

	// Include agent validation results if available
	if input.AgentValidation != nil {
		b.WriteString("\n")
		b.WriteString(buildValidationComment(input.AgentValidation))
	}

	body := b.String()

	// Check if we already posted a success comment (with pagination)
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

	// Look for existing success comment
	var existingComment *github.IssueComment
	for _, comment := range allComments {
		if strings.Contains(comment.GetBody(), "✅ Claude Plugin Updated") {
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

// Helper functions

func buildReminderComment(codeFiles []string) string {
	var b strings.Builder

	b.WriteString("## ⚠️ Claude Plugin Update Reminder\n\n")
	b.WriteString("This PR modifies files that may require Claude plugin updates:\n\n")

	b.WriteString("### Changed Files\n```\n")
	for _, file := range codeFiles {
		b.WriteString(file)
		b.WriteString("\n")
	}
	b.WriteString("```\n\n")

	b.WriteString("### Required Actions\n")
	b.WriteString("Please review the **Claude Plugin Updates** section in the PR description and check applicable items:\n\n")
	b.WriteString("- [ ] Update MCP tool schemas if tools added/changed\n")
	b.WriteString("- [ ] Update/add relevant skills for new features\n")
	b.WriteString("- [ ] Update command documentation\n")
	b.WriteString("- [ ] Review and update code examples in skills\n\n")

	b.WriteString("### Files to Check\n")
	b.WriteString("- `.claude-plugin/schemas/contextd-mcp-tools.schema.json` - MCP tool definitions\n")
	b.WriteString("- `.claude-plugin/skills/*/SKILL.md` - Skill documentation\n")
	b.WriteString("- `.claude-plugin/commands/*.md` - Command documentation\n")
	b.WriteString("- `.claude-plugin/includes/*.md` - Shared documentation\n\n")

	b.WriteString("See [CLAUDE.md Priority #3](https://github.com/fyrsmithlabs/contextd/blob/main/CLAUDE.md#critical-update-claude-plugin-on-changes-priority-3) for details.\n\n")
	b.WriteString("---\n\n")
	b.WriteString("**Note**: This is a reminder, not a blocker. If your changes don't affect user-facing functionality, you can check \"Not applicable\" in the PR description.\n")

	return b.String()
}
