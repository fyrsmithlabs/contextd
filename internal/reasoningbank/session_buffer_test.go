package reasoningbank

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBufferTurn_Basic(t *testing.T) {
	t.Parallel()

	mgr := NewSessionBufferManager(0)

	entry := TurnEntry{
		Title:   "Fix auth bug",
		Content: "Resolved nil pointer in token validation",
		Outcome: OutcomeSuccess,
		Tags:    []string{"auth", "bugfix"},
	}

	err := mgr.BufferTurn("proj-1", "sess-1", entry)
	require.NoError(t, err)

	assert.Equal(t, 1, mgr.Count("proj-1", "sess-1"))

	buf := mgr.GetBuffer("proj-1", "sess-1")
	require.NotNil(t, buf)
	assert.Equal(t, "proj-1", buf.ProjectID)
	assert.Equal(t, "sess-1", buf.SessionID)
	require.Len(t, buf.Turns, 1)
	assert.Equal(t, "Fix auth bug", buf.Turns[0].Title)
	assert.Equal(t, "Resolved nil pointer in token validation", buf.Turns[0].Content)
	assert.Equal(t, OutcomeSuccess, buf.Turns[0].Outcome)
	assert.Equal(t, []string{"auth", "bugfix"}, buf.Turns[0].Tags)
	assert.False(t, buf.Turns[0].Timestamp.IsZero(), "timestamp should be auto-set")
}

func TestBufferTurn_MultipleTurns(t *testing.T) {
	t.Parallel()

	mgr := NewSessionBufferManager(0)

	entries := []TurnEntry{
		{Title: "Turn 1", Content: "First turn content", Outcome: OutcomeSuccess},
		{Title: "Turn 2", Content: "Second turn content", Outcome: OutcomeFailure},
		{Title: "Turn 3", Content: "Third turn content", Outcome: OutcomeSuccess},
	}

	for _, e := range entries {
		err := mgr.BufferTurn("proj-1", "sess-1", e)
		require.NoError(t, err)
	}

	assert.Equal(t, 3, mgr.Count("proj-1", "sess-1"))

	buf := mgr.GetBuffer("proj-1", "sess-1")
	require.NotNil(t, buf)
	require.Len(t, buf.Turns, 3)

	// Verify order is preserved
	for i, e := range entries {
		assert.Equal(t, e.Title, buf.Turns[i].Title, "turn %d title mismatch", i)
		assert.Equal(t, e.Content, buf.Turns[i].Content, "turn %d content mismatch", i)
		assert.Equal(t, e.Outcome, buf.Turns[i].Outcome, "turn %d outcome mismatch", i)
	}

	// Verify timestamps are monotonically non-decreasing
	for i := 1; i < len(buf.Turns); i++ {
		assert.False(t, buf.Turns[i].Timestamp.Before(buf.Turns[i-1].Timestamp),
			"turn %d timestamp should not be before turn %d", i, i-1)
	}
}

func TestBufferTurn_MaxTurnsEnforced(t *testing.T) {
	t.Parallel()

	maxTurns := 3
	mgr := NewSessionBufferManager(maxTurns)

	// Buffer 5 turns, only last 3 should remain
	for i := 0; i < 5; i++ {
		entry := TurnEntry{
			Title:   fmt.Sprintf("Turn %d", i),
			Content: fmt.Sprintf("Content %d", i),
			Outcome: OutcomeSuccess,
		}
		err := mgr.BufferTurn("proj-1", "sess-1", entry)
		require.NoError(t, err)
	}

	assert.Equal(t, maxTurns, mgr.Count("proj-1", "sess-1"))

	buf := mgr.GetBuffer("proj-1", "sess-1")
	require.NotNil(t, buf)
	require.Len(t, buf.Turns, maxTurns)

	// Oldest turns (0, 1) should have been dropped; turns 2, 3, 4 remain
	assert.Equal(t, "Turn 2", buf.Turns[0].Title)
	assert.Equal(t, "Turn 3", buf.Turns[1].Title)
	assert.Equal(t, "Turn 4", buf.Turns[2].Title)
}

