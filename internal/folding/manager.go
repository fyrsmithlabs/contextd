package folding

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
)

// FoldingConfig holds configuration for context-folding.
type FoldingConfig struct {
	DefaultBudget              int     `json:"default_budget" koanf:"default_budget"`
	MaxBudget                  int     `json:"max_budget" koanf:"max_budget"`
	MaxDepth                   int     `json:"max_depth" koanf:"max_depth"`
	DefaultTimeoutSeconds      int     `json:"default_timeout_seconds" koanf:"default_timeout_seconds"`
	MaxTimeoutSeconds          int     `json:"max_timeout_seconds" koanf:"max_timeout_seconds"`
	InjectionBudgetRatio       float64 `json:"injection_budget_ratio" koanf:"injection_budget_ratio"`
	MemoryMinConfidence        float64 `json:"memory_min_confidence" koanf:"memory_min_confidence"`
	MemoryMaxItems             int     `json:"memory_max_items" koanf:"memory_max_items"`
	MaxConcurrentPerSession    int     `json:"max_concurrent_per_session" koanf:"max_concurrent_per_session"`
	MaxConcurrentPerInstance   int     `json:"max_concurrent_per_instance" koanf:"max_concurrent_per_instance"`
}

// DefaultFoldingConfig returns sensible defaults.
func DefaultFoldingConfig() *FoldingConfig {
	return &FoldingConfig{
		DefaultBudget:              8192,
		MaxBudget:                  32768,
		MaxDepth:                   3,
		DefaultTimeoutSeconds:      300,
		MaxTimeoutSeconds:          600,
		InjectionBudgetRatio:       0.2,
		MemoryMinConfidence:        0.7,
		MemoryMaxItems:             10,
		MaxConcurrentPerSession:    10,
		MaxConcurrentPerInstance:   100,
	}
}

// BranchManager orchestrates branch lifecycle.
type BranchManager struct {
	repo     BranchRepository
	budget   *BudgetTracker
	scrubber SecretScrubber
	emitter  EventEmitter
	config   *FoldingConfig

	// Timeout management
	timeoutMu      sync.Mutex
	timeoutCancels map[string]context.CancelFunc

	// Rate limiting
	instanceBranchCount int64
}

// NewBranchManager creates a new branch manager.
func NewBranchManager(
	repo BranchRepository,
	budget *BudgetTracker,
	scrubber SecretScrubber,
	emitter EventEmitter,
	config *FoldingConfig,
) *BranchManager {
	if config == nil {
		config = DefaultFoldingConfig()
	}

	m := &BranchManager{
		repo:           repo,
		budget:         budget,
		scrubber:       scrubber,
		emitter:        emitter,
		config:         config,
		timeoutCancels: make(map[string]context.CancelFunc),
	}

	// Subscribe to budget events
	if emitter != nil {
		emitter.Subscribe(m.handleEvent)
	}

	return m
}

// handleEvent processes events from BudgetTracker.
func (m *BranchManager) handleEvent(event BranchEvent) {
	switch e := event.(type) {
	case BudgetExhaustedEvent:
		// Force return the branch
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = m.ForceReturn(ctx, e.BranchID(), "budget exhausted")

	case TimeoutEvent:
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = m.ForceReturn(ctx, e.BranchID(), "timeout exceeded")
	}
}

