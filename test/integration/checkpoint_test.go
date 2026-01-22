package integration

import (
	"context"
	"testing"
	"time"

	"github.com/fyrsmithlabs/contextd/internal/checkpoint"
	"github.com/fyrsmithlabs/contextd/internal/vectorstore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// TestCheckpoint_SaveAndResume validates the complete checkpoint lifecycle.
func TestCheckpoint_SaveAndResume(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	logger := zap.NewNop()

	store, cleanup := createTestVectorStore(t)
	defer cleanup()

	tenant := &vectorstore.TenantInfo{
		TenantID:  "test-org",
		ProjectID: "test-project",
	}
	tenantCtx := vectorstore.ContextWithTenant(ctx, tenant)

	// Create checkpoint service using legacy wrapper
	cfg := checkpoint.DefaultServiceConfig()
	cs, err := checkpoint.NewServiceWithStore(cfg, store, logger)
	require.NoError(t, err, "Should create checkpoint service")

	// 1. Save a checkpoint
	saveReq := &checkpoint.SaveRequest{
		SessionID:   "session-123",
		TenantID:    tenant.TenantID,
		ProjectID:   tenant.ProjectID,
		ProjectPath: "/test/project",
		Name:        "Test Checkpoint",
		Description: "Integration test checkpoint",
		Summary:     "Analyzed codebase structure, identified integration points",
		Context:     "User asked: How do I implement feature X?\nAssistant: Here's how to implement feature X...\nUser: What about edge case Y?",
		FullState:   "Complete conversation history with all tool calls and responses...",
		TokenCount:  1500,
		Metadata: map[string]string{
			"feature": "authentication",
			"sprint":  "2",
		},
	}

	ckpt, err := cs.Save(tenantCtx, saveReq)
	require.NoError(t, err, "Should save checkpoint successfully")
	require.NotEmpty(t, ckpt.ID, "Should return checkpoint ID")

	t.Logf("✅ Saved checkpoint: %s", ckpt.ID)

	// 2. List checkpoints
	listReq := &checkpoint.ListRequest{
		TenantID:  tenant.TenantID,
		ProjectID: tenant.ProjectID,
		Limit:     10,
	}
	checkpoints, err := cs.List(tenantCtx, listReq)
	require.NoError(t, err, "Should list checkpoints successfully")
	assert.GreaterOrEqual(t, len(checkpoints), 1, "Should have at least one checkpoint")

	found := false
	for _, cp := range checkpoints {
		if cp.SessionID == "session-123" {
			found = true
			assert.Equal(t, "Test Checkpoint", cp.Name, "Should have correct name")
			assert.Equal(t, int32(1500), cp.TokenCount, "Should have correct token count")
			break
		}
	}
	assert.True(t, found, "Should find the saved checkpoint")

	t.Logf("✅ Found checkpoint in list")

	// 3. Resume from checkpoint
	resumeReq := &checkpoint.ResumeRequest{
		CheckpointID: ckpt.ID,
		TenantID:     tenant.TenantID,
		ProjectID:    tenant.ProjectID,
		Level:        "full",
	}
	resumeResp, err := cs.Resume(tenantCtx, resumeReq)
	require.NoError(t, err, "Should resume checkpoint successfully")
	require.NotNil(t, resumeResp.Checkpoint, "Should have checkpoint in response")

	resumed := resumeResp.Checkpoint
	assert.Equal(t, saveReq.SessionID, resumed.SessionID, "Session ID should match")
	assert.Equal(t, saveReq.Name, resumed.Name, "Name should match")
	assert.Equal(t, saveReq.Summary, resumed.Summary, "Summary should match")

	t.Logf("✅ Resumed checkpoint successfully")
}

// TestCheckpoint_MultiSession validates multiple concurrent sessions.
func TestCheckpoint_MultiSession(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	logger := zap.NewNop()

	store, cleanup := createTestVectorStore(t)
	defer cleanup()

	tenant := &vectorstore.TenantInfo{TenantID: "test-org"}
	tenantCtx := vectorstore.ContextWithTenant(ctx, tenant)

	cfg := checkpoint.DefaultServiceConfig()
	cs, err := checkpoint.NewServiceWithStore(cfg, store, logger)
	require.NoError(t, err)

	// Create checkpoints for multiple sessions
	sessionIDs := []string{"session-a", "session-b", "session-c"}
	for _, sessionID := range sessionIDs {
		saveReq := &checkpoint.SaveRequest{
			SessionID:   sessionID,
			TenantID:    tenant.TenantID,
			ProjectPath: "/test/project",
			Name:        "Checkpoint for " + sessionID,
			Summary:     "Test checkpoint for " + sessionID,
			Context:     "Test message for " + sessionID,
		}
		_, err := cs.Save(tenantCtx, saveReq)
		require.NoError(t, err, "Should save checkpoint for %s", sessionID)
	}

	// List all checkpoints
	listReq := &checkpoint.ListRequest{
		TenantID: tenant.TenantID,
		Limit:    10,
	}
	checkpoints, err := cs.List(tenantCtx, listReq)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(checkpoints), 3, "Should have at least 3 checkpoints")

	// Verify all sessions are present
	foundSessions := make(map[string]bool)
	for _, cp := range checkpoints {
		foundSessions[cp.SessionID] = true
	}

	for _, sessionID := range sessionIDs {
		assert.True(t, foundSessions[sessionID], "Should find session %s", sessionID)
	}

	t.Logf("✅ Multi-session checkpoints validated")
}