func TestBufferTurn_ValidationErrors(t *testing.T) {
	t.Parallel()

	mgr := NewSessionBufferManager(0)
	entry := TurnEntry{Title: "Test", Content: "Content"}

	t.Run("empty_project_id", func(t *testing.T) {
		t.Parallel()
		err := mgr.BufferTurn("", "sess-1", entry)
		assert.ErrorIs(t, err, ErrEmptyProjectID)
	})

	t.Run("empty_session_id", func(t *testing.T) {
		t.Parallel()
		err := mgr.BufferTurn("proj-1", "", entry)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "session ID cannot be empty")
	})
}

func TestGetBuffer_ReturnsNilForMissing(t *testing.T) {
	t.Parallel()

	mgr := NewSessionBufferManager(0)

	buf := mgr.GetBuffer("nonexistent-proj", "nonexistent-sess")
	assert.Nil(t, buf)
}

func TestGetBuffer_ReturnsCopy(t *testing.T) {
	t.Parallel()

	mgr := NewSessionBufferManager(0)

	err := mgr.BufferTurn("proj-1", "sess-1", TurnEntry{
		Title:   "Original",
		Content: "Original content",
		Outcome: OutcomeSuccess,
	})
	require.NoError(t, err)

	// Get a copy
	buf1 := mgr.GetBuffer("proj-1", "sess-1")
	require.NotNil(t, buf1)
	require.Len(t, buf1.Turns, 1)

	// Mutate the returned copy
	buf1.Turns[0].Title = "Mutated"
	buf1.Turns = append(buf1.Turns, TurnEntry{Title: "Extra"})

	// Get another copy -- should still show original data
	buf2 := mgr.GetBuffer("proj-1", "sess-1")
	require.NotNil(t, buf2)
	require.Len(t, buf2.Turns, 1, "internal buffer should not be affected by external mutation")
	assert.Equal(t, "Original", buf2.Turns[0].Title, "internal turn title should not be mutated")
}

func TestFlushBuffer_RemovesBuffer(t *testing.T) {
	t.Parallel()

	mgr := NewSessionBufferManager(0)

	err := mgr.BufferTurn("proj-1", "sess-1", TurnEntry{
		Title:   "Turn to flush",
		Content: "Will be flushed",
	})
	require.NoError(t, err)
	assert.Equal(t, 1, mgr.ActiveSessions())

	// Flush
	flushed := mgr.FlushBuffer("proj-1", "sess-1")
	require.NotNil(t, flushed)
	assert.Equal(t, "sess-1", flushed.SessionID)
	assert.Equal(t, "proj-1", flushed.ProjectID)
	require.Len(t, flushed.Turns, 1)
	assert.Equal(t, "Turn to flush", flushed.Turns[0].Title)

	// Subsequent get returns nil
	assert.Nil(t, mgr.GetBuffer("proj-1", "sess-1"))
	assert.Equal(t, 0, mgr.Count("proj-1", "sess-1"))
	assert.Equal(t, 0, mgr.ActiveSessions())
}

func TestFlushBuffer_ReturnsNilForMissing(t *testing.T) {
	t.Parallel()

	mgr := NewSessionBufferManager(0)

	flushed := mgr.FlushBuffer("nonexistent-proj", "nonexistent-sess")
	assert.Nil(t, flushed)
}

func TestCount_EmptySession(t *testing.T) {
	t.Parallel()

	mgr := NewSessionBufferManager(0)

	assert.Equal(t, 0, mgr.Count("proj-1", "sess-nonexistent"))
}

