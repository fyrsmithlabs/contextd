package checkpoint

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/fyrsmithlabs/contextd/internal/vectorstore"
)

func TestDefaultServiceConfig(t *testing.T) {
	cfg := DefaultServiceConfig()

	assert.Equal(t, uint64(1536), cfg.VectorSize)
	assert.Equal(t, 10, cfg.MaxCheckpointsPerSession)
	assert.Len(t, cfg.AutoCheckpointThresholds, 4)
}

func TestNewService_RequiresStoreProvider(t *testing.T) {
	_, err := NewService(nil, nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "store provider is required")
}

func TestNewServiceWithStore_RequiresStore(t *testing.T) {
	_, err := NewServiceWithStore(nil, nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "vector store is required")
}

func TestNewServiceWithStore_RequiresLogger(t *testing.T) {
	store := newMockStore()
	_, err := NewServiceWithStore(nil, store, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "logger is required for checkpoint service")
}

func TestCheckpointToPayload(t *testing.T) {
	now := time.Now()
	cp := &Checkpoint{
		ID:          "cp_123",
		SessionID:   "sess_456",
		TenantID:    "tenant_1",
		TeamID:      "team_1",
		ProjectID:   "proj_1",
		ProjectPath: "/test/project",
		Name:        "Test Checkpoint",
		Description: "A test checkpoint",
		Summary:     "Summary of work",
		Context:     "Relevant context",
		FullState:   "Full state content",
		TokenCount:  1000,
		Threshold:   0.5,
		AutoCreated: true,
		Metadata:    map[string]string{"key": "value"},
		CreatedAt:   now,
	}

	payload := checkpointToPayload(cp)

	assert.Equal(t, "sess_456", payload["session_id"])
	assert.Equal(t, "tenant_1", payload["tenant_id"])
	assert.Equal(t, "team_1", payload["team_id"])
	assert.Equal(t, "proj_1", payload["project_id"])
	assert.Equal(t, "/test/project", payload["project_path"])
	assert.Equal(t, "Test Checkpoint", payload["name"])
	assert.Equal(t, "Summary of work", payload["summary"])
	assert.Equal(t, int64(1000), payload["token_count"])
	assert.Equal(t, 0.5, payload["threshold"])
	assert.Equal(t, true, payload["auto_created"])
	assert.Equal(t, "value", payload["meta_key"])
}

func TestPayloadToCheckpoint(t *testing.T) {
	now := time.Now()
	payload := map[string]interface{}{
		"session_id":   "sess_456",
		"tenant_id":    "tenant_1",
		"team_id":      "team_1",
		"project_id":   "proj_1",
		"project_path": "/test/project",
		"name":         "Test Checkpoint",
		"description":  "A test checkpoint",
		"summary":      "Summary of work",
		"context":      "Relevant context",
		"full_state":   "Full state content",
		"token_count":  int64(1000),
		"threshold":    0.5,
		"auto_created": true,
		"created_at":   now.Unix(),
		"meta_key":     "value",
	}

	cp := payloadToCheckpoint(payload)

	require.NotNil(t, cp)
	assert.Equal(t, "sess_456", cp.SessionID)
	assert.Equal(t, "tenant_1", cp.TenantID)
	assert.Equal(t, "team_1", cp.TeamID)
	assert.Equal(t, "proj_1", cp.ProjectID)
	assert.Equal(t, "/test/project", cp.ProjectPath)
	assert.Equal(t, "Test Checkpoint", cp.Name)
	assert.Equal(t, "Summary of work", cp.Summary)
	assert.Equal(t, int32(1000), cp.TokenCount)
	assert.Equal(t, 0.5, cp.Threshold)
	assert.True(t, cp.AutoCreated)
	assert.Equal(t, "value", cp.Metadata["key"])
}

func TestPayloadToCheckpoint_Nil(t *testing.T) {
	cp := payloadToCheckpoint(nil)
	assert.Nil(t, cp)
}

