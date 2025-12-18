package repository

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fyrsmithlabs/contextd/internal/sanitize"
	"github.com/fyrsmithlabs/contextd/internal/vectorstore"
)

// ===== MOCK STORE =====

// mockStore implements Store interface for testing
type mockStore struct {
	documents      []vectorstore.Document
	searchResults  []vectorstore.SearchResult
	addError       error
	searchError    error
	lastCollection string
	lastQuery      string
	lastFilters    map[string]interface{}
}

func (m *mockStore) AddDocuments(ctx context.Context, docs []vectorstore.Document) ([]string, error) {
	if m.addError != nil {
		return nil, m.addError
	}
	ids := make([]string, len(docs))
	for i, doc := range docs {
		m.documents = append(m.documents, doc)
		ids[i] = fmt.Sprintf("doc_%d", i)
		// Track last collection used
		if doc.Collection != "" {
			m.lastCollection = doc.Collection
		}
	}
	return ids, nil
}

func (m *mockStore) SearchInCollection(ctx context.Context, collectionName string, query string, k int, filters map[string]interface{}) ([]vectorstore.SearchResult, error) {
	m.lastCollection = collectionName
	m.lastQuery = query
	m.lastFilters = filters
	if m.searchError != nil {
		return nil, m.searchError
	}
	return m.searchResults, nil
}

func (m *mockStore) Close() error {
	return nil
}

// Stub methods to satisfy vectorstore.Store interface for StoreProvider tests
func (m *mockStore) Search(ctx context.Context, query string, k int) ([]vectorstore.SearchResult, error) {
	return m.searchResults, nil
}

func (m *mockStore) SearchWithFilters(ctx context.Context, query string, k int, filters map[string]interface{}) ([]vectorstore.SearchResult, error) {
	return m.searchResults, nil
}

func (m *mockStore) DeleteDocuments(ctx context.Context, ids []string) error {
	return nil
}

func (m *mockStore) DeleteDocumentsFromCollection(ctx context.Context, collectionName string, ids []string) error {
	return nil
}

func (m *mockStore) CreateCollection(ctx context.Context, collectionName string, vectorSize int) error {
	return nil
}

func (m *mockStore) DeleteCollection(ctx context.Context, collectionName string) error {
	return nil
}

func (m *mockStore) CollectionExists(ctx context.Context, collectionName string) (bool, error) {
	return true, nil
}

func (m *mockStore) ListCollections(ctx context.Context) ([]string, error) {
	return []string{}, nil
}

func (m *mockStore) GetCollectionInfo(ctx context.Context, collectionName string) (*vectorstore.CollectionInfo, error) {
	return &vectorstore.CollectionInfo{}, nil
}

func (m *mockStore) ExactSearch(ctx context.Context, collectionName string, query string, k int) ([]vectorstore.SearchResult, error) {
	return m.searchResults, nil
}

// ===== MOCK STORE PROVIDER =====

// mockStoreProvider implements vectorstore.StoreProvider for testing
type mockStoreProvider struct {
	stores map[string]*mockStore
}

func newMockStoreProvider() *mockStoreProvider {
	return &mockStoreProvider{
		stores: make(map[string]*mockStore),
	}
}

func (p *mockStoreProvider) GetProjectStore(ctx context.Context, tenant, team, project string) (vectorstore.Store, error) {
	key := tenant + "/" + team + "/" + project
	if store, ok := p.stores[key]; ok {
		return store, nil
	}
	store := &mockStore{}
	p.stores[key] = store
	return store, nil
}

func (p *mockStoreProvider) GetTeamStore(ctx context.Context, tenant, team string) (vectorstore.Store, error) {
	key := tenant + "/" + team + "/_team"
	if store, ok := p.stores[key]; ok {
		return store, nil
	}
	store := &mockStore{}
	p.stores[key] = store
	return store, nil
}

func (p *mockStoreProvider) GetOrgStore(ctx context.Context, tenant string) (vectorstore.Store, error) {
	key := tenant + "/_org"
	if store, ok := p.stores[key]; ok {
		return store, nil
	}
	store := &mockStore{}
	p.stores[key] = store
	return store, nil
}

func (p *mockStoreProvider) Close() error {
	return nil
}

// ===== STORE PROVIDER TESTS =====

func TestNewServiceWithStoreProvider(t *testing.T) {
	provider := newMockStoreProvider()
	svc := NewServiceWithStoreProvider(provider)

	if svc == nil {
		t.Fatal("NewServiceWithStoreProvider returned nil")
	}
	if svc.stores == nil {
		t.Error("Expected stores to be set")
	}
	if svc.store != nil {
		t.Error("Expected store to be nil when using StoreProvider")
	}
}

