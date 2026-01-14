// Package main demonstrates the checkpoint lifecycle pattern in contextd.
//
// This example shows the fundamental workflow:
// 1. Save checkpoints at strategic points
// 2. List available checkpoints
// 3. Resume from checkpoint at chosen granularity level
// 4. Manage token budgets with resume levels
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/fyrsmithlabs/contextd/internal/checkpoint"
	"github.com/fyrsmithlabs/contextd/internal/embeddings"
	"github.com/fyrsmithlabs/contextd/internal/vectorstore"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

const (
	projectID   = "checkpoints-demo"
	sessionID   = "demo-session-001"
	tenant      = "demo-user"
	teamID      = "demo-team"
	projectPath = "/demo/project"
)

func main() {
	fmt.Println("Checkpoints Example - Demonstrating save->list->resume workflow")
	fmt.Println("===============================================================\n")

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
		DefaultCollection: "checkpoints",
		VectorSize:        384,
	}, embedder, logger)
	if err != nil {
		log.Fatalf("Failed to create vector store: %v", err)
	}

	// Wrap single store as StoreProvider for checkpoint service
	storeProvider := &mockStoreProvider{store: store}

	// Create checkpoint service
	service, err := checkpoint.NewService(checkpoint.DefaultServiceConfig(), storeProvider, logger)
	if err != nil {
		log.Fatalf("Failed to create checkpoint service: %v", err)
	}
	defer service.Close()

	// Run the checkpoint lifecycle demo
	if err := runCheckpointLifecycle(ctx, service); err != nil {
		log.Fatalf("Checkpoint lifecycle failed: %v", err)
	}

	fmt.Println("\n✓ Checkpoint workflow complete!")
}

// mockStoreProvider implements vectorstore.StoreProvider for the demo.
// In production, use the real StoreProvider that manages per-project stores.
type mockStoreProvider struct {
	store vectorstore.Store
}

func (m *mockStoreProvider) GetProjectStore(ctx context.Context, tenantID, teamID, projectID string) (vectorstore.Store, error) {
	return m.store, nil
}

func (m *mockStoreProvider) GetTeamStore(ctx context.Context, tenantID, teamID string) (vectorstore.Store, error) {
	return m.store, nil
}

func (m *mockStoreProvider) GetOrgStore(ctx context.Context, tenantID string) (vectorstore.Store, error) {
	return m.store, nil
}

func (m *mockStoreProvider) Close() error {
	return m.store.Close()
}

