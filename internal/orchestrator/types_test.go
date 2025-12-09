package orchestrator

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAllPhases(t *testing.T) {
	phases := AllPhases()

	require.Len(t, phases, 6, "should have 6 phases")
	assert.Equal(t, PhaseInit, phases[0], "init should be first")
	assert.Equal(t, PhaseTest, phases[1], "test should be second")
	assert.Equal(t, PhaseImplement, phases[2], "implement should be third")
	assert.Equal(t, PhaseVerify, phases[3], "verify should be fourth")
	assert.Equal(t, PhaseCommit, phases[4], "commit should be fifth")
	assert.Equal(t, PhaseReport, phases[5], "report should be last")
}

func TestDefaultTaskConfig(t *testing.T) {
	config := DefaultTaskConfig()

	assert.True(t, config.EnforceTDD, "TDD should be enforced by default")
	assert.True(t, config.RequireSeparateCommits, "separate commits should be required")
	assert.Equal(t, 20, config.MaxTurns, "max turns should be 20")
	assert.True(t, config.RecordToMemory, "memory recording should be enabled")
	assert.NotEmpty(t, config.Model, "model should have a default value")
}

func TestNewTaskState(t *testing.T) {
	config := TaskConfig{
		ID:          "test-task-1",
		Description: "Test task",
	}

	state := NewTaskState(config)

	assert.Equal(t, config.ID, state.Config.ID)
	assert.Equal(t, PhaseInit, state.Phase, "should start at init phase")
	assert.NotNil(t, state.Results, "results map should be initialized")
	assert.Empty(t, state.Violations, "violations should be empty")
	assert.Equal(t, StatusPending, state.Status, "status should be pending")
	assert.False(t, state.StartedAt.IsZero(), "started_at should be set")
}

func TestTaskState_CanTransition_Success(t *testing.T) {
	config := DefaultTaskConfig()
	state := NewTaskState(config)

	// Complete init phase
	state.Results[PhaseInit] = &PhaseResult{
		Phase:       PhaseInit,
		Status:      StatusCompleted,
		CompletedAt: time.Now(),
	}

	err := state.CanTransition(PhaseTest)
	assert.NoError(t, err, "should allow transition from init to test")
}

func TestTaskState_CanTransition_NotCompleted(t *testing.T) {
	config := DefaultTaskConfig()
	state := NewTaskState(config)

	// Init phase not completed
	err := state.CanTransition(PhaseTest)
	assert.Error(t, err, "should not allow transition when phase not completed")
	assert.Contains(t, err.Error(), "not completed")
}

func TestTaskState_CanTransition_SkipPhase(t *testing.T) {
	config := DefaultTaskConfig()
	state := NewTaskState(config)

	// Complete init phase
	state.Results[PhaseInit] = &PhaseResult{
		Phase:       PhaseInit,
		Status:      StatusCompleted,
		CompletedAt: time.Now(),
	}

	// Try to skip to implement (skipping test)
	err := state.CanTransition(PhaseImplement)
	assert.Error(t, err, "should not allow skipping phases")
	assert.Contains(t, err.Error(), "sequential order")
}

func TestTaskState_CanTransition_InvalidPhase(t *testing.T) {
	config := DefaultTaskConfig()
	state := NewTaskState(config)

	err := state.CanTransition(Phase("invalid"))
	assert.Error(t, err, "should reject invalid phase")
	assert.Contains(t, err.Error(), "invalid target phase")
}

func TestPhaseResult_Fields(t *testing.T) {
	now := time.Now()
	result := PhaseResult{
		Phase:       PhaseTest,
		Status:      StatusCompleted,
		StartedAt:   now,
		CompletedAt: now.Add(time.Minute),
		Output:      "tests passed",
		Artifacts: []Artifact{
			{Type: ArtifactTypeTestFile, Path: "foo_test.go"},
		},
	}

	assert.Equal(t, PhaseTest, result.Phase)
	assert.Equal(t, StatusCompleted, result.Status)
	assert.Len(t, result.Artifacts, 1)
	assert.Equal(t, ArtifactTypeTestFile, result.Artifacts[0].Type)
}

func TestViolation_Fields(t *testing.T) {
	violation := Violation{
		Type:        ViolationTDDNotFollowed,
		Phase:       PhaseImplement,
		Description: "implementation without tests",
		Severity:    SeverityError,
		DetectedAt:  time.Now(),
	}

	assert.Equal(t, ViolationTDDNotFollowed, violation.Type)
	assert.Equal(t, PhaseImplement, violation.Phase)
	assert.Equal(t, SeverityError, violation.Severity)
	assert.NotEmpty(t, violation.Description)
}

func TestViolationTypes(t *testing.T) {
	// Ensure all violation types are distinct
	types := []ViolationType{
		ViolationTDDNotFollowed,
		ViolationPhaseSkipped,
		ViolationNoStatusReport,
		ViolationTestsNotRun,
		ViolationHelpAsVerification,
		ViolationBundledChanges,
		ViolationCommitMixedContent,
	}

	seen := make(map[ViolationType]bool)
	for _, vt := range types {
		assert.False(t, seen[vt], "violation type %s should be unique", vt)
		seen[vt] = true
	}
}

func TestArtifactTypes(t *testing.T) {
	types := []ArtifactType{
		ArtifactTypeTestFile,
		ArtifactTypeImplementation,
		ArtifactTypeTestCommit,
		ArtifactTypeImplCommit,
		ArtifactTypeTestOutput,
	}

	seen := make(map[ArtifactType]bool)
	for _, at := range types {
		assert.False(t, seen[at], "artifact type %s should be unique", at)
		seen[at] = true
	}
}

func TestSeverityLevels(t *testing.T) {
	assert.Equal(t, Severity("warning"), SeverityWarning)
	assert.Equal(t, Severity("error"), SeverityError)
	assert.Equal(t, Severity("critical"), SeverityCritical)
}
