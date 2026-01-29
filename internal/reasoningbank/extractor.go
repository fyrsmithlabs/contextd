package reasoningbank

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// maxTextLength is the maximum allowed input text length (100KB).
// This prevents denial of service attacks via extremely large inputs.
const maxTextLength = 100000

// SimpleExtractor is a rule-based fact extractor using regex patterns.
//
// This implementation uses regex-based patterns to identify common structures:
//   - "I went to X" or "I attended X" -> (I, attended, X)
//   - "I'm thinking about X" or "I'm considering X" -> (I, considering, X)
//   - "I learned X" -> (I, learned, X)
//
// Temporal references are resolved against a reference date:
//   - "yesterday" -> reference date - 1 day
//   - "today" -> reference date
//   - "last week" -> reference date - 7 days
//   - "last Monday/Tuesday/etc." -> most recent occurrence of that day
//
// Confidence scores reflect extraction certainty:
//   - 1.0 for explicit, well-formed patterns
//   - 0.8 for clear but slightly ambiguous matches
//   - 0.6 for implicit or derived relations
type SimpleExtractor struct {
	// Pattern definitions for different fact types
	patterns []*patternRule
}

// patternRule defines a regex pattern and how to extract subject/predicate/object from it.
type patternRule struct {
	// regex matches candidate sentences
	regex *regexp.Regexp
	// extractFn extracts subject, predicate, object from match groups
	extractFn func([]string) (subject, predicate, object string)
	// confidence is the base confidence score for this pattern
	confidence float64
}

// NewSimpleExtractor creates a new simple fact extractor.
func NewSimpleExtractor() *SimpleExtractor {
	return &SimpleExtractor{
		patterns: buildPatterns(),
	}
}

// buildPatterns creates the set of regex patterns for fact extraction.
// All patterns use [^.!?]{1,200} instead of .+? to prevent ReDoS attacks.
func buildPatterns() []*patternRule {
	return []*patternRule{
		{
			// Pattern: "I went to/attended X" or "I attended X"
			// Using [^.!?]{1,200} instead of .+? to prevent ReDoS attacks
			regex: regexp.MustCompile(`(?i)\b(?:I|we)\s+(?:went to|attended|visited)\s+([^.!?]{1,200})(?:\s+(?:yesterday|today|tomorrow|last|this|next))?\.?$`),
			extractFn: func(groups []string) (string, string, string) {
				obj := strings.TrimSpace(groups[1])
				// Remove trailing temporal references
				obj = strings.TrimSuffix(obj, "yesterday")
				obj = strings.TrimSuffix(obj, "today")
				obj = strings.TrimSuffix(obj, "tomorrow")
				obj = strings.TrimSpace(obj)
				return "I", "attended", obj
			},
			confidence: 1.0,
		},
		{
			// Pattern: "I'm thinking about X" or "I'm considering X"
			// Using [^.!?]{1,200} instead of .+? to prevent ReDoS attacks
			regex: regexp.MustCompile(`(?i)\b(?:I'm|I am|we're|we are)\s+(?:thinking about|considering|pondering|planning)\s+([^.!?]{1,200})\.?$`),
			extractFn: func(groups []string) (string, string, string) {
				return "I", "considering", strings.TrimSpace(groups[1])
			},
			confidence: 0.9,
		},
		{
			// Pattern: "I learned X" or "I learned about X"
			// Using [^.!?]{1,200} instead of .+? to prevent ReDoS attacks
			regex: regexp.MustCompile(`(?i)\b(?:I|we)\s+learned\s+(?:about\s+)?([^.!?]{1,200})\.?$`),
			extractFn: func(groups []string) (string, string, string) {
				obj := strings.TrimSpace(groups[1])
				// Remove common patterns and temporal references
				obj = strings.TrimSuffix(obj, " yesterday")
				obj = strings.TrimSuffix(obj, " today")
				obj = strings.TrimSuffix(obj, " tomorrow")
				obj = strings.TrimSpace(obj)
				return "I", "learned", obj
			},
			confidence: 0.95,
		},
		{
			// Pattern: "I implemented X" or "I built X"
			// Using [^.!?]{1,200} instead of .+? to prevent ReDoS attacks
			regex: regexp.MustCompile(`(?i)\b(?:I|we)\s+(?:implemented|built|created|developed)\s+([^.!?]{1,200})\.?$`),
			extractFn: func(groups []string) (string, string, string) {
				obj := strings.TrimSpace(groups[1])
				// Remove trailing temporal references
				obj = strings.TrimSuffix(obj, " last week")
				obj = strings.TrimSuffix(obj, " yesterday")
				obj = strings.TrimSpace(obj)
				return "I", "implemented", obj
			},
			confidence: 1.0,
		},
		{
			// Pattern: "X is Y" (property assignment)
			// Using [^.!?]{1,200} instead of .+? to prevent ReDoS attacks
			regex: regexp.MustCompile(`(?i)\b([A-Za-z]+)\s+is\s+([^.!?]{1,200})\.?$`),
			extractFn: func(groups []string) (string, string, string) {
				return strings.TrimSpace(groups[1]), "is", strings.TrimSpace(groups[2])
			},
			confidence: 0.8,
		},
		{
			// Pattern: "I did X" or "I had X"
			// Using [^.!?]{1,200} instead of .+? to prevent ReDoS attacks
			regex: regexp.MustCompile(`(?i)\b(?:I|we)\s+(?:did|had|made|fixed|resolved)\s+([^.!?]{1,200})\.?$`),
			extractFn: func(groups []string) (string, string, string) {
				return "I", "did", strings.TrimSpace(groups[1])
			},
			confidence: 0.85,
		},
	}
}

