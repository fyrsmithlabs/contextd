package orchestrator

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTDDGate_Name(t *testing.T) {
	gate := NewTDDGate()
	assert.Equal(t, "tdd-enforcement", gate.Name())
}

func TestTDDGate_Check_NoTestPhaseResult(t *testing.T) {
	gate := NewTDDGate()
	state := NewTaskState(TaskConfig{
		EnforceTDD: true,
	})
	// No test phase result means tests weren't written

	ctx := context.Background()
	violations, err := gate.Check(ctx, state)

	require.NoError(t, err)
	require.Len(t, violations, 1)
	assert.Equal(t, ViolationTDDNotFollowed, violations[0].Type)
	assert.Equal(t, SeverityError, violations[0].Severity)
}

func TestTDDGate_Check_TestPhaseNotCompleted(t *testing.T) {
	gate := NewTDDGate()
	state := NewTaskState(TaskConfig{
		EnforceTDD: true,
	})
	state.Results[PhaseTest] = &PhaseResult{
		Phase:  PhaseTest,
		Status: StatusFailed,
	}

	ctx := context.Background()
	violations, err := gate.Check(ctx, state)

	require.NoError(t, err)
	require.Len(t, violations, 1)
	assert.Equal(t, ViolationTDDNotFollowed, violations[0].Type)
}

func TestTDDGate_Check_NoTestArtifacts(t *testing.T) {
	gate := NewTDDGate()
	state := NewTaskState(TaskConfig{
		EnforceTDD: true,
	})
	state.Results[PhaseTest] = &PhaseResult{
		Phase:       PhaseTest,
		Status:      StatusCompleted,
		CompletedAt: time.Now(),
		Artifacts:   []Artifact{}, // No test files
	}

	ctx := context.Background()
	violations, err := gate.Check(ctx, state)

	require.NoError(t, err)
	require.Len(t, violations, 1)
	assert.Equal(t, ViolationTDDNotFollowed, violations[0].Type)
	assert.Contains(t, violations[0].Description, "no test files")
}

func TestTDDGate_Check_Success(t *testing.T) {
	gate := NewTDDGate()
	state := NewTaskState(TaskConfig{
		EnforceTDD: true,
	})
	state.Results[PhaseTest] = &PhaseResult{
		Phase:       PhaseTest,
		Status:      StatusCompleted,
		CompletedAt: time.Now(),
		Artifacts: []Artifact{
			{Type: ArtifactTypeTestFile, Path: "foo_test.go"},
		},
	}

	ctx := context.Background()
	violations, err := gate.Check(ctx, state)

	require.NoError(t, err)
	assert.Empty(t, violations)
}

func TestTDDGate_Check_TDDNotEnforced(t *testing.T) {
	gate := NewTDDGate()
	state := NewTaskState(TaskConfig{
		EnforceTDD: false, // TDD not enforced
	})
	// No test phase result

	ctx := context.Background()
	violations, err := gate.Check(ctx, state)

	require.NoError(t, err)
	assert.Empty(t, violations, "should not report violations when TDD not enforced")
}

func TestVerificationGate_Name(t *testing.T) {
	gate := NewVerificationGate()
	assert.Equal(t, "verification-gate", gate.Name())
}

func TestVerificationGate_Check_NoVerifyResult(t *testing.T) {
	gate := NewVerificationGate()
	state := NewTaskState(DefaultTaskConfig())
	// No verify phase result

	ctx := context.Background()
	violations, err := gate.Check(ctx, state)

	require.NoError(t, err)
	require.Len(t, violations, 1)
	assert.Equal(t, ViolationTestsNotRun, violations[0].Type)
}

func TestVerificationGate_Check_HelpOutputDetected(t *testing.T) {
	gate := NewVerificationGate()
	state := NewTaskState(DefaultTaskConfig())
	state.Results[PhaseVerify] = &PhaseResult{
		Phase:       PhaseVerify,
		Status:      StatusCompleted,
		CompletedAt: time.Now(),
		Output:      "Usage: go test [flags]\n  -v verbose\n  --help show help",
	}

	ctx := context.Background()
	violations, err := gate.Check(ctx, state)

	require.NoError(t, err)
	require.Len(t, violations, 1)
	assert.Equal(t, ViolationHelpAsVerification, violations[0].Type)
	assert.Equal(t, SeverityCritical, violations[0].Severity)
}

