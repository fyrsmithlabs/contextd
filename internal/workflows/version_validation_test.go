package workflows

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
)

// TestVersionValidationWorkflow tests the main version validation workflow.
func TestVersionValidationWorkflow(t *testing.T) {
	t.Run("detects version mismatch", func(t *testing.T) {
		testSuite := &testsuite.WorkflowTestSuite{}
		env := testSuite.NewTestWorkflowEnvironment()

		// Register workflow
		env.RegisterWorkflow(VersionValidationWorkflow)

		// Mock activities
		// Step 1: Fetch VERSION file - returns "1.2.3"
		env.OnActivity(FetchFileContentActivity, mock.Anything, FetchFileContentInput{
			Owner: "test-owner",
			Repo:  "test-repo",
			Path:  "VERSION",
			Ref:   "abc123",
		}).Return("1.2.3\n", nil)

		// Step 2: Fetch plugin.json - returns JSON with version "0.0.0"
		pluginJSON := `{
			"name": "contextd",
			"version": "0.0.0",
			"description": "Test"
		}`
		env.OnActivity(FetchFileContentActivity, mock.Anything, FetchFileContentInput{
			Owner: "test-owner",
			Repo:  "test-repo",
			Path:  ".claude-plugin/plugin.json",
			Ref:   "abc123",
		}).Return(pluginJSON, nil)

		// Step 3: Post mismatch comment
		env.OnActivity(PostVersionMismatchCommentActivity, mock.Anything, mock.Anything).Return(&PostCommentResult{URL: "https://github.com/test/test/pull/1#issuecomment-1"}, nil)

		// Execute workflow
		config := VersionValidationConfig{
			Owner:    "test-owner",
			Repo:     "test-repo",
			PRNumber: 1,
			HeadSHA:  "abc123",
		}
		env.ExecuteWorkflow(VersionValidationWorkflow, config)

		require.True(t, env.IsWorkflowCompleted())
		require.NoError(t, env.GetWorkflowError())

		var result VersionValidationResult
		require.NoError(t, env.GetWorkflowResult(&result))
		assert.False(t, result.VersionMatches)
		assert.Equal(t, "1.2.3", result.VersionFile)
		assert.Equal(t, "0.0.0", result.PluginVersion)
		assert.True(t, result.CommentPosted)
		assert.Equal(t, "https://github.com/test/test/pull/1#issuecomment-1", result.CommentURL)
	})

	t.Run("validates matching versions", func(t *testing.T) {
		testSuite := &testsuite.WorkflowTestSuite{}
		env := testSuite.NewTestWorkflowEnvironment()

		env.RegisterWorkflow(VersionValidationWorkflow)

		// Mock activities - both return "1.2.3"
		env.OnActivity(FetchFileContentActivity, mock.Anything, FetchFileContentInput{
			Owner: "test-owner",
			Repo:  "test-repo",
			Path:  "VERSION",
			Ref:   "def456",
		}).Return("1.2.3", nil)

		pluginJSON := `{
			"name": "contextd",
			"version": "1.2.3",
			"description": "Test"
		}`
		env.OnActivity(FetchFileContentActivity, mock.Anything, FetchFileContentInput{
			Owner: "test-owner",
			Repo:  "test-repo",
			Path:  ".claude-plugin/plugin.json",
			Ref:   "def456",
		}).Return(pluginJSON, nil)

		// No comment should be posted since versions match

		config := VersionValidationConfig{
			Owner:    "test-owner",
			Repo:     "test-repo",
			PRNumber: 2,
			HeadSHA:  "def456",
		}
		env.ExecuteWorkflow(VersionValidationWorkflow, config)

		require.True(t, env.IsWorkflowCompleted())
		require.NoError(t, env.GetWorkflowError())

		var result VersionValidationResult
		require.NoError(t, env.GetWorkflowResult(&result))
		assert.True(t, result.VersionMatches)
		assert.Equal(t, "1.2.3", result.VersionFile)
		assert.Equal(t, "1.2.3", result.PluginVersion)
		assert.False(t, result.CommentPosted)
	})

	t.Run("handles whitespace in VERSION file", func(t *testing.T) {
		testSuite := &testsuite.WorkflowTestSuite{}
		env := testSuite.NewTestWorkflowEnvironment()

		env.RegisterWorkflow(VersionValidationWorkflow)

		// VERSION file has whitespace
		env.OnActivity(FetchFileContentActivity, mock.Anything, FetchFileContentInput{
			Owner: "test-owner",
			Repo:  "test-repo",
			Path:  "VERSION",
			Ref:   "ghi789",
		}).Return("  1.2.3\n\n", nil)

		pluginJSON := `{
			"name": "contextd",
			"version": "1.2.3",
			"description": "Test"
		}`
		env.OnActivity(FetchFileContentActivity, mock.Anything, FetchFileContentInput{
			Owner: "test-owner",
			Repo:  "test-repo",
			Path:  ".claude-plugin/plugin.json",
			Ref:   "ghi789",
		}).Return(pluginJSON, nil)

		config := VersionValidationConfig{
			Owner:    "test-owner",
			Repo:     "test-repo",
			PRNumber: 3,
			HeadSHA:  "ghi789",
		}
		env.ExecuteWorkflow(VersionValidationWorkflow, config)

		require.True(t, env.IsWorkflowCompleted())
		require.NoError(t, env.GetWorkflowError())

		var result VersionValidationResult
		require.NoError(t, env.GetWorkflowResult(&result))
		assert.True(t, result.VersionMatches) // Whitespace should be trimmed
		assert.Equal(t, "1.2.3", result.VersionFile)
	})

	t.Run("handles pre-release versions", func(t *testing.T) {
		testSuite := &testsuite.WorkflowTestSuite{}
		env := testSuite.NewTestWorkflowEnvironment()

		env.RegisterWorkflow(VersionValidationWorkflow)

		// Both use pre-release version
		env.OnActivity(FetchFileContentActivity, mock.Anything, FetchFileContentInput{
			Owner: "test-owner",
			Repo:  "test-repo",
			Path:  "VERSION",
			Ref:   "jkl012",
		}).Return("1.0.0-rc.1", nil)

		pluginJSON := `{
			"name": "contextd",
			"version": "1.0.0-rc.1",
			"description": "Test"
		}`
		env.OnActivity(FetchFileContentActivity, mock.Anything, FetchFileContentInput{
			Owner: "test-owner",
			Repo:  "test-repo",
			Path:  ".claude-plugin/plugin.json",
			Ref:   "jkl012",
		}).Return(pluginJSON, nil)

		config := VersionValidationConfig{
			Owner:    "test-owner",
			Repo:     "test-repo",
			PRNumber: 4,
			HeadSHA:  "jkl012",
		}
		env.ExecuteWorkflow(VersionValidationWorkflow, config)

		require.True(t, env.IsWorkflowCompleted())
		require.NoError(t, env.GetWorkflowError())

		var result VersionValidationResult
		require.NoError(t, env.GetWorkflowResult(&result))
		assert.True(t, result.VersionMatches)
		assert.Equal(t, "1.0.0-rc.1", result.VersionFile)
		assert.Equal(t, "1.0.0-rc.1", result.PluginVersion)
	})
}

// TestBuildVersionMismatchComment tests comment generation.
func TestBuildVersionMismatchComment(t *testing.T) {
	comment := buildVersionMismatchComment("1.2.3", "0.0.0")

	// Check for required elements
	assert.Contains(t, comment, "⚠️ Version Mismatch Detected")
	assert.Contains(t, comment, "1.2.3")
	assert.Contains(t, comment, "0.0.0")
	assert.Contains(t, comment, "./scripts/sync-version.sh")
	assert.Contains(t, comment, "VERSIONING.md")
	assert.Contains(t, comment, "Temporal workflows")
}
