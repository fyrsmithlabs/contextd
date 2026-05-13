package mcp

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/fyrsmithlabs/contextd/internal/checkpoint"
	"github.com/fyrsmithlabs/contextd/internal/conversation"
	"github.com/fyrsmithlabs/contextd/internal/reasoningbank"
	"github.com/fyrsmithlabs/contextd/internal/remediation"
	"github.com/fyrsmithlabs/contextd/internal/repository"
	"github.com/fyrsmithlabs/contextd/internal/secrets"
	"github.com/fyrsmithlabs/contextd/internal/troubleshoot"
	"github.com/fyrsmithlabs/contextd/internal/vectorstore"
)

// capturingConversationService records the context passed into Index/Search
// so tests can assert tenant info has been propagated. It also lets tests
// drive controlled responses back through the MCP handler.
type capturingConversationService struct {
	lastIndexCtx  context.Context
	lastSearchCtx context.Context
	lastIndexOpts conversation.IndexOptions
	lastSearchOps conversation.SearchOptions

	indexResult  *conversation.IndexResult
	searchResult *conversation.SearchResult
	indexErr     error
	searchErr    error
}

func (c *capturingConversationService) Index(ctx context.Context, opts conversation.IndexOptions) (*conversation.IndexResult, error) {
	c.lastIndexCtx = ctx
	c.lastIndexOpts = opts
	if c.indexErr != nil {
		return nil, c.indexErr
	}
	if c.indexResult != nil {
		return c.indexResult, nil
	}
	return &conversation.IndexResult{
		SessionsIndexed:    1,
		MessagesIndexed:    3,
		DecisionsExtracted: 0,
		FilesReferenced:    []string{"a.go"},
	}, nil
}

func (c *capturingConversationService) Search(ctx context.Context, opts conversation.SearchOptions) (*conversation.SearchResult, error) {
	c.lastSearchCtx = ctx
	c.lastSearchOps = opts
	if c.searchErr != nil {
		return nil, c.searchErr
	}
	if c.searchResult != nil {
		return c.searchResult, nil
	}
	return &conversation.SearchResult{
		Query:   opts.Query,
		Results: nil,
		Total:   0,
		Took:    time.Millisecond,
	}, nil
}

// newMcpTestServer wires a contextd Server with mock stores and an optional
// conversation service stub, ready to be connected over an in-memory MCP
// transport in tests.
func newMcpTestServer(t *testing.T, convSvc conversation.ConversationService) *Server {
	t.Helper()
	logger := zap.NewNop()

	troubleshootStore := &mockTroubleshootStore{}
	vectorStore := &mockVectorStore{}

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

	cfg := &Config{Name: "test-server", Version: "1.0.0", Logger: logger}
	srv, err := NewServer(cfg, checkpointSvc, remediationSvc, repositorySvc, troubleshootSvc, reasoningbankSvc, nil, nil, scrubber)
	require.NoError(t, err)

	if convSvc != nil {
		srv.SetConversationService(convSvc)
		// Re-register conversation tools now that the service is set;
		// NewServer registered tools before SetConversationService was called.
		srv.registerConversationTools()
	}

	t.Cleanup(func() { _ = srv.Close() })
	return srv
}

// connectClient pairs an in-memory client to the test server and returns the
// connected client session. The session is closed automatically on cleanup.
func connectClient(t *testing.T, srv *Server) *sdkmcp.ClientSession {
	t.Helper()
	ctx := context.Background()
	clientTransport, serverTransport := sdkmcp.NewInMemoryTransports()

	_, err := srv.mcp.Connect(ctx, serverTransport, nil)
	require.NoError(t, err)

	client := sdkmcp.NewClient(&sdkmcp.Implementation{Name: "test-client", Version: "1.0.0"}, nil)
	cs, err := client.Connect(ctx, clientTransport, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = cs.Close() })
	return cs
}

// TestConversationIndex_TenantContext verifies that conversation_index plumbs
// tenant info onto the context handed to the service via tenantCtx.
func TestConversationIndex_TenantContext(t *testing.T) {
	cap := &capturingConversationService{}
	srv := newMcpTestServer(t, cap)
	cs := connectClient(t, srv)
	ctx := context.Background()

	tmpDir := t.TempDir()

	res, err := cs.CallTool(ctx, &sdkmcp.CallToolParams{
		Name: "conversation_index",
		Arguments: map[string]any{
			"project_path": tmpDir,
			"tenant_id":    "test_tenant",
		},
	})
	require.NoError(t, err)
	require.False(t, res.IsError, "tool reported an error: %+v", res)

	// Service must have been invoked with a context bearing tenant info.
	require.NotNil(t, cap.lastIndexCtx, "service was not called")
	ti, err := vectorstore.TenantFromContext(cap.lastIndexCtx)
	require.NoError(t, err, "tenant info must be on context")
	assert.Equal(t, "test_tenant", ti.TenantID)
	assert.NotEmpty(t, ti.ProjectID, "project_id should be derived from project_path")

	// Resolved tenant must also flow into the service request.
	assert.Equal(t, "test_tenant", cap.lastIndexOpts.TenantID)
}

// TestConversationIndex_RejectsMissingProjectPath asserts the malformed-input
// guard fires before tenant derivation.
func TestConversationIndex_RejectsMissingProjectPath(t *testing.T) {
	cap := &capturingConversationService{}
	srv := newMcpTestServer(t, cap)
	cs := connectClient(t, srv)
	ctx := context.Background()

	res, err := cs.CallTool(ctx, &sdkmcp.CallToolParams{
		Name:      "conversation_index",
		Arguments: map[string]any{},
	})
	// SDK marshals handler errors into the JSON-RPC error channel.
	if err == nil {
		require.NotNil(t, res, "expected non-nil result when err is nil")
		assert.True(t, res.IsError, "missing project_path must surface as an error")
	}
	assert.Nil(t, cap.lastIndexCtx, "service must not be called for malformed input")
}

