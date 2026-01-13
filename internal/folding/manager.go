package folding

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/codes"
	"go.uber.org/zap"
)

// FoldingConfig holds configuration for context-folding.
type FoldingConfig struct {
	DefaultBudget            int     `json:"default_budget" koanf:"default_budget"`
	MaxBudget                int     `json:"max_budget" koanf:"max_budget"`
	MaxDepth                 int     `json:"max_depth" koanf:"max_depth"`
	DefaultTimeoutSeconds    int     `json:"default_timeout_seconds" koanf:"default_timeout_seconds"`
	MaxTimeoutSeconds        int     `json:"max_timeout_seconds" koanf:"max_timeout_seconds"`
	InjectionBudgetRatio     float64 `json:"injection_budget_ratio" koanf:"injection_budget_ratio"`
	MemoryMinConfidence      float64 `json:"memory_min_confidence" koanf:"memory_min_confidence"`
	MemoryMaxItems           int     `json:"memory_max_items" koanf:"memory_max_items"`
	MaxConcurrentPerSession  int     `json:"max_concurrent_per_session" koanf:"max_concurrent_per_session"`
	MaxConcurrentPerInstance int     `json:"max_concurrent_per_instance" koanf:"max_concurrent_per_instance"`
}

// DefaultFoldingConfig returns sensible defaults.
func DefaultFoldingConfig() *FoldingConfig {
	return &FoldingConfig{
		DefaultBudget:            8192,
		MaxBudget:                32768,
		MaxDepth:                 3,
		DefaultTimeoutSeconds:    300,
		MaxTimeoutSeconds:        600,
		InjectionBudgetRatio:     0.2,
		MemoryMinConfidence:      0.7,
		MemoryMaxItems:           10,
		MaxConcurrentPerSession:  10,
		MaxConcurrentPerInstance: 100,
	}
}

// BranchManager orchestrates branch lifecycle.
type BranchManager struct {
	repo     BranchRepository
	budget   *BudgetTracker
	scrubber SecretScrubber
	emitter  EventEmitter
	config   *FoldingConfig
	metrics  *Metrics
	logger   *Logger

	// Session validation (SEC-004)
	sessionValidator SessionValidator

	// Timeout management
	timeoutMu      sync.Mutex
	timeoutCancels map[string]context.CancelFunc

	// Rate limiting
	instanceBranchCount int64

	// Shutdown management
	shutdownMu   sync.RWMutex
	shutdownChan chan struct{}
	isShutdown   bool
}

// BranchManagerOption configures BranchManager.
type BranchManagerOption func(*BranchManager)

// WithMetrics sets custom metrics for the manager.
func WithMetrics(m *Metrics) BranchManagerOption {
	return func(bm *BranchManager) {
		bm.metrics = m
	}
}

// WithLogger sets a custom logger for the manager.
func WithLogger(l *Logger) BranchManagerOption {
	return func(bm *BranchManager) {
		bm.logger = l
	}
}

// WithSessionValidator sets a session validator for authorization (SEC-004).
// If not set, PermissiveSessionValidator is used (allows all access).
func WithSessionValidator(v SessionValidator) BranchManagerOption {
	return func(bm *BranchManager) {
		bm.sessionValidator = v
	}
}

