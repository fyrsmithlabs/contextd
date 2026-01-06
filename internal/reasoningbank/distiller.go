package reasoningbank

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"go.uber.org/zap"
)

// SessionOutcome represents the overall outcome of a session.
type SessionOutcome string

const (
	// SessionSuccess indicates the session achieved its goal.
	SessionSuccess SessionOutcome = "success"

	// SessionFailure indicates the session did not achieve its goal.
	SessionFailure SessionOutcome = "failure"

	// SessionPartial indicates partial success or mixed results.
	SessionPartial SessionOutcome = "partial"
)

// SessionSummary contains distilled information from a completed session.
type SessionSummary struct {
	// SessionID uniquely identifies the session.
	SessionID string

	// ProjectID identifies the project this session belongs to.
	ProjectID string

	// Outcome is the overall session result.
	Outcome SessionOutcome

	// Task is a brief description of what the session was trying to accomplish.
	Task string

	// Approach is the strategy or method used (extracted from session).
	Approach string

	// Result describes what happened (success details or failure reasons).
	Result string

	// Tags are labels for categorization (language, domain, problem type).
	Tags []string

	// Duration is how long the session lasted.
	Duration time.Duration

	// CompletedAt is when the session ended.
	CompletedAt time.Time
}

// LLMClient provides an interface for interacting with LLM backends.
//
// This interface allows pluggable LLM providers (Claude, OpenAI, local models)
// to be used for memory synthesis and consolidation tasks. Implementations
// should handle retries, rate limiting, and error handling internally.
type LLMClient interface {
	// Complete generates a completion from the given prompt.
	//
	// The context can be used for cancellation and deadline control.
	// Returns the generated text or an error if the request fails.
	Complete(ctx context.Context, prompt string) (string, error)
}

// Distiller extracts learnings from completed sessions and creates memories.
//
// FR-006: Distillation pipeline for async memory extraction
// FR-009: Outcome differentiation (success vs failure)
type Distiller struct {
	service   *Service
	logger    *zap.Logger
	llmClient LLMClient // Optional LLM client for memory consolidation
}

// DistillerOption configures a Distiller.
type DistillerOption func(*Distiller)

// WithLLMClient sets the LLM client for memory consolidation.
// This is required for MergeCluster to work.
func WithLLMClient(client LLMClient) DistillerOption {
	return func(d *Distiller) {
		d.llmClient = client
	}
}

// NewDistiller creates a new session distiller.
func NewDistiller(service *Service, logger *zap.Logger, opts ...DistillerOption) (*Distiller, error) {
	if service == nil {
		return nil, fmt.Errorf("service cannot be nil")
	}
	if logger == nil {
		return nil, fmt.Errorf("logger cannot be nil")
	}

	d := &Distiller{
		service: service,
		logger:  logger,
	}

	// Apply options
	for _, opt := range opts {
		opt(d)
	}

	return d, nil
}

// DistillSession extracts learnings from a completed session and creates memories.
//
// This is called asynchronously after a session ends, so it should not block.
//
// Success patterns (outcome="success") become positive memories.
// Failure patterns (outcome="failure") become anti-pattern warnings.
//
// Initial confidence is set to DistilledConfidence (0.6) since distilled
// memories are less reliable than explicit captures (0.8).
func (d *Distiller) DistillSession(ctx context.Context, summary SessionSummary) error {
	if summary.ProjectID == "" {
		return fmt.Errorf("project ID cannot be empty")
	}
	if summary.SessionID == "" {
		return fmt.Errorf("session ID cannot be empty")
	}

	d.logger.Info("distilling session",
		zap.String("session_id", summary.SessionID),
		zap.String("project_id", summary.ProjectID),
		zap.String("outcome", string(summary.Outcome)))

	// Extract memories based on outcome
	var memories []*Memory
	var err error

	switch summary.Outcome {
	case SessionSuccess:
		memories, err = d.extractSuccessPatterns(summary)
	case SessionFailure:
		memories, err = d.extractFailurePatterns(summary)
	case SessionPartial:
		// For partial outcomes, extract both success and failure patterns
		successMems, err1 := d.extractSuccessPatterns(summary)
		failureMems, err2 := d.extractFailurePatterns(summary)
		if err1 != nil {
			d.logger.Warn("error extracting success patterns from partial session",
				zap.Error(err1))
		}
		if err2 != nil {
			d.logger.Warn("error extracting failure patterns from partial session",
				zap.Error(err2))
		}
		memories = append(successMems, failureMems...)
	default:
		return fmt.Errorf("unknown session outcome: %s", summary.Outcome)
	}

	if err != nil {
		return fmt.Errorf("extracting patterns: %w", err)
	}

	// Record extracted memories
	for _, memory := range memories {
		if err := d.service.Record(ctx, memory); err != nil {
			d.logger.Error("failed to record distilled memory",
				zap.String("session_id", summary.SessionID),
				zap.String("memory_title", memory.Title),
				zap.Error(err))
			// Continue with other memories even if one fails
		} else {
			d.logger.Info("distilled memory recorded",
				zap.String("session_id", summary.SessionID),
				zap.String("memory_id", memory.ID),
				zap.String("title", memory.Title))
		}
	}

	d.logger.Info("session distillation completed",
		zap.String("session_id", summary.SessionID),
		zap.Int("memories_extracted", len(memories)))

	return nil
}

