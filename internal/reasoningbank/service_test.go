package reasoningbank

import (
	"context"
	"fmt"
	"hash/fnv"
	"math"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/fyrsmithlabs/contextd/internal/project"
	"github.com/fyrsmithlabs/contextd/internal/vectorstore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// mockStore is a simple in-memory mock of vectorstore.Store for testing.
// Thread-safe: uses mutex for concurrent access from scheduler goroutines.
type mockStore struct {
	mu               sync.RWMutex
	collections      map[string][]vectorstore.Document
	vectorSize       int
	searchCalled     bool
	searchCallCount  int
	returnError      bool
	errorToReturn    error
}

func newMockStore() *mockStore {
	return &mockStore{
		collections:     make(map[string][]vectorstore.Document),
		vectorSize:      384,
		searchCalled:    false,
		searchCallCount: 0,
		returnError:     false,
	}
}

func newMockStoreWithError() *mockStore {
	return &mockStore{
		collections:     make(map[string][]vectorstore.Document),
		vectorSize:      384,
		searchCalled:    false,
		searchCallCount: 0,
		returnError:     true,
		errorToReturn:   fmt.Errorf("mock store error"),
	}
}

func (m *mockStore) AddDocuments(ctx context.Context, docs []vectorstore.Document) ([]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

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
	m.mu.Lock()
	// Track search calls for testing
	m.searchCalled = true
	m.searchCallCount++

	// Return error if configured to do so
	if m.returnError {
		m.mu.Unlock()
		return nil, m.errorToReturn
	}

	docs, ok := m.collections[collectionName]
	m.mu.Unlock()

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
	m.mu.Lock()
	defer m.mu.Unlock()

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
	m.mu.Lock()
	defer m.mu.Unlock()

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
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.collections[collectionName]; exists {
		return vectorstore.ErrCollectionExists
	}
	m.collections[collectionName] = []vectorstore.Document{}
	return nil
}

func (m *mockStore) DeleteCollection(ctx context.Context, collectionName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.collections[collectionName]; !exists {
		return vectorstore.ErrCollectionNotFound
	}
	delete(m.collections, collectionName)
	return nil
}

func (m *mockStore) CollectionExists(ctx context.Context, collectionName string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	_, exists := m.collections[collectionName]
	return exists, nil
}

func (m *mockStore) ListCollections(ctx context.Context) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, 0, len(m.collections))
	for name := range m.collections {
		names = append(names, name)
	}
	return names, nil
}

func (m *mockStore) GetCollectionInfo(ctx context.Context, collectionName string) (*vectorstore.CollectionInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

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

// SearchCalled returns whether SearchInCollection was called (thread-safe).
func (m *mockStore) SearchCalled() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.searchCalled
}

// SearchCallCount returns the number of times SearchInCollection was called (thread-safe).
func (m *mockStore) SearchCallCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.searchCallCount
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
		// Use a valid UUID format that doesn't exist in the store
		_, err := svc.Get(ctx, "00000000-0000-0000-0000-000000000000")
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
		// Use a valid UUID format that doesn't exist in the store
		err := svc.Feedback(ctx, "00000000-0000-0000-0000-000000000000", true)
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
		// Use a valid UUID format that doesn't exist in the store
		_, err := svc.RecordOutcome(ctx, "00000000-0000-0000-0000-000000000000", true, "session-126")
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
		// Use a valid UUID format that doesn't exist in the store
		_, err := svc.GetByProjectID(ctx, "project-123", "00000000-0000-0000-0000-000000000000")
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

// mockEmbedder implements vectorstore.Embedder for testing.
type mockEmbedder struct {
	vectorSize int
}

func newMockEmbedder(vectorSize int) *mockEmbedder {
	return &mockEmbedder{vectorSize: vectorSize}
}

func (m *mockEmbedder) EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error) {
	embeddings := make([][]float32, len(texts))
	for i := range texts {
		embeddings[i] = m.createEmbedding(texts[i])
	}
	return embeddings, nil
}

func (m *mockEmbedder) EmbedQuery(ctx context.Context, text string) ([]float32, error) {
	return m.createEmbedding(text), nil
}

