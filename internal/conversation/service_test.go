package conversation

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.uber.org/zap"

	"github.com/fyrsmithlabs/contextd/internal/vectorstore"
)

// mockStore implements vectorstore.Store for testing.
type mockStore struct {
	collections   map[string]bool
	documents     []vectorstore.Document
	searchResults []vectorstore.SearchResult
	isolationMode vectorstore.IsolationMode
}

func newMockStore() *mockStore {
	return &mockStore{
		collections:   make(map[string]bool),
		documents:     []vectorstore.Document{},
		isolationMode: vectorstore.NewNoIsolation(),
	}
}

func (m *mockStore) AddDocuments(ctx context.Context, docs []vectorstore.Document) ([]string, error) {
	ids := make([]string, len(docs))
	for i, doc := range docs {
		m.documents = append(m.documents, doc)
		ids[i] = doc.ID
	}
	return ids, nil
}

func (m *mockStore) Search(ctx context.Context, query string, k int) ([]vectorstore.SearchResult, error) {
	return m.searchResults, nil
}

func (m *mockStore) SearchWithFilters(ctx context.Context, query string, k int, filters map[string]interface{}) ([]vectorstore.SearchResult, error) {
	return m.searchResults, nil
}

func (m *mockStore) SearchInCollection(ctx context.Context, collection string, query string, k int, filters map[string]interface{}) ([]vectorstore.SearchResult, error) {
	return m.searchResults, nil
}

func (m *mockStore) DeleteDocuments(ctx context.Context, ids []string) error {
	return nil
}

func (m *mockStore) DeleteDocumentsFromCollection(ctx context.Context, collectionName string, ids []string) error {
	return nil
}

func (m *mockStore) CreateCollection(ctx context.Context, name string, vectorSize int) error {
	m.collections[name] = true
	return nil
}

func (m *mockStore) DeleteCollection(ctx context.Context, name string) error {
	delete(m.collections, name)
	return nil
}

func (m *mockStore) CollectionExists(ctx context.Context, name string) (bool, error) {
	return m.collections[name], nil
}

func (m *mockStore) ListCollections(ctx context.Context) ([]string, error) {
	names := make([]string, 0, len(m.collections))
	for name := range m.collections {
		names = append(names, name)
	}
	return names, nil
}

func (m *mockStore) GetCollectionInfo(ctx context.Context, collectionName string) (*vectorstore.CollectionInfo, error) {
	if _, ok := m.collections[collectionName]; !ok {
		return nil, vectorstore.ErrCollectionNotFound
	}
	return &vectorstore.CollectionInfo{
		Name:       collectionName,
		PointCount: len(m.documents),
		VectorSize: 384,
	}, nil
}

func (m *mockStore) ExactSearch(ctx context.Context, collectionName string, query string, k int) ([]vectorstore.SearchResult, error) {
	return m.searchResults, nil
}

func (m *mockStore) SetIsolationMode(mode vectorstore.IsolationMode) {
	m.isolationMode = mode
}

func (m *mockStore) IsolationMode() vectorstore.IsolationMode {
	return m.isolationMode
}

func (m *mockStore) Close() error {
	return nil
}

// mockScrubber implements Scrubber for testing.
type mockScrubber struct{}

func (m *mockScrubber) Scrub(content string) ScrubResult {
	return &mockScrubResult{scrubbed: content}
}

type mockScrubResult struct {
	scrubbed string
}

func (m *mockScrubResult) GetScrubbed() string {
	return m.scrubbed
}

