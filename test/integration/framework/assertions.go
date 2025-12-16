// Package framework provides the integration test harness for contextd.
package framework

import (
	"fmt"
	"regexp"
)

// BinaryAssertion is a pass/fail assertion.
type BinaryAssertion struct {
	Check  string // What to check: "tool_called", "search_has_results", "file_exists"
	Method string // How to check: "record_memory", "memory_search", etc.
	Target string // Specific target if needed
}

// ThresholdAssertion is a comparison-based assertion.
type ThresholdAssertion struct {
	Check     string  // What to check: "confidence", "result_count", "latency"
	Method    string  // How to check: "first_result", "latest_search", etc.
	Threshold float64 // Value to compare against
	Operator  string  // Comparison operator: ">", "<", ">=", "<=", "=="
}

// BehavioralAssertion is a pattern-based assertion.
type BehavioralAssertion struct {
	Check            string   // What to check: "content_pattern", "ast_pattern"
	Method           string   // How to check: "regex_match", "ast_pattern_match"
	Patterns         []string // Patterns that SHOULD be present
	NegativePatterns []string // Patterns that should NOT be present
	LLMJudge         bool     // Use LLM to evaluate if automated check insufficient
}

// AssertionSet groups multiple assertions for a test.
type AssertionSet struct {
	Binary     []BinaryAssertion
	Threshold  []ThresholdAssertion
	Behavioral []BehavioralAssertion
}

// AssertionResult contains the result of evaluating an assertion.
type AssertionResult struct {
	Type        string  // "binary", "threshold", "behavioral"
	Check       string  // What was checked
	Passed      bool    // Whether assertion passed
	Message     string  // Human-readable explanation
	ActualValue float64 // For threshold assertions
}

// EvaluateBinaryAssertion evaluates a binary assertion against session results.
func EvaluateBinaryAssertion(assertion BinaryAssertion, result *SessionResult) AssertionResult {
	ar := AssertionResult{
		Type:  "binary",
		Check: assertion.Check,
	}

	switch assertion.Check {
	case "tool_called":
		switch assertion.Method {
		case "record_memory":
			ar.Passed = len(result.MemoryIDs) > 0
			if ar.Passed {
				ar.Message = fmt.Sprintf("record_memory was called, created %d memories", len(result.MemoryIDs))
			} else {
				ar.Message = "record_memory was not called"
			}
		case "search_memory", "memory_search":
			ar.Passed = len(result.SearchResults) > 0
			if ar.Passed {
				ar.Message = fmt.Sprintf("search_memory was called %d times", len(result.SearchResults))
			} else {
				ar.Message = "search_memory was not called"
			}
		default:
			ar.Passed = false
			ar.Message = fmt.Sprintf("unknown method: %s", assertion.Method)
		}

	case "search_has_results":
		if len(result.SearchResults) == 0 {
			ar.Passed = false
			ar.Message = "no search was performed"
		} else {
			// Check the most recent search
			lastSearch := result.SearchResults[len(result.SearchResults)-1]
			ar.Passed = len(lastSearch) > 0
			if ar.Passed {
				ar.Message = fmt.Sprintf("search returned %d results", len(lastSearch))
			} else {
				ar.Message = "search returned no results"
			}
		}

	case "no_errors":
		ar.Passed = len(result.Errors) == 0
		if ar.Passed {
			ar.Message = "no errors occurred"
		} else {
			ar.Message = fmt.Sprintf("%d errors occurred: %v", len(result.Errors), result.Errors)
		}

	default:
		ar.Passed = false
		ar.Message = fmt.Sprintf("unknown check type: %s", assertion.Check)
	}

	return ar
}

// EvaluateThresholdAssertion evaluates a threshold assertion against session results.
func EvaluateThresholdAssertion(assertion ThresholdAssertion, result *SessionResult) AssertionResult {
	ar := AssertionResult{
		Type:  "threshold",
		Check: assertion.Check,
	}

	var value float64

	switch assertion.Check {
	case "confidence":
		if len(result.SearchResults) == 0 || len(result.SearchResults[0]) == 0 {
			ar.Passed = false
			ar.Message = "no search results to evaluate confidence"
			return ar
		}

		switch assertion.Method {
		case "first_result":
			value = result.SearchResults[0][0].Confidence
		case "latest_search":
			lastSearch := result.SearchResults[len(result.SearchResults)-1]
			if len(lastSearch) > 0 {
				value = lastSearch[0].Confidence
			}
		case "average":
			var sum float64
			var count int
			for _, search := range result.SearchResults {
				for _, r := range search {
					sum += r.Confidence
					count++
				}
			}
			if count > 0 {
				value = sum / float64(count)
			}
		default:
			value = result.SearchResults[0][0].Confidence
		}

	case "result_count":
		if len(result.SearchResults) == 0 {
			ar.Passed = false
			ar.Message = "no search results to count"
			return ar
		}

		switch assertion.Method {
		case "first_search":
			value = float64(len(result.SearchResults[0]))
		case "latest_search":
			lastSearch := result.SearchResults[len(result.SearchResults)-1]
			value = float64(len(lastSearch))
		case "total":
			var total int
			for _, search := range result.SearchResults {
				total += len(search)
			}
			value = float64(total)
		default:
			lastSearch := result.SearchResults[len(result.SearchResults)-1]
			value = float64(len(lastSearch))
		}

	default:
		ar.Passed = false
		ar.Message = fmt.Sprintf("unknown check type: %s", assertion.Check)
		return ar
	}

	ar.ActualValue = value

	// Evaluate comparison
	switch assertion.Operator {
	case ">":
		ar.Passed = value > assertion.Threshold
	case "<":
		ar.Passed = value < assertion.Threshold
	case ">=":
		ar.Passed = value >= assertion.Threshold
	case "<=":
		ar.Passed = value <= assertion.Threshold
	case "==":
		ar.Passed = value == assertion.Threshold
	default:
		ar.Passed = value >= assertion.Threshold // Default to >=
	}

	if ar.Passed {
		ar.Message = fmt.Sprintf("%s: %.2f %s %.2f", assertion.Check, value, assertion.Operator, assertion.Threshold)
	} else {
		ar.Message = fmt.Sprintf("%s FAILED: %.2f %s %.2f", assertion.Check, value, assertion.Operator, assertion.Threshold)
	}

	return ar
}

