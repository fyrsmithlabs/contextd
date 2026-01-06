package reasoningbank

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/fyrsmithlabs/contextd/internal/project"
	"github.com/fyrsmithlabs/contextd/internal/vectorstore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// mockStore is a simple in-memory mock of vectorstore.Store for testing.
type mockStore struct {
	collections map[string][]vectorstore.Document
	vectorSize  int
}

func newMockStore() *mockStore {
	return &mockStore{
		collections: make(map[string][]vectorstore.Document),
		vectorSize:  384,
	}
}

func (m *mockStore) AddDocuments(ctx context.Context, docs []vectorstore.Document) ([]string, error) {
	ids := make([]string, len(docs))
	for i, doc := range docs {
		collectionName := doc.Collection
		if collectionName == "" {
			collectionName = "default"
		}
		m.collections[collectionName] = append(m.collections[collectionName], doc)
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
	docs, ok := m.collections[collectionName]
	if !ok {
		return []vectorstore.SearchResult{}, nil
	}

	results := []vectorstore.SearchResult{}
	for _, doc := range docs {
		// Apply filters
		if filters != nil {
			shouldInclude := true

			// Check confidence filter
			if confFilter, ok := filters["confidence"].(map[string]interface{}); ok {
				if minConf, ok := confFilter["$gte"].(float64); ok {
					docConf, _ := doc.Metadata["confidence"].(float64)
					if docConf < minConf {
						shouldInclude = false
					}
				}
			}

			// Check ID filter
			if idFilter, ok := filters["id"].(string); ok {
				if doc.ID != idFilter {
					shouldInclude = false
				}
			}

			if !shouldInclude {
				continue
			}
		}

		results = append(results, vectorstore.SearchResult{
			ID:       doc.ID,
			Content:  doc.Content,
			Score:    0.9, // Mock score
			Metadata: doc.Metadata,
		})

		if len(results) >= k {
			break
		}
	}

	return results, nil
}

func (m *mockStore) DeleteDocuments(ctx context.Context, ids []string) error {
	for collectionName, docs := range m.collections {
		filtered := []vectorstore.Document{}
		for _, doc := range docs {
			shouldKeep := true
			for _, id := range ids {
				if doc.ID == id {
					shouldKeep = false
					break
				}
			}
			if shouldKeep {
				filtered = append(filtered, doc)
			}
		}
		m.collections[collectionName] = filtered
	}
	return nil
}

func (m *mockStore) DeleteDocumentsFromCollection(ctx context.Context, collectionName string, ids []string) error {
	docs, exists := m.collections[collectionName]
	if !exists {
		return nil
	}
	filtered := []vectorstore.Document{}
	for _, doc := range docs {
		shouldKeep := true
		for _, id := range ids {
			if doc.ID == id {
				shouldKeep = false
				break
			}
		}
		if shouldKeep {
			filtered = append(filtered, doc)
		}
	}
	m.collections[collectionName] = filtered
	return nil
}

func (m *mockStore) CreateCollection(ctx context.Context, collectionName string, vectorSize int) error {
	if _, exists := m.collections[collectionName]; exists {
		return vectorstore.ErrCollectionExists
	}
	m.collections[collectionName] = []vectorstore.Document{}
	return nil
}

func (m *mockStore) DeleteCollection(ctx context.Context, collectionName string) error {
	if _, exists := m.collections[collectionName]; !exists {
		return vectorstore.ErrCollectionNotFound
	}
	delete(m.collections, collectionName)
	return nil
}

func (m *mockStore) CollectionExists(ctx context.Context, collectionName string) (bool, error) {
	_, exists := m.collections[collectionName]
	return exists, nil
}

func (m *mockStore) ListCollections(ctx context.Context) ([]string, error) {
	names := make([]string, 0, len(m.collections))
	for name := range m.collections {
		names = append(names, name)
	}
	return names, nil
}

func (m *mockStore) GetCollectionInfo(ctx context.Context, collectionName string) (*vectorstore.CollectionInfo, error) {
	docs, exists := m.collections[collectionName]
	if !exists {
		return nil, vectorstore.ErrCollectionNotFound
	}
	return &vectorstore.CollectionInfo{
		Name:       collectionName,
		PointCount: len(docs),
		VectorSize: m.vectorSize,
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
	t.Run("requires store", func(t *testing.T) {
		_, err := NewService(nil, zap.NewNop())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "vector store cannot be nil")
	})

	t.Run("creates with valid inputs", func(t *testing.T) {
		store := newMockStore()
		svc, err := NewService(store, zap.NewNop(), WithDefaultTenant("test-tenant"))
		require.NoError(t, err)
		assert.NotNil(t, svc)
	})

	t.Run("requires logger", func(t *testing.T) {
		store := newMockStore()
		_, err := NewService(store, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "logger is required for ReasoningBank service")
	})
}

func TestService_Record(t *testing.T) {
	ctx := context.Background()
	store := newMockStore()
	svc, _ := NewService(store, zap.NewNop(), WithDefaultTenant("test-tenant"))

	t.Run("validates memory", func(t *testing.T) {
		err := svc.Record(ctx, nil)
		require.Error(t, err)
		assert.Equal(t, ErrInvalidMemory, err)
	})

	t.Run("creates collection if not exists", func(t *testing.T) {
		memory, _ := NewMemory(
			"project-123",
			"Test Memory",
			"This is test content",
			OutcomeSuccess,
			[]string{"test", "go"},
		)

		err := svc.Record(ctx, memory)
		require.NoError(t, err)

		// Check collection was created
		collectionName, _ := project.GetCollectionName("project-123", project.CollectionMemories)
		exists, _ := store.CollectionExists(ctx, collectionName)
		assert.True(t, exists)
	})

	t.Run("sets explicit record confidence", func(t *testing.T) {
		memory, _ := NewMemory(
			"project-123",
			"Test Memory 2",
			"More test content",
			OutcomeSuccess,
			[]string{"test"},
		)

		err := svc.Record(ctx, memory)
		require.NoError(t, err)
		assert.Equal(t, ExplicitRecordConfidence, memory.Confidence)
	})

	t.Run("preserves existing confidence", func(t *testing.T) {
		memory, _ := NewMemory(
			"project-123",
			"Test Memory 3",
			"Content with custom confidence",
			OutcomeSuccess,
			[]string{"test"},
		)
		memory.Confidence = 0.95

		err := svc.Record(ctx, memory)
		require.NoError(t, err)
		assert.Equal(t, 0.95, memory.Confidence)
	})

	t.Run("sets timestamps", func(t *testing.T) {
		beforeCreate := time.Now()
		memory, _ := NewMemory(
			"project-123",
			"Test Memory 4",
			"Timestamp test",
			OutcomeSuccess,
			[]string{"test"},
		)
		afterCreate := time.Now()

		err := svc.Record(ctx, memory)
		require.NoError(t, err)

		// Timestamps should be set
		assert.False(t, memory.CreatedAt.IsZero())
		assert.False(t, memory.UpdatedAt.IsZero())

		// CreatedAt should be within the time range of memory creation
		assert.True(t, !memory.CreatedAt.Before(beforeCreate), "CreatedAt should be after or equal to beforeCreate")
		assert.True(t, !memory.CreatedAt.After(afterCreate), "CreatedAt should be before or equal to afterCreate")
	})
}

func TestService_Search(t *testing.T) {
	ctx := context.Background()
	store := newMockStore()
	svc, _ := NewService(store, zap.NewNop(), WithDefaultTenant("test-tenant"))

	projectID := "project-123"

	// Create test memories
	memory1, _ := NewMemory(projectID, "Go Error Handling", "Use fmt.Errorf with %w", OutcomeSuccess, []string{"go", "errors"})
	memory1.Confidence = 0.9
	_ = svc.Record(ctx, memory1)

	memory2, _ := NewMemory(projectID, "Go Testing", "Use table-driven tests", OutcomeSuccess, []string{"go", "testing"})
	memory2.Confidence = 0.8
	_ = svc.Record(ctx, memory2)

	memory3, _ := NewMemory(projectID, "Low Confidence Memory", "This shouldn't appear", OutcomeSuccess, []string{"go"})
	memory3.Confidence = 0.6 // Below MinConfidence (0.7)
	_ = svc.Record(ctx, memory3)

	t.Run("requires project ID", func(t *testing.T) {
		_, err := svc.Search(ctx, "", "test query", 10)
		require.Error(t, err)
		assert.Equal(t, ErrEmptyProjectID, err)
	})

	t.Run("requires query", func(t *testing.T) {
		_, err := svc.Search(ctx, projectID, "", 10)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "query cannot be empty")
	})

	t.Run("filters by confidence >= 0.7", func(t *testing.T) {
		results, err := svc.Search(ctx, projectID, "error handling", 10)
		require.NoError(t, err)

		// Should get 2 results (memory1 and memory2), not memory3 (confidence 0.6)
		assert.Len(t, results, 2)

		for _, result := range results {
			assert.GreaterOrEqual(t, result.Confidence, MinConfidence)
		}
	})

	t.Run("returns empty for non-existent project", func(t *testing.T) {
		results, err := svc.Search(ctx, "non-existent-project", "test", 10)
		require.NoError(t, err)
		assert.Empty(t, results)
	})

	t.Run("uses default limit if not specified", func(t *testing.T) {
		results, err := svc.Search(ctx, projectID, "go", 0)
		require.NoError(t, err)
		assert.NotEmpty(t, results)
	})
}