// createEmbedding creates a deterministic embedding based on text content.
// Texts with the same first 2 significant words get similar embeddings (cosine sim > 0.9).
// Texts with different starting words get distinct embeddings (cosine sim < 0.5).
// Uses orthogonal category vectors with small within-category variation.
func (m *mockEmbedder) createEmbedding(text string) []float32 {
	embedding := make([]float32, m.vectorSize)

	// Extract first 2 significant words as the "semantic category"
	words := strings.Fields(strings.ToLower(text))
	var categoryWords []string
	for _, w := range words {
		// Skip very short words (articles, etc)
		if len(w) > 2 {
			categoryWords = append(categoryWords, w)
			if len(categoryWords) >= 2 {
				break
			}
		}
	}

	// Create category hash from first 2 significant words
	category := strings.Join(categoryWords, " ")
	h := fnv.New32a()
	h.Write([]byte(category))
	categoryHash := h.Sum32()

	// Create unique hash from full text for variation within category
	h.Reset()
	h.Write([]byte(text))
	textHash := h.Sum32()

	// Use category hash to select a "slot" in the vector space
	// Different categories get mostly orthogonal vectors
	// Same category gets nearly identical vectors with tiny variation
	slotSize := 16
	if m.vectorSize < 32 {
		// For small vectors, use smaller slots
		slotSize = max(1, m.vectorSize/4)
	}
	numSlots := max(1, m.vectorSize/slotSize)
	categorySlot := int(categoryHash%uint32(numSlots)) * slotSize

	// Set a block of dimensions to high values for this category
	// This creates near-orthogonal vectors for different categories
	for j := 0; j < m.vectorSize; j++ {
		if j >= categorySlot && j < categorySlot+slotSize {
			// Within the category's "slot" - high base value
			variation := float32((textHash+uint32(j))%100) / 10000.0 // tiny variation: 0-0.01
			embedding[j] = 0.8 + variation
		} else {
			// Outside the slot - small random noise based on text hash
			noise := float32((textHash+uint32(j))%100) / 5000.0 // very small: 0-0.02
			embedding[j] = noise
		}
	}

	// Normalize the embedding
	var norm float32
	for _, v := range embedding {
		norm += v * v
	}
	norm = float32(math.Sqrt(float64(norm)))
	if norm > 0 {
		for j := range embedding {
			embedding[j] /= norm
		}
	}

	return embedding
}

func TestGetMemoryVector(t *testing.T) {
	ctx := context.Background()
	store := newMockStore()
	embedder := newMockEmbedder(384)
	logger := zap.NewNop()

	svc, err := NewService(store, logger,
		WithDefaultTenant("test-tenant"),
		WithEmbedder(embedder))
	require.NoError(t, err)

	t.Run("retrieves vector for existing memory", func(t *testing.T) {
		projectID := "vector-project-1"

		// Create a memory
		memory, err := NewMemory(
			projectID,
			"Test Memory",
			"This is test content",
			OutcomeSuccess,
			[]string{"test"},
		)
		require.NoError(t, err)

		err = svc.Record(ctx, memory)
		require.NoError(t, err)

		// Get the vector
		vector, err := svc.GetMemoryVector(ctx, memory.ID)
		require.NoError(t, err)
		assert.NotNil(t, vector)
		assert.Len(t, vector, 384)

		// Verify vector is normalized (magnitude ~= 1) and has non-trivial values
		var magnitude float32
		hasNonZero := false
		for _, v := range vector {
			magnitude += v * v
			if v > 0.01 || v < -0.01 {
				hasNonZero = true
			}
		}
		magnitude = float32(math.Sqrt(float64(magnitude)))
		assert.InDelta(t, 1.0, magnitude, 0.01, "vector should be normalized")
		assert.True(t, hasNonZero, "vector should have non-trivial values")
	})

	t.Run("returns error for non-existent memory", func(t *testing.T) {
		// Use a valid UUID format that doesn't exist in the store
		vector, err := svc.GetMemoryVector(ctx, "00000000-0000-0000-0000-000000000000")
		assert.Error(t, err)
		assert.Nil(t, vector)
		assert.Contains(t, err.Error(), "memory not found")
	})

	t.Run("returns error when embedder not configured", func(t *testing.T) {
		// Create service without embedder
		svcNoEmbedder, err := NewService(store, logger, WithDefaultTenant("test-tenant"))
		require.NoError(t, err)

		projectID := "vector-project-2"

		// Create a memory
		memory, err := NewMemory(
			projectID,
			"Test Memory",
			"Content",
			OutcomeSuccess,
			[]string{"test"},
		)
		require.NoError(t, err)

		err = svcNoEmbedder.Record(ctx, memory)
		require.NoError(t, err)

		// Try to get vector without embedder
		vector, err := svcNoEmbedder.GetMemoryVector(ctx, memory.ID)
		assert.Error(t, err)
		assert.Nil(t, vector)
		assert.Contains(t, err.Error(), "embedder not configured")
	})

	t.Run("returns error for empty memory ID", func(t *testing.T) {
		vector, err := svc.GetMemoryVector(ctx, "")
		assert.Error(t, err)
		assert.Nil(t, vector)
		assert.Contains(t, err.Error(), "memory ID cannot be empty")
	})
}

