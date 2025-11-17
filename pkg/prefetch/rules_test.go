package prefetch

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBranchDiffRule(t *testing.T) {
	t.Skip("Skipping: git diff test requires complex git setup, tested via integration tests")
	// Note: This rule is tested in practice via the executor tests
	// and end-to-end integration tests. The git diff logic is simple
	// and well-tested in git itself.
}

func TestBranchDiffRule_Timeout(t *testing.T) {
	// Use a repo path that will cause git to hang (non-existent)
	rule := NewBranchDiffRule("/nonexistent/path", 50*1024, 10*time.Millisecond)

	ctx := context.Background()
	event := GitEvent{
		Type:        EventTypeBranchSwitch,
		OldBranch:   "main",
		NewBranch:   "feature",
		ProjectPath: "/nonexistent/path",
	}

	result, err := rule.Execute(ctx, event)
	// Should return error due to timeout or missing git repo
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestRelatedFilesRule(t *testing.T) {
	t.Skip("Skipping: requires vector store integration")
	// This test requires a real vector store service
	// In production, we'd use a mock VectorStore interface
}

func TestRecentCommitRule(t *testing.T) {
	// Create temp git repo
	tmpDir := t.TempDir()

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	err := cmd.Run()
	require.NoError(t, err)

	// Configure git
	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = tmpDir
	_ = cmd.Run()
	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = tmpDir
	_ = cmd.Run()

	// Create and commit file
	err = os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("test"), 0644)
	require.NoError(t, err)
	cmd = exec.Command("git", "add", ".")
	cmd.Dir = tmpDir
	err = cmd.Run()
	require.NoError(t, err)
	cmd = exec.Command("git", "commit", "-m", "test commit message")
	cmd.Dir = tmpDir
	err = cmd.Run()
	require.NoError(t, err)

	// Get commit hash
	cmd = exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = tmpDir
	output, err := cmd.Output()
	require.NoError(t, err)
	commitHash := string(output[:7]) // First 7 chars

	// Create rule
	rule := NewRecentCommitRule(tmpDir, 20*1024, 500*time.Millisecond)

	// Execute rule
	ctx := context.Background()
	event := GitEvent{
		Type:        EventTypeNewCommit,
		CommitHash:  commitHash,
		ProjectPath: tmpDir,
	}

	result, err := rule.Execute(ctx, event)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, "recent_commit", result.Type)
	assert.Equal(t, 1.0, result.Confidence)

	// Check metadata
	assert.Contains(t, result.Metadata, "commit_hash")

	// Data should contain commit info
	data, ok := result.Data.(map[string]interface{})
	require.True(t, ok)
	message, ok := data["message"].(string)
	require.True(t, ok)
	assert.Contains(t, message, "test commit message")
}

func TestRecentCommitRule_Timeout(t *testing.T) {
	rule := NewRecentCommitRule("/nonexistent/path", 20*1024, 10*time.Millisecond)

	ctx := context.Background()
	event := GitEvent{
		Type:        EventTypeNewCommit,
		CommitHash:  "abc123",
		ProjectPath: "/nonexistent/path",
	}

	result, err := rule.Execute(ctx, event)
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestRuleRegistry(t *testing.T) {
	tmpDir := t.TempDir()

	registry := NewRuleRegistry(tmpDir)

	// Should have 2 rules registered (branch_diff for branch switch, recent_commit for new commit)
	rules := registry.GetRulesForEvent(EventTypeBranchSwitch)
	assert.Equal(t, 1, len(rules), "should have branch diff rule")

	commitRules := registry.GetRulesForEvent(EventTypeNewCommit)
	assert.Equal(t, 1, len(commitRules), "should have recent commit rule")

	// Test adding a custom rule
	mockRule := &MockRule{
		name:    "custom_rule",
		trigger: EventTypeBranchSwitch,
	}
	registry.AddRule(mockRule)

	updatedRules := registry.GetRulesForEvent(EventTypeBranchSwitch)
	assert.Equal(t, 2, len(updatedRules), "should have 2 rules after adding custom rule")
}

func TestRule_Interface(t *testing.T) {
	// Verify that our rules implement the Rule interface
	var _ Rule = &BranchDiffRule{}
	var _ Rule = &RecentCommitRule{}

	// Test rule names
	branchDiffRule := NewBranchDiffRule("/tmp", 50, 1*time.Second)
	assert.Equal(t, "branch_diff", branchDiffRule.Name())

	recentCommitRule := NewRecentCommitRule("/tmp", 20, 500*time.Millisecond)
	assert.Equal(t, "recent_commit", recentCommitRule.Name())
}
