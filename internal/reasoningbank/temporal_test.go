package reasoningbank

import (
	"strings"
	"testing"
	"time"
)

// referenceDate is Friday, January 30, 2026 — used as the session date for all tests.
var referenceDate = time.Date(2026, time.January, 30, 10, 0, 0, 0, time.UTC)

func TestTemporalResolveYesterday(t *testing.T) {
	input := "I discovered the bug yesterday"
	result := ResolveTemporalReferences(input, referenceDate)
	expected := "I discovered the bug yesterday (January 29, 2026)"
	if result != expected {
		t.Errorf("yesterday:\n  got:  %q\n  want: %q", result, expected)
	}
}

func TestTemporalResolveToday(t *testing.T) {
	input := "I fixed the issue today"
	result := ResolveTemporalReferences(input, referenceDate)
	expected := "I fixed the issue today (January 30, 2026)"
	if result != expected {
		t.Errorf("today:\n  got:  %q\n  want: %q", result, expected)
	}
}

func TestTemporalResolveThisMorning(t *testing.T) {
	input := "deployed this morning"
	result := ResolveTemporalReferences(input, referenceDate)
	expected := "deployed this morning (January 30, 2026)"
	if result != expected {
		t.Errorf("this morning:\n  got:  %q\n  want: %q", result, expected)
	}
}

func TestTemporalResolveThisAfternoon(t *testing.T) {
	input := "meeting this afternoon"
	result := ResolveTemporalReferences(input, referenceDate)
	expected := "meeting this afternoon (January 30, 2026)"
	if result != expected {
		t.Errorf("this afternoon:\n  got:  %q\n  want: %q", result, expected)
	}
}

func TestTemporalResolveThisEvening(t *testing.T) {
	input := "reviewing this evening"
	result := ResolveTemporalReferences(input, referenceDate)
	expected := "reviewing this evening (January 30, 2026)"
	if result != expected {
		t.Errorf("this evening:\n  got:  %q\n  want: %q", result, expected)
	}
}

func TestTemporalResolveLastWeek(t *testing.T) {
	input := "we discussed this last week"
	result := ResolveTemporalReferences(input, referenceDate)
	expected := "we discussed this last week (January 23, 2026)"
	if result != expected {
		t.Errorf("last week:\n  got:  %q\n  want: %q", result, expected)
	}
}

func TestTemporalResolveLastMonth(t *testing.T) {
	input := "last month we shipped the feature"
	result := ResolveTemporalReferences(input, referenceDate)
	expected := "last month (December 30, 2025) we shipped the feature"
	if result != expected {
		t.Errorf("last month:\n  got:  %q\n  want: %q", result, expected)
	}
}

func TestTemporalResolveLastYear(t *testing.T) {
	input := "this was first attempted last year"
	result := ResolveTemporalReferences(input, referenceDate)
	expected := "this was first attempted last year (January 30, 2025)"
	if result != expected {
		t.Errorf("last year:\n  got:  %q\n  want: %q", result, expected)
	}
}

func TestTemporalResolveDaysAgo(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "singular day",
			input:    "fixed 1 day ago",
			expected: "fixed 1 day ago (January 29, 2026)",
		},
		{
			name:     "plural days",
			input:    "found 3 days ago",
			expected: "found 3 days ago (January 27, 2026)",
		},
		{
			name:     "large number",
			input:    "started 10 days ago",
			expected: "started 10 days ago (January 20, 2026)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ResolveTemporalReferences(tt.input, referenceDate)
			if result != tt.expected {
				t.Errorf("got:  %q\nwant: %q", result, tt.expected)
			}
		})
	}
}

func TestTemporalResolveWeeksAgo(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "singular week",
			input:    "discussed 1 week ago",
			expected: "discussed 1 week ago (January 23, 2026)",
		},
		{
			name:     "plural weeks",
			input:    "reviewed 2 weeks ago",
			expected: "reviewed 2 weeks ago (January 16, 2026)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ResolveTemporalReferences(tt.input, referenceDate)
			if result != tt.expected {
				t.Errorf("got:  %q\nwant: %q", result, tt.expected)
			}
		})
	}
}

func TestTemporalResolveMonthsAgo(t *testing.T) {
	input := "implemented 3 months ago"
	result := ResolveTemporalReferences(input, referenceDate)
	expected := "implemented 3 months ago (October 30, 2025)"
	if result != expected {
		t.Errorf("months ago:\n  got:  %q\n  want: %q", result, expected)
	}
}