// runCheckpointLifecycle demonstrates the complete save->list->resume pattern.
func runCheckpointLifecycle(ctx context.Context, service checkpoint.Service) error {
	var checkpoint1ID, checkpoint2ID string

	// Step 1: Simulate work and save first checkpoint
	fmt.Println("Step 1: Simulating work session (Task A)...")
	fmt.Println("Working on user authentication...")
	time.Sleep(300 * time.Millisecond)
	fmt.Println("✓ Completed Task A (context: 1200 tokens)\n")

	// Step 2: Save checkpoint after Task A
	fmt.Println("Step 2: Saving checkpoint \"After Task A\"...")
	saveReq1 := &checkpoint.SaveRequest{
		SessionID:   sessionID,
		TenantID:    tenant,
		TeamID:      teamID,
		ProjectID:   projectID,
		ProjectPath: projectPath,
		Name:        "After Task A",
		Description: "Completed user authentication implementation",
		Summary:     "Implemented OAuth2 flow with JWT tokens. User service handles login, logout, and token refresh.",
		Context:     "OAuth2 configuration: client_id from env, redirect_uri to /auth/callback. JWT tokens expire in 1 hour, refresh tokens in 7 days. User service has endpoints: POST /login, POST /logout, POST /refresh.",
		FullState:   "Full conversation history would go here... (1200 tokens)\nUser: Implement authentication\nAssistant: I'll implement OAuth2...\n[... full conversation ...]",
		TokenCount:  1200,
		Threshold:   0.24,
		AutoCreated: false,
		Metadata: map[string]string{
			"phase":   "implementation",
			"feature": "authentication",
		},
	}

	cp1, err := service.Save(ctx, saveReq1)
	if err != nil {
		return fmt.Errorf("saving checkpoint 1: %w", err)
	}
	checkpoint1ID = cp1.ID
	fmt.Printf("✓ Saved checkpoint: \"%s\" (ID: %s)\n\n", cp1.Name, cp1.ID[:8])

	// Step 3: Continue work
	fmt.Println("Step 3: Continuing work (Task B)...")
	fmt.Println("Implementing user authorization...")
	time.Sleep(300 * time.Millisecond)
	fmt.Println("✓ Completed Task B (context: 2800 tokens)\n")

	// Step 4: Save checkpoint after Task B
	fmt.Println("Step 4: Saving checkpoint \"After Task B\"...")
	saveReq2 := &checkpoint.SaveRequest{
		SessionID:   sessionID,
		TenantID:    tenant,
		TeamID:      teamID,
		ProjectID:   projectID,
		ProjectPath: projectPath,
		Name:        "After Task B",
		Description: "Completed authorization with role-based access control",
		Summary:     "Added RBAC with three roles: admin, user, guest. Authorization middleware checks JWT claims for roles. Protected endpoints require specific roles.",
		Context:     "RBAC implementation: roles stored in JWT claims as array. Middleware function `requireRole(role)` extracts JWT, checks roles array. Admin role has full access, user role has read/write, guest role has read-only. Protected routes use middleware: app.post('/admin/*', requireRole('admin'), handler).",
		FullState:   "Full conversation history would go here... (2800 tokens)\nUser: Implement authentication\nAssistant: I'll implement OAuth2...\nUser: Now add authorization\nAssistant: I'll add RBAC...\n[... full conversation ...]",
		TokenCount:  2800,
		Threshold:   0.56,
		AutoCreated: false,
		Metadata: map[string]string{
			"phase":   "implementation",
			"feature": "authorization",
		},
	}

	cp2, err := service.Save(ctx, saveReq2)
	if err != nil {
		return fmt.Errorf("saving checkpoint 2: %w", err)
	}
	checkpoint2ID = cp2.ID
	fmt.Printf("✓ Saved checkpoint: \"%s\" (ID: %s)\n\n", cp2.Name, cp2.ID[:8])

	// Step 5: List checkpoints
	fmt.Println("Step 5: Listing available checkpoints...")
	listReq := &checkpoint.ListRequest{
		SessionID: sessionID,
		TenantID:  tenant,
		TeamID:    teamID,
		ProjectID: projectID,
		Limit:     10,
		AutoOnly:  false,
	}

	checkpoints, err := service.List(ctx, listReq)
	if err != nil {
		return fmt.Errorf("listing checkpoints: %w", err)
	}

	fmt.Printf("Found %d checkpoints:\n", len(checkpoints))
	for _, cp := range checkpoints {
		fmt.Printf("  - [%s] %s (%d tokens, auto: %v)\n",
			cp.ID[:8], cp.Name, cp.TokenCount, cp.AutoCreated)
	}
	fmt.Println()

	// Step 6: Simulate session interruption
	fmt.Println("Step 6: Simulating session interruption...")
	fmt.Println("Session interrupted! Context lost.\n")
	time.Sleep(200 * time.Millisecond)

	// Step 7: Resume from checkpoint at different levels
	fmt.Println("Step 7: Resuming from checkpoint at 'summary' level...")
	if err := demonstrateResumeLevel(ctx, service, checkpoint2ID, checkpoint.ResumeSummary); err != nil {
		return err
	}

	fmt.Println("\nStep 8: Resuming from checkpoint at 'context' level...")
	if err := demonstrateResumeLevel(ctx, service, checkpoint2ID, checkpoint.ResumeContext); err != nil {
		return err
	}

	fmt.Println("\nStep 9: Resuming from checkpoint at 'full' level...")
	if err := demonstrateResumeLevel(ctx, service, checkpoint2ID, checkpoint.ResumeFull); err != nil {
		return err
	}

	// Step 10: Demonstrate checkpoint filtering
	fmt.Println("\nStep 10: Demonstrating checkpoint filtering...")
	if err := demonstrateFiltering(ctx, service); err != nil {
		return err
	}

	// Cleanup: Demonstrate delete
	fmt.Println("\nStep 11: Cleaning up checkpoints...")
	if err := service.Delete(ctx, tenant, teamID, projectID, checkpoint1ID); err != nil {
		fmt.Printf("⚠ Warning: failed to delete checkpoint: %v\n", err)
	} else {
		fmt.Printf("✓ Deleted checkpoint %s\n", checkpoint1ID[:8])
	}

	return nil
}

