// Package orchestrator provides agent orchestration with phase gates and workflow enforcement.
//
// # Overview
//
// The orchestrator implements a structured approach to agent task execution that enforces
// TDD compliance, sequential processing, and automatic memory recording via contextd.
//
// # Architecture
//
// The orchestrator follows a phase-based execution model:
//
//	Init → Test → Implement → Verify → Commit → Report
//
// Each phase transition is guarded by configurable gates that validate workflow compliance.
//
// # Key Components
//
// ## Executor
//
// The Executor is the main entry point for task orchestration. It manages:
//   - Phase handler registration
//   - Gate registration and validation
//   - Progress reporting
//   - Memory recording via contextd
//
// ## Phase Gates
//
// Gates are validators that run before phase transitions:
//   - TDDGate: Ensures tests are written before implementation
//   - VerificationGate: Validates actual tests ran (not --help output)
//   - CommitGate: Enforces separate test and implementation commits
//   - SequentialGate: Warns against bundled changes
//   - StatusReportGate: Ensures phases report their status
//
// ## Violations
//
// The system detects and reports workflow violations:
//   - ViolationTDDNotFollowed: Implementation without tests
//   - ViolationPhaseSkipped: Attempting to skip required phases
//   - ViolationHelpAsVerification: Using --help output as fake tests
//   - ViolationTestsNotRun: No actual test execution
//   - ViolationBundledChanges: Too many changes in single phase
//   - ViolationCommitMixedContent: Mixed test/impl in single commit
//
// # Usage Example
//
//	// Create executor with contextd integration
//	mcpClient := ...  // Your MCP client
//	recorder := orchestrator.NewContextdRecorder(mcpClient)
//	executor := orchestrator.NewExecutor(claudeClient, recorder)
//
//	// Register gates
//	executor.RegisterGate(orchestrator.PhaseImplement, orchestrator.NewTDDGate())
//	executor.RegisterGate(orchestrator.PhaseCommit, orchestrator.NewVerificationGate())
//
//	// Register phase handlers
//	executor.RegisterHandler(myInitHandler)
//	executor.RegisterHandler(myTestHandler)
//	// ... register all handlers
//
//	// Execute task
//	config := orchestrator.TaskConfig{
//	    ID:          "task-001",
//	    Description: "Add new feature",
//	    EnforceTDD:  true,
//	}
//	state, err := executor.Execute(ctx, config)
//
// # contextd Integration
//
// The orchestrator integrates with contextd MCP tools:
//   - memory_search/memory_record: Cross-session learning persistence
//   - remediation_search/remediation_record: Error pattern tracking
//   - checkpoint_save/checkpoint_resume: Context preservation
//   - memory_feedback: Confidence adjustment based on outcomes
//
// # Success Criteria (from Issue #20)
//
//   - TDD compliance with distinct test and implementation commits
//   - Enforced phase gates preventing step skipping
//   - All decisions recorded in contextd memory
//   - Clear phase-level progress feedback
//   - Detection and reporting of workflow violations
//
// # Design Decisions
//
// 1. Direct API Approach: Uses Anthropic Go SDK directly for full control
//    over the agent loop, rather than subprocess-based CLI calls.
//
// 2. Interface-Based: All major components (ClaudeClient, MemoryRecorder,
//    PhaseHandler, PhaseGate) are defined as interfaces for testability.
//
// 3. Gate-First: Gates run before phase execution, not after, to prevent
//    violations rather than just detect them.
//
// 4. Progressive Severity: Violations have severity levels (warning, error,
//    critical) allowing flexible enforcement policies.
//
// 5. contextd-First: All learnings and violations are recorded to contextd
//    memory for cross-session benefit.
package orchestrator