func TestService_Index(t *testing.T) {
	// Create temp directory with test conversation
	tmpDir := t.TempDir()

	testContent := `{"type":"user","message":{"id":"msg1","content":[{"type":"text","text":"Hello"}],"model":"claude-3","role":"user"},"timestamp":"2025-01-01T10:00:00Z","uuid":"uuid-1"}
{"type":"assistant","message":{"id":"msg2","content":[{"type":"text","text":"Hi there!"}],"model":"claude-3","role":"assistant"},"timestamp":"2025-01-01T10:00:30Z","uuid":"uuid-2"}`

	sessionFile := filepath.Join(tmpDir, "test-session.jsonl")
	if err := os.WriteFile(sessionFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	store := newMockStore()
	logger := zap.NewNop()

	service := NewService(store, &mockScrubber{}, logger, ServiceConfig{
		ConversationsPath: tmpDir,
	})

	result, err := service.Index(context.Background(), IndexOptions{
		TenantID:    "test-tenant",
		ProjectPath: tmpDir,
	})
	if err != nil {
		t.Fatalf("Index() error = %v", err)
	}

	if result.SessionsIndexed != 1 {
		t.Errorf("result.SessionsIndexed = %d, want 1", result.SessionsIndexed)
	}
	if result.MessagesIndexed != 2 {
		t.Errorf("result.MessagesIndexed = %d, want 2", result.MessagesIndexed)
	}
}

func TestService_Index_EmptyDir(t *testing.T) {
	tmpDir := t.TempDir()

	store := newMockStore()
	logger := zap.NewNop()

	service := NewService(store, &mockScrubber{}, logger, ServiceConfig{
		ConversationsPath: tmpDir,
	})

	_, err := service.Index(context.Background(), IndexOptions{
		TenantID:    "test-tenant",
		ProjectPath: tmpDir,
	})

	// Empty directories (no JSONL files) should now return an error
	if err == nil {
		t.Error("Index() expected error for empty directory, got nil")
	}
	if err != nil && !strings.Contains(err.Error(), "no conversation files found") {
		t.Errorf("Index() error = %v, want error containing 'no conversation files found'", err)
	}
}

func TestService_Index_NonexistentDir(t *testing.T) {
	store := newMockStore()
	logger := zap.NewNop()

	service := NewService(store, &mockScrubber{}, logger, ServiceConfig{
		ConversationsPath: "/nonexistent/path",
	})

	_, err := service.Index(context.Background(), IndexOptions{
		TenantID:    "test-tenant",
		ProjectPath: "/nonexistent/path",
	})
	if err == nil {
		t.Error("Index() expected error for nonexistent directory")
	}
}

func TestService_Index_FilterSessions(t *testing.T) {
	tmpDir := t.TempDir()

	// Create multiple session files
	session1 := `{"type":"user","message":{"id":"msg1","content":[{"type":"text","text":"S1"}],"model":"claude-3","role":"user"},"timestamp":"2025-01-01T10:00:00Z","uuid":"uuid-1"}`
	session2 := `{"type":"user","message":{"id":"msg2","content":[{"type":"text","text":"S2"}],"model":"claude-3","role":"user"},"timestamp":"2025-01-01T11:00:00Z","uuid":"uuid-2"}`

	if err := os.WriteFile(filepath.Join(tmpDir, "session1.jsonl"), []byte(session1), 0644); err != nil {
		t.Fatalf("failed to write session1: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "session2.jsonl"), []byte(session2), 0644); err != nil {
		t.Fatalf("failed to write session2: %v", err)
	}

	store := newMockStore()
	logger := zap.NewNop()

	service := NewService(store, &mockScrubber{}, logger, ServiceConfig{
		ConversationsPath: tmpDir,
	})

	// Index only session1
	result, err := service.Index(context.Background(), IndexOptions{
		TenantID:    "test-tenant",
		ProjectPath: tmpDir,
		SessionIDs:  []string{"session1"},
	})
	if err != nil {
		t.Fatalf("Index() error = %v", err)
	}

	if result.SessionsIndexed != 1 {
		t.Errorf("result.SessionsIndexed = %d, want 1", result.SessionsIndexed)
	}
}

func TestService_Search(t *testing.T) {
	store := newMockStore()
	store.searchResults = []vectorstore.SearchResult{
		{
			ID:      "doc1",
			Content: "Test content",
			Score:   0.95,
			Metadata: map[string]interface{}{
				"session_id": "session1",
				"type":       "message",
				"timestamp":  float64(1704106800),
			},
		},
	}

	logger := zap.NewNop()
	service := NewService(store, &mockScrubber{}, logger, ServiceConfig{})

	result, err := service.Search(context.Background(), SearchOptions{
		TenantID:    "test-tenant",
		ProjectPath: "/test/project",
		Query:       "test query",
		Limit:       10,
	})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	if result.Total != 1 {
		t.Errorf("result.Total = %d, want 1", result.Total)
	}
	if result.Query != "test query" {
		t.Errorf("result.Query = %q, want 'test query'", result.Query)
	}
	if len(result.Results) != 1 {
		t.Fatalf("len(result.Results) = %d, want 1", len(result.Results))
	}
	// Use approximate comparison for float64
	if result.Results[0].Score < 0.94 || result.Results[0].Score > 0.96 {
		t.Errorf("result.Results[0].Score = %f, want ~0.95", result.Results[0].Score)
	}
}

func TestService_Search_WithFilters(t *testing.T) {
	store := newMockStore()
	logger := zap.NewNop()
	service := NewService(store, &mockScrubber{}, logger, ServiceConfig{})

	// Test with various filters
	_, err := service.Search(context.Background(), SearchOptions{
		TenantID:    "test-tenant",
		ProjectPath: "/test/project",
		Query:       "test query",
		Types:       []DocumentType{TypeMessage},
		Tags:        []string{"important"},
		FilePath:    "/path/to/file.go",
		Domain:      "backend",
	})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
}

func TestService_Search_DefaultLimit(t *testing.T) {
	store := newMockStore()
	logger := zap.NewNop()
	service := NewService(store, &mockScrubber{}, logger, ServiceConfig{})

	result, err := service.Search(context.Background(), SearchOptions{
		TenantID:    "test-tenant",
		ProjectPath: "/test/project",
		Query:       "test query",
		// Limit not set, should default to 10
	})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	// Just verify it doesn't error - the default limit is applied internally
	_ = result
}

func TestService_CollectionName(t *testing.T) {
	store := newMockStore()
	logger := zap.NewNop()
	service := NewService(store, &mockScrubber{}, logger, ServiceConfig{})

	tests := []struct {
		name        string
		tenantID    string
		projectPath string
		want        string
	}{
		{
			name:        "simple names",
			tenantID:    "tenant1",
			projectPath: "/path/to/my-project",
			want:        "tenant1_my_project_conversations",
		},
		{
			name:        "underscores and spaces",
			tenantID:    "org_123",
			projectPath: "/home/user/Test Project",
			want:        "org_123_test_project_conversations",
		},
		{
			name:        "special characters in tenant",
			tenantID:    "org@domain.com",
			projectPath: "/path/to/project",
			want:        "orgdomain_com_project_conversations",
		},
		{
			name:        "dots and hyphens",
			tenantID:    "my.org",
			projectPath: "/path/my.project-v2",
			want:        "my_org_my_project_v2_conversations",
		},
		{
			name:        "unicode characters use hash fallback",
			tenantID:    "租户",
			projectPath: "/path/项目",
			// Each unicode string gets a unique hash, so we test the pattern not exact value
			want:        "", // Special case: verify in test body
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := service.collectionName(tt.tenantID, tt.projectPath)
			if tt.want == "" {
				// Special case: verify hash pattern for unicode input
				// Should be "h_<hash1>_h_<hash2>_conversations"
				if !strings.HasPrefix(got, "h_") || !strings.Contains(got, "_conversations") {
					t.Errorf("collectionName(%q, %q) = %q, want hash-based collection name", tt.tenantID, tt.projectPath, got)
				}
			} else if got != tt.want {
				t.Errorf("collectionName(%q, %q) = %q, want %q", tt.tenantID, tt.projectPath, got, tt.want)
			}
		})
	}
}

func TestSanitizeForCollectionName(t *testing.T) {
	tests := []struct {
		input       string
		want        string
		wantHashPrefix bool // When true, verify starts with "h_" instead of exact match
	}{
		{"simple", "simple", false},
		{"with-dash", "with_dash", false},
		{"with space", "with_space", false},
		{"with.dot", "with_dot", false},
		{"UPPERCASE", "uppercase", false},
		{"mix123", "mix123", false},
		{"special@#$chars", "specialchars", false},
		{"", "", true},        // Empty string gets hash
		{"!@#$%", "", true},   // All special chars gets hash
		{"test__double", "test__double", false},
	}

	for _, tt := range tests {
		name := tt.input
		if name == "" {
			name = "empty"
		}
		t.Run(name, func(t *testing.T) {
			got := sanitizeForCollectionName(tt.input)
			if tt.wantHashPrefix {
				if !strings.HasPrefix(got, "h_") || len(got) != 34 { // "h_" + 32 hex chars (16 bytes)
					t.Errorf("sanitizeForCollectionName(%q) = %q, want h_<32 hex chars>", tt.input, got)
				}
			} else if got != tt.want {
				t.Errorf("sanitizeForCollectionName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSanitizeForCollectionName_DifferentUnicode(t *testing.T) {
	// Verify different unicode strings produce different hashes
	hash1 := sanitizeForCollectionName("租户")
	hash2 := sanitizeForCollectionName("项目")
	hash3 := sanitizeForCollectionName("租户") // Same as hash1

	if hash1 == hash2 {
		t.Errorf("Different unicode strings should produce different hashes: %q == %q", hash1, hash2)
	}
	if hash1 != hash3 {
		t.Errorf("Same unicode strings should produce same hash: %q != %q", hash1, hash3)
	}
	if !strings.HasPrefix(hash1, "h_") {
		t.Errorf("Unicode string should produce hash prefix: %q", hash1)
	}
}

func TestService_NilScrubber(t *testing.T) {
	tmpDir := t.TempDir()

	testContent := `{"type":"user","message":{"id":"msg1","content":[{"type":"text","text":"Hello"}],"model":"claude-3","role":"user"},"timestamp":"2025-01-01T10:00:00Z","uuid":"uuid-1"}`

	sessionFile := filepath.Join(tmpDir, "test-session.jsonl")
	if err := os.WriteFile(sessionFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	store := newMockStore()
	logger := zap.NewNop()

	// Create service with nil scrubber
	service := NewService(store, nil, logger, ServiceConfig{
		ConversationsPath: tmpDir,
	})

	result, err := service.Index(context.Background(), IndexOptions{
		TenantID:    "test-tenant",
		ProjectPath: tmpDir,
	})
	if err != nil {
		t.Fatalf("Index() error = %v", err)
	}

	if result.MessagesIndexed != 1 {
		t.Errorf("result.MessagesIndexed = %d, want 1", result.MessagesIndexed)
	}
}

func TestService_DefaultConversationsPath(t *testing.T) {
	store := newMockStore()
	logger := zap.NewNop()

	// Create service with empty conversations path
	service := NewService(store, &mockScrubber{}, logger, ServiceConfig{})

	// Should default to ~/.claude/projects
	home, _ := os.UserHomeDir()
	expectedPath := filepath.Join(home, ".claude", "projects")

	if service.conversationsPath != expectedPath {
		t.Errorf("conversationsPath = %q, want %q", service.conversationsPath, expectedPath)
	}
}
