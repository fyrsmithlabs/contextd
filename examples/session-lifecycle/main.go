// Package main demonstrates the session lifecycle pattern in contextd.
//
// This example shows the fundamental workflow:
// 1. Search for relevant past experiences
// 2. Perform task using learned strategies
// 3. Record new learnings
// 4. Provide feedback on usefulness
// 5. Record task outcome for confidence adjustment
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/fyrsmithlabs/contextd/internal/embeddings"
	"github.com/fyrsmithlabs/contextd/internal/reasoningbank"
	"github.com/fyrsmithlabs/contextd/internal/vectorstore"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

const (
	projectID = "session-lifecycle-demo"
	sessionID = "demo-session-001"
	tenant    = "demo-user"
)

func main() {
	fmt.Println("Session Lifecycle Example - Demonstrating search->do->record pattern")
	fmt.Println("=====================================================================\n")

	// Initialize components
	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatal(err)
	}
	defer logger.Sync()

	ctx := context.Background()

	// Create embeddings provider
	embedder, err := embeddings.NewProvider(embeddings.ProviderConfig{
		Provider: "fastembed",
		Model:    "BAAI/bge-small-en-v1.5",
		CacheDir: "/tmp/fastembed-cache",
	})
	if err != nil {
		log.Fatalf("Failed to create embedder: %v", err)
	}

	// Create in-memory vector store for demo
	store, err := vectorstore.NewChromemStore(vectorstore.ChromemConfig{
		Path:              "", // Empty path = in-memory
		DefaultCollection: "memories",
		VectorSize:        384,
	}, embedder, logger)
	if err != nil {
		log.Fatalf("Failed to create vector store: %v", err)
	}

	// Create ReasoningBank service
	service, err := reasoningbank.NewService(
		store,
		logger,
		reasoningbank.WithDefaultTenant(tenant),
		reasoningbank.WithEmbedder(embedder),
	)
	if err != nil {
		log.Fatalf("Failed to create service: %v", err)
	}

	// Run the session lifecycle demo
	if err := runSessionLifecycle(ctx, service); err != nil {
		log.Fatalf("Session lifecycle failed: %v", err)
	}

	fmt.Println("\n✓ Session complete! New memories are available for future sessions.")
}

