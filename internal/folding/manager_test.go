package folding

import (
	"context"
	"strings"
	"testing"
	"time"
)

// MockScrubber is a test implementation of SecretScrubber.
type MockScrubber struct {
	ScrubFunc func(content string) (string, error)
}

func (m *MockScrubber) Scrub(content string) (string, error) {
	if m.ScrubFunc != nil {
		return m.ScrubFunc(content)
	}
	// Default: just return content unchanged
	return content, nil
}

func newTestManager() (*BranchManager, *SimpleEventEmitter, *MemoryBranchRepository) {
	repo := NewMemoryBranchRepository()
	emitter := NewSimpleEventEmitter()
	budget := NewBudgetTracker(emitter)
	scrubber := &MockScrubber{}
	config := DefaultFoldingConfig()

	manager := NewBranchManager(repo, budget, scrubber, emitter, config)
	return manager, emitter, repo
}

func TestBranchManager_Create(t *testing.T) {
	manager, _, _ := newTestManager()
	ctx := context.Background()

	req := BranchRequest{
		SessionID:   "sess_001",
		Description: "Find database config",
		Prompt:      "Search for DB connection settings",
	}

	resp, err := manager.Create(ctx, req)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if !strings.HasPrefix(resp.BranchID, "br_") {
		t.Errorf("BranchID = %s, want prefix 'br_'", resp.BranchID)
	}
	if resp.BudgetAllocated != DefaultBudget {
		t.Errorf("BudgetAllocated = %d, want %d", resp.BudgetAllocated, DefaultBudget)
	}
	if resp.Depth != 0 {
		t.Errorf("Depth = %d, want 0 for first branch", resp.Depth)
	}
}

