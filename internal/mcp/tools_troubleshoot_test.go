package mcp

import (
	"context"
	"encoding/json"
	"testing"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTroubleshootDiagnose_Smoke exercises the full MCP plumbing for
// troubleshoot_diagnose and verifies the typed output is populated.
//
// The mockTroubleshootStore returns no patterns and the service has no AI
// client configured, so we expect a low-confidence diagnosis with empty
// hypothesis and recommendation slices (initialized to non-nil for JSON).
func TestTroubleshootDiagnose_Smoke(t *testing.T) {
	srv := newMcpTestServer(t, nil)
	cs := connectClient(t, srv)
	ctx := context.Background()

	res, err := cs.CallTool(ctx, &sdkmcp.CallToolParams{
		Name: "troubleshoot_diagnose",
		Arguments: map[string]any{
			"error_message": "connection refused on port 8080",
			"error_context": "after upgrading to v2",
		},
	})
	require.NoError(t, err)
	require.False(t, res.IsError, "tool reported an error: %+v", res)

	require.NotNil(t, res.StructuredContent)
	raw, err := json.Marshal(res.StructuredContent)
	require.NoError(t, err)
	var out troubleshootDiagnoseOutput
	require.NoError(t, json.Unmarshal(raw, &out))
	assert.Equal(t, "connection refused on port 8080", out.ErrorMessage)
	assert.NotNil(t, out.Hypotheses, "hypotheses should be initialized to a slice, not nil")
	assert.NotNil(t, out.Recommendations, "recommendations should be initialized to a slice")
}

// TestTroubleshootDiagnose_RejectsEmptyErrorMessage covers the malformed
// input guard introduced during the HANDLER-GUIDE migration.
func TestTroubleshootDiagnose_RejectsEmptyErrorMessage(t *testing.T) {
	srv := newMcpTestServer(t, nil)
	cs := connectClient(t, srv)
	ctx := context.Background()

	res, err := cs.CallTool(ctx, &sdkmcp.CallToolParams{
		Name:      "troubleshoot_diagnose",
		Arguments: map[string]any{},
	})
	if err == nil {
		require.NotNil(t, res)
		assert.True(t, res.IsError, "missing error_message must be rejected")
	}
}