// demonstrateResumeLevel shows resuming at a specific level.
func demonstrateResumeLevel(ctx context.Context, service checkpoint.Service, checkpointID string, level checkpoint.ResumeLevel) error {
	resumeReq := &checkpoint.ResumeRequest{
		CheckpointID: checkpointID,
		TenantID:     tenant,
		TeamID:       teamID,
		ProjectID:    projectID,
		Level:        level,
	}

	response, err := service.Resume(ctx, resumeReq)
	if err != nil {
		return fmt.Errorf("resuming checkpoint: %w", err)
	}

	fmt.Printf("✓ Resumed checkpoint: \"%s\"\n", response.Checkpoint.Name)
	fmt.Printf("  Level: %s\n", level)
	fmt.Printf("  Restored tokens: %d\n", response.TokenCount)
	fmt.Printf("  Content length: %d chars\n", len(response.Content))

	// Show preview of content
	preview := response.Content
	if len(preview) > 150 {
		preview = preview[:150] + "..."
	}
	fmt.Printf("  Preview: %s\n", preview)

	return nil
}

// demonstrateFiltering shows different filtering options.
func demonstrateFiltering(ctx context.Context, service checkpoint.Service) error {
	// Create an auto-checkpoint for filtering demo
	autoCheckpoint := &checkpoint.SaveRequest{
		SessionID:   sessionID,
		TenantID:    tenant,
		TeamID:      teamID,
		ProjectID:   projectID,
		ProjectPath: projectPath,
		Name:        "Auto-checkpoint at 75%",
		Description: "Automatically created at 75% threshold",
		Summary:     "Auto-saved context at 75% capacity",
		Context:     "Automatic checkpoint created by threshold trigger",
		FullState:   "Auto checkpoint full state...",
		TokenCount:  3750,
		Threshold:   0.75,
		AutoCreated: true, // This is an auto-checkpoint
	}

	autoCP, err := service.Save(ctx, autoCheckpoint)
	if err != nil {
		return fmt.Errorf("saving auto checkpoint: %w", err)
	}

	fmt.Printf("Created auto-checkpoint for demo (ID: %s)\n\n", autoCP.ID[:8])

	// Filter 1: All checkpoints
	fmt.Println("Filter 1: All checkpoints for this session")
	allReq := &checkpoint.ListRequest{
		SessionID: sessionID,
		TenantID:  tenant,
		TeamID:    teamID,
		ProjectID: projectID,
		Limit:     20,
		AutoOnly:  false,
	}

	allCPs, err := service.List(ctx, allReq)
	if err != nil {
		return fmt.Errorf("listing all checkpoints: %w", err)
	}
	fmt.Printf("  Found %d total checkpoints\n\n", len(allCPs))

	// Filter 2: Auto-checkpoints only
	fmt.Println("Filter 2: Auto-checkpoints only")
	autoReq := &checkpoint.ListRequest{
		SessionID: sessionID,
		TenantID:  tenant,
		TeamID:    teamID,
		ProjectID: projectID,
		Limit:     20,
		AutoOnly:  true,
	}

	autoCPs, err := service.List(ctx, autoReq)
	if err != nil {
		return fmt.Errorf("listing auto checkpoints: %w", err)
	}
	fmt.Printf("  Found %d auto-checkpoints\n", len(autoCPs))
	for _, cp := range autoCPs {
		fmt.Printf("    - [%s] %s (threshold: %.0f%%)\n",
			cp.ID[:8], cp.Name, cp.Threshold*100)
	}
	fmt.Println()

	// Filter 3: By project path (simulated - all our checkpoints have same path)
	fmt.Println("Filter 3: By project path")
	pathReq := &checkpoint.ListRequest{
		TenantID:    tenant,
		TeamID:      teamID,
		ProjectID:   projectID,
		ProjectPath: projectPath,
		Limit:       20,
	}

	pathCPs, err := service.List(ctx, pathReq)
	if err != nil {
		return fmt.Errorf("listing by path: %w", err)
	}
	fmt.Printf("  Found %d checkpoints for path: %s\n", len(pathCPs), projectPath)

	return nil
}

// Additional utility for examples

// generateCheckpointID creates a unique checkpoint ID.
func generateCheckpointID() string {
	return uuid.New().String()
}

// estimateTokensFromContent provides a rough token estimate.
func estimateTokensFromContent(summary, context, fullState string) int32 {
	totalChars := len(summary) + len(context) + len(fullState)
	// Rough estimate: ~4 chars per token
	return int32(totalChars / 4)
}
