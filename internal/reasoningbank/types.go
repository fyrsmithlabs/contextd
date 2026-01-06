package reasoningbank

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

// Common errors for ReasoningBank operations.
var (
	ErrMemoryNotFound      = errors.New("memory not found")
	ErrInvalidMemory       = errors.New("invalid memory")
	ErrEmptyTitle          = errors.New("memory title cannot be empty")
	ErrEmptyContent        = errors.New("memory content cannot be empty")
	ErrInvalidConfidence   = errors.New("confidence must be between 0.0 and 1.0")
	ErrInvalidOutcome      = errors.New("outcome must be 'success' or 'failure'")
	ErrEmptyProjectID      = errors.New("project ID cannot be empty")
)

// Outcome represents the result type of a memory.
type Outcome string

const (
	// OutcomeSuccess indicates a successful strategy or pattern.
	OutcomeSuccess Outcome = "success"

	// OutcomeFailure indicates an anti-pattern or failed approach.
	OutcomeFailure Outcome = "failure"
)

// MemoryState represents the lifecycle state of a memory.
type MemoryState string

const (
	// MemoryStateActive indicates the memory is actively used in searches.
	MemoryStateActive MemoryState = "active"

	// MemoryStateArchived indicates the memory has been consolidated into another memory.
	// Archived memories are preserved for attribution but excluded from normal searches.
	MemoryStateArchived MemoryState = "archived"
)

// Memory represents a cross-session memory in the ReasoningBank.
//
// Memories are distilled strategies learned from agent interactions.
// They can represent successful patterns (outcome="success") or
// anti-patterns to avoid (outcome="failure").
//
// Confidence is tracked and adjusted based on feedback signals:
//   - Explicit ratings from users
//   - Implicit success (memory helped solve a task)
//   - Code stability (solution didn't need rework)
type Memory struct {
	// ID is the unique memory identifier (UUID).
	ID string `json:"id"`

	// ProjectID identifies which project this memory belongs to.
	ProjectID string `json:"project_id"`

	// Title is a brief summary of the memory (e.g., "Go error handling with context").
	Title string `json:"title"`

	// Description provides additional context about when/why this memory is useful.
	Description string `json:"description,omitempty"`

	// Content is the main memory content (strategy, anti-pattern, code example).
	Content string `json:"content"`

	// Outcome indicates if this is a success pattern or failure anti-pattern.
	Outcome Outcome `json:"outcome"`

	// Confidence is a score from 0.0 to 1.0 indicating reliability.
	// Higher confidence memories are prioritized in search results.
	// Adjusted based on feedback and usage patterns.
	Confidence float64 `json:"confidence"`

	// UsageCount tracks how many times this memory has been retrieved.
	UsageCount int `json:"usage_count"`

	// Tags are labels for categorization (e.g., "go", "error-handling", "auth").
	Tags []string `json:"tags,omitempty"`

	// ConsolidationID links this memory to a consolidated memory it was merged into.
	// When a memory is consolidated with others, this field is set to the ID of the
	// resulting ConsolidatedMemory. The original memory is preserved for attribution.
	ConsolidationID *string `json:"consolidation_id,omitempty"`

	// State indicates the lifecycle state of this memory (active or archived).
	// Archived memories have been consolidated into other memories but are preserved
	// for attribution and traceability. They are excluded from normal searches.
	State MemoryState `json:"state"`

	// CreatedAt is when the memory was created.
	CreatedAt time.Time `json:"created_at"`

	// UpdatedAt is when the memory was last modified.
	UpdatedAt time.Time `json:"updated_at"`
}