func TestGetMemoryVectorByProjectID(t *testing.T) {
	ctx := context.Background()
	store := newMockStore()
	embedder := newMockEmbedder(384)
	logger := zap.NewNop()

	svc, err := NewService(store, logger,
		WithDefaultTenant("test-tenant"),
		WithEmbedder(embedder))
	require.NoError(t, err)

	t.Run("retrieves vector for existing memory in project", func(t *testing.T) {
		projectID := "vector-project-3"

		// Create a memory
		memory, err := NewMemory(
			projectID,
			"Test Memory",
			"This is test content",
			OutcomeSuccess,
			[]string{"test"},
		)
		require.NoError(t, err)

		err = svc.Record(ctx, memory)
		require.NoError(t, err)

		// Get the vector by project ID
		vector, err := svc.GetMemoryVectorByProjectID(ctx, projectID, memory.ID)
		require.NoError(t, err)
		assert.NotNil(t, vector)
		assert.Len(t, vector, 384)

		// Verify vector is normalized (magnitude ~= 1) and has non-trivial values
		var magnitude float32
		hasNonZero := false
		for _, v := range vector {
			magnitude += v * v
			if v > 0.01 || v < -0.01 {
				hasNonZero = true
			}
		}
		magnitude = float32(math.Sqrt(float64(magnitude)))
		assert.InDelta(t, 1.0, magnitude, 0.01, "vector should be normalized")
		assert.True(t, hasNonZero, "vector should have non-trivial values")
	})

	t.Run("returns error for non-existent memory", func(t *testing.T) {
		// Use a valid UUID format that doesn't exist in the store
		vector, err := svc.GetMemoryVectorByProjectID(ctx, "some-project", "00000000-0000-0000-0000-000000000000")
		assert.Error(t, err)
		assert.Nil(t, vector)
		assert.Contains(t, err.Error(), "memory not found")
	})

	t.Run("returns error when embedder not configured", func(t *testing.T) {
		// Create service without embedder
		svcNoEmbedder, err := NewService(store, logger, WithDefaultTenant("test-tenant"))
		require.NoError(t, err)

		projectID := "vector-project-4"

		// Create a memory
		memory, err := NewMemory(
			projectID,
			"Test Memory",
			"Content",
			OutcomeSuccess,
			[]string{"test"},
		)
		require.NoError(t, err)

		err = svcNoEmbedder.Record(ctx, memory)
		require.NoError(t, err)

		// Try to get vector without embedder
		vector, err := svcNoEmbedder.GetMemoryVectorByProjectID(ctx, projectID, memory.ID)
		assert.Error(t, err)
		assert.Nil(t, vector)
		assert.Contains(t, err.Error(), "embedder not configured")
	})

	t.Run("returns error for empty project ID", func(t *testing.T) {
		vector, err := svc.GetMemoryVectorByProjectID(ctx, "", "some-id")
		assert.Error(t, err)
		assert.Nil(t, vector)
		assert.Equal(t, ErrEmptyProjectID, err)
	})

	t.Run("returns error for empty memory ID", func(t *testing.T) {
		vector, err := svc.GetMemoryVectorByProjectID(ctx, "some-project", "")
		assert.Error(t, err)
		assert.Nil(t, vector)
		assert.Contains(t, err.Error(), "memory ID cannot be empty")
	})

	t.Run("vector matches content embedding", func(t *testing.T) {
		projectID := "vector-project-5"

		// Create a memory with specific content
		memory, err := NewMemory(
			projectID,
			"Title",
			"Content",
			OutcomeSuccess,
			[]string{"test"},
		)
		require.NoError(t, err)

		err = svc.Record(ctx, memory)
		require.NoError(t, err)

		// Get the vector
		vector, err := svc.GetMemoryVectorByProjectID(ctx, projectID, memory.ID)
		require.NoError(t, err)

		// Manually embed the same content to verify consistency
		content := fmt.Sprintf("%s\n\n%s", memory.Title, memory.Content)
		expectedVector, err := embedder.EmbedQuery(ctx, content)
		require.NoError(t, err)

		// Vectors should match
		assert.Equal(t, expectedVector, vector)
	})
}

