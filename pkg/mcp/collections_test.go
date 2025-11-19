package mcp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fyrsmithlabs/contextd/pkg/vectorstore"
)

// mockVectorStore implements collection management methods for testing.
type mockVectorStore struct {
	collections map[string]*vectorstore.CollectionInfo
}

func newMockVectorStore() *mockVectorStore {
	return &mockVectorStore{
		collections: make(map[string]*vectorstore.CollectionInfo),
	}
}

func (m *mockVectorStore) CreateCollection(ctx context.Context, name string, vectorSize int) error {
	if _, exists := m.collections[name]; exists {
		return vectorstore.ErrCollectionExists
	}
	m.collections[name] = &vectorstore.CollectionInfo{
		Name:       name,
		VectorSize: vectorSize,
		PointCount: 0,
	}
	return nil
}

func (m *mockVectorStore) DeleteCollection(ctx context.Context, name string) error {
	if _, exists := m.collections[name]; !exists {
		return vectorstore.ErrCollectionNotFound
	}
	delete(m.collections, name)
	return nil
}

func (m *mockVectorStore) ListCollections(ctx context.Context) ([]string, error) {
	names := make([]string, 0, len(m.collections))
	for name := range m.collections {
		names = append(names, name)
	}
	return names, nil
}

func (m *mockVectorStore) GetCollectionInfo(ctx context.Context, name string) (*vectorstore.CollectionInfo, error) {
	info, exists := m.collections[name]
	if !exists {
		return nil, vectorstore.ErrCollectionNotFound
	}
	return info, nil
}

// TestMCPServer_CollectionCreate tests POST /mcp/collection/create.
func TestMCPServer_CollectionCreate(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    string
		ownerID        string
		expectedStatus int
		expectSuccess  bool
		expectedError  string
	}{
		{
			name: "valid collection creation",
			requestBody: `{
				"jsonrpc": "2.0",
				"id": "test-1",
				"params": {
					"collection_name": "owner_abc123/project_def456/main",
					"vector_size": 384
				}
			}`,
			ownerID:        "abc1230000000000000000000000000000000000000000000000000000000000",
			expectedStatus: http.StatusOK,
			expectSuccess:  true,
		},
		{
			name: "missing collection_name",
			requestBody: `{
				"jsonrpc": "2.0",
				"id": "test-2",
				"params": {
					"vector_size": 384
				}
			}`,
			ownerID:        "abc1230000000000000000000000000000000000000000000000000000000000",
			expectedStatus: http.StatusOK,
			expectSuccess:  false,
			expectedError:  "collection_name is required",
		},
		{
			name: "missing vector_size",
			requestBody: `{
				"jsonrpc": "2.0",
				"id": "test-3",
				"params": {
					"collection_name": "owner_abc/project_def/main"
				}
			}`,
			ownerID:        "abc1230000000000000000000000000000000000000000000000000000000000",
			expectedStatus: http.StatusOK,
			expectSuccess:  false,
			expectedError:  "vector_size is required",
		},
		{
			name: "invalid vector_size (zero)",
			requestBody: `{
				"jsonrpc": "2.0",
				"id": "test-4",
				"params": {
					"collection_name": "owner_abc/project_def/main",
					"vector_size": 0
				}
			}`,
			ownerID:        "abc1230000000000000000000000000000000000000000000000000000000000",
			expectedStatus: http.StatusOK,
			expectSuccess:  false,
			expectedError:  "vector_size is required", // Zero is treated as missing
		},
		{
			name: "invalid vector_size (negative)",
			requestBody: `{
				"jsonrpc": "2.0",
				"id": "test-5",
				"params": {
					"collection_name": "owner_abc/project_def/main",
					"vector_size": -100
				}
			}`,
			ownerID:        "abc1230000000000000000000000000000000000000000000000000000000000",
			expectedStatus: http.StatusOK,
			expectSuccess:  false,
			expectedError:  "vector_size must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			e := echo.New()
			natsServer := startTestNATSServer(t)
			nc, err := nats.Connect(natsServer.ClientURL())
			require.NoError(t, err)
			defer nc.Close()

			registry := NewOperationRegistry(nc)
			mockStore := newMockVectorStore()
			mcpServer := NewServer(e, registry, nc, nil, nil, nil, nil, nil, nil, nil)
			mcpServer.vectorStore = mockStore

			// Register routes (includes auth middleware)
			mcpServer.RegisterRoutes()

			// Execute
			req := httptest.NewRequest(http.MethodPost, "/mcp/collection/create", strings.NewReader(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			e.ServeHTTP(rec, req)

			// Verify
			assert.Equal(t, tt.expectedStatus, rec.Code)

			if tt.expectSuccess {
				assert.Contains(t, rec.Body.String(), "operation_id")
				assert.Contains(t, rec.Body.String(), "pending")
			} else {
				assert.Contains(t, rec.Body.String(), "error")
				if tt.expectedError != "" {
					assert.Contains(t, rec.Body.String(), tt.expectedError)
				}
			}
		})
	}
}

