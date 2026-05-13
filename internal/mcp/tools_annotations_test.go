package mcp

import (
	"context"
	"testing"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
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

// TestToolAnnotations verifies that the six handlers in the HANDLER-GUIDE
// polish wave (checkpoint_save, checkpoint_list, remediation_search,
// branch_create, branch_return, branch_status) report the expected
// ToolAnnotations on the wire per MCP 2025-06-18 §tools/list.
//
// We connect a fresh in-memory client/server pair so the assertions exercise
// the same code path real MCP clients see — protocol round-trip, not just
// the in-process Tool registry.
func TestToolAnnotations(t *testing.T) {
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

	foldingEmitter := folding.NewSimpleEventEmitter()
	foldingBudget := folding.NewBudgetTracker(foldingEmitter)
	foldingRepo := folding.NewMemoryBranchRepository()
	foldingScrubber := &testScrubberAdapter{scrubber: scrubber}
	foldingSvc := folding.NewBranchManager(
		foldingRepo,
		foldingBudget,
		foldingScrubber,
		foldingEmitter,
		folding.DefaultFoldingConfig(),
	)

	cfg := &Config{
		Name:    "test-annotations",
		Version: "1.0.0",
		Logger:  logger,
	}
	server, err := NewServer(cfg, checkpointSvc, remediationSvc, repositorySvc, troubleshootSvc, reasoningbankSvc, foldingSvc, nil, scrubber)
	require.NoError(t, err)
	defer server.Close()

	// Wire an in-memory client/server transport and call tools/list as a real
	// client would.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	serverT, clientT := sdkmcp.NewInMemoryTransports()
	sess, err := server.mcp.Connect(ctx, serverT, nil)
	require.NoError(t, err)
	defer sess.Close()

	client := sdkmcp.NewClient(&sdkmcp.Implementation{Name: "ann-test-client", Version: "0.0.1"}, nil)
	clientSess, err := client.Connect(ctx, clientT, nil)
	require.NoError(t, err)
	defer clientSess.Close()

	listed, err := clientSess.ListTools(ctx, nil)
	require.NoError(t, err)

	byName := map[string]*sdkmcp.Tool{}
	for _, tt := range listed.Tools {
		byName[tt.Name] = tt
	}

	// Expected annotations per HANDLER-GUIDE §2.1.
	cases := []struct {
		name            string
		readOnly        bool
		destructive     *bool
		idempotent      bool
		openWorld       *bool
		maxDescription  int
	}{
		// Append-only writes.
		{"checkpoint_save", false, ptrFalse(), false, ptrFalse(), 200},
		{"branch_create", false, ptrFalse(), false, ptrFalse(), 200},
		// Mutating write.
		{"branch_return", false, ptrTrue(), false, ptrFalse(), 200},
		// Pure reads.
		{"checkpoint_list", true, nil, false, ptrFalse(), 200},
		{"remediation_search", true, nil, false, ptrFalse(), 200},
		{"branch_status", true, nil, false, ptrFalse(), 200},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tool, ok := byName[tc.name]
			require.True(t, ok, "tool %q should be registered", tc.name)
			require.NotNil(t, tool.Annotations, "tool %q must have annotations", tc.name)

			ann := tool.Annotations
			assert.Equal(t, tc.readOnly, ann.ReadOnlyHint, "ReadOnlyHint")
			assert.Equal(t, tc.idempotent, ann.IdempotentHint, "IdempotentHint")
			assertBoolPtrEqual(t, tc.openWorld, ann.OpenWorldHint, "OpenWorldHint")
			if tc.destructive != nil {
				assertBoolPtrEqual(t, tc.destructive, ann.DestructiveHint, "DestructiveHint")
			}

			// Description budget (HANDLER-GUIDE §1.2).
			assert.LessOrEqual(t, len(tool.Description), tc.maxDescription, "Description exceeds %d chars: %q", tc.maxDescription, tool.Description)
			assert.NotEmpty(t, tool.Description, "Description must be non-empty")
		})
	}
}

// assertBoolPtrEqual compares two *bool values for equality, treating nil as a
// distinct value from a pointer to false. testify's assert.Equal already does
// this correctly, but using a helper makes the call sites read better.
func assertBoolPtrEqual(t *testing.T, want, got *bool, field string) {
	t.Helper()
	if want == nil {
		assert.Nil(t, got, "%s should be nil", field)
		return
	}
	require.NotNil(t, got, "%s should not be nil", field)
	assert.Equal(t, *want, *got, "%s", field)
}