// extractSuccessPatterns creates memories from successful sessions.
//
// Success patterns become positive guidance for future sessions.
func (d *Distiller) extractSuccessPatterns(summary SessionSummary) ([]*Memory, error) {
	// Create a success pattern memory
	title := d.generateTitle(summary.Task, "Success")
	content := d.formatSuccessContent(summary)

	memory, err := NewMemory(
		summary.ProjectID,
		title,
		content,
		OutcomeSuccess,
		summary.Tags,
	)
	if err != nil {
		return nil, fmt.Errorf("creating success memory: %w", err)
	}

	// Set distilled confidence
	memory.Confidence = DistilledConfidence

	// Add session metadata to description
	memory.Description = fmt.Sprintf("Learned from session %s (duration: %s)",
		summary.SessionID,
		summary.Duration.Round(time.Second))

	return []*Memory{memory}, nil
}

// extractFailurePatterns creates anti-pattern memories from failed sessions.
//
// Failure patterns become warnings about approaches to avoid.
func (d *Distiller) extractFailurePatterns(summary SessionSummary) ([]*Memory, error) {
	// Create an anti-pattern memory
	title := d.generateTitle(summary.Task, "Anti-pattern")
	content := d.formatFailureContent(summary)

	memory, err := NewMemory(
		summary.ProjectID,
		title,
		content,
		OutcomeFailure,
		summary.Tags,
	)
	if err != nil {
		return nil, fmt.Errorf("creating failure memory: %w", err)
	}

	// Set distilled confidence (slightly lower for failures since they're harder to generalize)
	memory.Confidence = DistilledConfidence - 0.1
	if memory.Confidence < 0.0 {
		memory.Confidence = 0.0
	}

	// Add session metadata to description
	memory.Description = fmt.Sprintf("Anti-pattern learned from session %s (duration: %s)",
		summary.SessionID,
		summary.Duration.Round(time.Second))

	return []*Memory{memory}, nil
}

// generateTitle creates a concise title for a memory.
func (d *Distiller) generateTitle(task string, outcome string) string {
	// Truncate task if too long
	maxTaskLen := 50
	if len(task) > maxTaskLen {
		task = task[:maxTaskLen] + "..."
	}

	// Capitalize first letter
	if len(task) > 0 {
		task = strings.ToUpper(task[:1]) + task[1:]
	}

	return fmt.Sprintf("%s: %s", outcome, task)
}

// formatSuccessContent formats a success pattern into memory content.
func (d *Distiller) formatSuccessContent(summary SessionSummary) string {
	var b strings.Builder

	b.WriteString("## Task\n")
	b.WriteString(summary.Task)
	b.WriteString("\n\n")

	b.WriteString("## Successful Approach\n")
	b.WriteString(summary.Approach)
	b.WriteString("\n\n")

	b.WriteString("## Result\n")
	b.WriteString(summary.Result)
	b.WriteString("\n\n")

	if len(summary.Tags) > 0 {
		b.WriteString("## Tags\n")
		b.WriteString(strings.Join(summary.Tags, ", "))
		b.WriteString("\n\n")
	}

	b.WriteString("## When to Use\n")
	b.WriteString("Apply this approach when facing similar tasks involving: ")
	b.WriteString(strings.Join(summary.Tags, ", "))
	b.WriteString(".\n")

	return b.String()
}

// formatFailureContent formats a failure pattern into memory content.
func (d *Distiller) formatFailureContent(summary SessionSummary) string {
	var b strings.Builder

	b.WriteString("## Task\n")
	b.WriteString(summary.Task)
	b.WriteString("\n\n")

	b.WriteString("## Failed Approach (Avoid This)\n")
	b.WriteString(summary.Approach)
	b.WriteString("\n\n")

	b.WriteString("## What Went Wrong\n")
	b.WriteString(summary.Result)
	b.WriteString("\n\n")

	if len(summary.Tags) > 0 {
		b.WriteString("## Tags\n")
		b.WriteString(strings.Join(summary.Tags, ", "))
		b.WriteString("\n\n")
	}

	b.WriteString("## Warning\n")
	b.WriteString("Avoid this approach when working with: ")
	b.WriteString(strings.Join(summary.Tags, ", "))
	b.WriteString(". Look for alternative strategies instead.\n")

	return b.String()
}