func TestVerificationGate_Check_NoTestOutput(t *testing.T) {
	gate := NewVerificationGate()
	state := NewTaskState(DefaultTaskConfig())
	state.Results[PhaseVerify] = &PhaseResult{
		Phase:       PhaseVerify,
		Status:      StatusCompleted,
		CompletedAt: time.Now(),
		Artifacts:   []Artifact{}, // No test output artifact
	}

	ctx := context.Background()
	violations, err := gate.Check(ctx, state)

	require.NoError(t, err)
	require.Len(t, violations, 1)
	assert.Equal(t, ViolationTestsNotRun, violations[0].Type)
}

func TestVerificationGate_Check_Success(t *testing.T) {
	gate := NewVerificationGate()
	state := NewTaskState(DefaultTaskConfig())
	state.Results[PhaseVerify] = &PhaseResult{
		Phase:       PhaseVerify,
		Status:      StatusCompleted,
		CompletedAt: time.Now(),
		Output:      "=== RUN TestFoo\n--- PASS: TestFoo (0.00s)\nPASS\nok  pkg 0.001s",
		Artifacts: []Artifact{
			{Type: ArtifactTypeTestOutput, Content: "PASS"},
		},
	}

	ctx := context.Background()
	violations, err := gate.Check(ctx, state)

	require.NoError(t, err)
	assert.Empty(t, violations)
}

func TestCommitGate_Name(t *testing.T) {
	gate := NewCommitGate()
	assert.Equal(t, "commit-gate", gate.Name())
}

func TestCommitGate_Check_NoSeparateCommitsWhenRequired(t *testing.T) {
	gate := NewCommitGate()
	state := NewTaskState(TaskConfig{
		RequireSeparateCommits: true,
	})
	state.Results[PhaseCommit] = &PhaseResult{
		Phase:       PhaseCommit,
		Status:      StatusCompleted,
		CompletedAt: time.Now(),
		Artifacts: []Artifact{
			// Only one commit with both test and impl
			{Type: ArtifactTypeImplCommit, CommitID: "abc123"},
		},
	}

	ctx := context.Background()
	violations, err := gate.Check(ctx, state)

	require.NoError(t, err)
	require.Len(t, violations, 1)
	assert.Equal(t, ViolationCommitMixedContent, violations[0].Type)
}

func TestCommitGate_Check_SeparateCommitsSuccess(t *testing.T) {
	gate := NewCommitGate()
	state := NewTaskState(TaskConfig{
		RequireSeparateCommits: true,
	})
	state.Results[PhaseCommit] = &PhaseResult{
		Phase:       PhaseCommit,
		Status:      StatusCompleted,
		CompletedAt: time.Now(),
		Artifacts: []Artifact{
			{Type: ArtifactTypeTestCommit, CommitID: "abc123"},
			{Type: ArtifactTypeImplCommit, CommitID: "def456"},
		},
	}

	ctx := context.Background()
	violations, err := gate.Check(ctx, state)

	require.NoError(t, err)
	assert.Empty(t, violations)
}

func TestCommitGate_Check_NotRequired(t *testing.T) {
	gate := NewCommitGate()
	state := NewTaskState(TaskConfig{
		RequireSeparateCommits: false, // Not required
	})
	state.Results[PhaseCommit] = &PhaseResult{
		Phase:       PhaseCommit,
		Status:      StatusCompleted,
		CompletedAt: time.Now(),
		Artifacts: []Artifact{
			{Type: ArtifactTypeImplCommit, CommitID: "abc123"},
		},
	}

	ctx := context.Background()
	violations, err := gate.Check(ctx, state)

	require.NoError(t, err)
	assert.Empty(t, violations, "should not require separate commits when not configured")
}

func TestSequentialGate_Name(t *testing.T) {
	gate := NewSequentialGate()
	assert.Equal(t, "sequential-gate", gate.Name())
}

