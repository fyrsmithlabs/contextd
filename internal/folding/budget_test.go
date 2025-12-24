package folding

import (
	"sync"
	"testing"
)

func TestBudgetTracker_Allocate(t *testing.T) {
	emitter := NewSimpleEventEmitter()
	tracker := NewBudgetTracker(emitter)

	err := tracker.Allocate("br_001", 8192)
	if err != nil {
		t.Fatalf("Allocate() error = %v", err)
	}

	remaining, err := tracker.Remaining("br_001")
	if err != nil {
		t.Fatalf("Remaining() error = %v", err)
	}
	if remaining != 8192 {
		t.Errorf("Remaining() = %d, want 8192", remaining)
	}
}

func TestBudgetTracker_AllocateInvalidBudget(t *testing.T) {
	emitter := NewSimpleEventEmitter()
	tracker := NewBudgetTracker(emitter)

	tests := []struct {
		name   string
		budget int
	}{
		{"zero", 0},
		{"negative", -100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tracker.Allocate("br_test", tt.budget)
			if err != ErrInvalidBudget {
				t.Errorf("Allocate(%d) error = %v, want ErrInvalidBudget", tt.budget, err)
			}
		})
	}
}

func TestBudgetTracker_Consume(t *testing.T) {
	emitter := NewSimpleEventEmitter()
	tracker := NewBudgetTracker(emitter)
	_ = tracker.Allocate("br_001", 1000)

	// Consume some tokens
	err := tracker.Consume("br_001", 300)
	if err != nil {
		t.Fatalf("Consume() error = %v", err)
	}

	remaining, _ := tracker.Remaining("br_001")
	if remaining != 700 {
		t.Errorf("Remaining() after consume = %d, want 700", remaining)
	}

	used, _ := tracker.Used("br_001")
	if used != 300 {
		t.Errorf("Used() = %d, want 300", used)
	}
}

func TestBudgetTracker_ConsumeExhausted(t *testing.T) {
	emitter := NewSimpleEventEmitter()
	tracker := NewBudgetTracker(emitter)
	_ = tracker.Allocate("br_001", 1000)

	// Consume more than budget
	err := tracker.Consume("br_001", 1100)
	if err != ErrBudgetExhausted {
		t.Errorf("Consume() error = %v, want ErrBudgetExhausted", err)
	}

	// Check event was emitted
	events := emitter.Events()
	if len(events) != 1 {
		t.Fatalf("Expected 1 event, got %d", len(events))
	}
	if events[0].Type() != "budget_exhausted" {
		t.Errorf("Event type = %s, want budget_exhausted", events[0].Type())
	}
}

func TestBudgetTracker_ConsumeWarning(t *testing.T) {
	emitter := NewSimpleEventEmitter()
	tracker := NewBudgetTracker(emitter)
	_ = tracker.Allocate("br_001", 1000)

	// Consume below 80%
	_ = tracker.Consume("br_001", 700)
	if len(emitter.Events()) != 0 {
		t.Errorf("Expected no events at 70%%, got %d", len(emitter.Events()))
	}

	// Consume to cross 80% threshold
	_ = tracker.Consume("br_001", 150) // now at 85%
	events := emitter.Events()
	if len(events) != 1 {
		t.Fatalf("Expected 1 warning event at 85%%, got %d", len(events))
	}
	if events[0].Type() != "budget_warning" {
		t.Errorf("Event type = %s, want budget_warning", events[0].Type())
	}
}

func TestBudgetTracker_ConsumeWarningOnlyOnce(t *testing.T) {
	emitter := NewSimpleEventEmitter()
	tracker := NewBudgetTracker(emitter)
	_ = tracker.Allocate("br_001", 1000)

	// Cross 80% threshold
	_ = tracker.Consume("br_001", 850)
	if len(emitter.Events()) != 1 {
		t.Fatalf("Expected 1 warning event, got %d", len(emitter.Events()))
	}

	// Consume more (stay above 80%)
	_ = tracker.Consume("br_001", 50) // now at 90%

	// Should NOT emit another warning
	if len(emitter.Events()) != 1 {
		t.Errorf("Expected still 1 warning event (not duplicate), got %d", len(emitter.Events()))
	}
}

func TestBudgetTracker_ConsumeNotFound(t *testing.T) {
	emitter := NewSimpleEventEmitter()
	tracker := NewBudgetTracker(emitter)

	err := tracker.Consume("nonexistent", 100)
	if err != ErrBudgetNotFound {
		t.Errorf("Consume() error = %v, want ErrBudgetNotFound", err)
	}
}

