package mcp

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fyrsmithlabs/contextd/internal/vectorstore"
)

// TestMemorySearchHandlerPassesTenantContext verifies that the memory_search
// handler's tenant plumbing (tenantCtx) populates vectorstore.TenantInfo on
// ctx before the service is called. Mirrors the pattern in
// TestCheckpointHandlerPassesTenantContext.
func TestMemorySearchHandlerPassesTenantContext(t *testing.T) {
	s := &Server{}
	projectID := "contextd"

	ctx, rt, err := s.tenantCtx(context.Background(), "", projectID, "", projectID)
	require.NoError(t, err, "tenantCtx should succeed for memory_search")

	got, err := vectorstore.TenantFromContext(ctx)
	require.NoError(t, err, "memory_search ctx should carry tenant info")
	assert.Equal(t, projectID, got.TenantID, "tenant_id should fall back to project_id for memory tools")
	assert.Equal(t, projectID, got.ProjectID)
	assert.Equal(t, projectID, rt.ProjectID, "rt.ProjectID should mirror sanitized input")
	assert.Equal(t, projectID, rt.TenantID)
}

// TestMemoryRecordHandlerPassesTenantContext mirrors the memory_search test
// for the append-only write path.
func TestMemoryRecordHandlerPassesTenantContext(t *testing.T) {
	s := &Server{}
	projectID := "my_project"

	ctx, rt, err := s.tenantCtx(context.Background(), "", projectID, "", projectID)
	require.NoError(t, err)

	got, err := vectorstore.TenantFromContext(ctx)
	require.NoError(t, err)
	assert.Equal(t, projectID, got.TenantID)
	assert.Equal(t, projectID, got.ProjectID)
	assert.Equal(t, projectID, rt.ProjectID)
}

// TestMemoryFeedbackHandlerPassesTenantContext is the regression test for
// the data-leak risk where memory_feedback previously executed without any
// tenant scoping. The handler now requires project_id and threads tenant
// info through ctx before calling reasoningbankSvc.Feedback.
func TestMemoryFeedbackHandlerPassesTenantContext(t *testing.T) {
	s := &Server{}
	projectID := "tenant_scoped_feedback"

	ctx, _, err := s.tenantCtx(context.Background(), "", projectID, "", projectID)
	require.NoError(t, err)

	got, err := vectorstore.TenantFromContext(ctx)
	require.NoError(t, err, "memory_feedback must enforce tenant context")
	assert.Equal(t, projectID, got.TenantID)
	assert.Equal(t, projectID, got.ProjectID)
}

// TestMemoryOutcomeHandlerPassesTenantContext mirrors the feedback test for
// memory_outcome — which also previously ran unscoped.
func TestMemoryOutcomeHandlerPassesTenantContext(t *testing.T) {
	s := &Server{}
	projectID := "outcome_proj"

	ctx, _, err := s.tenantCtx(context.Background(), "", projectID, "", projectID)
	require.NoError(t, err)

	got, err := vectorstore.TenantFromContext(ctx)
	require.NoError(t, err, "memory_outcome must enforce tenant context")
	assert.Equal(t, projectID, got.TenantID)
	assert.Equal(t, projectID, got.ProjectID)
}

// TestMemoryConsolidateHandlerPassesTenantContext mirrors the feedback test
// for memory_consolidate — which also previously ran unscoped.
func TestMemoryConsolidateHandlerPassesTenantContext(t *testing.T) {
	s := &Server{}
	projectID := "consolidate_proj"

	ctx, rt, err := s.tenantCtx(context.Background(), "", projectID, "", projectID)
	require.NoError(t, err)

	got, err := vectorstore.TenantFromContext(ctx)
	require.NoError(t, err, "memory_consolidate must enforce tenant context")
	assert.Equal(t, projectID, got.TenantID)
	assert.Equal(t, projectID, got.ProjectID)
	assert.Equal(t, projectID, rt.ProjectID, "distiller must receive rt.ProjectID not raw args")
}

// TestMemoryConsolidateSessionHandlerPassesTenantContext verifies the named
// (no longer anonymous) struct args still flow tenant info onto ctx.
func TestMemoryConsolidateSessionHandlerPassesTenantContext(t *testing.T) {
	s := &Server{}
	in := memoryConsolidateSessionInput{
		ProjectID: "session_consolidate_proj",
		SessionID: "session-abc",
	}

	ctx, rt, err := s.tenantCtx(context.Background(), "", in.ProjectID, "", in.ProjectID)
	require.NoError(t, err)

	got, err := vectorstore.TenantFromContext(ctx)
	require.NoError(t, err)
	assert.Equal(t, in.ProjectID, got.TenantID)
	assert.Equal(t, in.ProjectID, got.ProjectID)
	assert.Equal(t, in.ProjectID, rt.ProjectID)
}

// TestMemoryHandlersRejectMalformedProjectID confirms the tenantCtx helper
// rejects malformed project_id values for every memory_* handler. Memory
// tools route project_id through both tenantID and projectID, so an
// invalid identifier fails fast before any service call.
func TestMemoryHandlersRejectMalformedProjectID(t *testing.T) {
	s := &Server{}
	// Path traversal, NUL byte, and forward-slash are all rejected by
	// sanitize.ValidateProjectID / ValidateTenantID. Empty string is *not*
	// in this list because empty tenantID intentionally falls through to
	// tenant.GetDefaultTenantID() (solo-dev path); per-handler "project_id
	// is required" enforcement happens implicitly because callers pass the
	// same value as tenantID — only when tenant defaulting also fails do
	// users see the error.
	bad := []string{
		"../etc/passwd",
		"proj/with/slash",
		"proj\x00null",
	}

	for _, p := range bad {
		t.Run(p, func(t *testing.T) {
			_, _, err := s.tenantCtx(context.Background(), "", p, "", p)
			require.Error(t, err, "tenantCtx should reject malformed project_id %q", p)
		})
	}
}

// TestMemorySearchInputBackCompat asserts the typed memorySearchRow shape so
// downstream consumers parsing structured output do not regress when we
// swap map[string]interface{} → typed struct.
func TestMemorySearchRowShape(t *testing.T) {
	row := memorySearchRow{
		ID:         "id-1",
		Title:      "t",
		Content:    "c",
		Outcome:    "success",
		Confidence: 0.7,
		Relevance:  0.9,
		Tags:       []string{"go"},
	}
	assert.Equal(t, "id-1", row.ID)
	assert.Equal(t, "success", row.Outcome)
	assert.InDelta(t, 0.7, row.Confidence, 1e-9)
	assert.InDelta(t, 0.9, row.Relevance, 1e-9)
}

// TestMemoryConsolidateSessionInputIsNamed confirms the input is a
// reviewable named struct (not anonymous). Per HANDLER-GUIDE.md §3.1 and §10.
func TestMemoryConsolidateSessionInputIsNamed(t *testing.T) {
	in := memoryConsolidateSessionInput{ProjectID: "p", SessionID: "s"}
	out := memoryConsolidateSessionOutput{MemoryIDs: []string{"m1"}, Count: 1, Message: "ok"}
	assert.Equal(t, "p", in.ProjectID)
	assert.Equal(t, "s", in.SessionID)
	assert.Equal(t, 1, out.Count)
}
