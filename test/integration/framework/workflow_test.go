// Package framework provides the integration test framework for contextd.
package framework

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
)

// TestTestOrchestratorWorkflow validates the main test orchestrator workflow.
func TestTestOrchestratorWorkflow(t *testing.T) {
	t.Run("runs all enabled suites in parallel", func(t *testing.T) {
		testSuite := &testsuite.WorkflowTestSuite{}
		env := testSuite.NewTestWorkflowEnvironment()

		// Register workflows - including DeveloperSessionWorkflow which is called by suite workflows
		env.RegisterWorkflow(TestOrchestratorWorkflow)
		env.RegisterWorkflow(PolicyComplianceWorkflow)
		env.RegisterWorkflow(BugfixLearningWorkflow)
		env.RegisterWorkflow(MultiSessionWorkflow)
		env.RegisterWorkflow(DeveloperSessionWorkflow)

		// Mock activities for DeveloperSessionWorkflow
		env.OnActivity(StartContextdActivity, mock.Anything, mock.Anything).Return(ContextdHandle{ID: "ctx-1"}, nil)
		env.OnActivity(StopContextdActivity, mock.Anything, mock.Anything).Return(nil)
		env.OnActivity(RecordMemoryActivity, mock.Anything, mock.Anything).Return("mem-1", nil)
		env.OnActivity(SearchMemoryActivity, mock.Anything, mock.Anything).Return([]MemoryResult{{ID: "mem-1", Confidence: 0.9}}, nil)
		env.OnActivity(CheckpointSaveActivity, mock.Anything, mock.Anything).Return("ckpt-1", nil)
		env.OnActivity(CheckpointResumeActivity, mock.Anything, mock.Anything).Return(nil)
		env.OnActivity(ClearContextActivity, mock.Anything, mock.Anything).Return(nil)

		// Execute with all suites enabled
		config := TestConfig{
			RunPolicy:       true,
			RunBugfix:       true,
			RunMultiSession: true,
			ProjectID:       "test_project",
		}
		env.ExecuteWorkflow(TestOrchestratorWorkflow, config)

		require.True(t, env.IsWorkflowCompleted())
		require.NoError(t, env.GetWorkflowError())

		var report TestReport
		require.NoError(t, env.GetWorkflowResult(&report))
		assert.Len(t, report.Suites, 3)
	})

	t.Run("runs only policy suite when others disabled", func(t *testing.T) {
		testSuite := &testsuite.WorkflowTestSuite{}
		env := testSuite.NewTestWorkflowEnvironment()

		env.RegisterWorkflow(TestOrchestratorWorkflow)
		env.RegisterWorkflow(PolicyComplianceWorkflow)
		env.RegisterWorkflow(DeveloperSessionWorkflow)

		// Mock activities for DeveloperSessionWorkflow
		env.OnActivity(StartContextdActivity, mock.Anything, mock.Anything).Return(ContextdHandle{ID: "ctx-1"}, nil)
		env.OnActivity(StopContextdActivity, mock.Anything, mock.Anything).Return(nil)
		env.OnActivity(RecordMemoryActivity, mock.Anything, mock.Anything).Return("mem-1", nil)
		env.OnActivity(SearchMemoryActivity, mock.Anything, mock.Anything).Return([]MemoryResult{{ID: "mem-1", Content: "TDD test [REDACTED]", Confidence: 0.9}}, nil)

		config := TestConfig{
			RunPolicy:       true,
			RunBugfix:       false,
			RunMultiSession: false,
			ProjectID:       "test_project",
		}
		env.ExecuteWorkflow(TestOrchestratorWorkflow, config)

		require.True(t, env.IsWorkflowCompleted())
		require.NoError(t, env.GetWorkflowError())

		var report TestReport
		require.NoError(t, env.GetWorkflowResult(&report))
		assert.Len(t, report.Suites, 1)
		assert.Equal(t, "policy_compliance", report.Suites[0].SuiteName)
	})
}