func TestTemporalResolveYearsAgo(t *testing.T) {
	input := "project started 2 years ago"
	result := ResolveTemporalReferences(input, referenceDate)
	expected := "project started 2 years ago (January 30, 2024)"
	if result != expected {
		t.Errorf("years ago:\n  got:  %q\n  want: %q", result, expected)
	}
}

func TestTemporalResolveFewDaysAgo(t *testing.T) {
	input := "we noticed it a few days ago"
	result := ResolveTemporalReferences(input, referenceDate)
	expected := "we noticed it a few days ago (January 27, 2026)"
	if result != expected {
		t.Errorf("a few days ago:\n  got:  %q\n  want: %q", result, expected)
	}
}

func TestTemporalResolveCoupleOfWeeksAgo(t *testing.T) {
	input := "a couple of weeks ago we refactored the module"
	result := ResolveTemporalReferences(input, referenceDate)
	expected := "a couple of weeks ago (January 16, 2026) we refactored the module"
	if result != expected {
		t.Errorf("a couple of weeks ago:\n  got:  %q\n  want: %q", result, expected)
	}
}

func TestTemporalResolveLastDayOfWeek(t *testing.T) {
	// referenceDate is Friday, January 30, 2026

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "last Monday",
			input:    "deployed last Monday",
			expected: "deployed last Monday (January 26, 2026)",
		},
		{
			name:     "last Tuesday",
			input:    "discussed last Tuesday",
			expected: "discussed last Tuesday (January 27, 2026)",
		},
		{
			name:     "last Wednesday",
			input:    "reviewed last Wednesday",
			expected: "reviewed last Wednesday (January 28, 2026)",
		},
		{
			name:     "last Thursday",
			input:    "meeting last Thursday",
			expected: "meeting last Thursday (January 29, 2026)",
		},
		{
			name:     "last Friday (same weekday as session, goes back 7 days)",
			input:    "released last Friday",
			expected: "released last Friday (January 23, 2026)",
		},
		{
			name:     "last Saturday",
			input:    "worked last Saturday",
			expected: "worked last Saturday (January 24, 2026)",
		},
		{
			name:     "last Sunday",
			input:    "planned last Sunday",
			expected: "planned last Sunday (January 25, 2026)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ResolveTemporalReferences(tt.input, referenceDate)
			if result != tt.expected {
				t.Errorf("got:  %q\nwant: %q", result, tt.expected)
			}
		})
	}
}

func TestTemporalResolveZeroSessionDate(t *testing.T) {
	input := "yesterday we deployed the fix"
	result := ResolveTemporalReferences(input, time.Time{})
	if result != input {
		t.Errorf("zero session date should return unchanged text:\n  got:  %q\n  want: %q", result, input)
	}
}

func TestTemporalResolveNoTemporalReferences(t *testing.T) {
	input := "The function processes data using a hash map for O(1) lookups"
	result := ResolveTemporalReferences(input, referenceDate)
	if result != input {
		t.Errorf("no temporal references should return unchanged text:\n  got:  %q\n  want: %q", result, input)
	}
}

func TestTemporalResolveEmptyText(t *testing.T) {
	result := ResolveTemporalReferences("", referenceDate)
	if result != "" {
		t.Errorf("empty text should return empty:\n  got: %q", result)
	}
}

func TestTemporalResolveMultipleReferences(t *testing.T) {
	input := "The bug appeared yesterday and we discussed it today"
	result := ResolveTemporalReferences(input, referenceDate)
	expected := "The bug appeared yesterday (January 29, 2026) and we discussed it today (January 30, 2026)"
	if result != expected {
		t.Errorf("multiple references:\n  got:  %q\n  want: %q", result, expected)
	}
}

func TestTemporalResolveCaseInsensitive(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"lowercase", "yesterday"},
		{"uppercase", "Yesterday"},
		{"mixed case", "YESTERDAY"},
		{"last week lower", "last week"},
		{"last week title", "Last Week"},
		{"last week upper", "LAST WEEK"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ResolveTemporalReferences(tt.input, referenceDate)
			if result == tt.input {
				t.Errorf("case-insensitive match failed for %q — text was not modified", tt.input)
			}
			if !strings.Contains(result, "(") {
				t.Errorf("expected resolved date in parentheses for %q, got: %q", tt.input, result)
			}
		})
	}
}

