package reasoningbank

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/fyrsmithlabs/contextd/internal/project"
	"github.com/fyrsmithlabs/contextd/internal/vectorstore"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"go.uber.org/zap"
)

// collectionMemories is the simple collection name used within each project store.
// With StoreProvider, each project gets its own chromem.DB instance, so we don't
// need prefixed collection names like "{projectID}_memories".
const collectionMemories = "memories"

const instrumentationName = "github.com/fyrsmithlabs/contextd/internal/reasoningbank"

const (
	// MinConfidence is the minimum confidence threshold for search results.
	MinConfidence = 0.7

	// ExplicitRecordConfidence is the initial confidence for explicitly recorded memories.
	ExplicitRecordConfidence = 0.8

	// DistilledConfidence is the initial confidence for distilled memories.
	DistilledConfidence = 0.6

	// DefaultSearchLimit is the default maximum number of search results.
	DefaultSearchLimit = 10
)

// Service provides cross-session memory storage and retrieval.
//
// It stores memories in Qdrant using semantic search to surface relevant
// strategies based on similarity to the current task. Memories can be
// created explicitly via Record() or extracted asynchronously from sessions
// via the Distiller.
//
// The service uses a Bayesian confidence system that learns which signals
// (explicit feedback, usage, outcomes) best predict memory usefulness.
type Service struct {
	store         vectorstore.Store
	stores        vectorstore.StoreProvider // For database-per-project isolation
	defaultTenant string                    // Default tenant for StoreProvider (usually git username)
	signalStore   SignalStore
	confCalc      *ConfidenceCalculator
	logger        *zap.Logger

	// Telemetry
	meter      metric.Meter
	totalGauge metric.Int64ObservableGauge

	// Stats tracking for statusline
	statsMu        sync.RWMutex
	lastConfidence float64
}

// Stats contains memory service statistics for statusline display.
type Stats struct {
	LastConfidence float64
}

// ServiceOption configures a Service.
type ServiceOption func(*Service)

// WithSignalStore sets a custom signal store.
// If not provided, an in-memory signal store is used.
func WithSignalStore(ss SignalStore) ServiceOption {
	return func(s *Service) {
		s.signalStore = ss
	}
}

// WithDefaultTenant sets the default tenant ID for single-store mode.
// Required when using a single vectorstore instead of StoreProvider.
func WithDefaultTenant(tenantID string) ServiceOption {
	return func(s *Service) {
		s.defaultTenant = tenantID
	}
}

// NewService creates a new ReasoningBank service.
func NewService(store vectorstore.Store, logger *zap.Logger, opts ...ServiceOption) (*Service, error) {
	if store == nil {
		return nil, fmt.Errorf("vector store cannot be nil")
	}
	if logger == nil {
		return nil, fmt.Errorf("logger is required for ReasoningBank service")
	}

	svc := &Service{
		store:  store,
		logger: logger,
		meter:  otel.Meter(instrumentationName),
	}

	// Apply options
	for _, opt := range opts {
		opt(svc)
	}

	// Default to in-memory signal store if not provided
	if svc.signalStore == nil {
		svc.signalStore = NewInMemorySignalStore()
	}

	// Create confidence calculator
	svc.confCalc = NewConfidenceCalculator(svc.signalStore)

	// Initialize metrics
	svc.initMetrics()

	return svc, nil
}