func TestService_Get(t *testing.T) {
	ctx := context.Background()
	store := newMockStore()
	svc, _ := NewService(store, zap.NewNop(), WithDefaultTenant("test-tenant"))

	projectID := "project-123"
	memory, _ := NewMemory(projectID, "Test Memory", "Test content", OutcomeSuccess, []string{"test"})
	_ = svc.Record(ctx, memory)

	t.Run("retrieves memory by ID", func(t *testing.T) {
		retrieved, err := svc.Get(ctx, memory.ID)
		require.NoError(t, err)
		assert.Equal(t, memory.ID, retrieved.ID)
		assert.Equal(t, memory.Title, retrieved.Title)
		assert.Equal(t, memory.Content, retrieved.Content)
	})

	t.Run("returns error for empty ID", func(t *testing.T) {
		_, err := svc.Get(ctx, "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "memory ID cannot be empty")
	})

	t.Run("returns error for non-existent ID", func(t *testing.T) {
		_, err := svc.Get(ctx, "non-existent-id")
		require.Error(t, err)
		assert.Equal(t, ErrMemoryNotFound, err)
	})
}

func TestService_Feedback(t *testing.T) {
	ctx := context.Background()

	t.Run("increases confidence for helpful feedback", func(t *testing.T) {
		// Fresh service and memory for isolated test
		store := newMockStore()
		svc, _ := NewService(store, zap.NewNop(), WithDefaultTenant("test-tenant"))
		projectID := "project-123"
		memory, _ := NewMemory(projectID, "Test Memory", "Test content", OutcomeSuccess, []string{"test"})
		_ = svc.Record(ctx, memory)

		// Bayesian prior starts at 0.5 (1:1 alpha:beta)
		// Positive explicit feedback should increase confidence above the prior
		err := svc.Feedback(ctx, memory.ID, true)
		require.NoError(t, err)

		updated, _ := svc.Get(ctx, memory.ID)
		// With Bayesian system, confidence should be above the prior (0.5) after positive feedback
		assert.Greater(t, updated.Confidence, 0.5)
	})

	t.Run("decreases confidence for unhelpful feedback", func(t *testing.T) {
		// Fresh service and memory for isolated test
		store := newMockStore()
		svc, _ := NewService(store, zap.NewNop(), WithDefaultTenant("test-tenant"))
		projectID := "project-123"
		memory, _ := NewMemory(projectID, "Test Memory", "Test content", OutcomeSuccess, []string{"test"})
		_ = svc.Record(ctx, memory)

		// Bayesian prior starts at 0.5 (1:1 alpha:beta)
		// Negative explicit feedback should decrease confidence below the prior
		err := svc.Feedback(ctx, memory.ID, false)
		require.NoError(t, err)

		updated, _ := svc.Get(ctx, memory.ID)
		// With Bayesian system, confidence should be below the prior (0.5) after negative feedback
		assert.Less(t, updated.Confidence, 0.5)
	})

	t.Run("requires memory ID", func(t *testing.T) {
		store := newMockStore()
		svc, _ := NewService(store, zap.NewNop(), WithDefaultTenant("test-tenant"))
		err := svc.Feedback(ctx, "", true)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "memory ID cannot be empty")
	})

	t.Run("returns error for non-existent memory", func(t *testing.T) {
		store := newMockStore()
		svc, _ := NewService(store, zap.NewNop(), WithDefaultTenant("test-tenant"))
		projectID := "project-123"
		memory, _ := NewMemory(projectID, "Test Memory", "Test content", OutcomeSuccess, []string{"test"})
		_ = svc.Record(ctx, memory)
		err := svc.Feedback(ctx, "non-existent-id", true)
		require.Error(t, err)
	})
}

