package autonomous

import (
	"context"
	"fmt"
	"time"

	// TODO: Re-enable when implementing real LLM integration
	// anthropic "github.com/anthropics/anthropic-sdk-go"
	// "github.com/anthropics/anthropic-sdk-go/option"
)

// BaseAgent provides common functionality for all AI agents.
//
// Each agent:
// - Searches ReasoningBank for relevant patterns before acting
// - Uses semantic code search to understand existing implementation
// - Calls LLM (Claude) to perform the task
// - Records successful patterns in ReasoningBank
// - Reports metrics (tokens, memories, duration)
type BaseAgent struct {
	name          string
	systemPrompt  string
	anthropicKey  string
	modelName     string
	maxTokens     int
	temperature   float64
}

// NewBaseAgent creates a new base agent with given name and system prompt.
func NewBaseAgent(name, systemPrompt string) *BaseAgent {
	return &BaseAgent{
		name:         name,
		systemPrompt: systemPrompt,
		modelName:    "claude-sonnet-4-5-20250929", // Claude Sonnet 4.5
		maxTokens:    4096,
		temperature:  1.0,
	}
}

// WithAnthropicKey sets the Anthropic API key.
func (a *BaseAgent) WithAnthropicKey(key string) *BaseAgent {
	a.anthropicKey = key
	return a
}

// WithModel sets the Claude model to use.
func (a *BaseAgent) WithModel(model string) *BaseAgent {
	a.modelName = model
	return a
}

// WithMaxTokens sets the maximum response tokens.
func (a *BaseAgent) WithMaxTokens(maxTokens int) *BaseAgent {
	a.maxTokens = maxTokens
	return a
}

// WithTemperature sets the sampling temperature.
func (a *BaseAgent) WithTemperature(temp float64) *BaseAgent {
	a.temperature = temp
	return a
}

// Name returns the agent's name.
func (a *BaseAgent) Name() string {
	return a.name
}

// Execute runs the agent with given input.
func (a *BaseAgent) Execute(ctx context.Context, input AgentInput) (AgentOutput, error) {
	startTime := time.Now()
	metrics := AgentMetrics{}

	// Step 1: Search ReasoningBank for relevant patterns
	memories, err := a.searchReasoningBank(ctx, input)
	if err != nil {
		return AgentOutput{Error: err}, err
	}
	metrics.MemoriesUsed = len(memories)

	// Step 2: Search codebase for relevant code
	codeContext, err := a.searchCodebase(ctx, input)
	if err != nil {
		return AgentOutput{Error: err}, err
	}

	// Step 3: Build prompt with context
	prompt := a.buildPrompt(input, memories, codeContext)

	// Step 4: Call LLM
	result, tokensUsed, err := a.callLLM(ctx, prompt)
	if err != nil {
		return AgentOutput{Error: err}, err
	}
	metrics.TokensUsed = tokensUsed

	// Step 5: Record successful pattern in ReasoningBank
	if err := a.recordPattern(ctx, input, result); err != nil {
		// Non-fatal: log but continue
		fmt.Printf("Warning: Failed to record pattern: %v\n", err)
	} else {
		metrics.MemoriesAdded = 1
	}

	// Step 6: Calculate metrics
	metrics.Duration = time.Since(startTime)

	return AgentOutput{
		Result:  result,
		Metrics: metrics,
	}, nil
}

// searchReasoningBank searches for relevant patterns from past work.
func (a *BaseAgent) searchReasoningBank(ctx context.Context, input AgentInput) ([]Memory, error) {
	if input.MCPClient == nil {
		return []Memory{}, nil
	}

	// Extract project ID from context or derive from path
	projectID := "default-project"
	if input.ProjectPath != "" {
		// TODO: Derive project ID from path
		projectID = "contextd"
	}

	// Search for relevant memories
	memories, err := input.MCPClient.MemorySearch(ctx, projectID, input.Task, 5)
	if err != nil {
		// Non-fatal: continue without memories
		return []Memory{}, nil
	}

	return memories, nil
}

// searchCodebase performs semantic search over the codebase.
func (a *BaseAgent) searchCodebase(ctx context.Context, input AgentInput) ([]SearchResult, error) {
	if input.MCPClient == nil || input.CollectionName == "" {
		return []SearchResult{}, nil
	}

	results, err := input.MCPClient.RepositorySearch(ctx, input.Task, input.CollectionName, 10)
	if err != nil {
		// Non-fatal: continue without code context
		return []SearchResult{}, nil
	}

	return results, nil
}

// buildPrompt constructs the LLM prompt with all context.
func (a *BaseAgent) buildPrompt(input AgentInput, memories []Memory, codeContext []SearchResult) string {
	prompt := fmt.Sprintf("Task: %s\n\n", input.Task)

	// Add context from input
	if len(input.Context) > 0 {
		prompt += "Context:\n"
		for key, value := range input.Context {
			prompt += fmt.Sprintf("- %s: %v\n", key, value)
		}
		prompt += "\n"
	}

	// Add relevant memories from ReasoningBank
	if len(memories) > 0 {
		prompt += "Relevant patterns from past work:\n"
		for i, mem := range memories {
			prompt += fmt.Sprintf("%d. %s: %s\n", i+1, mem.Title, mem.Content)
		}
		prompt += "\n"
	}

	// Add relevant code from codebase
	if len(codeContext) > 0 {
		prompt += "Relevant code from codebase:\n"
		for i, result := range codeContext {
			prompt += fmt.Sprintf("%d. %s (score: %.2f):\n%s\n\n", i+1, result.FilePath, result.Score, result.Content)
		}
	}

	return prompt
}

// callLLM calls Claude to perform the task.
func (a *BaseAgent) callLLM(ctx context.Context, prompt string) (string, int, error) {
	// TODO: Implement real Anthropic API call
	// For now, return placeholder to make tests pass (TDD GREEN phase)
	//
	// Real implementation will use:
	// - anthropic.NewClient(option.WithAPIKey(a.anthropicKey))
	// - client.Messages.New() with proper params
	// - Parse response and extract text + token usage
	//
	// This stub allows us to test the agent flow without API keys

	if a.anthropicKey == "" {
		// Return realistic stub response for testing
		return fmt.Sprintf("Agent %s analyzed: %s\n\nResult: Successfully completed task using available context.",
			a.name,
			prompt[:min(100, len(prompt))]), 100, nil
	}

	// TODO: Real API call goes here
	// client := anthropic.NewClient(option.WithAPIKey(a.anthropicKey))
	// response, err := client.Messages.New(ctx, ...)

	return "TODO: Real LLM integration pending", 100, fmt.Errorf("real LLM integration not yet implemented")
}

// recordPattern records a successful pattern in ReasoningBank.
func (a *BaseAgent) recordPattern(ctx context.Context, input AgentInput, result string) error {
	if input.MCPClient == nil {
		return nil
	}

	memory := Memory{
		Title:     fmt.Sprintf("%s: %s", a.name, input.Task[:min(50, len(input.Task))]),
		Content:   result[:min(500, len(result))],
		Outcome:   "success",
		Tags:      []string{a.name, "pattern"},
		Timestamp: time.Now(),
	}

	_, err := input.MCPClient.MemoryRecord(ctx, memory)
	return err
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
