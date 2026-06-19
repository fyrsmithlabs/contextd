package mcp

import (
	"context"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/fyrsmithlabs/contextd/internal/checkpoint"
	"github.com/fyrsmithlabs/contextd/internal/secrets"
)

// fakeCheckpointSvc is a minimal checkpoint.Service for testing checkpoint
// resolution. Only List is meaningful; the rest satisfy the interface.
type fakeCheckpointSvc struct {
	list    []*checkpoint.Checkpoint
	listErr error
}

func (f *fakeCheckpointSvc) Save(context.Context, *checkpoint.SaveRequest) (*checkpoint.Checkpoint, error) {
	return nil, nil
}
func (f *fakeCheckpointSvc) List(context.Context, *checkpoint.ListRequest) ([]*checkpoint.Checkpoint, error) {
	return f.list, f.listErr
}
func (f *fakeCheckpointSvc) Resume(context.Context, *checkpoint.ResumeRequest) (*checkpoint.ResumeResponse, error) {
	return nil, nil
}
func (f *fakeCheckpointSvc) Get(context.Context, string, string, string, string) (*checkpoint.Checkpoint, error) {
	return nil, nil
}
func (f *fakeCheckpointSvc) Delete(context.Context, string, string, string, string) error {
	return nil
}
func (f *fakeCheckpointSvc) Close() error { return nil }

func newCheckpointTestServer(f *fakeCheckpointSvc) *Server {
	return &Server{
		checkpointSvc: f,
		scrubber:      &secrets.NoopScrubber{},
		logger:        zap.NewNop(),
	}
}

func TestClientLog_NilSessionNoPanic(t *testing.T) {
	s := newCheckpointTestServer(&fakeCheckpointSvc{})
	// Must not panic with a nil session.
	s.clientLog(context.Background(), nil, "info", "hello")
}

func TestChooseCheckpoint_NoneIsError(t *testing.T) {
	s := newCheckpointTestServer(&fakeCheckpointSvc{list: nil})
	id, msg, err := s.chooseCheckpointViaElicit(context.Background(), nil, "tenant")
	if err == nil {
		t.Fatalf("expected error for zero checkpoints, got id=%q msg=%q", id, msg)
	}
}

func TestChooseCheckpoint_SingleAutoSelects(t *testing.T) {
	s := newCheckpointTestServer(&fakeCheckpointSvc{list: []*checkpoint.Checkpoint{
		{ID: "cp-1", Summary: "only one", CreatedAt: time.Now()},
	}})
	id, msg, err := s.chooseCheckpointViaElicit(context.Background(), nil, "tenant")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "cp-1" {
		t.Errorf("id = %q, want cp-1", id)
	}
	if msg != "" {
		t.Errorf("msg = %q, want empty", msg)
	}
}

func TestChooseCheckpoint_MultipleNoSessionReturnsList(t *testing.T) {
	s := newCheckpointTestServer(&fakeCheckpointSvc{list: []*checkpoint.Checkpoint{
		{ID: "cp-1", Summary: "first", CreatedAt: time.Now()},
		{ID: "cp-2", Summary: "second", CreatedAt: time.Now()},
	}})
	// nil session => cannot elicit => fall back to a list message.
	id, msg, err := s.chooseCheckpointViaElicit(context.Background(), nil, "tenant")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "" {
		t.Errorf("id = %q, want empty (manual selection)", id)
	}
	if !strings.Contains(msg, "cp-1") || !strings.Contains(msg, "cp-2") {
		t.Errorf("list message missing checkpoint ids: %q", msg)
	}
}
