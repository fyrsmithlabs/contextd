package orchestrator

import (
	"context"
	"regexp"
	"strings"
	"time"
)

// TDDGate enforces test-driven development workflow
type TDDGate struct{}

// NewTDDGate creates a new TDD enforcement gate
func NewTDDGate() *TDDGate {
	return &TDDGate{}
}

// Name returns the gate identifier
func (g *TDDGate) Name() string {
	return "tdd-enforcement"
}

// Check validates TDD compliance
func (g *TDDGate) Check(ctx context.Context, state *TaskState) ([]Violation, error) {
	// Skip if TDD not enforced
	if !state.Config.EnforceTDD {
		return []Violation{}, nil
	}

	var violations []Violation

	// Check if test phase exists and completed
	testResult, ok := state.Results[PhaseTest]
	if !ok {
		violations = append(violations, Violation{
			Type:        ViolationTDDNotFollowed,
			Phase:       PhaseImplement,
			Description: "test phase was not executed before implementation",
			Severity:    SeverityError,
			DetectedAt:  time.Now(),
		})
		return violations, nil
	}

	if testResult.Status != StatusCompleted {
		violations = append(violations, Violation{
			Type:        ViolationTDDNotFollowed,
			Phase:       PhaseImplement,
			Description: "test phase did not complete successfully",
			Severity:    SeverityError,
			DetectedAt:  time.Now(),
		})
		return violations, nil
	}

	// Check for test file artifacts
	hasTestFiles := false
	for _, artifact := range testResult.Artifacts {
		if artifact.Type == ArtifactTypeTestFile {
			hasTestFiles = true
			break
		}
	}

	if !hasTestFiles {
		violations = append(violations, Violation{
			Type:        ViolationTDDNotFollowed,
			Phase:       PhaseImplement,
			Description: "no test files created in test phase",
			Severity:    SeverityError,
			DetectedAt:  time.Now(),
		})
	}

	return violations, nil
}

// VerificationGate ensures tests actually run (not just help output)
type VerificationGate struct{}

// NewVerificationGate creates a new verification gate
func NewVerificationGate() *VerificationGate {
	return &VerificationGate{}
}

// Name returns the gate identifier
func (g *VerificationGate) Name() string {
	return "verification-gate"
}

// Check validates test verification was performed
func (g *VerificationGate) Check(ctx context.Context, state *TaskState) ([]Violation, error) {
	var violations []Violation

	verifyResult, ok := state.Results[PhaseVerify]
	if !ok {
		violations = append(violations, Violation{
			Type:        ViolationTestsNotRun,
			Phase:       PhaseCommit,
			Description: "verify phase was not executed",
			Severity:    SeverityError,
			DetectedAt:  time.Now(),
		})
		return violations, nil
	}

	// Check for help output being used as verification
	if isHelpOutput(verifyResult.Output) {
		violations = append(violations, Violation{
			Type:        ViolationHelpAsVerification,
			Phase:       PhaseVerify,
			Description: "detected --help output used as test verification instead of actual test run",
			Severity:    SeverityCritical,
			DetectedAt:  time.Now(),
		})
		return violations, nil
	}

	// Check for test output artifact
	hasTestOutput := false
	for _, artifact := range verifyResult.Artifacts {
		if artifact.Type == ArtifactTypeTestOutput {
			hasTestOutput = true
			break
		}
	}

	if !hasTestOutput {
		violations = append(violations, Violation{
			Type:        ViolationTestsNotRun,
			Phase:       PhaseVerify,
			Description: "no test output artifact found in verify phase",
			Severity:    SeverityError,
			DetectedAt:  time.Now(),
		})
	}

	return violations, nil
}

// CommitGate ensures commits follow the required pattern
type CommitGate struct{}

// NewCommitGate creates a new commit gate
func NewCommitGate() *CommitGate {
	return &CommitGate{}
}

// Name returns the gate identifier
func (g *CommitGate) Name() string {
	return "commit-gate"
}

