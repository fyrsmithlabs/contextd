package remediation

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
		name        string
		remediation *Remediation
		mockFunc    func(ctx context.Context, docs []vectorstore.Document) error
		wantErr     bool
	}{
		{
			name: "valid remediation",
			remediation: &Remediation{
				ProjectPath: "/tmp/test",
				ErrorMsg:    "connection refused",
				Solution:    "start the server",
				Context:     "additional details",
			},
			mockFunc: func(ctx context.Context, docs []vectorstore.Document) error {
				if len(docs) != 1 {
					t.Errorf("expected 1 document, got %d", len(docs))
				}
				doc := docs[0]
				if doc.ID == "" {
					t.Error("document ID not generated")
				}
				// Content includes error msg + solution + context
				expected := "connection refused\n\nstart the server\n\nadditional details"
				if doc.Content != expected {
					t.Errorf("unexpected embed content: got %q, want %q", doc.Content, expected)
				}
				// Verify metadata
				if doc.Metadata["project_path"] != "/tmp/test" {
					t.Error("project_path not set in metadata")
				}
				if doc.Metadata["error_msg"] != "connection refused" {
					t.Error("error_msg not set in metadata")
				}
				// Check patterns were extracted
				patterns, ok := doc.Metadata["patterns"]
				if !ok {
					t.Error("patterns not set in metadata")
				}
				if patterns == nil {
					t.Error("patterns is nil")
				}
				return nil
			},
			wantErr: false,
		},
		{
			name: "invalid remediation (missing project path)",
			remediation: &Remediation{
				ErrorMsg: "error",
				Solution: "fix",
			},
			wantErr: true,
		},
		{
			name: "invalid remediation (missing error message)",
			remediation: &Remediation{
				ProjectPath: "/tmp/test",
				Solution:    "fix",
			},
			wantErr: true,
		},
		{
			name: "invalid remediation (missing solution)",
			remediation: &Remediation{
				ProjectPath: "/tmp/test",
				ErrorMsg:    "error",
			},
			wantErr: true,
		},
		{
			name: "vector store error",
			remediation: &Remediation{
				ProjectPath: "/tmp/test",
				ErrorMsg:    "error",
				Solution:    "fix",
			},
			mockFunc: func(ctx context.Context, docs []vectorstore.Document) error {
				return errors.New("database error")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStore := &mockVectorStore{
				addDocsFunc: tt.mockFunc,
			}

			svc := NewService(mockStore, zap.NewNop())
			err := svc.Save(context.Background(), tt.remediation)

			if (err != nil) != tt.wantErr {
				t.Errorf("Save() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Verify ID and timestamps were set on success
			if err == nil {
				if tt.remediation.ID == "" {
					t.Error("ID not set after save")
				}
				if tt.remediation.CreatedAt.IsZero() {
					t.Error("CreatedAt not set after save")
				}
				// Verify patterns were extracted
				if len(tt.remediation.Patterns) == 0 {
					// Only some error messages have patterns
					t.Logf("No patterns extracted for error: %s", tt.remediation.ErrorMsg)
				}
			}
		})
	}
}

func TestService_Search(t *testing.T) {
	tests := []struct {
		name      string
		opts      *SearchOptions
		mockFunc  func(ctx context.Context, query string, k int, filters map[string]interface{}) ([]vectorstore.SearchResult, error)
		wantErr   bool
		wantCount int
	}{
		{
			name: "valid search with results",
			opts: &SearchOptions{
				ProjectPath: "/tmp/test",
				Limit:       5,
				Threshold:   0.5, // Set low threshold to include both results
			},
			mockFunc: func(ctx context.Context, query string, k int, filters map[string]interface{}) ([]vectorstore.SearchResult, error) {
				// Verify project filter is set
				if filters["project_path"] != "/tmp/test" {
					t.Errorf("project_path filter not set correctly: %v", filters)
				}
				// Return mock results
				return []vectorstore.SearchResult{
					{
						ID:    "rem_1",
						Score: 0.85,
						Metadata: map[string]interface{}{
							"project_path": "/tmp/test",
							"error_msg":    "connection refused",
							"solution":     "start server",
							"context":      "test context",
							"patterns":     []string{"connection refused"},
						},
					},
					{
						ID:    "rem_2",
						Score: 0.75,
						Metadata: map[string]interface{}{
							"project_path": "/tmp/test",
							"error_msg":    "timeout",
							"solution":     "increase timeout",
							"patterns":     []string{"timeout"},
						},
					},
				}, nil
			},
			wantErr:   false,
			wantCount: 2,
		},
		{
			name: "invalid options (missing project path)",
			opts: &SearchOptions{
				Limit: 5,
			},
			wantErr: true,
		},
		{
			name: "no results",
			opts: &SearchOptions{
				ProjectPath: "/tmp/test",
				Limit:       5,
			},
			mockFunc: func(ctx context.Context, query string, k int, filters map[string]interface{}) ([]vectorstore.SearchResult, error) {
				return []vectorstore.SearchResult{}, nil
			},
			wantErr:   false,
			wantCount: 0,
		},
		{
			name: "vector store error",
			opts: &SearchOptions{
				ProjectPath: "/tmp/test",
				Limit:       5,
			},
			mockFunc: func(ctx context.Context, query string, k int, filters map[string]interface{}) ([]vectorstore.SearchResult, error) {
				return nil, errors.New("database error")
			},
			wantErr: true,
		},
		{
			name: "apply defaults",
			opts: &SearchOptions{
				ProjectPath: "/tmp/test",
				// No limit or threshold specified
			},
			mockFunc: func(ctx context.Context, query string, k int, filters map[string]interface{}) ([]vectorstore.SearchResult, error) {
				if k != DefaultLimit {
					t.Errorf("expected default limit %d, got %d", DefaultLimit, k)
				}
				return []vectorstore.SearchResult{}, nil
			},
			wantErr:   false,
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStore := &mockVectorStore{
				searchFiltersFunc: tt.mockFunc,
			}

			svc := NewService(mockStore, zap.NewNop())
			results, err := svc.Search(context.Background(), "test query", tt.opts)

			if (err != nil) != tt.wantErr {
				t.Errorf("Search() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err == nil && len(results) != tt.wantCount {
				t.Errorf("Search() returned %d results, want %d", len(results), tt.wantCount)
			}

			// Verify results structure
			if err == nil && len(results) > 0 {
				for _, res := range results {
					if res.Remediation == nil {
						t.Error("Search result missing Remediation")
					}
					if res.Remediation.ID == "" {
						t.Error("Search result Remediation missing ID")
					}
					if res.Score < 0 || res.Score > 1 {
						t.Errorf("Invalid score: %f (must be 0-1)", res.Score)
					}
				}
			}
		})
	}
}

func TestService_List(t *testing.T) {
	tests := []struct {
		name        string
		projectPath string
		limit       int
		mockFunc    func(ctx context.Context, query string, k int, filters map[string]interface{}) ([]vectorstore.SearchResult, error)
		wantErr     bool
		wantCount   int
	}{
		{
			name:        "valid list",
			projectPath: "/tmp/test",
			limit:       10,
			mockFunc: func(ctx context.Context, query string, k int, filters map[string]interface{}) ([]vectorstore.SearchResult, error) {
				if k != 10 {
					t.Errorf("expected limit 10, got %d", k)
				}
				// Verify project filter
				if filters["project_path"] != "/tmp/test" {
					t.Errorf("project_path filter not set correctly: %v", filters)
				}
				return []vectorstore.SearchResult{
					{
						ID:    "rem_1",
						Score: 1.0, // List results have score 1.0
						Metadata: map[string]interface{}{
							"project_path": "/tmp/test",
							"error_msg":    "error 1",
							"solution":     "fix 1",
						},
					},
				}, nil
			},
			wantErr:   false,
			wantCount: 1,
		},
		{
			name:        "missing project path",
			projectPath: "",
			limit:       10,
			wantErr:     true,
		},
		{
			name:        "vector store error",
			projectPath: "/tmp/test",
			limit:       10,
			mockFunc: func(ctx context.Context, query string, k int, filters map[string]interface{}) ([]vectorstore.SearchResult, error) {
				return nil, errors.New("database error")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStore := &mockVectorStore{
				searchFiltersFunc: tt.mockFunc,
			}

			svc := NewService(mockStore, zap.NewNop())
			results, err := svc.List(context.Background(), tt.projectPath, tt.limit)

			if (err != nil) != tt.wantErr {
				t.Errorf("List() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err == nil && len(results) != tt.wantCount {
				t.Errorf("List() returned %d results, want %d", len(results), tt.wantCount)
			}
		})
	}
}

func TestPatternExtractionDuringSave(t *testing.T) {
	// Test that pattern extraction is called during save
	mockStore := &mockVectorStore{
		addDocsFunc: func(ctx context.Context, docs []vectorstore.Document) error {
			if len(docs) != 1 {
				return nil
			}
			patterns, ok := docs[0].Metadata["patterns"]
			if !ok {
				t.Error("patterns not extracted during save")
			}
			patternsSlice, ok := patterns.([]string)
			if !ok {
				t.Errorf("patterns is not []string: %T", patterns)
			}
			if len(patternsSlice) == 0 {
				t.Error("expected patterns to be extracted for connection refused error")
			}
			// Verify "connection refused" pattern was extracted
			found := false
			for _, p := range patternsSlice {
				if p == "connection refused" {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected 'connection refused' pattern, got %v", patternsSlice)
			}
			return nil
		},
	}

	svc := NewService(mockStore, zap.NewNop())
	rem := &Remediation{
		ProjectPath: "/tmp/test",
		ErrorMsg:    "dial tcp 127.0.0.1:8080: connection refused",
		Solution:    "start server",
	}

	if err := svc.Save(context.Background(), rem); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Verify patterns were set on the remediation
	if len(rem.Patterns) == 0 {
		t.Error("patterns not set on remediation after save")
	}
}