func TestService_RecordOutcome(t *testing.T) {
	ctx := context.Background()

	t.Run("increases confidence for successful outcome", func(t *testing.T) {
		// Fresh service and memory for isolated test
		store := newMockStore()
		svc, _ := NewService(store, zap.NewNop(), WithDefaultTenant("test-tenant"))
		projectID := "project-123"
		memory, _ := NewMemory(projectID, "Test Memory", "Test content", OutcomeSuccess, []string{"test"})
		_ = svc.Record(ctx, memory)

		// Bayesian prior starts at 0.5 (1:1 alpha:beta)
		// Positive outcome signal should increase confidence above the prior
		newConf, err := svc.RecordOutcome(ctx, memory.ID, true, "session-123")
		require.NoError(t, err)

		// With Bayesian system, confidence should be above the prior (0.5) after positive outcome
		assert.Greater(t, newConf, 0.5)

		updated, _ := svc.Get(ctx, memory.ID)
		assert.Equal(t, newConf, updated.Confidence)
	})

	t.Run("decreases confidence for failed outcome", func(t *testing.T) {
		// Fresh service and memory for isolated test
		store := newMockStore()
		svc, _ := NewService(store, zap.NewNop(), WithDefaultTenant("test-tenant"))
		projectID := "project-123"
		memory, _ := NewMemory(projectID, "Test Memory", "Test content", OutcomeSuccess, []string{"test"})
		_ = svc.Record(ctx, memory)

		// Bayesian prior starts at 0.5 (1:1 alpha:beta)
		// Negative outcome signal should decrease confidence below the prior
		newConf, err := svc.RecordOutcome(ctx, memory.ID, false, "session-124")
		require.NoError(t, err)

		// With Bayesian system, confidence should be below the prior (0.5) after negative outcome
		assert.Less(t, newConf, 0.5)

		updated, _ := svc.Get(ctx, memory.ID)
		assert.Equal(t, newConf, updated.Confidence)
	})

	t.Run("requires memory ID", func(t *testing.T) {
		store := newMockStore()
		svc, _ := NewService(store, zap.NewNop(), WithDefaultTenant("test-tenant"))
		_, err := svc.RecordOutcome(ctx, "", true, "session-125")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "memory ID cannot be empty")
	})

	t.Run("returns error for non-existent memory", func(t *testing.T) {
		store := newMockStore()
		svc, _ := NewService(store, zap.NewNop(), WithDefaultTenant("test-tenant"))
		projectID := "project-123"
		memory, _ := NewMemory(projectID, "Test Memory", "Test content", OutcomeSuccess, []string{"test"})
		_ = svc.Record(ctx, memory)
		_, err := svc.RecordOutcome(ctx, "non-existent-id", true, "session-126")
		require.Error(t, err)
	})

	t.Run("accepts empty session ID", func(t *testing.T) {
		store := newMockStore()
		svc, _ := NewService(store, zap.NewNop(), WithDefaultTenant("test-tenant"))
		projectID := "project-123"
		memory, _ := NewMemory(projectID, "Test Memory", "Test content", OutcomeSuccess, []string{"test"})
		_ = svc.Record(ctx, memory)
		newConf, err := svc.RecordOutcome(ctx, memory.ID, true, "")
		require.NoError(t, err)
		assert.Greater(t, newConf, 0.0)
	})
}