// NewMemory creates a new memory with a generated UUID and default values.
func NewMemory(projectID, title, content string, outcome Outcome, tags []string) (*Memory, error) {
	if projectID == "" {
		return nil, ErrEmptyProjectID
	}
	if title == "" {
		return nil, ErrEmptyTitle
	}
	if content == "" {
		return nil, ErrEmptyContent
	}
	if outcome != OutcomeSuccess && outcome != OutcomeFailure {
		return nil, ErrInvalidOutcome
	}

	now := time.Now()
	return &Memory{
		ID:         uuid.New().String(),
		ProjectID:  projectID,
		Title:      title,
		Content:    content,
		Outcome:    outcome,
		Confidence: 0.5, // Default confidence (neutral)
		UsageCount: 0,
		Tags:       tags,
		State:      MemoryStateActive, // New memories are active by default
		CreatedAt:  now,
		UpdatedAt:  now,
	}, nil
}

// Validate checks if the memory has valid fields.
func (m *Memory) Validate() error {
	if m.ID == "" {
		return errors.New("memory ID cannot be empty")
	}
	if _, err := uuid.Parse(m.ID); err != nil {
		return errors.New("invalid memory ID format")
	}
	if m.ProjectID == "" {
		return ErrEmptyProjectID
	}
	if m.Title == "" {
		return ErrEmptyTitle
	}
	if m.Content == "" {
		return ErrEmptyContent
	}
	if m.Outcome != OutcomeSuccess && m.Outcome != OutcomeFailure {
		return ErrInvalidOutcome
	}
	if m.Confidence < 0.0 || m.Confidence > 1.0 {
		return ErrInvalidConfidence
	}
	if m.UsageCount < 0 {
		return errors.New("usage count cannot be negative")
	}
	if m.State != MemoryStateActive && m.State != MemoryStateArchived {
		return errors.New("state must be 'active' or 'archived'")
	}
	return nil
}

// AdjustConfidence updates the confidence based on feedback.
//
// For helpful feedback:
//   - Increases confidence by up to 0.1 (capped at 1.0)
//
// For unhelpful feedback:
//   - Decreases confidence by up to 0.15 (floored at 0.0)
func (m *Memory) AdjustConfidence(helpful bool) {
	if helpful {
		m.Confidence += 0.1
		if m.Confidence > 1.0 {
			m.Confidence = 1.0
		}
	} else {
		m.Confidence -= 0.15
		if m.Confidence < 0.0 {
			m.Confidence = 0.0
		}
	}
	m.UpdatedAt = time.Now()
}

// IncrementUsage increments the usage count and updates timestamp.
func (m *Memory) IncrementUsage() {
	m.UsageCount++
	m.UpdatedAt = time.Now()
}

// ConsolidationType represents the method used to create a consolidated memory.
type ConsolidationType string

const (
	// ConsolidationMerged indicates memories were merged into a single synthesized memory.
	ConsolidationMerged ConsolidationType = "merged"

	// ConsolidationDeduplicated indicates duplicate or near-duplicate memories were combined.
	ConsolidationDeduplicated ConsolidationType = "deduplicated"

	// ConsolidationSynthesized indicates memories were synthesized into higher-level knowledge.
	ConsolidationSynthesized ConsolidationType = "synthesized"
)

// ConsolidatedMemory represents a memory created by consolidating multiple source memories.
//
// ConsolidatedMemories are created by the Distiller when it detects similar or related
// memories that can be merged into more valuable synthesized knowledge. The original
// source memories are preserved with their ConsolidationID field pointing to this
// consolidated memory.
type ConsolidatedMemory struct {
	// Memory is the consolidated memory record.
	*Memory

	// SourceIDs contains the IDs of all source memories that were consolidated.
	SourceIDs []string `json:"source_ids"`

	// ConsolidationType indicates the method used for consolidation.
	ConsolidationType ConsolidationType `json:"consolidation_type"`

	// SourceAttribution provides context about how the source memories contributed.
	// This is a human-readable description generated by the LLM during synthesis.
	SourceAttribution string `json:"source_attribution,omitempty"`
}