// CosineSimilarity computes the cosine similarity between two embedding vectors.
//
// Cosine similarity measures the cosine of the angle between two vectors,
// producing a value between -1 and 1:
//   - 1.0: vectors point in the same direction (identical)
//   - 0.0: vectors are orthogonal (unrelated)
//   - -1.0: vectors point in opposite directions (opposite)
//
// For embedding vectors, similarity is typically in the range [0, 1] since
// embeddings generally have positive components.
//
// Formula: cos(θ) = (A · B) / (||A|| * ||B||)
//
// Returns 0.0 for invalid inputs (empty vectors, zero-magnitude vectors,
// or vectors of different lengths).
func CosineSimilarity(vec1, vec2 []float32) float64 {
	// Validate inputs
	if len(vec1) == 0 || len(vec2) == 0 {
		return 0.0
	}
	if len(vec1) != len(vec2) {
		return 0.0
	}

	// Compute dot product and magnitudes
	var dotProduct float64
	var magnitude1 float64
	var magnitude2 float64

	for i := 0; i < len(vec1); i++ {
		v1 := float64(vec1[i])
		v2 := float64(vec2[i])
		dotProduct += v1 * v2
		magnitude1 += v1 * v1
		magnitude2 += v2 * v2
	}

	// Check for zero-magnitude vectors
	if magnitude1 == 0.0 || magnitude2 == 0.0 {
		return 0.0
	}

	// Compute cosine similarity
	// Use sqrt for magnitudes: ||A|| = sqrt(A · A)
	magnitude1 = math.Sqrt(magnitude1)
	magnitude2 = math.Sqrt(magnitude2)

	return dotProduct / (magnitude1 * magnitude2)
}