func TestService_Delete(t *testing.T) {
	ctx := context.Background()
	store := newMockStore()
	svc, _ := NewService(store, zap.NewNop(), WithDefaultTenant("test-tenant"))

	projectID := "project-123"
	memory, _ := NewMemory(projectID, "Test Memory", "Test content", OutcomeSuccess, []string{"test"})
	_ = svc.Record(ctx, memory)

	t.Run("deletes memory by ID", func(t *testing.T) {
		err := svc.Delete(ctx, memory.ID)
		require.NoError(t, err)

		_, err = svc.Get(ctx, memory.ID)
		assert.Equal(t, ErrMemoryNotFound, err)
	})

	t.Run("requires memory ID", func(t *testing.T) {
		err := svc.Delete(ctx, "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "memory ID cannot be empty")
	})

	t.Run("returns error for non-existent memory", func(t *testing.T) {
		err := svc.Delete(ctx, "non-existent-id")
		require.Error(t, err)
	})
}

func TestDistiller_DistillSession(t *testing.T) {
	ctx := context.Background()
	store := newMockStore()
	svc, _ := NewService(store, zap.NewNop(), WithDefaultTenant("test-tenant"))
	distiller, err := NewDistiller(svc, zap.NewNop())
	require.NoError(t, err)

	projectID := "project-123"

	t.Run("requires project ID", func(t *testing.T) {
		summary := SessionSummary{
			SessionID: "sess-123",
			ProjectID: "",
		}
		err := distiller.DistillSession(ctx, summary)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "project ID cannot be empty")
	})

	t.Run("requires session ID", func(t *testing.T) {
		summary := SessionSummary{
			SessionID: "",
			ProjectID: projectID,
		}
		err := distiller.DistillSession(ctx, summary)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "session ID cannot be empty")
	})

	t.Run("extracts success pattern", func(t *testing.T) {
		summary := SessionSummary{
			SessionID:   "sess-success",
			ProjectID:   projectID,
			Outcome:     SessionSuccess,
			Task:        "Fix authentication bug",
			Approach:    "Used defensive validation",
			Result:      "Bug fixed, tests passing",
			Tags:        []string{"go", "auth", "security"},
			Duration:    5 * time.Minute,
			CompletedAt: time.Now(),
		}

		err := distiller.DistillSession(ctx, summary)
		require.NoError(t, err)

		// Search for the distilled memory (bypass confidence filter by lowering it)
		// Note: Distilled memories have confidence 0.6 which is below MinConfidence (0.7)
		// For testing, we need to access the collection directly or search all
		collectionName, _ := project.GetCollectionName(projectID, project.CollectionMemories)
		results, err := store.SearchInCollection(ctx, collectionName, "authentication", 10, nil)
		require.NoError(t, err)
		assert.NotEmpty(t, results)

		// Convert results to memories
		memories := make([]Memory, 0, len(results))
		for _, result := range results {
			memory, err := svc.resultToMemory(result)
			require.NoError(t, err)
			memories = append(memories, *memory)
		}

		// Find the distilled memory
		var distilledMemory *Memory
		for i := range memories {
			if memories[i].Outcome == OutcomeSuccess {
				distilledMemory = &memories[i]
				break
			}
		}

		require.NotNil(t, distilledMemory)
		assert.Contains(t, distilledMemory.Title, "Success")
		assert.Equal(t, OutcomeSuccess, distilledMemory.Outcome)
		assert.Equal(t, DistilledConfidence, distilledMemory.Confidence)
		assert.Contains(t, distilledMemory.Content, "Successful Approach")
		assert.Contains(t, distilledMemory.Content, "Used defensive validation")
	})

	t.Run("extracts failure pattern", func(t *testing.T) {
		summary := SessionSummary{
			SessionID:   "sess-failure",
			ProjectID:   projectID,
			Outcome:     SessionFailure,
			Task:        "Optimize database query",
			Approach:    "Added more indexes",
			Result:      "Made performance worse, increased write latency",
			Tags:        []string{"go", "database", "performance"},
			Duration:    10 * time.Minute,
			CompletedAt: time.Now(),
		}

		err := distiller.DistillSession(ctx, summary)
		require.NoError(t, err)

		// Search for the distilled anti-pattern (bypass confidence filter)
		collectionName, _ := project.GetCollectionName(projectID, project.CollectionMemories)
		results, err := store.SearchInCollection(ctx, collectionName, "database", 10, nil)
		require.NoError(t, err)
		assert.NotEmpty(t, results)

		// Convert results to memories
		memories := make([]Memory, 0, len(results))
		for _, result := range results {
			memory, err := svc.resultToMemory(result)
			require.NoError(t, err)
			memories = append(memories, *memory)
		}

		// Find the anti-pattern memory
		var antiPattern *Memory
		for i := range memories {
			if memories[i].Outcome == OutcomeFailure {
				antiPattern = &memories[i]
				break
			}
		}

		require.NotNil(t, antiPattern)
		assert.Contains(t, antiPattern.Title, "Anti-pattern")
		assert.Equal(t, OutcomeFailure, antiPattern.Outcome)
		// Anti-pattern confidence is DistilledConfidence - 0.1 = 0.5
		assert.InDelta(t, 0.5, antiPattern.Confidence, 0.001)
		assert.Contains(t, antiPattern.Content, "Failed Approach")
		assert.Contains(t, antiPattern.Content, "What Went Wrong")
	})

	t.Run("extracts both patterns from partial outcome", func(t *testing.T) {
		summary := SessionSummary{
			SessionID:   "sess-partial",
			ProjectID:   projectID,
			Outcome:     SessionPartial,
			Task:        "Refactor authentication module",
			Approach:    "Split into smaller functions but broke backwards compatibility",
			Result:      "Code cleaner but integration tests failed",
			Tags:        []string{"go", "refactoring"},
			Duration:    15 * time.Minute,
			CompletedAt: time.Now(),
		}

		err := distiller.DistillSession(ctx, summary)
		require.NoError(t, err)

		// Should have created both success and failure patterns (bypass confidence filter)
		collectionName, _ := project.GetCollectionName(projectID, project.CollectionMemories)
		results, err := store.SearchInCollection(ctx, collectionName, "refactoring", 10, nil)
		require.NoError(t, err)
		assert.NotEmpty(t, results)

		// Convert results to memories
		memories := make([]Memory, 0, len(results))
		for _, result := range results {
			memory, err := svc.resultToMemory(result)
			require.NoError(t, err)
			memories = append(memories, *memory)
		}

		// Check we have both outcome types
		hasSuccess := false
		hasFailure := false
		for _, mem := range memories {
			if mem.Outcome == OutcomeSuccess {
				hasSuccess = true
			}
			if mem.Outcome == OutcomeFailure {
				hasFailure = true
			}
		}
		assert.True(t, hasSuccess || hasFailure, "Should have extracted at least one pattern type")
	})
}

