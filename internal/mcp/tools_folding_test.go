package mcp

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/fyrsmithlabs/contextd/internal/checkpoint"
	"github.com/fyrsmithlabs/contextd/internal/folding"
	"github.com/fyrsmithlabs/contextd/internal/reasoningbank"
	"github.com/fyrsmithlabs/contextd/internal/remediation"
	"github.com/fyrsmithlabs/contextd/internal/repository"
	"github.com/fyrsmithlabs/contextd/internal/secrets"
	"github.com/fyrsmithlabs/contextd/internal/troubleshoot"
)

// testScrubberAdapter adapts secrets.Scrubber to folding.SecretScrubber.
type testScrubberAdapter struct {
	scrubber secrets.Scrubber
}

func (a *testScrubberAdapter) Scrub(content string) (string, error) {
	result := a.scrubber.Scrub(content)
	return result.Scrubbed, nil
}

// setupFoldingTestServer creates an MCP server with folding enabled for testing.
func setupFoldingTestServer(t *testing.T) (*Server, *folding.BranchManager) {
	t.Helper()
	logger := zap.NewNop()

	// Create mock stores
	troubleshootStore := &mockTroubleshootStore{}
	vectorStore := &mockVectorStore{}

	// Create required services
	checkpointSvc, err := checkpoint.NewServiceWithStore(checkpoint.DefaultServiceConfig(), vectorStore, logger)
	require.NoError(t, err)

	remediationSvc, err := remediation.NewService(remediation.DefaultServiceConfig(), vectorStore, logger)
	require.NoError(t, err)

	repositorySvc := repository.NewService(vectorStore)
	troubleshootSvc, err := troubleshoot.NewService(troubleshootStore, logger, nil)
	require.NoError(t, err)
	reasoningbankSvc, err := reasoningbank.NewService(vectorStore, logger)
	require.NoError(t, err)
	scrubber := secrets.MustNew(secrets.DefaultConfig())

	// Create folding service with all dependencies
	foldingEmitter := folding.NewSimpleEventEmitter()
	foldingBudget := folding.NewBudgetTracker(foldingEmitter)
	foldingRepo := folding.NewMemoryBranchRepository()
	foldingScrubber := &testScrubberAdapter{scrubber: scrubber}
	foldingConfig := folding.DefaultFoldingConfig()

	foldingSvc := folding.NewBranchManager(
		foldingRepo,
		foldingBudget,
		foldingScrubber,
		foldingEmitter,
		foldingConfig,
	)

	// Create MCP server with folding enabled
	cfg := &Config{
		Name:    "test-server-with-folding",
		Version: "1.0.0",
		Logger:  logger,
	}

	server, err := NewServer(cfg, checkpointSvc, remediationSvc, repositorySvc, troubleshootSvc, reasoningbankSvc, foldingSvc, nil, scrubber)
	require.NoError(t, err)

	return server, foldingSvc
}

// TestFoldingTools_BranchCreateIntegration tests branch_create via the actual service.
func TestFoldingTools_BranchCreateIntegration(t *testing.T) {
	server, foldingSvc := setupFoldingTestServer(t)
	defer server.Close()

	ctx := context.Background()

	t.Run("create branch successfully", func(t *testing.T) {
		req := folding.BranchRequest{
			SessionID:      "test-session-001",
			Description:    "Test branch for integration testing",
			Prompt:         "Execute the test task",
			Budget:         4096,
			TimeoutSeconds: 60,
		}

		resp, err := foldingSvc.Create(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, resp)

		assert.NotEmpty(t, resp.BranchID)
		assert.True(t, len(resp.BranchID) > 0, "branch ID should be generated")
		assert.Equal(t, 4096, resp.BudgetAllocated)
		assert.Equal(t, 0, resp.Depth, "first branch should be at depth 0")
	})

	t.Run("create nested branch", func(t *testing.T) {
		// Create parent branch
		parentReq := folding.BranchRequest{
			SessionID:   "test-session-002",
			Description: "Parent branch",
			Prompt:      "Parent task",
		}
		parentResp, err := foldingSvc.Create(ctx, parentReq)
		require.NoError(t, err)

		// Create child branch in same session
		childReq := folding.BranchRequest{
			SessionID:   "test-session-002",
			Description: "Child branch",
			Prompt:      "Child task",
		}
		childResp, err := foldingSvc.Create(ctx, childReq)
		require.NoError(t, err)

		assert.Equal(t, 1, childResp.Depth, "child branch should be at depth 1")
		assert.NotEqual(t, parentResp.BranchID, childResp.BranchID)
	})

	t.Run("reject empty session ID", func(t *testing.T) {
		req := folding.BranchRequest{
			SessionID:   "",
			Description: "Invalid branch",
			Prompt:      "Task",
		}

		_, err := foldingSvc.Create(ctx, req)
		require.Error(t, err)
	})

	t.Run("reject empty description", func(t *testing.T) {
		req := folding.BranchRequest{
			SessionID:   "test-session",
			Description: "",
			Prompt:      "Task",
		}

		_, err := foldingSvc.Create(ctx, req)
		require.Error(t, err)
	})
}

