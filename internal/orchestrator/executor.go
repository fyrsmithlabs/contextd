package orchestrator

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// PhaseProgress reports progress during execution
type PhaseProgress struct {
	Phase      Phase       `json:"phase"`
	Status     PhaseStatus `json:"status"`
	Message    string      `json:"message"`
	Percentage int         `json:"percentage"`
}

// ProgressCallback receives progress updates during execution
type ProgressCallback func(progress PhaseProgress)

// Executor orchestrates task execution through phases with gates
type Executor struct {
	client           ClaudeClient
	recorder         MemoryRecorder
	handlers         map[Phase]PhaseHandler
	gates            map[Phase][]PhaseGate
	progressCallback ProgressCallback
}

// NewExecutor creates a new executor with the given client and recorder
func NewExecutor(client ClaudeClient, recorder MemoryRecorder) *Executor {
	return &Executor{
		client:   client,
		recorder: recorder,
		handlers: make(map[Phase]PhaseHandler),
		gates:    make(map[Phase][]PhaseGate),
	}
}

// RegisterHandler registers a phase handler
func (e *Executor) RegisterHandler(handler PhaseHandler) {
	e.handlers[handler.Phase()] = handler
}

// RegisterGate registers a gate for a phase
func (e *Executor) RegisterGate(phase Phase, gate PhaseGate) {
	e.gates[phase] = append(e.gates[phase], gate)
}

// OnProgress sets the progress callback
func (e *Executor) OnProgress(callback ProgressCallback) {
	e.progressCallback = callback
}

// Execute runs a task through all phases with gate checks
func (e *Executor) Execute(ctx context.Context, config TaskConfig) (*TaskState, error) {
	state := NewTaskState(config)
	state.Status = StatusInProgress

	phases := AllPhases()
	totalPhases := len(phases)

	for i, phase := range phases {
		// Check context cancellation
		select {
		case <-ctx.Done():
			state.Status = StatusFailed
			return state, ctx.Err()
		default:
		}

		// Report progress start
		percentage := (i * 100) / totalPhases
		e.reportProgress(PhaseProgress{
			Phase:      phase,
			Status:     StatusInProgress,
			Message:    fmt.Sprintf("Starting phase: %s", phase),
			Percentage: percentage,
		})

		// Check gates before executing phase (except for init)
		if phase != PhaseInit {
			violations, err := e.checkGates(ctx, phase, state)
			if err != nil {
				state.Status = StatusFailed
				return state, fmt.Errorf("gate check error for phase %s: %w", phase, err)
			}

			// Record and handle violations
			for _, v := range violations {
				state.Violations = append(state.Violations, v)
				if e.recorder != nil && config.RecordToMemory {
					_ = e.recorder.RecordViolation(ctx, v)
				}
			}

			// Check for critical violations
			if hasCriticalViolation(violations) {
				state.Status = StatusFailed
				return state, fmt.Errorf("critical violation in phase %s: %s", phase, describeViolations(violations))
			}

			// Check for blocking violations when enforcement is enabled
			if config.EnforceTDD && hasBlockingViolation(violations) {
				state.Status = StatusFailed
				return state, fmt.Errorf("gate violation for phase %s: %s", phase, describeViolations(violations))
			}
		}

		// Get handler for phase
		handler, ok := e.handlers[phase]
		if !ok {
			state.Status = StatusFailed
			return state, fmt.Errorf("no handler registered for phase %s", phase)
		}

		// Execute phase
		result, err := handler.Execute(ctx, state)
		if err != nil {
			state.Status = StatusFailed
			state.Results[phase] = &PhaseResult{
				Phase:     phase,
				Status:    StatusFailed,
				StartedAt: time.Now(),
				Error:     err.Error(),
			}
			return state, err
		}

		// Store result and update state
		state.Results[phase] = result
		state.Phase = phase

		// Report progress completion
		e.reportProgress(PhaseProgress{
			Phase:      phase,
			Status:     StatusCompleted,
			Message:    fmt.Sprintf("Completed phase: %s", phase),
			Percentage: ((i + 1) * 100) / totalPhases,
		})
	}

	state.Status = StatusCompleted

	// Record learnings to memory
	if config.RecordToMemory && e.recorder != nil {
		if err := e.recordToMemory(ctx, state); err != nil {
			// Log but don't fail on memory recording error
			e.reportProgress(PhaseProgress{
				Phase:   PhaseReport,
				Status:  StatusCompleted,
				Message: fmt.Sprintf("Warning: failed to record to memory: %v", err),
			})
		}
	}

	return state, nil
}

// checkGates runs all gates for a phase and returns violations
func (e *Executor) checkGates(ctx context.Context, phase Phase, state *TaskState) ([]Violation, error) {
	gates, ok := e.gates[phase]
	if !ok {
		return []Violation{}, nil
	}

	var allViolations []Violation
	for _, gate := range gates {
		violations, err := gate.Check(ctx, state)
		if err != nil {
			return nil, fmt.Errorf("gate %s check failed: %w", gate.Name(), err)
		}
		allViolations = append(allViolations, violations...)
	}

	return allViolations, nil
}

// reportProgress sends progress updates to the callback
func (e *Executor) reportProgress(progress PhaseProgress) {
	if e.progressCallback != nil {
		e.progressCallback(progress)
	}
}

// recordToMemory saves task learnings to contextd memory
func (e *Executor) recordToMemory(ctx context.Context, state *TaskState) error {
	// Build learning content
	var content strings.Builder
	content.WriteString(fmt.Sprintf("Task: %s\n", state.Config.Description))
	content.WriteString(fmt.Sprintf("Status: %s\n", state.Status))

	if len(state.Violations) > 0 {
		content.WriteString("\nViolations encountered:\n")
		for _, v := range state.Violations {
			content.WriteString(fmt.Sprintf("- [%s] %s: %s\n", v.Severity, v.Type, v.Description))
		}
	}

	// Record successful patterns
	for phase, result := range state.Results {
		if result.Status == StatusCompleted {
			content.WriteString(fmt.Sprintf("\nPhase %s completed successfully\n", phase))
			if len(result.Artifacts) > 0 {
				content.WriteString("Artifacts:\n")
				for _, a := range result.Artifacts {
					content.WriteString(fmt.Sprintf("  - %s: %s\n", a.Type, a.Path))
				}
			}
		}
	}

	tags := []string{"orchestrator", "task-execution"}
	if state.Status == StatusCompleted {
		tags = append(tags, "success")
	} else {
		tags = append(tags, "failure")
	}

	return e.recorder.RecordLearning(ctx, content.String(), tags)
}

// hasCriticalViolation checks if any violation is critical
func hasCriticalViolation(violations []Violation) bool {
	for _, v := range violations {
		if v.Severity == SeverityCritical {
			return true
		}
	}
	return false
}

// hasBlockingViolation checks if any violation should block execution
func hasBlockingViolation(violations []Violation) bool {
	for _, v := range violations {
		if v.Severity == SeverityError || v.Severity == SeverityCritical {
			return true
		}
	}
	return false
}

// describeViolations creates a summary of violations
func describeViolations(violations []Violation) string {
	if len(violations) == 0 {
		return ""
	}
	var parts []string
	for _, v := range violations {
		parts = append(parts, fmt.Sprintf("[%s] %s", v.Type, v.Description))
	}
	return strings.Join(parts, "; ")
}
