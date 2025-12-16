// Package framework provides the integration test framework for contextd.
package framework

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBinaryAssertions validates binary (pass/fail) assertion types.
func TestBinaryAssertions(t *testing.T) {
	t.Run("tool_called assertion passes when tool was called", func(t *testing.T) {
		sessionResult := &SessionResult{
			Developer: DeveloperConfig{ID: "dev-a"},
			MemoryIDs: []string{"mem-1"},
		}

		assertion := BinaryAssertion{
			Check:  "tool_called",
			Method: "record_memory",
			Target: "memory_record",
		}

		result := EvaluateBinaryAssertion(assertion, sessionResult)
		assert.True(t, result.Passed)
	})

	t.Run("tool_called assertion fails when tool not called", func(t *testing.T) {
		sessionResult := &SessionResult{
			Developer: DeveloperConfig{ID: "dev-a"},
			MemoryIDs: []string{}, // No memory recorded
		}

		assertion := BinaryAssertion{
			Check:  "tool_called",
			Method: "record_memory",
			Target: "memory_record",
		}

		result := EvaluateBinaryAssertion(assertion, sessionResult)
		assert.False(t, result.Passed)
	})

	t.Run("search_has_results assertion passes with results", func(t *testing.T) {
		sessionResult := &SessionResult{
			SearchResults: [][]MemoryResult{
				{{ID: "mem-1", Title: "Test memory"}},
			},
		}

		assertion := BinaryAssertion{
			Check:  "search_has_results",
			Method: "memory_search",
			Target: "",
		}

		result := EvaluateBinaryAssertion(assertion, sessionResult)
		assert.True(t, result.Passed)
	})

	t.Run("search_has_results assertion fails without results", func(t *testing.T) {
		sessionResult := &SessionResult{
			SearchResults: [][]MemoryResult{{}}, // Empty results
		}

		assertion := BinaryAssertion{
			Check:  "search_has_results",
			Method: "memory_search",
			Target: "",
		}

		result := EvaluateBinaryAssertion(assertion, sessionResult)
		assert.False(t, result.Passed)
	})
}

// TestThresholdAssertions validates threshold-based assertions.
func TestThresholdAssertions(t *testing.T) {
	t.Run("confidence_threshold passes when confidence >= threshold", func(t *testing.T) {
		sessionResult := &SessionResult{
			SearchResults: [][]MemoryResult{
				{{ID: "mem-1", Confidence: 0.85}},
			},
		}

		assertion := ThresholdAssertion{
			Check:     "confidence",
			Method:    "first_result",
			Threshold: 0.7,
			Operator:  ">=",
		}

		result := EvaluateThresholdAssertion(assertion, sessionResult)
		assert.True(t, result.Passed)
		assert.Equal(t, 0.85, result.ActualValue)
	})

	t.Run("confidence_threshold fails when confidence < threshold", func(t *testing.T) {
		sessionResult := &SessionResult{
			SearchResults: [][]MemoryResult{
				{{ID: "mem-1", Confidence: 0.5}},
			},
		}

		assertion := ThresholdAssertion{
			Check:     "confidence",
			Method:    "first_result",
			Threshold: 0.7,
			Operator:  ">=",
		}

		result := EvaluateThresholdAssertion(assertion, sessionResult)
		assert.False(t, result.Passed)
		assert.Equal(t, 0.5, result.ActualValue)
	})

	t.Run("result_count_threshold passes when count >= minimum", func(t *testing.T) {
		sessionResult := &SessionResult{
			SearchResults: [][]MemoryResult{
				{
					{ID: "mem-1"},
					{ID: "mem-2"},
					{ID: "mem-3"},
				},
			},
		}

		assertion := ThresholdAssertion{
			Check:     "result_count",
			Method:    "latest_search",
			Threshold: 2,
			Operator:  ">=",
		}

		result := EvaluateThresholdAssertion(assertion, sessionResult)
		assert.True(t, result.Passed)
		assert.Equal(t, 3.0, result.ActualValue)
	})

	t.Run("handles operator variations", func(t *testing.T) {
		sessionResult := &SessionResult{
			SearchResults: [][]MemoryResult{
				{{ID: "mem-1", Confidence: 0.5}},
			},
		}

		// Test < operator
		ltAssertion := ThresholdAssertion{
			Check:     "confidence",
			Method:    "first_result",
			Threshold: 0.7,
			Operator:  "<",
		}
		ltResult := EvaluateThresholdAssertion(ltAssertion, sessionResult)
		assert.True(t, ltResult.Passed, "0.5 < 0.7 should pass")

		// Test == operator
		eqAssertion := ThresholdAssertion{
			Check:     "confidence",
			Method:    "first_result",
			Threshold: 0.5,
			Operator:  "==",
		}
		eqResult := EvaluateThresholdAssertion(eqAssertion, sessionResult)
		assert.True(t, eqResult.Passed, "0.5 == 0.5 should pass")
	})
}