// TestFoldingTools_BranchReturnIntegration tests branch_return via the actual service.
func TestFoldingTools_BranchReturnIntegration(t *testing.T) {
	server, foldingSvc := setupFoldingTestServer(t)
	defer server.Close()

	ctx := context.Background()

	t.Run("return from branch successfully", func(t *testing.T) {
		// Create a branch first
		createReq := folding.BranchRequest{
			SessionID:   "test-session-return-001",
			Description: "Branch for return testing",
			Prompt:      "Test task",
		}
		createResp, err := foldingSvc.Create(ctx, createReq)
		require.NoError(t, err)

		// Return from the branch
		returnReq := folding.ReturnRequest{
			BranchID: createResp.BranchID,
			Message:  "Task completed successfully",
		}
		returnResp, err := foldingSvc.Return(ctx, returnReq)
		require.NoError(t, err)

		assert.True(t, returnResp.Success)
		assert.Equal(t, "Task completed successfully", returnResp.ScrubbedMsg)
	})

	t.Run("scrub secrets from return message", func(t *testing.T) {
		// Create a branch
		createReq := folding.BranchRequest{
			SessionID:   "test-session-secret-001",
			Description: "Branch for secret scrubbing test",
			Prompt:      "Test task",
		}
		createResp, err := foldingSvc.Create(ctx, createReq)
		require.NoError(t, err)

		// Return with a message containing a secret (AWS key pattern)
		returnReq := folding.ReturnRequest{
			BranchID: createResp.BranchID,
			Message:  "Found API key: AKIAIOSFODNN7EXAMPLE in the config",
		}
		returnResp, err := foldingSvc.Return(ctx, returnReq)
		require.NoError(t, err)

		assert.True(t, returnResp.Success)
		// The AWS key should be redacted
		assert.NotContains(t, returnResp.ScrubbedMsg, "AKIAIOSFODNN7EXAMPLE",
			"secret should be scrubbed from return message")
		assert.Contains(t, returnResp.ScrubbedMsg, "[REDACTED]",
			"scrubbed message should contain redaction marker")
	})

	t.Run("return from non-existent branch fails", func(t *testing.T) {
		returnReq := folding.ReturnRequest{
			BranchID: "br_nonexistent",
			Message:  "This should fail",
		}
		_, err := foldingSvc.Return(ctx, returnReq)
		require.Error(t, err)
	})

	t.Run("double return from same branch fails", func(t *testing.T) {
		// Create a branch
		createReq := folding.BranchRequest{
			SessionID:   "test-session-double-001",
			Description: "Branch for double return test",
			Prompt:      "Test task",
		}
		createResp, err := foldingSvc.Create(ctx, createReq)
		require.NoError(t, err)

		// First return succeeds
		returnReq := folding.ReturnRequest{
			BranchID: createResp.BranchID,
			Message:  "First return",
		}
		_, err = foldingSvc.Return(ctx, returnReq)
		require.NoError(t, err)

		// Second return should fail (branch no longer active)
		_, err = foldingSvc.Return(ctx, returnReq)
		require.Error(t, err)
	})
}

