package reasoningbank

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSimpleExtractor(t *testing.T) {
	extractor := NewSimpleExtractor()
	assert.NotNil(t, extractor)
	assert.NotEmpty(t, extractor.patterns)
	assert.True(t, len(extractor.patterns) > 0)
}

func TestSimpleExtractorExtract(t *testing.T) {
	extractor := NewSimpleExtractor()
	ctx := context.Background()
	referenceDate := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC) // Monday

	tests := []struct {
		name          string
		text          string
		referenceDate time.Time
		expectError   bool
		expectFacts   int
		expectedFacts []struct {
			subject    string
			predicate  string
			object     string
			confidence float64
		}
	}{
		{
			name:          "I attended meeting",
			text:          "I attended the architecture review meeting.",
			referenceDate: referenceDate,
			expectError:   false,
			expectFacts:   1,
			expectedFacts: []struct {
				subject    string
				predicate  string
				object     string
				confidence float64
			}{
				{subject: "I", predicate: "attended", object: "the architecture review meeting", confidence: 1.0},
			},
		},
		{
			name:          "multiple sentences with different patterns",
			text:          "I learned about Go concurrency patterns. I'm considering implementing a worker pool.",
			referenceDate: referenceDate,
			expectError:   false,
			expectFacts:   2,
			expectedFacts: []struct {
				subject    string
				predicate  string
				object     string
				confidence float64
			}{
				{subject: "I", predicate: "learned", object: "Go concurrency patterns", confidence: 0.95},
				{subject: "I", predicate: "considering", object: "implementing a worker pool", confidence: 0.9},
			},
		},
		{
			name:          "I implemented feature",
			text:          "I implemented a new authentication service.",
			referenceDate: referenceDate,
			expectError:   false,
			expectFacts:   1,
			expectedFacts: []struct {
				subject    string
				predicate  string
				object     string
				confidence float64
			}{
				{subject: "I", predicate: "implemented", object: "a new authentication service", confidence: 1.0},
			},
		},
		{
			name:          "I did something",
			text:          "I fixed the database connection pool issue.",
			referenceDate: referenceDate,
			expectError:   false,
			expectFacts:   1,
			expectedFacts: []struct {
				subject    string
				predicate  string
				object     string
				confidence float64
			}{
				{subject: "I", predicate: "did", object: "the database connection pool issue", confidence: 0.85},
			},
		},
		{
			name:          "property assignment",
			text:          "Go is powerful for backend development.",
			referenceDate: referenceDate,
			expectError:   false,
			expectFacts:   1,
			expectedFacts: []struct {
				subject    string
				predicate  string
				object     string
				confidence float64
			}{
				{subject: "Go", predicate: "is", object: "powerful for backend development", confidence: 0.8},
			},
		},
		{
			name:          "empty text",
			text:          "",
			referenceDate: referenceDate,
			expectError:   true,
		},
		{
			name:          "text with no extractable patterns",
			text:          "The sunshine feels wonderful outside.",
			referenceDate: referenceDate,
			expectError:   false,
			expectFacts:   0,
		},
		{
			name:          "multiple sentences mixed",
			text:          "I attended the planning session. We learned about the new deployment pipeline. I'm thinking about optimization strategies.",
			referenceDate: referenceDate,
			expectError:   false,
			expectFacts:   3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			facts, err := extractor.Extract(ctx, tt.text, tt.referenceDate)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectFacts, len(facts), "expected %d facts but got %d", tt.expectFacts, len(facts))

				for i, expectedFact := range tt.expectedFacts {
					if i < len(facts) {
						fact := facts[i]
						assert.Equal(t, expectedFact.subject, fact.Subject)
						assert.Equal(t, expectedFact.predicate, fact.Predicate)
						assert.Equal(t, expectedFact.object, fact.Object)
						assert.Equal(t, expectedFact.confidence, fact.Confidence)
						assert.NotEmpty(t, fact.Provenance)
					}
				}
			}
		})
	}
}

