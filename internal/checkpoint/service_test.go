package checkpoint

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/fyrsmithlabs/contextd/internal/qdrant"
)

func TestDefaultServiceConfig(t *testing.T) {
	cfg := DefaultServiceConfig()

	assert.Equal(t, uint64(1536), cfg.VectorSize)
	assert.Equal(t, 10, cfg.MaxCheckpointsPerSession)
	assert.Len(t, cfg.AutoCheckpointThresholds, 4)
}

func TestNewService_RequiresQdrant(t *testing.T) {
	_, err := NewService(nil, nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "qdrant client is required")
}

func TestNewService_RequiresLogger(t *testing.T) {
	qc := newMockQdrantClient()
	_, err := NewService(nil, qc, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "logger is required for checkpoint service")
}

func TestCheckpointToPayload(t *testing.T) {
	now := time.Now()
	cp := &Checkpoint{
		ID:          "cp-123",
		SessionID:   "sess-456",
		TenantID:    "tenant-1",
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

	assert.Equal(t, "sess-456", payload["session_id"])
	assert.Equal(t, "tenant-1", payload["tenant_id"])
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
		"session_id":   "sess-456",
		"tenant_id":    "tenant-1",
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
	assert.Equal(t, "sess-456", cp.SessionID)
	assert.Equal(t, "tenant-1", cp.TenantID)
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
		ID:          "cp-123",
		SessionID:   "sess-456",
		TenantID:    "tenant-1",
		Name:        "Test",
		Summary:     "Summary",
		TokenCount:  500,
		AutoCreated: false,
		CreatedAt:   now,
	}

	assert.Equal(t, "cp-123", cp.ID)
	assert.Equal(t, "sess-456", cp.SessionID)
	assert.Equal(t, int32(500), cp.TokenCount)
	assert.False(t, cp.AutoCreated)
}

func TestSaveRequest(t *testing.T) {
	req := &SaveRequest{
		SessionID:   "sess-123",
		TenantID:    "tenant-1",
		ProjectPath: "/test",
		Name:        "Manual checkpoint",
		Summary:     "Summary text",
		TokenCount:  1000,
		AutoCreated: false,
	}

	assert.Equal(t, "sess-123", req.SessionID)
	assert.Equal(t, "Manual checkpoint", req.Name)
	assert.False(t, req.AutoCreated)
}

func TestListRequest(t *testing.T) {
	req := &ListRequest{
		SessionID: "sess-123",
		TenantID:  "tenant-1",
		Limit:     10,
		AutoOnly:  true,
	}

	assert.Equal(t, "sess-123", req.SessionID)
	assert.True(t, req.AutoOnly)
}

func TestResumeRequest(t *testing.T) {
	req := &ResumeRequest{
		CheckpointID: "cp-123",
		TenantID:     "tenant-1",
		Level:        ResumeContext,
	}

	assert.Equal(t, "cp-123", req.CheckpointID)
	assert.Equal(t, ResumeContext, req.Level)
}

func TestResumeResponse(t *testing.T) {
	resp := &ResumeResponse{
		Checkpoint: &Checkpoint{ID: "cp-123"},
		Content:    "Restored content",
		TokenCount: 50,
	}

	assert.Equal(t, "cp-123", resp.Checkpoint.ID)
	assert.Equal(t, "Restored content", resp.Content)
	assert.Equal(t, int32(50), resp.TokenCount)
}

// Mock Qdrant client for testing

type mockQdrantClient struct {
	collections map[string]bool
	points      map[string][]*qdrant.Point
}

func newMockQdrantClient() *mockQdrantClient {
	return &mockQdrantClient{
		collections: make(map[string]bool),
		points:      make(map[string][]*qdrant.Point),
	}
}

func (m *mockQdrantClient) CreateCollection(ctx context.Context, name string, vectorSize uint64) error {
	m.collections[name] = true
	return nil
}

func (m *mockQdrantClient) DeleteCollection(ctx context.Context, name string) error {
	delete(m.collections, name)
	return nil
}

func (m *mockQdrantClient) CollectionExists(ctx context.Context, name string) (bool, error) {
	return m.collections[name], nil
}

func (m *mockQdrantClient) ListCollections(ctx context.Context) ([]string, error) {
	var names []string
	for name := range m.collections {
		names = append(names, name)
	}
	return names, nil
}

func (m *mockQdrantClient) Upsert(ctx context.Context, collection string, points []*qdrant.Point) error {
	m.points[collection] = append(m.points[collection], points...)
	return nil
}

func (m *mockQdrantClient) Search(ctx context.Context, collection string, vector []float32, limit uint64, filter *qdrant.Filter) ([]*qdrant.ScoredPoint, error) {
	var results []*qdrant.ScoredPoint
	for _, p := range m.points[collection] {
		results = append(results, &qdrant.ScoredPoint{
			Point: *p,
			Score: 1.0,
		})
	}
	return results, nil
}

func (m *mockQdrantClient) Get(ctx context.Context, collection string, ids []string) ([]*qdrant.Point, error) {
	var results []*qdrant.Point
	for _, p := range m.points[collection] {
		for _, id := range ids {
			if p.ID == id {
				results = append(results, p)
			}
		}
	}
	return results, nil
}

func (m *mockQdrantClient) Delete(ctx context.Context, collection string, ids []string) error {
	return nil
}

func (m *mockQdrantClient) Health(ctx context.Context) error {
	return nil
}

func (m *mockQdrantClient) Close() error {
	return nil
}

func TestService_SaveAndGet(t *testing.T) {
	qc := newMockQdrantClient()
	logger := zap.NewNop()
	svc, err := NewService(nil, qc, logger)
	require.NoError(t, err)
	defer svc.Close()

	ctx := context.Background()

	// Save a checkpoint
	saveReq := &SaveRequest{
		SessionID:   "sess-123",
		TenantID:    "tenant-1",
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
	assert.Equal(t, "sess-123", cp.SessionID)
	assert.Equal(t, "Test Checkpoint", cp.Name)
}

func TestService_List(t *testing.T) {
	qc := newMockQdrantClient()
	logger := zap.NewNop()
	svc, err := NewService(nil, qc, logger)
	require.NoError(t, err)
	defer svc.Close()

	ctx := context.Background()

	// List should return empty initially
	listReq := &ListRequest{
		TenantID: "tenant-1",
		Limit:    10,
	}

	checkpoints, err := svc.List(ctx, listReq)
	require.NoError(t, err)
	assert.Empty(t, checkpoints)
}

func TestService_Close(t *testing.T) {
	qc := newMockQdrantClient()
	logger := zap.NewNop()
	svc, err := NewService(nil, qc, logger)
	require.NoError(t, err)

	// Close should work
	err = svc.Close()
	assert.NoError(t, err)

	// Operations should fail after close
	_, err = svc.Save(context.Background(), &SaveRequest{TenantID: "t1"})
	assert.Error(t, err)
}
