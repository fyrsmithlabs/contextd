package reasoningbank

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// maxTemporalInputLength is the maximum text length for temporal resolution.
// Prevents ReDoS attacks by limiting input size before regex execution.
const maxTemporalInputLength = 10000

// maxTemporalNumericValue is the upper bound for numeric temporal values
// (e.g., "N days ago"). Prevents nonsensical dates from unbounded input.
const maxTemporalNumericValue = 1000

// dayNames maps day-of-week names (lowercase) to time.Weekday values.
var dayNames = map[string]time.Weekday{
	"sunday":    time.Sunday,
	"monday":    time.Monday,
	"tuesday":   time.Tuesday,
	"wednesday": time.Wednesday,
	"thursday":  time.Thursday,
	"friday":    time.Friday,
	"saturday":  time.Saturday,
}

// temporalPatterns defines the regex patterns and their resolution functions.
// Order matters: more specific patterns must come before general ones to avoid
// partial matches (e.g., "last monday" before "last week").
var temporalPatterns = []struct {
	pattern *regexp.Regexp
	resolve func(match []string, sessionDate time.Time) string
}{
	// "X days/weeks/months/years ago" — e.g., "3 days ago", "2 weeks ago"
	{
		pattern: regexp.MustCompile(`(?i)\b(\d+)\s+(days?|weeks?|months?|years?)\s+ago\b`),
		resolve: func(match []string, sessionDate time.Time) string {
			n, err := strconv.Atoi(match[1])
			if err != nil || n <= 0 || n > maxTemporalNumericValue {
				return match[0]
			}
			resolved := subtractUnits(sessionDate, n, match[2])
			return fmt.Sprintf("%s (%s)", match[0], formatDate(resolved))
		},
	},
	// "a couple of weeks ago"
	{
		pattern: regexp.MustCompile(`(?i)\ba\s+couple\s+of\s+weeks\s+ago\b`),
		resolve: func(match []string, sessionDate time.Time) string {
			resolved := sessionDate.AddDate(0, 0, -14)
			return fmt.Sprintf("%s (%s)", match[0], formatDate(resolved))
		},
	},
	// "a few days ago"
	{
		pattern: regexp.MustCompile(`(?i)\ba\s+few\s+days\s+ago\b`),
		resolve: func(match []string, sessionDate time.Time) string {
			resolved := sessionDate.AddDate(0, 0, -3)
			return fmt.Sprintf("%s (%s)", match[0], formatDate(resolved))
		},
	},
	// "last Monday/Tuesday/..." — most recent [day] before sessionDate
	{
		pattern: regexp.MustCompile(`(?i)\blast\s+(sunday|monday|tuesday|wednesday|thursday|friday|saturday)\b`),
		resolve: func(match []string, sessionDate time.Time) string {
			targetDay, ok := dayNames[strings.ToLower(match[1])]
			if !ok {
				return match[0]
			}
			resolved := mostRecentWeekday(sessionDate, targetDay)
			return fmt.Sprintf("%s (%s)", match[0], formatDate(resolved))
		},
	},
	// "last week"
	{
		pattern: regexp.MustCompile(`(?i)\blast\s+week\b`),
		resolve: func(match []string, sessionDate time.Time) string {
			resolved := sessionDate.AddDate(0, 0, -7)
			return fmt.Sprintf("%s (%s)", match[0], formatDate(resolved))
		},
	},
	// "last month"
	{
		pattern: regexp.MustCompile(`(?i)\blast\s+month\b`),
		resolve: func(match []string, sessionDate time.Time) string {
			resolved := sessionDate.AddDate(0, -1, 0)
			return fmt.Sprintf("%s (%s)", match[0], formatDate(resolved))
		},
	},
	// "last year"
	{
		pattern: regexp.MustCompile(`(?i)\blast\s+year\b`),
		resolve: func(match []string, sessionDate time.Time) string {
			resolved := sessionDate.AddDate(-1, 0, 0)
			return fmt.Sprintf("%s (%s)", match[0], formatDate(resolved))
		},
	},
	// "this morning" / "this afternoon" / "this evening"
	{
		pattern: regexp.MustCompile(`(?i)\bthis\s+(morning|afternoon|evening)\b`),
		resolve: func(match []string, sessionDate time.Time) string {
			return fmt.Sprintf("%s (%s)", match[0], formatDate(sessionDate))
		},
	},
	// "yesterday"
	{
		pattern: regexp.MustCompile(`(?i)\byesterday\b`),
		resolve: func(match []string, sessionDate time.Time) string {
			resolved := sessionDate.AddDate(0, 0, -1)
			return fmt.Sprintf("%s (%s)", match[0], formatDate(resolved))
		},
	},
	// "today"
	{
		pattern: regexp.MustCompile(`(?i)\btoday\b`),
		resolve: func(match []string, sessionDate time.Time) string {
			return fmt.Sprintf("%s (%s)", match[0], formatDate(sessionDate))
		},
	},
}