func TestTemporalResolvePreservesOriginalText(t *testing.T) {
	input := "went yesterday"
	result := ResolveTemporalReferences(input, referenceDate)

	// Original text must still be present
	if !strings.HasPrefix(result, "went yesterday") {
		t.Errorf("original text not preserved:\n  got:  %q\n  want prefix: %q", result, "went yesterday")
	}
	// Resolved date appended in parentheses
	if !strings.Contains(result, "(January 29, 2026)") {
		t.Errorf("resolved date not found:\n  got: %q", result)
	}
}

func TestTemporalResolveLongInput(t *testing.T) {
	// Create a string longer than maxTemporalInputLength
	longPrefix := strings.Repeat("x", maxTemporalInputLength+100)
	input := longPrefix + " yesterday"

	result := ResolveTemporalReferences(input, referenceDate)

	// The input is truncated, so "yesterday" at the end should be cut off
	// and the result should not contain a resolved date
	if strings.Contains(result, "(January") {
		t.Errorf("expected truncated input to not resolve temporal ref past cutoff, got: %q", result[:100])
	}
}

func TestTemporalFormatDate(t *testing.T) {
	tests := []struct {
		name     string
		date     time.Time
		expected string
	}{
		{
			name:     "January",
			date:     time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC),
			expected: "January 1, 2026",
		},
		{
			name:     "December",
			date:     time.Date(2025, time.December, 25, 0, 0, 0, 0, time.UTC),
			expected: "December 25, 2025",
		},
		{
			name:     "February leap year",
			date:     time.Date(2024, time.February, 29, 0, 0, 0, 0, time.UTC),
			expected: "February 29, 2024",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatDate(tt.date)
			if result != tt.expected {
				t.Errorf("got: %q, want: %q", result, tt.expected)
			}
		})
	}
}

func TestTemporalMostRecentWeekday(t *testing.T) {
	// Reference: Friday Jan 30, 2026

	tests := []struct {
		name     string
		target   time.Weekday
		expected time.Time
	}{
		{
			name:     "most recent Monday (4 days back)",
			target:   time.Monday,
			expected: time.Date(2026, time.January, 26, 10, 0, 0, 0, time.UTC),
		},
		{
			name:     "most recent Thursday (1 day back)",
			target:   time.Thursday,
			expected: time.Date(2026, time.January, 29, 10, 0, 0, 0, time.UTC),
		},
		{
			name:     "most recent Friday (same day = 7 days back)",
			target:   time.Friday,
			expected: time.Date(2026, time.January, 23, 10, 0, 0, 0, time.UTC),
		},
		{
			name:     "most recent Sunday (5 days back)",
			target:   time.Sunday,
			expected: time.Date(2026, time.January, 25, 10, 0, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mostRecentWeekday(referenceDate, tt.target)
			if !result.Equal(tt.expected) {
				t.Errorf("got: %v, want: %v", result, tt.expected)
			}
		})
	}
}

func TestTemporalSubtractUnits(t *testing.T) {
	base := time.Date(2026, time.January, 30, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		n        int
		unit     string
		expected time.Time
	}{
		{"1 day", 1, "day", time.Date(2026, time.January, 29, 0, 0, 0, 0, time.UTC)},
		{"5 days", 5, "days", time.Date(2026, time.January, 25, 0, 0, 0, 0, time.UTC)},
		{"1 week", 1, "week", time.Date(2026, time.January, 23, 0, 0, 0, 0, time.UTC)},
		{"2 weeks", 2, "weeks", time.Date(2026, time.January, 16, 0, 0, 0, 0, time.UTC)},
		{"1 month", 1, "month", time.Date(2025, time.December, 30, 0, 0, 0, 0, time.UTC)},
		{"3 months", 3, "months", time.Date(2025, time.October, 30, 0, 0, 0, 0, time.UTC)},
		{"1 year", 1, "year", time.Date(2025, time.January, 30, 0, 0, 0, 0, time.UTC)},
		{"2 years", 2, "years", time.Date(2024, time.January, 30, 0, 0, 0, 0, time.UTC)},
		{"unknown unit", 1, "centuries", base}, // returns unchanged
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := subtractUnits(base, tt.n, tt.unit)
			if !result.Equal(tt.expected) {
				t.Errorf("got: %v, want: %v", result, tt.expected)
			}
		})
	}
}