// NewServiceWithStoreProvider creates a ReasoningBank service using StoreProvider
// for database-per-project isolation.
//
// The defaultTenant is used when deriving the store path from projectID.
// Typically this is the git username or "default" for local-first usage.
//
// This constructor enables the new architecture where each project gets its own
// chromem.DB instance at a unique filesystem path, providing physical isolation.
func NewServiceWithStoreProvider(stores vectorstore.StoreProvider, defaultTenant string, logger *zap.Logger, opts ...ServiceOption) (*Service, error) {
	if stores == nil {
		return nil, fmt.Errorf("store provider cannot be nil")
	}
	if defaultTenant == "" {
		return nil, fmt.Errorf("default tenant cannot be empty")
	}
	if logger == nil {
		return nil, fmt.Errorf("logger is required for ReasoningBank service")
	}

	svc := &Service{
		stores:        stores,
		defaultTenant: defaultTenant,
		logger:        logger,
		meter:         otel.Meter(instrumentationName),
	}

	// Apply options
	for _, opt := range opts {
		opt(svc)
	}

	// Default to in-memory signal store if not provided
	if svc.signalStore == nil {
		svc.signalStore = NewInMemorySignalStore()
	}

	// Create confidence calculator
	svc.confCalc = NewConfidenceCalculator(svc.signalStore)

	// Initialize metrics
	svc.initMetrics()

	return svc, nil
}

// getStore returns the appropriate store for the given project.
// If StoreProvider is configured, it uses database-per-project isolation.
// Otherwise, it falls back to the legacy single-store approach.
func (s *Service) getStore(ctx context.Context, projectID string) (vectorstore.Store, string, error) {
	if s.stores != nil {
		// Use StoreProvider for database-per-project isolation
		// Team is empty for direct project path (tenant/project)
		store, err := s.stores.GetProjectStore(ctx, s.defaultTenant, "", projectID)
		if err != nil {
			return nil, "", fmt.Errorf("getting project store: %w", err)
		}
		// With StoreProvider, we use simple collection names (no prefix)
		return store, collectionMemories, nil
	}

	// Legacy: single store with prefixed collection names
	if s.store == nil {
		return nil, "", fmt.Errorf("no store configured")
	}
	collectionName, err := project.GetCollectionName(projectID, project.CollectionMemories)
	if err != nil {
		return nil, "", fmt.Errorf("getting collection name: %w", err)
	}
	return s.store, collectionName, nil
}

// initMetrics initializes OpenTelemetry metrics.
func (s *Service) initMetrics() {
	var err error

	// Observable gauge for total memory count (queried on metrics scrape)
	s.totalGauge, err = s.meter.Int64ObservableGauge(
		"contextd.memory.count",
		metric.WithDescription("Current number of memories stored"),
		metric.WithUnit("{memory}"),
		metric.WithInt64Callback(s.observeMemoryCount),
	)
	if err != nil {
		s.logger.Warn("failed to create memory count gauge", zap.Error(err))
	}
}

// observeMemoryCount is called when metrics are collected to report current memory count.
func (s *Service) observeMemoryCount(ctx context.Context, observer metric.Int64Observer) error {
	// With StoreProvider only, we can't enumerate all project stores for metrics
	// This would require a registry of known projects (future enhancement)
	if s.store == nil {
		s.logger.Debug("memory count metrics unavailable with StoreProvider-only mode")
		observer.Observe(0)
		return nil
	}

	// Get count from all memory collections
	collections, err := s.store.ListCollections(ctx)
	if err != nil {
		s.logger.Debug("failed to list collections for memory count", zap.Error(err))
		return nil // Don't fail metrics collection
	}

	var total int64
	for _, coll := range collections {
		// Only count memory/reasoning collections
		if strings.Contains(coll, "memor") || strings.Contains(coll, "reasoning") {
			info, err := s.store.GetCollectionInfo(ctx, coll)
			if err == nil && info != nil {
				total += int64(info.PointCount)
			}
		}
	}

	observer.Observe(total)
	return nil
}