// SimilarityCluster represents a group of similar memories detected during consolidation.
//
// The Distiller uses vector similarity search to find clusters of related memories
// that can be merged. Each cluster contains memories above a similarity threshold
// and statistics about their relationships.
type SimilarityCluster struct {
	// Members contains all memories in this similarity cluster.
	Members []*Memory `json:"members"`

	// CentroidVector is the average embedding vector of all cluster members.
	// Used to represent the cluster's semantic center.
	CentroidVector []float32 `json:"centroid_vector,omitempty"`

	// AverageSimilarity is the mean pairwise similarity score between cluster members.
	// Range: 0.0 to 1.0, where 1.0 means all members are identical.
	AverageSimilarity float64 `json:"average_similarity"`

	// MinSimilarity is the lowest pairwise similarity score in the cluster.
	// Indicates the cluster's cohesion - higher values mean tighter clustering.
	MinSimilarity float64 `json:"min_similarity"`
}

// ConsolidationResult contains the results of a memory consolidation operation.
//
// This structure tracks the outcome of running memory consolidation, including
// which memories were created (consolidated memories), which were archived
// (source memories linked to consolidated versions), how many were skipped
// (didn't meet consolidation criteria), and performance metrics.
type ConsolidationResult struct {
	// CreatedMemories contains the IDs of newly created consolidated memories.
	CreatedMemories []string `json:"created_memories"`

	// ArchivedMemories contains the IDs of source memories that were archived
	// after being consolidated into new memories. These memories are preserved
	// with their ConsolidationID field pointing to the consolidated memory.
	ArchivedMemories []string `json:"archived_memories"`

	// SkippedCount is the number of memories that were evaluated but not
	// consolidated (e.g., no similar memories found, below threshold).
	SkippedCount int `json:"skipped_count"`

	// TotalProcessed is the total number of memories examined during consolidation.
	TotalProcessed int `json:"total_processed"`

	// Duration is how long the consolidation operation took to complete.
	Duration time.Duration `json:"duration"`
}

// MemoryConsolidator defines the interface for memory consolidation operations.
//
// Implementations of this interface (such as the Distiller) are responsible for
// detecting similar memories, merging them into consolidated entries, and
// orchestrating the overall consolidation process.
//
// The consolidation workflow:
//  1. FindSimilarClusters detects groups of similar memories above a threshold
//  2. MergeCluster synthesizes each cluster into a single consolidated memory
//  3. Consolidate orchestrates the full process with configurable options
//
// Original memories are preserved with back-links to their consolidated versions
// via the ConsolidationID field.
type MemoryConsolidator interface {
	// FindSimilarClusters detects groups of similar memories for a project.
	//
	// Searches all memories in the project and groups those with similarity
	// scores above the threshold. Uses greedy clustering: for each memory,
	// finds all similar memories above threshold, forms cluster if >=2 members.
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeouts
	//   - projectID: Project to search for similar memories
	//   - threshold: Minimum similarity score (0.0-1.0, typically 0.8)
	//
	// Returns:
	//   - Slice of similarity clusters, each containing related memories
	//   - Error if clustering fails
	FindSimilarClusters(ctx context.Context, projectID string, threshold float64) ([]SimilarityCluster, error)

	// MergeCluster synthesizes a cluster of similar memories into one consolidated memory.
	//
	// Uses an LLM to analyze the cluster members and create a synthesized memory
	// that captures their common themes and key insights. The consolidated memory
	// includes source attribution and links back to the original memories.
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeouts
	//   - cluster: Similarity cluster to merge
	//
	// Returns:
	//   - The newly created consolidated memory
	//   - Error if synthesis or storage fails
	MergeCluster(ctx context.Context, cluster *SimilarityCluster) (*Memory, error)

	// Consolidate runs the full memory consolidation process for a project.
	//
	// Orchestrates the complete workflow:
	//  1. Find all similarity clusters above threshold
	//  2. Merge each cluster into a consolidated memory
	//  3. Link source memories to their consolidated versions
	//  4. Return statistics about the consolidation run
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeouts
	//   - projectID: Project to consolidate memories for
	//   - opts: Configuration options (threshold, limits, dry-run mode, etc.)
	//
	// Returns:
	//   - ConsolidationResult with statistics and outcomes
	//   - Error if consolidation fails
	Consolidate(ctx context.Context, projectID string, opts interface{}) (*ConsolidationResult, error)
}