// Extract parses text and returns structured facts.
func (e *SimpleExtractor) Extract(ctx context.Context, text string, referenceDate time.Time) ([]Fact, error) {
	// Check context at start for timeout/cancellation
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if text == "" {
		return nil, ErrEmptyFactText
	}

	// Validate input length to prevent DoS
	if len(text) > maxTextLength {
		return nil, fmt.Errorf("text exceeds maximum length of %d bytes", maxTextLength)
	}

	// Split text into sentences for independent processing
	sentences := splitSentences(text)
	var facts []Fact

	for i, sentence := range sentences {
		// Periodically check context for cancellation during long processing
		if i%100 == 0 && i > 0 {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
		}

		sentence = strings.TrimSpace(sentence)
		if sentence == "" {
			continue
		}

		// Try each pattern against the sentence
		for _, pattern := range e.patterns {
			matches := pattern.regex.FindStringSubmatch(sentence)
			if matches != nil {
				subject, predicate, object := pattern.extractFn(matches)

				// Skip if extraction resulted in empty fields
				if subject == "" || predicate == "" || object == "" {
					continue
				}

				// Resolve temporal references in timestamp
				timestamp := resolveTemporalReference(sentence, referenceDate)

				fact := Fact{
					Subject:    subject,
					Predicate:  predicate,
					Object:     object,
					Timestamp:  timestamp,
					Confidence: pattern.confidence,
					Provenance: sentence,
				}

				facts = append(facts, fact)
				break // Move to next sentence after first match
			}
		}
	}

	return facts, nil
}

// splitSentences splits text into sentences by common delimiters.
// Empty strings are filtered out to avoid unnecessary processing.
func splitSentences(text string) []string {
	// Split by sentence terminators: period, exclamation, question mark
	re := regexp.MustCompile(`[.!?]+`)
	sentences := re.Split(text, -1)

	// Filter out empty strings
	filtered := sentences[:0]
	for _, s := range sentences {
		s = strings.TrimSpace(s)
		if s != "" {
			filtered = append(filtered, s)
		}
	}
	return filtered
}

// resolveTemporalReference extracts and resolves temporal references from text.
//
// Supports:
//   - "today" -> reference date
//   - "yesterday" -> reference date - 1 day
//   - "last week" -> reference date - 7 days
//   - "last Monday/Tuesday/etc." -> most recent occurrence of that weekday
//   - No temporal reference -> returns reference date
func resolveTemporalReference(text string, referenceDate time.Time) time.Time {
	text = strings.ToLower(text)

	// Check for "today"
	if strings.Contains(text, "today") {
		return referenceDate
	}

	// Check for "yesterday"
	if strings.Contains(text, "yesterday") {
		return referenceDate.AddDate(0, 0, -1)
	}

	// Check for "tomorrow"
	if strings.Contains(text, "tomorrow") {
		return referenceDate.AddDate(0, 0, 1)
	}

	// Check for "last week"
	if strings.Contains(text, "last week") {
		return referenceDate.AddDate(0, 0, -7)
	}

	// Check for "this week"
	if strings.Contains(text, "this week") {
		return referenceDate
	}

	// Check for "last month"
	if strings.Contains(text, "last month") {
		return referenceDate.AddDate(0, -1, 0)
	}

	// Check for "last year"
	if strings.Contains(text, "last year") {
		return referenceDate.AddDate(-1, 0, 0)
	}

	// Check for day names: "last Monday", "last Tuesday", etc.
	dayNames := []string{"sunday", "monday", "tuesday", "wednesday", "thursday", "friday", "saturday"}
	for i, dayName := range dayNames {
		if strings.Contains(text, "last "+dayName) {
			// Calculate days back to the most recent occurrence of this weekday
			targetWeekday := time.Weekday(i)
			daysBack := int(referenceDate.Weekday() - targetWeekday)
			if daysBack <= 0 {
				daysBack += 7 // If target day is today or in future, go back to previous week
			}
			return referenceDate.AddDate(0, 0, -daysBack)
		}
	}

	// No temporal reference found, use reference date
	return referenceDate
}