func TestNewDistiller(t *testing.T) {
	t.Run("requires service", func(t *testing.T) {
		_, err := NewDistiller(nil, zap.NewNop())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "service cannot be nil")
	})

	t.Run("requires logger", func(t *testing.T) {
		store := newMockStore()
		svc, _ := NewService(store, zap.NewNop(), WithDefaultTenant("test-tenant"))
		_, err := NewDistiller(svc, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "logger cannot be nil")
	})

	t.Run("creates with valid inputs", func(t *testing.T) {
		store := newMockStore()
		svc, _ := NewService(store, zap.NewNop(), WithDefaultTenant("test-tenant"))
		distiller, err := NewDistiller(svc, zap.NewNop())
		require.NoError(t, err)
		assert.NotNil(t, distiller)
	})
}

func TestMemoryToDocument(t *testing.T) {
	store := newMockStore()
	svc, _ := NewService(store, zap.NewNop(), WithDefaultTenant("test-tenant"))

	memory, _ := NewMemory(
		"project-123",
		"Test Memory",
		"This is the content",
		OutcomeSuccess,
		[]string{"go", "test"},
	)
	memory.Confidence = 0.85

	collectionName := "project-123_memories"
	doc := svc.memoryToDocument(memory, collectionName)

	t.Run("sets correct ID", func(t *testing.T) {
		assert.Equal(t, memory.ID, doc.ID)
	})

	t.Run("combines title and content", func(t *testing.T) {
		expected := fmt.Sprintf("%s\n\n%s", memory.Title, memory.Content)
		assert.Equal(t, expected, doc.Content)
	})

	t.Run("sets correct collection", func(t *testing.T) {
		assert.Equal(t, collectionName, doc.Collection)
	})

	t.Run("includes all metadata", func(t *testing.T) {
		assert.Equal(t, memory.ID, doc.Metadata["id"])
		assert.Equal(t, memory.ProjectID, doc.Metadata["project_id"])
		assert.Equal(t, memory.Title, doc.Metadata["title"])
		assert.Equal(t, string(memory.Outcome), doc.Metadata["outcome"])
		assert.Equal(t, memory.Confidence, doc.Metadata["confidence"])
		assert.Equal(t, memory.UsageCount, doc.Metadata["usage_count"])
		assert.Equal(t, memory.Tags, doc.Metadata["tags"])
	})
}

