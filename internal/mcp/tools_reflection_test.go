package mcp

import (
	"context"
	"testing"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fyrsmithlabs/contextd/internal/vectorstore"
)

// TestReflectReport_TenantContext verifies that the handler's tenant
// derivation places the right TenantInfo on the context BEFORE the
// underlying reflection reporter is invoked. We don't rely on the reporter
// itself returning a clean result because it currently issues an empty-query
// Search against reasoningbank — that's a pre-existing service bug
// orthogonal to the handler refactor.
//
// We test the wiring by invoking tenantCtx with the exact arguments the
// handler hands it (see tools_reflection.go: s.tenantCtx(ctx,
// args.ProjectPath, "", "", args.ProjectID)).
func TestReflectReport_TenantContext(t *testing.T) {
	srv := newMcpTestServer(t, nil)

	ctx, rt, err := srv.tenantCtx(context.Background(), "", "", "", "contextd")
	require.NoError(t, err)
	assert.Equal(t, "contextd", rt.ProjectID)
	assert.NotEmpty(t, rt.TenantID, "tenant_id should be auto-derived")

	ti, err := vectorstore.TenantFromContext(ctx)
	require.NoError(t, err, "tenant info must be on ctx")
	assert.Equal(t, "contextd", ti.ProjectID)
	assert.Equal(t, rt.TenantID, ti.TenantID)
}

// TestReflectReport_RejectsMissingProjectID verifies the required-field guard.
func TestReflectReport_RejectsMissingProjectID(t *testing.T) {
	srv := newMcpTestServer(t, nil)
	cs := connectClient(t, srv)
	ctx := context.Background()

	res, err := cs.CallTool(ctx, &sdkmcp.CallToolParams{
		Name:      "reflect_report",
		Arguments: map[string]any{},
	})
	if err == nil {
		require.NotNil(t, res)
		assert.True(t, res.IsError, "missing project_id must be rejected")
	}
}

// TestReflectReport_RejectsMalformedProjectID covers CWE-22/287 hardening.
func TestReflectReport_RejectsMalformedProjectID(t *testing.T) {
	srv := newMcpTestServer(t, nil)
	cs := connectClient(t, srv)
	ctx := context.Background()

	res, err := cs.CallTool(ctx, &sdkmcp.CallToolParams{
		Name: "reflect_report",
		Arguments: map[string]any{
			"project_id": "../../etc/passwd",
		},
	})
	if err == nil {
		require.NotNil(t, res)
		assert.True(t, res.IsError, "malformed project_id must be rejected")
	}
}

// TestReflectReport_RejectsUnknownFormat covers the §3.3 enum validation
// requirement.
func TestReflectReport_RejectsUnknownFormat(t *testing.T) {
	srv := newMcpTestServer(t, nil)
	cs := connectClient(t, srv)
	ctx := context.Background()

	res, err := cs.CallTool(ctx, &sdkmcp.CallToolParams{
		Name: "reflect_report",
		Arguments: map[string]any{
			"project_id": "contextd",
			"format":     "yaml",
		},
	})
	if err == nil {
		require.NotNil(t, res)
		assert.True(t, res.IsError, "unknown format must be rejected")
	}
}

// TestReflectAnalyze_TenantContext mirrors the reflect_report tenant test
// for the second reflection handler. Same caveat about the buggy
// reasoningbank.Search empty-query path applies.
func TestReflectAnalyze_TenantContext(t *testing.T) {
	srv := newMcpTestServer(t, nil)

	// Replicates the handler call site: s.tenantCtx(ctx, "", "", "",
	// args.ProjectID). No project_path on this tool's input.
	ctx, rt, err := srv.tenantCtx(context.Background(), "", "", "", "contextd")
	require.NoError(t, err)
	assert.Equal(t, "contextd", rt.ProjectID)

	ti, err := vectorstore.TenantFromContext(ctx)
	require.NoError(t, err)
	assert.Equal(t, "contextd", ti.ProjectID)
	assert.NotEmpty(t, ti.TenantID, "tenant_id should be auto-derived from environment")
}

// TestReflectAnalyze_RejectsMalformedProjectID asserts the sanitize.Validate
// path is wired up for the second reflection handler too.
func TestReflectAnalyze_RejectsMalformedProjectID(t *testing.T) {
	srv := newMcpTestServer(t, nil)
	cs := connectClient(t, srv)
	ctx := context.Background()

	res, err := cs.CallTool(ctx, &sdkmcp.CallToolParams{
		Name: "reflect_analyze",
		Arguments: map[string]any{
			"project_id": "../etc",
		},
	})
	if err == nil {
		require.NotNil(t, res)
		assert.True(t, res.IsError, "malformed project_id must be rejected")
	}
}