// runSessionLifecycle demonstrates the complete search->do->record pattern.
func runSessionLifecycle(ctx context.Context, service *reasoningbank.Service) error {
	// Seed some existing memories to make the demo more realistic
	if err := seedMemories(ctx, service); err != nil {
		return fmt.Errorf("seeding memories: %w", err)
	}

	// Step 1: Search for existing memories
	fmt.Println("Step 1: Searching for existing memories about \"error handling in Go\"...")
	searchQuery := "error handling in Go"
	results, err := service.Search(ctx, projectID, searchQuery, 5)
	if err != nil {
		return fmt.Errorf("searching memories: %w", err)
	}

	if len(results) == 0 {
		fmt.Println("No relevant memories found. Starting fresh.")
	} else {
		fmt.Printf("Found %d relevant memories:\n", len(results))
		for _, memory := range results {
			fmt.Printf("  - [ID: %s] %s (confidence: %.2f)\n",
				memory.ID[:8], memory.Title, memory.Confidence)
		}
	}
	fmt.Println()

	// Step 2: Perform task (simulated)
	fmt.Println("Step 2: Performing task using retrieved strategies...")
	fmt.Println("Task: Implementing error handling based on past learnings")

	var appliedMemoryID string
	if len(results) > 0 {
		appliedMemoryID = results[0].ID
		fmt.Printf("✓ Applied strategy: %s\n\n", results[0].Title)
	} else {
		fmt.Println("✓ Developed new approach (no prior memories found)\n")
	}

	// Simulate task execution time
	time.Sleep(500 * time.Millisecond)

	// Step 3: Record new learning
	fmt.Println("Step 3: Recording new learning...")
	newMemory := &reasoningbank.Memory{
		ID:          uuid.New().String(),
		ProjectID:   projectID,
		Title:       "Use %w verb for error wrapping",
		Content:     "When wrapping errors in Go, use %w verb instead of %v to preserve error chain for errors.Is/As. Example: return fmt.Errorf(\"doing work: %w\", err)",
		Description: "Learned from session: error-handling-task",
		Outcome:     reasoningbank.OutcomeSuccess,
		Tags:        []string{"go", "errors", "best-practice"},
		Confidence:  0.8,
		State:       reasoningbank.MemoryStateActive,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := service.Record(ctx, newMemory); err != nil {
		return fmt.Errorf("recording memory: %w", err)
	}
	fmt.Printf("✓ Recorded memory: \"%s\" (ID: %s)\n\n", newMemory.Title, newMemory.ID[:8])

	// Step 4: Provide feedback (optional but recommended)
	if appliedMemoryID != "" {
		fmt.Println("Step 4: Providing feedback on helpful memory...")
		err := service.Feedback(ctx, appliedMemoryID, true)
		if err != nil {
			// Don't fail the whole demo on feedback error
			fmt.Printf("⚠ Warning: feedback failed: %v\n\n", err)
		} else {
			fmt.Printf("✓ Marked memory %s as helpful\n\n", appliedMemoryID[:8])
		}
	} else {
		fmt.Println("Step 4: Skipping feedback (no memories were applied)\n")
	}

	// Step 5: Record outcome
	if appliedMemoryID != "" {
		fmt.Println("Step 5: Recording successful outcome...")
		taskSucceeded := true // In real usage, this would be based on actual task result

		newConfidence, err := service.RecordOutcome(ctx, appliedMemoryID, taskSucceeded, sessionID)
		if err != nil {
			fmt.Printf("⚠ Warning: outcome recording failed: %v\n\n", err)
		} else {
			fmt.Printf("✓ Task succeeded using memory %s (confidence updated to %.2f)\n",
				appliedMemoryID[:8], newConfidence)
		}
	} else {
		fmt.Println("Step 5: No outcome to record (no memories were used)")
	}

	return nil
}

// seedMemories creates some initial memories to make the demo more realistic.
// In a real scenario, these would come from previous sessions.
func seedMemories(ctx context.Context, service *reasoningbank.Service) error {
	memories := []*reasoningbank.Memory{
		{
			ID:          uuid.New().String(),
			ProjectID:   projectID,
			Title:       "Always wrap errors with context",
			Content:     "When returning errors, always add context about what operation failed. Use fmt.Errorf with %w to wrap the original error.",
			Description: "Core Go error handling pattern",
			Outcome:     reasoningbank.OutcomeSuccess,
			Tags:        []string{"go", "errors", "pattern"},
			Confidence:  0.85,
			UsageCount:  5,
			State:       reasoningbank.MemoryStateActive,
			CreatedAt:   time.Now().Add(-7 * 24 * time.Hour),
			UpdatedAt:   time.Now().Add(-24 * time.Hour),
		},
		{
			ID:          uuid.New().String(),
			ProjectID:   projectID,
			Title:       "Use errors.Is for error comparison",
			Content:     "Don't use == to compare errors. Use errors.Is for sentinel errors and errors.As for type assertions. This works with wrapped errors.",
			Description: "Modern Go error comparison",
			Outcome:     reasoningbank.OutcomeSuccess,
			Tags:        []string{"go", "errors", "comparison"},
			Confidence:  0.78,
			UsageCount:  3,
			State:       reasoningbank.MemoryStateActive,
			CreatedAt:   time.Now().Add(-5 * 24 * time.Hour),
			UpdatedAt:   time.Now().Add(-48 * time.Hour),
		},
		{
			ID:          uuid.New().String(),
			ProjectID:   projectID,
			Title:       "Don't use panic for expected errors",
			Content:     "Reserve panic for truly exceptional situations (programmer errors). Expected errors should be returned as error values, not panicked.",
			Description: "Anti-pattern learned from debugging session",
			Outcome:     reasoningbank.OutcomeFailure, // This is an anti-pattern
			Tags:        []string{"go", "errors", "anti-pattern"},
			Confidence:  0.72,
			UsageCount:  2,
			State:       reasoningbank.MemoryStateActive,
			CreatedAt:   time.Now().Add(-10 * 24 * time.Hour),
			UpdatedAt:   time.Now().Add(-72 * time.Hour),
		},
	}

	for _, memory := range memories {
		if err := service.Record(ctx, memory); err != nil {
			return fmt.Errorf("seeding memory %s: %w", memory.ID, err)
		}
	}

	return nil
}