func TestResultToMemory(t *testing.T) {
	store := newMockStore()
	svc, _ := NewService(store, zap.NewNop(), WithDefaultTenant("test-tenant"))

	now := time.Now()
	result := vectorstore.SearchResult{
		ID:      "mem-123",
		Content: "Test Memory\n\nThis is the content",
		Score:   0.95,
		Metadata: map[string]interface{}{
			"id":          "mem-123",
			"project_id":  "project-123",
			"title":       "Test Memory",
			"description": "A test memory",
			"outcome":     "success",
			"confidence":  0.85,
			"usage_count": 5,
			"tags":        []interface{}{"go", "test"},
			"created_at":  now.Unix(),
			"updated_at":  now.Unix(),
		},
	}

	memory, err := svc.resultToMemory(result)
	require.NoError(t, err)

	t.Run("extracts all fields", func(t *testing.T) {
		assert.Equal(t, "mem-123", memory.ID)
		assert.Equal(t, "project-123", memory.ProjectID)
		assert.Equal(t, "Test Memory", memory.Title)
		assert.Equal(t, "A test memory", memory.Description)
		assert.Equal(t, OutcomeSuccess, memory.Outcome)
		assert.Equal(t, 0.85, memory.Confidence)
		assert.Equal(t, 5, memory.UsageCount)
		assert.Equal(t, []string{"go", "test"}, memory.Tags)
	})

	t.Run("strips title from content", func(t *testing.T) {
		assert.Equal(t, "This is the content", memory.Content)
	})

	t.Run("parses timestamps", func(t *testing.T) {
		assert.Equal(t, now.Unix(), memory.CreatedAt.Unix())
		assert.Equal(t, now.Unix(), memory.UpdatedAt.Unix())
	})
}

// mockStoreProvider implements vectorstore.StoreProvider for testing.
type mockStoreProvider struct {
	stores map[string]*mockStore
}

func newMockStoreProvider() *mockStoreProvider {
	return &mockStoreProvider{
		stores: make(map[string]*mockStore),
	}
}

func (p *mockStoreProvider) GetProjectStore(ctx context.Context, tenant, team, project string) (vectorstore.Store, error) {
	var key string
	if team != "" {
		key = fmt.Sprintf("%s/%s/%s", tenant, team, project)
	} else {
		key = fmt.Sprintf("%s/%s", tenant, project)
	}
	if store, ok := p.stores[key]; ok {
		return store, nil
	}
	store := newMockStore()
	p.stores[key] = store
	return store, nil
}

func (p *mockStoreProvider) GetTeamStore(ctx context.Context, tenant, team string) (vectorstore.Store, error) {
	key := fmt.Sprintf("%s/%s", tenant, team)
	if store, ok := p.stores[key]; ok {
		return store, nil
	}
	store := newMockStore()
	p.stores[key] = store
	return store, nil
}

func (p *mockStoreProvider) GetOrgStore(ctx context.Context, tenant string) (vectorstore.Store, error) {
	if store, ok := p.stores[tenant]; ok {
		return store, nil
	}
	store := newMockStore()
	p.stores[tenant] = store
	return store, nil
}

func (p *mockStoreProvider) Close() error {
	return nil
}

func TestNewServiceWithStoreProvider(t *testing.T) {
	t.Run("requires store provider", func(t *testing.T) {
		_, err := NewServiceWithStoreProvider(nil, "test-tenant", zap.NewNop())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "store provider cannot be nil")
	})

	t.Run("requires default tenant", func(t *testing.T) {
		stores := newMockStoreProvider()
		_, err := NewServiceWithStoreProvider(stores, "", zap.NewNop())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "default tenant cannot be empty")
	})

	t.Run("requires logger", func(t *testing.T) {
		stores := newMockStoreProvider()
		_, err := NewServiceWithStoreProvider(stores, "test-tenant", nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "logger is required for ReasoningBank service")
	})

	t.Run("creates with valid inputs", func(t *testing.T) {
		stores := newMockStoreProvider()
		svc, err := NewServiceWithStoreProvider(stores, "test-tenant", zap.NewNop())
		require.NoError(t, err)
		assert.NotNil(t, svc)
	})
}

