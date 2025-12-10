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

	// Internal services (simplified for testing - uses in-memory store)
	reasoningBank *reasoningbank.Service
	vectorStore   vectorstore.Store
}

// NewDeveloper creates a new developer simulator.
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

	// Create test embedder for in-memory testing
	embedder := newTestEmbedder(384)

	// Create in-memory vector store for testing
	store, err := vectorstore.NewChromemStore(vectorstore.ChromemConfig{
		Path: "", // Empty = in-memory
	}, embedder, d.logger)
	if err != nil {
		return fmt.Errorf("creating vector store: %w", err)
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

	if d.vectorStore != nil {
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
