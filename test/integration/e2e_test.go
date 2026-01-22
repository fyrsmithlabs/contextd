package integration

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/fyrsmithlabs/contextd/internal/checkpoint"
	"github.com/fyrsmithlabs/contextd/internal/reasoningbank"
	"github.com/fyrsmithlabs/contextd/internal/remediation"
	"github.com/fyrsmithlabs/contextd/internal/vectorstore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// TestE2E_DevelopmentWorkflow validates a complete development workflow:
// 1. Record a past learning (memory)
// 2. Search for similar past approaches
// 3. Create checkpoint during work
// 4. Encounter an error
// 5. Search for remediation
// 6. Record new remediation
// 7. Save final checkpoint
func TestE2E_DevelopmentWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	ctx := context.Background()
	logger := zap.NewNop()

	// Setup services
	store, cleanup := createTestVectorStore(t)
	defer cleanup()

	tenant := &vectorstore.TenantInfo{
		TenantID:  "acme-corp",
		ProjectID: "auth-service",
	}
	tenantCtx := vectorstore.ContextWithTenant(ctx, tenant)

	rb, err := reasoningbank.NewService(store, logger, reasoningbank.WithDefaultTenant(tenant.TenantID))
	require.NoError(t, err)

	ckptCfg := checkpoint.DefaultServiceConfig()
	cs, err := checkpoint.NewServiceWithStore(ckptCfg, store, logger)
	require.NoError(t, err)

	remCfg := &remediation.Config{}
	rs, err := remediation.NewService(remCfg, store, logger)
	require.NoError(t, err)

	// ═══════════════════════════════════════════════════════════════
	// Phase 1: Start new feature - search for past learnings
	// ═══════════════════════════════════════════════════════════════

	// Record past successful approach
	mem, err := reasoningbank.NewMemory(
		tenant.ProjectID,
		"Secure password hashing with bcrypt",
		"Use bcrypt for password hashing with cost factor 12. This provides strong protection against brute force attacks.",
		reasoningbank.OutcomeSuccess,
		[]string{"security", "authentication", "passwords", "bcrypt"},
	)
	require.NoError(t, err)
	mem.Description = "Implemented secure authentication in user-service"
	mem.Confidence = 0.95

	err = rb.Record(tenantCtx, mem)
	require.NoError(t, err)

	// Developer searches for password security best practices
	memories, err := rb.Search(tenantCtx, tenant.ProjectID, "password hashing security", 5)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(memories), 1, "Should find past password hashing approach")

	foundBcrypt := false
	for _, m := range memories {
		if m.ID == mem.ID {
			foundBcrypt = true
			t.Logf("✅ Phase 1: Found past approach - bcrypt with cost 12")
			break
		}
	}
	assert.True(t, foundBcrypt, "Should find bcrypt recommendation")

	// ═══════════════════════════════════════════════════════════════
	// Phase 2: Create checkpoint before making changes
	// ═══════════════════════════════════════════════════════════════

	saveReq1 := &checkpoint.SaveRequest{
		SessionID:   "auth-feature-session",
		TenantID:    tenant.TenantID,
		ProjectID:   tenant.ProjectID,
		ProjectPath: "/acme/auth-service",
		Name:        "Auth Feature - Start",
		Summary:     "Researched password hashing approaches, found bcrypt recommendation from past work",
		Context:     "User: Implement password reset functionality\nAssistant: I'll implement secure password reset using bcrypt...",
		Metadata: map[string]string{
			"phase": "start",
		},
	}

	ckpt1, err := cs.Save(tenantCtx, saveReq1)
	require.NoError(t, err)
	t.Logf("✅ Phase 2: Saved checkpoint - %s", ckpt1.ID)

	// ═══════════════════════════════════════════════════════════════
	// Phase 3: Encounter an error during implementation
	// ═══════════════════════════════════════════════════════════════

	errorMessage := "Error: bcrypt: password length exceeds 72 bytes"

	// Search for existing remediations
	remResults, err := rs.Search(tenantCtx, &remediation.SearchRequest{
		Query:         errorMessage,
		Limit:         5,
		MinConfidence: 0.5,
		TenantID:      tenant.TenantID,
		ProjectPath:   "/acme/auth-service",
		Scope:         remediation.ScopeProject,
	})
	require.NoError(t, err)

	if len(remResults) == 0 {
		// No existing fix - record new remediation
		newRem, err := rs.Record(tenantCtx, &remediation.RecordRequest{
			Title:   "Fix bcrypt password length limit",
			Problem: "bcrypt: password length exceeds 72 bytes",
			Symptoms: []string{
				"panic when hashing long passwords",
				"bcrypt error during authentication",
			},
			RootCause: "bcrypt has a hard 72-byte password limit",
			Solution:  "Hash password with SHA-256 before bcrypt if length > 72 bytes",
			CodeDiff: `// Pre-hash long passwords
if len(password) > 72 {
    hash := sha256.Sum256([]byte(password))
    password = base64.StdEncoding.EncodeToString(hash[:])
}
hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), 12)`,
			Category:    remediation.ErrorRuntime,
			Scope:       remediation.ScopeProject,
			TenantID:    tenant.TenantID,
			ProjectPath: "/acme/auth-service",
			Tags:        []string{"bcrypt", "password-length", "golang"},
		})
		require.NoError(t, err)
		t.Logf("✅ Phase 3: Recorded new remediation - %s", newRem.ID)
	} else {
		t.Logf("✅ Phase 3: Found existing remediation for bcrypt length error")
	}

	// ═══════════════════════════════════════════════════════════════
	// Phase 4: Apply fix and update checkpoint
	// ═══════════════════════════════════════════════════════════════

	saveReq2 := &checkpoint.SaveRequest{
		SessionID:   "auth-feature-session",
		TenantID:    tenant.TenantID,
		ProjectID:   tenant.ProjectID,
		ProjectPath: "/acme/auth-service",
		Name:        "Auth Feature - Error Fixed",
		Summary:     "Resolved bcrypt 72-byte limit with SHA-256 pre-hashing, implemented password reset with token generation",
		Context:     "User: Got bcrypt length error\nAssistant: Applied SHA-256 pre-hash fix",
		Metadata: map[string]string{
			"phase": "implementation",
		},
	}

	ckpt2, err := cs.Save(tenantCtx, saveReq2)
	require.NoError(t, err)
	t.Logf("✅ Phase 4: Updated checkpoint after fix - %s", ckpt2.ID)

	// ═══════════════════════════════════════════════════════════════
	// Phase 5: Complete feature and record success
	// ═══════════════════════════════════════════════════════════════

	// Mark the bcrypt memory as used successfully
	for _, mem := range memories {
		if mem.Content == "Use bcrypt for password hashing with cost factor 12. This provides strong protection against brute force attacks." {
			err = rb.Feedback(tenantCtx, mem.ID, true)
			require.NoError(t, err)
			t.Logf("✅ Phase 5: Recorded successful use of bcrypt memory")
			break
		}
	}

	// Final checkpoint
	saveReq3 := &checkpoint.SaveRequest{
		SessionID:   "auth-feature-session",
		TenantID:    tenant.TenantID,
		ProjectID:   tenant.ProjectID,
		ProjectPath: "/acme/auth-service",
		Name:        "Auth Feature - Complete",
		Summary:     "Implemented secure password reset, applied bcrypt with SHA-256 pre-hash for long passwords, added comprehensive test coverage, updated documentation",
		Context:     "Assistant: Feature completed with tests",
		Metadata: map[string]string{
			"phase":    "complete",
			"features": "password-reset,bcrypt,sha256-prehash",
		},
	}

	ckpt3, err := cs.Save(tenantCtx, saveReq3)
	require.NoError(t, err)
	t.Logf("✅ Phase 5: Final checkpoint - %s", ckpt3.ID)

	// Verify we can resume from final checkpoint
	resumeReq := &checkpoint.ResumeRequest{
		CheckpointID: ckpt3.ID,
		TenantID:     tenant.TenantID,
		ProjectID:    tenant.ProjectID,
		Level:        "summary",
	}
	resumeResp, err := cs.Resume(tenantCtx, resumeReq)
	require.NoError(t, err)
	assert.NotNil(t, resumeResp.Checkpoint)
	assert.Equal(t, "Auth Feature - Complete", resumeResp.Checkpoint.Name, "Should resume final checkpoint")

	t.Logf("✅ E2E Workflow Complete: Memory → Checkpoint → Error → Remediation → Success")
}

// TestE2E_CodebaseExploration validates using context-folding with repository search:
// 1. Index a codebase
// 2. Create branch for exploration
// 3. Search for functions in branch
// 4. Return condensed findings
func TestE2E_CodebaseExploration(t *testing.T) {
	t.Skip("TODO: Fix repository and folding API usage - requires complex setup")
	// The repository and folding APIs need to be properly initialized
	// This test is disabled until the correct initialization code is determined
}

// Helper functions

func createLargeTestRepo(t *testing.T, dir string) {
	files := map[string]string{
		"auth.go": `package main
import "errors"
func Authenticate(username, password string) error {
	if username == "" { return errors.New("invalid") }
	return nil
}`,
		"login.go": `package main
func HandleLogin(user string) bool {
	// Login handler
	return true
}`,
		"middleware.go": `package main
func AuthMiddleware() {
	// Check authentication
}`,
		"db.go": `package main
func ConnectDB() {}`,
		"api.go": `package main
func StartAPI() {}`,
	}

	for filename, content := range files {
		err := os.WriteFile(filepath.Join(dir, filename), []byte(content), 0644)
		require.NoError(t, err)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