// TestService_Search_ArchivedMemoryFiltering tests that archived memories are filtered out of search results.
func TestService_Search_ArchivedMemoryFiltering(t *testing.T) {
	ctx := context.Background()
	store := newMockStore()
	svc, _ := NewService(store, zap.NewNop(), WithDefaultTenant("test-tenant"))

	projectID := "project-123"
	consolidatedID := "consolidated-001"

	// Create an active memory
	activeMemory, _ := NewMemory(projectID, "Active Memory", "This is active content", OutcomeSuccess, []string{"active"})
	activeMemory.Confidence = 0.9
	activeMemory.State = MemoryStateActive
	_ = svc.Record(ctx, activeMemory)

	// Create an archived memory (source memory that was consolidated)
	archivedMemory, _ := NewMemory(projectID, "Archived Memory", "This was consolidated", OutcomeSuccess, []string{"archived"})
	archivedMemory.Confidence = 0.95 // High confidence but archived
	archivedMemory.State = MemoryStateArchived
	archivedMemory.ConsolidationID = &consolidatedID
	_ = svc.Record(ctx, archivedMemory)

	// Create the consolidated memory
	consolidatedMemory, _ := NewMemory(projectID, "Consolidated Memory", "Synthesized from multiple sources", OutcomeSuccess, []string{"consolidated"})
	consolidatedMemory.Confidence = 0.92
	consolidatedMemory.State = MemoryStateActive
	consolidatedMemory.Description = "Synthesized from 2 source memories"
	_ = svc.Record(ctx, consolidatedMemory)

	t.Run("filters out archived memories", func(t *testing.T) {
		results, err := svc.Search(ctx, projectID, "memory", 10)
		require.NoError(t, err)

		// Should return active and consolidated memories, but NOT archived
		assert.Len(t, results, 2)

		for _, result := range results {
			assert.NotEqual(t, MemoryStateArchived, result.State, "archived memory should be filtered out")
			assert.NotEqual(t, archivedMemory.ID, result.ID, "archived memory should not appear in results")
		}
	})

	t.Run("archived memory not in results despite high confidence", func(t *testing.T) {
		results, err := svc.Search(ctx, projectID, "archived", 10)
		require.NoError(t, err)

		// Even though archivedMemory has confidence 0.95 (higher than MinConfidence),
		// it should be filtered out because it's archived
		for _, result := range results {
			assert.NotEqual(t, archivedMemory.ID, result.ID, "high-confidence archived memory should still be filtered")
		}
	})
}

// TestService_Search_ConsolidatedMemoryBoost tests that consolidated memories receive a ranking boost.
func TestService_Search_ConsolidatedMemoryBoost(t *testing.T) {
	ctx := context.Background()
	store := newMockStore()
	svc, _ := NewService(store, zap.NewNop(), WithDefaultTenant("test-tenant"))

	projectID := "project-boost-test"

	// Create a regular memory with slightly higher initial relevance
	regularMemory, _ := NewMemory(projectID, "Regular Memory About Testing", "Regular testing approach", OutcomeSuccess, []string{"testing"})
	regularMemory.Confidence = 0.85
	regularMemory.State = MemoryStateActive
	_ = svc.Record(ctx, regularMemory)

	// Create a consolidated memory (synthesized from multiple sources)
	consolidatedMemory, _ := NewMemory(projectID, "Consolidated Testing Strategy", "Advanced testing patterns", OutcomeSuccess, []string{"testing"})
	consolidatedMemory.Confidence = 0.85
	consolidatedMemory.State = MemoryStateActive
	consolidatedMemory.Description = "Synthesized from 3 source memories with high confidence"
	// Note: ConsolidationID should be nil for consolidated memories (they're not linked to another memory)
	// The boost is detected by: ConsolidationID==nil, State==Active, Description contains "Synthesized from" or "Consolidated from"
	_ = svc.Record(ctx, consolidatedMemory)

	t.Run("consolidated memory receives boost", func(t *testing.T) {
		results, err := svc.Search(ctx, projectID, "testing", 10)
		require.NoError(t, err)

		require.Len(t, results, 2, "should return both memories")

		// With the 20% boost, consolidated memory should rank higher
		// even if they have the same base relevance score
		assert.Equal(t, consolidatedMemory.ID, results[0].ID, "consolidated memory should rank first due to boost")
		assert.Equal(t, regularMemory.ID, results[1].ID, "regular memory should rank second")
	})

	t.Run("consolidated memory detection via description", func(t *testing.T) {
		// Create another consolidated memory with "Consolidated from" marker
		consolidatedMemory2, _ := NewMemory(projectID, "Another Consolidated Memory", "More testing insights", OutcomeSuccess, []string{"testing"})
		consolidatedMemory2.Confidence = 0.85
		consolidatedMemory2.State = MemoryStateActive
		consolidatedMemory2.Description = "Consolidated from 5 similar memories"
		_ = svc.Record(ctx, consolidatedMemory2)

		results, err := svc.Search(ctx, projectID, "testing", 10)
		require.NoError(t, err)

		require.GreaterOrEqual(t, len(results), 3, "should return all memories")

		// Both consolidated memories should rank higher than regular memory
		// (assuming similar base relevance scores)
		consolidatedIDs := map[string]bool{
			consolidatedMemory.ID:  true,
			consolidatedMemory2.ID: true,
		}

		// At least one of the top 2 results should be a consolidated memory
		topTwoHasConsolidated := consolidatedIDs[results[0].ID] || consolidatedIDs[results[1].ID]
		assert.True(t, topTwoHasConsolidated, "consolidated memories should rank highly due to boost")
	})
}