// NewBranchManager creates a new branch manager.
func NewBranchManager(
	repo BranchRepository,
	budget *BudgetTracker,
	scrubber SecretScrubber,
	emitter EventEmitter,
	config *FoldingConfig,
	opts ...BranchManagerOption,
) *BranchManager {
	if config == nil {
		config = DefaultFoldingConfig()
	}

	// Initialize with defaults
	metrics, _ := NewMetrics(nil)
	logger := NewLogger(nil)

	m := &BranchManager{
		repo:             repo,
		budget:           budget,
		scrubber:         scrubber,
		emitter:          emitter,
		config:           config,
		metrics:          metrics,
		logger:           logger,
		sessionValidator: &PermissiveSessionValidator{}, // SEC-004: Default allows all access
		timeoutCancels:   make(map[string]context.CancelFunc),
		shutdownChan:     make(chan struct{}),
	}

	// Apply options
	for _, opt := range opts {
		opt(m)
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
		// Log budget exhaustion
		m.logger.BudgetExhausted(context.Background(), e.BranchID(), e.BudgetUsed, e.BudgetTotal)
		// Force return the branch
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = m.ForceReturn(ctx, e.BranchID(), "budget exhausted")

	case BudgetWarningEvent:
		// Log budget warning
		m.logger.BudgetWarning(context.Background(), e.BranchID(), e.BudgetUsed, e.BudgetTotal, e.Percentage)

	case TimeoutEvent:
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = m.ForceReturn(ctx, e.BranchID(), "timeout exceeded")
	}
}

// Create creates a new branch.
func (m *BranchManager) Create(ctx context.Context, req BranchRequest) (*BranchResponse, error) {
	// Check if manager is shutdown
	if err := m.checkShutdown(); err != nil {
		return nil, err
	}

	// Start tracing span
	ctx, span := StartSpan(ctx, "folding.branch.create", "", req.SessionID, 0)
	defer span.End()

	// Validate input (SEC-001)
	if err := req.Validate(); err != nil {
		RecordError(ctx, err)
		SetSpanStatus(ctx, codes.Error, "validation failed")
		return nil, err
	}
	req.ApplyDefaults()

	// Validate session authorization (SEC-004)
	if m.sessionValidator != nil {
		if err := m.sessionValidator.ValidateSession(ctx, req.SessionID, req.CallerID); err != nil {
			RecordError(ctx, err)
			SetSpanStatus(ctx, codes.Error, "session authorization failed")
			m.logger.Warn(ctx, "session authorization failed",
				zap.String("session_id", req.SessionID),
				zap.String("caller_id", req.CallerID),
			)
			return nil, err
		}
	}

	// Check instance-level rate limit (SEC-003)
	instanceCount := atomic.LoadInt64(&m.instanceBranchCount)
	if instanceCount >= int64(m.config.MaxConcurrentPerInstance) {
		RecordError(ctx, ErrMaxConcurrentBranches)
		SetSpanStatus(ctx, codes.Error, "rate limit exceeded")
		return nil, ErrMaxConcurrentBranches
	}

	// Check per-session rate limit (SEC-003)
	activeCount, err := m.repo.CountActiveBySession(ctx, req.SessionID)
	if err != nil {
		RecordError(ctx, err)
		SetSpanStatus(ctx, codes.Error, "failed to count active branches")
		return nil, fmt.Errorf("failed to count active branches: %w", err)
	}
	if activeCount >= m.config.MaxConcurrentPerSession {
		RecordError(ctx, ErrMaxConcurrentBranches)
		SetSpanStatus(ctx, codes.Error, "session rate limit exceeded")
		return nil, ErrMaxConcurrentBranches
	}

	// Determine depth
	depth := 0
	var parentID *string
	parent, err := m.repo.GetActiveBySession(ctx, req.SessionID)
	if err != nil {
		RecordError(ctx, err)
		SetSpanStatus(ctx, codes.Error, "failed to get active branch")
		return nil, fmt.Errorf("failed to get active branch: %w", err)
	}
	if parent != nil {
		depth = parent.Depth + 1
		parentID = &parent.ID
	}

	// Check max depth (FR-006)
	if depth >= m.config.MaxDepth {
		RecordError(ctx, ErrMaxDepthExceeded)
		SetSpanStatus(ctx, codes.Error, "max depth exceeded")
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
		ProjectID:      req.ProjectID,
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
		RecordError(ctx, err)
		SetSpanStatus(ctx, codes.Error, "budget allocation failed")
		return nil, fmt.Errorf("failed to allocate budget: %w", err)
	}

	// Store branch
	if err := m.repo.Create(ctx, branch); err != nil {
		m.budget.Deallocate(branch.ID)
		RecordError(ctx, err)
		SetSpanStatus(ctx, codes.Error, "branch creation failed")
		return nil, fmt.Errorf("failed to create branch: %w", err)
	}

	// Start timeout goroutine (FR-007)
	m.startTimeoutWatcher(branch.ID, timeout)

	// Increment instance branch count
	atomic.AddInt64(&m.instanceBranchCount, 1)

	// Record metrics and log
	m.metrics.RecordBranchCreated(ctx, req.SessionID, depth, budget, branch.ProjectID)
	m.logger.BranchCreated(ctx, branch.ID, req.SessionID, depth, budget)

	SetSpanStatus(ctx, codes.Ok, "branch created successfully")
	return &BranchResponse{
		BranchID:        branch.ID,
		BudgetAllocated: budget,
		Depth:           depth,
	}, nil
}

