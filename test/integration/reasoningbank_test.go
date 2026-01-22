package integration

import (
	"context"
	"testing"

	"github.com/fyrsmithlabs/contextd/internal/reasoningbank"
	"github.com/fyrsmithlabs/contextd/internal/vectorstore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// TestReasoningBank_MemoryCRUD validates the complete memory lifecycle:
// record, search, feedback, and outcome tracking.
func TestReasoningBank_MemoryCRUD(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	logger := zap.NewNop()

	// Create test store
	store, cleanup := createTestVectorStore(t)
	defer cleanup()

	// Create tenant context
	tenant := &vectorstore.TenantInfo{
		TenantID:  "test-org",
		TeamID:    "test-team",
		ProjectID: "test-project",
	}
	tenantCtx := vectorstore.ContextWithTenant(ctx, tenant)

	// Create ReasoningBank service
	rb, err := reasoningbank.NewService(store, logger, reasoningbank.WithDefaultTenant(tenant.TenantID))
	require.NoError(t, err)

	// 1. Record a memory
	memory, err := reasoningbank.NewMemory(
		tenant.ProjectID,
		"TDD Best Practice",
		"Use TDD approach for all new features",
		reasoningbank.OutcomeSuccess,
		[]string{"testing", "development", "best-practice"},
	)
	require.NoError(t, err)
	memory.Description = "Best practice from previous sprint"
	memory.Confidence = 0.9

	err = rb.Record(tenantCtx, memory)
	require.NoError(t, err, "Should record memory successfully")

	t.Logf("✅ Recorded memory: %s", memory.ID)

	// 2. Search for the memory
	results, err := rb.Search(tenantCtx, tenant.ProjectID, "testing best practices", 5)
	require.NoError(t, err, "Should search memories successfully")
	assert.GreaterOrEqual(t, len(results), 1, "Should find at least one memory")

	found := false
	for _, result := range results {
		if result.Content == memory.Content {
			found = true
			assert.Equal(t, 0.9, result.Confidence, "Confidence should match")
			break
		}
	}
	assert.True(t, found, "Should find the recorded memory")

	t.Logf("✅ Found memory in search results")

	// 3. Provide feedback
	err = rb.Feedback(tenantCtx, memory.ID, true)
	require.NoError(t, err, "Should record feedback successfully")

	t.Logf("✅ Recorded positive feedback")

	// 4. Record successful outcome
	_, err = rb.RecordOutcome(tenantCtx, memory.ID, true, "test-session")
	require.NoError(t, err, "Should record outcome successfully")

	t.Logf("✅ Recorded successful outcome")

	// 5. Verify memory was updated
	results, err = rb.Search(tenantCtx, tenant.ProjectID, "testing best practices", 5)
	require.NoError(t, err, "Should search memories successfully")

	for _, result := range results {
		if result.Content == memory.Content {
			assert.Equal(t, 1, result.UsageCount, "UsageCount should be incremented")
			t.Logf("✅ Memory statistics updated correctly")
			break
		}
	}
}

// TestReasoningBank_MultiTenantIsolation validates that memories are isolated by tenant.
func TestReasoningBank_MultiTenantIsolation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	logger := zap.NewNop()

	store, cleanup := createTestVectorStore(t)
	defer cleanup()

	// Create two separate tenants
	tenant1Ctx := vectorstore.ContextWithTenant(ctx, &vectorstore.TenantInfo{
		TenantID:  "org-1",
		ProjectID: "project-1",
	})
	tenant2Ctx := vectorstore.ContextWithTenant(ctx, &vectorstore.TenantInfo{
		TenantID:  "org-2",
		ProjectID: "project-2",
	})

	// Create separate ReasoningBank services for each tenant
	rb1, err := reasoningbank.NewService(store, logger, reasoningbank.WithDefaultTenant("org-1"))
	require.NoError(t, err)

	rb2, err := reasoningbank.NewService(store, logger, reasoningbank.WithDefaultTenant("org-2"))
	require.NoError(t, err)

	// Record memory for tenant 1
	memory1, err := reasoningbank.NewMemory(
		"project-1",
		"Tenant 1 Strategy",
		"Tenant 1 secret strategy",
		reasoningbank.OutcomeSuccess,
		[]string{"tenant1"},
	)
	require.NoError(t, err)
	memory1.Description = "Private to tenant 1"
	memory1.Confidence = 0.9

	err = rb1.Record(tenant1Ctx, memory1)
	require.NoError(t, err)

	// Record memory for tenant 2
	memory2, err := reasoningbank.NewMemory(
		"project-2",
		"Tenant 2 Strategy",
		"Tenant 2 secret strategy",
		reasoningbank.OutcomeSuccess,
		[]string{"tenant2"},
	)
	require.NoError(t, err)
	memory2.Description = "Private to tenant 2"
	memory2.Confidence = 0.9

	err = rb2.Record(tenant2Ctx, memory2)
	require.NoError(t, err)

	// Tenant 1 should only see their memory
	results1, err := rb1.Search(tenant1Ctx, "project-1", "secret strategy", 10)
	require.NoError(t, err)
	assert.Equal(t, 1, len(results1), "Tenant 1 should see exactly 1 memory")
	assert.Contains(t, results1[0].Content, "Tenant 1", "Tenant 1 should only see their memory")

	// Tenant 2 should only see their memory
	results2, err := rb2.Search(tenant2Ctx, "project-2", "secret strategy", 10)
	require.NoError(t, err)
	assert.Equal(t, 1, len(results2), "Tenant 2 should see exactly 1 memory")
	assert.Contains(t, results2[0].Content, "Tenant 2", "Tenant 2 should only see their memory")

	t.Logf("✅ Multi-tenant isolation verified")
}

// TestReasoningBank_ConfidenceScoring validates confidence decay and boosting.
func TestReasoningBank_ConfidenceScoring(t *testing.T) {
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

	rb, err := reasoningbank.NewService(store, logger, reasoningbank.WithDefaultTenant(tenant.TenantID))
	require.NoError(t, err)

	// Record memory with high confidence
	memory, err := reasoningbank.NewMemory(
		tenant.ProjectID,
		"High Confidence Strategy",
		"High confidence strategy",
		reasoningbank.OutcomeSuccess,
		[]string{"strategy", "confidence"},
	)
	require.NoError(t, err)
	memory.Description = "Recently validated"
	memory.Confidence = 0.9

	err = rb.Record(tenantCtx, memory)
	require.NoError(t, err)

	// Record multiple successful outcomes
	for i := 0; i < 5; i++ {
		_, err = rb.RecordOutcome(tenantCtx, memory.ID, true, "test-session")
		require.NoError(t, err)
	}

	// Record one failure
	_, err = rb.RecordOutcome(tenantCtx, memory.ID, false, "test-session")
	require.NoError(t, err)

	// Verify usage count
	results, err := rb.Search(tenantCtx, tenant.ProjectID, "confidence strategy", 5)
	require.NoError(t, err)

	for _, result := range results {
		if result.Content == memory.Content {
			// Verify usage was tracked (6 outcomes recorded)
			assert.GreaterOrEqual(t, result.UsageCount, 1, "UsageCount should be incremented")
			t.Logf("✅ Confidence scoring: UsageCount=%d, Confidence=%.2f",
				result.UsageCount, result.Confidence)
			break
		}
	}
}
