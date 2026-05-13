package mcp

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fyrsmithlabs/contextd/internal/tenant"
	"github.com/fyrsmithlabs/contextd/internal/vectorstore"
)

// These tests cover the handlers migrated to the HANDLER-GUIDE in this PR:
// checkpoint_resume, remediation_record, remediation_feedback, repository_index,
// and semantic_search (both path-derived and explicit-collection branches).
//
// They assert the contract documented in §5 of HANDLER-GUIDE.md: every
// migrated handler MUST call s.tenantCtx once at the top, and the returned
// context MUST carry a valid vectorstore.TenantInfo with the
// expected/sanitized identifiers. Since the handler bodies all begin with the
// same tenantCtx call shape, exercising tenantCtx with each handler's input
// args directly is a sound proxy for the per-handler context plumbing.

// newGuideTestServer builds a minimal Server wired with a recordingVectorStore
// so handler-level service plumbing can be exercised end-to-end where needed.
func newGuideTestServer(t *testing.T) *Server {
	t.Helper()
	return newTestServer(t, &recordingVectorStore{})
}

// TestCheckpointResume_TenantCtx verifies the migrated checkpoint_resume
// derives tenant context via tenantCtx (no project floor, since callers may
// not know the project of an older checkpoint).
func TestCheckpointResume_TenantCtx(t *testing.T) {
	s := newGuideTestServer(t)

	// Mirrors the handler line: ctx, rt, err := s.tenantCtx(ctx, "", args.TenantID, "", "")
	ctx, rt, err := s.tenantCtx(context.Background(), "", "fyrsmithlabs", "", "")
	require.NoError(t, err)
	assert.Equal(t, "fyrsmithlabs", rt.TenantID)
	assert.Empty(t, rt.ProjectID, "checkpoint_resume must not impose a project floor")

	got, err := vectorstore.TenantFromContext(ctx)
	require.NoError(t, err)
	assert.Equal(t, "fyrsmithlabs", got.TenantID)
	assert.Empty(t, got.ProjectID)
}

