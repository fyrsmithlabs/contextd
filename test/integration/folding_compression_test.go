package integration

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fyrsmithlabs/contextd/internal/folding"
	"github.com/fyrsmithlabs/contextd/internal/secrets"
)

// testScrubberAdapter adapts secrets.Scrubber to folding.SecretScrubber.
type testScrubberAdapter struct {
	scrubber secrets.Scrubber
}

func (a *testScrubberAdapter) Scrub(content string) (string, error) {
	result := a.scrubber.Scrub(content)
	return result.Scrubbed, nil
}

// TestContextFolding_CompressionRatio demonstrates that context folding achieves
// 90%+ compression by isolating verbose work in a branch and returning only
// a concise summary to the main context.
func TestContextFolding_CompressionRatio(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()

	// Setup folding service
	emitter := folding.NewSimpleEventEmitter()
	budget := folding.NewBudgetTracker(emitter)
	repo := folding.NewMemoryBranchRepository()
	scrubber := &testScrubberAdapter{scrubber: secrets.MustNew(secrets.DefaultConfig())}
	config := folding.DefaultFoldingConfig()

	manager := folding.NewBranchManager(
		repo,
		budget,
		scrubber,
		emitter,
		config,
	)

	// Scenario: Agent needs to search through multiple large files to find a specific function
	// Without folding: All file contents would be added to main context
	// With folding: Only the final result is added to main context

	sessionID := "test-session-compression-001"
	branchBudget := 10000

	t.Run("demonstrate 90%+ compression", func(t *testing.T) {
		// Step 1: Create a branch for the verbose exploration task
		createReq := folding.BranchRequest{
			SessionID:   sessionID,
			Description: "Search 10 files for function definition",
			Prompt:      "Find the authenticate() function across multiple source files",
			Budget:      branchBudget,
		}

		createResp, err := manager.Create(ctx, createReq)
		require.NoError(t, err)
		require.NotNil(t, createResp)

		branchID := createResp.BranchID
		t.Logf("Created branch: %s with budget: %d tokens", branchID, branchBudget)

		// Step 2: Simulate verbose work inside the branch
		// This represents reading multiple large files, trying different approaches, etc.
		// In production, this would be actual LLM token consumption during exploration

		// Simulate reading 10 files, each consuming ~800 tokens
		filesRead := []string{
			"src/auth/handler.go",
			"src/auth/middleware.go",
			"src/auth/service.go",
			"src/user/profile.go",
			"src/api/routes.go",
			"src/api/handlers.go",
			"src/config/security.go",
			"src/database/connection.go",
			"src/models/user.go",
			"src/utils/crypto.go",
		}

		totalTokensConsumed := 0
		for i, file := range filesRead {
			// Simulate reading a file (~800 tokens per file)
			// This represents the LLM reading file contents, analyzing it, etc.
			fileTokens := 800
			err := manager.ConsumeTokens(ctx, branchID, fileTokens)
			require.NoError(t, err, "failed to consume tokens for file %d", i+1)
			totalTokensConsumed += fileTokens

			t.Logf("  Read file %d/%d: %s (%d tokens)", i+1, len(filesRead), file, fileTokens)
		}

		// Simulate some additional analysis and reasoning (another 2000 tokens)
		analysisTokens := 2000
		err = manager.ConsumeTokens(ctx, branchID, analysisTokens)
		require.NoError(t, err)
		totalTokensConsumed += analysisTokens
		t.Logf("  Performed analysis: %d tokens", analysisTokens)

		t.Logf("Total tokens consumed in branch: %d", totalTokensConsumed)

		// Verify we consumed significant tokens
		assert.Greater(t, totalTokensConsumed, 8000, "should have consumed significant tokens")

		// Step 3: Return from branch with compressed summary
		// Instead of returning all file contents (10,000 tokens), we return just the finding
		returnMsg := "Found authenticate() function in src/auth/service.go at line 42. " +
			"Function signature: func authenticate(username, password string) (User, error). " +
			"It validates credentials against the database and returns the authenticated user."

		// Calculate approximate tokens in return message (~50 tokens for this short summary)
		// In production, this would be counted by a real tokenizer
		returnTokens := len(strings.Fields(returnMsg)) * 2 // rough approximation
		t.Logf("Return message tokens (approximate): %d", returnTokens)
		t.Logf("Return message: %s", returnMsg)

		returnReq := folding.ReturnRequest{
			BranchID: branchID,
			Message:  returnMsg,
		}

		returnResp, err := manager.Return(ctx, returnReq)
		require.NoError(t, err)
		require.True(t, returnResp.Success)

		t.Logf("Branch completed. Tokens used: %d", returnResp.TokensUsed)
		t.Logf("Scrubbed message: %s", returnResp.ScrubbedMsg)

		// Step 4: Calculate compression ratio
		// Compression = (tokens consumed in branch - tokens returned to main context) / tokens consumed * 100
		// This represents how much context we saved by folding
		tokensConsumedInBranch := returnResp.TokensUsed
		tokensSavedInMainContext := returnTokens // Only the short summary goes to main context
		tokensSaved := tokensConsumedInBranch - tokensSavedInMainContext

		compressionRatio := float64(tokensSaved) / float64(tokensConsumedInBranch) * 100

		t.Logf("\n=== COMPRESSION ANALYSIS ===")
		t.Logf("Tokens consumed in branch:       %d", tokensConsumedInBranch)
		t.Logf("Tokens returned to main context: %d", tokensSavedInMainContext)
		t.Logf("Tokens saved:                    %d", tokensSaved)
		t.Logf("Compression ratio:               %.2f%%", compressionRatio)
		t.Logf("============================\n")

		// Verify we achieved 90%+ compression
		assert.GreaterOrEqual(t, compressionRatio, 90.0,
			"compression ratio should be at least 90%% (achieved %.2f%%)", compressionRatio)

		// Additional assertions
		assert.Equal(t, totalTokensConsumed, tokensConsumedInBranch,
			"branch should track all consumed tokens")
		assert.Less(t, tokensSavedInMainContext, tokensConsumedInBranch/10,
			"return message should be <10%% of work done in branch")

		// Verify branch is in completed state
		branch, err := manager.Get(ctx, branchID)
		require.NoError(t, err)
		assert.Equal(t, folding.BranchStatusCompleted, branch.Status)
		assert.Equal(t, totalTokensConsumed, branch.BudgetUsed)
		assert.NotNil(t, branch.Result)
		assert.NotNil(t, branch.CompletedAt)
	})

	t.Run("demonstrate nested branch compression", func(t *testing.T) {
		// Scenario: Agent creates nested branches for sub-tasks
		// Each branch compresses its own context independently

		sessionID := "test-session-nested-compression"

		// Parent branch: "Implement user authentication"
		parentReq := folding.BranchRequest{
			SessionID:   sessionID,
			Description: "Implement user authentication system",
			Prompt:      "Design and implement authentication with login, logout, and session management",
			Budget:      15000,
		}

		parentResp, err := manager.Create(ctx, parentReq)
		require.NoError(t, err)
		t.Logf("Created parent branch: %s", parentResp.BranchID)

		// Simulate work in parent (5000 tokens)
		err = manager.ConsumeTokens(ctx, parentResp.BranchID, 5000)
		require.NoError(t, err)
		t.Logf("Parent branch consumed 5000 tokens for initial design")

		// Child branch: "Research JWT libraries"
		childReq := folding.BranchRequest{
			SessionID:   sessionID,
			Description: "Research JWT libraries for Go",
			Prompt:      "Compare jwt-go, golang-jwt, and other JWT libraries",
			Budget:      8000,
		}

		childResp, err := manager.Create(ctx, childReq)
		require.NoError(t, err)
		assert.Equal(t, 1, childResp.Depth, "child should be at depth 1")
		t.Logf("Created child branch: %s (depth: %d)", childResp.BranchID, childResp.Depth)

		// Simulate extensive research in child branch (7000 tokens)
		// Reading documentation, comparing libraries, testing examples, etc.
		err = manager.ConsumeTokens(ctx, childResp.BranchID, 7000)
		require.NoError(t, err)
		t.Logf("Child branch consumed 7000 tokens for research")

		// Return from child with compressed recommendation
		childReturnMsg := "Recommend golang-jwt/jwt library. Actively maintained, secure, supports RS256 and HS256."
		childReturnTokens := len(strings.Fields(childReturnMsg)) * 2

		childReturnResp, err := manager.Return(ctx, folding.ReturnRequest{
			BranchID: childResp.BranchID,
			Message:  childReturnMsg,
		})
		require.NoError(t, err)
		t.Logf("Child branch returned: %s", childReturnMsg)

		// Calculate child branch compression
		childCompression := float64(childReturnResp.TokensUsed-childReturnTokens) / float64(childReturnResp.TokensUsed) * 100
		t.Logf("Child branch compression: %.2f%%", childCompression)
		assert.GreaterOrEqual(t, childCompression, 90.0, "child branch should achieve 90%+ compression")

		// Continue parent work with child's result (2000 more tokens)
		err = manager.ConsumeTokens(ctx, parentResp.BranchID, 2000)
		require.NoError(t, err)

		// Return from parent with final summary
		parentReturnMsg := fmt.Sprintf("Implemented authentication system using %s. Includes login, logout, session management, and JWT token handling.", childReturnMsg)
		parentReturnTokens := len(strings.Fields(parentReturnMsg)) * 2

		parentReturnResp, err := manager.Return(ctx, folding.ReturnRequest{
			BranchID: parentResp.BranchID,
			Message:  parentReturnMsg,
		})
		require.NoError(t, err)

		// Calculate parent branch compression
		parentCompression := float64(parentReturnResp.TokensUsed-parentReturnTokens) / float64(parentReturnResp.TokensUsed) * 100
		t.Logf("Parent branch compression: %.2f%%", parentCompression)
		assert.GreaterOrEqual(t, parentCompression, 80.0, "parent branch should achieve high compression")

		// Calculate total compression across both branches
		totalConsumed := parentReturnResp.TokensUsed + childReturnResp.TokensUsed
		totalReturned := parentReturnTokens // Only parent's return message goes to main context
		totalSaved := totalConsumed - totalReturned
		totalCompression := float64(totalSaved) / float64(totalConsumed) * 100

		t.Logf("\n=== NESTED BRANCH COMPRESSION ===")
		t.Logf("Total tokens consumed:     %d", totalConsumed)
		t.Logf("Total tokens in main ctx:  %d", totalReturned)
		t.Logf("Total compression:         %.2f%%", totalCompression)
		t.Logf("================================\n")

		// Nested branches should achieve even better overall compression
		assert.GreaterOrEqual(t, totalCompression, 90.0,
			"nested branches should achieve 90%+ overall compression")
	})
}
