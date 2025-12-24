package agent

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
)

// Runner executes test scenarios.
type Runner struct {
	client ContextdClient
	llm    LLMClient
	logger *zap.Logger
}

// RunnerConfig configures a Runner.
type RunnerConfig struct {
	Client ContextdClient
	LLM    LLMClient
	Logger *zap.Logger
}

// NewRunner creates a new scenario runner.
func NewRunner(cfg RunnerConfig) (*Runner, error) {
	if cfg.Client == nil {
		return nil, fmt.Errorf("contextd client is required")
	}

	logger := cfg.Logger
	if logger == nil {
		logger = zap.NewNop()
	}

	return &Runner{
		client: cfg.Client,
		llm:    cfg.LLM,
		logger: logger,
	}, nil
}

// RunScenario executes a single scenario and returns results.
func (r *Runner) RunScenario(ctx context.Context, scenario Scenario) (*TestResult, error) {
	start := time.Now()

	r.logger.Info("starting scenario",
		zap.String("name", scenario.Name),
		zap.String("project_id", scenario.ProjectID))

	// Create agent for this scenario
	agent, err := New(Config{
		Client:    r.client,
		LLM:       r.llm,
		Persona:   scenario.Persona,
		ProjectID: scenario.ProjectID,
		Logger:    r.logger,
	})
	if err != nil {
		return &TestResult{
			Scenario: scenario.Name,
			Passed:   false,
			Error:    fmt.Sprintf("creating agent: %v", err),
			Duration: time.Since(start),
		}, nil
	}

	// Execute actions
	if len(scenario.Actions) > 0 {
		// Scripted scenario
		err = r.executeScriptedActions(ctx, agent, scenario.Actions)
	} else {
		// Autonomous scenario (LLM-driven)
		err = r.executeAutonomous(ctx, agent, scenario.MaxTurns)
	}

	if err != nil {
		return &TestResult{
			Scenario: scenario.Name,
			Passed:   false,
			Session:  agent.GetSession(),
			Error:    fmt.Sprintf("executing scenario: %v", err),
			Duration: time.Since(start),
		}, nil
	}

	// Run assertions
	assertResults := r.runAssertions(ctx, agent, scenario.Assertions)

	// Determine overall pass/fail
	passed := true
	for _, ar := range assertResults {
		if !ar.Passed {
			passed = false
			break
		}
	}

	session := agent.GetSession()
	session.EndTime = time.Now()

	return &TestResult{
		Scenario:   scenario.Name,
		Passed:     passed,
		Session:    session,
		Assertions: assertResults,
		Duration:   time.Since(start),
	}, nil
}

// executeScriptedActions runs predefined actions.
func (r *Runner) executeScriptedActions(ctx context.Context, agent *Agent, actions []Action) error {
	for i, action := range actions {
		r.logger.Debug("executing action",
			zap.Int("index", i),
			zap.String("type", action.Type))

		switch action.Type {
		case "record":
			title, _ := action.Args["title"].(string)
			content, _ := action.Args["content"].(string)
			outcome, _ := action.Args["outcome"].(string)
			if outcome == "" {
				outcome = "success"
			}
			var tags []string
			if t, ok := action.Args["tags"].([]string); ok {
				tags = t
			}
			_, err := agent.RecordMemory(ctx, title, content, outcome, tags)
			if err != nil {
				return fmt.Errorf("action %d (record): %w", i, err)
			}

		case "search":
			query, _ := action.Args["query"].(string)
			limit := 5
			if l, ok := action.Args["limit"].(int); ok {
				limit = l
			}
			_, err := agent.SearchMemories(ctx, query, limit)
			if err != nil {
				return fmt.Errorf("action %d (search): %w", i, err)
			}

		case "feedback":
			memoryID, _ := action.Args["memory_id"].(string)
			helpful, _ := action.Args["helpful"].(bool)
			reasoning, _ := action.Args["reasoning"].(string)

			// If memory_id is "last", use the last retrieved memory
			if memoryID == "last" && len(agent.memoriesRetrieved) > 0 {
				memoryID = agent.memoriesRetrieved[len(agent.memoriesRetrieved)-1].ID
			}

			_, err := agent.GiveFeedback(ctx, memoryID, helpful, reasoning)
			if err != nil {
				return fmt.Errorf("action %d (feedback): %w", i, err)
			}

		case "outcome":
			memoryID, _ := action.Args["memory_id"].(string)
			succeeded, _ := action.Args["succeeded"].(bool)
			taskDesc, _ := action.Args["task_description"].(string)

			// If memory_id is "last", use the last retrieved memory
			if memoryID == "last" && len(agent.memoriesRetrieved) > 0 {
				memoryID = agent.memoriesRetrieved[len(agent.memoriesRetrieved)-1].ID
			}

			_, err := agent.ReportOutcome(ctx, memoryID, succeeded, taskDesc)
			if err != nil {
				return fmt.Errorf("action %d (outcome): %w", i, err)
			}

		default:
			return fmt.Errorf("action %d: unknown type %q", i, action.Type)
		}
	}

	return nil
}