func TestSequentialGate_Check_BundledChanges(t *testing.T) {
	gate := NewSequentialGate()
	state := NewTaskState(DefaultTaskConfig())
	// Simulate multiple changes bundled together
	state.Results[PhaseImplement] = &PhaseResult{
		Phase:       PhaseImplement,
		Status:      StatusCompleted,
		CompletedAt: time.Now(),
		Artifacts: []Artifact{
			{Type: ArtifactTypeImplementation, Path: "file1.go"},
			{Type: ArtifactTypeImplementation, Path: "file2.go"},
			{Type: ArtifactTypeImplementation, Path: "file3.go"},
			{Type: ArtifactTypeImplementation, Path: "file4.go"},
			{Type: ArtifactTypeImplementation, Path: "file5.go"},
			{Type: ArtifactTypeImplementation, Path: "file6.go"}, // 6+ files = bundled
		},
	}

	ctx := context.Background()
	violations, err := gate.Check(ctx, state)

	require.NoError(t, err)
	require.Len(t, violations, 1)
	assert.Equal(t, ViolationBundledChanges, violations[0].Type)
	assert.Equal(t, SeverityWarning, violations[0].Severity)
}

func TestSequentialGate_Check_ReasonableChanges(t *testing.T) {
	gate := NewSequentialGate()
	state := NewTaskState(DefaultTaskConfig())
	state.Results[PhaseImplement] = &PhaseResult{
		Phase:       PhaseImplement,
		Status:      StatusCompleted,
		CompletedAt: time.Now(),
		Artifacts: []Artifact{
			{Type: ArtifactTypeImplementation, Path: "file1.go"},
			{Type: ArtifactTypeImplementation, Path: "file2.go"},
		},
	}

	ctx := context.Background()
	violations, err := gate.Check(ctx, state)

	require.NoError(t, err)
	assert.Empty(t, violations)
}

func TestStatusReportGate_Name(t *testing.T) {
	gate := NewStatusReportGate()
	assert.Equal(t, "status-report-gate", gate.Name())
}

func TestStatusReportGate_Check_NoStatusInPhases(t *testing.T) {
	gate := NewStatusReportGate()
	state := NewTaskState(DefaultTaskConfig())

	// Phases without proper status reporting
	for _, phase := range []Phase{PhaseInit, PhaseTest, PhaseImplement} {
		state.Results[phase] = &PhaseResult{
			Phase:       phase,
			Status:      StatusCompleted,
			CompletedAt: time.Now(),
			Output:      "", // Empty output = no status report
		}
	}

	ctx := context.Background()
	violations, err := gate.Check(ctx, state)

	require.NoError(t, err)
	assert.NotEmpty(t, violations)
	assert.Equal(t, ViolationNoStatusReport, violations[0].Type)
}

func TestStatusReportGate_Check_WithStatusReports(t *testing.T) {
	gate := NewStatusReportGate()
	state := NewTaskState(DefaultTaskConfig())

	for _, phase := range []Phase{PhaseInit, PhaseTest, PhaseImplement} {
		state.Results[phase] = &PhaseResult{
			Phase:       phase,
			Status:      StatusCompleted,
			CompletedAt: time.Now(),
			Output:      "Phase completed: gathered context and ready to proceed",
		}
	}

	ctx := context.Background()
	violations, err := gate.Check(ctx, state)

	require.NoError(t, err)
	assert.Empty(t, violations)
}

func TestHelpDetectionPatterns(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		isHelp   bool
	}{
		{
			name:   "go test help",
			output: "Usage: go test [build/test flags] [packages]\n  -v verbose\n  --help show this help",
			isHelp: true,
		},
		{
			name:   "pytest help",
			output: "usage: pytest [options] [file_or_dir]\n  --help, -h     show this help message",
			isHelp: true,
		},
		{
			name:   "actual test output",
			output: "=== RUN TestFoo\n--- PASS: TestFoo (0.00s)\nPASS\nok  pkg 0.001s",
			isHelp: false,
		},
		{
			name:   "npm test output",
			output: "> test\n> jest\n\n PASS  src/test.js\n  Test Suite\n    ✓ should pass (5ms)\n\nTest Suites: 1 passed\nTests:       1 passed",
			isHelp: false,
		},
		{
			name:   "generic help flag",
			output: "mycommand --help\n\nOptions:\n  --help  Show help",
			isHelp: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isHelpOutput(tt.output)
			assert.Equal(t, tt.isHelp, result, "isHelpOutput mismatch for: %s", tt.name)
		})
	}
}