// TestMCPServer_CollectionDelete tests POST /mcp/collection/delete.
func TestMCPServer_CollectionDelete(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    string
		ownerID        string
		setupFunc      func(*mockVectorStore)
		expectedStatus int
		expectSuccess  bool
		expectedError  string
	}{
		{
			name: "valid collection deletion",
			requestBody: `{
				"jsonrpc": "2.0",
				"id": "test-1",
				"params": {
					"collection_name": "owner_abc123/project_def456/main"
				}
			}`,
			ownerID: "abc1230000000000000000000000000000000000000000000000000000000000",
			setupFunc: func(m *mockVectorStore) {
				m.collections["owner_abc123/project_def456/main"] = &vectorstore.CollectionInfo{
					Name:       "owner_abc123/project_def456/main",
					VectorSize: 384,
					PointCount: 100,
				}
			},
			expectedStatus: http.StatusOK,
			expectSuccess:  true,
		},
		{
			name: "missing collection_name",
			requestBody: `{
				"jsonrpc": "2.0",
				"id": "test-2",
				"params": {}
			}`,
			ownerID:        "abc1230000000000000000000000000000000000000000000000000000000000",
			expectedStatus: http.StatusOK,
			expectSuccess:  false,
			expectedError:  "collection_name is required",
		},
		{
			name: "collection not found - async operation",
			requestBody: `{
				"jsonrpc": "2.0",
				"id": "test-3",
				"params": {
					"collection_name": "owner_xyz/project_nonexistent/main"
				}
			}`,
			ownerID:        "abc1230000000000000000000000000000000000000000000000000000000000",
			expectedStatus: http.StatusOK,
			expectSuccess:  true, // Returns async operation, error happens in background
			expectedError:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			e := echo.New()
			natsServer := startTestNATSServer(t)
			nc, err := nats.Connect(natsServer.ClientURL())
			require.NoError(t, err)
			defer nc.Close()

			registry := NewOperationRegistry(nc)
			mockStore := newMockVectorStore()
			if tt.setupFunc != nil {
				tt.setupFunc(mockStore)
			}
			mcpServer := NewServer(e, registry, nc, nil, nil, nil, nil, nil, nil, nil)
			mcpServer.vectorStore = mockStore

			// Register routes (includes auth middleware)
			mcpServer.RegisterRoutes()

			// Execute
			req := httptest.NewRequest(http.MethodPost, "/mcp/collection/delete", strings.NewReader(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			e.ServeHTTP(rec, req)

			// Verify
			assert.Equal(t, tt.expectedStatus, rec.Code)

			if tt.expectSuccess {
				assert.Contains(t, rec.Body.String(), "operation_id")
				assert.Contains(t, rec.Body.String(), "pending")
			} else {
				assert.Contains(t, rec.Body.String(), "error")
				if tt.expectedError != "" {
					assert.Contains(t, rec.Body.String(), tt.expectedError)
				}
			}
		})
	}
}

