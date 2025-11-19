package checkpoint

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/zap"

	"github.com/fyrsmithlabs/contextd/pkg/vectorstore"
)

// Mock vectorstore.Service for testing
type mockVectorStore struct {
	addDocsFunc       func(ctx context.Context, docs []vectorstore.Document) error
	searchFiltersFunc func(ctx context.Context, query string, k int, filters map[string]interface{}) ([]vectorstore.SearchResult, error)
}

func (m *mockVectorStore) AddDocuments(ctx context.Context, docs []vectorstore.Document) error {
	if m.addDocsFunc != nil {
		return m.addDocsFunc(ctx, docs)
	}
	return nil
}

func (m *mockVectorStore) Search(ctx context.Context, query string, k int) ([]vectorstore.SearchResult, error) {
	return nil, errors.New("not implemented")
}

func (m *mockVectorStore) SearchWithFilters(ctx context.Context, query string, k int, filters map[string]interface{}) ([]vectorstore.SearchResult, error) {
	if m.searchFiltersFunc != nil {
		return m.searchFiltersFunc(ctx, query, k, filters)
	}
	return nil, nil
}

func (m *mockVectorStore) DeleteDocuments(ctx context.Context, ids []string) error {
	return errors.New("not implemented")
}

func TestNewService(t *testing.T) {
	logger := zap.NewNop()

	svc := NewService(nil, logger)
	if svc == nil {
		t.Fatal("NewService() returned nil")
	}

	// Test with nil logger (should create nop logger)
	svcNoLogger := NewService(nil, nil)
	if svcNoLogger == nil {
		t.Fatal("NewService() with nil logger returned nil")
	}
}

func TestService_Save(t *testing.T) {
	tests := []struct {
		name       string
		checkpoint *Checkpoint
		mockFunc   func(ctx context.Context, docs []vectorstore.Document) error
		wantErr    bool
	}{
		{
			name: "valid checkpoint",
			checkpoint: &Checkpoint{
				ProjectPath: "/tmp/test",
				Summary:     "test checkpoint",
				Content:     "test content",
			},
			mockFunc: func(ctx context.Context, docs []vectorstore.Document) error {
				if len(docs) != 1 {
					t.Errorf("expected 1 document, got %d", len(docs))
				}
				doc := docs[0]
				if doc.ID == "" {
					t.Error("document ID not generated")
				}
				if doc.Content != "test checkpoint\n\ntest content" {
					t.Errorf("unexpected embed content: %s", doc.Content)
				}
				// Verify metadata
				if doc.Metadata["project_path"] != "/tmp/test" {
					t.Error("project_path not set in metadata")
				}
				if doc.Metadata["summary"] != "test checkpoint" {
					t.Error("summary not set in metadata")
				}
				return nil
			},
			wantErr: false,
		},
		{
			name: "invalid checkpoint (missing project path)",
			checkpoint: &Checkpoint{
				Summary: "test",
			},
			wantErr: true,
		},
		{
			name: "invalid checkpoint (missing summary)",
			checkpoint: &Checkpoint{
				ProjectPath: "/tmp/test",
			},
			wantErr: true,
		},
		{
			name: "vector store error",
			checkpoint: &Checkpoint{
				ProjectPath: "/tmp/test",
				Summary:     "test",
			},
			mockFunc: func(ctx context.Context, docs []vectorstore.Document) error {
				return errors.New("storage error")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockVectorStore{
				addDocsFunc: tt.mockFunc,
			}
			svc := &Service{
				vectorStore: mock,
				logger:      zap.NewNop(),
			}

			err := svc.Save(context.Background(), tt.checkpoint)
			if (err != nil) != tt.wantErr {
				t.Errorf("Save() error = %v, wantErr %v", err, tt.wantErr)
			}

			// If no error, verify ID and timestamps were set
			if err == nil {
				if tt.checkpoint.ID == "" {
					t.Error("Save() did not set checkpoint ID")
				}
				if tt.checkpoint.CreatedAt.IsZero() {
					t.Error("Save() did not set CreatedAt")
				}
				if tt.checkpoint.UpdatedAt.IsZero() {
					t.Error("Save() did not set UpdatedAt")
				}
			}
		})
	}
}

