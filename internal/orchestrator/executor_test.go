package orchestrator

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockClaudeClient is a mock implementation of ClaudeClient
type MockClaudeClient struct {
	mock.Mock
}

func (m *MockClaudeClient) SendMessage(ctx context.Context, messages []Message, tools []Tool) (*Response, error) {
	args := m.Called(ctx, messages, tools)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Response), args.Error(1)
}

// MockMemoryRecorder is a mock implementation of MemoryRecorder
type MockMemoryRecorder struct {
	mock.Mock
}

func (m *MockMemoryRecorder) RecordLearning(ctx context.Context, content string, tags []string) error {
	args := m.Called(ctx, content, tags)
	return args.Error(0)
}

func (m *MockMemoryRecorder) RecordViolation(ctx context.Context, violation Violation) error {
	args := m.Called(ctx, violation)
	return args.Error(0)
}

// MockPhaseHandler is a mock implementation of PhaseHandler
type MockPhaseHandler struct {
	mock.Mock
	phase Phase
}

func NewMockPhaseHandler(phase Phase) *MockPhaseHandler {
	return &MockPhaseHandler{phase: phase}
}

func (m *MockPhaseHandler) Phase() Phase {
	return m.phase
}

func (m *MockPhaseHandler) Execute(ctx context.Context, state *TaskState) (*PhaseResult, error) {
	args := m.Called(ctx, state)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*PhaseResult), args.Error(1)
}

// MockPhaseGate is a mock implementation of PhaseGate
type MockPhaseGate struct {
	mock.Mock
	name string
}

func NewMockPhaseGate(name string) *MockPhaseGate {
	return &MockPhaseGate{name: name}
}

func (m *MockPhaseGate) Name() string {
	return m.name
}

func (m *MockPhaseGate) Check(ctx context.Context, state *TaskState) ([]Violation, error) {
	args := m.Called(ctx, state)
	return args.Get(0).([]Violation), args.Error(1)
}

func TestNewExecutor(t *testing.T) {
	client := &MockClaudeClient{}
	recorder := &MockMemoryRecorder{}

	executor := NewExecutor(client, recorder)

	require.NotNil(t, executor)
	assert.NotNil(t, executor.handlers)
	assert.NotNil(t, executor.gates)
}

func TestExecutor_RegisterHandler(t *testing.T) {
	executor := NewExecutor(&MockClaudeClient{}, &MockMemoryRecorder{})
	handler := NewMockPhaseHandler(PhaseTest)

	executor.RegisterHandler(handler)

	assert.Len(t, executor.handlers, 1)
	assert.Equal(t, handler, executor.handlers[PhaseTest])
}

func TestExecutor_RegisterGate(t *testing.T) {
	executor := NewExecutor(&MockClaudeClient{}, &MockMemoryRecorder{})
	gate := NewMockPhaseGate("tdd-gate")

	executor.RegisterGate(PhaseImplement, gate)

	assert.Len(t, executor.gates[PhaseImplement], 1)
}

