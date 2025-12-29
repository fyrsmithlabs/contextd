// Package workflows provides Temporal workflow definitions for contextd automation.
package workflows

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"

	"github.com/fyrsmithlabs/contextd/internal/config"
)

// VersionValidationConfig configures the version validation workflow.
type VersionValidationConfig struct {
	Owner       string        // GitHub repository owner
	Repo        string        // GitHub repository name
	PRNumber    int           // Pull request number
	HeadSHA     string        // PR commit SHA
	GitHubToken config.Secret // GitHub API token for activities
}

// VersionValidationResult contains validation results.
type VersionValidationResult struct {
	VersionMatches bool     // Whether VERSION file matches plugin.json
	VersionFile    string   // Version from VERSION file
	PluginVersion  string   // Version from plugin.json
	CommentPosted  bool     // Whether we posted a comment
	CommentURL     string   // URL of posted comment
	Errors         []string // Any errors encountered
}

// VersionValidationWorkflow validates that VERSION file matches plugin.json.
//
// This workflow:
// 1. Fetches VERSION file content from PR
// 2. Fetches plugin.json and extracts version
// 3. Compares versions
// 4. Posts comment if versions don't match
func VersionValidationWorkflow(ctx workflow.Context, config VersionValidationConfig) (*VersionValidationResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting version validation",
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

	// Step 4: Compare versions
	result.VersionMatches = (result.VersionFile == result.PluginVersion)

	logger.Info("Version comparison complete",
		"version_file", result.VersionFile,
		"plugin_version", result.PluginVersion,
		"matches", result.VersionMatches)

	// Step 5: Post comment if versions don't match
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
	}

	logger.Info("Version validation complete",
		"version_matches", result.VersionMatches,
		"comment_posted", result.CommentPosted)

	return result, nil
}

// Activity input types

type FetchFileContentInput struct {
	Owner       string
	Repo        string
	Path        string
	Ref         string        // Commit SHA or branch name
	GitHubToken config.Secret
}

type PostVersionCommentInput struct {
	Owner         string
	Repo          string
	PRNumber      int
	VersionFile   string
	PluginVersion string
	GitHubToken   config.Secret
}