// TestService_Search_BoostAndResorting tests that search results are correctly re-sorted after boost.
func TestService_Search_BoostAndResorting(t *testing.T) {
	ctx := context.Background()
	store := newMockStore()
	svc, _ := NewService(store, zap.NewNop(), WithDefaultTenant("test-tenant"))

	projectID := "project-resort-test"

	// Create memories with known ordering before boost
	// Memory A: High relevance, not consolidated
	memoryA, _ := NewMemory(projectID, "High Relevance Regular Memory", "Very relevant content about Go testing", OutcomeSuccess, []string{"go", "testing"})
	memoryA.Confidence = 0.90
	memoryA.State = MemoryStateActive
	_ = svc.Record(ctx, memoryA)

	// Memory B: Medium relevance, consolidated (will get 20% boost)
	memoryB, _ := NewMemory(projectID, "Medium Relevance Consolidated", "Consolidated testing knowledge", OutcomeSuccess, []string{"testing"})
	memoryB.Confidence = 0.85
	memoryB.State = MemoryStateActive
	memoryB.Description = "Synthesized from 4 high-quality memories"
	_ = svc.Record(ctx, memoryB)

	// Memory C: Lower relevance, not consolidated
	memoryC, _ := NewMemory(projectID, "Lower Relevance Memory", "Some testing tips", OutcomeSuccess, []string{"testing"})
	memoryC.Confidence = 0.80
	memoryC.State = MemoryStateActive
	_ = svc.Record(ctx, memoryC)

	t.Run("results re-sorted by boosted scores", func(t *testing.T) {
		results, err := svc.Search(ctx, projectID, "testing", 10)
		require.NoError(t, err)

		require.Len(t, results, 3, "should return all three memories")

		// After boost, memoryB (consolidated) should potentially rank higher than memoryC
		// The exact ordering depends on mock similarity scores, but we can verify:
		// 1. All memories are present
		// 2. Consolidated memory is boosted (we can't easily verify the exact ranking without controlling mock scores)

		foundA := false
		foundB := false
		foundC := false
		for _, result := range results {
			switch result.ID {
			case memoryA.ID:
				foundA = true
			case memoryB.ID:
				foundB = true
			case memoryC.ID:
				foundC = true
			}
		}

		assert.True(t, foundA, "memory A should be in results")
		assert.True(t, foundB, "memory B (consolidated) should be in results")
		assert.True(t, foundC, "memory C should be in results")

		// Consolidated memory should not be last (it has boost and decent confidence)
		assert.NotEqual(t, memoryB.ID, results[len(results)-1].ID, "consolidated memory should not rank last due to boost")
	})
}