// FindSimilarClusters detects groups of similar memories for a project.
//
// Searches all memories in the project and groups those with similarity
// scores above the threshold. Uses greedy clustering: for each memory,
// finds all similar memories above threshold, forms cluster if >=2 members.
//
// The algorithm:
//  1. Retrieve all memories for the project
//  2. Get embedding vectors for each memory
//  3. For each memory, compute similarity with all other memories
//  4. Group memories with similarity > threshold
//  5. Form clusters only if they have >= 2 members
//  6. Calculate cluster statistics (centroid, average similarity, min similarity)
//
// Parameters:
//   - ctx: Context for cancellation and timeouts
//   - projectID: Project to search for similar memories
//   - threshold: Minimum similarity score (0.0-1.0, typically 0.8)
//
// Returns:
//   - Slice of similarity clusters, each containing related memories
//   - Error if clustering fails
func (d *Distiller) FindSimilarClusters(ctx context.Context, projectID string, threshold float64) ([]SimilarityCluster, error) {
	if projectID == "" {
		return nil, ErrEmptyProjectID
	}
	if threshold < 0.0 || threshold > 1.0 {
		return nil, fmt.Errorf("threshold must be between 0.0 and 1.0, got %f", threshold)
	}

	d.logger.Info("finding similar memory clusters",
		zap.String("project_id", projectID),
		zap.Float64("threshold", threshold))

	// Get all memories for the project
	memories, err := d.service.ListMemories(ctx, projectID, 0, 0)
	if err != nil {
		return nil, fmt.Errorf("listing memories: %w", err)
	}

	if len(memories) < 2 {
		// Need at least 2 memories to form a cluster
		d.logger.Debug("not enough memories for clustering",
			zap.Int("count", len(memories)))
		return []SimilarityCluster{}, nil
	}

	d.logger.Debug("retrieved memories for clustering",
		zap.Int("count", len(memories)))

	// Get embedding vectors for all memories
	type memoryWithVector struct {
		memory *Memory
		vector []float32
	}

	memVecs := make([]memoryWithVector, 0, len(memories))
	for i := range memories {
		var vector []float32
		var err error

		// Try project-specific method first (for StoreProvider), fall back to legacy
		if d.service.stores != nil {
			vector, err = d.service.GetMemoryVectorByProjectID(ctx, projectID, memories[i].ID)
		} else {
			vector, err = d.service.GetMemoryVector(ctx, memories[i].ID)
		}

		if err != nil {
			d.logger.Warn("failed to get memory vector, skipping",
				zap.String("memory_id", memories[i].ID),
				zap.Error(err))
			continue
		}

		memVecs = append(memVecs, memoryWithVector{
			memory: &memories[i],
			vector: vector,
		})
	}

	if len(memVecs) < 2 {
		d.logger.Debug("not enough memories with vectors for clustering",
			zap.Int("count", len(memVecs)))
		return []SimilarityCluster{}, nil
	}

	// Track which memories have already been clustered
	clustered := make(map[string]bool)
	var clusters []SimilarityCluster

	// Greedy clustering: for each memory, find all similar memories above threshold
	for i := 0; i < len(memVecs); i++ {
		// Skip if already in a cluster
		if clustered[memVecs[i].memory.ID] {
			continue
		}

		// Find all memories similar to this one
		similar := []*Memory{memVecs[i].memory}
		similarVectors := [][]float32{memVecs[i].vector}
		similarities := []float64{}

		for j := 0; j < len(memVecs); j++ {
			if i == j {
				continue
			}
			if clustered[memVecs[j].memory.ID] {
				continue
			}

			similarity := CosineSimilarity(memVecs[i].vector, memVecs[j].vector)
			if similarity > threshold {
				similar = append(similar, memVecs[j].memory)
				similarVectors = append(similarVectors, memVecs[j].vector)
				similarities = append(similarities, similarity)
			}
		}

		// Only form cluster if >= 2 members
		if len(similar) < 2 {
			continue
		}

		// Mark all members as clustered
		for _, mem := range similar {
			clustered[mem.ID] = true
		}

		// Calculate cluster statistics
		centroid := calculateCentroid(similarVectors)
		avgSim, minSim := calculateSimilarityStats(similarities)

		cluster := SimilarityCluster{
			Members:           similar,
			CentroidVector:    centroid,
			AverageSimilarity: avgSim,
			MinSimilarity:     minSim,
		}

		clusters = append(clusters, cluster)

		d.logger.Debug("formed cluster",
			zap.Int("members", len(similar)),
			zap.Float64("avg_similarity", avgSim),
			zap.Float64("min_similarity", minSim))
	}

	d.logger.Info("clustering completed",
		zap.String("project_id", projectID),
		zap.Int("clusters", len(clusters)),
		zap.Int("total_memories", len(memories)),
		zap.Int("clustered_memories", len(clustered)))

	return clusters, nil
}

// calculateCentroid computes the average (centroid) vector from a set of vectors.
func calculateCentroid(vectors [][]float32) []float32 {
	if len(vectors) == 0 {
		return nil
	}

	vectorSize := len(vectors[0])
	centroid := make([]float32, vectorSize)

	// Sum all vectors
	for _, vec := range vectors {
		for i := 0; i < vectorSize; i++ {
			centroid[i] += vec[i]
		}
	}

	// Divide by count to get average
	count := float32(len(vectors))
	for i := 0; i < vectorSize; i++ {
		centroid[i] /= count
	}

	return centroid
}

// calculateSimilarityStats computes average and minimum similarity from a set of similarity scores.
func calculateSimilarityStats(similarities []float64) (avg float64, min float64) {
	if len(similarities) == 0 {
		return 0.0, 0.0
	}

	min = 1.0
	var sum float64

	for _, sim := range similarities {
		sum += sim
		if sim < min {
			min = sim
		}
	}

	avg = sum / float64(len(similarities))
	return avg, min
}

