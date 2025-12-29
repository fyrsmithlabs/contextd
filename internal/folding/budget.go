package folding

import (
	"math"
	"sync"
	"sync/atomic"
)

// budgetState tracks budget for a single branch.
type budgetState struct {
	total int64
	used  int64 // accessed atomically for lock-free reads
}

// BudgetTracker tracks token budgets per branch and emits events on exhaustion.
// It does NOT directly call BranchManager to avoid circular dependencies.
// Instead, it emits events that BranchManager subscribes to.
type BudgetTracker struct {
	mu      sync.RWMutex
	budgets map[string]*budgetState
	emitter EventEmitter
}

// NewBudgetTracker creates a new budget tracker with the given event emitter.
func NewBudgetTracker(emitter EventEmitter) *BudgetTracker {
	return &BudgetTracker{
		budgets: make(map[string]*budgetState),
		emitter: emitter,
	}
}

// Allocate initializes budget tracking for a branch.
func (t *BudgetTracker) Allocate(branchID string, budget int) error {
	if budget <= 0 {
		return ErrInvalidBudget
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	t.budgets[branchID] = &budgetState{
		total: int64(budget),
		used:  0,
	}

	return nil
}

// Consume attempts to consume tokens from a branch's budget.
// Returns ErrBudgetExhausted if the budget would be exceeded.
// Emits BudgetWarningEvent at 80% usage, BudgetExhaustedEvent when exceeded.
// NOTE: Events are emitted AFTER releasing the lock to prevent deadlocks when
// event handlers call back into BudgetTracker methods.
func (t *BudgetTracker) Consume(branchID string, tokens int) error {
	// Collect event data while holding the lock, emit after release
	var exhaustedEvent *BudgetExhaustedEvent
	var warningEvent *BudgetWarningEvent
	var err error

	func() {
		t.mu.Lock()
		defer t.mu.Unlock()

		state, exists := t.budgets[branchID]
		if !exists {
			err = ErrBudgetNotFound
			return
		}

		currentUsed := atomic.LoadInt64(&state.used)

		// Check for integer overflow (SEC-004)
		if tokens < 0 || int64(tokens) > math.MaxInt64-currentUsed {
			err = ErrInvalidBudget
			return
		}

		newUsed := currentUsed + int64(tokens)

		// Check if this would exceed budget
		if newUsed > state.total {
			exhaustedEvent = &BudgetExhaustedEvent{
				branchID:    branchID,
				BudgetUsed:  int(currentUsed),
				BudgetTotal: int(state.total),
			}
			err = ErrBudgetExhausted
			return
		}

		// Update usage
		atomic.StoreInt64(&state.used, newUsed)

		// Check for warning threshold (80%)
		percentage := float64(newUsed) / float64(state.total)
		if percentage >= 0.8 {
			// Only emit warning once when crossing threshold
			prevPercentage := float64(currentUsed) / float64(state.total)
			if prevPercentage < 0.8 {
				warningEvent = &BudgetWarningEvent{
					branchID:    branchID,
					BudgetUsed:  int(newUsed),
					BudgetTotal: int(state.total),
					Percentage:  percentage,
				}
			}
		}
	}()

	// Emit events AFTER lock is released to prevent deadlock
	if t.emitter != nil {
		if exhaustedEvent != nil {
			t.emitter.Emit(*exhaustedEvent)
		}
		if warningEvent != nil {
			t.emitter.Emit(*warningEvent)
		}
	}

	return err
}

// Remaining returns the remaining tokens for a branch.
func (t *BudgetTracker) Remaining(branchID string) (int, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	state, exists := t.budgets[branchID]
	if !exists {
		return 0, ErrBudgetNotFound
	}

	used := atomic.LoadInt64(&state.used)
	return int(state.total - used), nil
}

// Used returns the used tokens for a branch.
func (t *BudgetTracker) Used(branchID string) (int, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	state, exists := t.budgets[branchID]
	if !exists {
		return 0, ErrBudgetNotFound
	}

	return int(atomic.LoadInt64(&state.used)), nil
}

// IsExhausted returns true if a branch's budget is exhausted.
func (t *BudgetTracker) IsExhausted(branchID string) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()

	state, exists := t.budgets[branchID]
	if !exists {
		return false
	}

	used := atomic.LoadInt64(&state.used)
	return used >= state.total
}

// Deallocate removes budget tracking for a branch.
func (t *BudgetTracker) Deallocate(branchID string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	delete(t.budgets, branchID)
}

// SimpleEventEmitter is a basic implementation of EventEmitter for testing.
type SimpleEventEmitter struct {
	mu       sync.RWMutex
	handlers []func(BranchEvent)
	events   []BranchEvent // for testing
}

// NewSimpleEventEmitter creates a new simple event emitter.
func NewSimpleEventEmitter() *SimpleEventEmitter {
	return &SimpleEventEmitter{
		handlers: make([]func(BranchEvent), 0),
		events:   make([]BranchEvent, 0),
	}
}

// Emit sends an event to all subscribers.
func (e *SimpleEventEmitter) Emit(event BranchEvent) {
	e.mu.Lock()
	e.events = append(e.events, event)
	handlers := make([]func(BranchEvent), len(e.handlers))
	copy(handlers, e.handlers)
	e.mu.Unlock()

	for _, h := range handlers {
		h(event)
	}
}

// Subscribe registers a handler for events.
func (e *SimpleEventEmitter) Subscribe(handler func(BranchEvent)) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.handlers = append(e.handlers, handler)
}

// Events returns all emitted events (for testing).
func (e *SimpleEventEmitter) Events() []BranchEvent {
	e.mu.RLock()
	defer e.mu.RUnlock()
	result := make([]BranchEvent, len(e.events))
	copy(result, e.events)
	return result
}

// Clear clears all recorded events (for testing).
func (e *SimpleEventEmitter) Clear() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.events = make([]BranchEvent, 0)
}
