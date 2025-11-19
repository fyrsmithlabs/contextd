package skills

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/fyrsmithlabs/contextd/pkg/vectorstore"
)

// mockVectorStore is a mock implementation of VectorStore for testing.
type mockVectorStore struct {
	addDocumentsFunc      func(ctx context.Context, docs []vectorstore.Document) error
	searchWithFiltersFunc func(ctx context.Context, query string, k int, filters map[string]interface{}) ([]vectorstore.SearchResult, error)
}

func (m *mockVectorStore) AddDocuments(ctx context.Context, docs []vectorstore.Document) error {
	if m.addDocumentsFunc != nil {
		return m.addDocumentsFunc(ctx, docs)
	}
	return nil
}

func (m *mockVectorStore) SearchWithFilters(ctx context.Context, query string, k int, filters map[string]interface{}) ([]vectorstore.SearchResult, error) {
	if m.searchWithFiltersFunc != nil {
		return m.searchWithFiltersFunc(ctx, query, k, filters)
	}
	return nil, nil
}

// Test helpers
func newTestSkill() *Skill {
	return &Skill{
		Name:        "Test Skill",
		Description: "A test skill for unit testing",
		Content:     "This is test content",
		Tags:        []string{"test", "unit"},
	}
}

// TestSkill_Validate tests skill validation logic.
func TestSkill_Validate(t *testing.T) {
	tests := []struct {
		name    string
		skill   *Skill
		wantErr bool
	}{
		{
			name:    "valid skill",
			skill:   newTestSkill(),
			wantErr: false,
		},
		{
			name: "missing name",
			skill: &Skill{
				Description: "Test",
				Content:     "Content",
			},
			wantErr: true,
		},
		{
			name: "name too long",
			skill: &Skill{
				Name:        string(make([]byte, 201)),
				Description: "Test",
				Content:     "Content",
			},
			wantErr: true,
		},
		{
			name: "missing description",
			skill: &Skill{
				Name:    "Test",
				Content: "Content",
			},
			wantErr: true,
		},
		{
			name: "description too long",
			skill: &Skill{
				Name:        "Test",
				Description: string(make([]byte, 2001)),
				Content:     "Content",
			},
			wantErr: true,
		},
		{
			name: "missing content",
			skill: &Skill{
				Name:        "Test",
				Description: "Description",
			},
			wantErr: true,
		},
		{
			name: "content too long",
			skill: &Skill{
				Name:        "Test",
				Description: "Description",
				Content:     string(make([]byte, 50001)),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.skill.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestService_Save tests the Save method.
func TestService_Save(t *testing.T) {
	tests := []struct {
		name      string
		skill     *Skill
		mockError error
		wantErr   bool
	}{
		{
			name:      "valid skill",
			skill:     newTestSkill(),
			mockError: nil,
			wantErr:   false,
		},
		{
			name:    "nil skill",
			skill:   nil,
			wantErr: true,
		},
		{
			name: "invalid skill",
			skill: &Skill{
				Name: "Test",
				// Missing description and content
			},
			wantErr: true,
		},
		{
			name:      "vector store error",
			skill:     newTestSkill(),
			mockError: errors.New("vector store error"),
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockVectorStore{
				addDocumentsFunc: func(ctx context.Context, docs []vectorstore.Document) error {
					return tt.mockError
				},
			}

			svc := NewService(mock)
			ctx := context.Background()

			id, err := svc.Save(ctx, tt.skill)
			if (err != nil) != tt.wantErr {
				t.Errorf("Save() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && id == "" {
				t.Error("Save() returned empty ID for valid skill")
			}
		})
	}
}

// TestService_Search tests the Search method.
func TestService_Search(t *testing.T) {
	tests := []struct {
		name        string
		query       string
		limit       int
		mockResults []vectorstore.SearchResult
		mockError   error
		wantCount   int
		wantErr     bool
	}{
		{
			name:  "successful search",
			query: "docker deployment",
			limit: 5,
			mockResults: []vectorstore.SearchResult{
				{
					ID:      "skill_1",
					Content: "Docker deployment skill",
					Score:   0.95,
					Metadata: map[string]interface{}{
						"id":          "skill_1",
						"name":        "Docker Deploy",
						"description": "Deploy with Docker",
						"content":     "Step 1...",
						"created_at":  time.Now().Format(time.RFC3339),
						"updated_at":  time.Now().Format(time.RFC3339),
					},
				},
			},
			wantCount: 1,
			wantErr:   false,
		},
		{
			name:      "empty query",
			query:     "",
			limit:     5,
			wantCount: 0,
			wantErr:   true,
		},
		{
			name:      "zero limit",
			query:     "test",
			limit:     0,
			wantCount: 0,
			wantErr:   true,
		},
		{
			name:      "negative limit",
			query:     "test",
			limit:     -1,
			wantCount: 0,
			wantErr:   true,
		},
		{
			name:      "vector store error",
			query:     "test",
			limit:     5,
			mockError: errors.New("search error"),
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockVectorStore{
				searchWithFiltersFunc: func(ctx context.Context, query string, k int, filters map[string]interface{}) ([]vectorstore.SearchResult, error) {
					if tt.mockError != nil {
						return nil, tt.mockError
					}
					return tt.mockResults, nil
				},
			}

			svc := NewService(mock)
			ctx := context.Background()

			results, err := svc.Search(ctx, tt.query, tt.limit)
			if (err != nil) != tt.wantErr {
				t.Errorf("Search() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && len(results) != tt.wantCount {
				t.Errorf("Search() returned %d results, want %d", len(results), tt.wantCount)
			}
		})
	}
}

// TestService_Create tests the Create method (alias for Save).
func TestService_Create(t *testing.T) {
	mock := &mockVectorStore{
		addDocumentsFunc: func(ctx context.Context, docs []vectorstore.Document) error {
			return nil
		},
	}

	svc := NewService(mock)
	ctx := context.Background()

	skill := newTestSkill()
	id, err := svc.Create(ctx, skill)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if id == "" {
		t.Error("Create() returned empty ID")
	}

	// Verify ID was set
	if skill.ID == "" {
		t.Error("Create() did not set skill ID")
	}
}

// TestService_Get tests the Get method.
func TestService_Get(t *testing.T) {
	tests := []struct {
		name        string
		id          string
		mockResults []vectorstore.SearchResult
		mockError   error
		wantErr     bool
	}{
		{
			name: "skill found",
			id:   "skill_1",
			mockResults: []vectorstore.SearchResult{
				{
					ID:      "skill_1",
					Content: "Docker skill",
					Score:   1.0,
					Metadata: map[string]interface{}{
						"id":          "skill_1",
						"name":        "Docker Deploy",
						"description": "Deploy with Docker",
						"content":     "Step 1...",
						"created_at":  time.Now().Format(time.RFC3339),
						"updated_at":  time.Now().Format(time.RFC3339),
					},
				},
			},
			wantErr: false,
		},
		{
			name:        "skill not found",
			id:          "nonexistent",
			mockResults: []vectorstore.SearchResult{},
			wantErr:     true,
		},
		{
			name:    "empty id",
			id:      "",
			wantErr: true,
		},
		{
			name:      "vector store error",
			id:        "skill_1",
			mockError: errors.New("get error"),
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockVectorStore{
				searchWithFiltersFunc: func(ctx context.Context, query string, k int, filters map[string]interface{}) ([]vectorstore.SearchResult, error) {
					if tt.mockError != nil {
						return nil, tt.mockError
					}
					return tt.mockResults, nil
				},
			}

			svc := NewService(mock)
			ctx := context.Background()

			skill, err := svc.Get(ctx, tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("Get() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if skill == nil {
					t.Error("Get() returned nil skill")
					return
				}
				if skill.ID != tt.id {
					t.Errorf("Get() returned skill with ID %s, want %s", skill.ID, tt.id)
				}
			}
		})
	}
}

// TestService_timestampHandling tests timestamp management.
func TestService_timestampHandling(t *testing.T) {
	mock := &mockVectorStore{
		addDocumentsFunc: func(ctx context.Context, docs []vectorstore.Document) error {
			// Verify timestamps were set
			if len(docs) == 0 {
				t.Error("No documents to save")
				return nil
			}

			doc := docs[0]
			if _, ok := doc.Metadata["created_at"]; !ok {
				t.Error("created_at not set")
			}
			if _, ok := doc.Metadata["updated_at"]; !ok {
				t.Error("updated_at not set")
			}

			return nil
		},
	}

	svc := NewService(mock)
	ctx := context.Background()

	skill := newTestSkill()
	_, err := svc.Save(ctx, skill)
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Verify timestamps were set on skill
	if skill.CreatedAt.IsZero() {
		t.Error("CreatedAt not set")
	}
	if skill.UpdatedAt.IsZero() {
		t.Error("UpdatedAt not set")
	}
}

// TestService_idGeneration tests ID generation.
func TestService_idGeneration(t *testing.T) {
	mock := &mockVectorStore{
		addDocumentsFunc: func(ctx context.Context, docs []vectorstore.Document) error {
			return nil
		},
	}

	svc := NewService(mock)
	ctx := context.Background()

	skill := newTestSkill()
	id1, err := svc.Save(ctx, skill)
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	skill2 := newTestSkill()
	id2, err := svc.Save(ctx, skill2)
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	if id1 == id2 {
		t.Error("Save() generated duplicate IDs")
	}

	// Verify IDs have correct prefix
	if len(id1) < 6 || id1[:6] != "skill_" {
		t.Errorf("Save() generated invalid ID format: %s", id1)
	}
}

// TestService_tagsHandling tests tag storage and retrieval.
func TestService_tagsHandling(t *testing.T) {
	mock := &mockVectorStore{
		addDocumentsFunc: func(ctx context.Context, docs []vectorstore.Document) error {
			// Verify tags in metadata
			if len(docs) == 0 {
				return nil
			}

			tags, ok := docs[0].Metadata["tags"]
			if !ok {
				t.Error("Tags not stored in metadata")
				return nil
			}

			tagSlice, ok := tags.([]string)
			if !ok {
				t.Error("Tags not stored as string slice")
				return nil
			}

			if len(tagSlice) != 2 {
				t.Errorf("Expected 2 tags, got %d", len(tagSlice))
			}

			return nil
		},
	}

	svc := NewService(mock)
	ctx := context.Background()

	skill := newTestSkill()
	skill.Tags = []string{"docker", "deployment"}

	_, err := svc.Save(ctx, skill)
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}
}

// TestService_metadataHandling tests custom metadata.
func TestService_metadataHandling(t *testing.T) {
	mock := &mockVectorStore{
		addDocumentsFunc: func(ctx context.Context, docs []vectorstore.Document) error {
			// Verify custom metadata
			if len(docs) == 0 {
				return nil
			}

			author, ok := docs[0].Metadata["author"]
			if !ok {
				t.Error("Custom metadata not stored")
				return nil
			}

			if author != "test-author" {
				t.Errorf("Expected author 'test-author', got '%s'", author)
			}

			return nil
		},
	}

	svc := NewService(mock)
	ctx := context.Background()

	skill := newTestSkill()
	skill.Metadata = map[string]interface{}{
		"author": "test-author",
		"team":   "platform",
	}

	_, err := svc.Save(ctx, skill)
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}
}

// TestService_contextCancellation tests context handling.
func TestService_contextCancellation(t *testing.T) {
	mock := &mockVectorStore{
		addDocumentsFunc: func(ctx context.Context, docs []vectorstore.Document) error {
			// Simulate slow operation
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(100 * time.Millisecond):
				return nil
			}
		},
	}

	svc := NewService(mock)

	// Create canceled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	skill := newTestSkill()
	_, err := svc.Save(ctx, skill)
	if err == nil {
		t.Error("Save() should fail with canceled context")
	}
}
