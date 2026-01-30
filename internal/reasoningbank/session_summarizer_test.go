package reasoningbank

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestNewSessionSummarizer(t *testing.T) {
	t.Parallel()

	logger := zaptest.NewLogger(t)
	extractor := NewSimpleExtractor()

	t.Run("valid inputs", func(t *testing.T) {
		t.Parallel()
		s, err := NewSessionSummarizer(extractor, logger)
		require.NoError(t, err)
		require.NotNil(t, s)
	})

	t.Run("nil extractor", func(t *testing.T) {
		t.Parallel()
		s, err := NewSessionSummarizer(nil, logger)
		assert.Error(t, err)
		assert.Nil(t, s)
		assert.Contains(t, err.Error(), "fact extractor cannot be nil")
	})

	t.Run("nil logger", func(t *testing.T) {
		t.Parallel()
		s, err := NewSessionSummarizer(extractor, nil)
		assert.Error(t, err)
		assert.Nil(t, s)
		assert.Contains(t, err.Error(), "logger cannot be nil")
	})
}

func TestSummarize_NilBuffer(t *testing.T) {
	t.Parallel()

	s := newTestSummarizer(t)
	memories, err := s.Summarize(context.Background(), nil)
	assert.Error(t, err)
	assert.Nil(t, memories)
	assert.Contains(t, err.Error(), "session buffer cannot be nil")
}

func TestSummarize_EmptyBuffer(t *testing.T) {
	t.Parallel()

	s := newTestSummarizer(t)
	buf := &SessionBuffer{
		SessionID:   "sess-1",
		ProjectID:   "proj-1",
		SessionDate: time.Now(),
		Turns:       []TurnEntry{},
	}

	memories, err := s.Summarize(context.Background(), buf)
	assert.NoError(t, err)
	assert.Nil(t, memories)
}

func TestSummarize_SingleSuccessTurn(t *testing.T) {
	t.Parallel()

	s := newTestSummarizer(t)
	sessionDate := time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC)

	buf := &SessionBuffer{
		SessionID:   "sess-abc",
		ProjectID:   "proj-1",
		SessionDate: sessionDate,
		Turns: []TurnEntry{
			{
				Title:   "Fix auth bug",
				Content: "I implemented a retry mechanism for token refresh",
				Outcome: OutcomeSuccess,
				Tags:    []string{"go", "auth"},
			},
		},
	}

	memories, err := s.Summarize(context.Background(), buf)
	require.NoError(t, err)
	require.Len(t, memories, 1)

	mem := memories[0]
	assert.Equal(t, "proj-1", mem.ProjectID)
	assert.Equal(t, "sess-abc", mem.SessionID)
	assert.Equal(t, &sessionDate, mem.SessionDate)
	assert.Equal(t, GranularitySession, mem.Granularity)
	assert.Equal(t, OutcomeSuccess, mem.Outcome)
	assert.Equal(t, DistilledConfidence, mem.Confidence)
	assert.Contains(t, mem.Title, "Success")
	assert.Contains(t, mem.Title, "sess-abc")
	assert.Contains(t, mem.Content, "Fix auth bug")
	assert.Contains(t, mem.Content, "retry mechanism")
	assert.Contains(t, mem.Description, "1 turns")
}

func TestSummarize_MixedOutcomes(t *testing.T) {
	t.Parallel()

	s := newTestSummarizer(t)
	sessionDate := time.Date(2025, 7, 1, 14, 0, 0, 0, time.UTC)

	buf := &SessionBuffer{
		SessionID:   "sess-mixed",
		ProjectID:   "proj-2",
		SessionDate: sessionDate,
		Turns: []TurnEntry{
			{
				Title:   "Add caching",
				Content: "I implemented Redis caching for API responses",
				Outcome: OutcomeSuccess,
				Tags:    []string{"redis", "caching"},
			},
			{
				Title:   "Query optimization",
				Content: "I learned about index optimization for PostgreSQL",
				Outcome: OutcomeSuccess,
				Tags:    []string{"postgres", "performance"},
			},
			{
				Title:   "Deploy failure",
				Content: "Deployment failed due to missing env var",
				Outcome: OutcomeFailure,
				Tags:    []string{"deploy"},
			},
		},
	}

	memories, err := s.Summarize(context.Background(), buf)
	require.NoError(t, err)
	require.Len(t, memories, 2, "should produce one memory per outcome")

	// First should be success (deterministic ordering)
	successMem := memories[0]
	assert.Equal(t, OutcomeSuccess, successMem.Outcome)
	assert.Equal(t, GranularitySession, successMem.Granularity)
	assert.Contains(t, successMem.Title, "Success")
	assert.Contains(t, successMem.Content, "Redis caching")
	assert.Contains(t, successMem.Content, "index optimization")
	assert.Contains(t, successMem.Description, "2 turns")

	// Second should be failure
	failureMem := memories[1]
	assert.Equal(t, OutcomeFailure, failureMem.Outcome)
	assert.Equal(t, GranularitySession, failureMem.Granularity)
	assert.Contains(t, failureMem.Title, "Anti-pattern")
	assert.Contains(t, failureMem.Content, "missing env var")
	assert.Contains(t, failureMem.Description, "1 turns")
}