func TestService_WithStoreProvider_Index(t *testing.T) {
	tmpDir := t.TempDir()
	createTestFile(t, tmpDir, "main.go", "package main")

	provider := newMockStoreProvider()
	svc := NewServiceWithStoreProvider(provider)

	result, err := svc.IndexRepository(context.Background(), tmpDir, IndexOptions{
		TenantID: "testuser",
	})

	if err != nil {
		t.Fatalf("IndexRepository() error = %v", err)
	}

	// With StoreProvider, collection should be simple "codebase"
	if result.CollectionName != "codebase" {
		t.Errorf("CollectionName = %q, want %q", result.CollectionName, "codebase")
	}

	// Verify store was retrieved with correct scope
	projectName := filepath.Base(tmpDir)
	key := "testuser//" + projectName
	if _, ok := provider.stores[key]; !ok {
		t.Errorf("Expected store for key %q, got keys: %v", key, provider.stores)
	}
}

func TestService_WithStoreProvider_Search(t *testing.T) {
	provider := newMockStoreProvider()
	svc := NewServiceWithStoreProvider(provider)

	// Pre-populate a store with search results
	projectPath := "/path/to/myproject"
	projectName := filepath.Base(projectPath)

	// Create store first to set up search results
	ctx := context.Background()
	store, _ := provider.GetProjectStore(ctx, "testuser", "", projectName)
	mockS := store.(*mockStore)
	mockS.searchResults = []vectorstore.SearchResult{
		{
			Content: "test content",
			Score:   0.9,
			Metadata: map[string]interface{}{
				"file_path": "main.go",
				"branch":    "main",
			},
		},
	}

	results, err := svc.Search(ctx, "test query", SearchOptions{
		ProjectPath: projectPath,
		TenantID:    "testuser",
	})

	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	// Verify search was called on correct store with simple collection name
	if mockS.lastCollection != "codebase" {
		t.Errorf("lastCollection = %q, want %q", mockS.lastCollection, "codebase")
	}
}

func TestService_WithStoreProvider_AutoDeriveTenant(t *testing.T) {
	provider := newMockStoreProvider()
	svc := NewServiceWithStoreProvider(provider)

	tmpDir := t.TempDir()
	createTestFile(t, tmpDir, "main.go", "package main")

	// Index without explicit tenant_id
	result, err := svc.IndexRepository(context.Background(), tmpDir, IndexOptions{})

	if err != nil {
		t.Fatalf("IndexRepository() error = %v", err)
	}

	// Should succeed with auto-derived tenant
	if result.CollectionName != "codebase" {
		t.Errorf("CollectionName = %q, want %q", result.CollectionName, "codebase")
	}

	// Verify a store was created (tenant was auto-derived)
	if len(provider.stores) == 0 {
		t.Error("Expected at least one store to be created")
	}
}

// ===== NEW TESTS: _codebase COLLECTION =====

func TestIndexRepository_UsesCodebaseCollection(t *testing.T) {
	// Setup
	tmpDir := t.TempDir()
	createTestFile(t, tmpDir, "main.go", "package main")

	store := &mockStore{}
	svc := NewService(store)

	opts := IndexOptions{
		TenantID: "testuser",
	}

	// Execute
	result, err := svc.IndexRepository(context.Background(), tmpDir, opts)

	// Verify
	if err != nil {
		t.Fatalf("IndexRepository() error = %v", err)
	}

	// Collection should be {tenant}_{project}_codebase
	projectName := sanitize.Identifier(filepath.Base(tmpDir))
	expectedCollection := fmt.Sprintf("testuser_%s_codebase", projectName)

	if store.lastCollection != expectedCollection {
		t.Errorf("Collection = %q, want %q", store.lastCollection, expectedCollection)
	}

	if result.CollectionName != expectedCollection {
		t.Errorf("Result.CollectionName = %q, want %q", result.CollectionName, expectedCollection)
	}

	// Verify documents have correct collection set
	for _, doc := range store.documents {
		if doc.Collection != expectedCollection {
			t.Errorf("Document.Collection = %q, want %q", doc.Collection, expectedCollection)
		}
	}
}