// Search retrieves memories by semantic similarity to the query.
//
// Returns memories with confidence >= MinConfidence, ordered by similarity score.
// Filters to only memories belonging to the specified project.
//
// FR-003: Semantic search by similarity
// FR-002: Memories include required fields
func (s *Service) Search(ctx context.Context, projectID, query string, limit int) ([]Memory, error) {
	if projectID == "" {
		return nil, ErrEmptyProjectID
	}
	if query == "" {
		return nil, fmt.Errorf("query cannot be empty")
	}
	if limit <= 0 {
		limit = DefaultSearchLimit
	}

	// Get store and collection name for this project
	store, collectionName, err := s.getStore(ctx, projectID)
	if err != nil {
		return nil, err
	}

	// Inject tenant context for payload-based isolation
	// Fail-closed: require tenant ID to be set (no fallback)
	tenantID := s.defaultTenant
	if tenantID == "" {
		return nil, fmt.Errorf("tenant ID not configured for reasoningbank service")
	}
	ctx = vectorstore.ContextWithTenant(ctx, &vectorstore.TenantInfo{
		TenantID:  tenantID,
		ProjectID: projectID,
	})

	// Check if collection exists
	exists, err := store.CollectionExists(ctx, collectionName)
	if err != nil {
		return nil, fmt.Errorf("checking collection existence: %w", err)
	}
	if !exists {
		// No memories yet for this project
		s.logger.Debug("collection does not exist",
			zap.String("collection", collectionName),
			zap.String("project_id", projectID))
		return []Memory{}, nil
	}

	// Search without store-level confidence filter (post-filter in service layer)
	// This makes the service store-agnostic - works with any vectorstore implementation
	// regardless of filter operator support ($gte, range queries, etc.)
	// Use 3x multiplier to ensure enough results after filtering, with bounds
	searchLimit := limit * 3
	if searchLimit < 30 {
		searchLimit = 30
	}
	if searchLimit > 200 {
		searchLimit = 200 // Cap to prevent excessive fetching
	}

	results, err := store.SearchInCollection(ctx, collectionName, query, searchLimit, nil)
	if err != nil {
		return nil, fmt.Errorf("searching memories: %w", err)
	}

	// Convert results to Memory structs, filter by confidence, and record usage signals
	memories := make([]Memory, 0, len(results))
	for _, result := range results {
		memory, err := s.resultToMemory(result)
		if err != nil {
			s.logger.Warn("skipping invalid memory",
				zap.String("id", result.ID),
				zap.Error(err))
			continue
		}

		// Post-filter: skip memories below confidence threshold
		if memory.Confidence < MinConfidence {
			s.logger.Debug("skipping low-confidence memory",
				zap.String("id", memory.ID),
				zap.Float64("confidence", memory.Confidence),
				zap.Float64("min_confidence", MinConfidence))
			continue
		}

		// Record usage signal for this memory (positive = retrieved in search)
		signal, err := NewSignal(memory.ID, projectID, SignalUsage, true, "")
		if err == nil {
			if err := s.signalStore.StoreSignal(ctx, signal); err != nil {
				s.logger.Warn("failed to record usage signal",
					zap.String("memory_id", memory.ID),
					zap.Error(err))
			}
		}

		memories = append(memories, *memory)

		// Stop once we have enough results
		if len(memories) >= limit {
			break
		}
	}

	// Track last confidence for statusline (use first result's confidence)
	if len(memories) > 0 {
		s.statsMu.Lock()
		s.lastConfidence = memories[0].Confidence
		s.statsMu.Unlock()
	}

	s.logger.Debug("search completed",
		zap.String("project_id", projectID),
		zap.String("query", query),
		zap.Int("limit", limit),
		zap.Int("results", len(memories)))

	return memories, nil
}

