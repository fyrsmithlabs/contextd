package workflows

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
)

// TestPluginUpdateValidationWorkflow tests the main plugin validation workflow.
func TestPluginUpdateValidationWorkflow(t *testing.T) {
	t.Run("detects code changes requiring plugin update", func(t *testing.T) {
		testSuite := &testsuite.WorkflowTestSuite{}
		env := testSuite.NewTestWorkflowEnvironment()

		// Register workflow
		env.RegisterWorkflow(PluginUpdateValidationWorkflow)

		// Mock activities
		// Step 1: Fetch PR files
		fileChanges := []FileChange{
			{Path: "internal/mcp/tools.go", Status: "modified"},
			{Path: "internal/reasoningbank/service.go", Status: "modified"},
		}
		env.OnActivity(FetchPRFilesActivity, mock.Anything, mock.Anything).Return(fileChanges, nil)

		// Step 2: Categorize files
		categorization := &CategorizedFiles{
			CodeFiles:   []string{"internal/mcp/tools.go", "internal/reasoningbank/service.go"},
			PluginFiles: []string{},
		}
		env.OnActivity(CategorizeFilesActivity, mock.Anything, mock.Anything).Return(categorization, nil)

		// Step 3: Post reminder comment
		env.OnActivity(PostReminderCommentActivity, mock.Anything, mock.Anything).Return(&PostCommentResult{URL: "https://github.com/test/test/pull/1#issuecomment-1"}, nil)

		// Execute workflow
		config := PluginUpdateValidationConfig{
			Owner:      "test-owner",
			Repo:       "test-repo",
			PRNumber:   1,
			BaseBranch: "main",
			HeadBranch: "feature/new-tool",
			HeadSHA:    "abc123",
		}
		env.ExecuteWorkflow(PluginUpdateValidationWorkflow, config)

		require.True(t, env.IsWorkflowCompleted())
		require.NoError(t, env.GetWorkflowError())

		var result PluginUpdateValidationResult
		require.NoError(t, env.GetWorkflowResult(&result))
		assert.True(t, result.NeedsUpdate)
		assert.True(t, result.CommentPosted)
		assert.Equal(t, "https://github.com/test/test/pull/1#issuecomment-1", result.CommentURL)
		assert.Len(t, result.CodeFilesChanged, 2)
	})

	t.Run("validates plugin schemas when modified", func(t *testing.T) {
		testSuite := &testsuite.WorkflowTestSuite{}
		env := testSuite.NewTestWorkflowEnvironment()

		env.RegisterWorkflow(PluginUpdateValidationWorkflow)

		// Mock activities
		fileChanges := []FileChange{
			{Path: "internal/mcp/tools.go", Status: "modified"},
			{Path: ".claude-plugin/schemas/contextd-mcp-tools.schema.json", Status: "modified"},
		}
		env.OnActivity(FetchPRFilesActivity, mock.Anything, mock.Anything).Return(fileChanges, nil)

		categorization := &CategorizedFiles{
			CodeFiles:   []string{"internal/mcp/tools.go"},
			PluginFiles: []string{".claude-plugin/schemas/contextd-mcp-tools.schema.json"},
		}
		env.OnActivity(CategorizeFilesActivity, mock.Anything, mock.Anything).Return(categorization, nil)

		// Validate schemas
		env.OnActivity(ValidatePluginSchemasActivity, mock.Anything, mock.Anything).Return(&SchemaValidationResult{Valid: true}, nil)

		// Post success comment
		env.OnActivity(PostSuccessCommentActivity, mock.Anything, mock.Anything).Return(&PostCommentResult{URL: "https://github.com/test/test/pull/1#issuecomment-2"}, nil)

		config := PluginUpdateValidationConfig{
			Owner:      "test-owner",
			Repo:       "test-repo",
			PRNumber:   1,
			BaseBranch: "main",
			HeadBranch: "feature/update-schema",
			HeadSHA:    "def456",
		}
		env.ExecuteWorkflow(PluginUpdateValidationWorkflow, config)

		require.True(t, env.IsWorkflowCompleted())
		require.NoError(t, env.GetWorkflowError())

		var result PluginUpdateValidationResult
		require.NoError(t, env.GetWorkflowResult(&result))
		assert.True(t, result.NeedsUpdate)
		assert.True(t, result.SchemaValid)
		assert.True(t, result.CommentPosted)
		assert.Len(t, result.PluginFilesChanged, 1)
	})

	t.Run("no action when only non-plugin code changes", func(t *testing.T) {
		testSuite := &testsuite.WorkflowTestSuite{}
		env := testSuite.NewTestWorkflowEnvironment()

		env.RegisterWorkflow(PluginUpdateValidationWorkflow)

		// Mock activities
		fileChanges := []FileChange{
			{Path: "README.md", Status: "modified"},
			{Path: "docs/architecture.md", Status: "modified"},
		}
		env.OnActivity(FetchPRFilesActivity, mock.Anything, mock.Anything).Return(fileChanges, nil)

		categorization := &CategorizedFiles{
			CodeFiles:   []string{},
			PluginFiles: []string{},
		}
		env.OnActivity(CategorizeFilesActivity, mock.Anything, mock.Anything).Return(categorization, nil)

		config := PluginUpdateValidationConfig{
			Owner:      "test-owner",
			Repo:       "test-repo",
			PRNumber:   1,
			BaseBranch: "main",
			HeadBranch: "docs/update",
			HeadSHA:    "ghi789",
		}
		env.ExecuteWorkflow(PluginUpdateValidationWorkflow, config)

		require.True(t, env.IsWorkflowCompleted())
		require.NoError(t, env.GetWorkflowError())

		var result PluginUpdateValidationResult
		require.NoError(t, env.GetWorkflowResult(&result))
		assert.False(t, result.NeedsUpdate)
		assert.False(t, result.CommentPosted)
	})
}

