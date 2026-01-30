package reasoningbank

import (
	"fmt"
	"sync"
	"time"
)

// TurnEntry represents a single turn buffered for session summarization.
type TurnEntry struct {
	Title     string
	Content   string
	Outcome   Outcome
	Tags      []string
	Timestamp time.Time
}

// SessionBuffer holds buffered turns for a single session.
type SessionBuffer struct {
	SessionID   string
	ProjectID   string
	SessionDate time.Time
	Turns       []TurnEntry
}

// SessionBufferManager manages in-memory buffers for session-level memory accumulation.
// Thread-safe for concurrent access from multiple MCP tool calls.
type SessionBufferManager struct {
	mu       sync.RWMutex
	buffers  map[string]*SessionBuffer // keyed by "projectID:sessionID"
	maxTurns int
}

// NewSessionBufferManager creates a new buffer manager.
// maxTurns limits the number of turns per session buffer (0 = unlimited).
func NewSessionBufferManager(maxTurns int) *SessionBufferManager {
	return &SessionBufferManager{
		buffers:  make(map[string]*SessionBuffer),
		maxTurns: maxTurns,
	}
}

// bufferKey returns the map key for a project+session pair.
func bufferKey(projectID, sessionID string) string {
	return projectID + ":" + sessionID
}

// BufferTurn adds a turn entry to the session buffer.
// Creates the buffer if it doesn't exist yet.
// If maxTurns is exceeded, the oldest turn is dropped.
func (m *SessionBufferManager) BufferTurn(projectID, sessionID string, entry TurnEntry) error {
	if projectID == "" {
		return ErrEmptyProjectID
	}
	if sessionID == "" {
		return fmt.Errorf("session ID cannot be empty")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	key := bufferKey(projectID, sessionID)
	buf, ok := m.buffers[key]
	if !ok {
		buf = &SessionBuffer{
			SessionID:   sessionID,
			ProjectID:   projectID,
			SessionDate: time.Now(),
			Turns:       make([]TurnEntry, 0, 64),
		}
		m.buffers[key] = buf
	}

	// Set timestamp if not provided
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}

	buf.Turns = append(buf.Turns, entry)

	// Enforce max turns limit by dropping oldest
	if m.maxTurns > 0 && len(buf.Turns) > m.maxTurns {
		excess := len(buf.Turns) - m.maxTurns
		buf.Turns = buf.Turns[excess:]
	}

	return nil
}

// GetBuffer returns the current buffer for a session. Returns nil if no buffer exists.
func (m *SessionBufferManager) GetBuffer(projectID, sessionID string) *SessionBuffer {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := bufferKey(projectID, sessionID)
	buf, ok := m.buffers[key]
	if !ok {
		return nil
	}

	// Return a copy to avoid external mutation
	cp := *buf
	cp.Turns = make([]TurnEntry, len(buf.Turns))
	copy(cp.Turns, buf.Turns)
	return &cp
}

// FlushBuffer removes and returns the buffer for a session.
// Returns nil if no buffer exists.
func (m *SessionBufferManager) FlushBuffer(projectID, sessionID string) *SessionBuffer {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := bufferKey(projectID, sessionID)
	buf, ok := m.buffers[key]
	if !ok {
		return nil
	}

	delete(m.buffers, key)
	return buf
}

// Count returns the number of buffered turns for a session.
// Returns 0 if no buffer exists.
func (m *SessionBufferManager) Count(projectID, sessionID string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := bufferKey(projectID, sessionID)
	buf, ok := m.buffers[key]
	if !ok {
		return 0
	}
	return len(buf.Turns)
}

// ActiveSessions returns the number of sessions with active buffers.
func (m *SessionBufferManager) ActiveSessions() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.buffers)
}