// Record creates a new memory explicitly (bypasses distillation).
//
// Sets initial confidence to ExplicitRecordConfidence (0.8) since
// explicit captures are more reliable than distilled ones.
//
// FR-007: Explicit capture via memory_record
// FR-002: Memory schema validation
func (s *Service) Record(ctx context.Context, memory *Memory) error {
	if memory == nil {
		return ErrInvalidMemory
	}

	// Set explicit record confidence ONLY if default from NewMemory (0.5)
	// AND the description doesn't indicate it's from distillation
	// This allows distilled memories and custom confidence to be preserved
	isDistilled := strings.Contains(memory.Description, "Learned from session") ||
		strings.Contains(memory.Description, "Anti-pattern learned from session")

	if !isDistilled && memory.Confidence == 0.5 {
		memory.Confidence = ExplicitRecordConfidence
	}
	if memory.Confidence == 0.0 {
		memory.Confidence = ExplicitRecordConfidence
	}

	// Set timestamps
	now := time.Now()
	if memory.CreatedAt.IsZero() {
		memory.CreatedAt = now
	}
	memory.UpdatedAt = now

	// Validate memory
	if err := memory.Validate(); err != nil {
		return fmt.Errorf("validating memory: %w", err)
	}

	// Get store and collection name
	store, collectionName, err := s.getStore(ctx, memory.ProjectID)
	if err != nil {
		return err
	}

	// Inject tenant context for payload-based isolation
	// Fail-closed: require tenant ID to be set (no fallback)
	tenantID := s.defaultTenant
	if tenantID == "" {
		return fmt.Errorf("tenant ID not configured for reasoningbank service")
	}
	ctx = vectorstore.ContextWithTenant(ctx, &vectorstore.TenantInfo{
		TenantID:  tenantID,
		ProjectID: memory.ProjectID,
	})

	// Ensure collection exists
	exists, err := store.CollectionExists(ctx, collectionName)
	if err != nil {
		return fmt.Errorf("checking collection existence: %w", err)
	}
	if !exists {
		// Create collection with store's configured vector size (0 = use default)
		if err := store.CreateCollection(ctx, collectionName, 0); err != nil {
			return fmt.Errorf("creating collection: %w", err)
		}
		s.logger.Info("created memories collection",
			zap.String("collection", collectionName),
			zap.String("project_id", memory.ProjectID))
	}

	// Convert to document
	doc := s.memoryToDocument(memory, collectionName)

	// Store in vector store
	_, err = store.AddDocuments(ctx, []vectorstore.Document{doc})
	if err != nil {
		return fmt.Errorf("storing memory: %w", err)
	}

	s.logger.Info("memory recorded",
		zap.String("id", memory.ID),
		zap.String("project_id", memory.ProjectID),
		zap.String("title", memory.Title),
		zap.Float64("confidence", memory.Confidence))

	return nil
}

// Feedback updates a memory's confidence based on user feedback.
//
// This method:
// 1. Records an explicit signal for the feedback
// 2. Learns which signal types predicted this feedback correctly (weight learning)
// 3. Recalculates the memory's confidence using the Bayesian system
//
// FR-008: Feedback loop affecting confidence
// FR-005: Confidence tracking
func (s *Service) Feedback(ctx context.Context, memoryID string, helpful bool) error {
	if memoryID == "" {
		return fmt.Errorf("memory ID cannot be empty")
	}

	// Get the memory first
	memory, err := s.Get(ctx, memoryID)
	if err != nil {
		return fmt.Errorf("getting memory: %w", err)
	}

	// Record explicit signal
	signal, err := NewSignal(memoryID, memory.ProjectID, SignalExplicit, helpful, "")
	if err != nil {
		return fmt.Errorf("creating signal: %w", err)
	}
	if err := s.signalStore.StoreSignal(ctx, signal); err != nil {
		return fmt.Errorf("storing signal: %w", err)
	}

	// Learn from feedback - update project weights based on prediction accuracy
	if err := s.confCalc.LearnFromFeedback(ctx, memory.ProjectID, memoryID, helpful); err != nil {
		s.logger.Warn("failed to learn from feedback",
			zap.String("memory_id", memoryID),
			zap.Error(err))
	}

	// Compute new confidence using Bayesian system
	newConfidence, err := s.confCalc.ComputeConfidence(ctx, memoryID, memory.ProjectID)
	if err != nil {
		// Fall back to simple adjustment if Bayesian calculation fails
		s.logger.Warn("falling back to simple confidence adjustment",
			zap.String("memory_id", memoryID),
			zap.Error(err))
		memory.AdjustConfidence(helpful)
		newConfidence = memory.Confidence
	} else {
		memory.Confidence = newConfidence
	}
	memory.UpdatedAt = time.Now()

	// Get store and collection name
	store, collectionName, err := s.getStore(ctx, memory.ProjectID)
	if err != nil {
		return err
	}

	// Delete old version from the correct collection
	if err := store.DeleteDocumentsFromCollection(ctx, collectionName, []string{memoryID}); err != nil {
		return fmt.Errorf("deleting old memory: %w", err)
	}

	// Re-add with updated confidence
	doc := s.memoryToDocument(memory, collectionName)
	_, err = store.AddDocuments(ctx, []vectorstore.Document{doc})
	if err != nil {
		return fmt.Errorf("updating memory: %w", err)
	}

	s.logger.Info("memory feedback recorded",
		zap.String("id", memoryID),
		zap.Bool("helpful", helpful),
		zap.Float64("new_confidence", memory.Confidence))

	return nil
}

