package reasoningbank

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fyrsmithlabs/contextd/internal/project"
	"github.com/fyrsmithlabs/contextd/internal/vectorstore"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.uber.org/zap"
)

// collectionMemories is the simple collection name used within each project store.
// With StoreProvider, each project gets its own chromem.DB instance, so we don't
// need prefixed collection names like "{projectID}_memories".
const collectionMemories = "memories"

const instrumentationName = "github.com/fyrsmithlabs/contextd/internal/reasoningbank"

const (
	// maxQueryLength is the maximum query length for regex-based processing.
	// Prevents ReDoS attacks by limiting input size before regex execution.
	maxQueryLength = 10000

	// MinConfidence is the minimum confidence threshold for search results.
	MinConfidence = 0.7

	// ExplicitRecordConfidence is the initial confidence for explicitly recorded memories.
	ExplicitRecordConfidence = 0.8

	// DistilledConfidence is the initial confidence for distilled memories.
	DistilledConfidence = 0.6

	// DefaultSearchLimit is the default maximum number of search results.
	DefaultSearchLimit = 10

	// entityBoostFactor multiplies relevance score when a memory mentions
	// named entities extracted from the query. This improves precision for
	// questions like "What is Caroline's identity?" by prioritizing memories
	// that explicitly mention "Caroline".
	entityBoostFactor = 1.3

	// conversationBoostFactor multiplies relevance score when a memory shares
	// the same conversation context as a top-ranked result. This improves
	// multi-hop query performance by ensuring related memories are retrieved
	// together.
	conversationBoostFactor = 1.15
)

// entityRegex extracts proper nouns (capitalized words) from queries.
// Examples: "What is Caroline's identity?" → ["Caroline"]
//
//	"Tell me about John and Alice" → ["John", "Alice"]
//
// Used for entity-based boosting in memory search.
var entityRegex = regexp.MustCompile(`\b[A-Z][a-z]+\b`)

// conversationIDRegex extracts conversation IDs from memory tags.
// Examples: "[conv-26 ...]" → "conv-26", "[locomo conv-42 ...]" → "conv-42"
// Used for conversation-aware boosting in multi-hop queries.
var conversationIDRegex = regexp.MustCompile(`\bconv-\d+\b`)

// entityStopwords contains common words that appear capitalized at sentence starts
// but are not named entities. Used to filter false positives from entityRegex.
var entityStopwords = map[string]struct{}{
	"what": {}, "when": {}, "where": {}, "which": {}, "who": {}, "why": {}, "how": {},
	"tell": {}, "show": {}, "find": {}, "get": {}, "give": {}, "let": {}, "make": {},
	"did": {}, "does": {}, "can": {}, "could": {}, "would": {}, "should": {}, "will": {},
	"the": {}, "this": {}, "that": {}, "these": {}, "those": {},
	"are": {}, "was": {}, "were": {}, "has": {}, "have": {}, "had": {}, "been": {},
	"about": {}, "for": {}, "from": {}, "into": {}, "with": {},
}

// temporalKeywords indicates queries that care about recency.
// When detected, recent memories get boosted and older memories get penalized.
var temporalKeywords = []string{
	"recent", "recently", "lately", "last", "yesterday", "today",
	"earlier", "previous", "previously", "past week", "few days",
	"just now", "this morning", "this week", "month ago", "before",
}

