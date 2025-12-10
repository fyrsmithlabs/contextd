// Package framework provides the integration test framework for contextd.
package framework

import (
	"context"
	"fmt"
	"sync"

	"github.com/fyrsmithlabs/contextd/internal/reasoningbank"
	"github.com/fyrsmithlabs/contextd/internal/vectorstore"
	"go.uber.org/zap"
)

// SharedStoreConfig configures a shared vector store for multiple developers.
type SharedStoreConfig struct {
	ProjectID string
	Logger    *zap.Logger
}

// SharedStore represents a shared vector store that multiple developers can use.
// This simulates the production scenario where developers share a Qdrant backend.
type SharedStore struct {
	store  vectorstore.Store
	logger *zap.Logger
}

// NewSharedStore creates a new shared store for cross-developer testing.
// Uses a mock store implementation that provides deterministic behavior for tests.
func NewSharedStore(cfg SharedStoreConfig) (*SharedStore, error) {
	logger := cfg.Logger
	if logger == nil {
		logger = zap.NewNop()
	}

	// Use mock store for deterministic testing behavior
	// Real chromem/qdrant would require actual embeddings with semantic similarity
	store := newMockVectorStore()

	return &SharedStore{
		store:  store,
		logger: logger,
	}, nil
}

// Store returns the underlying vector store.
func (s *SharedStore) Store() vectorstore.Store {
	return s.store
}

// Close closes the shared store.
func (s *SharedStore) Close() error {
	return s.store.Close()
}

// mockVectorStore provides a simple in-memory store for testing.
// Returns all documents that pass filters (no vector similarity).
type mockVectorStore struct {
	mu          sync.RWMutex
	collections map[string][]vectorstore.Document
}

func newMockVectorStore() *mockVectorStore {
	return &mockVectorStore{
		collections: make(map[string][]vectorstore.Document),
	}
}

func (m *mockVectorStore) AddDocuments(ctx context.Context, docs []vectorstore.Document) ([]string, error) {
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

func (m *mockVectorStore) Search(ctx context.Context, query string, k int) ([]vectorstore.SearchResult, error) {
	return m.SearchInCollection(ctx, "default", query, k, nil)
}

func (m *mockVectorStore) SearchWithFilters(ctx context.Context, query string, k int, filters map[string]interface{}) ([]vectorstore.SearchResult, error) {
	return m.SearchInCollection(ctx, "default", query, k, filters)
}

func (m *mockVectorStore) SearchInCollection(ctx context.Context, collectionName string, query string, k int, filters map[string]interface{}) ([]vectorstore.SearchResult, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

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

			if !shouldInclude {
				continue
			}
		}

		results = append(results, vectorstore.SearchResult{
			ID:       doc.ID,
			Content:  doc.Content,
			Metadata: doc.Metadata,
			Score:    0.9, // Mock high similarity
		})

		if len(results) >= k {
			break
		}
	}

	return results, nil
}

func (m *mockVectorStore) DeleteDocuments(ctx context.Context, ids []string) error {
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

func (m *mockVectorStore) DeleteDocumentsFromCollection(ctx context.Context, collectionName string, ids []string) error {
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

func (m *mockVectorStore) CreateCollection(ctx context.Context, collectionName string, vectorSize int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.collections[collectionName]; exists {
		return vectorstore.ErrCollectionExists
	}
	m.collections[collectionName] = []vectorstore.Document{}
	return nil
}

func (m *mockVectorStore) DeleteCollection(ctx context.Context, collectionName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.collections, collectionName)
	return nil
}

func (m *mockVectorStore) CollectionExists(ctx context.Context, collectionName string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	_, exists := m.collections[collectionName]
	return exists, nil
}

func (m *mockVectorStore) ListCollections(ctx context.Context) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, 0, len(m.collections))
	for name := range m.collections {
		names = append(names, name)
	}
	return names, nil
}

func (m *mockVectorStore) GetDocument(ctx context.Context, collectionName, docID string) (*vectorstore.Document, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	docs, exists := m.collections[collectionName]
	if !exists {
		return nil, vectorstore.ErrCollectionNotFound
	}
	for _, doc := range docs {
		if doc.ID == docID {
			return &doc, nil
		}
	}
	return nil, fmt.Errorf("document not found: %s", docID)
}

func (m *mockVectorStore) UpdateDocument(ctx context.Context, doc vectorstore.Document) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	collectionName := doc.Collection
	if collectionName == "" {
		collectionName = "default"
	}

	docs, exists := m.collections[collectionName]
	if !exists {
		return vectorstore.ErrCollectionNotFound
	}

	for i, d := range docs {
		if d.ID == doc.ID {
			m.collections[collectionName][i] = doc
			return nil
		}
	}
	return fmt.Errorf("document not found: %s", doc.ID)
}