func TestResolveTemporalReference(t *testing.T) {
	// Reference date: Monday, January 15, 2024
	referenceDate := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name          string
		text          string
		referenceDate time.Time
		expected      time.Time
	}{
		{
			name:          "today",
			text:          "I did this today.",
			referenceDate: referenceDate,
			expected:      referenceDate,
		},
		{
			name:          "yesterday",
			text:          "I attended the meeting yesterday.",
			referenceDate: referenceDate,
			expected:      referenceDate.AddDate(0, 0, -1), // Jan 14
		},
		{
			name:          "tomorrow",
			text:          "I plan to implement it tomorrow.",
			referenceDate: referenceDate,
			expected:      referenceDate.AddDate(0, 0, 1), // Jan 16
		},
		{
			name:          "last week",
			text:          "I learned about it last week.",
			referenceDate: referenceDate,
			expected:      referenceDate.AddDate(0, 0, -7), // Jan 8
		},
		{
			name:          "this week",
			text:          "I did it this week.",
			referenceDate: referenceDate,
			expected:      referenceDate,
		},
		{
			name:          "last month",
			text:          "I built this last month.",
			referenceDate: referenceDate,
			expected:      referenceDate.AddDate(0, -1, 0), // Dec 15, 2023
		},
		{
			name:          "last year",
			text:          "I implemented it last year.",
			referenceDate: referenceDate,
			expected:      referenceDate.AddDate(-1, 0, 0), // Jan 15, 2023
		},
		{
			name:          "no temporal reference",
			text:          "I did something.",
			referenceDate: referenceDate,
			expected:      referenceDate,
		},
		{
			name:          "last Monday",
			text:          "I attended last Monday.",
			referenceDate: referenceDate,                   // Monday Jan 15
			expected:      referenceDate.AddDate(0, 0, -7), // Previous Monday Jan 8
		},
		{
			name:          "last Friday",
			text:          "I met with the team last Friday.",
			referenceDate: referenceDate, // Monday Jan 15 (weekday 1)
			// Monday is weekday 1, Friday is weekday 5
			// daysBack = (1 - 5) % 7 = -4 % 7 = -4 (in Go, -4 % 7 = -4)
			// Since daysBack != 0, don't add 7, just use -4: but -4 means 3 days forward which is wrong
			// Actually: (1 - 5) = -4, -4 + 7 = 3, so 3 days back = Jan 12
			expected: referenceDate.AddDate(0, 0, -3), // Previous Friday Jan 12
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolveTemporalReference(tt.text, tt.referenceDate)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSplitSentences(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected int
	}{
		{
			name:     "single sentence with period",
			text:     "I attended the meeting.",
			expected: 1, // Empty string at end is now filtered out
		},
		{
			name:     "multiple sentences",
			text:     "I attended the meeting. I learned a lot! Do you agree?",
			expected: 3, // Empty string at end is now filtered out
		},
		{
			name:     "multiple punctuation",
			text:     "Really?! Yes. Indeed...",
			expected: 3, // Empty strings are filtered out
		},
		{
			name:     "no punctuation",
			text:     "I attended the meeting",
			expected: 1,
		},
		{
			name:     "empty string",
			text:     "",
			expected: 0, // Empty strings are filtered out
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := splitSentences(tt.text)
			assert.Equal(t, tt.expected, len(result))
		})
	}
}

func TestExtractorCaseInsensitive(t *testing.T) {
	extractor := NewSimpleExtractor()
	ctx := context.Background()
	referenceDate := time.Now()

	tests := []struct {
		name     string
		text     string
		expected string
	}{
		{
			name:     "lowercase",
			text:     "i attended the meeting.",
			expected: "attended",
		},
		{
			name:     "uppercase",
			text:     "I ATTENDED THE MEETING.",
			expected: "attended",
		},
		{
			name:     "mixed case",
			text:     "I Attended The Meeting.",
			expected: "attended",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			facts, err := extractor.Extract(ctx, tt.text, referenceDate)
			require.NoError(t, err)
			require.True(t, len(facts) > 0, "expected to extract facts")
			assert.Equal(t, tt.expected, facts[0].Predicate)
		})
	}
}

func TestExtractorWithSourceID(t *testing.T) {
	extractor := NewSimpleExtractor()
	ctx := context.Background()
	referenceDate := time.Now()
	text := "I learned about distributed systems yesterday."

	facts, err := extractor.Extract(ctx, text, referenceDate)
	require.NoError(t, err)
	require.True(t, len(facts) > 0)

	// Facts returned from Extract don't have SourceID set yet
	// (it will be set by the caller)
	assert.Empty(t, facts[0].SourceID)
	assert.NotEmpty(t, facts[0].Provenance)
}

func TestExtractorMaxTextLength(t *testing.T) {
	extractor := NewSimpleExtractor()
	ctx := context.Background()
	referenceDate := time.Now()

	// Create text that exceeds the 100KB limit
	largeText := strings.Repeat("I attended a meeting. ", 10000)
	require.Greater(t, len(largeText), maxTextLength, "test text should exceed maxTextLength")

	facts, err := extractor.Extract(ctx, largeText, referenceDate)
	require.Error(t, err)
	assert.Nil(t, facts)
	assert.Contains(t, err.Error(), "exceeds maximum length")
}

func TestExtractorContextCancellation(t *testing.T) {
	extractor := NewSimpleExtractor()
	referenceDate := time.Now()

	// Create a pre-cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	facts, err := extractor.Extract(ctx, "I attended a meeting.", referenceDate)
	require.Error(t, err)
	assert.Nil(t, facts)
	assert.Equal(t, context.Canceled, err)
}

func TestExtractorContextTimeout(t *testing.T) {
	extractor := NewSimpleExtractor()
	referenceDate := time.Now()

	// Create a context that times out immediately
	ctx, cancel := context.WithTimeout(context.Background(), 0)
	defer cancel()
	time.Sleep(time.Millisecond)

	facts, err := extractor.Extract(ctx, "I attended a meeting.", referenceDate)
	require.Error(t, err)
	assert.Nil(t, facts)
	assert.Equal(t, context.DeadlineExceeded, err)
}

func BenchmarkExtractorSimple(b *testing.B) {
	extractor := NewSimpleExtractor()
	ctx := context.Background()
	referenceDate := time.Now()
	text := "I attended the meeting yesterday. I learned about Go concurrency. I'm considering implementing a new service."

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = extractor.Extract(ctx, text, referenceDate)
	}
}
