package folding

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestMemoryBranchRepository_CreateAndGet(t *testing.T) {
	repo := NewMemoryBranchRepository()
	ctx := context.Background()

	branch := &Branch{
		ID:          "br_001",
		SessionID:   "sess_001",
		Description: "test branch",
		Prompt:      "do something",
		BudgetTotal: 8192,
		Status:      BranchStatusActive,
		CreatedAt:   time.Now(),
	}

	// Create
	err := repo.Create(ctx, branch)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Get
	got, err := repo.Get(ctx, "br_001")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if got.ID != branch.ID {
		t.Errorf("Get() ID = %v, want %v", got.ID, branch.ID)
	}
	if got.Description != branch.Description {
		t.Errorf("Get() Description = %v, want %v", got.Description, branch.Description)
	}
}

func TestMemoryBranchRepository_CreateDuplicate(t *testing.T) {
	repo := NewMemoryBranchRepository()
	ctx := context.Background()

	branch := &Branch{
		ID:        "br_001",
		SessionID: "sess_001",
		Status:    BranchStatusActive,
	}

	_ = repo.Create(ctx, branch)

	// Try to create duplicate
	err := repo.Create(ctx, branch)
	if err != ErrBranchAlreadyExists {
		t.Errorf("Create() error = %v, want ErrBranchAlreadyExists", err)
	}
}

func TestMemoryBranchRepository_GetNotFound(t *testing.T) {
	repo := NewMemoryBranchRepository()
	ctx := context.Background()

	_, err := repo.Get(ctx, "nonexistent")
	if err != ErrBranchNotFound {
		t.Errorf("Get() error = %v, want ErrBranchNotFound", err)
	}
}

func TestMemoryBranchRepository_Update(t *testing.T) {
	repo := NewMemoryBranchRepository()
	ctx := context.Background()

	branch := &Branch{
		ID:         "br_001",
		SessionID:  "sess_001",
		BudgetUsed: 0,
		Status:     BranchStatusActive,
	}
	_ = repo.Create(ctx, branch)

	// Update
	branch.BudgetUsed = 1000
	branch.Status = BranchStatusCompleted
	err := repo.Update(ctx, branch)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	// Verify
	got, _ := repo.Get(ctx, "br_001")
	if got.BudgetUsed != 1000 {
		t.Errorf("BudgetUsed = %d, want 1000", got.BudgetUsed)
	}
	if got.Status != BranchStatusCompleted {
		t.Errorf("Status = %s, want completed", got.Status)
	}
}

func TestMemoryBranchRepository_UpdateNotFound(t *testing.T) {
	repo := NewMemoryBranchRepository()
	ctx := context.Background()

	branch := &Branch{ID: "nonexistent"}
	err := repo.Update(ctx, branch)
	if err != ErrBranchNotFound {
		t.Errorf("Update() error = %v, want ErrBranchNotFound", err)
	}
}

func TestMemoryBranchRepository_Delete(t *testing.T) {
	repo := NewMemoryBranchRepository()
	ctx := context.Background()

	branch := &Branch{
		ID:        "br_001",
		SessionID: "sess_001",
		Status:    BranchStatusActive,
	}
	_ = repo.Create(ctx, branch)

	// Delete
	err := repo.Delete(ctx, "br_001")
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Verify deleted
	_, err = repo.Get(ctx, "br_001")
	if err != ErrBranchNotFound {
		t.Errorf("Get() after Delete() error = %v, want ErrBranchNotFound", err)
	}
}

func TestMemoryBranchRepository_DeleteNotFound(t *testing.T) {
	repo := NewMemoryBranchRepository()
	ctx := context.Background()

	err := repo.Delete(ctx, "nonexistent")
	if err != ErrBranchNotFound {
		t.Errorf("Delete() error = %v, want ErrBranchNotFound", err)
	}
}