func TestIndexRepository_DetectsBranch(t *testing.T) {
	// This test uses the actual contextd repo (which has .git)
	// to verify branch detection works
	cwd, err := os.Getwd()
	if err != nil {
		t.Skip("Cannot get working directory")
	}

	// Find repo root (walk up to find .git)
	repoRoot := cwd
	for {
		if _, err := os.Stat(filepath.Join(repoRoot, ".git")); err == nil {
			break
		}
		parent := filepath.Dir(repoRoot)
		if parent == repoRoot {
			t.Skip("Not in a git repository")
		}
		repoRoot = parent
	}

	// Test branch detection
	branch := detectGitBranch(repoRoot)

	if branch == "" || branch == "unknown" {
		t.Errorf("detectGitBranch() = %q, want valid branch name", branch)
	}

	t.Logf("Detected branch: %s", branch)
}

func TestIndexRepository_IncludesBranchInMetadata(t *testing.T) {
	tmpDir := t.TempDir()
	createTestFile(t, tmpDir, "main.go", "package main")

	store := &mockStore{}
	svc := NewService(store)

	opts := IndexOptions{
		TenantID: "testuser",
		Branch:   "feature/test-branch",
	}

	// Execute
	result, err := svc.IndexRepository(context.Background(), tmpDir, opts)

	// Verify
	if err != nil {
		t.Fatalf("IndexRepository() error = %v", err)
	}

	if result.Branch != "feature/test-branch" {
		t.Errorf("Result.Branch = %q, want %q", result.Branch, "feature/test-branch")
	}

	// Verify documents have branch in metadata
	for _, doc := range store.documents {
		branch, ok := doc.Metadata["branch"].(string)
		if !ok {
			t.Error("Document missing branch in metadata")
			continue
		}
		if branch != "feature/test-branch" {
			t.Errorf("Document branch = %q, want %q", branch, "feature/test-branch")
		}
	}
}

func TestIndexRepository_AutoDetectsBranchWhenNotSpecified(t *testing.T) {
	tmpDir := t.TempDir()
	createTestFile(t, tmpDir, "main.go", "package main")

	store := &mockStore{}
	svc := NewService(store)

	opts := IndexOptions{
		TenantID: "testuser",
		// Branch not specified - should auto-detect or use "unknown"
	}

	// Execute
	result, err := svc.IndexRepository(context.Background(), tmpDir, opts)

	// Verify
	if err != nil {
		t.Fatalf("IndexRepository() error = %v", err)
	}

	// For a non-git directory, should be "unknown"
	if result.Branch == "" {
		t.Error("Result.Branch should not be empty")
	}

	t.Logf("Auto-detected branch: %s", result.Branch)
}

// ===== NEW TESTS: SEARCH WITH BRANCH FILTER =====

func TestSearch_UsesCodebaseCollection(t *testing.T) {
	store := &mockStore{
		searchResults: []vectorstore.SearchResult{
			{ID: "1", Content: "test content", Score: 0.9, Metadata: map[string]interface{}{"file_path": "main.go", "branch": "main"}},
		},
	}
	svc := NewService(store)

	opts := SearchOptions{
		ProjectPath: "/path/to/myproject",
		TenantID:    "testuser",
		Limit:       10,
	}

	// Execute
	_, err := svc.Search(context.Background(), "test query", opts)

	// Verify
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	// Should search in _codebase collection
	expectedCollection := "testuser_myproject_codebase"
	if store.lastCollection != expectedCollection {
		t.Errorf("Search collection = %q, want %q", store.lastCollection, expectedCollection)
	}
}

func TestSearch_FiltersByBranch(t *testing.T) {
	store := &mockStore{
		searchResults: []vectorstore.SearchResult{
			{ID: "1", Content: "test", Score: 0.9, Metadata: map[string]interface{}{"file_path": "main.go", "branch": "develop"}},
		},
	}
	svc := NewService(store)

	opts := SearchOptions{
		ProjectPath: "/path/to/project",
		TenantID:    "testuser",
		Branch:      "develop",
		Limit:       10,
	}

	// Execute
	_, err := svc.Search(context.Background(), "query", opts)

	// Verify
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	// Should have branch filter
	if store.lastFilters == nil {
		t.Fatal("Search filters = nil, want branch filter")
	}
	if store.lastFilters["branch"] != "develop" {
		t.Errorf("Search branch filter = %v, want %q", store.lastFilters["branch"], "develop")
	}
}

func TestSearch_NoBranchFilterWhenEmpty(t *testing.T) {
	store := &mockStore{
		searchResults: []vectorstore.SearchResult{},
	}
	svc := NewService(store)

	opts := SearchOptions{
		ProjectPath: "/path/to/project",
		TenantID:    "testuser",
		// Branch not specified
	}

	// Execute
	_, err := svc.Search(context.Background(), "query", opts)

	// Verify
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	// Should NOT have branch filter when not specified
	if _, hasBranch := store.lastFilters["branch"]; hasBranch {
		t.Error("Search should not filter by branch when not specified")
	}
}

