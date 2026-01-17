// Package reasoningbank provides cross-session memory storage and retrieval for AI agents.
//
// The package stores memories (learned strategies and patterns) in a vector database,
// enabling semantic search to surface relevant knowledge based on similarity to the
// current task. Memories track their usefulness through a Bayesian confidence system
// that learns from usage signals, explicit feedback, and task outcomes.
//
// # Core Concepts
//
// Memories are distilled strategies learned from agent interactions. Each memory has:
//   - Title and content describing the strategy or pattern
//   - Outcome: "success" (pattern to follow) or "failure" (anti-pattern to avoid)
//   - Confidence score (0.0-1.0) adjusted by feedback and usage
//   - Project isolation via database-per-project architecture
//
// # Confidence System
//
// The Bayesian confidence system learns which signals predict memory usefulness:
//   - Explicit feedback: User ratings (helpful/unhelpful)
//   - Usage signals: Memory retrieved in semantic search
//   - Outcome signals: Task succeeded/failed after using memory
//
// The system learns signal weights per-project, adapting to each codebase's patterns.
// Memories below MinConfidence (0.7) are filtered from search results.
//
// # Memory Consolidation
//
// The Distiller detects similar memories and consolidates them into synthesized knowledge:
//   - Finds clusters of memories above similarity threshold (0.8 default)
//   - Uses LLM to merge clusters into consolidated memories
//   - Archives source memories with back-links for attribution
//   - Consolidated memories receive 20% similarity boost in search
//
// # Security
//
// The package implements defense-in-depth security:
//   - Multi-tenant isolation via payload-based filtering
//   - Fail-closed: operations require tenant context
//   - Per-project vectorstore databases (StoreProvider)
//   - Filter injection prevention (tenant_id in user filters rejected)
//
// # Usage
//
// Basic memory recording and search:
//
//	svc, err := reasoningbank.NewServiceWithStoreProvider(stores, "username", logger)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Record a new memory
//	memory, err := reasoningbank.NewMemory(
//	    "projectID",
//	    "Go error handling with context",
//	    "Always wrap errors with context using fmt.Errorf with %w verb",
//	    reasoningbank.OutcomeSuccess,
//	    []string{"go", "errors"},
//	)
//	if err := svc.Record(ctx, memory); err != nil {
//	    log.Fatal(err)
//	}
//
//	// Search for relevant memories
//	memories, err := svc.Search(ctx, "projectID", "how to handle errors", 5)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for _, mem := range memories {
//	    fmt.Printf("%.2f: %s\n", mem.Confidence, mem.Title)
//	}
//
//	// Record feedback to improve confidence
//	if err := svc.Feedback(ctx, memories[0].ID, true); err != nil {
//	    log.Fatal(err)
//	}
//
// # MCP Integration
//
// The package is exposed via MCP tools:
//   - memory_search: Find relevant memories by semantic similarity
//   - memory_record: Save new memory explicitly (bypasses distillation)
//   - memory_feedback: Rate memory helpfulness (helpful/unhelpful)
//   - memory_outcome: Report task success/failure after using memory
//
// See CLAUDE.md for MCP tool usage patterns.
package reasoningbank
