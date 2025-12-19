package remediation

import (
	"context"
	"testing"
	"time"

	"github.com/fyrsmithlabs/contextd/internal/vectorstore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// mockStore implements vectorstore.Store for testing.
type mockStore struct {
	collections map[string]bool
	documents   map[string][]vectorstore.Document
}

func newMockStore() *mockStore {
	return &mockStore{
		collections: make(map[string]bool),
		documents:   make(map[string][]vectorstore.Document),
	}
}

func (m *mockStore) AddDocuments(ctx context.Context, docs []vectorstore.Document) ([]string, error) {
	if len(docs) == 0 {
		return nil, nil
	}
	collection := docs[0].Collection
	if collection == "" {
		collection = "default"
	}
	m.documents[collection] = append(m.documents[collection], docs...)
	ids := make([]string, len(docs))
	for i, doc := range docs {
		ids[i] = doc.ID
	}
	return ids, nil
}

func (m *mockStore) Search(ctx context.Context, query string, k int) ([]vectorstore.SearchResult, error) {
	return m.SearchInCollection(ctx, "default", query, k, nil)
}

func (m *mockStore) SearchWithFilters(ctx context.Context, query string, k int, filters map[string]interface{}) ([]vectorstore.SearchResult, error) {
	return m.SearchInCollection(ctx, "default", query, k, filters)
}

func (m *mockStore) SearchInCollection(ctx context.Context, collectionName string, query string, k int, filters map[string]interface{}) ([]vectorstore.SearchResult, error) {
	var results []vectorstore.SearchResult
	for _, doc := range m.documents[collectionName] {
		// Apply filters if provided
		if filters != nil {
			match := true
			for key, val := range filters {
				if docVal, ok := doc.Metadata[key]; ok {
					if docVal != val {
						match = false
						break
					}
				}
			}
			if !match {
				continue
			}
		}
		results = append(results, vectorstore.SearchResult{
			ID:       doc.ID,
			Content:  doc.Content,
			Score:    0.9,
			Metadata: doc.Metadata,
		})
	}
	if len(results) > k {
		results = results[:k]
	}
	return results, nil
}

func (m *mockStore) DeleteDocuments(ctx context.Context, ids []string) error {
	return m.DeleteDocumentsFromCollection(ctx, "default", ids)
}

func (m *mockStore) DeleteDocumentsFromCollection(ctx context.Context, collectionName string, ids []string) error {
	idSet := make(map[string]bool)
	for _, id := range ids {
		idSet[id] = true
	}
	var remaining []vectorstore.Document
	for _, doc := range m.documents[collectionName] {
		if !idSet[doc.ID] {
			remaining = append(remaining, doc)
		}
	}
	m.documents[collectionName] = remaining
	return nil
}

func (m *mockStore) CreateCollection(ctx context.Context, collectionName string, vectorSize int) error {
	m.collections[collectionName] = true
	return nil
}

func (m *mockStore) DeleteCollection(ctx context.Context, collectionName string) error {
	delete(m.collections, collectionName)
	delete(m.documents, collectionName)
	return nil
}

func (m *mockStore) CollectionExists(ctx context.Context, collectionName string) (bool, error) {
	return m.collections[collectionName], nil
}

func (m *mockStore) ListCollections(ctx context.Context) ([]string, error) {
	var names []string
	for name := range m.collections {
		names = append(names, name)
	}
	return names, nil
}

func (m *mockStore) GetCollectionInfo(ctx context.Context, collectionName string) (*vectorstore.CollectionInfo, error) {
	if !m.collections[collectionName] {
		return nil, vectorstore.ErrCollectionNotFound
	}
	return &vectorstore.CollectionInfo{
		Name:       collectionName,
		PointCount: len(m.documents[collectionName]),
		VectorSize: 384,
	}, nil
}

func (m *mockStore) ExactSearch(ctx context.Context, collectionName string, query string, k int) ([]vectorstore.SearchResult, error) {
	return m.SearchInCollection(ctx, collectionName, query, k, nil)
}

func (m *mockStore) Close() error {
	return nil
}

func (m *mockStore) SetIsolationMode(mode vectorstore.IsolationMode) {
	// No-op for mock
}

func (m *mockStore) IsolationMode() vectorstore.IsolationMode {
	return vectorstore.NewNoIsolation()
}

func TestNewService(t *testing.T) {
	tests := []struct {
		name      string
		cfg       *Config
		store     vectorstore.Store
		logger    *zap.Logger
		wantErr   bool
		errSubstr string
	}{
		{
			name:    "success with all dependencies",
			cfg:     DefaultServiceConfig(),
			store:   newMockStore(),
			logger:  zap.NewNop(),
			wantErr: false,
		},
		{
			name:    "success with nil config uses defaults",
			cfg:     nil,
			store:   newMockStore(),
			logger:  zap.NewNop(),
			wantErr: false,
		},
		{
			name:      "fails without store",
			cfg:       DefaultServiceConfig(),
			store:     nil,
			logger:    zap.NewNop(),
			wantErr:   true,
			errSubstr: "vector store is required",
		},
		{
			name:      "fails without logger",
			cfg:       DefaultServiceConfig(),
			store:     newMockStore(),
			logger:    nil,
			wantErr:   true,
			errSubstr: "logger is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, err := NewService(tt.cfg, tt.store, tt.logger)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errSubstr != "" {
					assert.Contains(t, err.Error(), tt.errSubstr)
				}
				return
			}

			require.NoError(t, err)
			require.NotNil(t, svc)
		})
	}
}