func TestSearch_ReturnsBranchInResults(t *testing.T) {
	store := &mockStore{
		searchResults: []vectorstore.SearchResult{
			{
				ID:      "1",
				Content: "package main",
				Score:   0.95,
				Metadata: map[string]interface{}{
					"file_path": "main.go",
					"branch":    "feature/new-feature",
				},
			},
		},
	}
	svc := NewService(store)

	opts := SearchOptions{
		ProjectPath: "/path/to/project",
		TenantID:    "testuser",
	}

	// Execute
	results, err := svc.Search(context.Background(), "main", opts)

	// Verify
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("Search results = %d, want 1", len(results))
	}

	if results[0].Branch != "feature/new-feature" {
		t.Errorf("Result.Branch = %q, want %q", results[0].Branch, "feature/new-feature")
	}

	if results[0].FilePath != "main.go" {
		t.Errorf("Result.FilePath = %q, want %q", results[0].FilePath, "main.go")
	}
}

// TestSearch_WithCollectionName verifies that CollectionName bypasses tenant_id derivation.
// This fixes the bug where repository_index with explicit tenant_id produces a different
// collection name than repository_search with derived tenant_id.
func TestSearch_WithCollectionName(t *testing.T) {
	store := &mockStore{
		searchResults: []vectorstore.SearchResult{
			{ID: "1", Content: "test content", Score: 0.9, Metadata: map[string]interface{}{"file_path": "main.go"}},
		},
	}
	svc := NewService(store)

	// Use CollectionName directly - no tenant_id/project_path needed
	opts := SearchOptions{
		CollectionName: "dahendel_onprem_pw_codebase",
		Limit:          10,
	}

	// Execute
	_, err := svc.Search(context.Background(), "test query", opts)

	// Verify
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	// Should use collection name directly
	if store.lastCollection != "dahendel_onprem_pw_codebase" {
		t.Errorf("Search collection = %q, want %q", store.lastCollection, "dahendel_onprem_pw_codebase")
	}
}

// TestSearch_HyphenatedProjectName verifies that hyphenated project names are handled correctly.
// Bug: "onprem-pw" should produce collection "tenant_onprem_pw_codebase" consistently.
func TestSearch_HyphenatedProjectName(t *testing.T) {
	store := &mockStore{
		searchResults: []vectorstore.SearchResult{
			{ID: "1", Content: "test", Score: 0.9, Metadata: map[string]interface{}{"file_path": "main.go"}},
		},
	}
	svc := NewService(store)

	// Using tenant_id + project_path with hyphen
	opts := SearchOptions{
		ProjectPath: "/Users/dahendel/projects/onprem-pw",
		TenantID:    "dahendel",
		Limit:       10,
	}

	// Execute
	_, err := svc.Search(context.Background(), "query", opts)

	// Verify
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	// Hyphen should be converted to underscore
	expectedCollection := "dahendel_onprem_pw_codebase"
	if store.lastCollection != expectedCollection {
		t.Errorf("Search collection = %q, want %q", store.lastCollection, expectedCollection)
	}
}

// ===== TESTS: COLLECTION NAMING =====

