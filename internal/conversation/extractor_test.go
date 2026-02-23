package conversation

import (
	"testing"
	"time"
)

func TestExtractor_ExtractFileReferences(t *testing.T) {
	extractor := NewExtractor()

	tests := []struct {
		name       string
		msg        RawMessage
		wantLen    int
		wantPath   string
		wantAction Action
	}{
		{
			name: "Read tool call",
			msg: RawMessage{
				ToolCalls: []ToolCall{
					{
						Name:   "Read",
						Params: map[string]string{"file_path": "/path/to/file.go"},
					},
				},
			},
			wantLen:    1,
			wantPath:   "/path/to/file.go",
			wantAction: ActionRead,
		},
		{
			name: "Edit tool call",
			msg: RawMessage{
				ToolCalls: []ToolCall{
					{
						Name:   "Edit",
						Params: map[string]string{"file_path": "/path/to/edited.go"},
					},
				},
			},
			wantLen:    1,
			wantPath:   "/path/to/edited.go",
			wantAction: ActionEdited,
		},
		{
			name: "Write tool call",
			msg: RawMessage{
				ToolCalls: []ToolCall{
					{
						Name:   "Write",
						Params: map[string]string{"file_path": "/path/to/new.go"},
					},
				},
			},
			wantLen:    1,
			wantPath:   "/path/to/new.go",
			wantAction: ActionCreated,
		},
		{
			name: "Bash rm command",
			msg: RawMessage{
				ToolCalls: []ToolCall{
					{
						Name:   "Bash",
						Params: map[string]string{"command": "rm /path/to/deleted.txt"},
					},
				},
			},
			wantLen:    1,
			wantPath:   "/path/to/deleted.txt",
			wantAction: ActionDeleted,
		},
		{
			name: "File path in content",
			msg: RawMessage{
				Content: "I'll update the file at internal/service.go to fix this",
			},
			wantLen:    1,
			wantPath:   "internal/service.go",
			wantAction: ActionRead,
		},
		{
			name: "Multiple tool calls",
			msg: RawMessage{
				ToolCalls: []ToolCall{
					{
						Name:   "Read",
						Params: map[string]string{"file_path": "/a.go"},
					},
					{
						Name:   "Edit",
						Params: map[string]string{"file_path": "/b.go"},
					},
				},
			},
			wantLen: 2,
		},
		{
			name: "No file references",
			msg: RawMessage{
				Content: "Just a simple message with no files",
			},
			wantLen: 0,
		},
		{
			name: "Glob tool call",
			msg: RawMessage{
				ToolCalls: []ToolCall{
					{
						Name:   "Glob",
						Params: map[string]string{"path": "/search/path"},
					},
				},
			},
			wantLen:    1,
			wantPath:   "/search/path",
			wantAction: ActionRead,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			refs := extractor.ExtractFileReferences(tt.msg)
			if len(refs) != tt.wantLen {
				t.Errorf("ExtractFileReferences() got %d refs, want %d", len(refs), tt.wantLen)
				return
			}
			if tt.wantLen > 0 && tt.wantPath != "" {
				found := false
				for _, ref := range refs {
					if ref.Path == tt.wantPath {
						found = true
						if ref.Action != tt.wantAction {
							t.Errorf("ref.Action = %v, want %v", ref.Action, tt.wantAction)
						}
						break
					}
				}
				if !found {
					t.Errorf("ExtractFileReferences() did not find path %q", tt.wantPath)
				}
			}
		})
	}
}

func TestExtractor_ExtractFileReferences_Deduplication(t *testing.T) {
	extractor := NewExtractor()

	// Same file referenced multiple times
	msg := RawMessage{
		ToolCalls: []ToolCall{
			{
				Name:   "Read",
				Params: map[string]string{"file_path": "/same/file.go"},
			},
			{
				Name:   "Edit",
				Params: map[string]string{"file_path": "/same/file.go"},
			},
		},
	}

	refs := extractor.ExtractFileReferences(msg)
	if len(refs) != 1 {
		t.Errorf("ExtractFileReferences() got %d refs, want 1 (deduplicated)", len(refs))
	}
	// Should prefer Edit over Read action
	if len(refs) > 0 && refs[0].Action != ActionEdited {
		t.Errorf("ref.Action = %v, want %v (should prefer more specific action)", refs[0].Action, ActionEdited)
	}
}

