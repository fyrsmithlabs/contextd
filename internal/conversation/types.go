// Package conversation provides functionality for indexing and searching
// past Claude Code sessions. It parses JSONL conversation files, extracts
// messages and decisions, and stores them in a vector database for semantic search.
package conversation

import (
	"context"
	"time"
)

// DocumentType represents the type of conversation document.
type DocumentType string

const (
	// TypeMessage is an individual user or assistant message.
	TypeMessage DocumentType = "message"
	// TypeDecision is an extracted decision from conversation.
	TypeDecision DocumentType = "decision"
	// TypeSummary is a session summary (future enhancement).
	TypeSummary DocumentType = "summary"
)

// Role represents the role of a message sender.
type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
)

// Action represents the type of file operation.
type Action string

const (
	ActionRead    Action = "read"
	ActionEdited  Action = "edited"
	ActionCreated Action = "created"
	ActionDeleted Action = "deleted"
)

// RawMessage represents a parsed message from a Claude Code JSONL file.
type RawMessage struct {
	SessionID  string     `json:"session_id"`
	UUID       string     `json:"uuid"`
	Timestamp  time.Time  `json:"timestamp"`
	Role       Role       `json:"role"`
	Content    string     `json:"content"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	GitBranch  string     `json:"git_branch,omitempty"`
	ParentUUID string     `json:"parent_uuid,omitempty"`
}

// ToolCall represents a tool invocation within a message.
type ToolCall struct {
	Name   string            `json:"name"`
	Params map[string]string `json:"params,omitempty"`
	Result string            `json:"result,omitempty"`
}

// FileReference represents a file discussed in a conversation.
type FileReference struct {
	Path       string   `json:"path"`
	LineRanges []string `json:"line_ranges,omitempty"`
	Action     Action   `json:"action"`
}

// CommitReference represents a git commit linked to a conversation.
type CommitReference struct {
	SHA     string `json:"sha"`
	Message string `json:"message,omitempty"`
}

// ConversationDocument is the base document type stored in the vector database.
type ConversationDocument struct {
	// Identity
	ID        string       `json:"id"`
	SessionID string       `json:"session_id"`
	Type      DocumentType `json:"type"`
	Timestamp time.Time    `json:"timestamp"`

	// Content (embedded for search)
	Content string `json:"content"`

	// Context tags
	Tags   []string `json:"tags,omitempty"`
	Domain string   `json:"domain,omitempty"`

	// Cross-references
	FilesDiscussed []FileReference   `json:"files_discussed,omitempty"`
	CommitsMade    []CommitReference `json:"commits_made,omitempty"`

	// Metadata
	IndexedAt        time.Time `json:"indexed_at"`
	ExtractionMethod string    `json:"extraction_method"`
}

// MessageDocument represents an individual message from a conversation.
type MessageDocument struct {
	ConversationDocument

	// Message-specific fields
	Role         Role   `json:"role"`
	MessageUUID  string `json:"message_uuid"`
	MessageIndex int    `json:"message_index"`
}

// DecisionDocument represents an extracted decision from a conversation.
type DecisionDocument struct {
	ConversationDocument

	// Decision-specific fields
	Summary           string   `json:"summary"`
	Alternatives      []string `json:"alternatives,omitempty"`
	Reasoning         string   `json:"reasoning,omitempty"`
	Confidence        float64  `json:"confidence"`
	SourceMessageUUID string   `json:"source_message_uuid"`
}

// IndexOptions configures how conversations are indexed.
type IndexOptions struct {
	ProjectPath string   `json:"project_path"`
	TenantID    string   `json:"tenant_id"`
	SessionIDs  []string `json:"session_ids,omitempty"` // Empty = all sessions
	EnableLLM   bool     `json:"enable_llm"`
	Force       bool     `json:"force"` // Reindex existing
}

// IndexResult contains the results of an indexing operation.
type IndexResult struct {
	SessionsIndexed    int      `json:"sessions_indexed"`
	MessagesIndexed    int      `json:"messages_indexed"`
	DecisionsExtracted int      `json:"decisions_extracted"`
	FilesReferenced    []string `json:"files_referenced"`
	Errors             []error  `json:"errors,omitempty"`
}

// SearchOptions configures how conversations are searched.
type SearchOptions struct {
	Query       string         `json:"query"`
	ProjectPath string         `json:"project_path"`
	TenantID    string         `json:"tenant_id"`
	Types       []DocumentType `json:"types,omitempty"` // TypeMessage, TypeDecision, TypeSummary
	Tags        []string       `json:"tags,omitempty"`
	FilePath    string         `json:"file_path,omitempty"`
	Domain      string         `json:"domain,omitempty"`
	Limit       int            `json:"limit"`
}

// SearchResult contains the results of a search operation.
type SearchResult struct {
	Query   string           `json:"query"`
	Results []SearchHit      `json:"results"`
	Total   int              `json:"total"`
	Took    time.Duration    `json:"took"`
}

// SearchHit represents a single search result.
type SearchHit struct {
	Document ConversationDocument `json:"document"`
	Score    float64              `json:"score"`
}

// ConversationIndexMetadata tracks the indexing state for a project.
type ConversationIndexMetadata struct {
	ProjectPath       string    `json:"project_path"`
	TenantID          string    `json:"tenant_id"`
	LastIndexedAt     time.Time `json:"last_indexed_at"`
	SessionsIndexed   int       `json:"sessions_indexed"`
	MessagesIndexed   int       `json:"messages_indexed"`
	DecisionsIndexed  int       `json:"decisions_indexed"`
	IndexedSessionIDs []string  `json:"indexed_session_ids"`
}

// ConversationParser defines the interface for parsing conversation files.
type ConversationParser interface {
	// Parse reads a JSONL file and extracts messages.
	Parse(path string) ([]RawMessage, error)

	// ParseAll reads all JSONL files in a directory.
	ParseAll(dir string) (map[string][]RawMessage, error)
}

// ConversationService defines the interface for conversation operations.
type ConversationService interface {
	// Index processes and stores conversations for a project.
	Index(ctx context.Context, opts IndexOptions) (*IndexResult, error)

	// Search finds relevant conversations.
	Search(ctx context.Context, opts SearchOptions) (*SearchResult, error)
}