func TestService_Search(t *testing.T) {
	tests := []struct {
		name      string
		query     string
		opts      *SearchOptions
		mockFunc  func(ctx context.Context, query string, k int, filters map[string]interface{}) ([]vectorstore.SearchResult, error)
		wantCount int
		wantErr   bool
	}{
		{
			name:  "successful search",
			query: "test query",
			opts: &SearchOptions{
				ProjectPath: "/tmp/test",
				Limit:       10,
			},
			mockFunc: func(ctx context.Context, query string, k int, filters map[string]interface{}) ([]vectorstore.SearchResult, error) {
				// Verify filters use correct Qdrant filter structure
				must, ok := filters["must"].([]map[string]interface{})
				if !ok || len(must) == 0 {
					t.Error("filters must contain 'must' array with project_hash filter")
				}
				// Verify project_hash is in the must conditions
				foundProjectHash := false
				for _, condition := range must {
					if key, ok := condition["key"].(string); ok && key == "project_hash" {
						if match, ok := condition["match"].(map[string]interface{}); ok {
							if _, hasValue := match["value"]; hasValue {
								foundProjectHash = true
								break
							}
						}
					}
				}
				if !foundProjectHash {
					t.Error("project_hash filter not found in must conditions")
				}
				return []vectorstore.SearchResult{
					{
						ID:      "ckpt-1",
						Content: "test checkpoint",
						Score:   0.9,
						Metadata: map[string]interface{}{
							"id":           "ckpt-1",
							"project_path": "/tmp/test",
							"summary":      "test checkpoint",
							"created_at":   "2025-01-15T10:00:00Z",
							"updated_at":   "2025-01-15T10:00:00Z",
						},
					},
				}, nil
			},
			wantCount: 1,
			wantErr:   false,
		},
		{
			name:  "empty query",
			query: "",
			opts: &SearchOptions{
				ProjectPath: "/tmp/test",
			},
			wantErr: true,
		},
		{
			name:    "invalid options (missing project path)",
			query:   "test",
			opts:    &SearchOptions{},
			wantErr: true,
		},
		{
			name:  "filter by min score",
			query: "test",
			opts: &SearchOptions{
				ProjectPath: "/tmp/test",
				MinScore:    0.8,
			},
			mockFunc: func(ctx context.Context, query string, k int, filters map[string]interface{}) ([]vectorstore.SearchResult, error) {
				return []vectorstore.SearchResult{
					{
						ID:    "ckpt-1",
						Score: 0.9, // Above threshold
						Metadata: map[string]interface{}{
							"id":           "ckpt-1",
							"project_path": "/tmp/test",
							"summary":      "high score",
							"created_at":   "2025-01-15T10:00:00Z",
							"updated_at":   "2025-01-15T10:00:00Z",
						},
					},
					{
						ID:    "ckpt-2",
						Score: 0.6, // Below threshold - should be filtered
						Metadata: map[string]interface{}{
							"id":           "ckpt-2",
							"project_path": "/tmp/test",
							"summary":      "low score",
							"created_at":   "2025-01-15T10:00:00Z",
							"updated_at":   "2025-01-15T10:00:00Z",
						},
					},
				}, nil
			},
			wantCount: 1, // Only one result above threshold
			wantErr:   false,
		},
		{
			name:  "vector store error",
			query: "test",
			opts: &SearchOptions{
				ProjectPath: "/tmp/test",
			},
			mockFunc: func(ctx context.Context, query string, k int, filters map[string]interface{}) ([]vectorstore.SearchResult, error) {
				return nil, errors.New("search error")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockVectorStore{
				searchFiltersFunc: tt.mockFunc,
			}
			svc := &Service{
				vectorStore: mock,
				logger:      zap.NewNop(),
			}

			results, err := svc.Search(context.Background(), tt.query, tt.opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("Search() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && len(results) != tt.wantCount {
				t.Errorf("Search() returned %d results, want %d", len(results), tt.wantCount)
			}

			// Verify results have valid checkpoints
			if err == nil {
				for _, result := range results {
					if result.Checkpoint == nil {
						t.Error("Search() returned nil checkpoint")
					}
					if result.Checkpoint.ID == "" {
						t.Error("Search() returned checkpoint with empty ID")
					}
				}
			}
		})
	}
}

func TestService_List(t *testing.T) {
	tests := []struct {
		name      string
		opts      *ListOptions
		mockFunc  func(ctx context.Context, query string, k int, filters map[string]interface{}) ([]vectorstore.SearchResult, error)
		wantCount int
		wantErr   bool
	}{
		{
			name: "successful list",
			opts: &ListOptions{
				ProjectPath: "/tmp/test",
				Limit:       20,
			},
			mockFunc: func(ctx context.Context, query string, k int, filters map[string]interface{}) ([]vectorstore.SearchResult, error) {
				return []vectorstore.SearchResult{
					{
						ID:    "ckpt-1",
						Score: 0.9,
						Metadata: map[string]interface{}{
							"id":           "ckpt-1",
							"project_path": "/tmp/test",
							"summary":      "checkpoint 1",
							"created_at":   "2025-01-15T10:00:00Z",
							"updated_at":   "2025-01-15T10:00:00Z",
						},
					},
					{
						ID:    "ckpt-2",
						Score: 0.8,
						Metadata: map[string]interface{}{
							"id":           "ckpt-2",
							"project_path": "/tmp/test",
							"summary":      "checkpoint 2",
							"created_at":   "2025-01-15T09:00:00Z",
							"updated_at":   "2025-01-15T09:00:00Z",
						},
					},
				}, nil
			},
			wantCount: 2,
			wantErr:   false,
		},
		{
			name:    "invalid options (missing project path)",
			opts:    &ListOptions{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockVectorStore{
				searchFiltersFunc: tt.mockFunc,
			}
			svc := &Service{
				vectorStore: mock,
				logger:      zap.NewNop(),
			}

			checkpoints, err := svc.List(context.Background(), tt.opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("List() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && len(checkpoints) != tt.wantCount {
				t.Errorf("List() returned %d checkpoints, want %d", len(checkpoints), tt.wantCount)
			}
		})
	}
}

func TestService_Get(t *testing.T) {
	tests := []struct {
		name        string
		projectPath string
		id          string
		mockFunc    func(ctx context.Context, query string, k int, filters map[string]interface{}) ([]vectorstore.SearchResult, error)
		wantErr     bool
	}{
		{
			name:        "successful get",
			projectPath: "/tmp/test",
			id:          "ckpt-1",
			mockFunc: func(ctx context.Context, query string, k int, filters map[string]interface{}) ([]vectorstore.SearchResult, error) {
				// Verify filters use correct Qdrant filter structure
				must, ok := filters["must"].([]map[string]interface{})
				if !ok || len(must) == 0 {
					t.Error("filters must contain 'must' array")
				}
				// Verify both project_hash and id filters are present
				foundProjectHash := false
				foundID := false
				for _, condition := range must {
					if key, ok := condition["key"].(string); ok {
						if key == "project_hash" {
							if match, ok := condition["match"].(map[string]interface{}); ok {
								if _, hasValue := match["value"]; hasValue {
									foundProjectHash = true
								}
							}
						} else if key == "id" {
							if match, ok := condition["match"].(map[string]interface{}); ok {
								if value, ok := match["value"].(string); ok && value == "ckpt-1" {
									foundID = true
								}
							}
						}
					}
				}
				if !foundProjectHash {
					t.Error("project_hash filter not found in must conditions")
				}
				if !foundID {
					t.Error("id filter not found in must conditions or has wrong value")
				}
				return []vectorstore.SearchResult{
					{
						ID: "ckpt-1",
						Metadata: map[string]interface{}{
							"id":           "ckpt-1",
							"project_path": "/tmp/test",
							"summary":      "test checkpoint",
							"created_at":   "2025-01-15T10:00:00Z",
							"updated_at":   "2025-01-15T10:00:00Z",
						},
					},
				}, nil
			},
			wantErr: false,
		},
		{
			name:        "checkpoint not found",
			projectPath: "/tmp/test",
			id:          "nonexistent",
			mockFunc: func(ctx context.Context, query string, k int, filters map[string]interface{}) ([]vectorstore.SearchResult, error) {
				return []vectorstore.SearchResult{}, nil // Empty results
			},
			wantErr: true,
		},
		{
			name:        "missing project path",
			projectPath: "",
			id:          "ckpt-1",
			wantErr:     true,
		},
		{
			name:        "missing id",
			projectPath: "/tmp/test",
			id:          "",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockVectorStore{
				searchFiltersFunc: tt.mockFunc,
			}
			svc := &Service{
				vectorStore: mock,
				logger:      zap.NewNop(),
			}

			checkpoint, err := svc.Get(context.Background(), tt.projectPath, tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("Get() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && checkpoint == nil {
				t.Error("Get() returned nil checkpoint without error")
			}
			if !tt.wantErr && checkpoint.ID != tt.id {
				t.Errorf("Get() checkpoint ID = %s, want %s", checkpoint.ID, tt.id)
			}
		})
	}
}

func Test_projectHash(t *testing.T) {
	hash1 := projectHash("/tmp/test")
	hash2 := projectHash("/tmp/test")
	hash3 := projectHash("/tmp/other")

	// Same path should produce same hash
	if hash1 != hash2 {
		t.Error("projectHash() not deterministic")
	}

	// Different paths should produce different hashes
	if hash1 == hash3 {
		t.Error("projectHash() collision")
	}

	// Hash should be 16 characters (hex string)
	if len(hash1) != 16 {
		t.Errorf("projectHash() length = %d, want 16", len(hash1))
	}
}

func Test_hasAnyTag(t *testing.T) {
	tests := []struct {
		name           string
		checkpointTags []string
		searchTags     []string
		want           bool
	}{
		{
			name:           "has matching tag",
			checkpointTags: []string{"auth", "security"},
			searchTags:     []string{"auth"},
			want:           true,
		},
		{
			name:           "no matching tags",
			checkpointTags: []string{"auth", "security"},
			searchTags:     []string{"database"},
			want:           false,
		},
		{
			name:           "empty checkpoint tags",
			checkpointTags: []string{},
			searchTags:     []string{"auth"},
			want:           false,
		},
		{
			name:           "empty search tags",
			checkpointTags: []string{"auth"},
			searchTags:     []string{},
			want:           false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasAnyTag(tt.checkpointTags, tt.searchTags); got != tt.want {
				t.Errorf("hasAnyTag() = %v, want %v", got, tt.want)
			}
		})
	}
}