func TestBudgetTracker_RemainingNotFound(t *testing.T) {
	emitter := NewSimpleEventEmitter()
	tracker := NewBudgetTracker(emitter)

	_, err := tracker.Remaining("nonexistent")
	if err != ErrBudgetNotFound {
		t.Errorf("Remaining() error = %v, want ErrBudgetNotFound", err)
	}
}

func TestBudgetTracker_IsExhausted(t *testing.T) {
	emitter := NewSimpleEventEmitter()
	tracker := NewBudgetTracker(emitter)
	_ = tracker.Allocate("br_001", 100)

	if tracker.IsExhausted("br_001") {
		t.Error("IsExhausted() = true, want false initially")
	}

	_ = tracker.Consume("br_001", 100)

	if !tracker.IsExhausted("br_001") {
		t.Error("IsExhausted() = false, want true after consuming all")
	}
}

func TestBudgetTracker_IsExhaustedNotFound(t *testing.T) {
	emitter := NewSimpleEventEmitter()
	tracker := NewBudgetTracker(emitter)

	// Should return false for nonexistent (not panic)
	if tracker.IsExhausted("nonexistent") {
		t.Error("IsExhausted() = true for nonexistent, want false")
	}
}

func TestBudgetTracker_Deallocate(t *testing.T) {
	emitter := NewSimpleEventEmitter()
	tracker := NewBudgetTracker(emitter)
	_ = tracker.Allocate("br_001", 1000)

	tracker.Deallocate("br_001")

	_, err := tracker.Remaining("br_001")
	if err != ErrBudgetNotFound {
		t.Errorf("Remaining() after Deallocate() error = %v, want ErrBudgetNotFound", err)
	}
}

// Concurrent tests

func TestBudgetTracker_ConcurrentConsume(t *testing.T) {
	emitter := NewSimpleEventEmitter()
	tracker := NewBudgetTracker(emitter)
	_ = tracker.Allocate("br_001", 10000)

	const numGoroutines = 100
	const tokensPerConsume = 50

	var wg sync.WaitGroup
	successCount := int64(0)
	var mu sync.Mutex

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := tracker.Consume("br_001", tokensPerConsume); err == nil {
				mu.Lock()
				successCount++
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	// With 10000 budget and 50 tokens per consume, max 200 can succeed
	// Due to races, actual may be less, but should be reasonable
	if successCount > 200 {
		t.Errorf("successCount = %d, expected <= 200 (budget limit)", successCount)
	}

	// Verify final state is consistent
	used, _ := tracker.Used("br_001")
	remaining, _ := tracker.Remaining("br_001")

	if used+remaining != 10000 {
		t.Errorf("used(%d) + remaining(%d) != 10000", used, remaining)
	}
}

func TestBudgetTracker_ConcurrentAllocateDeallocate(t *testing.T) {
	emitter := NewSimpleEventEmitter()
	tracker := NewBudgetTracker(emitter)

	var wg sync.WaitGroup

	// Concurrent allocate/deallocate
	for i := 0; i < 50; i++ {
		wg.Add(2)
		branchID := "br_" + string(rune('A'+i%26))

		go func(id string) {
			defer wg.Done()
			_ = tracker.Allocate(id, 1000)
		}(branchID)

		go func(id string) {
			defer wg.Done()
			tracker.Deallocate(id)
		}(branchID)
	}

	wg.Wait()

	// Just verify no panics occurred
}

// Test event subscription

func TestSimpleEventEmitter_Subscribe(t *testing.T) {
	emitter := NewSimpleEventEmitter()

	received := make([]BranchEvent, 0)
	var mu sync.Mutex

	emitter.Subscribe(func(e BranchEvent) {
		mu.Lock()
		received = append(received, e)
		mu.Unlock()
	})

	// Emit some events
	emitter.Emit(BudgetWarningEvent{branchID: "br_001", BudgetUsed: 800, BudgetTotal: 1000, Percentage: 0.8})
	emitter.Emit(BudgetExhaustedEvent{branchID: "br_001", BudgetUsed: 1000, BudgetTotal: 1000})

	mu.Lock()
	defer mu.Unlock()

	if len(received) != 2 {
		t.Errorf("Received %d events, want 2", len(received))
	}
}
