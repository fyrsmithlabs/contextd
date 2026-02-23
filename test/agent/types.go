// Package agent provides a synthetic user agent for testing contextd's
// Bayesian confidence system. It simulates realistic user interactions
// including memory recording, searching, feedback, and outcome reporting.
package agent

import (
	"time"
)

// Persona defines the synthetic user's characteristics and goals.
// These influence how the agent interacts with contextd.
type Persona struct {
	// Name identifies this persona for logging
	Name string `json:"name"`

	// Description is passed to the LLM to guide behavior
	Description string `json:"description"`

	// Goals are what the persona is trying to accomplish
	Goals []string `json:"goals"`

	// Constraints limit how the persona behaves
	Constraints []string `json:"constraints"`

	// FeedbackStyle influences how the persona rates memories
	// Options: "generous", "critical", "realistic", "random"
	FeedbackStyle string `json:"feedback_style"`

	// SuccessRate is the probability tasks succeed (0.0-1.0)
	// Used for outcome signal generation
	SuccessRate float64 `json:"success_rate"`
}

// Turn represents a single interaction in a conversation.
type Turn struct {
	Timestamp time.Time  `json:"timestamp"`
	Role      string     `json:"role"` // "user" or "assistant"
	Content   string     `json:"content"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

// ToolCall represents an MCP tool invocation.
type ToolCall struct {
	Name   string                 `json:"name"`
	Args   map[string]interface{} `json:"args"`
	Result interface{}            `json:"result,omitempty"`
	Error  string                 `json:"error,omitempty"`
}

// FeedbackEvent records when feedback was given.
type FeedbackEvent struct {
	Timestamp time.Time `json:"timestamp"`
	MemoryID  string    `json:"memory_id"`
	Helpful   bool      `json:"helpful"`
	Reasoning string    `json:"reasoning,omitempty"`
}

// OutcomeEvent records task outcome signals.
type OutcomeEvent struct {
	Timestamp time.Time `json:"timestamp"`
	MemoryID  string    `json:"memory_id"`
	Succeeded bool      `json:"succeeded"`
	SessionID string    `json:"session_id,omitempty"`
	TaskDesc  string    `json:"task_description,omitempty"`
}

// Session represents a complete test session.
type Session struct {
	ID        string          `json:"id"`
	Persona   Persona         `json:"persona"`
	ProjectID string          `json:"project_id"`
	StartTime time.Time       `json:"start_time"`
	EndTime   time.Time       `json:"end_time"`
	Turns     []Turn          `json:"turns"`
	Feedback  []FeedbackEvent `json:"feedback"`
	Outcomes  []OutcomeEvent  `json:"outcomes"`
	Metrics   SessionMetrics  `json:"metrics"`
}

// SessionMetrics captures test results.
type SessionMetrics struct {
	MemoriesRecorded   int     `json:"memories_recorded"`
	MemoriesRetrieved  int     `json:"memories_retrieved"`
	FeedbackGiven      int     `json:"feedback_given"`
	PositiveFeedback   int     `json:"positive_feedback"`
	OutcomesRecorded   int     `json:"outcomes_recorded"`
	SuccessfulOutcomes int     `json:"successful_outcomes"`
	AvgConfidenceDelta float64 `json:"avg_confidence_delta"`
}

// Scenario defines a test scenario to run.
type Scenario struct {
	// Name identifies the scenario
	Name string `json:"name"`

	// Description explains what this scenario tests
	Description string `json:"description"`

	// Persona to use for this scenario
	Persona Persona `json:"persona"`

	// ProjectID for contextd operations
	ProjectID string `json:"project_id"`

	// MaxTurns limits the conversation length
	MaxTurns int `json:"max_turns"`

	// Actions defines the sequence of actions to take
	// If empty, agent decides autonomously via LLM
	Actions []Action `json:"actions,omitempty"`

	// Assertions to check after scenario completes
	Assertions []Assertion `json:"assertions"`
}

// Action represents a specific action in a scenario.
type Action struct {
	Type string                 `json:"type"` // "record", "search", "feedback", "outcome"
	Args map[string]interface{} `json:"args"`
}

// Assertion defines an expected outcome.
type Assertion struct {
	// Type of assertion
	// Options: "confidence_increased", "confidence_decreased",
	//          "confidence_above", "confidence_below",
	//          "memory_count", "weight_shifted"
	Type string `json:"type"`

	// Target identifies what to check (e.g., memory_id)
	Target string `json:"target,omitempty"`

	// Value for comparison
	Value interface{} `json:"value,omitempty"`

	// Message to show on failure
	Message string `json:"message,omitempty"`
}

// TestResult captures the outcome of running a scenario.
type TestResult struct {
	Scenario   string         `json:"scenario"`
	Passed     bool           `json:"passed"`
	Session    *Session       `json:"session"`
	Assertions []AssertResult `json:"assertions"`
	Error      string         `json:"error,omitempty"`
	Duration   time.Duration  `json:"duration"`
}

// AssertResult captures individual assertion outcomes.
type AssertResult struct {
	Assertion Assertion   `json:"assertion"`
	Passed    bool        `json:"passed"`
	Actual    interface{} `json:"actual,omitempty"`
	Message   string      `json:"message,omitempty"`
}