func TestService_Record(t *testing.T) {
	ctx := context.Background()
	store := newMockStore()

	svc, err := NewService(DefaultServiceConfig(), store, zap.NewNop())
	require.NoError(t, err)

	tests := []struct {
		name    string
		req     *RecordRequest
		wantErr bool
	}{
		{
			name: "record compile error remediation",
			req: &RecordRequest{
				Title:         "Fix undefined variable error",
				Problem:       "Variable 'foo' is not defined",
				Symptoms:      []string{"undefined: foo", "cannot find symbol"},
				RootCause:     "Variable used before declaration",
				Solution:      "Declare the variable before use or import the correct package",
				CodeDiff:      "- use(foo)\n+ var foo = 42\n+ use(foo)",
				AffectedFiles: []string{"main.go"},
				Category:      ErrorCompile,
				Tags:          []string{"go", "variable"},
				Scope:         ScopeProject,
				TenantID:      "tenant1",
				ProjectPath:   "/home/project",
			},
			wantErr: false,
		},
		{
			name: "record runtime error remediation",
			req: &RecordRequest{
				Title:     "Fix nil pointer dereference",
				Problem:   "Panic: nil pointer dereference",
				RootCause: "Accessing field on nil struct pointer",
				Solution:  "Add nil check before accessing struct fields",
				Category:  ErrorRuntime,
				Scope:     ScopeTeam,
				TenantID:  "tenant1",
				TeamID:    "platform",
			},
			wantErr: false,
		},
		{
			name: "record test failure remediation",
			req: &RecordRequest{
				Title:     "Fix flaky test",
				Problem:   "Test intermittently fails",
				Symptoms:  []string{"FAIL: TestAsync", "race condition"},
				RootCause: "Test relies on timing assumptions",
				Solution:  "Use sync.WaitGroup instead of time.Sleep",
				Category:  ErrorTest,
				Scope:     ScopeOrg,
				TenantID:  "tenant1",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rem, err := svc.Record(ctx, tt.req)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, rem)
			assert.NotEmpty(t, rem.ID)
			assert.Equal(t, tt.req.Title, rem.Title)
			assert.Equal(t, tt.req.Problem, rem.Problem)
			assert.Equal(t, tt.req.RootCause, rem.RootCause)
			assert.Equal(t, tt.req.Solution, rem.Solution)
			assert.Equal(t, tt.req.Category, rem.Category)
			assert.Equal(t, tt.req.Scope, rem.Scope)
			assert.False(t, rem.CreatedAt.IsZero())
			assert.False(t, rem.UpdatedAt.IsZero())
		})
	}
}

func TestService_Get(t *testing.T) {
	ctx := context.Background()
	store := newMockStore()

	svc, err := NewService(DefaultServiceConfig(), store, zap.NewNop())
	require.NoError(t, err)

	// Record a remediation first
	recorded, err := svc.Record(ctx, &RecordRequest{
		Title:       "Test Remediation",
		Problem:     "Test problem",
		RootCause:   "Test root cause",
		Solution:    "Test solution",
		Category:    ErrorOther,
		Scope:       ScopeOrg,
		TenantID:    "tenant1",
		ProjectPath: "/home/project",
	})
	require.NoError(t, err)

	tests := []struct {
		name          string
		tenantID      string
		remediationID string
		wantErr       bool
	}{
		{
			name:          "get existing remediation",
			tenantID:      "tenant1",
			remediationID: recorded.ID,
			wantErr:       false,
		},
		{
			name:          "get nonexistent remediation",
			tenantID:      "tenant1",
			remediationID: "nonexistent",
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rem, err := svc.Get(ctx, tt.tenantID, tt.remediationID)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, rem)
			assert.Equal(t, recorded.ID, rem.ID)
			assert.Equal(t, recorded.Title, rem.Title)
		})
	}
}