// TestDeveloperSessionWorkflow validates the developer session workflow.
func TestDeveloperSessionWorkflow(t *testing.T) {
	t.Run("executes scenario steps in order", func(t *testing.T) {
		testSuite := &testsuite.WorkflowTestSuite{}
		env := testSuite.NewTestWorkflowEnvironment()

		env.RegisterWorkflow(DeveloperSessionWorkflow)

		// Mock activities
		env.OnActivity(StartContextdActivity, mock.Anything, mock.Anything).Return(ContextdHandle{ID: "ctx-1"}, nil)
		env.OnActivity(StopContextdActivity, mock.Anything, mock.Anything).Return(nil)
		env.OnActivity(RecordMemoryActivity, mock.Anything, mock.Anything).Return("mem-1", nil)
		env.OnActivity(SearchMemoryActivity, mock.Anything, mock.Anything).Return([]MemoryResult{}, nil)

		session := SessionConfig{
			Developer: DeveloperConfig{
				ID:        "dev-a",
				TenantID:  "tenant-a",
				TeamID:    "test_team",
				ProjectID: "test_project",
			},
			Steps: []SessionStep{
				{Type: "record_memory", Memory: &MemoryRecord{Title: "test", Content: "content", Outcome: "success"}},
				{Type: "search_memory", Query: "test query", Limit: 5},
			},
		}
		env.ExecuteWorkflow(DeveloperSessionWorkflow, session)

		require.True(t, env.IsWorkflowCompleted())
		require.NoError(t, env.GetWorkflowError())

		var result SessionResult
		require.NoError(t, env.GetWorkflowResult(&result))
		assert.Equal(t, "dev-a", result.Developer.ID)
		assert.Empty(t, result.Errors)
	})

	t.Run("handles checkpoint save and resume", func(t *testing.T) {
		testSuite := &testsuite.WorkflowTestSuite{}
		env := testSuite.NewTestWorkflowEnvironment()

		env.RegisterWorkflow(DeveloperSessionWorkflow)

		// Mock activities
		env.OnActivity(StartContextdActivity, mock.Anything, mock.Anything).Return(ContextdHandle{ID: "ctx-1"}, nil)
		env.OnActivity(StopContextdActivity, mock.Anything, mock.Anything).Return(nil)
		env.OnActivity(CheckpointSaveActivity, mock.Anything, mock.Anything).Return("ckpt-1", nil)
		env.OnActivity(CheckpointResumeActivity, mock.Anything, mock.Anything).Return(nil)

		session := SessionConfig{
			Developer: DeveloperConfig{
				ID:        "dev-a",
				TenantID:  "tenant-a",
				TeamID:    "test_team",
				ProjectID: "test_project",
			},
			Steps: []SessionStep{
				{Type: "checkpoint_save", Summary: "test checkpoint"},
				{Type: "checkpoint_resume", CheckpointID: "ckpt-1"},
			},
		}
		env.ExecuteWorkflow(DeveloperSessionWorkflow, session)

		require.True(t, env.IsWorkflowCompleted())
		require.NoError(t, env.GetWorkflowError())
	})

	t.Run("rejects record_memory with nil Memory", func(t *testing.T) {
		testSuite := &testsuite.WorkflowTestSuite{}
		env := testSuite.NewTestWorkflowEnvironment()

		env.RegisterWorkflow(DeveloperSessionWorkflow)

		// Mock activities
		env.OnActivity(StartContextdActivity, mock.Anything, mock.Anything).Return(ContextdHandle{ID: "ctx-1"}, nil)
		env.OnActivity(StopContextdActivity, mock.Anything, mock.Anything).Return(nil)

		session := SessionConfig{
			Developer: DeveloperConfig{
				ID:        "dev-a",
				TenantID:  "tenant-a",
				TeamID:    "test_team",
				ProjectID: "test_project",
			},
			Steps: []SessionStep{
				{Type: "record_memory", Memory: nil}, // Invalid: nil Memory
			},
		}
		env.ExecuteWorkflow(DeveloperSessionWorkflow, session)

		require.True(t, env.IsWorkflowCompleted())
		require.NoError(t, env.GetWorkflowError())

		var result SessionResult
		require.NoError(t, env.GetWorkflowResult(&result))
		require.Len(t, result.Errors, 1)
		assert.Contains(t, result.Errors[0], "requires a non-nil Memory field")
	})

	t.Run("rejects search_memory with empty Query", func(t *testing.T) {
		testSuite := &testsuite.WorkflowTestSuite{}
		env := testSuite.NewTestWorkflowEnvironment()

		env.RegisterWorkflow(DeveloperSessionWorkflow)

		// Mock activities
		env.OnActivity(StartContextdActivity, mock.Anything, mock.Anything).Return(ContextdHandle{ID: "ctx-1"}, nil)
		env.OnActivity(StopContextdActivity, mock.Anything, mock.Anything).Return(nil)

		session := SessionConfig{
			Developer: DeveloperConfig{
				ID:        "dev-a",
				TenantID:  "tenant-a",
				TeamID:    "test_team",
				ProjectID: "test_project",
			},
			Steps: []SessionStep{
				{Type: "search_memory", Query: "", Limit: 5}, // Invalid: empty Query
			},
		}
		env.ExecuteWorkflow(DeveloperSessionWorkflow, session)

		require.True(t, env.IsWorkflowCompleted())
		require.NoError(t, env.GetWorkflowError())

		var result SessionResult
		require.NoError(t, env.GetWorkflowResult(&result))
		require.Len(t, result.Errors, 1)
		assert.Contains(t, result.Errors[0], "requires a non-empty Query field")
	})

	t.Run("rejects checkpoint_resume with empty CheckpointID", func(t *testing.T) {
		testSuite := &testsuite.WorkflowTestSuite{}
		env := testSuite.NewTestWorkflowEnvironment()

		env.RegisterWorkflow(DeveloperSessionWorkflow)

		// Mock activities
		env.OnActivity(StartContextdActivity, mock.Anything, mock.Anything).Return(ContextdHandle{ID: "ctx-1"}, nil)
		env.OnActivity(StopContextdActivity, mock.Anything, mock.Anything).Return(nil)

		session := SessionConfig{
			Developer: DeveloperConfig{
				ID:        "dev-a",
				TenantID:  "tenant-a",
				TeamID:    "test_team",
				ProjectID: "test_project",
			},
			Steps: []SessionStep{
				{Type: "checkpoint_resume", CheckpointID: ""}, // Invalid: empty CheckpointID
			},
		}
		env.ExecuteWorkflow(DeveloperSessionWorkflow, session)

		require.True(t, env.IsWorkflowCompleted())
		require.NoError(t, env.GetWorkflowError())

		var result SessionResult
		require.NoError(t, env.GetWorkflowResult(&result))
		require.Len(t, result.Errors, 1)
		assert.Contains(t, result.Errors[0], "requires a non-empty CheckpointID field")
	})
}