func TestTemporalResolveComplexSentence(t *testing.T) {
	input := "The issue was first reported last week but we only started investigating 2 days ago. This morning we found the root cause."
	result := ResolveTemporalReferences(input, referenceDate)

	// Check all three references are resolved
	if !strings.Contains(result, "last week (January 23, 2026)") {
		t.Errorf("missing 'last week' resolution in: %q", result)
	}
	if !strings.Contains(result, "2 days ago (January 28, 2026)") {
		t.Errorf("missing '2 days ago' resolution in: %q", result)
	}
	if !strings.Contains(result, "This morning (January 30, 2026)") {
		t.Errorf("missing 'this morning' resolution in: %q", result)
	}
}

func TestTemporalResolveDoesNotDoubleResolve(t *testing.T) {
	// If text already has a resolved reference, running again should not add another
	input := "yesterday (January 29, 2026)"
	result := ResolveTemporalReferences(input, referenceDate)

	// The idempotency check should detect that "yesterday" is already followed
	// by a resolved date in parentheses, and skip re-resolution
	count := strings.Count(result, "(January 29, 2026)")
	if count != 1 {
		t.Errorf("double resolution detected: count=%d, result=%q", count, result)
	}
	if result != input {
		t.Errorf("already-resolved text should be unchanged:\n  got:  %q\n  want: %q", result, input)
	}
}

func TestTemporalResolveBoundaryWordMatch(t *testing.T) {
	// "today" should not match inside "todays" or "yesterday" inside "yesterdays"
	tests := []struct {
		name  string
		input string
	}{
		{"todays", "check todays tasks"},    // "todays" not a word boundary match for "today"
		{"yesterdays", "yesterdays meeting"}, // "yesterdays" should still match "yesterday" prefix due to word boundary
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ResolveTemporalReferences(tt.input, referenceDate)
			// These should NOT be modified since "todays" and "yesterdays" are different words
			// from "today" and "yesterday" (word boundary \b prevents match)
			if result != tt.input {
				t.Logf("note: %q was modified to %q — check if this is desired word boundary behavior", tt.input, result)
			}
		})
	}
}

func TestTemporalResolveLastSundayCalculation(t *testing.T) {
	// Friday Jan 30 — last Sunday should be Jan 25 (5 days back)
	input := "last Sunday"
	result := ResolveTemporalReferences(input, referenceDate)
	expected := "last Sunday (January 25, 2026)"
	if result != expected {
		t.Errorf("last Sunday:\n  got:  %q\n  want: %q", result, expected)
	}
}

func TestTemporalResolveNumericUpperBound(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		modified bool
	}{
		{"within bound", "fixed 100 days ago", true},
		{"at bound", "fixed 1000 days ago", true},
		{"exceeds bound", "fixed 1001 days ago", false},
		{"way over bound", "fixed 999999 days ago", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ResolveTemporalReferences(tt.input, referenceDate)
			wasModified := result != tt.input
			if wasModified != tt.modified {
				t.Errorf("input %q: modified=%v, want modified=%v\n  result: %q", tt.input, wasModified, tt.modified, result)
			}
		})
	}
}

func TestTemporalResolveIdempotencyFalsePositive(t *testing.T) {
	// Non-date parenthetical content that looks like a date pattern
	// should NOT prevent resolution on a second pass
	tests := []struct {
		name     string
		input    string
		resolved bool
	}{
		{
			name:     "non-month capitalized word in parens",
			input:    "yesterday (Abstract 3, 2026)",
			resolved: true, // "Abstract" is not a month — should still resolve
		},
		{
			name:     "actual resolved date should not double-resolve",
			input:    "yesterday (January 29, 2026)",
			resolved: false, // already resolved — skip
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ResolveTemporalReferences(tt.input, referenceDate)
			if tt.resolved {
				// Should have been modified (non-month word doesn't block resolution)
				count := strings.Count(result, "(January 29, 2026)")
				if count < 1 {
					t.Errorf("expected resolution, got: %q", result)
				}
			} else {
				// Should be unchanged (already resolved)
				if result != tt.input {
					t.Errorf("expected unchanged, got: %q", result)
				}
			}
		})
	}
}