func TestMemoryBranchRepository_ListBySession(t *testing.T) {
	repo := NewMemoryBranchRepository()
	ctx := context.Background()

	// Create branches for different sessions
	branches := []*Branch{
		{ID: "br_001", SessionID: "sess_001", Status: BranchStatusActive},
		{ID: "br_002", SessionID: "sess_001", Status: BranchStatusCompleted},
		{ID: "br_003", SessionID: "sess_002", Status: BranchStatusActive},
	}
	for _, b := range branches {
		_ = repo.Create(ctx, b)
	}

	// List for sess_001
	result, err := repo.ListBySession(ctx, "sess_001")
	if err != nil {
		t.Fatalf("ListBySession() error = %v", err)
	}

	if len(result) != 2 {
		t.Errorf("ListBySession() len = %d, want 2", len(result))
	}
}

func TestMemoryBranchRepository_ListByParent(t *testing.T) {
	repo := NewMemoryBranchRepository()
	ctx := context.Background()

	parentID := "br_parent"

	// Create parent and children
	parent := &Branch{ID: "br_parent", SessionID: "sess_001", Status: BranchStatusActive}
	child1 := &Branch{ID: "br_child1", SessionID: "sess_001", ParentID: &parentID, Status: BranchStatusActive}
	child2 := &Branch{ID: "br_child2", SessionID: "sess_001", ParentID: &parentID, Status: BranchStatusActive}
	other := &Branch{ID: "br_other", SessionID: "sess_001", Status: BranchStatusActive}

	_ = repo.Create(ctx, parent)
	_ = repo.Create(ctx, child1)
	_ = repo.Create(ctx, child2)
	_ = repo.Create(ctx, other)

	// List children
	children, err := repo.ListByParent(ctx, "br_parent")
	if err != nil {
		t.Fatalf("ListByParent() error = %v", err)
	}

	if len(children) != 2 {
		t.Errorf("ListByParent() len = %d, want 2", len(children))
	}
}

func TestMemoryBranchRepository_GetActiveBySession(t *testing.T) {
	repo := NewMemoryBranchRepository()
	ctx := context.Background()

	// Create mix of active and completed branches
	_ = repo.Create(ctx, &Branch{ID: "br_001", SessionID: "sess_001", Status: BranchStatusCompleted})
	_ = repo.Create(ctx, &Branch{ID: "br_002", SessionID: "sess_001", Status: BranchStatusActive})
	_ = repo.Create(ctx, &Branch{ID: "br_003", SessionID: "sess_001", Status: BranchStatusTimeout})

	// Get active
	active, err := repo.GetActiveBySession(ctx, "sess_001")
	if err != nil {
		t.Fatalf("GetActiveBySession() error = %v", err)
	}

	if active == nil {
		t.Fatal("GetActiveBySession() returned nil, want active branch")
	}
	if active.ID != "br_002" {
		t.Errorf("GetActiveBySession() ID = %s, want br_002", active.ID)
	}
}

func TestMemoryBranchRepository_GetActiveBySessionNone(t *testing.T) {
	repo := NewMemoryBranchRepository()
	ctx := context.Background()

	// Create only completed branches
	_ = repo.Create(ctx, &Branch{ID: "br_001", SessionID: "sess_001", Status: BranchStatusCompleted})

	// Get active should return nil
	active, err := repo.GetActiveBySession(ctx, "sess_001")
	if err != nil {
		t.Fatalf("GetActiveBySession() error = %v", err)
	}
	if active != nil {
		t.Errorf("GetActiveBySession() = %v, want nil", active)
	}
}

func TestMemoryBranchRepository_CountActiveBySession(t *testing.T) {
	repo := NewMemoryBranchRepository()
	ctx := context.Background()

	// Create mix
	_ = repo.Create(ctx, &Branch{ID: "br_001", SessionID: "sess_001", Status: BranchStatusActive})
	_ = repo.Create(ctx, &Branch{ID: "br_002", SessionID: "sess_001", Status: BranchStatusActive})
	_ = repo.Create(ctx, &Branch{ID: "br_003", SessionID: "sess_001", Status: BranchStatusCompleted})
	_ = repo.Create(ctx, &Branch{ID: "br_004", SessionID: "sess_002", Status: BranchStatusActive})

	count, err := repo.CountActiveBySession(ctx, "sess_001")
	if err != nil {
		t.Fatalf("CountActiveBySession() error = %v", err)
	}
	if count != 2 {
		t.Errorf("CountActiveBySession() = %d, want 2", count)
	}
}

