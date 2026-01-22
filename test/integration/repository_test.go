package integration

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/fyrsmithlabs/contextd/internal/repository"
	"github.com/fyrsmithlabs/contextd/internal/vectorstore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRepository_IndexAndSearch validates repository indexing and semantic search.
func TestRepository_IndexAndSearch(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	// Create test repository
	tmpDir := t.TempDir()
	createTestRepo(t, tmpDir)

	// Create services
	store, cleanup := createTestVectorStore(t)
	defer cleanup()

	repoSvc := repository.NewService(store)

	tenant := &vectorstore.TenantInfo{
		TenantID:  "test-org",
		ProjectID: "test-project",
	}
	tenantCtx := vectorstore.ContextWithTenant(ctx, tenant)

	// 1. Index the repository
	indexOpts := repository.IndexOptions{
		TenantID: tenant.TenantID,
	}
	_, err := repoSvc.IndexRepository(tenantCtx, tmpDir, indexOpts)
	require.NoError(t, err, "Should index repository successfully")

	t.Logf("✅ Repository indexed")

	// 2. Search for code
	searchOpts := repository.SearchOptions{
		ProjectPath: tmpDir,
		TenantID:    tenant.TenantID,
		Limit:       5,
	}
	results, err := repoSvc.Search(tenantCtx, "user authentication", searchOpts)
	require.NoError(t, err, "Should search code successfully")
	assert.GreaterOrEqual(t, len(results), 1, "Should find authentication code")

	// Verify we found the auth.go file
	foundAuth := false
	for _, result := range results {
		if filepath.Base(result.FilePath) == "auth.go" {
			foundAuth = true
			assert.Contains(t, result.Content, "func Authenticate",
				"Should find Authenticate function")
			t.Logf("✅ Found auth.go with score: %.2f", result.Score)
			break
		}
	}
	assert.True(t, foundAuth, "Should find auth.go file")

	// 3. Search for different concepts
	results, err = repoSvc.Search(tenantCtx, "database connection", searchOpts)
	require.NoError(t, err)

	foundDB := false
	for _, result := range results {
		if filepath.Base(result.FilePath) == "db.go" {
			foundDB = true
			t.Logf("✅ Found db.go with score: %.2f", result.Score)
			break
		}
	}
	assert.True(t, foundDB, "Should find db.go file")
}

// TestRepository_GrepFallback validates grep fallback when semantic search fails.
func TestRepository_GrepFallback(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	tmpDir := t.TempDir()
	createTestRepo(t, tmpDir)

	store, cleanup := createTestVectorStore(t)
	defer cleanup()

	repoSvc := repository.NewService(store)

	tenant := &vectorstore.TenantInfo{
		TenantID:  "test-org",
		ProjectID: "test-project",
	}
	tenantCtx := vectorstore.ContextWithTenant(ctx, tenant)

	// Index repository
	indexOpts := repository.IndexOptions{
		TenantID: tenant.TenantID,
	}
	_, err := repoSvc.IndexRepository(tenantCtx, tmpDir, indexOpts)
	require.NoError(t, err)

	// Search for exact function name (should use grep fallback)
	searchOpts := repository.SearchOptions{
		ProjectPath: tmpDir,
		TenantID:    tenant.TenantID,
		Limit:       10,
	}
	results, err := repoSvc.Search(tenantCtx, "func Authenticate", searchOpts)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(results), 1, "Should find exact function name")

	// Verify result contains the exact match
	foundExact := false
	for _, result := range results {
		if filepath.Base(result.FilePath) == "auth.go" {
			assert.Contains(t, result.Content, "func Authenticate",
				"Should contain exact function signature")
			foundExact = true
			t.Logf("✅ Grep fallback found exact match")
			break
		}
	}
	assert.True(t, foundExact, "Should find exact match via grep")
}

// TestRepository_FileTypeFiltering validates filtering by file extensions.
func TestRepository_FileTypeFiltering(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	tmpDir := t.TempDir()
	createTestRepo(t, tmpDir)

	store, cleanup := createTestVectorStore(t)
	defer cleanup()

	repoSvc := repository.NewService(store)

	tenant := &vectorstore.TenantInfo{
		TenantID:  "test-org",
		ProjectID: "test-project",
	}
	tenantCtx := vectorstore.ContextWithTenant(ctx, tenant)

	// Index repository
	indexOpts := repository.IndexOptions{
		TenantID: tenant.TenantID,
	}
	_, err := repoSvc.IndexRepository(tenantCtx, tmpDir, indexOpts)
	require.NoError(t, err)

	// Search should only index .go files (not .md, .txt, etc.)
	searchOpts := repository.SearchOptions{
		ProjectPath: tmpDir,
		TenantID:    tenant.TenantID,
		Limit:       20,
	}
	results, err := repoSvc.Search(tenantCtx, "test", searchOpts)
	require.NoError(t, err)

	for _, result := range results {
		ext := filepath.Ext(result.FilePath)
		assert.Contains(t, []string{".go"}, ext,
			"Should only return .go files, found: %s", result.FilePath)
	}

	t.Logf("✅ File type filtering works correctly")
}

