// Package orchestrator provides agent orchestration with phase gates and workflow enforcement.
// It ensures TDD compliance, sequential processing, and automatic memory recording.
package orchestrator

import (
	"context"
	"fmt"
	"time"
)

// Phase represents a distinct execution phase with required gates
type Phase string

const (
	// PhaseInit initializes the task and gathers context
	PhaseInit Phase = "init"

	// PhaseTest writes tests before implementation (TDD)
	PhaseTest Phase = "test"

	// PhaseImplement implements the solution
	PhaseImplement Phase = "implement"

	// PhaseVerify runs tests and verifies implementation
	PhaseVerify Phase = "verify"

	// PhaseCommit creates separate commits for test and implementation
	PhaseCommit Phase = "commit"

	// PhaseReport reports status and records to memory
	PhaseReport Phase = "report"
)

// AllPhases returns all phases in execution order
func AllPhases() []Phase {
	return []Phase{PhaseInit, PhaseTest, PhaseImplement, PhaseVerify, PhaseCommit, PhaseReport}
}

// PhaseStatus represents the completion status of a phase
type PhaseStatus string

const (
	StatusPending    PhaseStatus = "pending"
	StatusInProgress PhaseStatus = "in_progress"
	StatusCompleted  PhaseStatus = "completed"
	StatusFailed     PhaseStatus = "failed"
	StatusSkipped    PhaseStatus = "skipped"
)

// PhaseResult captures the outcome of a phase execution
type PhaseResult struct {
	Phase       Phase       `json:"phase"`
	Status      PhaseStatus `json:"status"`
	StartedAt   time.Time   `json:"started_at"`
	CompletedAt time.Time   `json:"completed_at,omitempty"`
	Output      string      `json:"output,omitempty"`
	Error       string      `json:"error,omitempty"`
	Artifacts   []Artifact  `json:"artifacts,omitempty"`
}

// Artifact represents a work product from a phase
type Artifact struct {
	Type     ArtifactType `json:"type"`
	Path     string       `json:"path,omitempty"`
	Content  string       `json:"content,omitempty"`
	CommitID string       `json:"commit_id,omitempty"`
}

// ArtifactType categorizes phase outputs
type ArtifactType string

const (
	ArtifactTypeTestFile       ArtifactType = "test_file"
	ArtifactTypeImplementation ArtifactType = "implementation"
	ArtifactTypeTestCommit     ArtifactType = "test_commit"
	ArtifactTypeImplCommit     ArtifactType = "impl_commit"
	ArtifactTypeTestOutput     ArtifactType = "test_output"
)

// TaskConfig configures an orchestrated task
type TaskConfig struct {
	// ID is a unique identifier for this task
	ID string `json:"id"`

	// Description is a human-readable task description
	Description string `json:"description"`

	// WorkingDir is the directory for file operations
	WorkingDir string `json:"working_dir"`

	// EnforceTDD requires test-first development
	EnforceTDD bool `json:"enforce_tdd"`

	// RequireSeparateCommits requires separate test and implementation commits
	RequireSeparateCommits bool `json:"require_separate_commits"`

	// MaxTurns limits API calls per phase
	MaxTurns int `json:"max_turns"`

	// RecordToMemory saves learnings to contextd
	RecordToMemory bool `json:"record_to_memory"`

	// Model specifies the Claude model to use
	Model string `json:"model"`
}

// DefaultTaskConfig returns a configuration with sensible defaults
func DefaultTaskConfig() TaskConfig {
	return TaskConfig{
		EnforceTDD:             true,
		RequireSeparateCommits: true,
		MaxTurns:               20,
		RecordToMemory:         true,
		Model:                  "claude-sonnet-4-5-20250929",
	}
}

// Violation represents a workflow violation detected during execution
type Violation struct {
	Type        ViolationType `json:"type"`
	Phase       Phase         `json:"phase"`
	Description string        `json:"description"`
	Severity    Severity      `json:"severity"`
	DetectedAt  time.Time     `json:"detected_at"`
}