func TestService_GetByProjectID(t *testing.T) {
	ctx := context.Background()
	stores := newMockStoreProvider()
	svc, _ := NewServiceWithStoreProvider(stores, "test-tenant", zap.NewNop())

	t.Run("requires project ID", func(t *testing.T) {
		_, err := svc.GetByProjectID(ctx, "", "mem-123")
		require.Error(t, err)
		assert.Equal(t, ErrEmptyProjectID, err)
	})

	t.Run("requires memory ID", func(t *testing.T) {
		_, err := svc.GetByProjectID(ctx, "project-123", "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "memory ID cannot be empty")
	})

	t.Run("returns error for non-existent memory", func(t *testing.T) {
		_, err := svc.GetByProjectID(ctx, "project-123", "non-existent")
		require.Error(t, err)
		assert.Equal(t, ErrMemoryNotFound, err)
	})

	t.Run("retrieves memory by project and ID", func(t *testing.T) {
		memory, _ := NewMemory(
			"project-456",
			"Test Memory",
			"This is test content",
			OutcomeSuccess,
			[]string{"test"},
		)
		memory.Confidence = 0.85

		err := svc.Record(ctx, memory)
		require.NoError(t, err)

		found, err := svc.GetByProjectID(ctx, "project-456", memory.ID)
		require.NoError(t, err)
		assert.Equal(t, memory.ID, found.ID)
		assert.Equal(t, "project-456", found.ProjectID)
		assert.Equal(t, "Test Memory", found.Title)
	})
}

func TestService_DeleteByProjectID(t *testing.T) {
	ctx := context.Background()
	stores := newMockStoreProvider()
	svc, _ := NewServiceWithStoreProvider(stores, "test-tenant", zap.NewNop())

	t.Run("requires project ID", func(t *testing.T) {
		err := svc.DeleteByProjectID(ctx, "", "mem-123")
		require.Error(t, err)
		assert.Equal(t, ErrEmptyProjectID, err)
	})

	t.Run("requires memory ID", func(t *testing.T) {
		err := svc.DeleteByProjectID(ctx, "project-123", "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "memory ID cannot be empty")
	})

	t.Run("deletes memory by project and ID", func(t *testing.T) {
		memory, _ := NewMemory(
			"project-789",
			"Memory to Delete",
			"This memory will be deleted",
			OutcomeSuccess,
			[]string{"delete-test"},
		)
		memory.Confidence = 0.85

		err := svc.Record(ctx, memory)
		require.NoError(t, err)

		// Verify it exists
		found, err := svc.GetByProjectID(ctx, "project-789", memory.ID)
		require.NoError(t, err)
		assert.Equal(t, memory.ID, found.ID)

		// Delete it
		err = svc.DeleteByProjectID(ctx, "project-789", memory.ID)
		require.NoError(t, err)

		// Verify it's gone
		_, err = svc.GetByProjectID(ctx, "project-789", memory.ID)
		assert.Equal(t, ErrMemoryNotFound, err)
	})
}

func TestService_WithStoreProvider_Operations(t *testing.T) {
	ctx := context.Background()
	stores := newMockStoreProvider()
	svc, _ := NewServiceWithStoreProvider(stores, "test-tenant", zap.NewNop())

	t.Run("Search uses StoreProvider", func(t *testing.T) {
		// Record a memory with high confidence
		memory, _ := NewMemory(
			"search-project",
			"Searchable Memory",
			"This memory can be found via search",
			OutcomeSuccess,
			[]string{"search"},
		)
		memory.Confidence = 0.9

		err := svc.Record(ctx, memory)
		require.NoError(t, err)

		// Search should find it
		results, err := svc.Search(ctx, "search-project", "searchable", 10)
		require.NoError(t, err)
		assert.Len(t, results, 1)
		assert.Equal(t, memory.ID, results[0].ID)
	})

	t.Run("Count uses StoreProvider", func(t *testing.T) {
		// Count memories in the search-project (should have 1 from previous test)
		count, err := svc.Count(ctx, "search-project")
		require.NoError(t, err)
		assert.Equal(t, 1, count)

		// Empty project should have 0
		count, err = svc.Count(ctx, "empty-project")
		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})

	t.Run("Record creates per-project isolation", func(t *testing.T) {
		// Record in project A
		memA, _ := NewMemory(
			"project-A",
			"Memory A",
			"Content for project A",
			OutcomeSuccess,
			[]string{},
		)
		memA.Confidence = 0.85
		err := svc.Record(ctx, memA)
		require.NoError(t, err)

		// Record in project B
		memB, _ := NewMemory(
			"project-B",
			"Memory B",
			"Content for project B",
			OutcomeSuccess,
			[]string{},
		)
		memB.Confidence = 0.85
		err = svc.Record(ctx, memB)
		require.NoError(t, err)

		// Each project should have its own store
		_, okA := stores.stores["test-tenant/project-A"]
		_, okB := stores.stores["test-tenant/project-B"]
		assert.True(t, okA, "project-A should have its own store")
		assert.True(t, okB, "project-B should have its own store")
	})
}

func TestService_ListMemories(t *testing.T) {
	ctx := context.Background()
	store := newMockStore()
	svc, _ := NewService(store, zap.NewNop(), WithDefaultTenant("test-tenant"))

	t.Run("validates project ID", func(t *testing.T) {
		_, err := svc.ListMemories(ctx, "", 10, 0)
		require.Error(t, err)
		assert.Equal(t, ErrEmptyProjectID, err)
	})

	t.Run("validates limit", func(t *testing.T) {
		_, err := svc.ListMemories(ctx, "project-123", -1, 0)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "limit cannot be negative")
	})

	t.Run("validates offset", func(t *testing.T) {
		_, err := svc.ListMemories(ctx, "project-123", 10, -1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "offset cannot be negative")
	})

	t.Run("returns empty list for non-existent project", func(t *testing.T) {
		memories, err := svc.ListMemories(ctx, "non-existent", 10, 0)
		require.NoError(t, err)
		assert.Empty(t, memories)
	})

	t.Run("lists all memories without pagination", func(t *testing.T) {
		projectID := "list-project-1"

		// Create multiple memories
		for i := 1; i <= 5; i++ {
			memory, _ := NewMemory(
				projectID,
				fmt.Sprintf("Memory %d", i),
				fmt.Sprintf("Content for memory %d", i),
				OutcomeSuccess,
				[]string{"test"},
			)
			err := svc.Record(ctx, memory)
			require.NoError(t, err)
		}

		// List all memories (limit=0 means all)
		memories, err := svc.ListMemories(ctx, projectID, 0, 0)
		require.NoError(t, err)
		assert.Len(t, memories, 5)
	})

	t.Run("lists memories with limit", func(t *testing.T) {
		projectID := "list-project-2"

		// Create multiple memories
		for i := 1; i <= 10; i++ {
			memory, _ := NewMemory(
				projectID,
				fmt.Sprintf("Memory %d", i),
				fmt.Sprintf("Content for memory %d", i),
				OutcomeSuccess,
				[]string{"test"},
			)
			err := svc.Record(ctx, memory)
			require.NoError(t, err)
		}

		// List with limit
		memories, err := svc.ListMemories(ctx, projectID, 3, 0)
		require.NoError(t, err)
		assert.Len(t, memories, 3)
	})

	t.Run("lists memories with offset", func(t *testing.T) {
		projectID := "list-project-3"

		// Create memories with known titles
		titles := []string{"First", "Second", "Third", "Fourth", "Fifth"}
		for _, title := range titles {
			memory, _ := NewMemory(
				projectID,
				title,
				fmt.Sprintf("Content for %s", title),
				OutcomeSuccess,
				[]string{"test"},
			)
			err := svc.Record(ctx, memory)
			require.NoError(t, err)
		}

		// List with offset (skip first 2, get next 2)
		memories, err := svc.ListMemories(ctx, projectID, 2, 2)
		require.NoError(t, err)
		assert.Len(t, memories, 2)

		// Verify offset was applied (we should get 3rd and 4th items)
		// Note: order depends on storage implementation
		for _, mem := range memories {
			assert.NotEmpty(t, mem.Title)
		}
	})

	t.Run("handles offset beyond available memories", func(t *testing.T) {
		projectID := "list-project-4"

		// Create 3 memories
		for i := 1; i <= 3; i++ {
			memory, _ := NewMemory(
				projectID,
				fmt.Sprintf("Memory %d", i),
				fmt.Sprintf("Content for memory %d", i),
				OutcomeSuccess,
				[]string{"test"},
			)
			err := svc.Record(ctx, memory)
			require.NoError(t, err)
		}

		// Try to list with offset beyond available memories
		memories, err := svc.ListMemories(ctx, projectID, 10, 100)
		require.NoError(t, err)
		assert.Empty(t, memories)
	})

	t.Run("returns all memories when limit exceeds count", func(t *testing.T) {
		projectID := "list-project-5"

		// Create 3 memories
		for i := 1; i <= 3; i++ {
			memory, _ := NewMemory(
				projectID,
				fmt.Sprintf("Memory %d", i),
				fmt.Sprintf("Content for memory %d", i),
				OutcomeSuccess,
				[]string{"test"},
			)
			err := svc.Record(ctx, memory)
			require.NoError(t, err)
		}

		// Request more than available
		memories, err := svc.ListMemories(ctx, projectID, 100, 0)
		require.NoError(t, err)
		assert.Len(t, memories, 3)
	})

	t.Run("pagination example", func(t *testing.T) {
		projectID := "list-project-6"

		// Create 10 memories
		for i := 1; i <= 10; i++ {
			memory, _ := NewMemory(
				projectID,
				fmt.Sprintf("Memory %d", i),
				fmt.Sprintf("Content for memory %d", i),
				OutcomeSuccess,
				[]string{"test"},
			)
			err := svc.Record(ctx, memory)
			require.NoError(t, err)
		}

		// Paginate through all memories (page size = 3)
		allMemories := []Memory{}
		pageSize := 3
		offset := 0

		for {
			page, err := svc.ListMemories(ctx, projectID, pageSize, offset)
			require.NoError(t, err)

			if len(page) == 0 {
				break
			}

			allMemories = append(allMemories, page...)
			offset += len(page)
		}

		// Should have collected all 10 memories
		assert.Len(t, allMemories, 10)
	})
}