func TestService_Search(t *testing.T) {
	ctx := context.Background()
	store := newMockStore()

	svc, err := NewService(DefaultServiceConfig(), store, zap.NewNop())
	require.NoError(t, err)

	// Record multiple remediations
	remediations := []struct {
		title    string
		problem  string
		category ErrorCategory
		scope    Scope
	}{
		{"Compile Error 1", "undefined variable", ErrorCompile, ScopeOrg},
		{"Compile Error 2", "type mismatch", ErrorCompile, ScopeOrg},
		{"Runtime Error 1", "nil pointer", ErrorRuntime, ScopeOrg},
	}

	for _, r := range remediations {
		_, err := svc.Record(ctx, &RecordRequest{
			Title:     r.title,
			Problem:   r.problem,
			RootCause: "Test root cause",
			Solution:  "Test solution",
			Category:  r.category,
			Scope:     r.scope,
			TenantID:  "tenant1",
		})
		require.NoError(t, err)
	}

	tests := []struct {
		name       string
		req        *SearchRequest
		minResults int
		wantErr    bool
	}{
		{
			name: "search by query",
			req: &SearchRequest{
				Query:    "undefined variable error",
				TenantID: "tenant1",
				Scope:    ScopeOrg,
				Limit:    10,
			},
			minResults: 1,
		},
		{
			name: "search with limit",
			req: &SearchRequest{
				Query:    "error",
				TenantID: "tenant1",
				Scope:    ScopeOrg,
				Limit:    2,
			},
			minResults: 2,
		},
		{
			name: "search empty collection",
			req: &SearchRequest{
				Query:       "error",
				TenantID:    "tenant2",
				Scope:       ScopeOrg,
				ProjectPath: "/nonexistent",
				Limit:       10,
			},
			minResults: 0,
		},
		{
			name: "search requires query",
			req: &SearchRequest{
				TenantID: "tenant1",
				Scope:    ScopeOrg,
				Limit:    10,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := svc.Search(ctx, tt.req)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.GreaterOrEqual(t, len(results), tt.minResults)

			// Verify results have scores
			for _, r := range results {
				assert.Greater(t, r.Score, 0.0)
			}
		})
	}
}

func TestService_Feedback(t *testing.T) {
	ctx := context.Background()
	store := newMockStore()
	cfg := DefaultServiceConfig()

	svc, err := NewService(cfg, store, zap.NewNop())
	require.NoError(t, err)

	// Record a remediation first
	recorded, err := svc.Record(ctx, &RecordRequest{
		Title:      "Test Remediation",
		Problem:    "Test problem",
		RootCause:  "Test root cause",
		Solution:   "Test solution",
		Category:   ErrorOther,
		Scope:      ScopeOrg,
		TenantID:   "tenant1",
		Confidence: 0.5,
	})
	require.NoError(t, err)

	initialConfidence := recorded.Confidence

	tests := []struct {
		name         string
		rating       FeedbackRating
		wantIncrease bool
		wantDecrease bool
	}{
		{
			name:         "helpful feedback increases confidence",
			rating:       RatingHelpful,
			wantIncrease: true,
		},
		{
			name:         "not helpful feedback decreases confidence",
			rating:       RatingNotHelpful,
			wantDecrease: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := svc.Feedback(ctx, &FeedbackRequest{
				RemediationID: recorded.ID,
				TenantID:      "tenant1",
				Rating:        tt.rating,
				SessionID:     "session1",
			})
			require.NoError(t, err)

			// Get the updated remediation
			updated, err := svc.Get(ctx, "tenant1", recorded.ID)
			require.NoError(t, err)

			if tt.wantIncrease {
				assert.Greater(t, updated.Confidence, initialConfidence)
			}
			if tt.wantDecrease {
				assert.Less(t, updated.Confidence, initialConfidence)
			}

			// Update baseline for next test
			initialConfidence = updated.Confidence
		})
	}
}

func TestService_Delete(t *testing.T) {
	ctx := context.Background()
	store := newMockStore()

	svc, err := NewService(DefaultServiceConfig(), store, zap.NewNop())
	require.NoError(t, err)

	// Record a remediation first
	recorded, err := svc.Record(ctx, &RecordRequest{
		Title:     "Remediation to Delete",
		Problem:   "Test problem",
		RootCause: "Test root cause",
		Solution:  "Test solution",
		Category:  ErrorOther,
		Scope:     ScopeOrg,
		TenantID:  "tenant1",
	})
	require.NoError(t, err)

	// Delete the remediation
	err = svc.Delete(ctx, "tenant1", recorded.ID)
	require.NoError(t, err)

	// Verify it's deleted
	_, err = svc.Get(ctx, "tenant1", recorded.ID)
	require.Error(t, err)
}

