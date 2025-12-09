package orchestrator

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// MCPClient abstracts MCP tool calls
type MCPClient interface {
	CallTool(ctx context.Context, name string, args map[string]interface{}) (interface{}, error)
}

// MemoryResult represents a memory search result
type MemoryResult struct {
	ID      string  `json:"id"`
	Content string  `json:"content"`
	Score   float64 `json:"score"`
}

// RemediationResult represents a remediation search result
type RemediationResult struct {
	ID           string  `json:"id"`
	ErrorPattern string  `json:"error_pattern"`
	Solution     string  `json:"solution"`
	Score        float64 `json:"score"`
}

// Checkpoint represents a saved checkpoint
type Checkpoint struct {
	ID   string                 `json:"id"`
	Name string                 `json:"name"`
	Data map[string]interface{} `json:"data"`
}

// ContextdRecorder implements MemoryRecorder using contextd MCP tools
type ContextdRecorder struct {
	client MCPClient
}

// NewContextdRecorder creates a new recorder that uses contextd MCP tools
func NewContextdRecorder(client MCPClient) *ContextdRecorder {
	return &ContextdRecorder{
		client: client,
	}
}

// RecordLearning saves a learning to contextd memory
func (r *ContextdRecorder) RecordLearning(ctx context.Context, content string, tags []string) error {
	args := map[string]interface{}{
		"content": content,
		"tags":    tags,
	}

	_, err := r.client.CallTool(ctx, "memory_record", args)
	if err != nil {
		return fmt.Errorf("failed to record learning: %w", err)
	}

	return nil
}

// RecordViolation records a workflow violation to memory
func (r *ContextdRecorder) RecordViolation(ctx context.Context, violation Violation) error {
	content := fmt.Sprintf("[Violation] Type: %s, Phase: %s, Severity: %s\nDescription: %s\nDetected: %s",
		violation.Type,
		violation.Phase,
		violation.Severity,
		violation.Description,
		violation.DetectedAt.Format(time.RFC3339),
	)

	args := map[string]interface{}{
		"content": content,
		"tags":    []string{"violation", string(violation.Type), string(violation.Severity)},
	}

	_, err := r.client.CallTool(ctx, "memory_record", args)
	if err != nil {
		return fmt.Errorf("failed to record violation: %w", err)
	}

	return nil
}

// SearchMemory searches contextd memory for relevant learnings
func (r *ContextdRecorder) SearchMemory(ctx context.Context, query string, limit int) ([]MemoryResult, error) {
	args := map[string]interface{}{
		"query": query,
		"k":     limit,
	}

	result, err := r.client.CallTool(ctx, "memory_search", args)
	if err != nil {
		return nil, fmt.Errorf("failed to search memory: %w", err)
	}

	// Parse results
	results, ok := result.([]map[string]interface{})
	if !ok {
		return []MemoryResult{}, nil
	}

	var memories []MemoryResult
	for _, r := range results {
		memory := MemoryResult{
			ID:      getString(r, "id"),
			Content: getString(r, "content"),
			Score:   getFloat64(r, "score"),
		}
		memories = append(memories, memory)
	}

	return memories, nil
}

// RecordRemediation records an error pattern and solution
func (r *ContextdRecorder) RecordRemediation(ctx context.Context, errorPattern, solution string) error {
	args := map[string]interface{}{
		"error_pattern": errorPattern,
		"solution":      solution,
	}

	_, err := r.client.CallTool(ctx, "remediation_record", args)
	if err != nil {
		return fmt.Errorf("failed to record remediation: %w", err)
	}

	return nil
}

// SearchRemediation searches for known fixes to an error
func (r *ContextdRecorder) SearchRemediation(ctx context.Context, errorText string) ([]RemediationResult, error) {
	args := map[string]interface{}{
		"query": errorText,
	}

	result, err := r.client.CallTool(ctx, "remediation_search", args)
	if err != nil {
		return nil, fmt.Errorf("failed to search remediation: %w", err)
	}

	// Parse results
	results, ok := result.([]map[string]interface{})
	if !ok {
		return []RemediationResult{}, nil
	}

	var remediations []RemediationResult
	for _, r := range results {
		rem := RemediationResult{
			ID:           getString(r, "id"),
			ErrorPattern: getString(r, "error_pattern"),
			Solution:     getString(r, "solution"),
			Score:        getFloat64(r, "score"),
		}
		remediations = append(remediations, rem)
	}

	return remediations, nil
}

// SaveCheckpoint saves a checkpoint with the given name and data
func (r *ContextdRecorder) SaveCheckpoint(ctx context.Context, name string, data map[string]interface{}) (string, error) {
	args := map[string]interface{}{
		"name": name,
		"data": data,
	}

	result, err := r.client.CallTool(ctx, "checkpoint_save", args)
	if err != nil {
		return "", fmt.Errorf("failed to save checkpoint: %w", err)
	}

	// Extract checkpoint ID
	if resultMap, ok := result.(map[string]interface{}); ok {
		if id, ok := resultMap["id"].(string); ok {
			return id, nil
		}
	}

	return "", fmt.Errorf("invalid checkpoint response")
}