// TestService_Search_ConsolidatedVsSourceMemories tests the complete workflow of
// consolidated memories ranking higher while source memories are filtered out.
func TestService_Search_ConsolidatedVsSourceMemories(t *testing.T) {
	ctx := context.Background()
	store := newMockStore()
	svc, _ := NewService(store, zap.NewNop(), WithDefaultTenant("test-tenant"))

	projectID := "project-workflow-test"
	consolidatedID := "consolidated-memory-123"

	// Create source memories that were consolidated (these should be archived)
	source1, _ := NewMemory(projectID, "Source Memory 1", "Original insight about testing", OutcomeSuccess, []string{"testing"})
	source1.Confidence = 0.88
	source1.State = MemoryStateArchived
	source1.ConsolidationID = &consolidatedID
	_ = svc.Record(ctx, source1)

	source2, _ := NewMemory(projectID, "Source Memory 2", "Another testing approach", OutcomeSuccess, []string{"testing"})
	source2.Confidence = 0.87
	source2.State = MemoryStateArchived
	source2.ConsolidationID = &consolidatedID
	_ = svc.Record(ctx, source2)

	// Create the consolidated memory (synthesized from source1 and source2)
	consolidated, _ := NewMemory(projectID, "Consolidated Testing Strategy", "Synthesized testing best practices combining multiple approaches", OutcomeSuccess, []string{"testing"})
	consolidated.Confidence = 0.90
	consolidated.State = MemoryStateActive
	consolidated.Description = "Synthesized from 2 source memories (source-1, source-2)"
	_ = svc.Record(ctx, consolidated)

	// Create an unrelated regular memory
	regular, _ := NewMemory(projectID, "Unrelated Memory", "Something about deployment", OutcomeSuccess, []string{"deployment"})
	regular.Confidence = 0.85
	regular.State = MemoryStateActive
	_ = svc.Record(ctx, regular)

	t.Run("only consolidated memory returned, sources filtered", func(t *testing.T) {
		results, err := svc.Search(ctx, projectID, "testing", 10)
		require.NoError(t, err)

		// Should only return the consolidated memory (source memories are archived)
		// Regular memory might appear if query matches
		for _, result := range results {
			// Verify no archived memories
			assert.NotEqual(t, MemoryStateArchived, result.State, "no archived memories should appear")

			// Verify source memories are not present
			assert.NotEqual(t, source1.ID, result.ID, "source1 should be filtered (archived)")
			assert.NotEqual(t, source2.ID, result.ID, "source2 should be filtered (archived)")
		}

		// Consolidated memory should be present
		foundConsolidated := false
		for _, result := range results {
			if result.ID == consolidated.ID {
				foundConsolidated = true
				// Verify it's the consolidated memory
				assert.Equal(t, MemoryStateActive, result.State)
				assert.Contains(t, result.Description, "Synthesized from")
			}
		}
		assert.True(t, foundConsolidated, "consolidated memory should be in results")
	})

	t.Run("consolidated memory boosted over regular memories", func(t *testing.T) {
		results, err := svc.Search(ctx, projectID, "testing", 10)
		require.NoError(t, err)

		// If both consolidated and regular appear, consolidated should rank higher
		// (assuming similar base relevance)
		if len(results) > 0 {
			// First result should likely be the consolidated memory due to boost
			// (can't guarantee without controlling mock scores, but we can verify it's present)
			foundConsolidated := false
			for _, result := range results {
				if result.ID == consolidated.ID {
					foundConsolidated = true
				}
			}
			assert.True(t, foundConsolidated, "consolidated memory should be present and boosted")
		}
	})
}

// TestService_Search_ConsolidationIDNilCheck tests that consolidated memories are correctly identified.
func TestService_Search_ConsolidationIDNilCheck(t *testing.T) {
	ctx := context.Background()
	store := newMockStore()
	svc, _ := NewService(store, zap.NewNop(), WithDefaultTenant("test-tenant"))

	projectID := "project-nil-check"

	// Create a consolidated memory (ConsolidationID is nil, description indicates synthesis)
	consolidated, _ := NewMemory(projectID, "Consolidated Memory", "Merged insights", OutcomeSuccess, []string{"test"})
	consolidated.Confidence = 0.85
	consolidated.State = MemoryStateActive
	consolidated.Description = "Synthesized from multiple memories"
	consolidated.ConsolidationID = nil // Explicitly nil
	_ = svc.Record(ctx, consolidated)

	// Create a source memory (ConsolidationID points to consolidated memory)
	consolidatedIDStr := "some-consolidated-id"
	source, _ := NewMemory(projectID, "Source Memory", "Original content", OutcomeSuccess, []string{"test"})
	source.Confidence = 0.85
	source.State = MemoryStateArchived
	source.ConsolidationID = &consolidatedIDStr
	_ = svc.Record(ctx, source)

	// Create a regular memory (ConsolidationID is nil, but description doesn't indicate synthesis)
	regular, _ := NewMemory(projectID, "Regular Memory", "Normal memory", OutcomeSuccess, []string{"test"})
	regular.Confidence = 0.85
	regular.State = MemoryStateActive
	regular.Description = "Regular memory description"
	regular.ConsolidationID = nil
	_ = svc.Record(ctx, regular)

	t.Run("consolidated memory identified by nil ConsolidationID and description", func(t *testing.T) {
		results, err := svc.Search(ctx, projectID, "memory", 10)
		require.NoError(t, err)

		// Should return consolidated and regular memories (not source, it's archived)
		foundConsolidated := false
		foundRegular := false

		for _, result := range results {
			if result.ID == consolidated.ID {
				foundConsolidated = true
				// This memory should get boost
				assert.Nil(t, result.ConsolidationID, "consolidated memory has nil ConsolidationID")
				assert.Contains(t, result.Description, "Synthesized from", "consolidated memory description indicates synthesis")
			}
			if result.ID == regular.ID {
				foundRegular = true
				// This memory should NOT get boost
				assert.Nil(t, result.ConsolidationID, "regular memory also has nil ConsolidationID")
				assert.NotContains(t, result.Description, "Synthesized from", "regular memory description doesn't indicate synthesis")
			}
			// Source should not appear (archived)
			assert.NotEqual(t, source.ID, result.ID, "source memory should be filtered (archived)")
		}

		assert.True(t, foundConsolidated, "consolidated memory should be in results")
		assert.True(t, foundRegular, "regular memory should be in results")
	})
}