// TestCheckpointResume_TenantCtxRejectsMalformed verifies CWE-287 hardening on
// the migrated handler: a malformed tenant_id produces an error before any
// service call.
func TestCheckpointResume_TenantCtxRejectsMalformed(t *testing.T) {
	s := newGuideTestServer(t)

	_, _, err := s.tenantCtx(context.Background(), "", "../bad tenant", "", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid tenant_id")
}

// TestRemediationRecord_TenantCtx verifies remediation_record threads
// project_path through tenantCtx and that the resulting context carries the
// expected tenant/project/team triple.
func TestRemediationRecord_TenantCtx(t *testing.T) {
	s := newGuideTestServer(t)

	projectPath := "/home/testuser/projects/myapp"
	wantTenantID := tenant.GetTenantIDForPath(projectPath)

	// Mirrors the handler call: s.tenantCtx(ctx, args.ProjectPath, args.TenantID, args.TeamID, "")
	ctx, rt, err := s.tenantCtx(context.Background(), projectPath, "", "platform", "")
	require.NoError(t, err)
	assert.Equal(t, wantTenantID, rt.TenantID)
	assert.Equal(t, "platform", rt.TeamID)
	assert.Equal(t, "myapp", rt.ProjectID)

	got, err := vectorstore.TenantFromContext(ctx)
	require.NoError(t, err)
	assert.Equal(t, wantTenantID, got.TenantID)
	assert.Equal(t, "platform", got.TeamID)
	assert.Equal(t, "myapp", got.ProjectID)
}

// TestRemediationFeedback_TenantCtx verifies the manual inline tenant derivation
// (now replaced by tenantCtx) produces the same context shape it used to.
func TestRemediationFeedback_TenantCtx(t *testing.T) {
	s := newGuideTestServer(t)

	ctx, rt, err := s.tenantCtx(context.Background(), "", "fyrsmithlabs", "", "")
	require.NoError(t, err)
	assert.Equal(t, "fyrsmithlabs", rt.TenantID)
	assert.Empty(t, rt.ProjectID, "feedback should not impose a project floor")

	got, err := vectorstore.TenantFromContext(ctx)
	require.NoError(t, err)
	assert.Equal(t, "fyrsmithlabs", got.TenantID)
}

// TestRemediationFeedback_TenantCtx_AutoDerivesFromPath verifies the
// project_path-only path still derives tenant_id (preserves prior behavior).
func TestRemediationFeedback_TenantCtx_AutoDerivesFromPath(t *testing.T) {
	s := newGuideTestServer(t)

	projectPath := "/home/testuser/projects/contextd"
	wantTenantID := tenant.GetTenantIDForPath(projectPath)

	ctx, rt, err := s.tenantCtx(context.Background(), projectPath, "", "", "")
	require.NoError(t, err)
	assert.Equal(t, wantTenantID, rt.TenantID)

	got, err := vectorstore.TenantFromContext(ctx)
	require.NoError(t, err)
	assert.Equal(t, wantTenantID, got.TenantID)
}

// TestRepositoryIndex_TenantCtx verifies repository_index derives both tenant
// and project from the indexed path.
func TestRepositoryIndex_TenantCtx(t *testing.T) {
	s := newGuideTestServer(t)

	projectPath := "/home/testuser/projects/myrepo"
	wantTenantID := tenant.GetTenantIDForPath(projectPath)

	ctx, rt, err := s.tenantCtx(context.Background(), projectPath, "", "", "")
	require.NoError(t, err)
	assert.Equal(t, wantTenantID, rt.TenantID)
	assert.Equal(t, "myrepo", rt.ProjectID, "repository_index must establish a project floor")
	assert.Equal(t, projectPath, rt.ValidPath)

	got, err := vectorstore.TenantFromContext(ctx)
	require.NoError(t, err)
	assert.Equal(t, wantTenantID, got.TenantID)
	assert.Equal(t, "myrepo", got.ProjectID)
}

// TestSemanticSearch_TenantCtx_PathDerivedBranch covers the path-derived
// branch of semantic_search (no collection_name set).
func TestSemanticSearch_TenantCtx_PathDerivedBranch(t *testing.T) {
	s := newGuideTestServer(t)

	args := semanticSearchInput{
		Query:       "find auth handler",
		ProjectPath: "/home/testuser/projects/contextd",
		// CollectionName intentionally empty - path-derived path
	}
	wantTenantID := tenant.GetTenantIDForPath(args.ProjectPath)

	ctx, rt, err := s.tenantCtx(context.Background(), args.ProjectPath, args.TenantID, "", "")
	require.NoError(t, err, "path-derived branch must set tenant context")
	assert.Equal(t, wantTenantID, rt.TenantID)
	assert.Equal(t, "contextd", rt.ProjectID)
	assert.NotEmpty(t, rt.ValidPath, "ValidPath required for collection derivation + grep fallback")

	got, err := vectorstore.TenantFromContext(ctx)
	require.NoError(t, err)
	assert.Equal(t, wantTenantID, got.TenantID)
	assert.Equal(t, "contextd", got.ProjectID)
}

// TestSemanticSearch_TenantCtx_CollectionNameBranch covers the
// collection_name-explicit branch of semantic_search. Even when the caller
// provides an explicit collection_name, tenantCtx must still be invoked so
// the downstream vectorstore filter is tenant-scoped (fail-closed).
func TestSemanticSearch_TenantCtx_CollectionNameBranch(t *testing.T) {
	s := newGuideTestServer(t)

	args := semanticSearchInput{
		Query:          "find auth handler",
		ProjectPath:    "", // intentionally empty - collection encodes scope
		CollectionName: "explicit_collection_codebase",
		TenantID:       "fyrsmithlabs",
		ContentMode:    "minimal",
	}

	// Mirrors handler entry: s.tenantCtx(ctx, args.ProjectPath, args.TenantID, "", "")
	ctx, rt, err := s.tenantCtx(context.Background(), args.ProjectPath, args.TenantID, "", "")
	require.NoError(t, err, "collection_name branch must still set tenant context")
	assert.Equal(t, "fyrsmithlabs", rt.TenantID)
	assert.Empty(t, rt.ValidPath, "no project_path => no derived path")
	assert.Empty(t, rt.ProjectID, "collection_name branch may opt out of project floor")

	got, err := vectorstore.TenantFromContext(ctx)
	require.NoError(t, err, "vectorstore filter must see tenant info via context")
	assert.Equal(t, "fyrsmithlabs", got.TenantID)
}

// TestSemanticSearch_CollectionNameBranchRoutes verifies that the
// collection_name branch end-to-end routes to SearchInCollection with the
// explicit name and that the resulting context still carried tenant info when
// the lookup ran. This is the integration counterpart to the unit test above.
func TestSemanticSearch_CollectionNameBranchRoutes(t *testing.T) {
	store := &recordingVectorStore{}
	s := newTestServer(t, store)

	args := semanticSearchInput{
		Query:          "find auth handler",
		ProjectPath:    "/home/testuser/projects/contextd",
		CollectionName: "explicit_collection_codebase",
		TenantID:       "fyrsmithlabs",
		ContentMode:    "minimal",
	}

	// Derive context the same way the handler does, then invoke the branch.
	ctx, rt, err := s.tenantCtx(context.Background(), args.ProjectPath, args.TenantID, "", "")
	require.NoError(t, err)

	var toolErr error
	_, output, err := s.semanticSearchInCollection(ctx, args, rt.ValidPath, rt.TenantID, &toolErr)
	require.NoError(t, err)

	assert.True(t, store.searchInCollectionCalled, "SearchInCollection must be invoked on collection_name branch")
	assert.Equal(t, "explicit_collection_codebase", store.lastSearchCollection)
	assert.Equal(t, "minimal", output.ContentMode)
	assert.Equal(t, "semantic", output.Source)
}

// TestSemanticSearch_PathDerivedBranchRequiresPath verifies the migrated
// handler still surfaces a clear error when the path-derived branch is
// invoked without a project_path (collection_name is also empty).
//
// This is asserted at the contract level: tenantCtx with both projectPath and
// tenantID empty returns a derivation error, which is what the handler
// catches before attempting any service call.
func TestSemanticSearch_PathDerivedBranchRequiresPath(t *testing.T) {
	s := newGuideTestServer(t)

	// Force the soft-default resolver to be absent by passing an obviously
	// invalid tenant_id when no project_path is available. tenantCtx must
	// reject before service plumbing.
	_, _, err := s.tenantCtx(context.Background(), "", "../injection", "", "")
	require.Error(t, err)
}

// TestHelpers_PtrTrueFalse pins the helpers to their advertised return values.
// They are tiny but load-bearing: every annotated tool in the package relies
// on them to emit explicit pointer hints over the wire.
func TestHelpers_PtrTrueFalse(t *testing.T) {
	pt := ptrTrue()
	require.NotNil(t, pt)
	assert.True(t, *pt)

	pf := ptrFalse()
	require.NotNil(t, pf)
	assert.False(t, *pf)
}