func TestExecutor_Execute_Success(t *testing.T) {
	client := &MockClaudeClient{}
	recorder := &MockMemoryRecorder{}
	executor := NewExecutor(client, recorder)

	// Register handlers for all phases
	for _, phase := range AllPhases() {
		handler := NewMockPhaseHandler(phase)
		handler.On("Execute", mock.Anything, mock.Anything).Return(&PhaseResult{
			Phase:       phase,
			Status:      StatusCompleted,
			StartedAt:   time.Now(),
			CompletedAt: time.Now(),
		}, nil)
		executor.RegisterHandler(handler)
	}

	// Setup recorder expectations
	recorder.On("RecordLearning", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	config := TaskConfig{
		ID:             "test-task",
		Description:    "Test task execution",
		RecordToMemory: true,
	}

	ctx := context.Background()
	state, err := executor.Execute(ctx, config)

	require.NoError(t, err)
	require.NotNil(t, state)
	assert.Equal(t, PhaseReport, state.Phase)
	assert.Equal(t, StatusCompleted, state.Status)
	assert.Len(t, state.Results, 6)
}

func TestExecutor_Execute_GateViolation(t *testing.T) {
	client := &MockClaudeClient{}
	recorder := &MockMemoryRecorder{}
	executor := NewExecutor(client, recorder)

	// Register init handler
	initHandler := NewMockPhaseHandler(PhaseInit)
	initHandler.On("Execute", mock.Anything, mock.Anything).Return(&PhaseResult{
		Phase:       PhaseInit,
		Status:      StatusCompleted,
		StartedAt:   time.Now(),
		CompletedAt: time.Now(),
	}, nil)
	executor.RegisterHandler(initHandler)

	// Register gate that fails
	gate := NewMockPhaseGate("tdd-gate")
	gate.On("Check", mock.Anything, mock.Anything).Return([]Violation{
		{
			Type:        ViolationTDDNotFollowed,
			Phase:       PhaseTest,
			Description: "No tests found",
			Severity:    SeverityError,
		},
	}, nil)
	executor.RegisterGate(PhaseTest, gate)

	// Register test handler (should not be called due to gate)
	testHandler := NewMockPhaseHandler(PhaseTest)
	executor.RegisterHandler(testHandler)

	// Recorder should record violation
	recorder.On("RecordViolation", mock.Anything, mock.Anything).Return(nil)

	config := TaskConfig{
		ID:          "test-task",
		Description: "Test with gate violation",
		EnforceTDD:  true,
	}

	ctx := context.Background()
	state, err := executor.Execute(ctx, config)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "gate violation")
	assert.NotEmpty(t, state.Violations)
}

func TestExecutor_Execute_HandlerError(t *testing.T) {
	client := &MockClaudeClient{}
	recorder := &MockMemoryRecorder{}
	executor := NewExecutor(client, recorder)

	// Register handler that fails
	handler := NewMockPhaseHandler(PhaseInit)
	handler.On("Execute", mock.Anything, mock.Anything).Return(nil, errors.New("handler failed"))
	executor.RegisterHandler(handler)

	config := TaskConfig{
		ID:          "test-task",
		Description: "Test with handler error",
	}

	ctx := context.Background()
	state, err := executor.Execute(ctx, config)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "handler failed")
	assert.Equal(t, StatusFailed, state.Status)
}

func TestExecutor_Execute_ContextCancellation(t *testing.T) {
	client := &MockClaudeClient{}
	recorder := &MockMemoryRecorder{}
	executor := NewExecutor(client, recorder)

	// Register handler that blocks
	handler := NewMockPhaseHandler(PhaseInit)
	handler.On("Execute", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		ctx := args.Get(0).(context.Context)
		<-ctx.Done()
	}).Return(nil, context.Canceled)
	executor.RegisterHandler(handler)

	config := TaskConfig{
		ID:          "test-task",
		Description: "Test with context cancellation",
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel after a short delay
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	state, err := executor.Execute(ctx, config)

	require.Error(t, err)
	assert.True(t, errors.Is(err, context.Canceled))
	assert.Equal(t, StatusFailed, state.Status)
}

func TestExecutor_Execute_MissingHandler(t *testing.T) {
	client := &MockClaudeClient{}
	recorder := &MockMemoryRecorder{}
	executor := NewExecutor(client, recorder)

	// Don't register any handlers

	config := TaskConfig{
		ID:          "test-task",
		Description: "Test with missing handler",
	}

	ctx := context.Background()
	state, err := executor.Execute(ctx, config)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "no handler")
	assert.Equal(t, StatusFailed, state.Status)
}