func TestEstimateTokens(t *testing.T) {
	// 100 chars should be ~25 tokens
	text := "hello world hello world hello world hello world hello world hello world hello world hello world hello"
	tokens := estimateTokens(text)
	assert.Equal(t, int32(25), tokens)
}

func TestResumeLevel(t *testing.T) {
	assert.Equal(t, ResumeLevel("summary"), ResumeSummary)
	assert.Equal(t, ResumeLevel("context"), ResumeContext)
	assert.Equal(t, ResumeLevel("full"), ResumeFull)
}

func TestCheckpoint(t *testing.T) {
	now := time.Now()
	cp := &Checkpoint{
		ID:          "cp_123",
		SessionID:   "sess_456",
		TenantID:    "tenant_1",
		Name:        "Test",
		Summary:     "Summary",
		TokenCount:  500,
		AutoCreated: false,
		CreatedAt:   now,
	}

	assert.Equal(t, "cp_123", cp.ID)
	assert.Equal(t, "sess_456", cp.SessionID)
	assert.Equal(t, int32(500), cp.TokenCount)
	assert.False(t, cp.AutoCreated)
}

func TestSaveRequest(t *testing.T) {
	req := &SaveRequest{
		SessionID:   "sess_123",
		TenantID:    "tenant_1",
		TeamID:      "team_1",
		ProjectID:   "proj_1",
		ProjectPath: "/test",
		Name:        "Manual checkpoint",
		Summary:     "Summary text",
		TokenCount:  1000,
		AutoCreated: false,
	}

	assert.Equal(t, "sess_123", req.SessionID)
	assert.Equal(t, "Manual checkpoint", req.Name)
	assert.False(t, req.AutoCreated)
}

func TestListRequest(t *testing.T) {
	req := &ListRequest{
		SessionID: "sess_123",
		TenantID:  "tenant_1",
		Limit:     10,
		AutoOnly:  true,
	}

	assert.Equal(t, "sess_123", req.SessionID)
	assert.True(t, req.AutoOnly)
}

func TestResumeRequest(t *testing.T) {
	req := &ResumeRequest{
		CheckpointID: "cp_123",
		TenantID:     "tenant_1",
		Level:        ResumeContext,
	}

	assert.Equal(t, "cp_123", req.CheckpointID)
	assert.Equal(t, ResumeContext, req.Level)
}

func TestResumeResponse(t *testing.T) {
	resp := &ResumeResponse{
		Checkpoint: &Checkpoint{ID: "cp_123"},
		Content:    "Restored content",
		TokenCount: 50,
	}

	assert.Equal(t, "cp_123", resp.Checkpoint.ID)
	assert.Equal(t, "Restored content", resp.Content)
	assert.Equal(t, int32(50), resp.TokenCount)
}

// Mock Store for testing

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
			Score:    1.0,
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

func TestService_SaveAndGet(t *testing.T) {
	store := newMockStore()
	logger := zap.NewNop()
	svc, err := NewServiceWithStore(nil, store, logger)
	require.NoError(t, err)
	defer svc.Close()

	ctx := context.Background()

	// Save a checkpoint
	saveReq := &SaveRequest{
		SessionID:   "sess_123",
		TenantID:    "tenant_1",
		TeamID:      "team_1",
		ProjectID:   "proj_1",
		ProjectPath: "/test",
		Name:        "Test Checkpoint",
		Summary:     "Summary of work done",
		Context:     "Relevant context",
		FullState:   "Full session state",
		TokenCount:  1000,
		AutoCreated: false,
	}

	cp, err := svc.Save(ctx, saveReq)
	require.NoError(t, err)
	assert.NotEmpty(t, cp.ID)
	assert.Equal(t, "sess_123", cp.SessionID)
	assert.Equal(t, "Test Checkpoint", cp.Name)
}