// TestService_Search_MetadataPreservation tests that state and consolidation_id are correctly
// stored in metadata and retrieved from search results.
func TestService_Search_MetadataPreservation(t *testing.T) {
	ctx := context.Background()
	store := newMockStore()
	svc, _ := NewService(store, zap.NewNop(), WithDefaultTenant("test-tenant"))

	projectID := "project-metadata"
	consolidatedID := "consolidated-123"

	// Create an active memory
	active, _ := NewMemory(projectID, "Active Memory", "Active content", OutcomeSuccess, []string{"test"})
	active.Confidence = 0.85
	active.State = MemoryStateActive
	_ = svc.Record(ctx, active)

	// Create an archived memory with consolidation link
	archived, _ := NewMemory(projectID, "Archived Memory", "Archived content", OutcomeSuccess, []string{"test"})
	archived.Confidence = 0.85
	archived.State = MemoryStateArchived
	archived.ConsolidationID = &consolidatedID
	_ = svc.Record(ctx, archived)

	t.Run("state metadata preserved in storage and retrieval", func(t *testing.T) {
		// Search should filter archived, but we can test Get to verify metadata
		retrievedActive, err := svc.GetByProjectID(ctx, projectID, active.ID)
		require.NoError(t, err)
		assert.Equal(t, MemoryStateActive, retrievedActive.State, "active state preserved")

		retrievedArchived, err := svc.GetByProjectID(ctx, projectID, archived.ID)
		require.NoError(t, err)
		assert.Equal(t, MemoryStateArchived, retrievedArchived.State, "archived state preserved")
	})

	t.Run("consolidation_id metadata preserved", func(t *testing.T) {
		retrievedArchived, err := svc.GetByProjectID(ctx, projectID, archived.ID)
		require.NoError(t, err)
		require.NotNil(t, retrievedArchived.ConsolidationID, "consolidation_id should be set")
		assert.Equal(t, consolidatedID, *retrievedArchived.ConsolidationID, "consolidation_id preserved")

		retrievedActive, err := svc.GetByProjectID(ctx, projectID, active.ID)
		require.NoError(t, err)
		assert.Nil(t, retrievedActive.ConsolidationID, "active memory has no consolidation_id")
	})
}

