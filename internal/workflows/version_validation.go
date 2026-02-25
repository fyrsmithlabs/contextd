// Package workflows provides Temporal workflow definitions for contextd automation.
package workflows

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// Type definitions moved to types.go

// VersionValidationWorkflow validates that VERSION file matches plugin.json.
//
// This workflow:
// 0. Validates input configuration
// 1. Fetches VERSION file content from PR
// 2. Fetches plugin.json and extracts version
// 3. Validates both versions are valid semantic versions
// 4. Compares versions
// 5. Posts comment if versions don't match
func VersionValidationWorkflow(ctx workflow.Context, config VersionValidationConfig) (*VersionValidationResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting version validation",
		"owner", config.Owner,
		"repo", config.Repo,
		"pr", config.PRNumber)

	// Step 0: Validate input configuration
	if err := config.ValidateComprehensive(); err != nil {
		logger.Error("Input validation failed", "error", err)
		return &VersionValidationResult{
			Errors: []string{fmt.Sprintf("input validation failed: %v", err)},
		}, fmt.Errorf("validate config: %w", err)
	}

	// Configure activity options
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 2 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	result := &VersionValidationResult{}

	// Step 1: Fetch VERSION file
	logger.Info("Fetching VERSION file")
	var versionFileContent string
	err := workflow.ExecuteActivity(ctx, FetchFileContentActivity, FetchFileContentInput{
		Owner:       config.Owner,
		Repo:        config.Repo,
		Path:        "VERSION",
		Ref:         config.HeadSHA,
		GitHubToken: config.GitHubToken,
	}).Get(ctx, &versionFileContent)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("failed to fetch VERSION file: %v", err))
		return result, err
	}

	// Clean version string (trim whitespace)
	result.VersionFile = strings.TrimSpace(versionFileContent)

	// Validate VERSION file is not empty
	if result.VersionFile == "" {
		result.Errors = append(result.Errors, "VERSION file is empty")
		return result, fmt.Errorf("VERSION file is empty")
	}

	// Validate VERSION file is valid semantic version
	if err := validateSemanticVersion(result.VersionFile); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("invalid VERSION format: %v", err))
		return result, fmt.Errorf("invalid VERSION format: %w", err)
	}

	// Step 2: Fetch plugin.json
	logger.Info("Fetching plugin.json")
	var pluginContent string
	err = workflow.ExecuteActivity(ctx, FetchFileContentActivity, FetchFileContentInput{
		Owner:       config.Owner,
		Repo:        config.Repo,
		Path:        ".claude-plugin/plugin.json",
		Ref:         config.HeadSHA,
		GitHubToken: config.GitHubToken,
	}).Get(ctx, &pluginContent)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("failed to fetch plugin.json: %v", err))
		return result, err
	}

	// Step 3: Parse plugin.json and extract version
	var pluginData struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal([]byte(pluginContent), &pluginData); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("failed to parse plugin.json: %v", err))
		return result, err
	}
	result.PluginVersion = pluginData.Version

	// Validate plugin.json version is not empty
	if result.PluginVersion == "" {
		result.Errors = append(result.Errors, "plugin.json has empty or missing version field")
		return result, fmt.Errorf("plugin.json version is empty")
	}

	// Validate plugin.json version is valid semantic version
	if err := validateSemanticVersion(result.PluginVersion); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("invalid plugin.json version format: %v", err))
		return result, fmt.Errorf("invalid plugin.json version format: %w", err)
	}

	// Step 4: Compare versions
	result.VersionMatches = (result.VersionFile == result.PluginVersion)

	logger.Info("Version comparison complete",
		"version_file", result.VersionFile,
		"plugin_version", result.PluginVersion,
		"matches", result.VersionMatches)

	// Step 5: Post comment if versions don't match, or remove comment if they do
	if !result.VersionMatches {
		logger.Info("Posting version mismatch comment")
		var commentResult PostCommentResult
		err = workflow.ExecuteActivity(ctx, PostVersionMismatchCommentActivity, PostVersionCommentInput{
			Owner:         config.Owner,
			Repo:          config.Repo,
			PRNumber:      config.PRNumber,
			VersionFile:   result.VersionFile,
			PluginVersion: result.PluginVersion,
			GitHubToken:   config.GitHubToken,
		}).Get(ctx, &commentResult)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("failed to post comment: %v", err))
		} else {
			result.CommentPosted = true
			result.CommentURL = commentResult.URL
		}
	} else {
		// Versions match - remove any existing mismatch comment
		logger.Info("Versions match, removing any existing mismatch comment")
		err = workflow.ExecuteActivity(ctx, RemoveVersionMismatchCommentActivity, PostVersionCommentInput{
			Owner:       config.Owner,
			Repo:        config.Repo,
			PRNumber:    config.PRNumber,
			GitHubToken: config.GitHubToken,
		}).Get(ctx, nil)
		if err != nil {
			// Don't fail the workflow if we can't remove the comment
			logger.Warn("Failed to remove comment (non-fatal)", "error", err)
			result.Errors = append(result.Errors, fmt.Sprintf("failed to remove comment: %v", err))
		}
	}

	logger.Info("Version validation complete",
		"version_matches", result.VersionMatches,
		"comment_posted", result.CommentPosted)

	return result, nil
}

// validateSemanticVersion checks if a version string is valid semantic versioning.
//
// Valid formats:
//   - MAJOR.MINOR.PATCH (e.g., 1.2.3)
//   - MAJOR.MINOR.PATCH-prerelease (e.g., 1.2.3-alpha, 1.2.3-rc.1)
//   - MAJOR.MINOR.PATCH+build (e.g., 1.2.3+20241223)
//   - MAJOR.MINOR.PATCH-prerelease+build (e.g., 1.2.3-alpha.1+build.123)
//
// Returns an error with a clear message if the version is invalid.
func validateSemanticVersion(version string) error {
	if version == "" {
		return fmt.Errorf("version string is empty")
	}

	// Use semver library to parse and validate
	_, err := semver.NewVersion(version)
	if err != nil {
		return fmt.Errorf("not a valid semantic version (expected format: MAJOR.MINOR.PATCH[-prerelease][+build], e.g., 1.2.3, 1.2.3-alpha, 1.2.3+build): %w", err)
	}

	return nil
}
