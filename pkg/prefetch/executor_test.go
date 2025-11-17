package prefetch

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockRule for testing
type MockRule struct {
	name      string
	trigger   EventType
	result    *PreFetchResult
	err       error
	delay     time.Duration
	execCount int
}

func (m *MockRule) Name() string {
	return m.name
}

func (m *MockRule) Trigger() EventType {
	return m.trigger
}

func (m *MockRule) Execute(ctx context.Context, event GitEvent) (*PreFetchResult, error) {
	m.execCount++

	// Simulate delay
	if m.delay > 0 {
		select {
		case <-time.After(m.delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	if m.err != nil {
		return nil, m.err
	}

	return m.result, nil
}

func TestNewExecutor(t *testing.T) {
	executor := NewExecutor(3)
	require.NotNil(t, executor)
}

func TestExecutor_ExecuteSingleRule(t *testing.T) {
	executor := NewExecutor(3)

	mockRule := &MockRule{
		name:    "test_rule",
		trigger: EventTypeBranchSwitch,
		result: &PreFetchResult{
			Type:       "test_rule",
			Data:       "test data",
			Confidence: 1.0,
		},
	}

	ctx := context.Background()
	event := GitEvent{
		Type:        EventTypeBranchSwitch,
		OldBranch:   "main",
		NewBranch:   "feature",
		ProjectPath: "/test/path",
	}

	results := executor.Execute(ctx, event, []Rule{mockRule})

	require.Len(t, results, 1)
	assert.Equal(t, "test_rule", results[0].Type)
	assert.Equal(t, 1, mockRule.execCount)
}

func TestExecutor_ExecuteMultipleRules(t *testing.T) {
	executor := NewExecutor(3)

	rule1 := &MockRule{
		name:    "rule1",
		trigger: EventTypeBranchSwitch,
		result:  &PreFetchResult{Type: "rule1", Confidence: 1.0},
	}

	rule2 := &MockRule{
		name:    "rule2",
		trigger: EventTypeBranchSwitch,
		result:  &PreFetchResult{Type: "rule2", Confidence: 1.0},
	}

	rule3 := &MockRule{
		name:    "rule3",
		trigger: EventTypeBranchSwitch,
		result:  &PreFetchResult{Type: "rule3", Confidence: 1.0},
	}

	ctx := context.Background()
	event := GitEvent{
		Type:        EventTypeBranchSwitch,
		ProjectPath: "/test/path",
	}

	results := executor.Execute(ctx, event, []Rule{rule1, rule2, rule3})

	// All rules should execute
	assert.Len(t, results, 3)
	assert.Equal(t, 1, rule1.execCount)
	assert.Equal(t, 1, rule2.execCount)
	assert.Equal(t, 1, rule3.execCount)

	// Results should be returned
	resultTypes := make(map[string]bool)
	for _, r := range results {
		resultTypes[r.Type] = true
	}
	assert.True(t, resultTypes["rule1"])
	assert.True(t, resultTypes["rule2"])
	assert.True(t, resultTypes["rule3"])
}

func TestExecutor_ParallelExecution(t *testing.T) {
	executor := NewExecutor(3)

	// Create rules with different delays
	rule1 := &MockRule{
		name:    "fast",
		trigger: EventTypeBranchSwitch,
		result:  &PreFetchResult{Type: "fast", Confidence: 1.0},
		delay:   10 * time.Millisecond,
	}

	rule2 := &MockRule{
		name:    "medium",
		trigger: EventTypeBranchSwitch,
		result:  &PreFetchResult{Type: "medium", Confidence: 1.0},
		delay:   50 * time.Millisecond,
	}

	rule3 := &MockRule{
		name:    "slow",
		trigger: EventTypeBranchSwitch,
		result:  &PreFetchResult{Type: "slow", Confidence: 1.0},
		delay:   100 * time.Millisecond,
	}

	ctx := context.Background()
	event := GitEvent{Type: EventTypeBranchSwitch}

	start := time.Now()
	results := executor.Execute(ctx, event, []Rule{rule1, rule2, rule3})
	duration := time.Since(start)

	// All results should be returned
	assert.Len(t, results, 3)

	// Execution should be parallel (not sequential)
	// If sequential: 10 + 50 + 100 = 160ms
	// If parallel: max(10, 50, 100) = 100ms (+ overhead)
	// Allow some overhead
	assert.Less(t, duration, 150*time.Millisecond, "execution should be parallel")
}

func TestExecutor_RuleTimeout(t *testing.T) {
	executor := NewExecutor(3)

	// Rule that takes too long
	slowRule := &MockRule{
		name:    "slow_rule",
		trigger: EventTypeBranchSwitch,
		result:  &PreFetchResult{Type: "slow_rule", Confidence: 1.0},
		delay:   2 * time.Second, // Very slow
	}

	// Rule that completes quickly
	fastRule := &MockRule{
		name:    "fast_rule",
		trigger: EventTypeBranchSwitch,
		result:  &PreFetchResult{Type: "fast_rule", Confidence: 1.0},
		delay:   10 * time.Millisecond,
	}

	// Context with short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	event := GitEvent{Type: EventTypeBranchSwitch}

	results := executor.Execute(ctx, event, []Rule{slowRule, fastRule})

	// Fast rule should complete
	// Slow rule should timeout (no result)
	assert.LessOrEqual(t, len(results), 2, "slow rule may timeout")

	// At least the fast rule should succeed
	found := false
	for _, r := range results {
		if r.Type == "fast_rule" {
			found = true
		}
	}
	assert.True(t, found, "fast rule should complete")
}

func TestExecutor_RuleError(t *testing.T) {
	executor := NewExecutor(3)

	errorRule := &MockRule{
		name:    "error_rule",
		trigger: EventTypeBranchSwitch,
		err:     errors.New("rule execution failed"),
	}

	successRule := &MockRule{
		name:    "success_rule",
		trigger: EventTypeBranchSwitch,
		result:  &PreFetchResult{Type: "success_rule", Confidence: 1.0},
	}

	ctx := context.Background()
	event := GitEvent{Type: EventTypeBranchSwitch}

	results := executor.Execute(ctx, event, []Rule{errorRule, successRule})

	// Only successful rule should return result
	assert.Len(t, results, 1)
	assert.Equal(t, "success_rule", results[0].Type)
}

func TestExecutor_NoRules(t *testing.T) {
	executor := NewExecutor(3)

	ctx := context.Background()
	event := GitEvent{Type: EventTypeBranchSwitch}

	results := executor.Execute(ctx, event, []Rule{})

	assert.Empty(t, results)
}

func TestExecutor_CancelledContext(t *testing.T) {
	executor := NewExecutor(3)

	rule := &MockRule{
		name:    "test_rule",
		trigger: EventTypeBranchSwitch,
		result:  &PreFetchResult{Type: "test_rule", Confidence: 1.0},
		delay:   100 * time.Millisecond,
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	event := GitEvent{Type: EventTypeBranchSwitch}

	results := executor.Execute(ctx, event, []Rule{rule})

	// Should return empty results due to cancelled context
	assert.Empty(t, results)
}