const (
	// temporalRecentBoost applied to memories < 7 days old for temporal queries
	temporalRecentBoost = 1.25
	// temporalMediumMultiplier for memories 7-30 days old (no change)
	temporalMediumMultiplier = 1.0
	// temporalOldPenalty applied to memories > 30 days old for temporal queries
	temporalOldPenalty = 0.8
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
	embedder      vectorstore.Embedder      // For re-embedding content to retrieve vectors
	signalStore   SignalStore
	confCalc      *ConfidenceCalculator
	logger        *zap.Logger

	// Telemetry
	meter           metric.Meter
	totalGauge          metric.Int64ObservableGauge
	searchCounter       metric.Int64Counter
	recordCounter       metric.Int64Counter
	feedbackCounter     metric.Int64Counter
	outcomeCounter      metric.Int64Counter
	errorCounter        metric.Int64Counter
	searchDuration      metric.Float64Histogram
	confidenceHistogram metric.Float64Histogram

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

// WithEmbedder sets a custom embedder for the service.
// Required for GetMemoryVector to re-embed memory content.
func WithEmbedder(embedder vectorstore.Embedder) ServiceOption {
	return func(s *Service) {
		s.embedder = embedder
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

	s.logger.Info("initializing OTEL metrics",
		zap.String("instrumentation_scope", instrumentationName),
		zap.Bool("meter_is_noop", s.meter == nil),
	)

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

	s.searchCounter, err = s.meter.Int64Counter(
		"contextd.memory.searches_total",
		metric.WithDescription("Total number of memory searches"),
		metric.WithUnit("{search}"),
	)
	if err != nil {
		s.logger.Warn("failed to create search counter", zap.Error(err))
	}

	s.recordCounter, err = s.meter.Int64Counter(
		"contextd.memory.records_total",
		metric.WithDescription("Total number of memories recorded"),
		metric.WithUnit("{record}"),
	)
	if err != nil {
		s.logger.Warn("failed to create record counter", zap.Error(err))
	}

	s.feedbackCounter, err = s.meter.Int64Counter(
		"contextd.memory.feedbacks_total",
		metric.WithDescription("Total number of feedback events"),
		metric.WithUnit("{feedback}"),
	)
	if err != nil {
		s.logger.Warn("failed to create feedback counter", zap.Error(err))
	}

	s.outcomeCounter, err = s.meter.Int64Counter(
		"contextd.memory.outcomes_total",
		metric.WithDescription("Total number of outcome events"),
		metric.WithUnit("{outcome}"),
	)
	if err != nil {
		s.logger.Warn("failed to create outcome counter", zap.Error(err))
	}

	s.errorCounter, err = s.meter.Int64Counter(
		"contextd.memory.errors_total",
		metric.WithDescription("Total number of memory errors by operation"),
		metric.WithUnit("{error}"),
	)
	if err != nil {
		s.logger.Warn("failed to create error counter", zap.Error(err))
	}

	s.searchDuration, err = s.meter.Float64Histogram(
		"contextd.memory.search_duration_seconds",
		metric.WithDescription("Duration of memory search operations"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0),
	)
	if err != nil {
		s.logger.Warn("failed to create search duration histogram", zap.Error(err))
	}

	s.confidenceHistogram, err = s.meter.Float64Histogram(
		"contextd.memory.confidence",
		metric.WithDescription("Confidence scores of retrieved memories"),
		metric.WithUnit("1"),
		metric.WithExplicitBucketBoundaries(0.0, 0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1.0),
	)
	if err != nil {
		s.logger.Warn("failed to create confidence histogram", zap.Error(err))
	}

}

// recordError records an error metric with operation and reason labels.
func (s *Service) recordError(ctx context.Context, operation, reason string) {
	if s.errorCounter != nil {
		s.errorCounter.Add(ctx, 1, metric.WithAttributes(
			attribute.String("operation", operation),
			attribute.String("reason", reason),
		))
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
	startTime := time.Now()

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
		s.recordError(ctx, "search", "get_store_failed")
		return nil, err
	}

	// Use tenant context from caller if set (MCP tools set this)
	// Otherwise fall back to defaultTenant for backward compatibility
	if _, err := vectorstore.TenantFromContext(ctx); err != nil {
		tenantID := s.defaultTenant
		if tenantID == "" {
			s.recordError(ctx, "search", "tenant_not_configured")
			return nil, fmt.Errorf("tenant ID not configured for reasoningbank service")
		}
		ctx = vectorstore.ContextWithTenant(ctx, &vectorstore.TenantInfo{
			TenantID:  tenantID,
			ProjectID: projectID,
		})
	}

	// Check if collection exists
	exists, err := store.CollectionExists(ctx, collectionName)
	if err != nil {
		s.recordError(ctx, "search", "check_collection_failed")
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
		s.recordError(ctx, "search", "search_failed")
		return nil, fmt.Errorf("searching memories: %w", err)
	}

	// Convert results to Memory structs, filter by confidence, and record usage signals
	// We'll track both the memory and its score for re-ranking with consolidated memory boost
	type scoredMemory struct {
		memory Memory
		score  float32
	}
	scoredMemories := make([]scoredMemory, 0, len(results))
	seenIDs := make(map[string]struct{}, len(results)) // Deduplication: track seen memory IDs

	const consolidatedMemoryBoost = 1.2 // 20% boost for consolidated memories

	// Extract named entities from query for boosting (e.g., "Caroline" from "What is Caroline's identity?")
	queryEntities := s.extractQueryEntities(query)
	if len(queryEntities) > 0 {
		s.logger.Debug("extracted query entities",
			zap.Strings("entities", queryEntities),
			zap.String("query", query))
	}

	// Check if query is temporal (mentions "recent", "yesterday", etc.)
	// Temporal queries boost recent memories and penalize old ones
	isTemporalQuery := s.isTemporalQuery(query)
	if isTemporalQuery {
		s.logger.Debug("detected temporal query",
			zap.String("query", query))
	}

	for _, result := range results {
		// Deduplication: skip if we've already seen this memory ID
		// This prevents duplicates from race conditions during memory updates (delete→add pattern)
		if _, seen := seenIDs[result.ID]; seen {
			s.logger.Debug("skipping duplicate memory",
				zap.String("id", result.ID))
			continue
		}
		seenIDs[result.ID] = struct{}{}

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

		// Filter out archived memories (they were consolidated into other memories)
		if memory.State == MemoryStateArchived {
			s.logger.Debug("skipping archived memory",
				zap.String("id", memory.ID),
				zap.String("consolidation_id", func() string {
					if memory.ConsolidationID != nil {
						return *memory.ConsolidationID
					}
					return ""
				}()))
			continue
		}

		// Apply boost to consolidated memories (synthesized knowledge from multiple sources)
		score := result.Score
		isConsolidated := memory.ConsolidationID == nil && memory.State == MemoryStateActive &&
			(strings.Contains(memory.Description, "Synthesized from") ||
				strings.Contains(memory.Description, "Consolidated from"))
		if isConsolidated {
			score *= consolidatedMemoryBoost
			s.logger.Debug("applying consolidated memory boost",
				zap.String("id", memory.ID),
				zap.Float32("original_score", result.Score),
				zap.Float32("boosted_score", score))
		}

		// Apply boost when memory mentions entities extracted from the query
		// This improves precision for questions like "What is Caroline's identity?"
		if len(queryEntities) > 0 && s.memoryContainsEntity(memory, queryEntities) {
			score *= entityBoostFactor
			s.logger.Debug("applying entity boost",
				zap.String("id", memory.ID),
				zap.Strings("matched_entities", queryEntities),
				zap.Float32("boosted_score", score))
		}

		// Apply temporal weighting for time-sensitive queries
		// Recent memories get boosted, old memories get penalized
		if isTemporalQuery {
			temporalMultiplier := s.getTemporalMultiplier(memory)
			if temporalMultiplier != 1.0 {
				score *= temporalMultiplier
				s.logger.Debug("applying temporal weight",
					zap.String("id", memory.ID),
					zap.Float32("multiplier", temporalMultiplier),
					zap.Float32("boosted_score", score))
			}
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

		scoredMemories = append(scoredMemories, scoredMemory{
			memory: *memory,
			score:  score,
		})
	}

	// Re-sort by boosted scores (higher score = more relevant)
	sort.Slice(scoredMemories, func(i, j int) bool {
		return scoredMemories[i].score > scoredMemories[j].score
	})

	// Extract memories up to limit
	memories := make([]Memory, 0, limit)
	for i := 0; i < len(scoredMemories) && i < limit; i++ {
		memories = append(memories, scoredMemories[i].memory)
	}

	// Track last confidence for statusline (use first result's confidence)
	if len(memories) > 0 {
		s.statsMu.Lock()
		s.lastConfidence = memories[0].Confidence
		s.statsMu.Unlock()
	}

	// Record search metric
	if s.searchCounter != nil {
		s.searchCounter.Add(ctx, 1, metric.WithAttributes(
			attribute.String("project_id", projectID),
			attribute.Int("result_count", len(memories)),
		))
	}

	// Record search duration histogram
	if s.searchDuration != nil {
		duration := time.Since(startTime).Seconds()
		s.searchDuration.Record(ctx, duration, metric.WithAttributes(
			attribute.String("project_id", projectID),
		))
	}

	// Record confidence histogram for each returned memory
	if s.confidenceHistogram != nil {
		for _, mem := range memories {
			s.confidenceHistogram.Record(ctx, mem.Confidence, metric.WithAttributes(
				attribute.String("project_id", projectID),
			))
		}
	}

	s.logger.Debug("search completed",
		zap.String("project_id", projectID),
		zap.String("query", query),
		zap.Int("limit", limit),
		zap.Int("results", len(memories)))

	return memories, nil
}

// SearchWithScores returns memories with their search relevance scores.
// Unlike Search(), this preserves the semantic similarity score from the
// vector search, which is useful for displaying result quality to users.
//
// The Relevance score (0.0-1.0) indicates how well the memory matches
// the query semantically, distinct from the memory's Confidence which
// represents reliability based on feedback.
func (s *Service) SearchWithScores(ctx context.Context, projectID, query string, limit int) ([]ScoredMemory, error) {
	startTime := time.Now()

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
		s.recordError(ctx, "search", "get_store_failed")
		return nil, err
	}

	// Check if collection exists
	exists, err := store.CollectionExists(ctx, collectionName)
	if err != nil {
		s.recordError(ctx, "search", "collection_check_failed")
		return nil, fmt.Errorf("checking collection: %w", err)
	}
	if !exists {
		s.logger.Debug("collection does not exist, returning empty results",
			zap.String("collection", collectionName),
			zap.String("project_id", projectID))
		return []ScoredMemory{}, nil
	}

	// Fetch more results than requested to account for filtering
	searchLimit := limit * 3
	if searchLimit < 30 {
		searchLimit = 30
	}
	if searchLimit > 200 {
		searchLimit = 200
	}

	results, err := store.SearchInCollection(ctx, collectionName, query, searchLimit, nil)
	if err != nil {
		s.recordError(ctx, "search", "search_failed")
		return nil, fmt.Errorf("searching memories: %w", err)
	}

	// Convert results to ScoredMemory, applying dedup and filtering
	type internalScored struct {
		memory Memory
		score  float32
	}
	internalResults := make([]internalScored, 0, len(results))
	seenIDs := make(map[string]struct{}, len(results))

	const consolidatedMemoryBoost = 1.2

	// Extract named entities from query for boosting
	queryEntities := s.extractQueryEntities(query)

	// Check if query is temporal
	isTemporalQuery := s.isTemporalQuery(query)

	for _, result := range results {
		// Deduplication
		if _, seen := seenIDs[result.ID]; seen {
			continue
		}
		seenIDs[result.ID] = struct{}{}

		memory, err := s.resultToMemory(result)
		if err != nil {
			continue
		}

		// Filter low confidence and archived
		if memory.Confidence < MinConfidence {
			continue
		}
		if memory.State == MemoryStateArchived {
			continue
		}

		// Apply consolidated memory boost
		score := result.Score
		isConsolidated := memory.ConsolidationID == nil && memory.State == MemoryStateActive &&
			(strings.Contains(memory.Description, "Synthesized from") ||
				strings.Contains(memory.Description, "Consolidated from"))
		if isConsolidated {
			score *= consolidatedMemoryBoost
		}

		// Apply entity boost when memory mentions entities from query
		if len(queryEntities) > 0 && s.memoryContainsEntity(memory, queryEntities) {
			score *= entityBoostFactor
		}

		// Apply temporal weighting for time-sensitive queries
		if isTemporalQuery {
			score *= s.getTemporalMultiplier(memory)
		}

		// Record usage signal
		signal, err := NewSignal(memory.ID, projectID, SignalUsage, true, "")
		if err == nil {
			_ = s.signalStore.StoreSignal(ctx, signal)
		}

		internalResults = append(internalResults, internalScored{
			memory: *memory,
			score:  score,
		})
	}

	// Sort by score (descending)
	sort.Slice(internalResults, func(i, j int) bool {
		return internalResults[i].score > internalResults[j].score
	})

	// Convert to ScoredMemory and limit
	scoredMemories := make([]ScoredMemory, 0, limit)
	for i := 0; i < len(internalResults) && i < limit; i++ {
		scoredMemories = append(scoredMemories, ScoredMemory{
			Memory:    internalResults[i].memory,
			Relevance: float64(internalResults[i].score),
		})
	}

	// Record metrics
	if s.searchCounter != nil {
		s.searchCounter.Add(ctx, 1, metric.WithAttributes(
			attribute.String("project_id", projectID),
			attribute.Int("result_count", len(scoredMemories)),
		))
	}
	if s.searchDuration != nil {
		s.searchDuration.Record(ctx, time.Since(startTime).Seconds(), metric.WithAttributes(
			attribute.String("project_id", projectID),
		))
	}

	s.logger.Debug("search with scores completed",
		zap.String("project_id", projectID),
		zap.String("query", query),
		zap.Int("limit", limit),
		zap.Int("results", len(scoredMemories)))

	return scoredMemories, nil
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
		s.recordError(ctx, "record", "validation_failed")
		return fmt.Errorf("validating memory: %w", err)
	}

	// Get store and collection name
	store, collectionName, err := s.getStore(ctx, memory.ProjectID)
	if err != nil {
		s.recordError(ctx, "record", "get_store_failed")
		return err
	}

	// Use tenant context from caller if set (MCP tools set this)
	// Otherwise fall back to defaultTenant for backward compatibility
	if _, err := vectorstore.TenantFromContext(ctx); err != nil {
		// No tenant context set by caller, inject default
		tenantID := s.defaultTenant
		if tenantID == "" {
			s.recordError(ctx, "record", "tenant_not_configured")
			return fmt.Errorf("tenant ID not configured for reasoningbank service")
		}
		ctx = vectorstore.ContextWithTenant(ctx, &vectorstore.TenantInfo{
			TenantID:  tenantID,
			ProjectID: memory.ProjectID,
		})
	}
	// Note: If tenant context is already set, we respect it (don't overwrite)

	// Ensure collection exists
	exists, err := store.CollectionExists(ctx, collectionName)
	if err != nil {
		s.recordError(ctx, "record", "check_collection_failed")
		return fmt.Errorf("checking collection existence: %w", err)
	}
	if !exists {
		// Create collection with store's configured vector size (0 = use default)
		if err := store.CreateCollection(ctx, collectionName, 0); err != nil {
			s.recordError(ctx, "record", "create_collection_failed")
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
		s.recordError(ctx, "record", "store_failed")
		return fmt.Errorf("storing memory: %w", err)
	}

	// Record metric
	if s.recordCounter != nil {
		s.recordCounter.Add(ctx, 1, metric.WithAttributes(
			attribute.String("project_id", memory.ProjectID),
			attribute.String("outcome", string(memory.Outcome)),
		))
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
		s.recordError(ctx, "feedback", "get_memory_failed")
		return fmt.Errorf("getting memory: %w", err)
	}

	// Capture original state for potential rollback
	originalConfidence := memory.Confidence
	originalUpdatedAt := memory.UpdatedAt

	// Record explicit signal
	signal, err := NewSignal(memoryID, memory.ProjectID, SignalExplicit, helpful, "")
	if err != nil {
		s.recordError(ctx, "feedback", "create_signal_failed")
		return fmt.Errorf("creating signal: %w", err)
	}
	if err := s.signalStore.StoreSignal(ctx, signal); err != nil {
		s.recordError(ctx, "feedback", "store_signal_failed")
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
	} else {
		memory.Confidence = newConfidence
	}
	memory.UpdatedAt = time.Now()

	// Get store and collection name
	store, collectionName, err := s.getStore(ctx, memory.ProjectID)
	if err != nil {
		s.recordError(ctx, "feedback", "get_store_failed")
		return err
	}

	// Delete-then-add with rollback: delete old version, add updated version.
	// If add fails, attempt to restore the original document using originalConfidence
	// captured at the start of the function (line 929).
	if err := store.DeleteDocumentsFromCollection(ctx, collectionName, []string{memoryID}); err != nil {
		s.recordError(ctx, "feedback", "delete_old_failed")
		return fmt.Errorf("deleting old memory: %w", err)
	}

	// Re-add with updated confidence
	doc := s.memoryToDocument(memory, collectionName)
	_, err = store.AddDocuments(ctx, []vectorstore.Document{doc})
	if err != nil {
		// Attempt rollback: restore original document with original state
		memory.Confidence = originalConfidence
		memory.UpdatedAt = originalUpdatedAt
		rollbackDoc := s.memoryToDocument(memory, collectionName)
		_, rollbackErr := store.AddDocuments(ctx, []vectorstore.Document{rollbackDoc})
		if rollbackErr != nil {
			s.logger.Error("failed to rollback memory after update failure",
				zap.String("id", memoryID),
				zap.Error(rollbackErr))
		}
		s.recordError(ctx, "feedback", "update_failed")
		return fmt.Errorf("updating memory: %w", err)
	}

	// Record feedback metric
	if s.feedbackCounter != nil {
		helpfulStr := "negative"
		if helpful {
			helpfulStr = "positive"
		}
		s.feedbackCounter.Add(ctx, 1, metric.WithAttributes(
			attribute.String("project_id", memory.ProjectID),
			attribute.String("helpful", helpfulStr),
		))
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

	// Validate UUID format to prevent filter injection
	if _, err := uuid.Parse(id); err != nil {
		return nil, fmt.Errorf("invalid memory ID format: must be a valid UUID")
	}

	// With StoreProvider only, we can't enumerate all project stores
	if s.store == nil {
		return nil, fmt.Errorf("Get requires legacy store; use GetByProjectID with StoreProvider")
	}

	// Use tenant context from caller if set (MCP tools set this)
	// Otherwise fall back to defaultTenant for backward compatibility
	if _, err := vectorstore.TenantFromContext(ctx); err != nil {
		tenantID := s.defaultTenant
		if tenantID == "" {
			s.recordError(ctx, "get", "tenant_not_configured")
			return nil, fmt.Errorf("tenant ID not configured for reasoningbank service")
		}
		// Note: We can't inject ProjectID here since we don't know which project yet
		ctx = vectorstore.ContextWithTenant(ctx, &vectorstore.TenantInfo{
			TenantID: tenantID,
		})
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

	// Validate UUID format to prevent filter injection
	if _, err := uuid.Parse(memoryID); err != nil {
		return nil, fmt.Errorf("invalid memory ID format: must be a valid UUID")
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

	// Validate UUID format to prevent filter injection
	if _, err := uuid.Parse(memoryID); err != nil {
		return fmt.Errorf("invalid memory ID format: must be a valid UUID")
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
		s.recordError(ctx, "outcome", "get_memory_failed")
		return 0, fmt.Errorf("getting memory: %w", err)
	}

	// Capture original state for potential rollback
	originalConfidence := memory.Confidence
	originalUpdatedAt := memory.UpdatedAt

	// Create and store outcome signal
	signal, err := NewSignal(memoryID, memory.ProjectID, SignalOutcome, succeeded, sessionID)
	if err != nil {
		s.recordError(ctx, "outcome", "create_signal_failed")
		return 0, fmt.Errorf("creating signal: %w", err)
	}
	if err := s.signalStore.StoreSignal(ctx, signal); err != nil {
		s.recordError(ctx, "outcome", "store_signal_failed")
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
		// newConfidence not needed in fallback - memory.Confidence is already updated
	} else {
		memory.Confidence = newConfidence
	}
	memory.UpdatedAt = time.Now()

	// Get store and collection name
	store, collectionName, err := s.getStore(ctx, memory.ProjectID)
	if err != nil {
		s.recordError(ctx, "outcome", "get_store_failed")
		return 0, err
	}

	// Delete-then-add with rollback: delete old version, add updated version.
	// If add fails, attempt to restore the original document using originalConfidence
	// captured at the start of the function.
	if err := store.DeleteDocumentsFromCollection(ctx, collectionName, []string{memoryID}); err != nil {
		s.recordError(ctx, "outcome", "delete_old_failed")
		return 0, fmt.Errorf("deleting old memory: %w", err)
	}

	// Re-add with updated confidence
	doc := s.memoryToDocument(memory, collectionName)
	_, err = store.AddDocuments(ctx, []vectorstore.Document{doc})
	if err != nil {
		// Attempt rollback: restore original document with original state
		memory.Confidence = originalConfidence
		memory.UpdatedAt = originalUpdatedAt
		rollbackDoc := s.memoryToDocument(memory, collectionName)
		_, rollbackErr := store.AddDocuments(ctx, []vectorstore.Document{rollbackDoc})
		if rollbackErr != nil {
			s.logger.Error("failed to rollback memory after update failure",
				zap.String("id", memoryID),
				zap.Error(rollbackErr))
		}
		s.recordError(ctx, "outcome", "update_failed")
		return 0, fmt.Errorf("updating memory: %w", err)
	}

	// Record outcome metric
	if s.outcomeCounter != nil {
		successStr := "failure"
		if succeeded {
			successStr = "success"
		}
		s.outcomeCounter.Add(ctx, 1, metric.WithAttributes(
			attribute.String("project_id", memory.ProjectID),
			attribute.String("outcome", successStr),
		))
	}

	s.logger.Info("outcome recorded",
		zap.String("id", memoryID),
		zap.String("signal_id", signal.ID),
		zap.Bool("succeeded", succeeded),
		zap.Float64("new_confidence", memory.Confidence))

	return memory.Confidence, nil
}

// extractQueryEntities extracts named entities (proper nouns) from a query.
// Returns a slice of unique capitalized words found in the query, excluding
// common stopwords (What, When, Where, etc.).
//
// Example: "What is Caroline's identity?" → ["Caroline"]
//
//	"Tell me about John and Alice" → ["John", "Alice"]
//
// Input is truncated to maxQueryLength to prevent ReDoS attacks.
func (s *Service) extractQueryEntities(query string) []string {
	// Limit input length to prevent ReDoS
	if len(query) > maxQueryLength {
		query = query[:maxQueryLength]
	}

	matches := entityRegex.FindAllString(query, -1)
	if len(matches) == 0 {
		return nil
	}

	// Deduplicate entities and filter stopwords
	seen := make(map[string]struct{}, len(matches))
	entities := make([]string, 0, len(matches))
	for _, match := range matches {
		lower := strings.ToLower(match)
		// Skip stopwords (common question/verb words that aren't entities)
		if _, isStopword := entityStopwords[lower]; isStopword {
			continue
		}
		if _, ok := seen[lower]; !ok {
			seen[lower] = struct{}{}
			entities = append(entities, match)
		}
	}
	if len(entities) == 0 {
		return nil
	}
	return entities
}

// memoryContainsEntity checks if a memory's content mentions any of the given entities.
// Matching is case-insensitive.
func (s *Service) memoryContainsEntity(memory *Memory, entities []string) bool {
	if len(entities) == 0 || memory == nil {
		return false
	}

	// Combine searchable fields
	searchText := strings.ToLower(memory.Title + " " + memory.Content + " " + memory.Description)

	for _, entity := range entities {
		if strings.Contains(searchText, strings.ToLower(entity)) {
			return true
		}
	}
	return false
}

// extractConversationID extracts a conversation ID from memory tags.
// Returns empty string if no conversation ID is found.
//
// Example tags: ["locomo", "conv-26", "Melanie", "turn_329", "session_15"]
// Returns: "conv-26"
func (s *Service) extractConversationID(memory *Memory) string {
	if memory == nil {
		return ""
	}

	// Check tags first (most common location)
	for _, tag := range memory.Tags {
		if match := conversationIDRegex.FindString(tag); match != "" {
			return match
		}
	}

	// Also check title (tags might be embedded there)
	if match := conversationIDRegex.FindString(memory.Title); match != "" {
		return match
	}

	return ""
}

// isTemporalQuery checks if a query contains keywords indicating time-sensitivity.
// Temporal queries benefit from recency boosting (recent memories rank higher).
// Input is truncated to maxQueryLength for safety.
func (s *Service) isTemporalQuery(query string) bool {
	// Limit input length for safety
	if len(query) > maxQueryLength {
		query = query[:maxQueryLength]
	}

	lowerQuery := strings.ToLower(query)
	for _, keyword := range temporalKeywords {
		if strings.Contains(lowerQuery, keyword) {
			return true
		}
	}
	return false
}

// getTemporalMultiplier returns a score multiplier based on memory age.
// Recent memories (<7 days) get boosted, old memories (>30 days) get penalized.
// Only applied for temporal queries (detected via isTemporalQuery).
func (s *Service) getTemporalMultiplier(memory *Memory) float32 {
	if memory == nil {
		return 1.0
	}

	age := time.Since(memory.UpdatedAt)
	switch {
	case age < 7*24*time.Hour:
		// Recent: boost
		return temporalRecentBoost
	case age < 30*24*time.Hour:
		// Medium: no change
		return temporalMediumMultiplier
	default:
		// Old: penalty
		return temporalOldPenalty
	}
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
		"state":       string(memory.State),
		"created_at":  memory.CreatedAt.Unix(),
		"updated_at":  memory.UpdatedAt.Unix(),
	}

	// Include consolidation_id if set (for source memories that were consolidated)
	if memory.ConsolidationID != nil {
		metadata["consolidation_id"] = *memory.ConsolidationID
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

// ListMemories retrieves all memories for a project with pagination support.
//
// This method is used by the memory consolidation system to iterate over all memories
// in a project. Unlike Search, it doesn't filter by semantic similarity - it returns
// memories in storage order.
//
// Parameters:
//   - limit: Maximum number of memories to return (0 = return all)
//   - offset: Number of memories to skip (for pagination)
//
// Returns memories in storage order. For large projects, use pagination to avoid
// loading all memories at once.
func (s *Service) ListMemories(ctx context.Context, projectID string, limit, offset int) ([]Memory, error) {
	if projectID == "" {
		return nil, ErrEmptyProjectID
	}
	if limit < 0 {
		return nil, fmt.Errorf("limit cannot be negative")
	}
	if offset < 0 {
		return nil, fmt.Errorf("offset cannot be negative")
	}

	// Get store and collection name for this project
	store, collectionName, err := s.getStore(ctx, projectID)
	if err != nil {
		return nil, err
	}

	// Use tenant context from caller if set (MCP tools set this)
	// Otherwise fall back to defaultTenant for backward compatibility
	if _, err := vectorstore.TenantFromContext(ctx); err != nil {
		tenantID := s.defaultTenant
		if tenantID == "" {
			return nil, fmt.Errorf("tenant ID not configured for reasoningbank service")
		}
		ctx = vectorstore.ContextWithTenant(ctx, &vectorstore.TenantInfo{
			TenantID:  tenantID,
			ProjectID: projectID,
		})
	}

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

	// Calculate fetch limit: need offset + limit documents
	// Use a high limit if limit=0 (return all)
	fetchLimit := limit + offset
	if limit == 0 {
		// Fetch all - use a very high limit
		// Most projects won't have more than 10k memories
		fetchLimit = 10000
	}
	if fetchLimit > 10000 {
		fetchLimit = 10000 // Cap to prevent excessive fetching
	}

	// Use SearchInCollection with an empty query to get all documents
	// The vectorstore will return results in storage order
	results, err := store.SearchInCollection(ctx, collectionName, "", fetchLimit, nil)
	if err != nil {
		return nil, fmt.Errorf("listing memories: %w", err)
	}

	// Skip offset documents and take up to limit
	start := offset
	if start > len(results) {
		return []Memory{}, nil
	}

	end := len(results)
	if limit > 0 && start+limit < len(results) {
		end = start + limit
	}

	// Convert results to Memory structs
	memories := make([]Memory, 0, end-start)
	for i := start; i < end; i++ {
		memory, err := s.resultToMemory(results[i])
		if err != nil {
			s.logger.Warn("skipping invalid memory",
				zap.String("id", results[i].ID),
				zap.Error(err))
			continue
		}
		memories = append(memories, *memory)
	}

	s.logger.Debug("list memories completed",
		zap.String("project_id", projectID),
		zap.Int("limit", limit),
		zap.Int("offset", offset),
		zap.Int("results", len(memories)))

	return memories, nil
}

// GetMemoryVector retrieves the embedding vector for a memory by ID.
//
// This method re-embeds the memory content to retrieve its vector representation.
// The content is embedded the same way as during storage (title + content).
//
// Note: This method requires the legacy single-store configuration.
// When using StoreProvider (database-per-project), use GetMemoryVectorByProjectID instead.
//
// Returns the embedding vector or an error if the memory doesn't exist or embedder is not configured.
func (s *Service) GetMemoryVector(ctx context.Context, memoryID string) ([]float32, error) {
	if memoryID == "" {
		return nil, fmt.Errorf("memory ID cannot be empty")
	}
	if s.embedder == nil {
		return nil, fmt.Errorf("embedder not configured for reasoningbank service")
	}

	// Get the memory first
	memory, err := s.Get(ctx, memoryID)
	if err != nil {
		return nil, fmt.Errorf("getting memory: %w", err)
	}

	// Re-embed the content (same format as when storing: title + content)
	content := fmt.Sprintf("%s\n\n%s", memory.Title, memory.Content)
	vector, err := s.embedder.EmbedQuery(ctx, content)
	if err != nil {
		return nil, fmt.Errorf("embedding memory content: %w", err)
	}

	s.logger.Debug("retrieved memory vector",
		zap.String("memory_id", memoryID),
		zap.String("project_id", memory.ProjectID),
		zap.Int("vector_size", len(vector)))

	return vector, nil
}

// GetMemoryVectorByProjectID retrieves the embedding vector for a memory within a specific project.
//
// This is the preferred method when using StoreProvider (database-per-project isolation)
// as it directly accesses the project's store without enumeration.
//
// The method re-embeds the memory content to retrieve its vector representation.
// The content is embedded the same way as during storage (title + content).
//
// Returns the embedding vector or an error if the memory doesn't exist or embedder is not configured.
func (s *Service) GetMemoryVectorByProjectID(ctx context.Context, projectID, memoryID string) ([]float32, error) {
	if projectID == "" {
		return nil, ErrEmptyProjectID
	}
	if memoryID == "" {
		return nil, fmt.Errorf("memory ID cannot be empty")
	}
	if s.embedder == nil {
		return nil, fmt.Errorf("embedder not configured for reasoningbank service")
	}

	// Get the memory first
	memory, err := s.GetByProjectID(ctx, projectID, memoryID)
	if err != nil {
		return nil, fmt.Errorf("getting memory: %w", err)
	}

	// Re-embed the content (same format as when storing: title + content)
	content := fmt.Sprintf("%s\n\n%s", memory.Title, memory.Content)
	vector, err := s.embedder.EmbedQuery(ctx, content)
	if err != nil {
		return nil, fmt.Errorf("embedding memory content: %w", err)
	}

	s.logger.Debug("retrieved memory vector",
		zap.String("memory_id", memoryID),
		zap.String("project_id", projectID),
		zap.Int("vector_size", len(vector)))

	return vector, nil
}

// parseFloat64 extracts a float64 from metadata, handling both float64 and string types.
// chromem-go stores metadata as JSON and may deserialize numbers as strings.
func parseFloat64(v interface{}) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case float32:
		return float64(val)
	case int:
		return float64(val)
	case int64:
		return float64(val)
	case string:
		f, _ := strconv.ParseFloat(val, 64)
		return f
	default:
		return 0
	}
}

// parseInt64 extracts an int64 from metadata, handling both numeric and string types.
func parseInt64(v interface{}) int64 {
	switch val := v.(type) {
	case int64:
		return val
	case int:
		return int64(val)
	case float64:
		return int64(val)
	case string:
		i, _ := strconv.ParseInt(val, 10, 64)
		return i
	default:
		return 0
	}
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
	confidence := parseFloat64(result.Metadata["confidence"])
	usageCount := int(parseInt64(result.Metadata["usage_count"]))

	// Parse tags - handle both []string (in-memory) and []interface{} (JSON deserialized)
	tags := []string{}
	if tagsIface, ok := result.Metadata["tags"]; ok {
		switch tagsList := tagsIface.(type) {
		case []string:
			tags = tagsList
		case []interface{}:
			for _, t := range tagsList {
				if tag, ok := t.(string); ok {
					tags = append(tags, tag)
				}
			}
		}
	}

	// Parse timestamps (handle both int64 and string from chromem)
	createdAtUnix := parseInt64(result.Metadata["created_at"])
	updatedAtUnix := parseInt64(result.Metadata["updated_at"])

	createdAt := time.Unix(createdAtUnix, 0)
	updatedAt := time.Unix(updatedAtUnix, 0)

	// Parse state (default to Active for backwards compatibility with existing memories)
	stateStr, _ := result.Metadata["state"].(string)
	state := MemoryStateActive
	if stateStr == string(MemoryStateArchived) {
		state = MemoryStateArchived
	}

	// Parse consolidation_id if present
	var consolidationID *string
	if consolidationIDStr, ok := result.Metadata["consolidation_id"].(string); ok && consolidationIDStr != "" {
		consolidationID = &consolidationIDStr
	}

	// Parse content (strip title from beginning if present)
	content := result.Content
	titlePrefix := title + "\n\n"
	if len(title) > 0 && len(content) >= len(titlePrefix) && strings.HasPrefix(content, titlePrefix) {
		// Remove "title\n\n" prefix
		content = content[len(titlePrefix):]
	}

	memory := &Memory{
		ID:              id,
		ProjectID:       projectID,
		Title:           title,
		Description:     description,
		Content:         content,
		Outcome:         Outcome(outcomeStr),
		Confidence:      confidence,
		UsageCount:      usageCount,
		Tags:            tags,
		ConsolidationID: consolidationID,
		State:           state,
		CreatedAt:       createdAt,
		UpdatedAt:       updatedAt,
	}

	return memory, nil
}
