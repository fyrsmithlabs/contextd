package agent

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestAgent_RecordAndSearch(t *testing.T) {
	ctx := context.Background()
	client := NewMockContextdClient()

	agent, err := New(Config{
		Client: client,
		Persona: Persona{
			Name:          "TestUser",
			Description:   "A test user",
			FeedbackStyle: "realistic",
			SuccessRate:   0.7,
		},
		ProjectID: "test-project",
		Logger:    zap.NewNop(),
	})
	require.NoError(t, err)

	// Record a memory
	memoryID, err := agent.RecordMemory(ctx, "Test Pattern", "Use this pattern for testing", "success", []string{"test"})
	require.NoError(t, err)
	assert.NotEmpty(t, memoryID)

	// Search for it
	results, err := agent.SearchMemories(ctx, "testing", 5)
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, memoryID, results[0].ID)
}

func TestAgent_FeedbackAffectsConfidence(t *testing.T) {
	ctx := context.Background()
	client := NewMockContextdClient()

	agent, err := New(Config{
		Client: client,
		Persona: Persona{
			Name:          "FeedbackTester",
			Description:   "Tests feedback",
			FeedbackStyle: "critical",
			SuccessRate:   0.5,
		},
		ProjectID: "test-project",
		Logger:    zap.NewNop(),
	})
	require.NoError(t, err)

	// Record a memory
	memoryID, err := agent.RecordMemory(ctx, "Test Memory", "Content", "success", nil)
	require.NoError(t, err)

	// Get initial confidence (0.8 for explicit record)
	initialHistory := agent.GetConfidenceHistory(memoryID)
	require.Len(t, initialHistory, 1)
	t.Logf("Initial confidence: %.4f", initialHistory[0])

	// Give multiple positive feedbacks to build up confidence
	var conf float64
	for i := 0; i < 5; i++ {
		conf, err = agent.GiveFeedback(ctx, memoryID, true, "helpful")
		require.NoError(t, err)
	}
	t.Logf("After 5 positive feedbacks: %.4f", conf)

	// With Bayesian system: prior (1,1) + 5 positive explicit (weight 0.5 each)
	// alpha = 1 + 5*0.5 = 3.5, beta = 1
	// confidence = 3.5 / 4.5 = 0.778
	assert.Greater(t, conf, 0.7, "5 positive feedbacks should keep confidence above 0.7")

	// Give negative feedback
	confAfterNeg, err := agent.GiveFeedback(ctx, memoryID, false, "not helpful")
	require.NoError(t, err)
	t.Logf("After 1 negative: %.4f", confAfterNeg)

	// Should decrease slightly but still be reasonably high
	assert.Less(t, confAfterNeg, conf, "Negative feedback should decrease confidence")

	// Multiple feedbacks should show in history
	history := agent.GetConfidenceHistory(memoryID)
	assert.Len(t, history, 7) // initial + 5 positive + 1 negative
}

func TestAgent_OutcomeSignals(t *testing.T) {
	ctx := context.Background()
	client := NewMockContextdClient()

	agent, err := New(Config{
		Client: client,
		Persona: Persona{
			Name:          "OutcomeTester",
			Description:   "Tests outcomes",
			FeedbackStyle: "realistic",
			SuccessRate:   0.8,
		},
		ProjectID: "test-project",
		Logger:    zap.NewNop(),
	})
	require.NoError(t, err)

	// Record a memory
	memoryID, err := agent.RecordMemory(ctx, "Outcome Test", "Content", "success", nil)
	require.NoError(t, err)

	initialHistory := agent.GetConfidenceHistory(memoryID)
	t.Logf("Initial confidence: %.4f", initialHistory[0])

	// Report positive outcomes
	var lastConf float64
	for i := 0; i < 5; i++ {
		lastConf, err = agent.ReportOutcome(ctx, memoryID, true, "Task succeeded")
		require.NoError(t, err)
	}

	history := agent.GetConfidenceHistory(memoryID)
	assert.Len(t, history, 6) // initial + 5 outcomes
	t.Logf("After 5 positive outcomes: %.4f", lastConf)

	// With Bayesian system: prior (1,1) + 5 positive outcomes (weight 0.3 each)
	// alpha = 1 + 5*0.3 = 2.5, beta = 1
	// confidence = 2.5 / 3.5 = 0.714
	// Initial is 0.8 (explicit record), but Bayesian recomputes from signals
	// So we check that confidence is above the Bayesian prior (0.5)
	assert.Greater(t, lastConf, 0.5, "Multiple positive outcomes should keep confidence above prior")
	assert.Greater(t, lastConf, 0.6, "5 positive outcomes should give confidence > 0.6")
}

func TestRunner_ScriptedScenario(t *testing.T) {
	ctx := context.Background()
	client := NewMockContextdClient()

	runner, err := NewRunner(RunnerConfig{
		Client: client,
		Logger: zap.NewNop(),
	})
	require.NoError(t, err)

	scenario := Scenario{
		Name:        "test_scenario",
		Description: "A test scenario",
		Persona: Persona{
			Name:          "ScenarioTester",
			Description:   "Tests scenarios",
			FeedbackStyle: "generous",
			SuccessRate:   0.9,
		},
		ProjectID: "test-project",
		MaxTurns:  10,
		Actions: []Action{
			{
				Type: "record",
				Args: map[string]interface{}{
					"title":   "Test Memory",
					"content": "Test content for scenario",
					"outcome": "success",
					"tags":    []string{"test"},
				},
			},
			{
				Type: "search",
				Args: map[string]interface{}{
					"query": "test",
					"limit": 5,
				},
			},
			{
				Type: "feedback",
				Args: map[string]interface{}{
					"memory_id": "last",
					"helpful":   true,
					"reasoning": "Very helpful",
				},
			},
		},
		Assertions: []Assertion{
			{
				Type:    "memory_count",
				Value:   1,
				Message: "Should have one memory",
			},
			{
				Type:    "feedback_count",
				Value:   1,
				Message: "Should have one feedback",
			},
		},
	}

	result, err := runner.RunScenario(ctx, scenario)
	require.NoError(t, err)

	assert.True(t, result.Passed, "Scenario should pass")
	assert.Len(t, result.Assertions, 2)

	for _, ar := range result.Assertions {
		assert.True(t, ar.Passed, "Assertion should pass: %s", ar.Assertion.Message)
	}
}

func TestMockClient_BayesianBehavior(t *testing.T) {
	ctx := context.Background()
	client := NewMockContextdClient()

	// Record a memory
	memoryID, initialConf, err := client.MemoryRecord(ctx, "test-project", "Test", "Content", "success", nil)
	require.NoError(t, err)
	t.Logf("Initial confidence: %.4f", initialConf)

	// Give 5 positive feedbacks
	var conf float64
	for i := 0; i < 5; i++ {
		conf, err = client.MemoryFeedback(ctx, memoryID, true)
		require.NoError(t, err)
	}
	t.Logf("After 5 positive: %.4f", conf)

	// Should be well above 0.5 (the prior)
	assert.Greater(t, conf, 0.6, "5 positive signals should increase confidence above 0.6")

	// Now give 10 negative feedbacks
	for i := 0; i < 10; i++ {
		conf, err = client.MemoryFeedback(ctx, memoryID, false)
		require.NoError(t, err)
	}
	t.Logf("After 10 negative: %.4f", conf)

	// Should have decreased
	assert.Less(t, conf, 0.5, "10 negative signals should push confidence below 0.5")
}