// Get retrieves a memory by ID.
//
// This searches across all project collections to find the memory.
// In practice, you'd typically know the project ID, but this provides
// a fallback for when you only have the memory ID.
//
// Note: This method requires the legacy single-store configuration.
// When using StoreProvider (database-per-project), use GetByProjectID instead.
func (s *Service) Get(ctx context.Context, id string) (*Memory, error) {
	if id == "" {
		return nil, fmt.Errorf("memory ID cannot be empty")
	}

	// With StoreProvider only, we can't enumerate all project stores
	if s.store == nil {
		return nil, fmt.Errorf("Get requires legacy store; use GetByProjectID with StoreProvider")
	}

	// List all collections and search each one
	collections, err := s.store.ListCollections(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing collections: %w", err)
	}

	// Search each memories collection
	for _, collectionName := range collections {
		// Skip non-memory collections
		if len(collectionName) < 9 || collectionName[len(collectionName)-9:] != "_memories" {
			continue
		}

		// Try to find memory with this ID
		filters := map[string]interface{}{
			"id": id,
		}

		// Use a dummy query since we're filtering by ID
		results, err := s.store.SearchInCollection(ctx, collectionName, "dummy", 1, filters)
		if err != nil {
			s.logger.Warn("error searching collection",
				zap.String("collection", collectionName),
				zap.Error(err))
			continue
		}

		if len(results) > 0 {
			memory, err := s.resultToMemory(results[0])
			if err != nil {
				return nil, fmt.Errorf("converting result to memory: %w", err)
			}
			return memory, nil
		}
	}

	return nil, ErrMemoryNotFound
}

// GetByProjectID retrieves a memory by ID within a specific project.
//
// This is the preferred method when using StoreProvider (database-per-project isolation)
// as it directly accesses the project's store without enumeration.
func (s *Service) GetByProjectID(ctx context.Context, projectID, memoryID string) (*Memory, error) {
	if projectID == "" {
		return nil, ErrEmptyProjectID
	}
	if memoryID == "" {
		return nil, fmt.Errorf("memory ID cannot be empty")
	}

	// Get store and collection name
	store, collectionName, err := s.getStore(ctx, projectID)
	if err != nil {
		return nil, err
	}

	// Check if collection exists
	exists, err := store.CollectionExists(ctx, collectionName)
	if err != nil {
		return nil, fmt.Errorf("checking collection existence: %w", err)
	}
	if !exists {
		return nil, ErrMemoryNotFound
	}

	// Try to find memory with this ID
	filters := map[string]interface{}{
		"id": memoryID,
	}

	results, err := store.SearchInCollection(ctx, collectionName, "dummy", 1, filters)
	if err != nil {
		return nil, fmt.Errorf("searching for memory: %w", err)
	}

	if len(results) == 0 {
		return nil, ErrMemoryNotFound
	}

	return s.resultToMemory(results[0])
}