// Create creates a new branch.
func (m *BranchManager) Create(ctx context.Context, req BranchRequest) (*BranchResponse, error) {
	// Validate input (SEC-001)
	if err := req.Validate(); err != nil {
		return nil, err
	}
	req.ApplyDefaults()

	// Check instance-level rate limit (SEC-003)
	instanceCount := atomic.LoadInt64(&m.instanceBranchCount)
	if instanceCount >= int64(m.config.MaxConcurrentPerInstance) {
		return nil, ErrMaxConcurrentBranches
	}

	// Check per-session rate limit (SEC-003)
	activeCount, err := m.repo.CountActiveBySession(ctx, req.SessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to count active branches: %w", err)
	}
	if activeCount >= m.config.MaxConcurrentPerSession {
		return nil, ErrMaxConcurrentBranches
	}

	// Determine depth
	depth := 0
	var parentID *string
	parent, err := m.repo.GetActiveBySession(ctx, req.SessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get active branch: %w", err)
	}
	if parent != nil {
		depth = parent.Depth + 1
		parentID = &parent.ID
	}

	// Check max depth (FR-006)
	if depth >= m.config.MaxDepth {
		return nil, ErrMaxDepthExceeded
	}

	// Cap budget
	budget := req.Budget
	if budget > m.config.MaxBudget {
		budget = m.config.MaxBudget
	}

	// Cap timeout
	timeout := req.TimeoutSeconds
	if timeout > m.config.MaxTimeoutSeconds {
		timeout = m.config.MaxTimeoutSeconds
	}

	// Create branch
	branch := &Branch{
		ID:             "br_" + uuid.New().String()[:8],
		SessionID:      req.SessionID,
		ParentID:       parentID,
		Depth:          depth,
		Description:    req.Description,
		Prompt:         req.Prompt,
		BudgetTotal:    budget,
		BudgetUsed:     0,
		TimeoutSeconds: timeout,
		Status:         BranchStatusActive,
		CreatedAt:      time.Now(),
	}

	// Allocate budget
	if err := m.budget.Allocate(branch.ID, budget); err != nil {
		return nil, fmt.Errorf("failed to allocate budget: %w", err)
	}

	// Store branch
	if err := m.repo.Create(ctx, branch); err != nil {
		m.budget.Deallocate(branch.ID)
		return nil, fmt.Errorf("failed to create branch: %w", err)
	}

	// Start timeout goroutine (FR-007)
	m.startTimeoutWatcher(branch.ID, timeout)

	// Increment instance branch count
	atomic.AddInt64(&m.instanceBranchCount, 1)

	return &BranchResponse{
		BranchID:        branch.ID,
		BudgetAllocated: budget,
		Depth:           depth,
	}, nil
}

// Return completes a branch with results.
func (m *BranchManager) Return(ctx context.Context, req ReturnRequest) (*ReturnResponse, error) {
	// Validate input
	if err := req.Validate(); err != nil {
		return nil, err
	}

	// Get branch
	branch, err := m.repo.Get(ctx, req.BranchID)
	if err != nil {
		return nil, err
	}

	// Verify active status
	if branch.Status != BranchStatusActive {
		return nil, ErrBranchNotActive
	}

	// Note: First branch in a session has no ParentID but is still returnable
	// ErrCannotReturnFromRoot would apply to a hypothetical "root session" entity,
	// but regular branches (even at depth 0) can always return.

	// Force-return any active children first (FR-009)
	children, err := m.repo.ListByParent(ctx, branch.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to list children: %w", err)
	}
	for _, child := range children {
		if child.Status == BranchStatusActive {
			if err := m.ForceReturn(ctx, child.ID, "parent returning"); err != nil {
				// Log but continue
			}
		}
	}

	// Scrub secrets (SEC-002 - CRITICAL)
	scrubbedMsg := req.Message
	if m.scrubber != nil {
		scrubbed, err := m.scrubber.Scrub(req.Message)
		if err != nil {
			// Fail closed - do not return unscrubbed content
			return nil, ErrScrubbingFailed
		}
		scrubbedMsg = scrubbed
	}

	// Cancel timeout
	m.cancelTimeout(branch.ID)

	// Get final token usage
	tokensUsed, _ := m.budget.Used(branch.ID)

	// Update branch state
	now := time.Now()
	branch.Status = BranchStatusCompleted
	branch.Result = &scrubbedMsg
	branch.CompletedAt = &now
	branch.BudgetUsed = tokensUsed

	if err := m.repo.Update(ctx, branch); err != nil {
		return nil, fmt.Errorf("failed to update branch: %w", err)
	}

	// Cleanup budget
	m.budget.Deallocate(branch.ID)

	// Decrement instance branch count
	atomic.AddInt64(&m.instanceBranchCount, -1)

	// Emit completion event
	if m.emitter != nil {
		m.emitter.Emit(BranchCompletedEvent{
			branchID:   branch.ID,
			TokensUsed: tokensUsed,
			Success:    true,
		})
	}

	return &ReturnResponse{
		Success:     true,
		TokensUsed:  tokensUsed,
		ScrubbedMsg: scrubbedMsg,
	}, nil
}

