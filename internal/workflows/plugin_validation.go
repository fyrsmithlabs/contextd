// Package workflows provides Temporal workflow definitions for contextd automation.
package workflows

import (
	"fmt"
	"strings"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"

)

// Type definitions moved to types.go

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
		// CRITICAL: Can't proceed without PR file list
		result.Errors = append(result.Errors, FormatErrorForResult("failed to fetch PR files", err))
		return result, WrapActivityError("failed to fetch PR files", err)
	}

	// Step 2: Categorize changes
	logger.Info("Categorizing file changes", "count", len(fileChanges))
	var categorized CategorizedFiles
	err = workflow.ExecuteActivity(ctx, CategorizeFilesActivity, CategorizeFilesInput{
		Files: fileChanges,
	}).Get(ctx, &categorized)
	if err != nil {
		// CRITICAL: Can't proceed without file categorization
		result.Errors = append(result.Errors, FormatErrorForResult("failed to categorize files", err))
		return result, WrapActivityError("failed to categorize files", err)
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
				// CRITICAL: Schema validation failed - invalid plugin files
				logger.Error("Schema validation activity failed", "file", jsonFile, "error", err)
				result.Errors = append(result.Errors, FormatErrorForResult(fmt.Sprintf("schema validation failed for %s", jsonFile), err))
				result.SchemaValid = false
			} else if !schemaResult.Valid {
				// CRITICAL: Schema is invalid JSON
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
			// HIGH: Agent validation failed, but core validation succeeded - record but continue
			logger.Error("Agent validation failed", "error", err)
			result.Errors = append(result.Errors, FormatErrorForResult("agent validation failed", err))
			// Don't return error - this is an optional enhancement
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
			// HIGH: Validation succeeded, but notification failed - record but continue
			logger.Error("Failed to post reminder comment", "error", err)
			result.Errors = append(result.Errors, FormatErrorForResult("failed to post reminder comment", err))
			// Don't return error - validation completed successfully
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
			// HIGH: Validation succeeded, but notification failed - record but continue
			logger.Error("Failed to post success comment", "error", err)
			result.Errors = append(result.Errors, FormatErrorForResult("failed to post success comment", err))
			// Don't return error - validation completed successfully
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

// Type definitions moved to types.go