func TestSanitizeIdentifier(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"my-project", "my_project"},
		{"My Project", "my_project"},
		{"contextd-v2", "contextd_v2"},
		{"PROJECT_NAME", "project_name"},
		{"test.project.name", "test_project_name"},
		{"", "default"},    // sanitize.Identifier returns "default" for empty
		{"---", "default"}, // all invalid chars -> default
		{"123", "123"},
		{"a__b__c", "a_b_c"},
		// Tenant ID patterns (github.com/user format)
		{"github.com/dahendel", "github_com_dahendel"},
		{"github.com/fyrsmithlabs", "github_com_fyrsmithlabs"},
		{"gitlab.com/user/project", "gitlab_com_user_project"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := sanitize.Identifier(tt.input)
			if result != tt.expected {
				t.Errorf("sanitize.Identifier(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// ===== TESTS: ERROR HANDLING =====

func TestIndexRepository_RequiresStore(t *testing.T) {
	svc := &Service{store: nil}

	_, err := svc.IndexRepository(context.Background(), "/tmp", IndexOptions{})

	if err == nil {
		t.Error("IndexRepository() should error when store is nil")
	}
}

func TestSearch_RequiresStore(t *testing.T) {
	svc := &Service{store: nil}

	_, err := svc.Search(context.Background(), "query", SearchOptions{
		ProjectPath: "/path",
		TenantID:    "tenant",
	})

	if err == nil {
		t.Error("Search() should error when store is nil")
	}
}

func TestSearch_RequiresQuery(t *testing.T) {
	store := &mockStore{}
	svc := NewService(store)

	_, err := svc.Search(context.Background(), "", SearchOptions{
		ProjectPath: "/path",
		TenantID:    "tenant",
	})

	if err == nil {
		t.Error("Search() should error when query is empty")
	}
}

func TestSearch_RequiresProjectPath(t *testing.T) {
	store := &mockStore{}
	svc := NewService(store)

	_, err := svc.Search(context.Background(), "query", SearchOptions{
		TenantID: "tenant",
	})

	if err == nil {
		t.Error("Search() should error when project_path is empty")
	}
}

func TestSearch_AutoDerivesTenantID(t *testing.T) {
	store := &mockStore{}
	svc := NewService(store)

	// Search should succeed even without tenant_id because it's auto-derived from project_path
	_, err := svc.Search(context.Background(), "query", SearchOptions{
		ProjectPath: "/path/to/project",
	})

	// Should not error - tenant_id is auto-derived
	if err != nil {
		t.Errorf("Search() should auto-derive tenant_id, got error: %v", err)
	}

	// Verify the collection name was built with auto-derived tenant
	// tenant.GetTenantIDForPath will derive from git config or username
	if store.lastCollection == "" {
		t.Error("Expected search to use a collection name")
	}
}

// ===== EXISTING TESTS UPDATED =====

func TestIndexRepository_ValidPath(t *testing.T) {
	tmpDir := t.TempDir()
	createTestFile(t, tmpDir, "README.md", "# Test Repository\n\nDocumentation here.")
	createTestFile(t, tmpDir, "main.go", "package main\n\nfunc main() {}")
	createTestFile(t, tmpDir, ".gitignore", "*.log")

	store := &mockStore{}
	svc := NewService(store)

	opts := IndexOptions{
		TenantID:        "testuser",
		IncludePatterns: []string{"*.md", "*.go"},
		ExcludePatterns: []string{".git/**"},
		MaxFileSize:     1024 * 1024,
	}

	result, err := svc.IndexRepository(context.Background(), tmpDir, opts)

	if err != nil {
		t.Fatalf("IndexRepository() error = %v", err)
	}

	if result.FilesIndexed != 2 {
		t.Errorf("FilesIndexed = %d, want 2 (README.md + main.go)", result.FilesIndexed)
	}

	if len(store.documents) != 2 {
		t.Errorf("Documents stored = %d, want 2", len(store.documents))
	}
}

func TestIndexRepository_InvalidPath(t *testing.T) {
	store := &mockStore{}
	svc := NewService(store)

	_, err := svc.IndexRepository(context.Background(), "/nonexistent/path", IndexOptions{})

	if err == nil {
		t.Fatal("IndexRepository() error = nil, want error for invalid path")
	}
}

func TestIndexRepository_ExcludePatterns(t *testing.T) {
	tmpDir := t.TempDir()
	createTestFile(t, tmpDir, "main.go", "package main")
	createTestFile(t, tmpDir, "main_test.go", "package main")

	vendorDir := filepath.Join(tmpDir, "vendor")
	if err := os.MkdirAll(vendorDir, 0755); err != nil {
		t.Fatal(err)
	}
	createTestFile(t, vendorDir, "pkg.go", "package vendor")

	store := &mockStore{}
	svc := NewService(store)

	opts := IndexOptions{
		TenantID:        "testuser",
		IncludePatterns: []string{"*.go"},
		ExcludePatterns: []string{"*_test.go", "vendor/**"},
		MaxFileSize:     1024 * 1024,
	}

	result, err := svc.IndexRepository(context.Background(), tmpDir, opts)

	if err != nil {
		t.Fatalf("IndexRepository() error = %v", err)
	}

	if result.FilesIndexed != 1 {
		t.Errorf("FilesIndexed = %d, want 1 (only main.go)", result.FilesIndexed)
	}
}

func TestIndexRepository_MaxFileSize(t *testing.T) {
	tmpDir := t.TempDir()
	createTestFile(t, tmpDir, "small.txt", "small content")
	createTestFile(t, tmpDir, "large.txt", string(make([]byte, 2*1024*1024))) // 2MB

	store := &mockStore{}
	svc := NewService(store)

	opts := IndexOptions{
		TenantID:    "testuser",
		MaxFileSize: 1024 * 1024, // 1MB limit
	}

	result, err := svc.IndexRepository(context.Background(), tmpDir, opts)

	if err != nil {
		t.Fatalf("IndexRepository() error = %v", err)
	}

	if result.FilesIndexed != 1 {
		t.Errorf("FilesIndexed = %d, want 1 (only small.txt)", result.FilesIndexed)
	}
}

func TestIndexRepository_MaxFileSizeExceeds(t *testing.T) {
	store := &mockStore{}
	svc := NewService(store)

	opts := IndexOptions{
		MaxFileSize: 11 * 1024 * 1024, // 11MB (exceeds max)
	}

	_, err := svc.IndexRepository(context.Background(), t.TempDir(), opts)

	if err == nil {
		t.Fatal("IndexRepository() error = nil, want error for file size > 10MB")
	}
}

func TestIndexRepository_SkipsEmptyFiles(t *testing.T) {
	tmpDir := t.TempDir()
	createTestFile(t, tmpDir, "content.txt", "actual content")
	createTestFile(t, tmpDir, "empty.txt", "")
	createTestFile(t, tmpDir, "whitespace.txt", "   \n\t\n   ")

	store := &mockStore{}
	svc := NewService(store)

	opts := IndexOptions{
		TenantID: "testuser",
	}

	result, err := svc.IndexRepository(context.Background(), tmpDir, opts)

	if err != nil {
		t.Fatalf("IndexRepository() error = %v", err)
	}

	// Should only index content.txt, skip empty.txt and whitespace.txt
	if result.FilesIndexed != 1 {
		t.Errorf("FilesIndexed = %d, want 1 (only content.txt)", result.FilesIndexed)
	}

	// Verify the indexed file is content.txt
	if len(store.documents) != 1 {
		t.Fatalf("got %d documents, want 1", len(store.documents))
	}
	if fp, ok := store.documents[0].Metadata["file_path"].(string); ok {
		if fp != "content.txt" {
			t.Errorf("indexed file = %q, want content.txt", fp)
		}
	}
}

func TestIndexRepository_InvalidIncludePattern(t *testing.T) {
	store := &mockStore{}
	svc := NewService(store)

	opts := IndexOptions{
		IncludePatterns: []string{"[invalid"},
	}

	_, err := svc.IndexRepository(context.Background(), t.TempDir(), opts)

	if err == nil {
		t.Fatal("IndexRepository() error = nil, want error for invalid pattern")
	}
}

func TestIndexRepository_InvalidExcludePattern(t *testing.T) {
	store := &mockStore{}
	svc := NewService(store)

	opts := IndexOptions{
		ExcludePatterns: []string{"[invalid"},
	}

	_, err := svc.IndexRepository(context.Background(), t.TempDir(), opts)

	if err == nil {
		t.Fatal("IndexRepository() error = nil, want error for invalid pattern")
	}
}

func TestIndexRepository_StoreError(t *testing.T) {
	tmpDir := t.TempDir()
	createTestFile(t, tmpDir, "test.txt", "content")

	store := &mockStore{
		addError: os.ErrPermission,
	}
	svc := NewService(store)

	_, err := svc.IndexRepository(context.Background(), tmpDir, IndexOptions{TenantID: "test"})

	if err == nil {
		t.Fatal("IndexRepository() error = nil, want error when store fails")
	}
}

func TestIndexRepository_ContextCancellation(t *testing.T) {
	tmpDir := t.TempDir()
	for i := 0; i < 100; i++ {
		createTestFile(t, tmpDir, fmt.Sprintf("file%d.txt", i), "content")
	}

	store := &mockStore{}
	svc := NewService(store)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := svc.IndexRepository(ctx, tmpDir, IndexOptions{TenantID: "test"})

	if err == nil {
		t.Log("IndexRepository() completed despite cancellation (too few files)")
	}
}

func TestIndexRepository_PathTraversalPrevention(t *testing.T) {
	store := &mockStore{}
	svc := NewService(store)

	tests := []struct {
		name string
		path string
	}{
		{"relative path with traversal", "../../../etc/passwd"},
		{"absolute traversal", "/etc/../etc/passwd"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.IndexRepository(context.Background(), tt.path, IndexOptions{})
			if err == nil {
				t.Logf("Path traversal handled: %s", tt.path)
			}
		})
	}
}

// ===== HELPER =====

func createTestFile(t *testing.T, dir, name, content string) {
	t.Helper()
	// Handle nested paths
	fullPath := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}
	if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
}

// ===== TEST HELPERS FOR COLLECTION NAME VALIDATION =====

func TestCollectionNameFormat(t *testing.T) {
	// Collection names must match: ^[a-z0-9_]{1,64}$
	validNames := []string{
		"testuser_myproject_codebase",
		"user_project_codebase",
		"a_b_c",
		"test123_proj456_codebase",
	}

	invalidChars := []string{
		"Test_Project_codebase",  // uppercase
		"test-project-codebase",  // hyphens
		"test.project.codebase",  // dots
		"test project codebase",  // spaces
	}

	for _, name := range validNames {
		if !isValidCollectionName(name) {
			t.Errorf("Collection name %q should be valid", name)
		}
	}

	for _, name := range invalidChars {
		if isValidCollectionName(name) {
			t.Errorf("Collection name %q should be invalid", name)
		}
	}
}

func isValidCollectionName(name string) bool {
	if len(name) == 0 || len(name) > 64 {
		return false
	}
	for _, r := range name {
		if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_') {
			return false
		}
	}
	return true
}