// ForceReturn terminates a branch with the given reason.
func (m *BranchManager) ForceReturn(ctx context.Context, branchID string, reason string) error {
	branch, err := m.repo.Get(ctx, branchID)
	if err != nil {
		return err
	}

	// Already terminal - idempotent
	if branch.Status.IsTerminal() {
		return nil
	}

	// Force-return children first (recursive)
	children, _ := m.repo.ListByParent(ctx, branchID)
	for _, child := range children {
		if child.Status == BranchStatusActive {
			_ = m.ForceReturn(ctx, child.ID, "parent force-returned")
		}
	}

	// Cancel timeout
	m.cancelTimeout(branchID)

	// Determine terminal status
	status := BranchStatusFailed
	if reason == "timeout exceeded" {
		status = BranchStatusTimeout
	}

	// Get token usage
	tokensUsed, _ := m.budget.Used(branchID)

	// Update branch
	now := time.Now()
	branch.Status = status
	branch.Error = &reason
	branch.CompletedAt = &now
	branch.BudgetUsed = tokensUsed

	if err := m.repo.Update(ctx, branch); err != nil {
		return err
	}

	// Cleanup budget
	m.budget.Deallocate(branchID)

	// Decrement instance branch count
	atomic.AddInt64(&m.instanceBranchCount, -1)

	return nil
}

// Get retrieves a branch by ID.
func (m *BranchManager) Get(ctx context.Context, branchID string) (*Branch, error) {
	return m.repo.Get(ctx, branchID)
}

// GetActive returns the currently active branch for a session.
func (m *BranchManager) GetActive(ctx context.Context, sessionID string) (*Branch, error) {
	return m.repo.GetActiveBySession(ctx, sessionID)
}

// ListBySession returns all branches for a session.
func (m *BranchManager) ListBySession(ctx context.Context, sessionID string) ([]*Branch, error) {
	return m.repo.ListBySession(ctx, sessionID)
}

// CleanupSession force-returns all active branches for a session (FR-010).
func (m *BranchManager) CleanupSession(ctx context.Context, sessionID string) error {
	branches, err := m.repo.ListBySession(ctx, sessionID)
	if err != nil {
		return err
	}

	// Sort by depth (deepest first) - cleanup children before parents
	sort.Slice(branches, func(i, j int) bool {
		return branches[i].Depth > branches[j].Depth
	})

	for _, branch := range branches {
		if branch.Status == BranchStatusActive {
			_ = m.ForceReturn(ctx, branch.ID, "session ended")
		}
	}

	return nil
}

// ConsumeTokens records token consumption for a branch.
func (m *BranchManager) ConsumeTokens(ctx context.Context, branchID string, tokens int) error {
	return m.budget.Consume(branchID, tokens)
}

// startTimeoutWatcher starts a goroutine to enforce timeout.
func (m *BranchManager) startTimeoutWatcher(branchID string, timeoutSeconds int) {
	ctx, cancel := context.WithCancel(context.Background())

	m.timeoutMu.Lock()
	m.timeoutCancels[branchID] = cancel
	m.timeoutMu.Unlock()

	go func() {
		select {
		case <-ctx.Done():
			// Cancelled - branch completed normally
			return
		case <-time.After(time.Duration(timeoutSeconds) * time.Second):
			// Timeout - clean up map entry and emit event
			m.timeoutMu.Lock()
			delete(m.timeoutCancels, branchID)
			m.timeoutMu.Unlock()

			if m.emitter != nil {
				m.emitter.Emit(TimeoutEvent{
					branchID:       branchID,
					TimeoutSeconds: timeoutSeconds,
				})
			}
		}
	}()
}

// cancelTimeout cancels the timeout watcher for a branch.
func (m *BranchManager) cancelTimeout(branchID string) {
	m.timeoutMu.Lock()
	defer m.timeoutMu.Unlock()

	if cancel, exists := m.timeoutCancels[branchID]; exists {
		cancel()
		delete(m.timeoutCancels, branchID)
	}
}