// TestRepository_TenantIsolation validates repository code is isolated by tenant.
func TestRepository_TenantIsolation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	// Create two separate repos
	tmpDir1 := t.TempDir()
	tmpDir2 := t.TempDir()

	// Repo 1: Has auth code
	createFileInDir(t, tmpDir1, "auth.go", `package main
func Tenant1Auth() {
	// Tenant 1 specific authentication
}`)

	// Repo 2: Has different auth code
	createFileInDir(t, tmpDir2, "auth.go", `package main
func Tenant2Auth() {
	// Tenant 2 specific authentication
}`)

	store, cleanup := createTestVectorStore(t)
	defer cleanup()

	repoSvc := repository.NewService(store)

	// Create tenant contexts
	tenant1Ctx := vectorstore.ContextWithTenant(ctx, &vectorstore.TenantInfo{
		TenantID:  "org-1",
		ProjectID: "project-1",
	})
	tenant2Ctx := vectorstore.ContextWithTenant(ctx, &vectorstore.TenantInfo{
		TenantID:  "org-2",
		ProjectID: "project-2",
	})

	// Index both repos for different tenants
	indexOpts1 := repository.IndexOptions{
		TenantID: "org-1",
	}
	_, err := repoSvc.IndexRepository(tenant1Ctx, tmpDir1, indexOpts1)
	require.NoError(t, err)

	indexOpts2 := repository.IndexOptions{
		TenantID: "org-2",
	}
	_, err = repoSvc.IndexRepository(tenant2Ctx, tmpDir2, indexOpts2)
	require.NoError(t, err)

	// Tenant 1 searches for auth
	searchOpts1 := repository.SearchOptions{
		ProjectPath: tmpDir1,
		TenantID:    "org-1",
		Limit:       10,
	}
	results1, err := repoSvc.Search(tenant1Ctx, "authentication", searchOpts1)
	require.NoError(t, err)

	for _, result := range results1 {
		assert.Contains(t, result.Content, "Tenant1Auth",
			"Tenant 1 should only see their code")
		assert.NotContains(t, result.Content, "Tenant2Auth",
			"Tenant 1 should not see Tenant 2 code")
	}

	// Tenant 2 searches for auth
	searchOpts2 := repository.SearchOptions{
		ProjectPath: tmpDir2,
		TenantID:    "org-2",
		Limit:       10,
	}
	results2, err := repoSvc.Search(tenant2Ctx, "authentication", searchOpts2)
	require.NoError(t, err)

	for _, result := range results2 {
		assert.Contains(t, result.Content, "Tenant2Auth",
			"Tenant 2 should only see their code")
		assert.NotContains(t, result.Content, "Tenant1Auth",
			"Tenant 2 should not see Tenant 1 code")
	}

	t.Logf("✅ Repository tenant isolation verified")
}

// Helper functions

func createTestRepo(t *testing.T, dir string) {
	// Create auth.go
	authCode := `package main

import "errors"

// Authenticate validates user credentials
func Authenticate(username, password string) error {
	if username == "" || password == "" {
		return errors.New("invalid credentials")
	}
	// Check credentials against database
	return nil
}

// ValidateToken checks JWT token
func ValidateToken(token string) (bool, error) {
	if token == "" {
		return false, errors.New("empty token")
	}
	return true, nil
}
`
	createFileInDir(t, dir, "auth.go", authCode)

	// Create db.go
	dbCode := `package main

import "database/sql"

// Connect establishes database connection
func Connect(dsn string) (*sql.DB, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}
	return db, nil
}

// Query executes a database query
func Query(db *sql.DB, query string) error {
	_, err := db.Exec(query)
	return err
}
`
	createFileInDir(t, dir, "db.go", dbCode)

	// Create handler.go
	handlerCode := `package main

import "net/http"

// HandleRequest processes HTTP requests
func HandleRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.WriteHeader(http.StatusOK)
}
`
	createFileInDir(t, dir, "handler.go", handlerCode)
}

func createFileInDir(t *testing.T, dir, filename, content string) {
	filePath := filepath.Join(dir, filename)
	err := os.WriteFile(filePath, []byte(content), 0644)
	require.NoError(t, err, "Should create file: %s", filename)
}