// TestFoldingTools_BranchStatusIntegration tests branch_status via the actual service.
func TestFoldingTools_BranchStatusIntegration(t *testing.T) {
	server, foldingSvc := setupFoldingTestServer(t)
	defer server.Close()

	ctx := context.Background()

	t.Run("get status of active branch", func(t *testing.T) {
		// Create a branch
		createReq := folding.BranchRequest{
			SessionID:   "test-session-status-001",
			Description: "Branch for status testing",
			Prompt:      "Test task",
			Budget:      2048,
		}
		createResp, err := foldingSvc.Create(ctx, createReq)
		require.NoError(t, err)

		// Get branch status by ID
		branch, err := foldingSvc.Get(ctx, createResp.BranchID)
		require.NoError(t, err)
		require.NotNil(t, branch)

		assert.Equal(t, createResp.BranchID, branch.ID)
		assert.Equal(t, folding.BranchStatusActive, branch.Status)
		assert.Equal(t, 2048, branch.BudgetTotal)
		assert.Equal(t, 0, branch.BudgetUsed)
	})

	t.Run("get active branch for session", func(t *testing.T) {
		sessionID := "test-session-status-002"

		// Create a branch
		createReq := folding.BranchRequest{
			SessionID:   sessionID,
			Description: "Branch for active status test",
			Prompt:      "Test task",
		}
		createResp, err := foldingSvc.Create(ctx, createReq)
		require.NoError(t, err)

		// Get active branch by session
		branch, err := foldingSvc.GetActive(ctx, sessionID)
		require.NoError(t, err)
		require.NotNil(t, branch)

		assert.Equal(t, createResp.BranchID, branch.ID)
		assert.Equal(t, sessionID, branch.SessionID)
	})

	t.Run("no active branch returns nil", func(t *testing.T) {
		branch, err := foldingSvc.GetActive(ctx, "nonexistent-session")
		require.NoError(t, err)
		assert.Nil(t, branch)
	})

	t.Run("status after return shows completed", func(t *testing.T) {
		// Create and return from a branch
		createReq := folding.BranchRequest{
			SessionID:   "test-session-status-003",
			Description: "Branch for completed status test",
			Prompt:      "Test task",
		}
		createResp, err := foldingSvc.Create(ctx, createReq)
		require.NoError(t, err)

		returnReq := folding.ReturnRequest{
			BranchID: createResp.BranchID,
			Message:  "Done",
		}
		_, err = foldingSvc.Return(ctx, returnReq)
		require.NoError(t, err)

		// Check status is completed
		branch, err := foldingSvc.Get(ctx, createResp.BranchID)
		require.NoError(t, err)
		require.NotNil(t, branch)

		assert.Equal(t, folding.BranchStatusCompleted, branch.Status)
	})
}

// TestFoldingTools_MaxDepthEnforcement tests that max depth limits are enforced.
func TestFoldingTools_MaxDepthEnforcement(t *testing.T) {
	server, foldingSvc := setupFoldingTestServer(t)
	defer server.Close()

	ctx := context.Background()
	sessionID := "test-session-depth"

	// Default max depth is 3, so we should be able to create 3 nested branches (depths 0, 1, 2)
	// The 4th branch (depth 3) should fail

	for i := 0; i < 3; i++ {
		req := folding.BranchRequest{
			SessionID:   sessionID,
			Description: "Branch at depth " + string(rune('0'+i)),
			Prompt:      "Task",
		}
		resp, err := foldingSvc.Create(ctx, req)
		require.NoError(t, err, "should create branch at depth %d", i)
		assert.Equal(t, i, resp.Depth)
	}

	// 4th branch should fail (would be depth 3, exceeds max depth of 3)
	req := folding.BranchRequest{
		SessionID:   sessionID,
		Description: "Branch exceeding max depth",
		Prompt:      "Task",
	}
	_, err := foldingSvc.Create(ctx, req)
	require.Error(t, err, "should reject branch exceeding max depth")
	assert.Contains(t, err.Error(), "depth", "error should mention depth")
}