func TestBranchManager_CreateValidationError(t *testing.T) {
	manager, _, _ := newTestManager()
	ctx := context.Background()

	tests := []struct {
		name    string
		req     BranchRequest
		wantErr error
	}{
		{
			name:    "empty description",
			req:     BranchRequest{SessionID: "sess_001", Prompt: "test"},
			wantErr: ErrEmptyDescription,
		},
		{
			name:    "description too long",
			req:     BranchRequest{SessionID: "sess_001", Description: strings.Repeat("a", 501), Prompt: "test"},
			wantErr: ErrDescriptionTooLong,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := manager.Create(ctx, tt.req)
			if err != tt.wantErr {
				t.Errorf("Create() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestBranchManager_CreateMaxDepthExceeded(t *testing.T) {
	repo := NewMemoryBranchRepository()
	emitter := NewSimpleEventEmitter()
	budget := NewBudgetTracker(emitter)
	config := DefaultFoldingConfig()
	config.MaxDepth = 2 // Allow only depth 0 and 1

	manager := NewBranchManager(repo, budget, &MockScrubber{}, emitter, config)
	ctx := context.Background()

	// Create depth 0
	_, err := manager.Create(ctx, BranchRequest{SessionID: "sess_001", Description: "d0", Prompt: "p0"})
	if err != nil {
		t.Fatalf("Create depth 0 error = %v", err)
	}

	// Create depth 1
	_, err = manager.Create(ctx, BranchRequest{SessionID: "sess_001", Description: "d1", Prompt: "p1"})
	if err != nil {
		t.Fatalf("Create depth 1 error = %v", err)
	}

	// Try depth 2 - should fail
	_, err = manager.Create(ctx, BranchRequest{SessionID: "sess_001", Description: "d2", Prompt: "p2"})
	if err != ErrMaxDepthExceeded {
		t.Errorf("Create depth 2 error = %v, want ErrMaxDepthExceeded", err)
	}
}

func TestBranchManager_CreateMaxConcurrentExceeded(t *testing.T) {
	repo := NewMemoryBranchRepository()
	emitter := NewSimpleEventEmitter()
	budget := NewBudgetTracker(emitter)
	config := DefaultFoldingConfig()
	config.MaxConcurrentPerSession = 2

	manager := NewBranchManager(repo, budget, &MockScrubber{}, emitter, config)
	ctx := context.Background()

	// Create 2 branches (complete first to allow second at depth 0)
	resp1, _ := manager.Create(ctx, BranchRequest{SessionID: "sess_001", Description: "d1", Prompt: "p1"})
	manager.Return(ctx, ReturnRequest{BranchID: resp1.BranchID, Message: "done"})

	manager.Create(ctx, BranchRequest{SessionID: "sess_001", Description: "d2", Prompt: "p2"})
	manager.Create(ctx, BranchRequest{SessionID: "sess_001", Description: "d3", Prompt: "p3"})

	// Third active should fail
	_, err := manager.Create(ctx, BranchRequest{SessionID: "sess_001", Description: "d4", Prompt: "p4"})
	if err != ErrMaxConcurrentBranches {
		t.Errorf("Create error = %v, want ErrMaxConcurrentBranches", err)
	}
}

func TestBranchManager_Return(t *testing.T) {
	manager, _, _ := newTestManager()
	ctx := context.Background()

	// Create branch
	createResp, _ := manager.Create(ctx, BranchRequest{SessionID: "sess_001", Description: "test", Prompt: "test"})

	// Return
	returnResp, err := manager.Return(ctx, ReturnRequest{BranchID: createResp.BranchID, Message: "found it"})
	if err != nil {
		t.Fatalf("Return() error = %v", err)
	}

	if !returnResp.Success {
		t.Error("Return() Success = false, want true")
	}
	if returnResp.ScrubbedMsg != "found it" {
		t.Errorf("ScrubbedMsg = %q, want %q", returnResp.ScrubbedMsg, "found it")
	}

	// Verify branch is completed
	branch, _ := manager.Get(ctx, createResp.BranchID)
	if branch.Status != BranchStatusCompleted {
		t.Errorf("Branch status = %s, want completed", branch.Status)
	}
}

func TestBranchManager_ReturnSecretScrubbing(t *testing.T) {
	repo := NewMemoryBranchRepository()
	emitter := NewSimpleEventEmitter()
	budget := NewBudgetTracker(emitter)

	// Scrubber that redacts secrets
	scrubber := &MockScrubber{
		ScrubFunc: func(content string) (string, error) {
			return strings.ReplaceAll(content, "AKIAIOSFODNN7EXAMPLE", "[REDACTED:aws_key]"), nil
		},
	}

	manager := NewBranchManager(repo, budget, scrubber, emitter, nil)
	ctx := context.Background()

	// Create branch
	createResp, _ := manager.Create(ctx, BranchRequest{SessionID: "sess_001", Description: "test", Prompt: "test"})

	// Return with secret
	returnResp, err := manager.Return(ctx, ReturnRequest{
		BranchID: createResp.BranchID,
		Message:  "Found config with key AKIAIOSFODNN7EXAMPLE",
	})
	if err != nil {
		t.Fatalf("Return() error = %v", err)
	}

	// Verify secret was scrubbed
	if strings.Contains(returnResp.ScrubbedMsg, "AKIAIOSFODNN7EXAMPLE") {
		t.Error("Return() message contains secret, should be scrubbed")
	}
	if !strings.Contains(returnResp.ScrubbedMsg, "[REDACTED:aws_key]") {
		t.Errorf("ScrubbedMsg = %q, want redaction placeholder", returnResp.ScrubbedMsg)
	}
}

func TestBranchManager_ReturnScrubberFailure(t *testing.T) {
	repo := NewMemoryBranchRepository()
	emitter := NewSimpleEventEmitter()
	budget := NewBudgetTracker(emitter)

	// Scrubber that always fails
	scrubber := &MockScrubber{
		ScrubFunc: func(content string) (string, error) {
			return "", ErrScrubbingFailed
		},
	}

	manager := NewBranchManager(repo, budget, scrubber, emitter, nil)
	ctx := context.Background()

	// Create branch
	createResp, _ := manager.Create(ctx, BranchRequest{SessionID: "sess_001", Description: "test", Prompt: "test"})

	// Return - should fail closed (not leak unscrubbed content)
	_, err := manager.Return(ctx, ReturnRequest{BranchID: createResp.BranchID, Message: "secret content"})
	if err != ErrScrubbingFailed {
		t.Errorf("Return() error = %v, want ErrScrubbingFailed", err)
	}
}

func TestBranchManager_ReturnWithActiveChildren(t *testing.T) {
	repo := NewMemoryBranchRepository()
	emitter := NewSimpleEventEmitter()
	budget := NewBudgetTracker(emitter)
	config := DefaultFoldingConfig()
	config.MaxDepth = 5

	manager := NewBranchManager(repo, budget, &MockScrubber{}, emitter, config)
	ctx := context.Background()

	// Create parent
	parentResp, _ := manager.Create(ctx, BranchRequest{SessionID: "sess_001", Description: "parent", Prompt: "test"})

	// Create child
	childResp, _ := manager.Create(ctx, BranchRequest{SessionID: "sess_001", Description: "child", Prompt: "test"})

	// Return parent (should force-return child first)
	_, err := manager.Return(ctx, ReturnRequest{BranchID: parentResp.BranchID, Message: "done"})
	if err != nil {
		t.Fatalf("Return() error = %v", err)
	}

	// Verify child was force-returned
	child, _ := manager.Get(ctx, childResp.BranchID)
	if child.Status != BranchStatusFailed {
		t.Errorf("Child status = %s, want failed", child.Status)
	}
	if child.Error == nil || *child.Error != "parent returning" {
		t.Errorf("Child error = %v, want 'parent returning'", child.Error)
	}
}

func TestBranchManager_ReturnFromDepth0Allowed(t *testing.T) {
	// Branches at depth 0 (first branch in session) should be returnable
	// They return to the session context, not to a parent branch
	manager, _, _ := newTestManager()
	ctx := context.Background()

	// Create first branch (depth 0, no parent)
	createResp, _ := manager.Create(ctx, BranchRequest{SessionID: "sess_001", Description: "test", Prompt: "test"})

	// Return should succeed
	_, err := manager.Return(ctx, ReturnRequest{BranchID: createResp.BranchID, Message: "done"})
	if err != nil {
		t.Errorf("Return from depth 0 error = %v, want nil", err)
	}
}

func TestBranchManager_ReturnNotActive(t *testing.T) {
	manager, _, _ := newTestManager()
	ctx := context.Background()

	// Create and complete branch
	createResp, _ := manager.Create(ctx, BranchRequest{SessionID: "sess_001", Description: "test", Prompt: "test"})
	manager.Return(ctx, ReturnRequest{BranchID: createResp.BranchID, Message: "done"})

	// Try to return again
	_, err := manager.Return(ctx, ReturnRequest{BranchID: createResp.BranchID, Message: "again"})
	if err != ErrBranchNotActive {
		t.Errorf("Return() error = %v, want ErrBranchNotActive", err)
	}
}

func TestBranchManager_ForceReturn(t *testing.T) {
	manager, _, _ := newTestManager()
	ctx := context.Background()

	// Create branch
	createResp, _ := manager.Create(ctx, BranchRequest{SessionID: "sess_001", Description: "test", Prompt: "test"})

	// Force return
	err := manager.ForceReturn(ctx, createResp.BranchID, "test reason")
	if err != nil {
		t.Fatalf("ForceReturn() error = %v", err)
	}

	// Verify state
	branch, _ := manager.Get(ctx, createResp.BranchID)
	if branch.Status != BranchStatusFailed {
		t.Errorf("Status = %s, want failed", branch.Status)
	}
	if branch.Error == nil || *branch.Error != "test reason" {
		t.Errorf("Error = %v, want 'test reason'", branch.Error)
	}
}

func TestBranchManager_ForceReturnTimeout(t *testing.T) {
	manager, _, _ := newTestManager()
	ctx := context.Background()

	// Create branch
	createResp, _ := manager.Create(ctx, BranchRequest{SessionID: "sess_001", Description: "test", Prompt: "test"})

	// Force return with timeout reason
	err := manager.ForceReturn(ctx, createResp.BranchID, "timeout exceeded")
	if err != nil {
		t.Fatalf("ForceReturn() error = %v", err)
	}

	// Verify status is timeout (not failed)
	branch, _ := manager.Get(ctx, createResp.BranchID)
	if branch.Status != BranchStatusTimeout {
		t.Errorf("Status = %s, want timeout", branch.Status)
	}
}

func TestBranchManager_ForceReturnIdempotent(t *testing.T) {
	manager, _, _ := newTestManager()
	ctx := context.Background()

	// Create and complete branch
	createResp, _ := manager.Create(ctx, BranchRequest{SessionID: "sess_001", Description: "test", Prompt: "test"})
	manager.Return(ctx, ReturnRequest{BranchID: createResp.BranchID, Message: "done"})

	// Force return on completed branch (should be no-op)
	err := manager.ForceReturn(ctx, createResp.BranchID, "test")
	if err != nil {
		t.Errorf("ForceReturn() on completed branch error = %v, want nil", err)
	}

	// Status should still be completed
	branch, _ := manager.Get(ctx, createResp.BranchID)
	if branch.Status != BranchStatusCompleted {
		t.Errorf("Status = %s, want completed", branch.Status)
	}
}

func TestBranchManager_CleanupSession(t *testing.T) {
	repo := NewMemoryBranchRepository()
	emitter := NewSimpleEventEmitter()
	budget := NewBudgetTracker(emitter)
	config := DefaultFoldingConfig()
	config.MaxDepth = 5

	manager := NewBranchManager(repo, budget, &MockScrubber{}, emitter, config)
	ctx := context.Background()

	// Create multiple branches at different depths
	manager.Create(ctx, BranchRequest{SessionID: "sess_001", Description: "d0", Prompt: "p0"}) // depth 0
	manager.Create(ctx, BranchRequest{SessionID: "sess_001", Description: "d1", Prompt: "p1"}) // depth 1
	manager.Create(ctx, BranchRequest{SessionID: "sess_001", Description: "d2", Prompt: "p2"}) // depth 2

	// Cleanup session
	err := manager.CleanupSession(ctx, "sess_001")
	if err != nil {
		t.Fatalf("CleanupSession() error = %v", err)
	}

	// All branches should be terminal
	branches, _ := manager.ListBySession(ctx, "sess_001")
	for _, b := range branches {
		if !b.Status.IsTerminal() {
			t.Errorf("Branch %s status = %s, want terminal", b.ID, b.Status)
		}
	}
}

func TestBranchManager_BudgetExhaustedEvent(t *testing.T) {
	repo := NewMemoryBranchRepository()
	emitter := NewSimpleEventEmitter()
	budget := NewBudgetTracker(emitter)
	config := DefaultFoldingConfig()

	manager := NewBranchManager(repo, budget, &MockScrubber{}, emitter, config)
	ctx := context.Background()

	// Create branch with small budget
	createResp, _ := manager.Create(ctx, BranchRequest{
		SessionID:   "sess_001",
		Description: "test",
		Prompt:      "test",
		Budget:      100,
	})

	// Consume more than budget (triggers event)
	manager.ConsumeTokens(ctx, createResp.BranchID, 150)

	// Give event handler time to process
	time.Sleep(100 * time.Millisecond)

	// Branch should be force-returned
	branch, _ := manager.Get(ctx, createResp.BranchID)
	if branch.Status != BranchStatusFailed {
		t.Errorf("Status = %s, want failed (from budget exhaustion)", branch.Status)
	}
}

func TestBranchManager_TimeoutEvent(t *testing.T) {
	repo := NewMemoryBranchRepository()
	emitter := NewSimpleEventEmitter()
	budget := NewBudgetTracker(emitter)
	config := DefaultFoldingConfig()

	manager := NewBranchManager(repo, budget, &MockScrubber{}, emitter, config)
	ctx := context.Background()

	// Create branch with very short timeout
	createResp, _ := manager.Create(ctx, BranchRequest{
		SessionID:      "sess_001",
		Description:    "test",
		Prompt:         "test",
		TimeoutSeconds: 1, // 1 second timeout
	})

	// Wait for timeout
	time.Sleep(1500 * time.Millisecond)

	// Branch should be force-returned due to timeout
	branch, _ := manager.Get(ctx, createResp.BranchID)
	if branch.Status != BranchStatusTimeout {
		t.Errorf("Status = %s, want timeout", branch.Status)
	}
}

func TestBranchManager_GetActive(t *testing.T) {
	manager, _, _ := newTestManager()
	ctx := context.Background()

	// No active branch initially
	active, err := manager.GetActive(ctx, "sess_001")
	if err != nil {
		t.Fatalf("GetActive() error = %v", err)
	}
	if active != nil {
		t.Errorf("GetActive() = %v, want nil initially", active)
	}

	// Create branch
	manager.Create(ctx, BranchRequest{SessionID: "sess_001", Description: "test", Prompt: "test"})

	// Now there should be an active branch
	active, err = manager.GetActive(ctx, "sess_001")
	if err != nil {
		t.Fatalf("GetActive() error = %v", err)
	}
	if active == nil {
		t.Error("GetActive() = nil, want branch")
	}
}
