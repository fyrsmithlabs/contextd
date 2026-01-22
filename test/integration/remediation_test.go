package integration

import (
	"context"
	"testing"

	"github.com/fyrsmithlabs/contextd/internal/remediation"
	"github.com/fyrsmithlabs/contextd/internal/vectorstore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// TestRemediation_RecordAndSearch validates error pattern recording and retrieval.
func TestRemediation_RecordAndSearch(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	logger := zap.NewNop()

	store, cleanup := createTestVectorStore(t)
	defer cleanup()

	tenant := &vectorstore.TenantInfo{
		TenantID:  "test-org",
		ProjectID: "test-project",
	}
	tenantCtx := vectorstore.ContextWithTenant(ctx, tenant)

	cfg := &remediation.Config{}
	rs, err := remediation.NewService(cfg, store, logger)
	require.NoError(t, err)

	// 1. Record a remediation
	recordReq := &remediation.RecordRequest{
		Title:    "Nil Pointer Panic Fix",
		Problem:  "panic: runtime error: invalid memory address",
		Symptoms: []string{"nil pointer dereference", "runtime panic"},
		RootCause: "Nil pointer dereference in HTTP handler",
		Solution:  "Add nil check before accessing user.Profile",
		CodeDiff: `if user == nil || user.Profile == nil {
    return nil, fmt.Errorf("user profile not found")
}`,
		AffectedFiles: []string{"internal/api/users.go"},
		Category:      remediation.ErrorRuntime,
		Tags:          []string{"golang", "nil-pointer", "http-handler"},
		Scope:         remediation.ScopeProject,
		TenantID:      tenant.TenantID,
		ProjectPath:   tenant.ProjectID,
	}

	rem, err := rs.Record(tenantCtx, recordReq)
	require.NoError(t, err, "Should record remediation successfully")
	require.NotEmpty(t, rem.ID, "Should return remediation ID")

	t.Logf("✅ Recorded remediation: %s", rem.ID)

	// 2. Search for the remediation
	searchReq := &remediation.SearchRequest{
		Query:       "nil pointer panic",
		Limit:       5,
		TenantID:    tenant.TenantID,
		ProjectPath: tenant.ProjectID,
	}
	results, err := rs.Search(tenantCtx, searchReq)
	require.NoError(t, err, "Should search remediations successfully")
	assert.GreaterOrEqual(t, len(results), 1, "Should find at least one remediation")

	found := false
	for _, result := range results {
		if result.Remediation.Problem == recordReq.Problem {
			found = true
			assert.Equal(t, recordReq.RootCause, result.Remediation.RootCause, "Root cause should match")
			assert.Equal(t, recordReq.Solution, result.Remediation.Solution, "Solution should match")
			assert.Contains(t, result.Remediation.Tags, "nil-pointer", "Tags should be preserved")
			break
		}
	}
	assert.True(t, found, "Should find the recorded remediation")

	t.Logf("✅ Found remediation in search results")
}

// TestRemediation_ErrorPatternMatching validates fuzzy error matching.
func TestRemediation_ErrorPatternMatching(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	logger := zap.NewNop()

	store, cleanup := createTestVectorStore(t)
	defer cleanup()

	tenant := &vectorstore.TenantInfo{
		TenantID:  "test-org",
		ProjectID: "test-project",
	}
	tenantCtx := vectorstore.ContextWithTenant(ctx, tenant)

	cfg := &remediation.Config{}
	rs, err := remediation.NewService(cfg, store, logger)
	require.NoError(t, err)

	// Record remediations for common errors
	remediations := []struct {
		title     string
		problem   string
		rootCause string
		solution  string
		tags      []string
	}{
		{
			title:     "Postgres Connection Refused",
			problem:   "Error: ECONNREFUSED connect ECONNREFUSED 127.0.0.1:5432",
			rootCause: "PostgreSQL database not running",
			solution:  "Start PostgreSQL: sudo systemctl start postgresql",
			tags:      []string{"database", "postgres", "connection"},
		},
		{
			title:     "Express Module Not Found",
			problem:   "Error: Cannot find module 'express'",
			rootCause: "Missing npm dependency",
			solution:  "Run: npm install express",
			tags:      []string{"nodejs", "npm", "dependencies"},
		},
		{
			title:     "Not a Git Repository",
			problem:   "fatal: not a git repository (or any of the parent directories): .git",
			rootCause: "Git not initialized",
			solution:  "Run: git init",
			tags:      []string{"git", "initialization"},
		},
	}

	for _, rem := range remediations {
		_, err := rs.Record(tenantCtx, &remediation.RecordRequest{
			Title:       rem.title,
			Problem:     rem.problem,
			RootCause:   rem.rootCause,
			Solution:    rem.solution,
			Tags:        rem.tags,
			Category:    remediation.ErrorRuntime,
			Scope:       remediation.ScopeProject,
			TenantID:    tenant.TenantID,
			ProjectPath: tenant.ProjectID,
		})
		require.NoError(t, err)
	}

	// Test fuzzy matching with similar error messages
	testCases := []struct {
		query         string
		expectedMatch string
		expectedTag   string
	}{
		{
			query:         "connection refused postgres",
			expectedMatch: "PostgreSQL database not running",
			expectedTag:   "postgres",
		},
		{
			query:         "cannot find express module",
			expectedMatch: "Missing npm dependency",
			expectedTag:   "npm",
		},
		{
			query:         "not a git repository",
			expectedMatch: "Git not initialized",
			expectedTag:   "git",
		},
	}

	for _, tc := range testCases {
		searchReq := &remediation.SearchRequest{
			Query:       tc.query,
			Limit:       5,
			TenantID:    tenant.TenantID,
			ProjectPath: tenant.ProjectID,
		}
		results, err := rs.Search(tenantCtx, searchReq)
		require.NoError(t, err, "Should search for: %s", tc.query)
		require.GreaterOrEqual(t, len(results), 1, "Should find remediation for: %s", tc.query)

		// Verify we found the expected remediation
		found := false
		for _, result := range results {
			if result.Remediation.RootCause == tc.expectedMatch {
				found = true
				assert.Contains(t, result.Remediation.Tags, tc.expectedTag,
					"Should have expected tag for query: %s", tc.query)
				t.Logf("✅ Matched '%s' -> '%s'", tc.query, tc.expectedMatch)
				break
			}
		}
		assert.True(t, found, "Should find expected remediation for: %s", tc.query)
	}
}

// TestRemediation_TenantIsolation validates remediations are isolated by tenant.
func TestRemediation_TenantIsolation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	logger := zap.NewNop()

	store, cleanup := createTestVectorStore(t)
	defer cleanup()

	cfg := &remediation.Config{}
	rs, err := remediation.NewService(cfg, store, logger)
	require.NoError(t, err)

	// Create two tenants
	tenant1Ctx := vectorstore.ContextWithTenant(ctx, &vectorstore.TenantInfo{
		TenantID:  "org-1",
		ProjectID: "project-1",
	})
	tenant2Ctx := vectorstore.ContextWithTenant(ctx, &vectorstore.TenantInfo{
		TenantID:  "org-2",
		ProjectID: "project-2",
	})

	// Tenant 1 records proprietary error fix
	_, err = rs.Record(tenant1Ctx, &remediation.RecordRequest{
		Title:       "Tenant 1 Auth Fix",
		Problem:     "Proprietary API Error: Auth Failed",
		RootCause:   "Tenant 1 custom authentication issue",
		Solution:    "Use Tenant 1 specific auth token format",
		Tags:        []string{"tenant1-specific"},
		Category:    remediation.ErrorRuntime,
		Scope:       remediation.ScopeProject,
		TenantID:    "org-1",
		ProjectPath: "project-1",
	})
	require.NoError(t, err)

	// Tenant 2 records different error fix
	_, err = rs.Record(tenant2Ctx, &remediation.RecordRequest{
		Title:       "Tenant 2 Auth Fix",
		Problem:     "Proprietary API Error: Auth Failed",
		RootCause:   "Tenant 2 custom authentication issue",
		Solution:    "Use Tenant 2 specific auth token format",
		Tags:        []string{"tenant2-specific"},
		Category:    remediation.ErrorRuntime,
		Scope:       remediation.ScopeProject,
		TenantID:    "org-2",
		ProjectPath: "project-2",
	})
	require.NoError(t, err)

	// Tenant 1 searches for auth errors
	results1, err := rs.Search(tenant1Ctx, &remediation.SearchRequest{
		Query:       "API Auth Failed",
		Limit:       10,
		TenantID:    "org-1",
		ProjectPath: "project-1",
	})
	require.NoError(t, err)
	assert.Equal(t, 1, len(results1), "Tenant 1 should see exactly 1 remediation")
	assert.Contains(t, results1[0].Remediation.Solution, "Tenant 1 specific",
		"Tenant 1 should see their solution")

	// Tenant 2 searches for auth errors
	results2, err := rs.Search(tenant2Ctx, &remediation.SearchRequest{
		Query:       "API Auth Failed",
		Limit:       10,
		TenantID:    "org-2",
		ProjectPath: "project-2",
	})
	require.NoError(t, err)
	assert.Equal(t, 1, len(results2), "Tenant 2 should see exactly 1 remediation")
	assert.Contains(t, results2[0].Remediation.Solution, "Tenant 2 specific",
		"Tenant 2 should see their solution")

	t.Logf("✅ Remediation tenant isolation verified")
}

