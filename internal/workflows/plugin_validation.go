// Package workflows provides Temporal workflow definitions for contextd automation.
package workflows

import (
	"fmt"
	"strings"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"

	"github.com/fyrsmithlabs/contextd/internal/config"
)

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

// FileChange represents a changed file in the PR.
type FileChange struct {
	Path      string
	Status    string // "added", "modified", "removed"
	Additions int
	Deletions int
}

// PluginUpdateValidationWorkflow validates plugin updates in a PR.
//
// This workflow:
// 1. Fetches PR file changes from GitHub
// 2. Detects if code changes require plugin updates
// 3. Validates plugin schemas if modified
// 4. Posts reminder comments if needed
// 5. Posts success message if plugin updated correctly
func PluginUpdateValidationWorkflow(ctx workflow.Context, config PluginUpdateValidationConfig) (*PluginUpdateValidationResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting plugin update validation",
		"owner", config.Owner,
		"repo", config.Repo,
		"pr", config.PRNumber)

	// Configure activity options
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 2 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	result := &PluginUpdateValidationResult{}

	// Step 1: Fetch PR file changes
	logger.Info("Fetching PR file changes")
	var fileChanges []FileChange
	err := workflow.ExecuteActivity(ctx, FetchPRFilesActivity, FetchPRFilesInput{
		Owner:       config.Owner,
		Repo:        config.Repo,
		PRNumber:    config.PRNumber,
		GitHubToken: config.GitHubToken,
	}).Get(ctx, &fileChanges)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("failed to fetch PR files: %v", err))
		return result, err
	}

	// Step 2: Categorize changes
	logger.Info("Categorizing file changes", "count", len(fileChanges))
	var categorized CategorizedFiles
	err = workflow.ExecuteActivity(ctx, CategorizeFilesActivity, CategorizeFilesInput{
		Files: fileChanges,
	}).Get(ctx, &categorized)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("failed to categorize files: %v", err))
		return result, err
	}

	result.CodeFilesChanged = categorized.CodeFiles
	result.PluginFilesChanged = categorized.PluginFiles
	result.NeedsUpdate = len(categorized.CodeFiles) > 0

	// Step 3: Validate plugin schemas if modified
	if len(categorized.PluginFiles) > 0 {
		logger.Info("Validating plugin schemas", "files", categorized.PluginFiles)

		// Find all JSON files in plugin changes
		var jsonFiles []string
		for _, file := range categorized.PluginFiles {
			if strings.HasSuffix(file, ".json") {
				jsonFiles = append(jsonFiles, file)
			}
		}

		// Create map of file statuses to filter deleted files
		fileStatusMap := make(map[string]string)
		for _, fc := range fileChanges {
			fileStatusMap[fc.Path] = fc.Status
		}

		// Filter out deleted JSON files before validation
		for i := 0; i < len(jsonFiles); {
			if fileStatusMap[jsonFiles[i]] == "removed" {
				// Skip deleted files - remove from slice
				jsonFiles = append(jsonFiles[:i], jsonFiles[i+1:]...)
			} else {
				i++
			}
		}

		// Validate each JSON file
		result.SchemaValid = true
		for _, jsonFile := range jsonFiles {
			var schemaResult SchemaValidationResult
			err = workflow.ExecuteActivity(ctx, ValidatePluginSchemasActivity, ValidateSchemasInput{
				Owner:       config.Owner,
				Repo:        config.Repo,
				HeadSHA:     config.HeadSHA,
				FilePath:    jsonFile,
				GitHubToken: config.GitHubToken,
			}).Get(ctx, &schemaResult)
			if err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("schema validation failed for %s: %v", jsonFile, err))
				result.SchemaValid = false
			} else if !schemaResult.Valid {
				result.SchemaValid = false
				result.Errors = append(result.Errors, schemaResult.Errors...)
			}
		}
	}

	// Step 3.5: Run agent-based documentation validation if enabled
	if config.UseAgentValidation && result.NeedsUpdate && len(categorized.PluginFiles) > 0 && result.SchemaValid {
		logger.Info("Running agent-based documentation validation")
		
		// Use longer timeout for AI agent validation (5 minutes instead of 2)
		agentCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
			StartToCloseTimeout: 5 * time.Minute,
			RetryPolicy: &temporal.RetryPolicy{
				MaximumAttempts: 2, // Fewer retries for expensive AI calls
			},
		})
		
		var agentResult DocumentationValidationResult
		err = workflow.ExecuteActivity(agentCtx, ValidateDocumentationActivity, DocumentationValidationInput{
			Owner:       config.Owner,
			Repo:        config.Repo,
			PRNumber:    config.PRNumber,
			HeadSHA:     config.HeadSHA,
			CodeFiles:   categorized.CodeFiles,
			PluginFiles: categorized.PluginFiles,
		}).Get(ctx, &agentResult)
		if err != nil {
			logger.Error("Agent validation failed", "error", err)
			result.Errors = append(result.Errors, fmt.Sprintf("agent validation failed: %v", err))
		} else {
			result.AgentValidation = &agentResult
			result.AgentValidationRan = true
			logger.Info("Agent validation complete",
				"valid", agentResult.Valid,
				"critical_issues", len(agentResult.CriticalIssues),
				"high_issues", len(agentResult.HighIssues))
		}
	}

	// Step 4: Post appropriate comment
	if result.NeedsUpdate && len(categorized.PluginFiles) == 0 {
		// Code changed but plugin didn't - post reminder
		logger.Info("Posting plugin update reminder")
		var commentResult PostCommentResult
		err = workflow.ExecuteActivity(ctx, PostReminderCommentActivity, PostCommentInput{
			Owner:       config.Owner,
			Repo:        config.Repo,
			PRNumber:    config.PRNumber,
			CodeFiles:   categorized.CodeFiles,
			PluginFiles: categorized.PluginFiles,
			GitHubToken: config.GitHubToken,
		}).Get(ctx, &commentResult)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("failed to post comment: %v", err))
		} else {
			result.CommentPosted = true
			result.CommentURL = commentResult.URL
		}
	} else if result.NeedsUpdate && len(categorized.PluginFiles) > 0 && result.SchemaValid {
		// Code changed AND plugin updated correctly - post success
		logger.Info("Posting success message")
		var commentResult PostCommentResult
		err = workflow.ExecuteActivity(ctx, PostSuccessCommentActivity, PostCommentInput{
			Owner:           config.Owner,
			Repo:            config.Repo,
			PRNumber:        config.PRNumber,
			CodeFiles:       categorized.CodeFiles,
			PluginFiles:     categorized.PluginFiles,
			GitHubToken:     config.GitHubToken,
			AgentValidation: result.AgentValidation,
		}).Get(ctx, &commentResult)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("failed to post comment: %v", err))
		} else {
			result.CommentPosted = true
			result.CommentURL = commentResult.URL
		}
	}

	logger.Info("Plugin validation complete",
		"needs_update", result.NeedsUpdate,
		"schema_valid", result.SchemaValid,
		"comment_posted", result.CommentPosted)

	return result, nil
}

// CategorizedFiles contains files categorized by type.
type CategorizedFiles struct {
	CodeFiles   []string // Files that affect plugin behavior
	PluginFiles []string // Files in .claude-plugin/
	OtherFiles  []string // Other files (tests, docs, etc.)
}

// Activity input/output types

type FetchPRFilesInput struct {
	Owner       string
	Repo        string
	PRNumber    int
	GitHubToken config.Secret
}

type CategorizeFilesInput struct {
	Files []FileChange
}

type ValidateSchemasInput struct {
	Owner       string
	Repo        string
	HeadSHA     string
	FilePath    string
	GitHubToken config.Secret
}

type SchemaValidationResult struct {
	Valid  bool
	Errors []string
}

type PostCommentInput struct {
	Owner           string
	Repo            string
	PRNumber        int
	CodeFiles       []string
	PluginFiles     []string
	GitHubToken     config.Secret
	AgentValidation *DocumentationValidationResult // Optional agent validation results
}

type PostCommentResult struct {
	URL string
}