func TestExecutor_CheckGates(t *testing.T) {
	executor := NewExecutor(&MockClaudeClient{}, &MockMemoryRecorder{})

	gate1 := NewMockPhaseGate("gate1")
	gate1.On("Check", mock.Anything, mock.Anything).Return([]Violation{}, nil)

	gate2 := NewMockPhaseGate("gate2")
	gate2.On("Check", mock.Anything, mock.Anything).Return([]Violation{
		{Type: ViolationTDDNotFollowed, Severity: SeverityWarning},
	}, nil)

	executor.RegisterGate(PhaseTest, gate1)
	executor.RegisterGate(PhaseTest, gate2)

	state := NewTaskState(DefaultTaskConfig())
	ctx := context.Background()

	violations, err := executor.checkGates(ctx, PhaseTest, state)

	require.NoError(t, err)
	assert.Len(t, violations, 1)
	gate1.AssertExpectations(t)
	gate2.AssertExpectations(t)
}

func TestExecutor_RecordMemory(t *testing.T) {
	client := &MockClaudeClient{}
	recorder := &MockMemoryRecorder{}

	recorder.On("RecordLearning", mock.Anything, mock.MatchedBy(func(content string) bool {
		return len(content) > 0
	}), mock.Anything).Return(nil)

	executor := NewExecutor(client, recorder)

	state := NewTaskState(TaskConfig{
		ID:             "test",
		RecordToMemory: true,
	})
	state.Status = StatusCompleted

	ctx := context.Background()
	err := executor.recordToMemory(ctx, state)

	require.NoError(t, err)
	recorder.AssertExpectations(t)
}

func TestExecutor_ProgressCallback(t *testing.T) {
	executor := NewExecutor(&MockClaudeClient{}, &MockMemoryRecorder{})

	var progressUpdates []PhaseProgress
	executor.OnProgress(func(progress PhaseProgress) {
		progressUpdates = append(progressUpdates, progress)
	})

	// Simulate progress updates
	executor.reportProgress(PhaseProgress{
		Phase:      PhaseInit,
		Status:     StatusInProgress,
		Message:    "Starting",
		Percentage: 0,
	})

	executor.reportProgress(PhaseProgress{
		Phase:      PhaseInit,
		Status:     StatusCompleted,
		Message:    "Done",
		Percentage: 100,
	})

	assert.Len(t, progressUpdates, 2)
	assert.Equal(t, PhaseInit, progressUpdates[0].Phase)
	assert.Equal(t, StatusInProgress, progressUpdates[0].Status)
	assert.Equal(t, StatusCompleted, progressUpdates[1].Status)
}

func TestExecutor_CriticalViolationStopsExecution(t *testing.T) {
	client := &MockClaudeClient{}
	recorder := &MockMemoryRecorder{}
	executor := NewExecutor(client, recorder)

	// Register init handler
	initHandler := NewMockPhaseHandler(PhaseInit)
	initHandler.On("Execute", mock.Anything, mock.Anything).Return(&PhaseResult{
		Phase:       PhaseInit,
		Status:      StatusCompleted,
		StartedAt:   time.Now(),
		CompletedAt: time.Now(),
	}, nil)
	executor.RegisterHandler(initHandler)

	// Register gate with critical violation
	gate := NewMockPhaseGate("critical-gate")
	gate.On("Check", mock.Anything, mock.Anything).Return([]Violation{
		{
			Type:        ViolationTDDNotFollowed,
			Phase:       PhaseTest,
			Description: "Critical: No tests",
			Severity:    SeverityCritical,
		},
	}, nil)
	executor.RegisterGate(PhaseTest, gate)

	recorder.On("RecordViolation", mock.Anything, mock.Anything).Return(nil)

	config := TaskConfig{
		ID:          "test-task",
		Description: "Test with critical violation",
		EnforceTDD:  true,
	}

	ctx := context.Background()
	state, err := executor.Execute(ctx, config)

	require.Error(t, err)
	assert.Equal(t, StatusFailed, state.Status)
	assert.Contains(t, err.Error(), "critical violation")
}