// ViolationType categorizes workflow violations
type ViolationType string

const (
	ViolationTDDNotFollowed      ViolationType = "tdd_not_followed"
	ViolationPhaseSkipped        ViolationType = "phase_skipped"
	ViolationNoStatusReport      ViolationType = "no_status_report"
	ViolationTestsNotRun         ViolationType = "tests_not_run"
	ViolationHelpAsVerification  ViolationType = "help_as_verification"
	ViolationBundledChanges      ViolationType = "bundled_changes"
	ViolationCommitMixedContent  ViolationType = "commit_mixed_content"
)

// Severity indicates how serious a violation is
type Severity string

const (
	SeverityWarning  Severity = "warning"
	SeverityError    Severity = "error"
	SeverityCritical Severity = "critical"
)

// TaskState represents the complete state of a task execution
type TaskState struct {
	Config     TaskConfig              `json:"config"`
	Phase      Phase                   `json:"current_phase"`
	Results    map[Phase]*PhaseResult  `json:"results"`
	Violations []Violation             `json:"violations"`
	StartedAt  time.Time               `json:"started_at"`
	Status     PhaseStatus             `json:"status"`
}

// NewTaskState creates a new task state with the given config
func NewTaskState(config TaskConfig) *TaskState {
	return &TaskState{
		Config:     config,
		Phase:      PhaseInit,
		Results:    make(map[Phase]*PhaseResult),
		Violations: []Violation{},
		StartedAt:  time.Now(),
		Status:     StatusPending,
	}
}

// CanTransition checks if the state can transition to the next phase
func (s *TaskState) CanTransition(next Phase) error {
	phases := AllPhases()
	currentIdx := -1
	nextIdx := -1

	for i, p := range phases {
		if p == s.Phase {
			currentIdx = i
		}
		if p == next {
			nextIdx = i
		}
	}

	if currentIdx == -1 {
		return fmt.Errorf("invalid current phase: %s", s.Phase)
	}
	if nextIdx == -1 {
		return fmt.Errorf("invalid target phase: %s", next)
	}

	// Check sequential order
	if nextIdx != currentIdx+1 {
		return fmt.Errorf("cannot transition from %s to %s: must follow sequential order", s.Phase, next)
	}

	// Check if current phase is completed
	result, ok := s.Results[s.Phase]
	if !ok || result.Status != StatusCompleted {
		return fmt.Errorf("cannot transition: phase %s not completed", s.Phase)
	}

	return nil
}

// PhaseGate defines requirements that must be met before transitioning
type PhaseGate interface {
	// Name returns the gate identifier
	Name() string

	// Check validates gate conditions, returning violations if any
	Check(ctx context.Context, state *TaskState) ([]Violation, error)
}

// PhaseHandler executes the work for a specific phase
type PhaseHandler interface {
	// Phase returns the phase this handler manages
	Phase() Phase

	// Execute runs the phase work and returns artifacts
	Execute(ctx context.Context, state *TaskState) (*PhaseResult, error)
}

// MemoryRecorder records learnings to contextd
type MemoryRecorder interface {
	// RecordLearning saves a learning to memory
	RecordLearning(ctx context.Context, content string, tags []string) error

	// RecordViolation records a workflow violation
	RecordViolation(ctx context.Context, violation Violation) error
}

// ClaudeClient abstracts Claude API interactions
type ClaudeClient interface {
	// SendMessage sends a message and returns the response
	SendMessage(ctx context.Context, messages []Message, tools []Tool) (*Response, error)
}

// Message represents a conversation message
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Tool represents a tool available to Claude
type Tool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema interface{} `json:"input_schema"`
}

// Response represents a Claude API response
type Response struct {
	Content    string      `json:"content"`
	ToolCalls  []ToolCall  `json:"tool_calls,omitempty"`
	StopReason string      `json:"stop_reason"`
}

// ToolCall represents a tool invocation request
type ToolCall struct {
	ID    string                 `json:"id"`
	Name  string                 `json:"name"`
	Input map[string]interface{} `json:"input"`
}
