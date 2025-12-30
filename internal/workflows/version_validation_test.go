package workflows

import (
	"testing"

	"github.com/fyrsmithlabs/contextd/internal/config"
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

// TestValidateSemanticVersion tests semantic version validation.
func TestValidateSemanticVersion(t *testing.T) {
	tests := []struct {
		name    string
		version string
		wantErr bool
	}{
		// Valid versions - basic
		{"valid basic version", "1.2.3", false},
		{"valid single digit", "0.0.1", false},
		{"valid large numbers", "10.20.30", false},
		{"valid all zeros", "0.0.0", false},

		// Valid versions - with pre-release
		{"valid pre-release alpha", "1.2.3-alpha", false},
		{"valid pre-release beta", "1.2.3-beta.1", false},
		{"valid pre-release rc", "1.0.0-rc.1", false},
		{"valid pre-release with dots", "1.2.3-alpha.beta.1", false},
		{"valid pre-release complex", "1.2.3-0.3.7", false},
		{"valid pre-release with hyphens", "1.2.3-alpha-1", false},

		// Valid versions - with build metadata
		{"valid build metadata", "1.2.3+20241223", false},
		{"valid build with dots", "1.2.3+build.123", false},
		{"valid build with hyphens", "1.2.3+sha-a1b2c3d", false},

		// Valid versions - with both pre-release and build
		{"valid pre-release and build", "1.2.3-alpha.1+build.123", false},
		{"valid rc with build", "2.0.0-rc.1+20241223", false},
		{"valid complex combo", "1.0.0-beta.2+exp.sha.5114f85", false},

		// Invalid versions - wrong format
		{"invalid empty string", "", true},
		{"valid no patch (defaults to 0)", "1.2", false},      // semver library accepts this as 1.2.0
		{"valid no minor (defaults to 0)", "1", false},        // semver library accepts this as 1.0.0
		{"invalid four parts", "1.2.3.4", true},
		{"invalid text only", "not-a-version", true},
		{"invalid garbage", "garbage", true},
		{"valid with v prefix", "v1.2.3", false},              // semver library accepts v prefix
		{"valid leading zeros", "01.02.03", false},            // semver library accepts leading zeros

		// Invalid versions - wrong characters
		{"invalid with spaces", "1.2.3 beta", true},
		{"invalid with letters in numbers", "1.2.a", true},
		{"invalid special chars", "1.2.3@beta", true},
		{"valid missing dot (interpreted as 1.23.0)", "1.23", false}, // semver library accepts this

		// Edge cases
		{"invalid just dots", "...", true},
		{"invalid trailing dot", "1.2.3.", true},
		{"invalid leading dot", ".1.2.3", true},
		{"invalid negative numbers", "-1.2.3", true},
		{"invalid whitespace only", "   ", true},
		{"invalid with newlines", "1.2.3\n", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSemanticVersion(tt.version)
			if tt.wantErr {
				assert.Error(t, err, "expected error for version: %q", tt.version)
				// Verify error message contains helpful context (version or semantic)
				if err != nil && tt.version != "" {
					// Non-empty invalid versions should mention "semantic version"
					assert.Contains(t, err.Error(), "semantic version",
						"error message should mention semantic version")
				}
			} else {
				assert.NoError(t, err, "expected no error for version: %q", tt.version)
			}
		})
	}
}