// TestFoldingTools_BudgetTracking tests that budget is tracked correctly.
func TestFoldingTools_BudgetTracking(t *testing.T) {
	server, foldingSvc := setupFoldingTestServer(t)
	defer server.Close()

	ctx := context.Background()

	t.Run("budget allocated correctly", func(t *testing.T) {
		req := folding.BranchRequest{
			SessionID:   "test-session-budget-001",
			Description: "Budget test branch",
			Prompt:      "Task",
			Budget:      1000,
		}
		resp, err := foldingSvc.Create(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, 1000, resp.BudgetAllocated)

		// Check branch has correct budget
		branch, err := foldingSvc.Get(ctx, resp.BranchID)
		require.NoError(t, err)
		assert.Equal(t, 1000, branch.BudgetTotal)
		assert.Equal(t, 0, branch.BudgetUsed)
		assert.Equal(t, 1000, branch.BudgetRemaining())
	})

	t.Run("default budget applied when not specified", func(t *testing.T) {
		req := folding.BranchRequest{
			SessionID:   "test-session-budget-002",
			Description: "Default budget test",
			Prompt:      "Task",
			// Budget not specified
		}
		resp, err := foldingSvc.Create(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, 8192, resp.BudgetAllocated, "should use default budget")
	})

	t.Run("consume tokens tracked and reflected on return", func(t *testing.T) {
		req := folding.BranchRequest{
			SessionID:   "test-session-budget-003",
			Description: "Token consumption test",
			Prompt:      "Task",
			Budget:      1000,
		}
		resp, err := foldingSvc.Create(ctx, req)
		require.NoError(t, err)

		// Consume some tokens (tracked in BudgetTracker)
		err = foldingSvc.ConsumeTokens(ctx, resp.BranchID, 250)
		require.NoError(t, err)

		// Note: Branch.BudgetUsed is only synchronized on Return()
		// The BudgetTracker tracks real-time consumption
		// Return from branch to synchronize
		returnReq := folding.ReturnRequest{
			BranchID: resp.BranchID,
			Message:  "Done",
		}
		returnResp, err := foldingSvc.Return(ctx, returnReq)
		require.NoError(t, err)

		// TokensUsed in return response should reflect consumed tokens
		assert.Equal(t, 250, returnResp.TokensUsed)

		// Branch should now have updated BudgetUsed
		branch, err := foldingSvc.Get(ctx, resp.BranchID)
		require.NoError(t, err)
		assert.Equal(t, 250, branch.BudgetUsed)
	})
}

// TestFoldingTools_ChildBranchCleanup tests that child branches are cleaned up on parent return.
func TestFoldingTools_ChildBranchCleanup(t *testing.T) {
	server, foldingSvc := setupFoldingTestServer(t)
	defer server.Close()

	ctx := context.Background()
	sessionID := "test-session-cleanup"

	// Create parent branch
	parentReq := folding.BranchRequest{
		SessionID:   sessionID,
		Description: "Parent branch",
		Prompt:      "Parent task",
	}
	parentResp, err := foldingSvc.Create(ctx, parentReq)
	require.NoError(t, err)

	// Create child branch
	childReq := folding.BranchRequest{
		SessionID:   sessionID,
		Description: "Child branch",
		Prompt:      "Child task",
	}
	childResp, err := foldingSvc.Create(ctx, childReq)
	require.NoError(t, err)
	assert.Equal(t, 1, childResp.Depth)

	// Return from parent (should force-return child first)
	returnReq := folding.ReturnRequest{
		BranchID: parentResp.BranchID,
		Message:  "Parent done",
	}
	_, err = foldingSvc.Return(ctx, returnReq)
	require.NoError(t, err)

	// Child should now be in failed/force-returned state
	childBranch, err := foldingSvc.Get(ctx, childResp.BranchID)
	require.NoError(t, err)
	assert.True(t, childBranch.Status.IsTerminal(), "child should be in terminal state")
}

// TestFoldingServerHealth tests the BranchManager health endpoint.
func TestFoldingServerHealth(t *testing.T) {
	server, foldingSvc := setupFoldingTestServer(t)
	defer server.Close()

	health := foldingSvc.Health()
	assert.True(t, health.Healthy)
	assert.Equal(t, int64(0), health.ActiveCount)
	assert.False(t, health.IsShutdown)
}