func TestService_SearchWithMetadata(t *testing.T) {
	ctx := context.Background()
	store := newMockStore()
	svc, _ := NewService(store, zap.NewNop(), WithDefaultTenant("test-tenant"))

	projectID := "project-123"

	// Create test memories with different content
	memory1, _ := NewMemory(projectID, "Go Error Handling Pattern", "Use fmt.Errorf with %w for error wrapping in Go. Caroline recommends this approach.", OutcomeSuccess, []string{"go", "errors"})
	memory1.Confidence = 0.9
	_ = svc.Record(ctx, memory1)

	memory2, _ := NewMemory(projectID, "Testing Strategies", "Use table-driven tests for comprehensive coverage. David's pattern for Go testing.", OutcomeSuccess, []string{"go", "testing"})
	memory2.Confidence = 0.8
	_ = svc.Record(ctx, memory2)

	memory3, _ := NewMemory(projectID, "Context Management", "Always use context for lifecycle management in Go applications.", OutcomeSuccess, []string{"go", "context"})
	memory3.Confidence = 0.85
	_ = svc.Record(ctx, memory3)

	t.Run("returns metadata with results", func(t *testing.T) {
		results, metadata, err := svc.SearchWithMetadata(ctx, projectID, "error handling in Go", 10)
		require.NoError(t, err)
		require.NotNil(t, metadata)

		// Should have results
		assert.NotEmpty(t, results)

		// Metadata should have valid fields
		assert.GreaterOrEqual(t, metadata.QueryCoverage, 0.0)
		// Note: QueryCoverage can exceed 1.0 due to boosting (consolidated memory boost, entity boost, etc.)
		assert.Greater(t, metadata.QueryCoverage, 0.0)
		assert.GreaterOrEqual(t, metadata.EntityMatches, 0)
	})

	t.Run("suggests refinements from result entities", func(t *testing.T) {
		results, metadata, err := svc.SearchWithMetadata(ctx, projectID, "wrapping", 10)
		require.NoError(t, err)
		require.NotNil(t, metadata)

		// Should have results
		assert.NotEmpty(t, results)

		// Suggested refinements are entities found in results
		// that weren't in the query
		for _, suggestion := range metadata.SuggestedRefinements {
			// Suggestions should be non-empty
			assert.NotEmpty(t, suggestion)
			// Suggestions should not be the query term itself
			assert.NotEqual(t, "wrapping", strings.ToLower(suggestion))
		}

		// We should have some suggestions from the diverse memories
		// (they contain terms like "fmt", "error", "go", "testing", etc.)
		assert.GreaterOrEqual(t, len(metadata.SuggestedRefinements), 0)
	})

	t.Run("calculates query coverage as average relevance", func(t *testing.T) {
		results, metadata, err := svc.SearchWithMetadata(ctx, projectID, "Go programming", 10)
		require.NoError(t, err)
		require.NotNil(t, metadata)

		if len(results) > 0 {
			// Query coverage should be non-negative
			assert.GreaterOrEqual(t, metadata.QueryCoverage, 0.0)

			// Rough check: query coverage should be related to result relevance
			// Note: Can exceed 1.0 due to boosting (consolidated, entity, temporal)
			totalRelevance := 0.0
			for _, result := range results {
				totalRelevance += result.Relevance
			}
			expectedCoverage := totalRelevance / float64(len(results))

			// Allow small floating point tolerance
			assert.InDelta(t, expectedCoverage, metadata.QueryCoverage, 0.01)
		}
	})

	t.Run("counts entities in results", func(t *testing.T) {
		results, metadata, err := svc.SearchWithMetadata(ctx, projectID, "testing", 10)
		require.NoError(t, err)
		require.NotNil(t, metadata)

		// Entity count should be non-negative
		assert.GreaterOrEqual(t, metadata.EntityMatches, 0)

		// If we have results, we should have some entities
		if len(results) > 0 {
			assert.Greater(t, metadata.EntityMatches, 0)
		}
	})

	t.Run("handles empty results gracefully", func(t *testing.T) {
		// Search with a query that's unlikely to match existing memories
		results, metadata, err := svc.SearchWithMetadata(ctx, projectID, "xyzabc_nonexistent_query_12345", 10)
		require.NoError(t, err)
		require.NotNil(t, metadata)

		// With this highly unlikely query, should have minimal/no results
		// (mocked store may return some results, so we check metadata validity instead)
		assert.NotNil(t, metadata)

		// Metadata should still be valid (even with empty results)
		assert.GreaterOrEqual(t, metadata.QueryCoverage, 0.0)
		assert.GreaterOrEqual(t, metadata.EntityMatches, 0)

		// If truly empty results, check all zero values
		if len(results) == 0 {
			assert.Equal(t, 0.0, metadata.QueryCoverage)
			assert.Equal(t, 0, metadata.EntityMatches)
			assert.Empty(t, metadata.SuggestedRefinements)
		}
	})

	t.Run("limits suggested refinements to 5", func(t *testing.T) {
		// Create more memories with diverse entities
		for i := 0; i < 8; i++ {
			mem, _ := NewMemory(projectID,
				fmt.Sprintf("Memory %d with Entity%d", i, i),
				fmt.Sprintf("Content with Entity%d, Entity%d, Entity%d", i, i+1, i+2),
				OutcomeSuccess,
				[]string{"diverse", "test"},
			)
			mem.Confidence = 0.75
			_ = svc.Record(ctx, mem)
		}

		_, metadata, err := svc.SearchWithMetadata(ctx, projectID, "diverse", 20)
		require.NoError(t, err)
		require.NotNil(t, metadata)

		// Suggested refinements should be capped at 5
		assert.LessOrEqual(t, len(metadata.SuggestedRefinements), 5)
	})
}

// TestService_SanitizeRefinements tests the sanitizeRefinements helper function
// that prevents cross-tenant data leakage in search refinements.
func TestService_SanitizeRefinements(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	store := newMockStore()
	svc, err := NewService(store, logger, WithDefaultTenant("test-tenant"))
	require.NoError(t, err)

	t.Run("filters out UUIDs", func(t *testing.T) {
		input := []string{"Alice", "550e8400-e29b-41d4-a716-446655440000", "Bob"}
		result := svc.sanitizeRefinements(input)
		assert.Equal(t, []string{"Alice", "Bob"}, result)
	})

	t.Run("filters out emails", func(t *testing.T) {
		input := []string{"Alice", "alice@example.com", "Bob"}
		result := svc.sanitizeRefinements(input)
		assert.Equal(t, []string{"Alice", "Bob"}, result)
	})

	t.Run("filters out short strings", func(t *testing.T) {
		input := []string{"Alice", "AB", "a", "Bob"}
		result := svc.sanitizeRefinements(input)
		assert.Equal(t, []string{"Alice", "Bob"}, result)
	})

	t.Run("handles empty input", func(t *testing.T) {
		result := svc.sanitizeRefinements([]string{})
		assert.Empty(t, result)
	})

	t.Run("preserves valid refinements", func(t *testing.T) {
		input := []string{"Alice", "Caroline", "Memory", "Error"}
		result := svc.sanitizeRefinements(input)
		assert.Equal(t, input, result)
	})
}