// TestCategorizeFilesActivity tests file categorization logic.
func TestCategorizeFilesActivity(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestActivityEnvironment()
	env.RegisterActivity(CategorizeFilesActivity)

	tests := []struct {
		name     string
		input    CategorizeFilesInput
		expected CategorizedFiles
	}{
		{
			name: "categorizes MCP tool changes",
			input: CategorizeFilesInput{
				Files: []FileChange{
					{Path: "internal/mcp/tools.go"},
					{Path: "internal/mcp/handlers/checkpoint.go"},
					{Path: "README.md"},
				},
			},
			expected: CategorizedFiles{
				CodeFiles:   []string{"internal/mcp/tools.go", "internal/mcp/handlers/checkpoint.go"},
				PluginFiles: []string{},
			},
		},
		{
			name: "categorizes plugin file changes",
			input: CategorizeFilesInput{
				Files: []FileChange{
					{Path: ".claude-plugin/skills/using-contextd/SKILL.md"},
					{Path: ".claude-plugin/schemas/contextd-mcp-tools.schema.json"},
				},
			},
			expected: CategorizedFiles{
				CodeFiles:   []string{},
				PluginFiles: []string{".claude-plugin/skills/using-contextd/SKILL.md", ".claude-plugin/schemas/contextd-mcp-tools.schema.json"},
			},
		},
		{
			name: "categorizes service changes",
			input: CategorizeFilesInput{
				Files: []FileChange{
					{Path: "internal/reasoningbank/service.go"},
					{Path: "internal/checkpoint/service.go"},
				},
			},
			expected: CategorizedFiles{
				CodeFiles:   []string{"internal/reasoningbank/service.go", "internal/checkpoint/service.go"},
				PluginFiles: []string{},
			},
		},
		{
			name: "categorizes config changes",
			input: CategorizeFilesInput{
				Files: []FileChange{
					{Path: "internal/config/types.go"},
					{Path: "internal/config/config.go"},
				},
			},
			expected: CategorizedFiles{
				CodeFiles:   []string{"internal/config/types.go", "internal/config/config.go"},
				PluginFiles: []string{},
			},
		},
		{
			name: "ignores non-relevant changes",
			input: CategorizeFilesInput{
				Files: []FileChange{
					{Path: "README.md"},
					{Path: "docs/architecture.md"},
					{Path: "test/integration/framework/workflow_test.go"},
				},
			},
			expected: CategorizedFiles{
				CodeFiles:   []string{},
				PluginFiles: []string{},
			},
		},
		{
			name: "handles mixed changes",
			input: CategorizeFilesInput{
				Files: []FileChange{
					{Path: "internal/mcp/tools.go"},
					{Path: ".claude-plugin/schemas/contextd-mcp-tools.schema.json"},
					{Path: "README.md"},
				},
			},
			expected: CategorizedFiles{
				CodeFiles:   []string{"internal/mcp/tools.go"},
				PluginFiles: []string{".claude-plugin/schemas/contextd-mcp-tools.schema.json"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, err := env.ExecuteActivity(CategorizeFilesActivity, tt.input)
			require.NoError(t, err)

			var result CategorizedFiles
			require.NoError(t, val.Get(&result))
			assert.ElementsMatch(t, tt.expected.CodeFiles, result.CodeFiles)
			assert.ElementsMatch(t, tt.expected.PluginFiles, result.PluginFiles)
		})
	}
}

// TestValidatePluginSchemasActivity tests schema validation logic.
func TestValidatePluginSchemasActivity(t *testing.T) {
	t.Run("validates valid JSON schema", func(t *testing.T) {
		// This test would need actual file reading, so we'll skip for now
		// In a real implementation, you'd mock the file system
		t.Skip("requires file system mocking")
	})

	t.Run("detects invalid JSON", func(t *testing.T) {
		// This test would need actual file reading, so we'll skip for now
		t.Skip("requires file system mocking")
	})
}
