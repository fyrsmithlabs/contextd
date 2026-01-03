package conversation

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParser_Parse(t *testing.T) {
	// Create temp directory with test JSONL file
	tmpDir := t.TempDir()

	// Create test JSONL content - note: tool_result only messages may be skipped as they lack text content
	testContent := `{"type":"user","message":{"id":"msg1","content":[{"type":"text","text":"Hello, help me fix this bug"}],"model":"claude-3","role":"user"},"timestamp":"2025-01-01T10:00:00Z","uuid":"uuid-1"}
{"type":"assistant","message":{"id":"msg2","content":[{"type":"text","text":"I'll help you fix that bug. Let me read the file first."},{"type":"tool_use","id":"tool1","name":"Read","input":{"file_path":"/path/to/file.go"}}],"model":"claude-3","role":"assistant"},"timestamp":"2025-01-01T10:00:30Z","uuid":"uuid-2"}`

	testFile := filepath.Join(tmpDir, "test-session.jsonl")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	parser := NewParser()
	messages, err := parser.Parse(testFile)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if len(messages) != 2 {
		t.Errorf("Parse() got %d messages, want 2", len(messages))
	}

	// Check first message
	if messages[0].Role != RoleUser {
		t.Errorf("messages[0].Role = %v, want %v", messages[0].Role, RoleUser)
	}
	if messages[0].Content != "Hello, help me fix this bug" {
		t.Errorf("messages[0].Content = %v, want 'Hello, help me fix this bug'", messages[0].Content)
	}

	// Check second message (assistant with tool call)
	if messages[1].Role != RoleAssistant {
		t.Errorf("messages[1].Role = %v, want %v", messages[1].Role, RoleAssistant)
	}
	if len(messages[1].ToolCalls) != 1 {
		t.Errorf("messages[1].ToolCalls = %d, want 1", len(messages[1].ToolCalls))
	}
	if messages[1].ToolCalls[0].Name != "Read" {
		t.Errorf("messages[1].ToolCalls[0].Name = %v, want 'Read'", messages[1].ToolCalls[0].Name)
	}
}

func TestParser_Parse_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "empty.jsonl")
	if err := os.WriteFile(testFile, []byte(""), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	parser := NewParser()
	messages, err := parser.Parse(testFile)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if len(messages) != 0 {
		t.Errorf("Parse() got %d messages, want 0", len(messages))
	}
}

func TestParser_Parse_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "invalid.jsonl")
	if err := os.WriteFile(testFile, []byte("not valid json\n"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	parser := NewParser()
	messages, err := parser.Parse(testFile)
	// Parser should skip invalid lines, not error
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if len(messages) != 0 {
		t.Errorf("Parse() got %d messages, want 0 (invalid lines skipped)", len(messages))
	}
}

func TestParser_Parse_NonexistentFile(t *testing.T) {
	parser := NewParser()
	_, err := parser.Parse("/nonexistent/file.jsonl")
	if err == nil {
		t.Error("Parse() expected error for nonexistent file")
	}
}

func TestParser_ParseAll(t *testing.T) {
	tmpDir := t.TempDir()

	// Create two session files
	session1 := `{"type":"user","message":{"id":"msg1","content":[{"type":"text","text":"Session 1 message"}],"model":"claude-3","role":"user"},"timestamp":"2025-01-01T10:00:00Z","uuid":"uuid-1"}`
	session2 := `{"type":"assistant","message":{"id":"msg2","content":[{"type":"text","text":"Session 2 message"}],"model":"claude-3","role":"assistant"},"timestamp":"2025-01-01T11:00:00Z","uuid":"uuid-2"}`

	if err := os.WriteFile(filepath.Join(tmpDir, "session1.jsonl"), []byte(session1), 0644); err != nil {
		t.Fatalf("failed to write session1: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "session2.jsonl"), []byte(session2), 0644); err != nil {
		t.Fatalf("failed to write session2: %v", err)
	}

	parser := NewParser()
	sessionMessages, err := parser.ParseAll(tmpDir)
	if err != nil {
		t.Fatalf("ParseAll() error = %v", err)
	}

	if len(sessionMessages) != 2 {
		t.Errorf("ParseAll() got %d sessions, want 2", len(sessionMessages))
	}

	// Check that each session has messages
	for sessionID, msgs := range sessionMessages {
		if len(msgs) != 1 {
			t.Errorf("session %s has %d messages, want 1", sessionID, len(msgs))
		}
	}
}

func TestParser_ParseAll_EmptyDir(t *testing.T) {
	tmpDir := t.TempDir()

	parser := NewParser()
	sessionMessages, err := parser.ParseAll(tmpDir)
	if err != nil {
		t.Fatalf("ParseAll() error = %v", err)
	}

	if len(sessionMessages) != 0 {
		t.Errorf("ParseAll() got %d sessions, want 0", len(sessionMessages))
	}
}

func TestParser_ParseAll_NonexistentDir(t *testing.T) {
	parser := NewParser()
	// Note: ParseAll doesn't error on nonexistent dirs since Glob returns empty
	sessionMessages, err := parser.ParseAll("/nonexistent/directory")
	if err != nil {
		t.Fatalf("ParseAll() unexpected error: %v", err)
	}
	// Should return empty map
	if len(sessionMessages) != 0 {
		t.Errorf("ParseAll() got %d sessions, want 0", len(sessionMessages))
	}
}

func TestExtractSessionID(t *testing.T) {
	tests := []struct {
		name     string
		filepath string
		want     string
	}{
		{
			name:     "simple jsonl file",
			filepath: "/home/user/.claude/projects/test/session-abc123.jsonl",
			want:     "session-abc123",
		},
		{
			name:     "uuid-style session",
			filepath: "/data/conversations/550e8400-e29b-41d4-a716-446655440000.jsonl",
			want:     "550e8400-e29b-41d4-a716-446655440000",
		},
		{
			name:     "simple name",
			filepath: "test.jsonl",
			want:     "test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractSessionID(tt.filepath)
			if got != tt.want {
				t.Errorf("extractSessionID(%q) = %q, want %q", tt.filepath, got, tt.want)
			}
		})
	}
}

func TestParser_TimestampParsing(t *testing.T) {
	tmpDir := t.TempDir()

	// Test various timestamp formats
	testContent := `{"type":"user","message":{"id":"msg1","content":[{"type":"text","text":"Test"}],"model":"claude-3","role":"user"},"timestamp":"2025-01-01T10:00:00Z","uuid":"uuid-1"}`

	testFile := filepath.Join(tmpDir, "test.jsonl")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	parser := NewParser()
	messages, err := parser.Parse(testFile)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if len(messages) != 1 {
		t.Fatalf("Parse() got %d messages, want 1", len(messages))
	}

	expected := time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)
	if !messages[0].Timestamp.Equal(expected) {
		t.Errorf("messages[0].Timestamp = %v, want %v", messages[0].Timestamp, expected)
	}
}
