package reasoningbank

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/fyrsmithlabs/contextd/internal/project"
	"github.com/fyrsmithlabs/contextd/internal/vectorstore"
	"go.uber.org/zap"
)

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
	store       vectorstore.Store
	signalStore SignalStore
	confCalc    *ConfidenceCalculator
	logger      *zap.Logger
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

	return svc, nil
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

	// Get collection name for this project's memories
	collectionName, err := project.GetCollectionName(projectID, project.CollectionMemories)
	if err != nil {
		return nil, fmt.Errorf("getting collection name: %w", err)
	}

	// Check if collection exists
	exists, err := s.store.CollectionExists(ctx, collectionName)
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

	// Search with confidence filter
	filters := map[string]interface{}{
		"confidence": map[string]interface{}{
			"$gte": MinConfidence,
		},
	}

	results, err := s.store.SearchInCollection(ctx, collectionName, query, limit, filters)
	if err != nil {
		return nil, fmt.Errorf("searching memories: %w", err)
	}

	// Convert results to Memory structs and record usage signals
	memories := make([]Memory, 0, len(results))
	for _, result := range results {
		memory, err := s.resultToMemory(result)
		if err != nil {
			s.logger.Warn("skipping invalid memory",
				zap.String("id", result.ID),
				zap.Error(err))
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

	// Get collection name
	collectionName, err := project.GetCollectionName(memory.ProjectID, project.CollectionMemories)
	if err != nil {
		return fmt.Errorf("getting collection name: %w", err)
	}

	// Ensure collection exists
	exists, err := s.store.CollectionExists(ctx, collectionName)
	if err != nil {
		return fmt.Errorf("checking collection existence: %w", err)
	}
	if !exists {
		// Create collection with store's configured vector size (0 = use default)
		if err := s.store.CreateCollection(ctx, collectionName, 0); err != nil {
			return fmt.Errorf("creating collection: %w", err)
		}
		s.logger.Info("created memories collection",
			zap.String("collection", collectionName),
			zap.String("project_id", memory.ProjectID))
	}

	// Convert to document
	doc := s.memoryToDocument(memory, collectionName)

	// Store in vector store
	_, err = s.store.AddDocuments(ctx, []vectorstore.Document{doc})
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

	// Update in vector store (delete and re-add with new confidence)
	collectionName, err := project.GetCollectionName(memory.ProjectID, project.CollectionMemories)
	if err != nil {
		return fmt.Errorf("getting collection name: %w", err)
	}

	// Delete old version from the correct collection
	if err := s.store.DeleteDocumentsFromCollection(ctx, collectionName, []string{memoryID}); err != nil {
		return fmt.Errorf("deleting old memory: %w", err)
	}

	// Re-add with updated confidence
	doc := s.memoryToDocument(memory, collectionName)
	_, err = s.store.AddDocuments(ctx, []vectorstore.Document{doc})
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
func (s *Service) Get(ctx context.Context, id string) (*Memory, error) {
	if id == "" {
		return nil, fmt.Errorf("memory ID cannot be empty")
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

// Delete removes a memory by ID.
func (s *Service) Delete(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("memory ID cannot be empty")
	}

	// Get the memory first to know which collection it's in
	memory, err := s.Get(ctx, id)
	if err != nil {
		return fmt.Errorf("getting memory: %w", err)
	}

	// Delete from vector store
	if err := s.store.DeleteDocuments(ctx, []string{id}); err != nil {
		return fmt.Errorf("deleting memory: %w", err)
	}

	s.logger.Info("memory deleted",
		zap.String("id", id),
		zap.String("project_id", memory.ProjectID))

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

	// Update in vector store
	collectionName, err := project.GetCollectionName(memory.ProjectID, project.CollectionMemories)
	if err != nil {
		return 0, fmt.Errorf("getting collection name: %w", err)
	}

	// Delete old version and re-add with updated confidence
	if err := s.store.DeleteDocumentsFromCollection(ctx, collectionName, []string{memoryID}); err != nil {
		return 0, fmt.Errorf("deleting old memory: %w", err)
	}

	doc := s.memoryToDocument(memory, collectionName)
	_, err = s.store.AddDocuments(ctx, []vectorstore.Document{doc})
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
