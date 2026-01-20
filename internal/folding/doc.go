// Package folding provides context-folding for AI agent context management.
//
// Context-folding enables AI agents to create isolated execution branches with
// dedicated token budgets. Each branch executes in isolation, and only a scrubbed
// summary returns to the parent context. This achieves 90%+ context compression
// by isolating verbose subtask reasoning from the main conversation context.
//
// # Core Concepts
//
// Branch: An isolated context with its own token budget and timeout. Branches
// can nest up to a configurable depth (default: 3 levels). Each branch tracks
// its token consumption and automatically terminates when budget is exhausted
// or timeout is reached.
//
// Budget: Token allocation for a branch. Budget tracking uses a centralized
// BudgetTracker that emits events at 80% usage (warning) and 100% (exhaustion).
// Budgets are capped at a maximum size (default: 32,768 tokens).
//
// Lifecycle: Branches transition through states: created → active → completed/timeout/failed.
// State transitions are validated by a state machine to prevent invalid operations.
//
// # Security
//
// The package implements defense-in-depth security:
//   - Input validation (SEC-001): Length limits on descriptions, prompts, and return messages
//   - Secret scrubbing (SEC-002): All return messages are scrubbed for secrets using gitleaks
//   - Rate limiting (SEC-003): Per-session and instance-level concurrent branch limits
//   - Session authorization (SEC-004): Caller identity validation via SessionValidator interface
//   - Multi-tenant isolation: Project-scoped metrics and session-based access control
//
// # Usage
//
// Basic branch creation and return:
//
//	config := folding.DefaultFoldingConfig()
//	manager := folding.NewBranchManager(repo, budget, scrubber, emitter, config)
//
//	// Create branch
//	req := folding.BranchRequest{
//	    SessionID:   "session_123",
//	    Description: "Search for function definition",
//	    Prompt:      "Find the authenticate() function in src/",
//	    Budget:      4096,
//	}
//	resp, err := manager.Create(ctx, req)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// ... agent performs work in branch context ...
//
//	// Return from branch
//	returnReq := folding.ReturnRequest{
//	    BranchID: resp.BranchID,
//	    Message:  "Found in src/auth.go:42",
//	}
//	returnResp, err := manager.Return(ctx, returnReq)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Used %d tokens, result: %s\n", returnResp.TokensUsed, returnResp.ScrubbedMsg)
//
// # Use Cases
//
// Context-folding is ideal for:
//   - File exploration: Read multiple files, return only relevant findings
//   - API research: Fetch documentation, return only applicable excerpts
//   - Trial-and-error debugging: Try multiple fixes, return only successful approach
//   - Batch operations: Process many items, return summary statistics
//
// Context-folding should NOT be used for:
//   - Single-file changes (no benefit)
//   - Tasks where full reasoning must be visible to user
//   - Already-focused tasks with minimal context overhead
//
// # Event System
//
// The package emits lifecycle events via the EventEmitter interface:
//   - BudgetWarningEvent: Emitted at 80% budget usage
//   - BudgetExhaustedEvent: Emitted when budget is fully consumed
//   - TimeoutEvent: Emitted when branch exceeds its timeout
//   - BranchCompletedEvent: Emitted when branch returns successfully
//
// Subscribe to events for monitoring and telemetry:
//
//	emitter.Subscribe(func(event folding.BranchEvent) {
//	    switch e := event.(type) {
//	    case folding.BudgetWarningEvent:
//	        log.Printf("Branch %s at %0.1f%% budget\n", e.BranchID(), e.Percentage)
//	    case folding.BudgetExhaustedEvent:
//	        log.Printf("Branch %s exhausted budget\n", e.BranchID())
//	    }
//	})
//
// # Shutdown and Cleanup
//
// BranchManager supports graceful shutdown via the Shutdown() method. This:
//   - Prevents new branch creation
//   - Cancels all active timeout watchers
//   - Returns a clean shutdown signal
//
// Sessions can be cleaned up individually via CleanupSession(), which force-returns
// all active branches for a given session. This is useful for session lifecycle hooks.
//
// # Performance
//
// Current implementation:
//   - In-memory budget tracking with O(1) operations
//   - Goroutine-per-branch for timeout enforcement
//   - Atomic counters for rate limiting
//
// Scalability considerations:
//   - Default max 10 concurrent branches per session
//   - Default max 100 concurrent branches per instance
//   - Timeout goroutines are cleaned up on branch completion
//   - Token counting is delegated to TokenCounter interface
//
// # Multi-Tenancy
//
// The package supports multi-tenant deployments via project scoping:
//   - Metrics are tagged with project_id for per-project monitoring
//   - Session authorization via SessionValidator interface
//   - No cross-session data leakage (branches are session-scoped)
//
// Session validation can be configured:
//   - PermissiveSessionValidator: Allows all access (single-user deployments)
//   - StrictSessionValidator: Requires session ownership match (multi-user)
//   - Custom validators can implement the SessionValidator interface
package folding