// TestPolicyComplianceWorkflow validates policy compliance test orchestration.
func TestPolicyComplianceWorkflow(t *testing.T) {
	t.Run("runs TDD policy test", func(t *testing.T) {
		testSuite := &testsuite.WorkflowTestSuite{}
		env := testSuite.NewTestWorkflowEnvironment()

		env.RegisterWorkflow(PolicyComplianceWorkflow)
		env.RegisterWorkflow(DeveloperSessionWorkflow)

		// Mock activities for DeveloperSessionWorkflow
		// Content includes keywords that satisfy all policy test validations
		env.OnActivity(StartContextdActivity, mock.Anything, mock.Anything).Return(ContextdHandle{ID: "ctx-1"}, nil)
		env.OnActivity(StopContextdActivity, mock.Anything, mock.Anything).Return(nil)
		env.OnActivity(RecordMemoryActivity, mock.Anything, mock.Anything).Return("mem-1", nil)
		env.OnActivity(SearchMemoryActivity, mock.Anything, mock.Anything).Return([]MemoryResult{{
			ID:         "mem-1",
			Title:      "TDD Conventional Commits Policy",
			Content:    "TDD test-driven development feat: conventional commit [REDACTED]",
			Confidence: 0.9,
		}}, nil)

		config := TestConfig{
			ProjectID: "test_project",
		}
		env.ExecuteWorkflow(PolicyComplianceWorkflow, config)

		require.True(t, env.IsWorkflowCompleted())
		require.NoError(t, env.GetWorkflowError())

		var result SuiteResult
		require.NoError(t, env.GetWorkflowResult(&result))
		assert.Equal(t, "policy_compliance", result.SuiteName)
		// All three tests should run (may pass or fail based on mock data)
		assert.Equal(t, 3, result.Passed+result.Failed, "all three policy tests should be executed")
	})
}