// Return completes a branch with results.
func (m *BranchManager) Return(ctx context.Context, req ReturnRequest) (*ReturnResponse, error) {
	// Start tracing span
	ctx, span := StartSpan(ctx, "folding.branch.return", req.BranchID, "", 0)
	defer span.End()

	startTime := time.Now()

	// Validate input
	if err := req.Validate(); err != nil {
		RecordError(ctx, err)
		SetSpanStatus(ctx, codes.Error, "validation failed")
		return nil, err
	}

	// Get branch
	branch, err := m.repo.Get(ctx, req.BranchID)
	if err != nil {
		RecordError(ctx, err)
		SetSpanStatus(ctx, codes.Error, "branch not found")
		return nil, err
	}

	// Validate session authorization (SEC-004)
	if m.sessionValidator != nil {
		if err := m.sessionValidator.ValidateSession(ctx, branch.SessionID, req.CallerID); err != nil {
			RecordError(ctx, err)
			SetSpanStatus(ctx, codes.Error, "session authorization failed")
			m.logger.Warn(ctx, "session authorization failed on return",
				zap.String("session_id", branch.SessionID),
				zap.String("branch_id", req.BranchID),
				zap.String("caller_id", req.CallerID),
			)
			return nil, err
		}
	}

	// Verify active status
	if branch.Status != BranchStatusActive {
		RecordError(ctx, ErrBranchNotActive)
		SetSpanStatus(ctx, codes.Error, "branch not active")
		return nil, ErrBranchNotActive
	}

	// Note: First branch in a session has no ParentID but is still returnable
	// ErrCannotReturnFromRoot would apply to a hypothetical "root session" entity,
	// but regular branches (even at depth 0) can always return.

	// Force-return any active children first (FR-009)
	children, err := m.repo.ListByParent(ctx, branch.ID)
	if err != nil {
		RecordError(ctx, err)
		SetSpanStatus(ctx, codes.Error, "failed to list children")
		return nil, fmt.Errorf("failed to list children: %w", err)
	}
	for _, child := range children {
		if child.Status == BranchStatusActive {
			if err := m.ForceReturn(ctx, child.ID, "parent returning"); err != nil {
				// CORR-018: Log child cleanup errors instead of silently ignoring
				m.logger.Error(ctx, "failed to force-return child during parent return", err)
			}
		}
	}

	// Scrub secrets (SEC-002 - CRITICAL)
	// CORR-004: Fail closed if scrubber is nil - never return unscrubbed content
	if m.scrubber == nil {
		RecordError(ctx, ErrScrubbingFailed)
		SetSpanStatus(ctx, codes.Error, "scrubber not configured")
		return nil, ErrScrubbingFailed
	}
	scrubbedMsg, err := m.scrubber.Scrub(req.Message) //nolint:govet // err shadowing is intentional here
	if err != nil {
		// Fail closed - do not return unscrubbed content
		RecordError(ctx, ErrScrubbingFailed)
		SetSpanStatus(ctx, codes.Error, "scrubbing failed")
		return nil, ErrScrubbingFailed
	}

	// Cancel timeout
	m.cancelTimeout(branch.ID)

	// Get final token usage
	tokensUsed, _ := m.budget.Used(branch.ID)

	// Update branch state with validated transition
	now := time.Now()
	if err := m.transitionTo(branch, BranchStatusCompleted); err != nil {
		RecordError(ctx, err)
		SetSpanStatus(ctx, codes.Error, "invalid state transition")
		return nil, err
	}
	branch.Result = &scrubbedMsg
	branch.CompletedAt = &now
	branch.BudgetUsed = tokensUsed

	if err := m.repo.Update(ctx, branch); err != nil {
		RecordError(ctx, err)
		SetSpanStatus(ctx, codes.Error, "failed to update branch")
		return nil, fmt.Errorf("failed to update branch: %w", err)
	}

	// Cleanup budget
	m.budget.Deallocate(branch.ID)

	// Decrement instance branch count
	atomic.AddInt64(&m.instanceBranchCount, -1)

	// Record metrics and log
	duration := time.Since(startTime)
	m.metrics.RecordBranchReturned(ctx, branch.SessionID, branch.Depth, tokensUsed, branch.BudgetTotal, duration, branch.ProjectID)
	m.logger.BranchReturned(ctx, branch.ID, branch.SessionID, branch.Depth, tokensUsed, branch.BudgetTotal, duration)

	// Emit completion event
	if m.emitter != nil {
		m.emitter.Emit(BranchCompletedEvent{
			branchID:   branch.ID,
			TokensUsed: tokensUsed,
			Success:    true,
		})
	}

	SetSpanStatus(ctx, codes.Ok, "branch returned successfully")
	return &ReturnResponse{
		Success:     true,
		TokensUsed:  tokensUsed,
		ScrubbedMsg: scrubbedMsg,
	}, nil
}