// ResolveTemporalReferences detects relative temporal expressions in text and appends
// their resolved absolute dates in parentheses. The original text is preserved.
//
// Example:
//
//	input:  "The bug appeared yesterday and was fixed today"
//	output: "The bug appeared yesterday (January 29, 2026) and was fixed today (January 30, 2026)"
//
// If sessionDate is the zero value, the text is returned unchanged (don't guess).
// Input is truncated to maxTemporalInputLength to prevent ReDoS.
func ResolveTemporalReferences(text string, sessionDate time.Time) string {
	if sessionDate.IsZero() {
		return text
	}
	if text == "" {
		return text
	}

	// Truncate to prevent ReDoS
	if len(text) > maxTemporalInputLength {
		text = text[:maxTemporalInputLength]
	}

	for _, tp := range temporalPatterns {
		text = replaceTemporalPattern(text, tp.pattern, tp.resolve, sessionDate)
	}

	return text
}

// alreadyResolvedSuffix matches a trailing " (Month Day, Year)" to detect
// text that was already resolved by a prior call to ResolveTemporalReferences.
// Uses explicit month names to avoid false positives from non-date parenthetical content.
var alreadyResolvedSuffix = regexp.MustCompile(`\s+\((?:January|February|March|April|May|June|July|August|September|October|November|December) \d{1,2}, \d{4}\)$`)

// replaceTemporalPattern applies a temporal regex to text, skipping matches that
// are already followed by a resolved date in parentheses (idempotency).
// Since Go's regexp doesn't support lookahead, we use FindAllStringIndex to locate
// matches and check the character after each match in the original text.
func replaceTemporalPattern(text string, pattern *regexp.Regexp, resolve func([]string, time.Time) string, sessionDate time.Time) string {
	indices := pattern.FindAllStringIndex(text, -1)
	if len(indices) == 0 {
		return text
	}

	var b strings.Builder
	b.Grow(len(text) + len(indices)*30) // Pre-allocate for resolved dates
	prev := 0

	for _, loc := range indices {
		start, end := loc[0], loc[1]
		b.WriteString(text[prev:start])

		matchStr := text[start:end]

		// Check if the match is already followed by " (Month Day, Year)"
		// by looking at the remainder of the text after the match
		remainder := text[end:]
		if len(remainder) > 0 && remainder[0] == ' ' && strings.HasPrefix(remainder, " (") {
			// Check if it looks like an already-resolved date
			closeParen := strings.Index(remainder, ")")
			if closeParen > 0 && alreadyResolvedSuffix.MatchString(remainder[:closeParen+1]) {
				// Already resolved — write match unchanged
				b.WriteString(matchStr)
				prev = end
				continue
			}
		}

		submatches := pattern.FindStringSubmatch(matchStr)
		if submatches == nil {
			b.WriteString(matchStr)
		} else {
			b.WriteString(resolve(submatches, sessionDate))
		}
		prev = end
	}
	b.WriteString(text[prev:])

	return b.String()
}

// formatDate formats a time.Time as "January 2, 2006" using time.Month names.
func formatDate(t time.Time) string {
	return fmt.Sprintf("%s %d, %d", t.Month().String(), t.Day(), t.Year())
}

// subtractUnits subtracts n units (days, weeks, months, years) from the given date.
// The unit string is normalized to handle both singular and plural forms.
func subtractUnits(date time.Time, n int, unit string) time.Time {
	unit = strings.ToLower(unit)
	switch {
	case strings.HasPrefix(unit, "day"):
		return date.AddDate(0, 0, -n)
	case strings.HasPrefix(unit, "week"):
		return date.AddDate(0, 0, -n*7)
	case strings.HasPrefix(unit, "month"):
		return date.AddDate(0, -n, 0)
	case strings.HasPrefix(unit, "year"):
		return date.AddDate(-n, 0, 0)
	default:
		return date
	}
}

// mostRecentWeekday returns the most recent occurrence of the target weekday
// strictly before the given date. If sessionDate is the target weekday,
// it returns 7 days earlier (the previous occurrence).
func mostRecentWeekday(sessionDate time.Time, target time.Weekday) time.Time {
	current := sessionDate.Weekday()
	daysBack := int(current) - int(target)
	if daysBack <= 0 {
		daysBack += 7
	}
	return sessionDate.AddDate(0, 0, -daysBack)
}
