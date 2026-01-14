// Package main demonstrates the error remediation pattern in contextd.
//
// This example shows the fundamental workflow:
// 1. Record error fix patterns (problem, root cause, solution)
// 2. Search for similar errors when they occur again
// 3. Provide feedback to improve confidence scores
// 4. Reuse proven solutions across projects and teams
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/fyrsmithlabs/contextd/internal/embeddings"
	"github.com/fyrsmithlabs/contextd/internal/remediation"
	"github.com/fyrsmithlabs/contextd/internal/vectorstore"
	"go.uber.org/zap"
)

const (
	tenant      = "demo-org"
	teamID      = "backend-team"
	projectPath = "/demo/api-service"
	sessionID   = "debug-session-001"
)

func main() {
	fmt.Println("Remediation Example - Demonstrating record->search->reuse workflow")
	fmt.Println("===================================================================\n")

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
		DefaultCollection: "remediations",
		VectorSize:        384,
	}, embedder, logger)
	if err != nil {
		log.Fatalf("Failed to create vector store: %v", err)
	}

	// Wrap single store as StoreProvider for remediation service
	storeProvider := &mockStoreProvider{store: store}

	// Create remediation service
	service, err := remediation.NewServiceWithStoreProvider(
		remediation.DefaultServiceConfig(),
		storeProvider,
		logger,
	)
	if err != nil {
		log.Fatalf("Failed to create remediation service: %v", err)
	}
	defer service.Close()

	// Run the remediation lifecycle demo
	if err := runRemediationLifecycle(ctx, service); err != nil {
		log.Fatalf("Remediation lifecycle failed: %v", err)
	}

	fmt.Println("\n✓ Remediation workflow complete!")
}

// mockStoreProvider implements vectorstore.StoreProvider for the demo.
// In production, use the real StoreProvider that manages per-scope stores.
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