// buildConsolidationPrompt creates a prompt for LLM-powered memory synthesis.
//
// This function formats a cluster of similar memories into a structured prompt
// that instructs the LLM to synthesize them into a consolidated memory.
// The prompt asks the LLM to:
//   - Identify the common theme across all memories
//   - Synthesize key insights into coherent knowledge
//   - Preserve important details that shouldn't be lost
//   - Note when and how to apply this consolidated knowledge
//
// The resulting prompt is designed to produce high-quality consolidated memories
// that are more valuable than the individual source memories.
func buildConsolidationPrompt(memories []*Memory) string {
	var b strings.Builder

	b.WriteString("You are a memory consolidation assistant. Your task is to analyze the following related memories ")
	b.WriteString("and synthesize them into a single, more valuable consolidated memory.\n\n")

	b.WriteString("## Source Memories\n\n")

	// Format each memory with clear separation
	for i, mem := range memories {
		b.WriteString(fmt.Sprintf("### Memory %d: %s\n\n", i+1, mem.Title))

		if mem.Description != "" {
			b.WriteString(fmt.Sprintf("**Description:** %s\n\n", mem.Description))
		}

		b.WriteString("**Content:**\n")
		b.WriteString(mem.Content)
		b.WriteString("\n\n")

		if len(mem.Tags) > 0 {
			b.WriteString(fmt.Sprintf("**Tags:** %s\n\n", strings.Join(mem.Tags, ", ")))
		}

		b.WriteString(fmt.Sprintf("**Outcome:** %s\n", mem.Outcome))
		b.WriteString(fmt.Sprintf("**Confidence:** %.2f\n", mem.Confidence))
		b.WriteString(fmt.Sprintf("**Usage Count:** %d\n\n", mem.UsageCount))

		// Add separator between memories
		if i < len(memories)-1 {
			b.WriteString("---\n\n")
		}
	}

	b.WriteString("## Your Task\n\n")
	b.WriteString("Please synthesize these memories into a single consolidated memory by following these steps:\n\n")

	b.WriteString("1. **Identify the Common Theme:** What underlying concept, pattern, or strategy connects these memories?\n\n")

	b.WriteString("2. **Synthesize Key Insights:** Combine the most important insights from all memories into a coherent narrative. ")
	b.WriteString("Don't just list them - create an integrated understanding that's more valuable than the parts.\n\n")

	b.WriteString("3. **Preserve Important Details:** Ensure critical information isn't lost. ")
	b.WriteString("Include specific examples, caveats, or edge cases mentioned in the source memories.\n\n")

	b.WriteString("4. **Note When to Apply:** Clearly describe the situations, contexts, or conditions where this ")
	b.WriteString("consolidated knowledge should be applied. Help future sessions recognize when this memory is relevant.\n\n")

	b.WriteString("## Output Format\n\n")
	b.WriteString("Provide your consolidated memory in the following format:\n\n")

	b.WriteString("```\n")
	b.WriteString("TITLE: [A clear, concise title for the consolidated memory]\n\n")
	b.WriteString("CONTENT:\n")
	b.WriteString("[The synthesized content following the structure above]\n\n")
	b.WriteString("TAGS: [Comma-separated tags that apply to this consolidated knowledge]\n\n")
	b.WriteString("OUTCOME: [Either 'success' or 'failure' based on the predominant outcome type]\n\n")
	b.WriteString("SOURCE_ATTRIBUTION:\n")
	b.WriteString("[A brief note about how the source memories contributed to this synthesis]\n")
	b.WriteString("```\n\n")

	b.WriteString("Remember: The goal is to create a MORE valuable memory than any individual source. ")
	b.WriteString("Synthesize insights, don't just summarize.\n")

	return b.String()
}