func (m *mockVectorStore) Close() error {
	return nil
}

func (m *mockVectorStore) SearchByCollection(ctx context.Context, collectionName string, query string, k int) ([]vectorstore.SearchResult, error) {
	return m.SearchInCollection(ctx, collectionName, query, k, nil)
}

func (m *mockVectorStore) ExactSearch(ctx context.Context, collectionName string, query string, k int) ([]vectorstore.SearchResult, error) {
	// For mock, exact search behaves the same as regular search
	return m.SearchInCollection(ctx, collectionName, query, k, nil)
}

func (m *mockVectorStore) GetCollectionInfo(ctx context.Context, collectionName string) (*vectorstore.CollectionInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	docs, exists := m.collections[collectionName]
	if !exists {
		return nil, vectorstore.ErrCollectionNotFound
	}

	return &vectorstore.CollectionInfo{
		Name:       collectionName,
		PointCount: len(docs),
		VectorSize: 384, // Mock vector size
	}, nil
}

// DeveloperConfig configures a simulated developer.
type DeveloperConfig struct {
	ID        string
	TenantID  string
	ProjectID string
	Logger    *zap.Logger
}

// MemoryRecord represents a memory to record.
type MemoryRecord struct {
	Title   string
	Content string
	Outcome string
	Tags    []string
}

// MemoryResult represents a search result.
type MemoryResult struct {
	ID         string
	Title      string
	Content    string
	Confidence float64
}

// SessionStats tracks tool usage during a session.
type SessionStats struct {
	MemoryRecords   int
	MemorySearches  int
	MemoryFeedbacks int
	Checkpoints     int
	TotalToolCalls  int
}

// Developer simulates a developer using contextd.
type Developer struct {
	id        string
	tenantID  string
	projectID string
	logger    *zap.Logger

	mu              sync.RWMutex
	contextdRunning bool
	stats           SessionStats

	// Shared store for cross-developer scenarios (nil if using own store)
	sharedStore *SharedStore

	// Internal services (simplified for testing - uses in-memory store)
	reasoningBank *reasoningbank.Service
	vectorStore   vectorstore.Store
	ownsStore     bool // true if we created the store and should close it
}

// NewDeveloper creates a new developer simulator with its own isolated store.
func NewDeveloper(cfg DeveloperConfig) (*Developer, error) {
	if cfg.ID == "" {
		return nil, fmt.Errorf("ID is required")
	}
	if cfg.TenantID == "" {
		return nil, fmt.Errorf("TenantID is required")
	}

	logger := cfg.Logger
	if logger == nil {
		logger = zap.NewNop()
	}

	return &Developer{
		id:        cfg.ID,
		tenantID:  cfg.TenantID,
		projectID: cfg.ProjectID,
		logger:    logger,
	}, nil
}

// NewDeveloperWithStore creates a developer simulator using a shared store.
// This enables cross-developer knowledge sharing scenarios.
func NewDeveloperWithStore(cfg DeveloperConfig, shared *SharedStore) (*Developer, error) {
	if cfg.ID == "" {
		return nil, fmt.Errorf("ID is required")
	}
	if cfg.TenantID == "" {
		return nil, fmt.Errorf("TenantID is required")
	}
	if shared == nil {
		return nil, fmt.Errorf("shared store is required")
	}

	logger := cfg.Logger
	if logger == nil {
		logger = shared.logger
	}

	return &Developer{
		id:          cfg.ID,
		tenantID:    cfg.TenantID,
		projectID:   cfg.ProjectID,
		logger:      logger,
		sharedStore: shared,
	}, nil
}

// ID returns the developer's ID.
func (d *Developer) ID() string {
	return d.id
}

// TenantID returns the developer's tenant ID.
func (d *Developer) TenantID() string {
	return d.tenantID
}

// StartContextd starts the contextd services for this developer.
func (d *Developer) StartContextd(ctx context.Context) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.contextdRunning {
		return fmt.Errorf("contextd already running")
	}

	var store vectorstore.Store

	if d.sharedStore != nil {
		// Use shared store for cross-developer scenarios
		store = d.sharedStore.Store()
		d.ownsStore = false
	} else {
		// Create own isolated store
		embedder := newTestEmbedder(384)
		var err error
		store, err = vectorstore.NewChromemStore(vectorstore.ChromemConfig{
			Path: "", // Empty = in-memory
		}, embedder, d.logger)
		if err != nil {
			return fmt.Errorf("creating vector store: %w", err)
		}
		d.ownsStore = true
	}
	d.vectorStore = store

	// Create reasoning bank service
	svc, err := reasoningbank.NewService(store, d.logger)
	if err != nil {
		return fmt.Errorf("creating reasoning bank: %w", err)
	}
	d.reasoningBank = svc

	d.contextdRunning = true
	d.stats = SessionStats{} // Reset stats

	return nil
}