// Check validates commit structure
func (g *CommitGate) Check(ctx context.Context, state *TaskState) ([]Violation, error) {
	// Skip if separate commits not required
	if !state.Config.RequireSeparateCommits {
		return []Violation{}, nil
	}

	var violations []Violation

	commitResult, ok := state.Results[PhaseCommit]
	if !ok {
		return violations, nil // Will be caught by other gates
	}

	// Count commit types
	hasTestCommit := false
	hasImplCommit := false

	for _, artifact := range commitResult.Artifacts {
		if artifact.Type == ArtifactTypeTestCommit {
			hasTestCommit = true
		}
		if artifact.Type == ArtifactTypeImplCommit {
			hasImplCommit = true
		}
	}

	// If we have implementation but no separate test commit
	if hasImplCommit && !hasTestCommit {
		violations = append(violations, Violation{
			Type:        ViolationCommitMixedContent,
			Phase:       PhaseCommit,
			Description: "test and implementation should be in separate commits",
			Severity:    SeverityError,
			DetectedAt:  time.Now(),
		})
	}

	return violations, nil
}

// SequentialGate prevents bundled changes
type SequentialGate struct {
	maxFilesPerPhase int
}

// NewSequentialGate creates a new sequential processing gate
func NewSequentialGate() *SequentialGate {
	return &SequentialGate{
		maxFilesPerPhase: 5, // Warning threshold
	}
}

// Name returns the gate identifier
func (g *SequentialGate) Name() string {
	return "sequential-gate"
}

// Check validates sequential processing
func (g *SequentialGate) Check(ctx context.Context, state *TaskState) ([]Violation, error) {
	var violations []Violation

	implResult, ok := state.Results[PhaseImplement]
	if !ok {
		return violations, nil
	}

	// Count implementation files
	implCount := 0
	for _, artifact := range implResult.Artifacts {
		if artifact.Type == ArtifactTypeImplementation {
			implCount++
		}
	}

	if implCount > g.maxFilesPerPhase {
		violations = append(violations, Violation{
			Type:        ViolationBundledChanges,
			Phase:       PhaseImplement,
			Description: "too many files modified in single phase; consider breaking into smaller changes",
			Severity:    SeverityWarning,
			DetectedAt:  time.Now(),
		})
	}

	return violations, nil
}

// StatusReportGate ensures phases report their status
type StatusReportGate struct{}

// NewStatusReportGate creates a new status report gate
func NewStatusReportGate() *StatusReportGate {
	return &StatusReportGate{}
}

// Name returns the gate identifier
func (g *StatusReportGate) Name() string {
	return "status-report-gate"
}

// Check validates status reporting
func (g *StatusReportGate) Check(ctx context.Context, state *TaskState) ([]Violation, error) {
	var violations []Violation

	// Check key phases have output
	checkPhases := []Phase{PhaseInit, PhaseTest, PhaseImplement}

	for _, phase := range checkPhases {
		result, ok := state.Results[phase]
		if !ok {
			continue // Phase not executed yet
		}

		if result.Status == StatusCompleted && result.Output == "" {
			violations = append(violations, Violation{
				Type:        ViolationNoStatusReport,
				Phase:       phase,
				Description: "phase completed without status report",
				Severity:    SeverityWarning,
				DetectedAt:  time.Now(),
			})
			break // Only report once
		}
	}

	return violations, nil
}

// isHelpOutput detects if output looks like --help output rather than test results
func isHelpOutput(output string) bool {
	if output == "" {
		return false
	}

	lower := strings.ToLower(output)

	// Help patterns to detect
	helpPatterns := []string{
		"usage:",
		"--help",
		"-h, --help",
		"show help",
		"show this help",
		"options:",
	}

	// Test result patterns that indicate real tests ran
	testPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)(pass|fail|error).*\d+`),           // "PASS", "1 passed"
		regexp.MustCompile(`(?i)test.*\([\d.]+s\)`),                // "TestFoo (0.00s)"
		regexp.MustCompile(`(?i)✓|✗`),                              // Checkmarks
		regexp.MustCompile(`(?i)ok\s+\S+\s+[\d.]+s`),               // "ok pkg 0.001s"
		regexp.MustCompile(`(?i)test suites?:\s*\d+`),              // "Test Suites: 1"
	}

	// If it looks like real test output, it's not help
	for _, pattern := range testPatterns {
		if pattern.MatchString(output) {
			return false
		}
	}

	// Check for help patterns
	helpCount := 0
	for _, pattern := range helpPatterns {
		if strings.Contains(lower, pattern) {
			helpCount++
		}
	}

	// If multiple help indicators, likely help output
	return helpCount >= 2
}