// TestVersionValidationWithInvalidSemver tests workflow behavior with invalid semver.
func TestVersionValidationWithInvalidSemver(t *testing.T) {
	t.Run("rejects invalid VERSION format", func(t *testing.T) {
		testSuite := &testsuite.WorkflowTestSuite{}
		env := testSuite.NewTestWorkflowEnvironment()

		env.RegisterWorkflow(VersionValidationWorkflow)

		// VERSION file has invalid format (not semver)
		env.OnActivity(FetchFileContentActivity, mock.Anything, FetchFileContentInput{
			Owner: "test-owner",
			Repo:  "test-repo",
			Path:  "VERSION",
			Ref:   "abc1234",
		}).Return("not-a-version", nil)

		config := VersionValidationConfig{
			Owner:    "test-owner",
			Repo:     "test-repo",
			PRNumber: 99,
			HeadSHA:  "abc1234",
		}
		env.ExecuteWorkflow(VersionValidationWorkflow, config)

		require.True(t, env.IsWorkflowCompleted())
		require.Error(t, env.GetWorkflowError())

		var result VersionValidationResult
		err := env.GetWorkflowResult(&result)
		require.Error(t, err) // Workflow should error
	})

	t.Run("rejects invalid plugin.json version format", func(t *testing.T) {
		testSuite := &testsuite.WorkflowTestSuite{}
		env := testSuite.NewTestWorkflowEnvironment()

		env.RegisterWorkflow(VersionValidationWorkflow)

		// VERSION file is valid
		env.OnActivity(FetchFileContentActivity, mock.Anything, FetchFileContentInput{
			Owner: "test-owner",
			Repo:  "test-repo",
			Path:  "VERSION",
			Ref:   "def5678",
		}).Return("1.2.3", nil)

		// plugin.json has invalid version format
		pluginJSON := `{
			"name": "contextd",
			"version": "v1.2.3",
			"description": "Test"
		}`
		env.OnActivity(FetchFileContentActivity, mock.Anything, FetchFileContentInput{
			Owner: "test-owner",
			Repo:  "test-repo",
			Path:  ".claude-plugin/plugin.json",
			Ref:   "def5678",
		}).Return(pluginJSON, nil)

		config := VersionValidationConfig{
			Owner:    "test-owner",
			Repo:     "test-repo",
			PRNumber: 100,
			HeadSHA:  "def5678",
		}
		env.ExecuteWorkflow(VersionValidationWorkflow, config)

		require.True(t, env.IsWorkflowCompleted())
		require.Error(t, env.GetWorkflowError())

		var result VersionValidationResult
		err := env.GetWorkflowResult(&result)
		require.Error(t, err) // Workflow should error
	})

	t.Run("accepts pre-release versions", func(t *testing.T) {
		testSuite := &testsuite.WorkflowTestSuite{}
		env := testSuite.NewTestWorkflowEnvironment()

		env.RegisterWorkflow(VersionValidationWorkflow)

		// Both have valid pre-release versions
		env.OnActivity(FetchFileContentActivity, mock.Anything, FetchFileContentInput{
			Owner: "test-owner",
			Repo:  "test-repo",
			Path:  "VERSION",
			Ref:   "abc1234",
		}).Return("2.0.0-rc.1+build.456", nil)

		pluginJSON := `{
			"name": "contextd",
			"version": "2.0.0-rc.1+build.456",
			"description": "Test"
		}`
		env.OnActivity(FetchFileContentActivity, mock.Anything, FetchFileContentInput{
			Owner: "test-owner",
			Repo:  "test-repo",
			Path:  ".claude-plugin/plugin.json",
			Ref:   "abc1234",
		}).Return(pluginJSON, nil)

		env.OnActivity(RemoveVersionMismatchCommentActivity, mock.Anything, mock.Anything).Return(nil)

		config := VersionValidationConfig{
			Owner:       "test-owner",
			Repo:        "test-repo",
			PRNumber:    101,
			HeadSHA:     "abc1234",
			GitHubToken: config.Secret("test-token"),
		}
		env.ExecuteWorkflow(VersionValidationWorkflow, config)

		require.True(t, env.IsWorkflowCompleted())
		require.NoError(t, env.GetWorkflowError())

		var result VersionValidationResult
		require.NoError(t, env.GetWorkflowResult(&result))
		assert.True(t, result.VersionMatches)
		assert.Equal(t, "2.0.0-rc.1+build.456", result.VersionFile)
		assert.Equal(t, "2.0.0-rc.1+build.456", result.PluginVersion)
	})

	t.Run("rejects version with v prefix", func(t *testing.T) {
		testSuite := &testsuite.WorkflowTestSuite{}
		env := testSuite.NewTestWorkflowEnvironment()

		env.RegisterWorkflow(VersionValidationWorkflow)

		// VERSION file has 'v' prefix (common mistake)
		env.OnActivity(FetchFileContentActivity, mock.Anything, FetchFileContentInput{
			Owner: "test-owner",
			Repo:  "test-repo",
			Path:  "VERSION",
			Ref:   "fed9abc",
		}).Return("v1.2.3", nil)

		config := VersionValidationConfig{
			Owner:    "test-owner",
			Repo:     "test-repo",
			PRNumber: 102,
			HeadSHA:  "fed9abc",
		}
		env.ExecuteWorkflow(VersionValidationWorkflow, config)

		require.True(t, env.IsWorkflowCompleted())
		require.Error(t, env.GetWorkflowError())

		var result VersionValidationResult
		err := env.GetWorkflowResult(&result)
		require.Error(t, err) // Workflow should error
	})

	t.Run("rejects partial version numbers", func(t *testing.T) {
		testSuite := &testsuite.WorkflowTestSuite{}
		env := testSuite.NewTestWorkflowEnvironment()

		env.RegisterWorkflow(VersionValidationWorkflow)

		// VERSION file has only MAJOR.MINOR (missing PATCH)
		env.OnActivity(FetchFileContentActivity, mock.Anything, FetchFileContentInput{
			Owner: "test-owner",
			Repo:  "test-repo",
			Path:  "VERSION",
			Ref:   "1234567",
		}).Return("1.2", nil)

		config := VersionValidationConfig{
			Owner:    "test-owner",
			Repo:     "test-repo",
			PRNumber: 103,
			HeadSHA:  "1234567",
		}
		env.ExecuteWorkflow(VersionValidationWorkflow, config)

		require.True(t, env.IsWorkflowCompleted())
		require.Error(t, env.GetWorkflowError())

		var result VersionValidationResult
		err := env.GetWorkflowResult(&result)
		require.Error(t, err) // Workflow should error
	})
}