// TestBugfixLearningWorkflow validates bug-fix learning test orchestration.
func TestBugfixLearningWorkflow(t *testing.T) {
	t.Run("tests cross-developer knowledge transfer", func(t *testing.T) {
		testSuite := &testsuite.WorkflowTestSuite{}
		env := testSuite.NewTestWorkflowEnvironment()

		env.RegisterWorkflow(BugfixLearningWorkflow)
		env.RegisterWorkflow(DeveloperSessionWorkflow)

		// Mock activities for DeveloperSessionWorkflow
		// Content includes keywords that satisfy bugfix test validations:
		// - "nil" and "user" for same_bug_retrieval
		// - "Include", "eager", or "relationship" for similar_bug_adaptation
		// - Generic content for false_positive_prevention
		env.OnActivity(StartContextdActivity, mock.Anything, mock.Anything).Return(ContextdHandle{ID: "ctx-1"}, nil)
		env.OnActivity(StopContextdActivity, mock.Anything, mock.Anything).Return(nil)
		env.OnActivity(RecordMemoryActivity, mock.Anything, mock.Anything).Return("mem-1", nil)
		env.OnActivity(SearchMemoryActivity, mock.Anything, mock.Anything).Return([]MemoryResult{{
			ID:         "mem-1",
			Title:      "nil pointer fix",
			Content:    "nil check with Include for eager relationship loading",
			Confidence: 0.9,
		}}, nil)

		config := TestConfig{
			ProjectID: "test_project",
		}
		env.ExecuteWorkflow(BugfixLearningWorkflow, config)

		require.True(t, env.IsWorkflowCompleted())
		require.NoError(t, env.GetWorkflowError())

		var result SuiteResult
		require.NoError(t, env.GetWorkflowResult(&result))
		assert.Equal(t, "bugfix_learning", result.SuiteName)
		// All three tests should run (may pass or fail based on mock data)
		assert.Equal(t, 3, result.Passed+result.Failed, "all three bugfix learning tests should be executed")
	})
}

// TestMultiSessionWorkflow validates multi-session continuity tests.
func TestMultiSessionWorkflow(t *testing.T) {
	t.Run("tests checkpoint resume across sessions", func(t *testing.T) {
		testSuite := &testsuite.WorkflowTestSuite{}
		env := testSuite.NewTestWorkflowEnvironment()

		env.RegisterWorkflow(MultiSessionWorkflow)
		env.RegisterWorkflow(DeveloperSessionWorkflow)

		// Mock activities for DeveloperSessionWorkflow
		env.OnActivity(StartContextdActivity, mock.Anything, mock.Anything).Return(ContextdHandle{ID: "ctx-1"}, nil)
		env.OnActivity(StopContextdActivity, mock.Anything, mock.Anything).Return(nil)
		env.OnActivity(RecordMemoryActivity, mock.Anything, mock.Anything).Return("mem-1", nil)
		env.OnActivity(SearchMemoryActivity, mock.Anything, mock.Anything).Return([]MemoryResult{{ID: "mem-1", Confidence: 0.9}}, nil)
		env.OnActivity(CheckpointSaveActivity, mock.Anything, mock.Anything).Return("ckpt-1", nil)
		env.OnActivity(CheckpointResumeActivity, mock.Anything, mock.Anything).Return(nil)

		config := TestConfig{
			ProjectID: "test_project",
		}
		env.ExecuteWorkflow(MultiSessionWorkflow, config)

		require.True(t, env.IsWorkflowCompleted())
		require.NoError(t, env.GetWorkflowError())

		var result SuiteResult
		require.NoError(t, env.GetWorkflowResult(&result))
		assert.Equal(t, "multi_session", result.SuiteName)
	})
}