// runRemediationLifecycle demonstrates the complete record->search->feedback pattern.
func runRemediationLifecycle(ctx context.Context, service remediation.Service) error {
	// Step 1: Record first error fix (nil pointer dereference)
	fmt.Println("Step 1: Recording error fix - Nil pointer dereference...")
	nilPointerFix := &remediation.RecordRequest{
		Title:    "Nil pointer dereference in user handler",
		Problem:  "Application crashes with panic: runtime error: invalid memory address or nil pointer dereference",
		Symptoms: []string{
			"Server returns 500 error",
			"Panic stack trace shows user.go:45",
			"Only happens when user is not authenticated",
		},
		RootCause: "User middleware doesn't check if context.User is nil before passing to handler",
		Solution:  "Add nil check in middleware before calling handler. Return 401 if user is nil.",
		CodeDiff: `diff --git a/middleware/auth.go b/middleware/auth.go
--- a/middleware/auth.go
+++ b/middleware/auth.go
@@ -10,6 +10,10 @@ func AuthMiddleware(next http.Handler) http.Handler {
     return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
         user := getUserFromContext(r.Context())
+        if user == nil {
+            http.Error(w, "Unauthorized", http.StatusUnauthorized)
+            return
+        }
         next.ServeHTTP(w, r)
     })
 }`,
		AffectedFiles: []string{
			"middleware/auth.go",
			"handlers/user.go",
		},
		Category:    remediation.ErrorRuntime,
		Tags:        []string{"panic", "authentication", "middleware"},
		Scope:       remediation.ScopeProject,
		TenantID:    tenant,
		TeamID:      teamID,
		ProjectPath: projectPath,
		SessionID:   sessionID,
		Confidence:  0.5, // Default initial confidence
	}

	rem1, err := service.Record(ctx, nilPointerFix)
	if err != nil {
		return fmt.Errorf("recording remediation 1: %w", err)
	}
	fmt.Printf("✓ Recorded remediation: \"%s\" (ID: %s)\n\n", rem1.Title, rem1.ID[:8])

	// Step 2: Record second error fix (database connection)
	fmt.Println("Step 2: Recording error fix - Database connection timeout...")
	dbTimeoutFix := &remediation.RecordRequest{
		Title:    "Database connection pool exhaustion",
		Problem:  "API requests timeout after 30 seconds with error: pq: connection pool exhausted",
		Symptoms: []string{
			"Slow response times during peak traffic",
			"Database connection timeout errors in logs",
			"Connection pool size reaches max",
		},
		RootCause: "Connection pool size too small (10 connections) and connections not being closed properly",
		Solution:  "Increase pool size to 50 and ensure all DB queries use defer rows.Close()",
		CodeDiff: `diff --git a/db/pool.go b/db/pool.go
--- a/db/pool.go
+++ b/db/pool.go
@@ -5,7 +5,7 @@ func NewPool() *sql.DB {
     db, err := sql.Open("postgres", dsn)
     // ...
-    db.SetMaxOpenConns(10)
+    db.SetMaxOpenConns(50)
     db.SetMaxIdleConns(10)
     db.SetConnMaxLifetime(time.Hour)
     return db
 }`,
		AffectedFiles: []string{
			"db/pool.go",
			"handlers/orders.go",
			"handlers/products.go",
		},
		Category:    remediation.ErrorPerformance,
		Tags:        []string{"database", "performance", "connection-pool"},
		Scope:       remediation.ScopeTeam, // Share across team
		TenantID:    tenant,
		TeamID:      teamID,
		SessionID:   sessionID,
	}

	rem2, err := service.Record(ctx, dbTimeoutFix)
	if err != nil {
		return fmt.Errorf("recording remediation 2: %w", err)
	}
	fmt.Printf("✓ Recorded remediation: \"%s\" (ID: %s)\n\n", rem2.Title, rem2.ID[:8])

	// Step 3: Record third error fix (test failure)
	fmt.Println("Step 3: Recording error fix - Test flakiness...")
	testFlakeFix := &remediation.RecordRequest{
		Title:    "Flaky integration test: TestUserLogin",
		Problem:  "Test intermittently fails with 'context deadline exceeded'",
		Symptoms: []string{
			"Test passes locally but fails in CI",
			"Failure rate around 20%",
			"Always times out at same assertion",
		},
		RootCause: "Test uses real database with 5 second timeout, but CI database is slower",
		Solution:  "Increase test timeout to 30 seconds and use database transactions for isolation",
		CodeDiff: `diff --git a/handlers/user_test.go b/handlers/user_test.go
--- a/handlers/user_test.go
+++ b/handlers/user_test.go
@@ -10,7 +10,7 @@ func TestUserLogin(t *testing.T) {
-    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
+    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
     defer cancel()

     // Use transaction for test isolation
+    tx, _ := db.BeginTx(ctx, nil)
+    defer tx.Rollback()`,
		AffectedFiles: []string{
			"handlers/user_test.go",
		},
		Category:    remediation.ErrorTest,
		Tags:        []string{"testing", "flaky-test", "ci"},
		Scope:       remediation.ScopeOrg, // Share across org
		TenantID:    tenant,
		SessionID:   sessionID,
	}

	rem3, err := service.Record(ctx, testFlakeFix)
	if err != nil {
		return fmt.Errorf("recording remediation 3: %w", err)
	}
	fmt.Printf("✓ Recorded remediation: \"%s\" (ID: %s)\n\n", rem3.Title, rem3.ID[:8])

	// Step 4: Search for similar error (similar to first one)
	fmt.Println("Step 4: Encountering similar error...")
	fmt.Println("Error: panic: runtime error: invalid memory address")
	fmt.Println("Looking for similar fixes...\n")

	searchReq := &remediation.SearchRequest{
		Query:            "panic nil pointer error user context",
		Limit:            5,
		MinConfidence:    0.3,
		TenantID:         tenant,
		TeamID:           teamID,
		ProjectPath:      projectPath,
		Scope:            remediation.ScopeProject,
		IncludeHierarchy: true, // Also search team and org scopes
	}

	results, err := service.Search(ctx, searchReq)
	if err != nil {
		return fmt.Errorf("searching remediations: %w", err)
	}

	fmt.Printf("Found %d similar fixes:\n", len(results))
	for i, result := range results {
		fmt.Printf("\n%d. [%s] %s (score: %.2f, confidence: %.2f)\n",
			i+1, result.ID[:8], result.Title, result.Score, result.Confidence)
		fmt.Printf("   Problem: %s\n", truncate(result.Problem, 80))
		fmt.Printf("   Solution: %s\n", truncate(result.Solution, 80))
	}
	fmt.Println()

	// Step 5: Provide feedback on helpful remediation
	if len(results) > 0 {
		mostRelevant := results[0]
		fmt.Printf("Step 5: Applying solution from \"%s\"...\n", mostRelevant.Title)
		fmt.Println("Applied fix successfully! ✓")
		fmt.Println("Providing feedback...\n")

		feedbackReq := &remediation.FeedbackRequest{
			RemediationID: mostRelevant.ID,
			TenantID:      tenant,
			Rating:        remediation.RatingHelpful,
			SessionID:     sessionID,
			Comment:       "Fixed the issue immediately",
		}

		if err := service.Feedback(ctx, feedbackReq); err != nil {
			return fmt.Errorf("providing feedback: %w", err)
		}

		fmt.Printf("✓ Provided positive feedback (confidence increased)\n\n")
	}

	// Step 6: Search by category
	fmt.Println("Step 6: Searching for performance-related fixes...")
	categorySearch := &remediation.SearchRequest{
		Query:            "slow performance timeout",
		Limit:            10,
		Category:         remediation.ErrorPerformance,
		TenantID:         tenant,
		TeamID:           teamID,
		Scope:            remediation.ScopeTeam,
		IncludeHierarchy: true,
	}

	perfResults, err := service.Search(ctx, categorySearch)
	if err != nil {
		return fmt.Errorf("searching performance fixes: %w", err)
	}

	fmt.Printf("Found %d performance-related fixes:\n", len(perfResults))
	for _, result := range perfResults {
		fmt.Printf("  - [%s] %s (tags: %v)\n",
			result.ID[:8], result.Title, result.Tags)
	}
	fmt.Println()

	// Step 7: Demonstrate scope hierarchy
	fmt.Println("Step 7: Demonstrating scope hierarchy...")
	fmt.Println("Searching from project scope with hierarchy enabled:")
	fmt.Println("  → Will search: project → team → org\n")

	hierarchySearch := &remediation.SearchRequest{
		Query:            "test failure",
		Limit:            10,
		TenantID:         tenant,
		TeamID:           teamID,
		ProjectPath:      projectPath,
		Scope:            remediation.ScopeProject,
		IncludeHierarchy: true, // Search up the hierarchy
	}

	hierResults, err := service.Search(ctx, hierarchySearch)
	if err != nil {
		return fmt.Errorf("searching with hierarchy: %w", err)
	}

	fmt.Printf("Found %d fixes across all scopes:\n", len(hierResults))
	for _, result := range hierResults {
		fmt.Printf("  - [%s] %s (scope: %s)\n",
			result.ID[:8], result.Title, result.Scope)
	}

	return nil
}

// truncate truncates a string to a maximum length with ellipsis.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