func TestSummarize_DefaultOutcome(t *testing.T) {
	t.Parallel()

	s := newTestSummarizer(t)

	buf := &SessionBuffer{
		SessionID:   "sess-default",
		ProjectID:   "proj-1",
		SessionDate: time.Now(),
		Turns: []TurnEntry{
			{
				Title:   "Task",
				Content: "Some content without explicit outcome",
				Outcome: "", // Empty outcome should default to success
				Tags:    []string{"general"},
			},
		},
	}

	memories, err := s.Summarize(context.Background(), buf)
	require.NoError(t, err)
	require.Len(t, memories, 1)
	assert.Equal(t, OutcomeSuccess, memories[0].Outcome)
}

func TestSummarize_WithFactExtraction(t *testing.T) {
	t.Parallel()

	s := newTestSummarizer(t)
	sessionDate := time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC)

	buf := &SessionBuffer{
		SessionID:   "sess-facts",
		ProjectID:   "proj-1",
		SessionDate: sessionDate,
		Turns: []TurnEntry{
			{
				Title:   "Learning",
				Content: "I learned about Go error handling patterns",
				Outcome: OutcomeSuccess,
				Tags:    []string{"go"},
			},
			{
				Title:   "Implementation",
				Content: "I implemented a circuit breaker for the API client",
				Outcome: OutcomeSuccess,
				Tags:    []string{"go", "resilience"},
			},
		},
	}

	memories, err := s.Summarize(context.Background(), buf)
	require.NoError(t, err)
	require.Len(t, memories, 1)

	// The SimpleExtractor should find facts from "I learned..." and "I implemented..."
	assert.Contains(t, memories[0].Content, "Extracted Facts")
}

func TestSummarize_ContextCancellation(t *testing.T) {
	t.Parallel()

	s := newTestSummarizer(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	buf := &SessionBuffer{
		SessionID:   "sess-cancel",
		ProjectID:   "proj-1",
		SessionDate: time.Now(),
		Turns: []TurnEntry{
			{Title: "t1", Content: "c1", Outcome: OutcomeSuccess},
			{Title: "t2", Content: "c2", Outcome: OutcomeFailure},
		},
	}

	memories, err := s.Summarize(ctx, buf)
	assert.Error(t, err)
	assert.Nil(t, memories)
}

func TestSummarize_SessionMetadata(t *testing.T) {
	t.Parallel()

	s := newTestSummarizer(t)
	sessionDate := time.Date(2025, 8, 20, 16, 30, 0, 0, time.UTC)

	buf := &SessionBuffer{
		SessionID:   "sess-metadata-test",
		ProjectID:   "proj-meta",
		SessionDate: sessionDate,
		Turns: []TurnEntry{
			{
				Title:   "Work item",
				Content: "Completed a task",
				Outcome: OutcomeSuccess,
			},
		},
	}

	memories, err := s.Summarize(context.Background(), buf)
	require.NoError(t, err)
	require.Len(t, memories, 1)

	mem := memories[0]
	assert.Equal(t, "sess-metadata-test", mem.SessionID)
	assert.Equal(t, &sessionDate, mem.SessionDate)
	assert.Equal(t, GranularitySession, mem.Granularity)
	assert.Equal(t, "proj-meta", mem.ProjectID)
	assert.NotEmpty(t, mem.ID)
	assert.NotZero(t, mem.CreatedAt)
	assert.NotZero(t, mem.UpdatedAt)
}

func TestSummarize_TagDeduplication(t *testing.T) {
	t.Parallel()

	s := newTestSummarizer(t)

	buf := &SessionBuffer{
		SessionID:   "sess-tags",
		ProjectID:   "proj-1",
		SessionDate: time.Now(),
		Turns: []TurnEntry{
			{Title: "t1", Content: "c1", Outcome: OutcomeSuccess, Tags: []string{"go", "auth"}},
			{Title: "t2", Content: "c2", Outcome: OutcomeSuccess, Tags: []string{"go", "testing"}},
			{Title: "t3", Content: "c3", Outcome: OutcomeSuccess, Tags: []string{"auth", "testing"}},
		},
	}

	memories, err := s.Summarize(context.Background(), buf)
	require.NoError(t, err)
	require.Len(t, memories, 1)

	// Tags should be deduplicated: go, auth, testing = 3 unique tags
	assert.Len(t, memories[0].Tags, 3)

	tagSet := make(map[string]bool)
	for _, tag := range memories[0].Tags {
		tagSet[tag] = true
	}
	assert.True(t, tagSet["go"])
	assert.True(t, tagSet["auth"])
	assert.True(t, tagSet["testing"])
}

func TestSummarize_ZeroSessionDate(t *testing.T) {
	t.Parallel()

	s := newTestSummarizer(t)

	buf := &SessionBuffer{
		SessionID:   "sess-nodate",
		ProjectID:   "proj-1",
		SessionDate: time.Time{}, // Zero value
		Turns: []TurnEntry{
			{Title: "t1", Content: "I learned something today", Outcome: OutcomeSuccess},
		},
	}

	// Should not panic - zero date means "use time.Now()" for fact extraction reference
	memories, err := s.Summarize(context.Background(), buf)
	require.NoError(t, err)
	require.Len(t, memories, 1)
}

// newTestSummarizer creates a SessionSummarizer for testing.
func newTestSummarizer(t *testing.T) *SessionSummarizer {
	t.Helper()
	logger := zaptest.NewLogger(t)
	extractor := NewSimpleExtractor()
	s, err := NewSessionSummarizer(extractor, logger)
	require.NoError(t, err)
	return s
}