func TestExtractor_ExtractCommitReferences(t *testing.T) {
	extractor := NewExtractor()

	tests := []struct {
		name    string
		msg     RawMessage
		wantLen int
		wantSHA string
	}{
		{
			name: "git commit output",
			msg: RawMessage{
				ToolCalls: []ToolCall{
					{
						Name:   "Bash",
						Params: map[string]string{"command": "git commit -m 'test'"},
						Result: "[main abc1234] test commit message\n 1 file changed, 10 insertions(+)",
					},
				},
			},
			wantLen: 1,
			wantSHA: "abc1234",
		},
		{
			name: "git log output",
			msg: RawMessage{
				ToolCalls: []ToolCall{
					{
						Name:   "Bash",
						Params: map[string]string{"command": "git log -1"},
						Result: "commit def5678901234567890123456789012345678901\nAuthor: Test <test@test.com>\nDate:   Mon Jan 1 10:00:00 2025\n\n    Initial commit",
					},
				},
			},
			wantLen: 1,
			wantSHA: "def5678901234567890123456789012345678901",
		},
		{
			name: "no git commands",
			msg: RawMessage{
				ToolCalls: []ToolCall{
					{
						Name:   "Bash",
						Params: map[string]string{"command": "ls -la"},
						Result: "total 0",
					},
				},
			},
			wantLen: 0,
		},
		{
			name:    "empty message",
			msg:     RawMessage{},
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			refs := extractor.ExtractCommitReferences(tt.msg)
			if len(refs) != tt.wantLen {
				t.Errorf("ExtractCommitReferences() got %d refs, want %d", len(refs), tt.wantLen)
				return
			}
			if tt.wantLen > 0 && tt.wantSHA != "" {
				if refs[0].SHA != tt.wantSHA {
					t.Errorf("refs[0].SHA = %q, want %q", refs[0].SHA, tt.wantSHA)
				}
			}
		})
	}
}

func TestExtractor_ExtractMetadata(t *testing.T) {
	extractor := NewExtractor()

	msg := RawMessage{
		Timestamp: time.Now(),
		Role:      RoleAssistant,
		Content:   "I'll update service.go",
		ToolCalls: []ToolCall{
			{
				Name:   "Read",
				Params: map[string]string{"file_path": "/path/to/file.go"},
			},
			{
				Name:   "Bash",
				Params: map[string]string{"command": "git commit -m 'fix'"},
				Result: "[main abc1234] fix",
			},
		},
	}

	files, commits := extractor.ExtractMetadata(msg)

	if len(files) == 0 {
		t.Error("ExtractMetadata() returned no file references")
	}
	if len(commits) == 0 {
		t.Error("ExtractMetadata() returned no commit references")
	}
}

func TestIsValidFilePath(t *testing.T) {
	tests := []struct {
		path  string
		valid bool
	}{
		{"internal/service.go", true},
		{"file.txt", true},
		{"path/to/file.json", true},
		{"http://example.com", false},
		{"https://example.com", false},
		{"v1.0.0", false},
		{"v2.3.4", false},
		{"e.g.", false},
		{"i.e.", false},
		{"ab", false},          // too short
		{"noextension", false}, // no extension
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := isValidFilePath(tt.path)
			if got != tt.valid {
				t.Errorf("isValidFilePath(%q) = %v, want %v", tt.path, got, tt.valid)
			}
		})
	}
}

func TestIsCommonHexValue(t *testing.T) {
	tests := []struct {
		value  string
		common bool
	}{
		{"0000000", true},
		{"fffffff", true},
		{"1234567", true},
		{"abcdefg", true},
		{"abc1234", false},
		{"def5678", false},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			got := isCommonHexValue(tt.value)
			if got != tt.common {
				t.Errorf("isCommonHexValue(%q) = %v, want %v", tt.value, got, tt.common)
			}
		})
	}
}

func TestExtractLineRanges(t *testing.T) {
	tests := []struct {
		name   string
		params map[string]string
		want   []string
	}{
		{
			name:   "offset and limit",
			params: map[string]string{"offset": "10", "limit": "50"},
			want:   []string{"10-50"},
		},
		{
			name:   "offset only",
			params: map[string]string{"offset": "10"},
			want:   []string{"10"},
		},
		{
			name:   "no range params",
			params: map[string]string{"file_path": "/test.go"},
			want:   nil,
		},
		{
			name:   "empty params",
			params: map[string]string{},
			want:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractLineRanges(tt.params)
			if len(got) != len(tt.want) {
				t.Errorf("extractLineRanges() got %v, want %v", got, tt.want)
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("extractLineRanges()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}