// TestMCPServer_CollectionList tests POST /mcp/collection/list.
func TestMCPServer_CollectionList(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    string
		ownerID        string
		setupFunc      func(*mockVectorStore)
		expectedStatus int
		expectSuccess  bool
		expectedError  string
		expectedCount  int
	}{
		{
			name: "list collections for authenticated owner",
			requestBody: `{
				"jsonrpc": "2.0",
				"id": "test-1",
				"params": {}
			}`,
			ownerID: "abc1230000000000000000000000000000000000000000000000000000000000",
			setupFunc: func(m *mockVectorStore) {
				// Add collections for this owner
				m.collections["owner_abc123/project_1/main"] = &vectorstore.CollectionInfo{Name: "owner_abc123/project_1/main", VectorSize: 384}
				m.collections["owner_abc123/project_2/main"] = &vectorstore.CollectionInfo{Name: "owner_abc123/project_2/main", VectorSize: 384}
				// Add collections for other owners (should be filtered out)
				m.collections["owner_xyz/project_3/main"] = &vectorstore.CollectionInfo{Name: "owner_xyz/project_3/main", VectorSize: 384}
			},
			expectedStatus: http.StatusOK,
			expectSuccess:  true,
			expectedCount:  2, // Only 2 collections match owner prefix
		},
		{
			name: "list collections with no matches",
			requestBody: `{
				"jsonrpc": "2.0",
				"id": "test-2",
				"params": {}
			}`,
			ownerID: "def4560000000000000000000000000000000000000000000000000000000000",
			setupFunc: func(m *mockVectorStore) {
				m.collections["owner_abc/project_1/main"] = &vectorstore.CollectionInfo{Name: "owner_abc/project_1/main", VectorSize: 384}
			},
			expectedStatus: http.StatusOK,
			expectSuccess:  true,
			expectedCount:  0, // No collections match
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			e := echo.New()
			natsServer := startTestNATSServer(t)
			nc, err := nats.Connect(natsServer.ClientURL())
			require.NoError(t, err)
			defer nc.Close()

			registry := NewOperationRegistry(nc)
			mockStore := newMockVectorStore()
			if tt.setupFunc != nil {
				tt.setupFunc(mockStore)
			}
			mcpServer := NewServer(e, registry, nc, nil, nil, nil, nil, nil, nil, nil)
			mcpServer.vectorStore = mockStore

			// Register routes (includes auth middleware)
			mcpServer.RegisterRoutes()

			// Execute
			req := httptest.NewRequest(http.MethodPost, "/mcp/collection/list", strings.NewReader(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			e.ServeHTTP(rec, req)

			// Verify
			assert.Equal(t, tt.expectedStatus, rec.Code)

			if tt.expectSuccess {
				assert.Contains(t, rec.Body.String(), "collections")
				// Could add more specific validation of collection count here
			} else {
				assert.Contains(t, rec.Body.String(), "error")
				if tt.expectedError != "" {
					assert.Contains(t, rec.Body.String(), tt.expectedError)
				}
			}
		})
	}
}

// TestCollectionManagement_AsyncOperations tests that create/delete return async operation IDs.
func TestCollectionManagement_AsyncOperations(t *testing.T) {
	e := echo.New()
	natsServer := startTestNATSServer(t)
	nc, err := nats.Connect(natsServer.ClientURL())
	require.NoError(t, err)
	defer nc.Close()

	registry := NewOperationRegistry(nc)
	mockStore := newMockVectorStore()
	mcpServer := NewServer(e, registry, nc, nil, nil, nil, nil, nil, nil, nil)
	mcpServer.vectorStore = mockStore

	// Register routes (includes auth middleware)
	mcpServer.RegisterRoutes()

	// Test create returns operation_id
	createReq := `{
		"jsonrpc": "2.0",
		"id": "async-1",
		"params": {
			"collection_name": "owner_abc/project_def/main",
			"vector_size": 384
		}
	}`

	req := httptest.NewRequest(http.MethodPost, "/mcp/collection/create", strings.NewReader(createReq))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "operation_id")
	assert.Contains(t, rec.Body.String(), "pending")

	// Test delete returns operation_id
	deleteReq := `{
		"jsonrpc": "2.0",
		"id": "async-2",
		"params": {
			"collection_name": "owner_abc/project_def/main"
		}
	}`

	req = httptest.NewRequest(http.MethodPost, "/mcp/collection/delete", strings.NewReader(deleteReq))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "operation_id")
	assert.Contains(t, rec.Body.String(), "pending")
}
