package mcp

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"

	"github.com/fyrsmithlabs/contextd/internal/secrets"
)

func newResourceTestServer(t *testing.T) *Server {
	t.Helper()
	s := &Server{
		mcp:      mcp.NewServer(&mcp.Implementation{Name: "contextd", Version: "test"}, nil),
		scrubber: &secrets.NoopScrubber{},
		logger:   zap.NewNop(),
		// Services intentionally left nil: passing tests exercise registration,
		// help, and template listing only — they never invoke a backing service.
	}
	s.registerResources()
	return s
}

func connectResourceSession(t *testing.T, s *Server) *mcp.ClientSession {
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

func TestRegisterResourcesDoesNotPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("registerResources panicked: %v", r)
		}
	}()
	_ = newResourceTestServer(t)
}

func TestRegisterResourcesListsHelp(t *testing.T) {
	s := newResourceTestServer(t)
	session := connectResourceSession(t, s)
	ctx := context.Background()

	res, err := session.ListResources(ctx, nil)
	if err != nil {
		t.Fatalf("ListResources: %v", err)
	}

	found := false
	for _, r := range res.Resources {
		if r.URI == "contextd://help" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected static resource contextd://help to be registered, got %+v", res.Resources)
	}
}

func TestRegisterResourcesListsTemplates(t *testing.T) {
	s := newResourceTestServer(t)
	session := connectResourceSession(t, s)
	ctx := context.Background()

	res, err := session.ListResourceTemplates(ctx, nil)
	if err != nil {
		t.Fatalf("ListResourceTemplates: %v", err)
	}

	got := make(map[string]bool, len(res.ResourceTemplates))
	for _, tmpl := range res.ResourceTemplates {
		got[tmpl.URITemplate] = true
	}

	want := []string{
		"contextd://{project_id}/memories",
		"contextd://{project_id}/memory/{id}",
		"contextd://{project_id}/checkpoints",
		"contextd://{project_id}/checkpoint/{id}",
		"contextd://{project_id}/remediation/{id}",
		"contextd://{project_id}/remediations{?query}",
	}
	for _, uri := range want {
		if !got[uri] {
			t.Errorf("expected resource template %q to be registered, got %v", uri, got)
		}
	}
}

func TestReadHelpResourceReturnsJSON(t *testing.T) {
	s := newResourceTestServer(t)
	session := connectResourceSession(t, s)
	ctx := context.Background()

	res, err := session.ReadResource(ctx, &mcp.ReadResourceParams{URI: "contextd://help"})
	if err != nil {
		t.Fatalf("ReadResource(help): %v", err)
	}
	if len(res.Contents) != 1 {
		t.Fatalf("expected 1 content, got %d", len(res.Contents))
	}
	c := res.Contents[0]
	if c.URI != "contextd://help" {
		t.Errorf("expected content URI contextd://help, got %q", c.URI)
	}
	if c.MIMEType != "application/json" {
		t.Errorf("expected MIME type application/json, got %q", c.MIMEType)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(c.Text), &parsed); err != nil {
		t.Fatalf("help content is not valid JSON: %v\ntext: %s", err, c.Text)
	}
	if parsed["scheme"] != "contextd" {
		t.Errorf("expected scheme \"contextd\" in help JSON, got %v", parsed["scheme"])
	}
	if _, ok := parsed["resources"]; !ok {
		t.Errorf("expected \"resources\" key in help JSON, got %v", parsed)
	}
}

func TestParseResourceURI(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		wantErr bool
		project string
		kind    string
		id      string
		query   string
	}{
		{name: "memories collection", raw: "contextd://proj/memories", project: "proj", kind: "memories"},
		{name: "single memory", raw: "contextd://proj/memory/abc123", project: "proj", kind: "memory", id: "abc123"},
		{name: "remediations with query", raw: "contextd://proj/remediations?query=null+pointer", project: "proj", kind: "remediations", query: "null pointer"},
		{name: "wrong scheme", raw: "http://proj/memories", wantErr: true},
		{name: "missing kind", raw: "contextd://proj", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseResourceURI(tt.raw)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error for %q, got none", tt.raw)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.projectID != tt.project || got.kind != tt.kind || got.id != tt.id || got.query != tt.query {
				t.Errorf("parseResourceURI(%q) = %+v, want project=%q kind=%q id=%q query=%q",
					tt.raw, got, tt.project, tt.kind, tt.id, tt.query)
			}
		})
	}
}

func TestResourceURIBuilders(t *testing.T) {
	if got := collectionResourceURI("proj", "memories"); got != "contextd://proj/memories" {
		t.Errorf("collectionResourceURI = %q", got)
	}
	if got := itemResourceURI("proj", "memory", "abc"); got != "contextd://proj/memory/abc" {
		t.Errorf("itemResourceURI = %q", got)
	}
}