// TestCheckpoint_Pagination validates checkpoint listing with limits.
func TestCheckpoint_Pagination(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	logger := zap.NewNop()

	store, cleanup := createTestVectorStore(t)
	defer cleanup()

	tenant := &vectorstore.TenantInfo{TenantID: "test-org"}
	tenantCtx := vectorstore.ContextWithTenant(ctx, tenant)

	cfg := checkpoint.DefaultServiceConfig()
	cs, err := checkpoint.NewServiceWithStore(cfg, store, logger)
	require.NoError(t, err)

	// Create 10 checkpoints
	for i := 0; i < 10; i++ {
		saveReq := &checkpoint.SaveRequest{
			SessionID:   "session-pagination",
			TenantID:    tenant.TenantID,
			ProjectPath: "/test/project",
			Name:        "Pagination checkpoint",
			Summary:     "Test checkpoint for pagination",
			Context:     "Message number",
		}
		_, err := cs.Save(tenantCtx, saveReq)
		require.NoError(t, err)
		time.Sleep(10 * time.Millisecond) // Ensure different timestamps
	}

	// List with limit of 5
	listReq5 := &checkpoint.ListRequest{
		TenantID: tenant.TenantID,
		Limit:    5,
	}
	checkpoints, err := cs.List(tenantCtx, listReq5)
	require.NoError(t, err)
	assert.LessOrEqual(t, len(checkpoints), 5, "Should respect limit of 5")

	// List with limit of 20 (should get all)
	listReq20 := &checkpoint.ListRequest{
		TenantID: tenant.TenantID,
		Limit:    20,
	}
	checkpointsAll, err := cs.List(tenantCtx, listReq20)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(checkpointsAll), 10, "Should get at least 10 checkpoints")

	t.Logf("✅ Pagination works correctly (limited: %d, all: %d)",
		len(checkpoints), len(checkpointsAll))
}

// TestCheckpoint_TenantIsolation validates checkpoints are isolated by tenant.
func TestCheckpoint_TenantIsolation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	logger := zap.NewNop()

	store, cleanup := createTestVectorStore(t)
	defer cleanup()

	cfg := checkpoint.DefaultServiceConfig()
	cs, err := checkpoint.NewServiceWithStore(cfg, store, logger)
	require.NoError(t, err)

	// Create checkpoints for two tenants
	tenant1Ctx := vectorstore.ContextWithTenant(ctx, &vectorstore.TenantInfo{
		TenantID: "org-1",
	})
	tenant2Ctx := vectorstore.ContextWithTenant(ctx, &vectorstore.TenantInfo{
		TenantID: "org-2",
	})

	// Tenant 1 checkpoint
	saveReq1 := &checkpoint.SaveRequest{
		SessionID:   "tenant1-session",
		TenantID:    "org-1",
		ProjectPath: "/tenant1/project",
		Name:        "Tenant 1 Checkpoint",
		Summary:     "Tenant 1 work",
		Context:     "Tenant 1 data",
	}
	ckpt1, err := cs.Save(tenant1Ctx, saveReq1)
	require.NoError(t, err)

	// Tenant 2 checkpoint
	saveReq2 := &checkpoint.SaveRequest{
		SessionID:   "tenant2-session",
		TenantID:    "org-2",
		ProjectPath: "/tenant2/project",
		Name:        "Tenant 2 Checkpoint",
		Summary:     "Tenant 2 work",
		Context:     "Tenant 2 data",
	}
	ckpt2, err := cs.Save(tenant2Ctx, saveReq2)
	require.NoError(t, err)

	// Tenant 1 should only see their checkpoint
	listReq1 := &checkpoint.ListRequest{
		TenantID: "org-1",
		Limit:    10,
	}
	checkpoints1, err := cs.List(tenant1Ctx, listReq1)
	require.NoError(t, err)

	for _, cp := range checkpoints1 {
		assert.NotEqual(t, "tenant2-session", cp.SessionID,
			"Tenant 1 should not see Tenant 2's checkpoints")
	}

	// Tenant 2 should not be able to resume Tenant 1's checkpoint
	resumeReq1 := &checkpoint.ResumeRequest{
		CheckpointID: ckpt1.ID,
		TenantID:     "org-2", // Wrong tenant
		Level:        "summary",
	}
	_, err = cs.Resume(tenant2Ctx, resumeReq1)
	assert.Error(t, err, "Tenant 2 should not access Tenant 1's checkpoint")

	// Tenant 2 should be able to resume their own checkpoint
	resumeReq2 := &checkpoint.ResumeRequest{
		CheckpointID: ckpt2.ID,
		TenantID:     "org-2",
		Level:        "summary",
	}
	_, err = cs.Resume(tenant2Ctx, resumeReq2)
	assert.NoError(t, err, "Tenant 2 should access their own checkpoint")

	t.Logf("✅ Checkpoint tenant isolation verified")
}