func TestService_Close(t *testing.T) {
	ctx := context.Background()
	store := newMockStore()

	svc, err := NewService(DefaultServiceConfig(), store, zap.NewNop())
	require.NoError(t, err)

	// Close should succeed
	err = svc.Close()
	require.NoError(t, err)

	// After close, operations should fail
	_, err = svc.Record(ctx, &RecordRequest{
		Title:    "Test",
		TenantID: "tenant1",
		Scope:    ScopeOrg,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "closed")
}

func TestDefaultServiceConfig(t *testing.T) {
	cfg := DefaultServiceConfig()
	require.NotNil(t, cfg)
	assert.Equal(t, "remediations", cfg.CollectionPrefix)
	assert.Equal(t, uint64(1536), cfg.VectorSize)
	assert.Equal(t, 0.5, cfg.DefaultConfidence)
	assert.Equal(t, 0.1, cfg.FeedbackDelta)
	assert.Equal(t, 0.1, cfg.MinConfidence)
	assert.Equal(t, 1.0, cfg.MaxConfidence)
}

func TestSanitizePath(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "alphanumeric path",
			input: "myproject123",
			want:  "myproject123",
		},
		{
			name:  "path with slashes",
			input: "/home/user/project",
			want:  "_home_user_project",
		},
		{
			name:  "path with special chars",
			input: "my-project.name",
			want:  "my_project_name",
		},
		{
			name:  "empty path",
			input: "",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizePath(tt.input)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestPayloadConversion(t *testing.T) {
	now := time.Now()

	original := &Remediation{
		ID:            "rem-123",
		Title:         "Test Remediation",
		Problem:       "Test problem",
		Symptoms:      []string{"symptom1", "symptom2"},
		RootCause:     "Test root cause",
		Solution:      "Test solution",
		CodeDiff:      "diff content",
		AffectedFiles: []string{"file1.go", "file2.go"},
		Category:      ErrorCompile,
		Confidence:    0.8,
		UsageCount:    5,
		Tags:          []string{"tag1", "tag2"},
		Scope:         ScopeProject,
		TenantID:      "tenant1",
		TeamID:        "team1",
		ProjectPath:   "/project",
		SessionID:     "session1",
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	// Convert to payload and back
	payload := remediationToPayload(original)
	require.NotNil(t, payload)

	converted := payloadToRemediation(payload)
	require.NotNil(t, converted)

	// Verify fields
	assert.Equal(t, original.Title, converted.Title)
	assert.Equal(t, original.Problem, converted.Problem)
	assert.Equal(t, original.RootCause, converted.RootCause)
	assert.Equal(t, original.Solution, converted.Solution)
	assert.Equal(t, original.CodeDiff, converted.CodeDiff)
	assert.Equal(t, original.Category, converted.Category)
	assert.Equal(t, original.Confidence, converted.Confidence)
	assert.Equal(t, original.Scope, converted.Scope)
	assert.Equal(t, original.TenantID, converted.TenantID)
	assert.Equal(t, original.TeamID, converted.TeamID)
	assert.Equal(t, original.ProjectPath, converted.ProjectPath)
	assert.Equal(t, original.SessionID, converted.SessionID)
	assert.Equal(t, original.Symptoms, converted.Symptoms)
	assert.Equal(t, original.AffectedFiles, converted.AffectedFiles)
	assert.Equal(t, original.Tags, converted.Tags)
}

func TestPayloadToRemediation_NilPayload(t *testing.T) {
	result := payloadToRemediation(nil)
	assert.Nil(t, result)
}

func TestSplitByDelimiter(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		delimiter string
		want      []string
	}{
		{
			name:      "split with delimiter",
			input:     "a||b||c",
			delimiter: "||",
			want:      []string{"a", "b", "c"},
		},
		{
			name:      "single element",
			input:     "single",
			delimiter: "||",
			want:      []string{"single"},
		},
		{
			name:      "empty string",
			input:     "",
			delimiter: "||",
			want:      []string{},
		},
		{
			name:      "delimiter at end",
			input:     "a||b||",
			delimiter: "||",
			want:      []string{"a", "b"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := splitByDelimiter(tt.input, tt.delimiter)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestSortAndLimit(t *testing.T) {
	remediations := []*ScoredRemediation{
		{Score: 0.5},
		{Score: 0.9},
		{Score: 0.7},
		{Score: 0.3},
		{Score: 0.8},
	}

	tests := []struct {
		name       string
		limit      int
		wantLen    int
		wantScores []float64
	}{
		{
			name:       "limit less than total",
			limit:      3,
			wantLen:    3,
			wantScores: []float64{0.9, 0.8, 0.7},
		},
		{
			name:       "limit greater than total",
			limit:      10,
			wantLen:    5,
			wantScores: []float64{0.9, 0.8, 0.7, 0.5, 0.3},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Make a copy to avoid mutating original
			input := make([]*ScoredRemediation, len(remediations))
			copy(input, remediations)

			result := sortAndLimit(input, tt.limit)
			assert.Len(t, result, tt.wantLen)

			for i, score := range tt.wantScores[:tt.wantLen] {
				assert.Equal(t, score, result[i].Score)
			}
		})
	}
}
