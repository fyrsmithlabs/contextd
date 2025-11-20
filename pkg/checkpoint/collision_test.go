package checkpoint

import (
	"context"
	"fmt"
	"testing"

	"go.uber.org/zap"

	"github.com/fyrsmithlabs/contextd/pkg/vectorstore"
)

// TestCollisionDetection tests the collision detection algorithm
// when two different projects hash to the same 8-char prefix
func TestCollisionDetection(t *testing.T) {
	projectPath1 := "/home/user/project1"
	projectPath2 := "/home/user/project2"
	projectPath3 := "/home/user/project3"

	// Compute the base hash for project1 (this is what would normally be used)
	hash1 := projectHash(projectPath1)
	baseCollection := "project_" + hash1 + "__checkpoints"

	tests := []struct {
		name                string
		setupMock           func(*mockCollisionDetector)
		newProjectPath      string
		expectedCollection  string
		wantErr             bool
	}{
		{
			name: "no collision - collection doesn't exist",
			setupMock: func(m *mockCollisionDetector) {
				// Empty - no existing collections
			},
			newProjectPath:     projectPath1,
			expectedCollection: baseCollection,
			wantErr:            false,
		},
		{
			name: "no collision - same project reuses collection",
			setupMock: func(m *mockCollisionDetector) {
				// Collection exists for same project
				m.collections[baseCollection] = projectPath1
			},
			newProjectPath:     projectPath1,
			expectedCollection: baseCollection,
			wantErr:            false,
		},
		{
			name: "collision - different project appends _01",
			setupMock: func(m *mockCollisionDetector) {
				// Simulate collision: base collection exists for different project
				m.collections[baseCollection] = projectPath2
			},
			newProjectPath:     projectPath1,
			expectedCollection: "project_" + hash1 + "_01__checkpoints",
			wantErr:            false,
		},
		{
			name: "multiple collisions - appends _02",
			setupMock: func(m *mockCollisionDetector) {
				// Simulate multiple collisions
				m.collections[baseCollection] = projectPath2
				m.collections["project_"+hash1+"_01__checkpoints"] = projectPath3
			},
			newProjectPath:     projectPath1,
			expectedCollection: "project_" + hash1 + "_02__checkpoints",
			wantErr:            false,
		},
		{
			name: "collision limit exceeded",
			setupMock: func(m *mockCollisionDetector) {
				// Create 100 collisions (limit)
				m.collections[baseCollection] = projectPath2
				for i := 1; i < 100; i++ {
					collName := fmt.Sprintf("project_%s_%02d__checkpoints", hash1, i)
					m.collections[collName] = fmt.Sprintf("/home/user/project%d", i+10)
				}
			},
			newProjectPath: projectPath1,
			wantErr:        true, // Should fail after 100 attempts
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock with empty collections
			mock := &mockCollisionDetector{
				collections: make(map[string]string),
				createdCollections: make(map[string]string),
			}

			// Setup mock state
			tt.setupMock(mock)

			service := &Service{
				vectorStore: mock,
				logger:      zap.NewNop(),
			}

			// Test collision detection
			collectionName, err := service.getOrCreateCollectionName(context.Background(), tt.newProjectPath, "checkpoints")

			if tt.wantErr {
				if err == nil {
					t.Errorf("getOrCreateCollectionName() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("getOrCreateCollectionName() unexpected error: %v", err)
				return
			}

			if collectionName != tt.expectedCollection {
				t.Errorf("getOrCreateCollectionName() = %q, want %q", collectionName, tt.expectedCollection)
			}
		})
	}
}

// TestGetOrCreateCollectionName_RealHash tests with actual project path hashing
func TestGetOrCreateCollectionName_RealHash(t *testing.T) {
	mock := &mockCollisionDetector{
		collections: make(map[string]string),
		createdCollections: make(map[string]string),
	}

	service := &Service{
		vectorStore: mock,
		logger:      zap.NewNop(),
	}

	projectPath := "/home/user/myproject"
	expectedHash := projectHash(projectPath)

	// First call should return collection name (collection doesn't exist yet)
	collectionName1, err := service.getOrCreateCollectionName(context.Background(), projectPath, "checkpoints")
	if err != nil {
		t.Fatalf("First call failed: %v", err)
	}

	expectedName := "project_" + expectedHash + "__checkpoints"
	if collectionName1 != expectedName {
		t.Errorf("First call: got %q, want %q", collectionName1, expectedName)
	}

	// Simulate collection being created (in real usage, Save() would create it)
	mock.collections[collectionName1] = projectPath

	// Second call with same project should return same collection name
	collectionName2, err := service.getOrCreateCollectionName(context.Background(), projectPath, "checkpoints")
	if err != nil {
		t.Fatalf("Second call failed: %v", err)
	}

	if collectionName2 != collectionName1 {
		t.Errorf("Second call should return same collection: got %q, want %q", collectionName2, collectionName1)
	}
}

// mockCollisionDetector simulates Qdrant collection operations for testing
type mockCollisionDetector struct {
	collections        map[string]string // collection name -> project_path
	createdCollections map[string]string // track newly created collections
}

func (m *mockCollisionDetector) GetCollectionInfo(ctx context.Context, collectionName string) (*vectorstore.CollectionInfo, error) {
	// Check if collection exists
	if _, exists := m.collections[collectionName]; exists {
		return &vectorstore.CollectionInfo{
			Name:       collectionName,
			VectorSize: 384,
			PointCount: 10,
			// In real implementation, project_path would be in collection metadata
			// For this mock, we track it separately
		}, nil
	}

	// Collection doesn't exist
	return nil, vectorstore.ErrCollectionNotFound
}

func (m *mockCollisionDetector) CreateCollection(ctx context.Context, collectionName, projectPath string) error {
	// Simulate creating collection with project_path metadata
	if m.collections == nil {
		m.collections = make(map[string]string)
	}
	if m.createdCollections == nil {
		m.createdCollections = make(map[string]string)
	}

	m.collections[collectionName] = projectPath
	m.createdCollections[collectionName] = projectPath
	return nil
}

func (m *mockCollisionDetector) GetCollectionMetadata(ctx context.Context, collectionName string) (map[string]interface{}, error) {
	if projectPath, exists := m.collections[collectionName]; exists {
		return map[string]interface{}{
			"project_path": projectPath,
		}, nil
	}
	return nil, vectorstore.ErrCollectionNotFound
}

// Implement remaining VectorStore interface methods (stubs for collision test)
func (m *mockCollisionDetector) AddDocuments(ctx context.Context, docs []vectorstore.Document) error {
	return nil
}

func (m *mockCollisionDetector) SearchWithFilters(ctx context.Context, query string, k int, filters map[string]interface{}) ([]vectorstore.SearchResult, error) {
	return nil, nil
}

func (m *mockCollisionDetector) SearchInCollection(ctx context.Context, collectionName string, query string, k int, filters map[string]interface{}) ([]vectorstore.SearchResult, error) {
	// If collection exists, return a sample document with project_path metadata
	if projectPath, exists := m.collections[collectionName]; exists {
		return []vectorstore.SearchResult{
			{
				ID:      "sample-doc-1",
				Content: "Sample checkpoint",
				Metadata: map[string]interface{}{
					"project_path": projectPath,
				},
				Score: 1.0,
			},
		}, nil
	}
	return nil, nil
}

func (m *mockCollisionDetector) ExactSearch(ctx context.Context, collectionName string, query string, k int) ([]vectorstore.SearchResult, error) {
	return nil, nil
}

// Helper function to pad numbers with leading zeros
func padNumber(n int) string {
	if n < 10 {
		return "0" + string(rune('0'+n))
	}
	return string(rune('0'+n/10)) + string(rune('0'+n%10))
}