// EvaluateBehavioralAssertion evaluates a behavioral assertion against session results.
func EvaluateBehavioralAssertion(assertion BehavioralAssertion, result *SessionResult) AssertionResult {
	ar := AssertionResult{
		Type:  "behavioral",
		Check: assertion.Check,
	}

	// Collect all content to search
	var contents []string
	for _, search := range result.SearchResults {
		for _, r := range search {
			contents = append(contents, r.Content)
			contents = append(contents, r.Title)
		}
	}

	if len(contents) == 0 {
		ar.Passed = false
		ar.Message = "no content to evaluate"
		return ar
	}

	switch assertion.Method {
	case "regex_match":
		// Check positive patterns (at least one must match)
		positivesPassed := len(assertion.Patterns) == 0 // Pass if no patterns required
		var matchedPositives []string
		for _, pattern := range assertion.Patterns {
			re, err := regexp.Compile(pattern)
			if err != nil {
				ar.Passed = false
				ar.Message = fmt.Sprintf("invalid regex pattern: %s", pattern)
				return ar
			}

			for _, content := range contents {
				if re.MatchString(content) {
					positivesPassed = true
					matchedPositives = append(matchedPositives, pattern)
					break
				}
			}
		}

		// Check negative patterns (none should match)
		negativesFailed := false
		var matchedNegatives []string
		for _, pattern := range assertion.NegativePatterns {
			re, err := regexp.Compile(pattern)
			if err != nil {
				ar.Passed = false
				ar.Message = fmt.Sprintf("invalid regex pattern: %s", pattern)
				return ar
			}

			for _, content := range contents {
				if re.MatchString(content) {
					negativesFailed = true
					matchedNegatives = append(matchedNegatives, pattern)
					break
				}
			}
		}

		ar.Passed = positivesPassed && !negativesFailed

		if ar.Passed {
			if len(matchedPositives) > 0 {
				ar.Message = fmt.Sprintf("matched patterns: %v", matchedPositives)
			} else {
				ar.Message = "no negative patterns found (passed)"
			}
		} else {
			if !positivesPassed {
				ar.Message = fmt.Sprintf("required patterns not found: %v", assertion.Patterns)
			} else if negativesFailed {
				ar.Message = fmt.Sprintf("forbidden patterns found: %v", matchedNegatives)
			}
		}

	default:
		ar.Passed = false
		ar.Message = fmt.Sprintf("unknown method: %s", assertion.Method)
	}

	return ar
}

// EvaluateAssertionSet evaluates all assertions in a set.
func EvaluateAssertionSet(set AssertionSet, result *SessionResult) []AssertionResult {
	var results []AssertionResult

	for _, a := range set.Binary {
		results = append(results, EvaluateBinaryAssertion(a, result))
	}

	for _, a := range set.Threshold {
		results = append(results, EvaluateThresholdAssertion(a, result))
	}

	for _, a := range set.Behavioral {
		results = append(results, EvaluateBehavioralAssertion(a, result))
	}

	return results
}

// AllPassed returns true if all assertion results passed.
func AllPassed(results []AssertionResult) bool {
	for _, r := range results {
		if !r.Passed {
			return false
		}
	}
	return true
}

// FailedAssertions returns only the failed assertions.
func FailedAssertions(results []AssertionResult) []AssertionResult {
	var failed []AssertionResult
	for _, r := range results {
		if !r.Passed {
			failed = append(failed, r)
		}
	}
	return failed
}

// SummaryMessage creates a human-readable summary of assertion results.
func SummaryMessage(results []AssertionResult) string {
	passed := 0
	failed := 0
	for _, r := range results {
		if r.Passed {
			passed++
		} else {
			failed++
		}
	}

	if failed == 0 {
		return fmt.Sprintf("All %d assertions passed", passed)
	}

	return fmt.Sprintf("%d/%d assertions passed, %d failed", passed, passed+failed, failed)
}