// TestConversationIndex_RejectsMalformedTenantID covers the sanitize.Validate*
// hardening path.
func TestConversationIndex_RejectsMalformedTenantID(t *testing.T) {
	cap := &capturingConversationService{}
	srv := newMcpTestServer(t, cap)
	cs := connectClient(t, srv)
	ctx := context.Background()

	res, err := cs.CallTool(ctx, &sdkmcp.CallToolParams{
		Name: "conversation_index",
		Arguments: map[string]any{
			"project_path": t.TempDir(),
			"tenant_id":    "../../etc/passwd",
		},
	})
	if err == nil {
		require.NotNil(t, res)
		assert.True(t, res.IsError, "malformed tenant_id must be rejected")
	}
	assert.Nil(t, cap.lastIndexCtx, "service must not be called when tenant_id is malformed")
}

// TestConversationIndex_RejectsEnableLLM ensures the explicit guard against
// the not-yet-implemented LLM extraction path stays in place.
func TestConversationIndex_RejectsEnableLLM(t *testing.T) {
	cap := &capturingConversationService{}
	srv := newMcpTestServer(t, cap)
	cs := connectClient(t, srv)
	ctx := context.Background()

	res, err := cs.CallTool(ctx, &sdkmcp.CallToolParams{
		Name: "conversation_index",
		Arguments: map[string]any{
			"project_path": t.TempDir(),
			"enable_llm":   true,
		},
	})
	if err == nil {
		require.NotNil(t, res)
		assert.True(t, res.IsError, "enable_llm=true must be rejected until implemented")
	}
	assert.Nil(t, cap.lastIndexCtx)
}

// TestConversationSearch_TenantContext verifies search threads tenant info
// onto the context and emits typed rows in the structured output.
func TestConversationSearch_TenantContext(t *testing.T) {
	cap := &capturingConversationService{
		searchResult: &conversation.SearchResult{
			Query: "deploy",
			Results: []conversation.SearchHit{
				{
					Document: conversation.ConversationDocument{
						ID:        "doc-1",
						SessionID: "sess-1",
						Type:      conversation.TypeMessage,
						Content:   "We decided to deploy to staging first.",
						Timestamp: time.Unix(1700000000, 0),
						Tags:      []string{"deploy"},
						Domain:    "kubernetes",
					},
					Score: 0.92,
				},
			},
			Total: 1,
			Took:  5 * time.Millisecond,
		},
	}
	srv := newMcpTestServer(t, cap)
	cs := connectClient(t, srv)
	ctx := context.Background()

	res, err := cs.CallTool(ctx, &sdkmcp.CallToolParams{
		Name: "conversation_search",
		Arguments: map[string]any{
			"project_path": t.TempDir(),
			"tenant_id":    "test_tenant",
			"query":        "deploy",
			"limit":        5,
		},
	})
	require.NoError(t, err)
	require.False(t, res.IsError, "tool reported an error: %+v", res)

	// Tenant must reach the service via context.
	require.NotNil(t, cap.lastSearchCtx)
	ti, err := vectorstore.TenantFromContext(cap.lastSearchCtx)
	require.NoError(t, err)
	assert.Equal(t, "test_tenant", ti.TenantID)

	// Structured output should be a typed conversationSearchOutput, not a
	// loose map. We round-trip through JSON to assert the typed shape.
	require.NotNil(t, res.StructuredContent, "structured content should be populated")
	raw, err := json.Marshal(res.StructuredContent)
	require.NoError(t, err)
	var out conversationSearchOutput
	require.NoError(t, json.Unmarshal(raw, &out))
	require.Len(t, out.Results, 1)
	row := out.Results[0]
	assert.Equal(t, "doc-1", row.ID)
	assert.Equal(t, "sess-1", row.SessionID)
	assert.Equal(t, "message", row.Type)
	assert.InDelta(t, 0.92, row.Score, 0.0001)
	assert.Equal(t, []string{"deploy"}, row.Tags)
	assert.Equal(t, "kubernetes", row.Domain)
}

// TestConversationSearch_RejectsMalformedProjectPath ensures path traversal
// attempts are blocked in tenantCtx.
func TestConversationSearch_RejectsMalformedProjectPath(t *testing.T) {
	cap := &capturingConversationService{}
	srv := newMcpTestServer(t, cap)
	cs := connectClient(t, srv)
	ctx := context.Background()

	res, err := cs.CallTool(ctx, &sdkmcp.CallToolParams{
		Name: "conversation_search",
		Arguments: map[string]any{
			"project_path": "../../etc",
			"query":        "anything",
		},
	})
	if err == nil {
		require.NotNil(t, res)
		assert.True(t, res.IsError, "path traversal must be rejected")
	}
	assert.Nil(t, cap.lastSearchCtx, "service must not be called for path traversal")
}

// TestConversationSearch_LimitCap enforces the §8.3 cap of 100.
func TestConversationSearch_LimitCap(t *testing.T) {
	cap := &capturingConversationService{}
	srv := newMcpTestServer(t, cap)
	cs := connectClient(t, srv)
	ctx := context.Background()

	res, err := cs.CallTool(ctx, &sdkmcp.CallToolParams{
		Name: "conversation_search",
		Arguments: map[string]any{
			"project_path": t.TempDir(),
			"tenant_id":    "test_tenant",
			"query":        "foo",
			"limit":        9999,
		},
	})
	require.NoError(t, err)
	require.False(t, res.IsError)
	assert.LessOrEqual(t, cap.lastSearchOps.Limit, 100, "limit must be capped at 100")
}