// executeAutonomous runs LLM-driven actions.
func (r *Runner) executeAutonomous(ctx context.Context, agent *Agent, maxTurns int) error {
	if agent.llm == nil {
		return fmt.Errorf("LLM client required for autonomous execution")
	}

	// TODO: Implement LLM-driven autonomous behavior
	// For now, return not implemented
	return fmt.Errorf("autonomous execution not yet implemented")
}

// runAssertions checks all assertions against the agent state.
func (r *Runner) runAssertions(ctx context.Context, agent *Agent, assertions []Assertion) []AssertResult {
	results := make([]AssertResult, 0, len(assertions))

	for _, assertion := range assertions {
		result := r.checkAssertion(ctx, agent, assertion)
		results = append(results, result)

		if result.Passed {
			r.logger.Debug("assertion passed",
				zap.String("type", assertion.Type),
				zap.String("target", assertion.Target))
		} else {
			r.logger.Warn("assertion failed",
				zap.String("type", assertion.Type),
				zap.String("target", assertion.Target),
				zap.String("message", result.Message))
		}
	}

	return results
}

func (r *Runner) checkAssertion(ctx context.Context, agent *Agent, assertion Assertion) AssertResult {
	result := AssertResult{
		Assertion: assertion,
		Passed:    false,
	}

	// Resolve "last" target to actual memory ID
	target := assertion.Target
	if target == "last" {
		if len(agent.memoriesRecorded) > 0 {
			target = agent.memoriesRecorded[len(agent.memoriesRecorded)-1]
		}
	}

	switch assertion.Type {
	case "confidence_increased":
		history := agent.GetConfidenceHistory(target)
		if len(history) < 2 {
			result.Message = "insufficient confidence history"
			return result
		}
		initial := history[0]
		final := history[len(history)-1]
		result.Actual = final - initial
		result.Passed = final > initial
		if !result.Passed {
			result.Message = fmt.Sprintf("confidence did not increase: %.4f -> %.4f", initial, final)
		}

	case "confidence_decreased":
		history := agent.GetConfidenceHistory(target)
		if len(history) < 2 {
			result.Message = "insufficient confidence history"
			return result
		}
		initial := history[0]
		final := history[len(history)-1]
		result.Actual = final - initial
		result.Passed = final < initial
		if !result.Passed {
			result.Message = fmt.Sprintf("confidence did not decrease: %.4f -> %.4f", initial, final)
		}

	case "confidence_above":
		threshold, ok := assertion.Value.(float64)
		if !ok {
			result.Message = "invalid threshold value"
			return result
		}
		history := agent.GetConfidenceHistory(target)
		if len(history) == 0 {
			result.Message = "no confidence history"
			return result
		}
		final := history[len(history)-1]
		result.Actual = final
		result.Passed = final > threshold
		if !result.Passed {
			result.Message = fmt.Sprintf("confidence %.4f not above threshold %.4f", final, threshold)
		}

	case "confidence_below":
		threshold, ok := assertion.Value.(float64)
		if !ok {
			result.Message = "invalid threshold value"
			return result
		}
		history := agent.GetConfidenceHistory(target)
		if len(history) == 0 {
			result.Message = "no confidence history"
			return result
		}
		final := history[len(history)-1]
		result.Actual = final
		result.Passed = final < threshold
		if !result.Passed {
			result.Message = fmt.Sprintf("confidence %.4f not below threshold %.4f", final, threshold)
		}

	case "memory_count":
		expected, ok := assertion.Value.(int)
		if !ok {
			// Try float64 (JSON default)
			if f, ok := assertion.Value.(float64); ok {
				expected = int(f)
			} else {
				result.Message = "invalid expected count"
				return result
			}
		}
		actual := len(agent.memoriesRecorded)
		result.Actual = actual
		result.Passed = actual == expected
		if !result.Passed {
			result.Message = fmt.Sprintf("expected %d memories, got %d", expected, actual)
		}

	case "feedback_count":
		expected, ok := assertion.Value.(int)
		if !ok {
			if f, ok := assertion.Value.(float64); ok {
				expected = int(f)
			} else {
				result.Message = "invalid expected count"
				return result
			}
		}
		actual := len(agent.feedback)
		result.Actual = actual
		result.Passed = actual == expected
		if !result.Passed {
			result.Message = fmt.Sprintf("expected %d feedback events, got %d", expected, actual)
		}

	default:
		result.Message = fmt.Sprintf("unknown assertion type: %s", assertion.Type)
	}

	return result
}

// RunScenarios executes multiple scenarios and aggregates results.
func (r *Runner) RunScenarios(ctx context.Context, scenarios []Scenario) ([]TestResult, error) {
	results := make([]TestResult, 0, len(scenarios))

	for _, scenario := range scenarios {
		result, err := r.RunScenario(ctx, scenario)
		if err != nil {
			return results, fmt.Errorf("scenario %q: %w", scenario.Name, err)
		}
		results = append(results, *result)
	}

	return results, nil
}