func TestService_List(t *testing.T) {
	store := newMockStore()
	logger := zap.NewNop()
	svc, err := NewServiceWithStore(nil, store, logger)
	require.NoError(t, err)
	defer svc.Close()

	ctx := context.Background()

	// List should return empty initially
	listReq := &ListRequest{
		TenantID:  "tenant_1",
		TeamID:    "team_1",
		ProjectID: "proj_1",
		Limit:     10,
	}

	checkpoints, err := svc.List(ctx, listReq)
	require.NoError(t, err)
	assert.Empty(t, checkpoints)
}

func TestService_Close(t *testing.T) {
	store := newMockStore()
	logger := zap.NewNop()
	svc, err := NewServiceWithStore(nil, store, logger)
	require.NoError(t, err)

	// Close should work
	err = svc.Close()
	assert.NoError(t, err)

	// Operations should fail after close
	_, err = svc.Save(context.Background(), &SaveRequest{TenantID: "t1", TeamID: "tm1", ProjectID: "p1"})
	assert.Error(t, err)
}

// TestService_ListFiltersProjectPath verifies that List() correctly filters by project_path,
// preventing checkpoints from one project leaking into queries for another project.
// This is a regression test for a critical data isolation bug.
func TestService_ListFiltersProjectPath(t *testing.T) {
	store := newMockStore()
	logger := zap.NewNop()
	svc, err := NewServiceWithStore(nil, store, logger)
	require.NoError(t, err)
	defer svc.Close()

	ctx := context.Background()

	// Save checkpoints for project A
	_, err = svc.Save(ctx, &SaveRequest{
		SessionID:   "sess_1",
		TenantID:    "tenant_1",
		TeamID:      "team_1",
		ProjectID:   "proj_a",
		ProjectPath: "/home/user/project-a",
		Name:        "Project A Checkpoint 1",
		Summary:     "Work on project A",
	})
	require.NoError(t, err)

	_, err = svc.Save(ctx, &SaveRequest{
		SessionID:   "sess_1",
		TenantID:    "tenant_1",
		TeamID:      "team_1",
		ProjectID:   "proj_a",
		ProjectPath: "/home/user/project-a",
		Name:        "Project A Checkpoint 2",
		Summary:     "More work on project A",
	})
	require.NoError(t, err)

	// Save checkpoints for project B
	_, err = svc.Save(ctx, &SaveRequest{
		SessionID:   "sess_2",
		TenantID:    "tenant_1",
		TeamID:      "team_1",
		ProjectID:   "proj_b",
		ProjectPath: "/home/user/project-b",
		Name:        "Project B Checkpoint",
		Summary:     "Work on project B",
	})
	require.NoError(t, err)

	// List checkpoints for project A only - should NOT include project B's checkpoint
	listReqA := &ListRequest{
		TenantID:    "tenant_1",
		TeamID:      "team_1",
		ProjectID:   "proj_a",
		ProjectPath: "/home/user/project-a",
		Limit:       10,
	}

	checkpointsA, err := svc.List(ctx, listReqA)
	require.NoError(t, err)
	assert.Len(t, checkpointsA, 2, "Should only return 2 checkpoints for project A")

	for _, cp := range checkpointsA {
		assert.Equal(t, "/home/user/project-a", cp.ProjectPath, "All checkpoints should be from project A")
	}

	// List checkpoints for project B only - should NOT include project A's checkpoints
	listReqB := &ListRequest{
		TenantID:    "tenant_1",
		TeamID:      "team_1",
		ProjectID:   "proj_b",
		ProjectPath: "/home/user/project-b",
		Limit:       10,
	}

	checkpointsB, err := svc.List(ctx, listReqB)
	require.NoError(t, err)
	assert.Len(t, checkpointsB, 1, "Should only return 1 checkpoint for project B")
	assert.Equal(t, "/home/user/project-b", checkpointsB[0].ProjectPath)
}
