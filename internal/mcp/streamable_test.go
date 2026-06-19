package mcp

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"
)

// newBareServer builds a Server with only the fields the Streamable HTTP
// transport needs (mcp + logger). It avoids standing up the full service graph.
func newBareServer(t *testing.T) *Server {
	t.Helper()
	return &Server{
		mcp:    mcpsdk.NewServer(&mcpsdk.Implementation{Name: "contextd", Version: "test"}, nil),
		logger: zap.NewNop(),
	}
}

// TestStreamableHandler_Initialize verifies the Streamable HTTP transport
// completes an MCP initialize handshake and reports the contextd serverInfo.
func TestStreamableHandler_Initialize(t *testing.T) {
	srv := httptest.NewServer(newBareServer(t).StreamableHandler(true /* stateless */))
	defer srv.Close()

	body := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{` +
		`"protocolVersion":"2025-06-18","capabilities":{},` +
		`"clientInfo":{"name":"test-client","version":"1.0.0"}}}`

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, srv.URL, strings.NewReader(body))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	out := string(raw)
	if !strings.Contains(out, `"protocolVersion"`) {
		t.Errorf("response missing protocolVersion: %s", out)
	}
	if !strings.Contains(out, "contextd") {
		t.Errorf("response missing contextd serverInfo: %s", out)
	}
}

// TestStreamableHandler_NotNil is a guard that the handler constructor wires up.
func TestStreamableHandler_NotNil(t *testing.T) {
	if newBareServer(t).StreamableHandler(false) == nil {
		t.Fatal("StreamableHandler returned nil")
	}
}