// Delete removes a memory by ID.
//
// Note: This method requires the legacy single-store configuration.
// When using StoreProvider (database-per-project), use DeleteByProjectID instead.
func (s *Service) Delete(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("memory ID cannot be empty")
	}

	// Get the memory first to know which collection it's in
	memory, err := s.Get(ctx, id)
	if err != nil {
		return fmt.Errorf("getting memory: %w", err)
	}

	// Delete from vector store (requires legacy store)
	if s.store == nil {
		return fmt.Errorf("Delete requires legacy store; use DeleteByProjectID with StoreProvider")
	}
	if err := s.store.DeleteDocuments(ctx, []string{id}); err != nil {
		return fmt.Errorf("deleting memory: %w", err)
	}

	s.logger.Info("memory deleted",
		zap.String("id", id),
		zap.String("project_id", memory.ProjectID))

	return nil
}

// DeleteByProjectID removes a memory by ID within a specific project.
//
// This is the preferred method when using StoreProvider (database-per-project isolation)
// as it directly accesses the project's store without enumeration.
func (s *Service) DeleteByProjectID(ctx context.Context, projectID, memoryID string) error {
	if projectID == "" {
		return ErrEmptyProjectID
	}
	if memoryID == "" {
		return fmt.Errorf("memory ID cannot be empty")
	}

	// Get store and collection name
	store, collectionName, err := s.getStore(ctx, projectID)
	if err != nil {
		return err
	}

	// Delete from the project's store
	if err := store.DeleteDocumentsFromCollection(ctx, collectionName, []string{memoryID}); err != nil {
		return fmt.Errorf("deleting memory: %w", err)
	}

	s.logger.Info("memory deleted",
		zap.String("id", memoryID),
		zap.String("project_id", projectID))

	return nil
}

// RecordOutcome records a task outcome signal for a memory.
//
// This is called by the memory_outcome MCP tool when an agent reports
// whether a task succeeded after using a retrieved memory.
//
// The outcome signal contributes to the memory's confidence score through
// the Bayesian confidence system. Positive outcomes increase confidence,
// negative outcomes decrease it based on learned weights.
//
// Returns the new confidence score after the update.
//
// FR-005d: Outcome reporting via memory_outcome tool
func (s *Service) RecordOutcome(ctx context.Context, memoryID string, succeeded bool, sessionID string) (float64, error) {
	if memoryID == "" {
		return 0, fmt.Errorf("memory ID cannot be empty")
	}

	// Get the memory first
	memory, err := s.Get(ctx, memoryID)
	if err != nil {
		return 0, fmt.Errorf("getting memory: %w", err)
	}

	// Create and store outcome signal
	signal, err := NewSignal(memoryID, memory.ProjectID, SignalOutcome, succeeded, sessionID)
	if err != nil {
		return 0, fmt.Errorf("creating signal: %w", err)
	}
	if err := s.signalStore.StoreSignal(ctx, signal); err != nil {
		return 0, fmt.Errorf("storing signal: %w", err)
	}

	// Compute new confidence using Bayesian system
	newConfidence, err := s.confCalc.ComputeConfidence(ctx, memoryID, memory.ProjectID)
	if err != nil {
		// Fall back to simple adjustment if Bayesian calculation fails
		s.logger.Warn("falling back to simple confidence adjustment",
			zap.String("memory_id", memoryID),
			zap.Error(err))
		if succeeded {
			memory.Confidence += 0.05
			if memory.Confidence > 1.0 {
				memory.Confidence = 1.0
			}
		} else {
			memory.Confidence -= 0.08
			if memory.Confidence < 0.0 {
				memory.Confidence = 0.0
			}
		}
		newConfidence = memory.Confidence
	} else {
		memory.Confidence = newConfidence
	}
	memory.UpdatedAt = time.Now()

	// Get store and collection name
	store, collectionName, err := s.getStore(ctx, memory.ProjectID)
	if err != nil {
		return 0, err
	}

	// Delete old version and re-add with updated confidence
	if err := store.DeleteDocumentsFromCollection(ctx, collectionName, []string{memoryID}); err != nil {
		return 0, fmt.Errorf("deleting old memory: %w", err)
	}

	doc := s.memoryToDocument(memory, collectionName)
	_, err = store.AddDocuments(ctx, []vectorstore.Document{doc})
	if err != nil {
		return 0, fmt.Errorf("updating memory: %w", err)
	}

	s.logger.Info("outcome recorded",
		zap.String("id", memoryID),
		zap.String("signal_id", signal.ID),
		zap.Bool("succeeded", succeeded),
		zap.Float64("new_confidence", memory.Confidence))

	return memory.Confidence, nil
}