// TestRemediation_TagFiltering validates tag-based search filtering.
func TestRemediation_TagFiltering(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	logger := zap.NewNop()

	store, cleanup := createTestVectorStore(t)
	defer cleanup()

	tenant := &vectorstore.TenantInfo{
		TenantID:  "test-org",
		ProjectID: "test-project",
	}
	tenantCtx := vectorstore.ContextWithTenant(ctx, tenant)

	cfg := &remediation.Config{}
	rs, err := remediation.NewService(cfg, store, logger)
	require.NoError(t, err)

	// Record remediations with different tags
	remediations := []struct {
		title   string
		problem string
		tags    []string
	}{
		{
			title:   "Python Import Error",
			problem: "Python ImportError: No module named 'requests'",
			tags:    []string{"python", "pip", "dependencies"},
		},
		{
			title:   "Go Undefined HTTP Server",
			problem: "Go: undefined: http.Server",
			tags:    []string{"golang", "imports", "http"},
		},
		{
			title:   "Rust Borrow Moved Value",
			problem: "Rust: error[E0382]: borrow of moved value",
			tags:    []string{"rust", "ownership", "borrowing"},
		},
	}

	for _, rem := range remediations {
		_, err := rs.Record(tenantCtx, &remediation.RecordRequest{
			Title:       rem.title,
			Problem:     rem.problem,
			RootCause:   "Test error",
			Solution:    "Test solution",
			Tags:        rem.tags,
			Category:    remediation.ErrorRuntime,
			Scope:       remediation.ScopeProject,
			TenantID:    tenant.TenantID,
			ProjectPath: tenant.ProjectID,
		})
		require.NoError(t, err)
	}

	// Search for Python-specific errors
	searchReq := &remediation.SearchRequest{
		Query:       "python import error",
		Limit:       10,
		TenantID:    tenant.TenantID,
		ProjectPath: tenant.ProjectID,
		Tags:        []string{"python"},
	}
	results, err := rs.Search(tenantCtx, searchReq)
	require.NoError(t, err)

	// Should find Python remediation
	pythonFound := false
	for _, result := range results {
		if contains(result.Remediation.Tags, "python") {
			pythonFound = true
		}
		// Should not find Rust or Go-specific errors in top results for Python query
		assert.False(t, contains(result.Remediation.Tags, "rust") && !contains(result.Remediation.Tags, "python"),
			"Python query should not prioritize Rust errors")
	}
	assert.True(t, pythonFound, "Should find Python-tagged remediation")

	t.Logf("✅ Tag-based filtering works correctly")
}

// Helper function
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