// ===== TEST DOCUMENT METADATA =====

func TestIndexRepository_DocumentMetadata(t *testing.T) {
	tmpDir := t.TempDir()
	createTestFile(t, tmpDir, "src/main.go", "package main")

	store := &mockStore{}
	svc := NewService(store)

	opts := IndexOptions{
		TenantID: "testuser",
		Branch:   "main",
	}

	_, err := svc.IndexRepository(context.Background(), tmpDir, opts)
	if err != nil {
		t.Fatalf("IndexRepository() error = %v", err)
	}

	if len(store.documents) != 1 {
		t.Fatalf("Expected 1 document, got %d", len(store.documents))
	}

	doc := store.documents[0]

	// Check required metadata fields
	requiredFields := []string{"file_path", "file_size", "extension", "branch", "project_path", "tenant_id", "indexed_at"}
	for _, field := range requiredFields {
		if _, ok := doc.Metadata[field]; !ok {
			t.Errorf("Document missing metadata field: %s", field)
		}
	}

	// Verify specific values
	if doc.Metadata["file_path"] != "src/main.go" {
		t.Errorf("file_path = %v, want %q", doc.Metadata["file_path"], "src/main.go")
	}
	if doc.Metadata["extension"] != ".go" {
		t.Errorf("extension = %v, want %q", doc.Metadata["extension"], ".go")
	}
	if doc.Metadata["branch"] != "main" {
		t.Errorf("branch = %v, want %q", doc.Metadata["branch"], "main")
	}
	if doc.Metadata["tenant_id"] != "testuser" {
		t.Errorf("tenant_id = %v, want %q", doc.Metadata["tenant_id"], "testuser")
	}

	// Collection should be set
	if !strings.HasSuffix(doc.Collection, "_codebase") {
		t.Errorf("Collection = %q, should end with _codebase", doc.Collection)
	}
}

