package mcp

import (
	"context"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"
)

func newPromptTestServer(t *testing.T) *Server {
	t.Helper()
	s := &Server{
		mcp:    mcp.NewServer(&mcp.Implementation{Name: "contextd", Version: "test"}, nil),
		logger: zap.NewNop(),
	}
	s.registerPrompts()
	return s
}

func connectPromptSession(t *testing.T, s *Server) *mcp.ClientSession {
	t.Helper()
	ctx := context.Background()

	serverTransport, clientTransport := mcp.NewInMemoryTransports()

	serverSession, err := s.mcp.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatalf("server connect: %v", err)
	}
	t.Cleanup(func() { _ = serverSession.Close() })

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "test"}, nil)
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	t.Cleanup(func() { _ = clientSession.Close() })

	return clientSession
}

func TestRegisterPromptsListsAllSix(t *testing.T) {
	s := newPromptTestServer(t)
	session := connectPromptSession(t, s)
	ctx := context.Background()

	res, err := session.ListPrompts(ctx, nil)
	if err != nil {
		t.Fatalf("ListPrompts: %v", err)
	}

	got := make(map[string]bool, len(res.Prompts))
	for _, p := range res.Prompts {
		got[p.Name] = true
	}

	want := []string{
		"contextd_checkpoint",
		"contextd_remember",
		"contextd_diagnose",
		"contextd_resume",
		"contextd_status",
		"contextd_search",
	}
	for _, name := range want {
		if !got[name] {
			t.Errorf("expected prompt %q to be registered, got %v", name, got)
		}
	}
}

func TestGetSearchPromptWeavesArgument(t *testing.T) {
	s := newPromptTestServer(t)
	session := connectPromptSession(t, s)
	ctx := context.Background()

	res, err := session.GetPrompt(ctx, &mcp.GetPromptParams{
		Name:      "contextd_search",
		Arguments: map[string]string{"query": "auth bug"},
	})
	if err != nil {
		t.Fatalf("GetPrompt: %v", err)
	}
	if len(res.Messages) == 0 {
		t.Fatal("expected at least one prompt message")
	}

	tc, ok := res.Messages[0].Content.(*mcp.TextContent)
	if !ok {
		t.Fatalf("expected *mcp.TextContent, got %T", res.Messages[0].Content)
	}
	text := tc.Text

	if !strings.Contains(text, "auth bug") {
		t.Errorf("expected message text to contain query %q, got: %s", "auth bug", text)
	}
	for _, tool := range []string{"memory_search", "remediation_search", "semantic_search"} {
		if !strings.Contains(text, tool) {
			t.Errorf("expected message text to mention tool %q, got: %s", tool, text)
		}
	}
}