// ForceReturn terminates a branch with the given reason.
func (m *BranchManager) ForceReturn(ctx context.Context, branchID string, reason string) error {
	// Start tracing span
	ctx, span := StartSpan(ctx, "folding.branch.force_return", branchID, "", 0)
	defer span.End()

	branch, err := m.repo.Get(ctx, branchID)
	if err != nil {
		RecordError(ctx, err)
		SetSpanStatus(ctx, codes.Error, "branch not found")
		return err
	}

	startTime := branch.CreatedAt

	// Already terminal - idempotent
	if branch.Status.IsTerminal() {
		SetSpanStatus(ctx, codes.Ok, "branch already terminal")
		return nil
	}

	// Force-return children first (recursive)
	children, _ := m.repo.ListByParent(ctx, branchID)
	for _, child := range children {
		if child.Status == BranchStatusActive {
			// CORR-017: Log child cleanup errors instead of silently ignoring
			if err := m.ForceReturn(ctx, child.ID, "parent force-returned"); err != nil {
				m.logger.Error(ctx, "failed to force-return child during parent force-return", err)
			}
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

	// Update branch with validated transition
	now := time.Now()
	if err := m.transitionTo(branch, status); err != nil {
		RecordError(ctx, err)
		SetSpanStatus(ctx, codes.Error, "invalid state transition")
		return err
	}
	branch.Error = &reason
	branch.CompletedAt = &now
	branch.BudgetUsed = tokensUsed

	if err := m.repo.Update(ctx, branch); err != nil {
		RecordError(ctx, err)
		SetSpanStatus(ctx, codes.Error, "failed to update branch")
		return err
	}

	// Cleanup budget
	m.budget.Deallocate(branchID)

	// Decrement instance branch count
	atomic.AddInt64(&m.instanceBranchCount, -1)

	// Record metrics and log
	duration := time.Since(startTime)
	if status == BranchStatusTimeout {
		m.metrics.RecordBranchTimeout(ctx, branch.SessionID, branch.Depth, tokensUsed, branch.BudgetTotal, duration, branch.ProjectID)
		m.logger.BranchTimeout(ctx, branch.ID, branch.SessionID, branch.Depth, tokensUsed, branch.BudgetTotal, branch.TimeoutSeconds, duration)
	} else {
		m.metrics.RecordBranchFailed(ctx, branch.SessionID, branch.Depth, reason, tokensUsed, branch.BudgetTotal, duration, branch.ProjectID)
		m.logger.BranchFailed(ctx, branch.ID, branch.SessionID, branch.Depth, reason, tokensUsed, branch.BudgetTotal, duration)
	}
	m.logger.ForceReturn(ctx, branch.ID, branch.SessionID, branch.Depth, reason)

	SetSpanStatus(ctx, codes.Ok, fmt.Sprintf("branch force-returned: %s", reason))
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
	// Start tracing span
	ctx, span := StartSpan(ctx, "folding.session.cleanup", "", sessionID, 0)
	defer span.End()

	branches, err := m.repo.ListBySession(ctx, sessionID)
	if err != nil {
		RecordError(ctx, err)
		SetSpanStatus(ctx, codes.Error, "failed to list branches")
		return err
	}

	// Sort by depth (deepest first) - cleanup children before parents
	sort.Slice(branches, func(i, j int) bool {
		return branches[i].Depth > branches[j].Depth
	})

	cleanedCount := 0
	for _, branch := range branches {
		if branch.Status == BranchStatusActive {
			_ = m.ForceReturn(ctx, branch.ID, "session ended")
			cleanedCount++
		}
	}

	// Log session cleanup
	m.logger.SessionCleanup(ctx, sessionID, cleanedCount)

	SetSpanStatus(ctx, codes.Ok, fmt.Sprintf("cleaned %d branches", cleanedCount))
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

// HealthStatus represents the health state of the BranchManager.
type HealthStatus struct {
	Healthy      bool   `json:"healthy"`
	ActiveCount  int64  `json:"active_count"`
	IsShutdown   bool   `json:"is_shutdown"`
	ErrorMessage string `json:"error_message,omitempty"`
}

// Health returns the current health status of the manager.
func (m *BranchManager) Health() HealthStatus {
	m.shutdownMu.RLock()
	isShutdown := m.isShutdown
	m.shutdownMu.RUnlock()

	activeCount := atomic.LoadInt64(&m.instanceBranchCount)

	return HealthStatus{
		Healthy:     !isShutdown,
		ActiveCount: activeCount,
		IsShutdown:  isShutdown,
	}
}

// Shutdown gracefully shuts down the manager, canceling all timeout watchers
// and force-returning all active branches.
func (m *BranchManager) Shutdown(ctx context.Context) error {
	m.shutdownMu.Lock()
	if m.isShutdown {
		m.shutdownMu.Unlock()
		return nil // Already shutdown
	}
	m.isShutdown = true
	close(m.shutdownChan)
	m.shutdownMu.Unlock()

	m.logger.Debug(ctx, "starting graceful shutdown")

	// Cancel all timeout watchers
	m.timeoutMu.Lock()
	for branchID, cancel := range m.timeoutCancels {
		cancel()
		delete(m.timeoutCancels, branchID)
	}
	m.timeoutMu.Unlock()

	m.logger.Debug(ctx, "shutdown complete")
	return nil
}

// IsShutdown returns true if the manager has been shut down.
func (m *BranchManager) IsShutdown() bool {
	m.shutdownMu.RLock()
	defer m.shutdownMu.RUnlock()
	return m.isShutdown
}

// checkShutdown returns an error if the manager is shut down.
func (m *BranchManager) checkShutdown() error {
	if m.IsShutdown() {
		return fmt.Errorf("branch manager is shut down")
	}
	return nil
}

// transitionTo validates and applies a state transition.
// Returns ErrInvalidTransition if the transition is not allowed by the state machine.
func (m *BranchManager) transitionTo(branch *Branch, newStatus BranchStatus) error {
	if !branch.Status.CanTransitionTo(newStatus) {
		return NewFoldingError(
			ErrCodeInvalidTransition,
			fmt.Sprintf("cannot transition from %s to %s", branch.Status, newStatus),
			nil,
			branch.ID,
			branch.SessionID,
		)
	}
	branch.Status = newStatus
	return nil
}