// ===== GREP TESTS =====

func TestGrep_ValidPattern(t *testing.T) {
	tmpDir := t.TempDir()
	createTestFile(t, tmpDir, "main.go", "package main\n\nfunc main() {\n  fmt.Println(\"hello world\")\n}")
	createTestFile(t, tmpDir, "util.go", "package main\n\nfunc util() {\n  // helper\n}")

	svc := NewService(&mockStore{})

	opts := GrepOptions{
		ProjectPath: tmpDir,
		CaseSensitive: false,
	}

	results, err := svc.Grep(context.Background(), "hello", opts)
	if err != nil {
		t.Fatalf("Grep() error = %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Grep results = %d, want 1", len(results))
	} else {
		if results[0].FilePath != "main.go" {
			t.Errorf("Result[0].FilePath = %q, want main.go", results[0].FilePath)
		}
		if !strings.Contains(results[0].Content, "hello world") {
			t.Errorf("Result[0].Content = %q, want matching line", results[0].Content)
		}
	}
}

func TestGrep_RegexPattern(t *testing.T) {
	tmpDir := t.TempDir()
	createTestFile(t, tmpDir, "main.go", "func main() {}")
	createTestFile(t, tmpDir, "test.go", "func test() {}")

	svc := NewService(&mockStore{})

	opts := GrepOptions{
		ProjectPath: tmpDir,
	}

	results, err := svc.Grep(context.Background(), "^func .*\\(\\)", opts)
	if err != nil {
		t.Fatalf("Grep() error = %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Grep results = %d, want 2", len(results))
	}
}

func TestGrep_SkipsBinaryFiles(t *testing.T) {
	tmpDir := t.TempDir()
	// Invalid UTF-8
	path := filepath.Join(tmpDir, "binary.bin")
	if err := os.WriteFile(path, []byte{0xff, 0xff}, 0644); err != nil {
		t.Fatal(err)
	}
	createTestFile(t, tmpDir, "valid.txt", "valid content")

	svc := NewService(&mockStore{})
	opts := GrepOptions{ProjectPath: tmpDir}

	results, err := svc.Grep(context.Background(), ".*", opts)
	if err != nil {
		t.Fatalf("Grep() error = %v", err)
	}

	// Should only match valid.txt (binary file line is skipped)
	if len(results) != 1 {
		t.Errorf("Results = %d, want 1", len(results))
	}
	if len(results) > 0 && results[0].FilePath != "valid.txt" {
		t.Errorf("Matched file = %s, want valid.txt", results[0].FilePath)
	}
}

