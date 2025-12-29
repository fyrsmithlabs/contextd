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

		// Mock removal activity (versions match, should remove old comment if exists)
		env.OnActivity(RemoveVersionMismatchCommentActivity, mock.Anything, mock.Anything).Return(nil)

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

		// Mock removal activity (versions match)
		env.OnActivity(RemoveVersionMismatchCommentActivity, mock.Anything, mock.Anything).Return(nil)

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

	t.Run("handles VERSION file fetch error", func(t *testing.T) {
		testSuite := &testsuite.WorkflowTestSuite{}
		env := testSuite.NewTestWorkflowEnvironment()

		env.RegisterWorkflow(VersionValidationWorkflow)

		// VERSION file fetch fails (404, network error, etc.)
		env.OnActivity(FetchFileContentActivity, mock.Anything, FetchFileContentInput{
			Owner: "test-owner",
			Repo:  "test-repo",
			Path:  "VERSION",
			Ref:   "missing",
		}).Return("", assert.AnError)

		config := VersionValidationConfig{
			Owner:    "test-owner",
			Repo:     "test-repo",
			PRNumber: 5,
			HeadSHA:  "missing",
		}
		env.ExecuteWorkflow(VersionValidationWorkflow, config)

		require.True(t, env.IsWorkflowCompleted())
		require.Error(t, env.GetWorkflowError())
		// Just verify workflow errored (error message is wrapped by Temporal)
	})

	t.Run("handles plugin.json fetch error", func(t *testing.T) {
		testSuite := &testsuite.WorkflowTestSuite{}
		env := testSuite.NewTestWorkflowEnvironment()

		env.RegisterWorkflow(VersionValidationWorkflow)

		// VERSION file succeeds
		env.OnActivity(FetchFileContentActivity, mock.Anything, FetchFileContentInput{
			Owner: "test-owner",
			Repo:  "test-repo",
			Path:  "VERSION",
			Ref:   "missing-plugin",
		}).Return("1.2.3", nil)

		// plugin.json fetch fails
		env.OnActivity(FetchFileContentActivity, mock.Anything, FetchFileContentInput{
			Owner: "test-owner",
			Repo:  "test-repo",
			Path:  ".claude-plugin/plugin.json",
			Ref:   "missing-plugin",
		}).Return("", assert.AnError)

		config := VersionValidationConfig{
			Owner:    "test-owner",
			Repo:     "test-repo",
			PRNumber: 6,
			HeadSHA:  "missing-plugin",
		}
		env.ExecuteWorkflow(VersionValidationWorkflow, config)

		require.True(t, env.IsWorkflowCompleted())
		require.Error(t, env.GetWorkflowError())
		// Just verify workflow errored (error message is wrapped by Temporal)
	})

	t.Run("handles invalid JSON in plugin.json", func(t *testing.T) {
		testSuite := &testsuite.WorkflowTestSuite{}
		env := testSuite.NewTestWorkflowEnvironment()

		env.RegisterWorkflow(VersionValidationWorkflow)

		env.OnActivity(FetchFileContentActivity, mock.Anything, FetchFileContentInput{
			Owner: "test-owner",
			Repo:  "test-repo",
			Path:  "VERSION",
			Ref:   "invalid-json",
		}).Return("1.2.3", nil)

		// Invalid JSON
		env.OnActivity(FetchFileContentActivity, mock.Anything, FetchFileContentInput{
			Owner: "test-owner",
			Repo:  "test-repo",
			Path:  ".claude-plugin/plugin.json",
			Ref:   "invalid-json",
		}).Return("{invalid json", nil)

		config := VersionValidationConfig{
			Owner:    "test-owner",
			Repo:     "test-repo",
			PRNumber: 7,
			HeadSHA:  "invalid-json",
		}
		env.ExecuteWorkflow(VersionValidationWorkflow, config)

		require.True(t, env.IsWorkflowCompleted())
		require.Error(t, env.GetWorkflowError())
		// Just verify workflow errored (error message is wrapped by Temporal)
	})

	t.Run("handles empty VERSION file", func(t *testing.T) {
		testSuite := &testsuite.WorkflowTestSuite{}
		env := testSuite.NewTestWorkflowEnvironment()

		env.RegisterWorkflow(VersionValidationWorkflow)

		// Empty VERSION file (just whitespace)
		env.OnActivity(FetchFileContentActivity, mock.Anything, FetchFileContentInput{
			Owner: "test-owner",
			Repo:  "test-repo",
			Path:  "VERSION",
			Ref:   "empty-version",
		}).Return("  \n\n  ", nil)

		config := VersionValidationConfig{
			Owner:    "test-owner",
			Repo:     "test-repo",
			PRNumber: 8,
			HeadSHA:  "empty-version",
		}
		env.ExecuteWorkflow(VersionValidationWorkflow, config)

		require.True(t, env.IsWorkflowCompleted())
		require.Error(t, env.GetWorkflowError())
		// Just verify workflow errored (error message is wrapped by Temporal)
	})

	t.Run("handles empty plugin.json version", func(t *testing.T) {
		testSuite := &testsuite.WorkflowTestSuite{}
		env := testSuite.NewTestWorkflowEnvironment()

		env.RegisterWorkflow(VersionValidationWorkflow)

		env.OnActivity(FetchFileContentActivity, mock.Anything, FetchFileContentInput{
			Owner: "test-owner",
			Repo:  "test-repo",
			Path:  "VERSION",
			Ref:   "empty-plugin-version",
		}).Return("1.2.3", nil)

		// plugin.json with empty version field
		pluginJSON := `{
			"name": "contextd",
			"version": "",
			"description": "Test"
		}`
		env.OnActivity(FetchFileContentActivity, mock.Anything, FetchFileContentInput{
			Owner: "test-owner",
			Repo:  "test-repo",
			Path:  ".claude-plugin/plugin.json",
			Ref:   "empty-plugin-version",
		}).Return(pluginJSON, nil)

		config := VersionValidationConfig{
			Owner:    "test-owner",
			Repo:     "test-repo",
			PRNumber: 9,
			HeadSHA:  "empty-plugin-version",
		}
		env.ExecuteWorkflow(VersionValidationWorkflow, config)

		require.True(t, env.IsWorkflowCompleted())
		require.Error(t, env.GetWorkflowError())
		// Just verify workflow errored (error message is wrapped by Temporal)
	})

	t.Run("removes comment when versions match", func(t *testing.T) {
		testSuite := &testsuite.WorkflowTestSuite{}
		env := testSuite.NewTestWorkflowEnvironment()

		env.RegisterWorkflow(VersionValidationWorkflow)

		// Both versions match
		env.OnActivity(FetchFileContentActivity, mock.Anything, FetchFileContentInput{
			Owner: "test-owner",
			Repo:  "test-repo",
			Path:  "VERSION",
			Ref:   "fixed",
		}).Return("2.0.0", nil)

		pluginJSON := `{
			"name": "contextd",
			"version": "2.0.0",
			"description": "Test"
		}`
		env.OnActivity(FetchFileContentActivity, mock.Anything, FetchFileContentInput{
			Owner: "test-owner",
			Repo:  "test-repo",
			Path:  ".claude-plugin/plugin.json",
			Ref:   "fixed",
		}).Return(pluginJSON, nil)

		// Expect RemoveVersionMismatchCommentActivity to be called
		env.OnActivity(RemoveVersionMismatchCommentActivity, mock.Anything, mock.Anything).Return(nil)

		config := VersionValidationConfig{
			Owner:    "test-owner",
			Repo:     "test-repo",
			PRNumber: 10,
			HeadSHA:  "fixed",
		}
		env.ExecuteWorkflow(VersionValidationWorkflow, config)

		require.True(t, env.IsWorkflowCompleted())
		require.NoError(t, env.GetWorkflowError())

		var result VersionValidationResult
		require.NoError(t, env.GetWorkflowResult(&result))
		assert.True(t, result.VersionMatches)
		assert.False(t, result.CommentPosted)
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