// TestBehavioralAssertions validates behavioral (pattern-based) assertions.
func TestBehavioralAssertions(t *testing.T) {
	t.Run("regex_match passes when pattern found in content", func(t *testing.T) {
		sessionResult := &SessionResult{
			SearchResults: [][]MemoryResult{
				{{Content: "Always check if user is nil before accessing user.ID"}},
			},
		}

		assertion := BehavioralAssertion{
			Check:    "content_pattern",
			Method:   "regex_match",
			Patterns: []string{`nil`, `check.*before`},
		}

		result := EvaluateBehavioralAssertion(assertion, sessionResult)
		assert.True(t, result.Passed)
	})

	t.Run("regex_match fails when pattern not found", func(t *testing.T) {
		sessionResult := &SessionResult{
			SearchResults: [][]MemoryResult{
				{{Content: "Use connection pooling"}},
			},
		}

		assertion := BehavioralAssertion{
			Check:    "content_pattern",
			Method:   "regex_match",
			Patterns: []string{`nil`, `null`},
		}

		result := EvaluateBehavioralAssertion(assertion, sessionResult)
		assert.False(t, result.Passed)
	})

	t.Run("negative_pattern fails when pattern found", func(t *testing.T) {
		sessionResult := &SessionResult{
			SearchResults: [][]MemoryResult{
				{{Content: "API_KEY=sk-1234567890"}},
			},
		}

		assertion := BehavioralAssertion{
			Check:            "content_pattern",
			Method:           "regex_match",
			NegativePatterns: []string{`API_KEY`, `sk-[a-zA-Z0-9]+`},
		}

		result := EvaluateBehavioralAssertion(assertion, sessionResult)
		assert.False(t, result.Passed, "Should fail when negative pattern is found (secrets)")
	})

	t.Run("negative_pattern passes when pattern not found", func(t *testing.T) {
		sessionResult := &SessionResult{
			SearchResults: [][]MemoryResult{
				{{Content: "Use environment variables for configuration"}},
			},
		}

		assertion := BehavioralAssertion{
			Check:            "content_pattern",
			Method:           "regex_match",
			NegativePatterns: []string{`API_KEY`, `sk-[a-zA-Z0-9]+`},
		}

		result := EvaluateBehavioralAssertion(assertion, sessionResult)
		assert.True(t, result.Passed, "Should pass when no secrets found")
	})
}

// TestAssertionSet validates running multiple assertions.
func TestAssertionSet(t *testing.T) {
	t.Run("all assertions pass", func(t *testing.T) {
		sessionResult := &SessionResult{
			MemoryIDs: []string{"mem-1"},
			SearchResults: [][]MemoryResult{
				{{ID: "mem-1", Confidence: 0.85, Content: "null check fix"}},
			},
		}

		assertions := AssertionSet{
			Binary: []BinaryAssertion{
				{Check: "tool_called", Method: "record_memory"},
				{Check: "search_has_results", Method: "memory_search"},
			},
			Threshold: []ThresholdAssertion{
				{Check: "confidence", Method: "first_result", Threshold: 0.7, Operator: ">="},
			},
			Behavioral: []BehavioralAssertion{
				{Check: "content_pattern", Method: "regex_match", Patterns: []string{`null`}},
			},
		}

		results := EvaluateAssertionSet(assertions, sessionResult)
		require.Len(t, results, 4)
		assert.True(t, AllPassed(results))
	})

	t.Run("reports failures correctly", func(t *testing.T) {
		sessionResult := &SessionResult{
			MemoryIDs:     []string{}, // No memory recorded
			SearchResults: [][]MemoryResult{},
		}

		assertions := AssertionSet{
			Binary: []BinaryAssertion{
				{Check: "tool_called", Method: "record_memory"},
			},
		}

		results := EvaluateAssertionSet(assertions, sessionResult)
		require.Len(t, results, 1)
		assert.False(t, AllPassed(results))
	})
}