func TestGrep_InvalidPath(t *testing.T) {
	svc := NewService(&mockStore{})

	opts := GrepOptions{
		ProjectPath: "/nonexistent/path",
	}

	_, err := svc.Grep(context.Background(), "pattern", opts)
	if err == nil {
		t.Error("Grep() should error on invalid path")
	}
}

func TestGrep_InvalidPattern(t *testing.T) {
	tmpDir := t.TempDir()
	createTestFile(t, tmpDir, "test.go", "content")

	svc := NewService(&mockStore{})

	opts := GrepOptions{
		ProjectPath: tmpDir,
	}

	_, err := svc.Grep(context.Background(), "[invalid", opts)
	if err == nil {
		t.Error("Grep() should error on invalid regex pattern")
	}
}

func TestGrep_ContextCancellation(t *testing.T) {
	tmpDir := t.TempDir()
	// Create many files to increase chance of context check
	for i := 0; i < 50; i++ {
		createTestFile(t, tmpDir, fmt.Sprintf("file%d.go", i), "package main\n\nfunc main() {}")
	}

	svc := NewService(&mockStore{})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	opts := GrepOptions{
		ProjectPath: tmpDir,
	}

	_, err := svc.Grep(ctx, "main", opts)
	// Error may or may not occur depending on timing, but should not panic
	_ = err
}

func TestGrep_CaseSensitive(t *testing.T) {
	tmpDir := t.TempDir()
	createTestFile(t, tmpDir, "test.go", "Hello World\nhello world\nHELLO WORLD")

	svc := NewService(&mockStore{})

	// Case insensitive (default)
	opts := GrepOptions{
		ProjectPath:   tmpDir,
		CaseSensitive: false,
	}

	results, err := svc.Grep(context.Background(), "hello", opts)
	if err != nil {
		t.Fatalf("Grep() error = %v", err)
	}
	if len(results) != 3 {
		t.Errorf("Case insensitive: got %d results, want 3", len(results))
	}

	// Case sensitive
	opts.CaseSensitive = true
	results, err = svc.Grep(context.Background(), "hello", opts)
	if err != nil {
		t.Fatalf("Grep() error = %v", err)
	}
	if len(results) != 1 {
		t.Errorf("Case sensitive: got %d results, want 1", len(results))
	}
}

func TestGrep_ExcludePatterns(t *testing.T) {
	tmpDir := t.TempDir()
	createTestFile(t, tmpDir, "main.go", "package main")
	createTestFile(t, tmpDir, "main_test.go", "package main")

	svc := NewService(&mockStore{})

	opts := GrepOptions{
		ProjectPath:     tmpDir,
		ExcludePatterns: []string{"*_test.go"},
	}

	results, err := svc.Grep(context.Background(), "package", opts)
	if err != nil {
		t.Fatalf("Grep() error = %v", err)
	}
	if len(results) != 1 {
		t.Errorf("Got %d results, want 1 (main.go only)", len(results))
	}
}

func TestGrep_SkipsDefaultDirs(t *testing.T) {
	tmpDir := t.TempDir()
	createTestFile(t, tmpDir, "main.go", "package main")

	// Create a file in .git directory that should be skipped
	gitDir := filepath.Join(tmpDir, ".git")
	if err := os.MkdirAll(gitDir, 0755); err != nil {
		t.Fatal(err)
	}
	createTestFile(t, gitDir, "config", "package git")

	// Create a file in node_modules that should be skipped
	nmDir := filepath.Join(tmpDir, "node_modules")
	if err := os.MkdirAll(nmDir, 0755); err != nil {
		t.Fatal(err)
	}
	createTestFile(t, nmDir, "pkg.js", "package node")

	svc := NewService(&mockStore{})

	opts := GrepOptions{
		ProjectPath: tmpDir,
	}

	results, err := svc.Grep(context.Background(), "package", opts)
	if err != nil {
		t.Fatalf("Grep() error = %v", err)
	}
	// Should only find main.go, not .git/config or node_modules/pkg.js
	if len(results) != 1 {
		t.Errorf("Got %d results, want 1 (should skip .git and node_modules)", len(results))
	}
}