func TestSessionIsolation(t *testing.T) {
	t.Parallel()

	mgr := NewSessionBufferManager(0)

	// Buffer turns in different sessions and projects
	err := mgr.BufferTurn("proj-1", "sess-1", TurnEntry{Title: "P1S1", Content: "Project 1 Session 1"})
	require.NoError(t, err)
	err = mgr.BufferTurn("proj-1", "sess-2", TurnEntry{Title: "P1S2", Content: "Project 1 Session 2"})
	require.NoError(t, err)
	err = mgr.BufferTurn("proj-2", "sess-1", TurnEntry{Title: "P2S1", Content: "Project 2 Session 1"})
	require.NoError(t, err)

	assert.Equal(t, 3, mgr.ActiveSessions())

	// Verify each session has exactly 1 turn with correct content
	buf11 := mgr.GetBuffer("proj-1", "sess-1")
	require.NotNil(t, buf11)
	require.Len(t, buf11.Turns, 1)
	assert.Equal(t, "P1S1", buf11.Turns[0].Title)

	buf12 := mgr.GetBuffer("proj-1", "sess-2")
	require.NotNil(t, buf12)
	require.Len(t, buf12.Turns, 1)
	assert.Equal(t, "P1S2", buf12.Turns[0].Title)

	buf21 := mgr.GetBuffer("proj-2", "sess-1")
	require.NotNil(t, buf21)
	require.Len(t, buf21.Turns, 1)
	assert.Equal(t, "P2S1", buf21.Turns[0].Title)

	// Flush one session, verify others are unaffected
	flushed := mgr.FlushBuffer("proj-1", "sess-1")
	require.NotNil(t, flushed)
	assert.Equal(t, 2, mgr.ActiveSessions())

	assert.Nil(t, mgr.GetBuffer("proj-1", "sess-1"))
	assert.NotNil(t, mgr.GetBuffer("proj-1", "sess-2"))
	assert.NotNil(t, mgr.GetBuffer("proj-2", "sess-1"))
}

func TestConcurrentAccess(t *testing.T) {
	t.Parallel()

	mgr := NewSessionBufferManager(100)

	const (
		numGoroutines = 20
		turnsPerGo    = 50
	)

	var wg sync.WaitGroup

	// Concurrent writers to the same session
	for g := 0; g < numGoroutines; g++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for i := 0; i < turnsPerGo; i++ {
				entry := TurnEntry{
					Title:   fmt.Sprintf("goroutine-%d-turn-%d", id, i),
					Content: fmt.Sprintf("content from goroutine %d turn %d", id, i),
					Outcome: OutcomeSuccess,
				}
				_ = mgr.BufferTurn("proj-concurrent", "sess-concurrent", entry)
			}
		}(g)
	}

	// Concurrent readers
	for g := 0; g < numGoroutines/2; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < turnsPerGo; i++ {
				_ = mgr.GetBuffer("proj-concurrent", "sess-concurrent")
				_ = mgr.Count("proj-concurrent", "sess-concurrent")
				_ = mgr.ActiveSessions()
			}
		}()
	}

	// Concurrent flushers on different sessions to avoid interfering with writers
	for g := 0; g < numGoroutines/4; g++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			sessID := fmt.Sprintf("flush-sess-%d", id)
			for i := 0; i < turnsPerGo; i++ {
				entry := TurnEntry{
					Title:   fmt.Sprintf("flush-turn-%d", i),
					Content: "flush content",
				}
				_ = mgr.BufferTurn("proj-flush", sessID, entry)
			}
			_ = mgr.FlushBuffer("proj-flush", sessID)
		}(g)
	}

	wg.Wait()

	// After all writers finish, the shared session should have exactly maxTurns entries
	// because 20 goroutines * 50 turns = 1000, capped at 100
	count := mgr.Count("proj-concurrent", "sess-concurrent")
	assert.Equal(t, 100, count, "buffer should be capped at maxTurns")

	// Flushed sessions should be gone
	for g := 0; g < numGoroutines/4; g++ {
		sessID := fmt.Sprintf("flush-sess-%d", g)
		assert.Nil(t, mgr.GetBuffer("proj-flush", sessID), "flushed session %s should be nil", sessID)
	}
}

func TestBufferTurn_PreservesExplicitTimestamp(t *testing.T) {
	t.Parallel()

	mgr := NewSessionBufferManager(0)

	explicit := time.Date(2025, 6, 15, 10, 30, 0, 0, time.UTC)
	entry := TurnEntry{
		Title:     "With timestamp",
		Content:   "Has explicit timestamp",
		Timestamp: explicit,
	}

	err := mgr.BufferTurn("proj-1", "sess-1", entry)
	require.NoError(t, err)

	buf := mgr.GetBuffer("proj-1", "sess-1")
	require.NotNil(t, buf)
	require.Len(t, buf.Turns, 1)
	assert.Equal(t, explicit, buf.Turns[0].Timestamp, "explicit timestamp should be preserved")
}