// memoryToDocument converts a Memory to a vectorstore Document.
func (s *Service) memoryToDocument(memory *Memory, collectionName string) vectorstore.Document {
	// Combine title and content for embedding
	content := fmt.Sprintf("%s\n\n%s", memory.Title, memory.Content)

	metadata := map[string]interface{}{
		"id":          memory.ID,
		"project_id":  memory.ProjectID,
		"title":       memory.Title,
		"description": memory.Description,
		"outcome":     string(memory.Outcome),
		"confidence":  memory.Confidence,
		"usage_count": memory.UsageCount,
		"tags":        memory.Tags,
		"created_at":  memory.CreatedAt.Unix(),
		"updated_at":  memory.UpdatedAt.Unix(),
	}

	return vectorstore.Document{
		ID:         memory.ID,
		Content:    content,
		Metadata:   metadata,
		Collection: collectionName,
	}
}

// Stats returns current memory statistics for statusline display.
func (s *Service) Stats() Stats {
	s.statsMu.RLock()
	defer s.statsMu.RUnlock()
	return Stats{
		LastConfidence: s.lastConfidence,
	}
}

// Count returns the number of memories for a specific project.
func (s *Service) Count(ctx context.Context, projectID string) (int, error) {
	if projectID == "" {
		return 0, ErrEmptyProjectID
	}

	// Get store and collection name
	store, collectionName, err := s.getStore(ctx, projectID)
	if err != nil {
		return 0, err
	}

	// Check if collection exists
	exists, err := store.CollectionExists(ctx, collectionName)
	if err != nil {
		return 0, fmt.Errorf("checking collection existence: %w", err)
	}
	if !exists {
		return 0, nil
	}

	// Use GetCollectionInfo to get the point count
	info, err := store.GetCollectionInfo(ctx, collectionName)
	if err != nil {
		return 0, fmt.Errorf("getting collection info: %w", err)
	}

	return info.PointCount, nil
}

// resultToMemory converts a vectorstore SearchResult to a Memory.
func (s *Service) resultToMemory(result vectorstore.SearchResult) (*Memory, error) {
	// Extract fields from metadata
	id, _ := result.Metadata["id"].(string)
	if id == "" {
		id = result.ID
	}

	projectID, _ := result.Metadata["project_id"].(string)
	title, _ := result.Metadata["title"].(string)
	description, _ := result.Metadata["description"].(string)
	outcomeStr, _ := result.Metadata["outcome"].(string)
	confidence, _ := result.Metadata["confidence"].(float64)
	usageCount, _ := result.Metadata["usage_count"].(int)

	// Parse tags
	tags := []string{}
	if tagsIface, ok := result.Metadata["tags"]; ok {
		if tagsList, ok := tagsIface.([]interface{}); ok {
			for _, t := range tagsList {
				if tag, ok := t.(string); ok {
					tags = append(tags, tag)
				}
			}
		}
	}

	// Parse timestamps
	createdAtUnix, _ := result.Metadata["created_at"].(int64)
	updatedAtUnix, _ := result.Metadata["updated_at"].(int64)

	createdAt := time.Unix(createdAtUnix, 0)
	updatedAt := time.Unix(updatedAtUnix, 0)

	// Parse content (strip title from beginning if present)
	content := result.Content
	if len(title) > 0 && len(content) > len(title)+2 {
		// Remove "title\n\n" prefix
		if content[:len(title)] == title {
			content = content[len(title)+2:]
		}
	}

	memory := &Memory{
		ID:          id,
		ProjectID:   projectID,
		Title:       title,
		Description: description,
		Content:     content,
		Outcome:     Outcome(outcomeStr),
		Confidence:  confidence,
		UsageCount:  usageCount,
		Tags:        tags,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}

	return memory, nil
}