// StopContextd stops the contextd services.
func (d *Developer) StopContextd(ctx context.Context) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if !d.contextdRunning {
		return nil
	}

	// Only close the store if we own it (not shared)
	if d.vectorStore != nil && d.ownsStore {
		d.vectorStore.Close()
	}

	d.contextdRunning = false
	d.reasoningBank = nil
	d.vectorStore = nil

	return nil
}

// IsContextdRunning returns whether contextd is running.
func (d *Developer) IsContextdRunning() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.contextdRunning
}

// RecordMemory records a memory via contextd.
func (d *Developer) RecordMemory(ctx context.Context, record MemoryRecord) (string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if !d.contextdRunning {
		return "", fmt.Errorf("contextd not running")
	}

	outcome := reasoningbank.OutcomeSuccess
	if record.Outcome == "failure" {
		outcome = reasoningbank.OutcomeFailure
	}

	memory, err := reasoningbank.NewMemory(d.projectID, record.Title, record.Content, outcome, record.Tags)
	if err != nil {
		return "", fmt.Errorf("creating memory: %w", err)
	}

	if err := d.reasoningBank.Record(ctx, memory); err != nil {
		return "", fmt.Errorf("recording memory: %w", err)
	}

	d.stats.MemoryRecords++
	d.stats.TotalToolCalls++

	return memory.ID, nil
}

// SearchMemory searches for memories via contextd.
func (d *Developer) SearchMemory(ctx context.Context, query string, limit int) ([]MemoryResult, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if !d.contextdRunning {
		return nil, fmt.Errorf("contextd not running")
	}

	results, err := d.reasoningBank.Search(ctx, d.projectID, query, limit)
	if err != nil {
		return nil, fmt.Errorf("searching memories: %w", err)
	}

	d.stats.MemorySearches++
	d.stats.TotalToolCalls++

	memoryResults := make([]MemoryResult, len(results))
	for i, r := range results {
		memoryResults[i] = MemoryResult{
			ID:         r.ID,
			Title:      r.Title,
			Content:    r.Content,
			Confidence: r.Confidence,
		}
	}

	return memoryResults, nil
}

// GiveFeedback gives feedback on a memory.
func (d *Developer) GiveFeedback(ctx context.Context, memoryID string, helpful bool, reasoning string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if !d.contextdRunning {
		return fmt.Errorf("contextd not running")
	}

	// Note: reasoning is not used by the current API but kept for future use
	if err := d.reasoningBank.Feedback(ctx, memoryID, helpful); err != nil {
		return fmt.Errorf("giving feedback: %w", err)
	}

	d.stats.MemoryFeedbacks++
	d.stats.TotalToolCalls++

	return nil
}

// SessionStats returns the current session statistics.
func (d *Developer) SessionStats() SessionStats {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.stats
}

// testEmbedder is a deterministic embedder for testing.
type testEmbedder struct {
	vectorSize int
}

func newTestEmbedder(vectorSize int) *testEmbedder {
	return &testEmbedder{vectorSize: vectorSize}
}

func (e *testEmbedder) EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error) {
	embeddings := make([][]float32, len(texts))
	for i, text := range texts {
		embeddings[i] = e.makeEmbedding(text)
	}
	return embeddings, nil
}

func (e *testEmbedder) EmbedQuery(ctx context.Context, text string) ([]float32, error) {
	return e.makeEmbedding(text), nil
}

// makeEmbedding creates a normalized embedding based on text hash.
// Similar texts will have similar embeddings for testing.
func (e *testEmbedder) makeEmbedding(text string) []float32 {
	embedding := make([]float32, e.vectorSize)
	// Create deterministic embedding based on text
	hash := 0
	for _, c := range text {
		hash = (hash*31 + int(c)) % 1000
	}
	// Fill with normalized values
	var sumSq float32
	for i := range embedding {
		embedding[i] = float32((hash+i)%100) / 100.0
		sumSq += embedding[i] * embedding[i]
	}
	// Normalize to unit vector (chromem requires normalized vectors)
	if sumSq > 0 {
		norm := float32(1.0) / sqrt32(sumSq)
		for i := range embedding {
			embedding[i] *= norm
		}
	}
	return embedding
}

func sqrt32(x float32) float32 {
	if x <= 0 {
		return 0
	}
	// Newton's method for square root
	z := x
	for i := 0; i < 10; i++ {
		z = (z + x/z) / 2
	}
	return z
}