// ResumeCheckpoint resumes from a saved checkpoint
func (r *ContextdRecorder) ResumeCheckpoint(ctx context.Context, checkpointID string) (*Checkpoint, error) {
	args := map[string]interface{}{
		"id": checkpointID,
	}

	result, err := r.client.CallTool(ctx, "checkpoint_resume", args)
	if err != nil {
		return nil, fmt.Errorf("failed to resume checkpoint: %w", err)
	}

	// Parse checkpoint
	resultMap, ok := result.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid checkpoint response")
	}

	checkpoint := &Checkpoint{
		ID:   getString(resultMap, "id"),
		Name: getString(resultMap, "name"),
	}

	if data, ok := resultMap["data"].(map[string]interface{}); ok {
		checkpoint.Data = data
	}

	return checkpoint, nil
}

// ProvideFeedback provides feedback on a memory's helpfulness
func (r *ContextdRecorder) ProvideFeedback(ctx context.Context, memoryID string, helpful bool) error {
	args := map[string]interface{}{
		"id":      memoryID,
		"helpful": helpful,
	}

	_, err := r.client.CallTool(ctx, "memory_feedback", args)
	if err != nil {
		return fmt.Errorf("failed to provide feedback: %w", err)
	}

	return nil
}

// Helper functions for type-safe map access
func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func getFloat64(m map[string]interface{}, key string) float64 {
	switch v := m[key].(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	default:
		return 0
	}
}

// BuildPhasePrompt creates a system prompt for a specific phase
func BuildPhasePrompt(phase Phase, config TaskConfig, previousResults map[Phase]*PhaseResult) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("You are executing phase: %s\n\n", phase))
	sb.WriteString(fmt.Sprintf("Task: %s\n\n", config.Description))

	// Phase-specific instructions
	switch phase {
	case PhaseInit:
		sb.WriteString("Instructions:\n")
		sb.WriteString("1. Analyze the task requirements\n")
		sb.WriteString("2. Search contextd memory for relevant past learnings\n")
		sb.WriteString("3. Search remediation patterns for known solutions\n")
		sb.WriteString("4. Report your analysis and proposed approach\n")

	case PhaseTest:
		sb.WriteString("Instructions (TDD - Tests First):\n")
		sb.WriteString("1. Write failing tests that define the expected behavior\n")
		sb.WriteString("2. Tests should be comprehensive and cover edge cases\n")
		sb.WriteString("3. Do NOT write implementation code in this phase\n")
		sb.WriteString("4. Report all test files created\n")

	case PhaseImplement:
		sb.WriteString("Instructions:\n")
		sb.WriteString("1. Write the minimum code to make tests pass\n")
		sb.WriteString("2. Follow existing patterns in the codebase\n")
		sb.WriteString("3. Keep changes focused and sequential\n")
		sb.WriteString("4. Report all implementation files modified\n")

	case PhaseVerify:
		sb.WriteString("Instructions:\n")
		sb.WriteString("1. Run the actual test suite (not --help)\n")
		sb.WriteString("2. Verify all tests pass\n")
		sb.WriteString("3. Check for any regressions\n")
		sb.WriteString("4. Report test output verbatim\n")

	case PhaseCommit:
		if config.RequireSeparateCommits {
			sb.WriteString("Instructions (Separate Commits Required):\n")
			sb.WriteString("1. Create a commit for test files only\n")
			sb.WriteString("2. Create a separate commit for implementation\n")
			sb.WriteString("3. Use descriptive commit messages\n")
		} else {
			sb.WriteString("Instructions:\n")
			sb.WriteString("1. Create a commit with all changes\n")
			sb.WriteString("2. Use a descriptive commit message\n")
		}
		sb.WriteString("4. Report commit IDs\n")

	case PhaseReport:
		sb.WriteString("Instructions:\n")
		sb.WriteString("1. Summarize what was accomplished\n")
		sb.WriteString("2. Record key learnings to contextd memory\n")
		sb.WriteString("3. Note any patterns that could help future tasks\n")
		sb.WriteString("4. Provide a final status report\n")
	}

	// Add context from previous phases
	if len(previousResults) > 0 {
		sb.WriteString("\n\nPrevious Phase Results:\n")
		for p, result := range previousResults {
			sb.WriteString(fmt.Sprintf("- %s: %s\n", p, result.Status))
			if result.Output != "" && len(result.Output) < 500 {
				sb.WriteString(fmt.Sprintf("  Output: %s\n", result.Output))
			}
		}
	}

	return sb.String()
}