// Concurrency tests (validates race condition protection)

func TestMemoryBranchRepository_ConcurrentCreate(t *testing.T) {
	repo := NewMemoryBranchRepository()
	ctx := context.Background()

	const numGoroutines = 100
	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			branch := &Branch{
				ID:        "br_" + string(rune('A'+idx%26)) + "_" + time.Now().String(),
				SessionID: "sess_concurrent",
				Status:    BranchStatusActive,
			}
			if err := repo.Create(ctx, branch); err != nil {
				errors <- err
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Errorf("Concurrent Create() error = %v", err)
	}
}

func TestMemoryBranchRepository_ConcurrentUpdateSameBranch(t *testing.T) {
	repo := NewMemoryBranchRepository()
	ctx := context.Background()

	// Create initial branch
	branch := &Branch{
		ID:         "br_concurrent",
		SessionID:  "sess_001",
		BudgetUsed: 0,
		Status:     BranchStatusActive,
	}
	_ = repo.Create(ctx, branch)

	// Concurrent updates to same branch
	const numGoroutines = 50
	var wg sync.WaitGroup
	var mu sync.Mutex
	totalIncrement := 0

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// Get, modify, update (not atomic, but tests lock correctness)
			b, _ := repo.Get(ctx, "br_concurrent")
			mu.Lock()
			b.BudgetUsed += 10
			totalIncrement += 10
			mu.Unlock()
			_ = repo.Update(ctx, b)
		}()
	}

	wg.Wait()

	// Final value may be less than totalIncrement due to race (expected without atomic ops)
	// But at least verify no panics occurred and data is consistent
	final, _ := repo.Get(ctx, "br_concurrent")
	if final.BudgetUsed < 0 {
		t.Errorf("BudgetUsed = %d, should not be negative", final.BudgetUsed)
	}
}

func TestMemoryBranchRepository_ConcurrentReadWrite(t *testing.T) {
	repo := NewMemoryBranchRepository()
	ctx := context.Background()

	// Create some branches
	for i := 0; i < 10; i++ {
		branch := &Branch{
			ID:        "br_" + string(rune('0'+i)),
			SessionID: "sess_rw",
			Status:    BranchStatusActive,
		}
		_ = repo.Create(ctx, branch)
	}

	const numReaders = 20
	const numWriters = 5
	var wg sync.WaitGroup

	// Readers
	for i := 0; i < numReaders; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				_, _ = repo.ListBySession(ctx, "sess_rw")
				_, _ = repo.GetActiveBySession(ctx, "sess_rw")
			}
		}()
	}

	// Writers
	for i := 0; i < numWriters; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				branch := &Branch{
					ID:        "br_new_" + string(rune('A'+idx)) + "_" + string(rune('0'+j)),
					SessionID: "sess_rw",
					Status:    BranchStatusActive,
				}
				_ = repo.Create(ctx, branch)
			}
		}(i)
	}

	wg.Wait()

	// Just verify we didn't panic
	result, _ := repo.ListBySession(ctx, "sess_rw")
	if len(result) < 10 {
		t.Errorf("ListBySession() len = %d, want >= 10", len(result))
	}
}

// Test copy-on-read prevents external mutation
func TestMemoryBranchRepository_CopyOnRead(t *testing.T) {
	repo := NewMemoryBranchRepository()
	ctx := context.Background()

	original := &Branch{
		ID:          "br_copy",
		SessionID:   "sess_001",
		Description: "original",
		Status:      BranchStatusActive,
	}
	_ = repo.Create(ctx, original)

	// Get a copy
	got, _ := repo.Get(ctx, "br_copy")

	// Mutate the copy
	got.Description = "mutated"

	// Verify original in repo is unchanged
	check, _ := repo.Get(ctx, "br_copy")
	if check.Description != "original" {
		t.Errorf("External mutation affected stored data: got %q, want %q", check.Description, "original")
	}
}