// parseConsolidatedMemory parses an LLM response into a Memory struct.
//
// This function extracts structured fields from the LLM's consolidation response
// and creates a Memory suitable for storage. The LLM response is expected to
// contain the following fields in the format produced by buildConsolidationPrompt:
//   - TITLE: A clear, concise title for the consolidated memory
//   - CONTENT: The synthesized content
//   - TAGS: Comma-separated tags (optional)
//   - OUTCOME: Either 'success' or 'failure'
//   - SOURCE_ATTRIBUTION: Attribution note about source memories (optional)
//
// Parameters:
//   - llmResponse: The raw text response from the LLM
//   - sourceIDs: The IDs of source memories that were consolidated
//
// Returns:
//   - A Memory struct populated with parsed fields (caller will wrap in ConsolidatedMemory)
//   - Error if required fields are missing or invalid
//
// The projectID field in the returned Memory will be empty and must be set by the caller.
// The SOURCE_ATTRIBUTION is stored in the Memory's Description field.
func parseConsolidatedMemory(llmResponse string, sourceIDs []string) (*Memory, error) {
	if llmResponse == "" {
		return nil, fmt.Errorf("llm response cannot be empty")
	}
	if len(sourceIDs) == 0 {
		return nil, fmt.Errorf("sourceIDs cannot be empty")
	}

	// Extract fields from the LLM response
	title := extractField(llmResponse, "TITLE:")
	content := extractField(llmResponse, "CONTENT:")
	tagsStr := extractField(llmResponse, "TAGS:")
	outcomeStr := extractField(llmResponse, "OUTCOME:")
	sourceAttribution := extractField(llmResponse, "SOURCE_ATTRIBUTION:")

	// Validate required fields
	if title == "" {
		return nil, fmt.Errorf("TITLE field is required in LLM response")
	}
	if content == "" {
		return nil, fmt.Errorf("CONTENT field is required in LLM response")
	}
	if outcomeStr == "" {
		return nil, fmt.Errorf("OUTCOME field is required in LLM response")
	}

	// Parse outcome
	outcomeStr = strings.ToLower(strings.TrimSpace(outcomeStr))
	var outcome Outcome
	switch outcomeStr {
	case "success":
		outcome = OutcomeSuccess
	case "failure":
		outcome = OutcomeFailure
	default:
		return nil, fmt.Errorf("invalid OUTCOME value: %s (must be 'success' or 'failure')", outcomeStr)
	}

	// Parse tags (comma-separated, optional)
	var tags []string
	if tagsStr != "" {
		for _, tag := range strings.Split(tagsStr, ",") {
			tag = strings.TrimSpace(tag)
			if tag != "" {
				tags = append(tags, tag)
			}
		}
	}

	// Create the memory
	// Note: ProjectID must be set by caller
	now := time.Now()
	memory := &Memory{
		ID:          "", // Will be set by caller when storing
		ProjectID:   "", // Must be set by caller
		Title:       strings.TrimSpace(title),
		Description: strings.TrimSpace(sourceAttribution), // Store attribution in description
		Content:     strings.TrimSpace(content),
		Outcome:     outcome,
		Confidence:  DistilledConfidence, // Start with distilled confidence
		UsageCount:  0,
		Tags:        tags,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	return memory, nil
}

// extractField extracts the value of a field from the LLM response.
//
// Searches for the field label (e.g., "TITLE:") and extracts everything
// after it until the next field label or end of string. Handles both
// single-line and multi-line field values.
//
// Returns empty string if the field is not found.
func extractField(text, fieldLabel string) string {
	// Find the field label
	startIdx := strings.Index(text, fieldLabel)
	if startIdx == -1 {
		return ""
	}

	// Start after the label
	startIdx += len(fieldLabel)

	// Find the next field label (all caps followed by colon)
	// Common field labels: TITLE:, CONTENT:, TAGS:, OUTCOME:, SOURCE_ATTRIBUTION:
	fieldLabels := []string{"TITLE:", "CONTENT:", "TAGS:", "OUTCOME:", "SOURCE_ATTRIBUTION:"}
	endIdx := len(text)

	for _, label := range fieldLabels {
		// Don't match the current field label
		if label == fieldLabel {
			continue
		}

		// Find next occurrence of this label after our field
		idx := strings.Index(text[startIdx:], label)
		if idx != -1 {
			absoluteIdx := startIdx + idx
			if absoluteIdx < endIdx {
				endIdx = absoluteIdx
			}
		}
	}

	// Extract the value
	value := text[startIdx:endIdx]

	// Clean up: trim whitespace and remove markdown code block markers
	value = strings.TrimSpace(value)
	value = strings.Trim(value, "`")
	value = strings.TrimSpace(value)

	// Remove leading newlines and excessive whitespace
	lines := strings.Split(value, "\n")
	var cleanedLines []string
	for _, line := range lines {
		// Trim trailing whitespace but preserve intentional indentation
		line = strings.TrimRight(line, " \t")
		cleanedLines = append(cleanedLines, line)
	}

	// Join back with newlines and trim outer whitespace
	value = strings.Join(cleanedLines, "\n")
	value = strings.TrimSpace(value)

	return value
}

// MergeCluster synthesizes a cluster of similar memories into one consolidated memory.
//
// This method uses the configured LLM client to analyze the cluster members and create
// a synthesized memory that captures their common themes and key insights. The process:
//   1. Validates the cluster has at least 2 members and LLM client is configured
//   2. Builds a consolidation prompt from cluster members
//   3. Calls the LLM to synthesize the memories
//   4. Parses the LLM response into a Memory struct
//   5. Calculates consolidated confidence from source memories
//   6. Stores the new consolidated memory
//   7. Links source memories to the consolidated version
//
// The consolidated memory includes source attribution and links back to the original
// memories via their ConsolidationID fields.
//
// Parameters:
//   - ctx: Context for cancellation and timeouts
//   - cluster: Similarity cluster to merge (must have >= 2 members)
//
// Returns:
//   - The newly created consolidated memory
//   - Error if LLM client not configured, synthesis fails, or storage fails
func (d *Distiller) MergeCluster(ctx context.Context, cluster *SimilarityCluster) (*Memory, error) {
	// Validate inputs
	if cluster == nil {
		return nil, fmt.Errorf("cluster cannot be nil")
	}
	if len(cluster.Members) < 2 {
		return nil, fmt.Errorf("cluster must have at least 2 members, got %d", len(cluster.Members))
	}
	if d.llmClient == nil {
		return nil, fmt.Errorf("LLM client not configured for memory consolidation")
	}

	// All members should belong to the same project - use first member's projectID
	projectID := cluster.Members[0].ProjectID
	if projectID == "" {
		return nil, fmt.Errorf("project ID cannot be empty")
	}

	d.logger.Info("merging memory cluster",
		zap.String("project_id", projectID),
		zap.Int("cluster_size", len(cluster.Members)),
		zap.Float64("avg_similarity", cluster.AverageSimilarity))

	// Build consolidation prompt
	prompt := buildConsolidationPrompt(cluster.Members)

	// Call LLM to synthesize memories
	d.logger.Debug("calling LLM for memory synthesis",
		zap.String("project_id", projectID),
		zap.Int("prompt_length", len(prompt)))

	llmResponse, err := d.llmClient.Complete(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("LLM synthesis failed: %w", err)
	}

	d.logger.Debug("received LLM synthesis response",
		zap.String("project_id", projectID),
		zap.Int("response_length", len(llmResponse)))

	// Extract source IDs
	sourceIDs := make([]string, len(cluster.Members))
	for i, mem := range cluster.Members {
		sourceIDs[i] = mem.ID
	}

	// Parse LLM response into Memory
	consolidatedMemory, err := parseConsolidatedMemory(llmResponse, sourceIDs)
	if err != nil {
		return nil, fmt.Errorf("parsing LLM response: %w", err)
	}

	// Set project ID (parseConsolidatedMemory leaves it empty)
	consolidatedMemory.ProjectID = projectID

	// Calculate merged confidence from source memories
	consolidatedMemory.Confidence = d.calculateMergedConfidence(cluster.Members)

	d.logger.Debug("calculated merged confidence",
		zap.String("project_id", projectID),
		zap.Float64("confidence", consolidatedMemory.Confidence))

	// Store the consolidated memory
	if err := d.service.Record(ctx, consolidatedMemory); err != nil {
		return nil, fmt.Errorf("storing consolidated memory: %w", err)
	}

	d.logger.Info("consolidated memory created",
		zap.String("id", consolidatedMemory.ID),
		zap.String("project_id", projectID),
		zap.String("title", consolidatedMemory.Title),
		zap.Float64("confidence", consolidatedMemory.Confidence))

	// Link source memories to consolidated version
	if err := d.linkMemoriesToConsolidated(ctx, projectID, sourceIDs, consolidatedMemory.ID); err != nil {
		// Log error but don't fail - the consolidated memory was created successfully
		d.logger.Warn("failed to link source memories to consolidated version",
			zap.String("consolidated_id", consolidatedMemory.ID),
			zap.Error(err))
	}

	return consolidatedMemory, nil
}

// calculateMergedConfidence computes the confidence score for a consolidated memory.
//
// The confidence is calculated as a weighted average of source memory confidences,
// where the weights are based on usage counts. Memories that have been used more
// frequently contribute more to the final confidence score.
//
// Formula: confidence = sum(confidence_i * weight_i) / sum(weight_i)
// where weight_i = usageCount_i + 1 (add 1 to avoid zero weights)
//
// This ensures that:
//   - Frequently used, high-confidence memories dominate the score
//   - Rarely used memories still contribute (via the +1)
//   - The result is bounded by [min_confidence, max_confidence] of sources
func (d *Distiller) calculateMergedConfidence(sources []*Memory) float64 {
	if len(sources) == 0 {
		return DistilledConfidence // Default if no sources
	}

	var weightedSum float64
	var totalWeight float64

	for _, mem := range sources {
		// Weight by usage count + 1 (to avoid zero weights)
		weight := float64(mem.UsageCount + 1)
		weightedSum += mem.Confidence * weight
		totalWeight += weight
	}

	if totalWeight == 0 {
		// Shouldn't happen due to +1, but guard against division by zero
		return DistilledConfidence
	}

	confidence := weightedSum / totalWeight

	// Ensure confidence is in valid range [0.0, 1.0]
	if confidence < 0.0 {
		confidence = 0.0
	}
	if confidence > 1.0 {
		confidence = 1.0
	}

	return confidence
}

// calculateConsolidatedConfidence computes the confidence score for a consolidated memory
// with a consensus bonus.
//
// The confidence is calculated as a weighted average of source memory confidences
// (weighted by usage counts), with an additional bonus for consensus among sources.
// The consensus bonus rewards situations where:
//   - Source memories have similar confidence scores (low variance)
//   - Multiple memories agree (more sources = higher potential bonus)
//
// Formula:
//   base = sum(confidence_i * weight_i) / sum(weight_i)
//   where weight_i = usageCount_i + 1
//
//   consensus_bonus = (1 - normalized_std_dev) * min(num_sources / 10, 1.0) * 0.1
//   final = base + consensus_bonus (capped at 1.0)
//
// This ensures:
//   - High agreement among many sources increases confidence
//   - Low variance (consensus) provides up to 0.1 bonus
//   - Bonus scales with number of sources (up to 10 sources)
//   - Result is always in valid range [0.0, 1.0]
func calculateConsolidatedConfidence(sources []*Memory) float64 {
	if len(sources) == 0 {
		return DistilledConfidence // Default if no sources
	}

	// Calculate weighted average (base confidence)
	var weightedSum float64
	var totalWeight float64

	for _, mem := range sources {
		// Weight by usage count + 1 (to avoid zero weights)
		weight := float64(mem.UsageCount + 1)
		weightedSum += mem.Confidence * weight
		totalWeight += weight
	}

	if totalWeight == 0 {
		// Shouldn't happen due to +1, but guard against division by zero
		return DistilledConfidence
	}

	baseConfidence := weightedSum / totalWeight

	// Calculate consensus bonus based on confidence variance
	if len(sources) == 1 {
		// Single source: no consensus bonus
		return clampConfidence(baseConfidence)
	}

	// Calculate mean confidence (unweighted, for variance calculation)
	var sumConfidence float64
	for _, mem := range sources {
		sumConfidence += mem.Confidence
	}
	meanConfidence := sumConfidence / float64(len(sources))

	// Calculate variance
	var varianceSum float64
	for _, mem := range sources {
		diff := mem.Confidence - meanConfidence
		varianceSum += diff * diff
	}
	variance := varianceSum / float64(len(sources))
	stdDev := math.Sqrt(variance)

	// Normalize std dev by the theoretical maximum (0.5 for range [0, 1])
	// This gives us a value in [0, 1] where 0 = perfect consensus, 1 = maximum disagreement
	normalizedStdDev := stdDev / 0.5
	if normalizedStdDev > 1.0 {
		normalizedStdDev = 1.0
	}

	// Calculate consensus factor: 1.0 for perfect agreement, 0.0 for maximum disagreement
	consensusFactor := 1.0 - normalizedStdDev

	// Scale by number of sources (more agreeing sources = higher bonus, max at 10 sources)
	numSourcesFactor := math.Min(float64(len(sources))/10.0, 1.0)

	// Calculate consensus bonus (up to 0.1)
	consensusBonus := consensusFactor * numSourcesFactor * 0.1

	// Combine base confidence with consensus bonus
	finalConfidence := baseConfidence + consensusBonus

	return clampConfidence(finalConfidence)
}

// clampConfidence ensures a confidence value is within the valid range [0.0, 1.0].
func clampConfidence(confidence float64) float64 {
	if confidence < 0.0 {
		return 0.0
	}
	if confidence > 1.0 {
		return 1.0
	}
	return confidence
}

// linkMemoriesToConsolidated updates source memories to link them to the consolidated version.
//
// This method updates each source memory's ConsolidationID field to point to the
// consolidated memory. The source memories are preserved with their original content
// for attribution and traceability.
//
// Note: This is a helper method and errors are logged but not propagated to avoid
// failing the consolidation if linking fails (the consolidated memory is already created).
func (d *Distiller) linkMemoriesToConsolidated(ctx context.Context, projectID string, sourceIDs []string, consolidatedID string) error {
	for _, sourceID := range sourceIDs {
		// Get the source memory
		memory, err := d.service.GetByProjectID(ctx, projectID, sourceID)
		if err != nil {
			d.logger.Warn("failed to get source memory for linking",
				zap.String("source_id", sourceID),
				zap.Error(err))
			continue
		}

		// Set consolidation ID
		memory.ConsolidationID = &consolidatedID
		memory.UpdatedAt = time.Now()

		// Update the memory in storage
		// We need to delete and re-add to update the ConsolidationID field
		if err := d.service.DeleteByProjectID(ctx, projectID, sourceID); err != nil {
			d.logger.Warn("failed to delete source memory for update",
				zap.String("source_id", sourceID),
				zap.Error(err))
			continue
		}

		if err := d.service.Record(ctx, memory); err != nil {
			d.logger.Warn("failed to re-add source memory with consolidation link",
				zap.String("source_id", sourceID),
				zap.Error(err))
			continue
		}

		d.logger.Debug("linked source memory to consolidated version",
			zap.String("source_id", sourceID),
			zap.String("consolidated_id", consolidatedID))
	}

	return nil
}
