// Package agent provides test agent functionality for contextd validation.
package agent

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ConversationEntry represents a single entry in a Claude Code JSONL export.
type ConversationEntry struct {
	Type        string          `json:"type"`
	UUID        string          `json:"uuid"`
	ParentUUID  *string         `json:"parentUuid"`
	SessionID   string          `json:"sessionId"`
	Timestamp   time.Time       `json:"timestamp"`
	Message     json.RawMessage `json:"message"`
	UserType    string          `json:"userType"`
	GitBranch   string          `json:"gitBranch"`
	CWD         string          `json:"cwd"`
	Summary     string          `json:"summary,omitempty"`
	ToolResult  json.RawMessage `json:"toolUseResult,omitempty"`
}

// MessageContent represents the content of a message.
type MessageContent struct {
	Role    string        `json:"role"`
	Content interface{}   `json:"content"` // Can be string or []ContentBlock
	Model   string        `json:"model,omitempty"`
}

// ContentBlock represents a content block in assistant messages.
type ContentBlock struct {
	Type      string          `json:"type"`
	Text      string          `json:"text,omitempty"`
	ToolUse   *ToolUseBlock   `json:"tool_use,omitempty"`
	ID        string          `json:"id,omitempty"`
	Name      string          `json:"name,omitempty"`
	Input     json.RawMessage `json:"input,omitempty"`
}

// ToolUseBlock represents a tool use in content.
type ToolUseBlock struct {
	ID    string          `json:"id"`
	Name  string          `json:"name"`
	Input json.RawMessage `json:"input"`
}

// ContextdToolCall represents a contextd MCP tool invocation.
type ContextdToolCall struct {
	Timestamp time.Time
	SessionID string
	Tool      string            // memory_search, memory_record, memory_feedback, etc.
	Input     map[string]interface{}
	Success   bool
	Output    string
}

// ConversationStats holds statistics about a conversation.
type ConversationStats struct {
	SessionID          string
	StartTime          time.Time
	EndTime            time.Time
	Duration           time.Duration
	UserMessages       int
	AssistantMessages  int
	ToolCalls          int
	ContextdToolCalls  []ContextdToolCall
	MemorySearches     int
	MemoryRecords      int
	MemoryFeedbacks    int
	CheckpointSaves    int
	CheckpointResumes  int
	Errors             int
}

// ParseConversation parses a JSONL conversation file.
func ParseConversation(path string) (*ConversationStats, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}
	defer file.Close()

	stats := &ConversationStats{
		ContextdToolCalls: make([]ContextdToolCall, 0),
	}

	scanner := bufio.NewScanner(file)
	// Increase buffer size for large lines
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 10*1024*1024) // 10MB max line size

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var entry ConversationEntry
		if err := json.Unmarshal(line, &entry); err != nil {
			// Skip malformed lines
			continue
		}

		// Track session info
		if stats.SessionID == "" && entry.SessionID != "" {
			stats.SessionID = entry.SessionID
		}

		// Track timestamps
		if !entry.Timestamp.IsZero() {
			if stats.StartTime.IsZero() || entry.Timestamp.Before(stats.StartTime) {
				stats.StartTime = entry.Timestamp
			}
			if entry.Timestamp.After(stats.EndTime) {
				stats.EndTime = entry.Timestamp
			}
		}

		switch entry.Type {
		case "user":
			stats.UserMessages++
		case "assistant":
			stats.AssistantMessages++
			parseAssistantMessage(entry, stats)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan file: %w", err)
	}

	stats.Duration = stats.EndTime.Sub(stats.StartTime)
	return stats, nil
}

func parseAssistantMessage(entry ConversationEntry, stats *ConversationStats) {
	if len(entry.Message) == 0 {
		return
	}

	var msg MessageContent
	if err := json.Unmarshal(entry.Message, &msg); err != nil {
		return
	}

	// Content can be a string or array of content blocks
	switch content := msg.Content.(type) {
	case []interface{}:
		for _, item := range content {
			if block, ok := item.(map[string]interface{}); ok {
				parseContentBlock(block, entry, stats)
			}
		}
	}
}

func parseContentBlock(block map[string]interface{}, entry ConversationEntry, stats *ConversationStats) {
	blockType, _ := block["type"].(string)
	if blockType != "tool_use" {
		return
	}

	stats.ToolCalls++

	name, _ := block["name"].(string)
	id, _ := block["id"].(string)

	// Check if it's a contextd tool
	if !isContextdTool(name) {
		return
	}

	var input map[string]interface{}
	if inputRaw, ok := block["input"].(map[string]interface{}); ok {
		input = inputRaw
	}

	toolCall := ContextdToolCall{
		Timestamp: entry.Timestamp,
		SessionID: entry.SessionID,
		Tool:      name,
		Input:     input,
		Success:   true, // Default, would need tool result to verify
	}

	stats.ContextdToolCalls = append(stats.ContextdToolCalls, toolCall)

	// Count specific tool types
	switch {
	case strings.Contains(name, "memory_search"):
		stats.MemorySearches++
	case strings.Contains(name, "memory_record"):
		stats.MemoryRecords++
	case strings.Contains(name, "memory_feedback"):
		stats.MemoryFeedbacks++
	case strings.Contains(name, "checkpoint_save"):
		stats.CheckpointSaves++
	case strings.Contains(name, "checkpoint_resume"):
		stats.CheckpointResumes++
	}

	_ = id // For future use with tool results
}

func isContextdTool(name string) bool {
	contextdTools := []string{
		"mcp__contextd__memory_search",
		"mcp__contextd__memory_record",
		"mcp__contextd__memory_feedback",
		"mcp__contextd__checkpoint_save",
		"mcp__contextd__checkpoint_list",
		"mcp__contextd__checkpoint_resume",
		"mcp__contextd__remediation_search",
		"mcp__contextd__remediation_record",
		"mcp__contextd__troubleshoot_diagnose",
		"mcp__contextd__repository_index",
		"mcp__contextd__repository_search",
	}

	for _, tool := range contextdTools {
		if name == tool {
			return true
		}
	}
	return false
}

// ParseConversationsDir parses all JSONL files in a directory.
func ParseConversationsDir(dir string) ([]*ConversationStats, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read dir: %w", err)
	}

	var results []*ConversationStats
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".jsonl") {
			continue
		}
		// Skip agent files (subagent conversations)
		if strings.HasPrefix(entry.Name(), "agent-") {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		stats, err := ParseConversation(path)
		if err != nil {
			// Log error but continue
			fmt.Fprintf(os.Stderr, "Warning: failed to parse %s: %v\n", path, err)
			continue
		}

		// Only include conversations with contextd usage
		if len(stats.ContextdToolCalls) > 0 {
			results = append(results, stats)
		}
	}

	return results, nil
}

// GenerateScenarioFromStats creates a test scenario from conversation statistics.
func GenerateScenarioFromStats(stats *ConversationStats) *Scenario {
	// Determine persona based on feedback patterns
	var feedbackStyle string
	var successRate float64

	// Calculate feedback ratio (would need more data for accurate calculation)
	totalFeedback := stats.MemoryFeedbacks
	if totalFeedback == 0 {
		feedbackStyle = "realistic"
		successRate = 0.7
	} else {
		// Default to realistic since we don't have positive/negative breakdown
		feedbackStyle = "realistic"
		successRate = 0.7
	}

	// Build actions from tool calls
	actions := make([]Action, 0)
	for _, tc := range stats.ContextdToolCalls {
		action := toolCallToAction(tc)
		if action != nil {
			actions = append(actions, *action)
		}
	}

	// Limit to reasonable number of actions
	if len(actions) > 50 {
		actions = actions[:50]
	}

	// Safe substring for session ID (avoid panic if < 8 chars)
	sessionPrefix := stats.SessionID
	if len(sessionPrefix) > 8 {
		sessionPrefix = sessionPrefix[:8]
	}

	return &Scenario{
		Name:        fmt.Sprintf("replay_%s", sessionPrefix),
		Description: fmt.Sprintf("Replay of session %s (%d contextd calls)", stats.SessionID, len(stats.ContextdToolCalls)),
		Persona: Persona{
			Name:          "ReplayUser",
			Description:   fmt.Sprintf("Replaying session from %s", stats.StartTime.Format("2006-01-02")),
			FeedbackStyle: feedbackStyle,
			SuccessRate:   successRate,
		},
		ProjectID: "test-replay",
		MaxTurns:  len(actions) + 10,
		Actions:   actions,
		Assertions: []Assertion{
			{
				Type:    "memory_count",
				Value:   stats.MemoryRecords,
				Message: fmt.Sprintf("Should record %d memories", stats.MemoryRecords),
			},
		},
	}
}

func toolCallToAction(tc ContextdToolCall) *Action {
	switch {
	case strings.Contains(tc.Tool, "memory_record"):
		title, _ := tc.Input["title"].(string)
		content, _ := tc.Input["content"].(string)
		outcome, _ := tc.Input["outcome"].(string)
		if outcome == "" {
			outcome = "success"
		}
		var tags []string
		if tagsRaw, ok := tc.Input["tags"].([]interface{}); ok {
			for _, t := range tagsRaw {
				if s, ok := t.(string); ok {
					tags = append(tags, s)
				}
			}
		}
		return &Action{
			Type: "record",
			Args: map[string]interface{}{
				"title":   title,
				"content": content,
				"outcome": outcome,
				"tags":    tags,
			},
		}

	case strings.Contains(tc.Tool, "memory_search"):
		query, _ := tc.Input["query"].(string)
		limit := 5
		if l, ok := tc.Input["limit"].(float64); ok {
			limit = int(l)
		}
		return &Action{
			Type: "search",
			Args: map[string]interface{}{
				"query": query,
				"limit": limit,
			},
		}

	case strings.Contains(tc.Tool, "memory_feedback"):
		// Use "last" instead of actual memory ID since we're replaying
		// and the original IDs won't exist
		helpful, _ := tc.Input["helpful"].(bool)
		return &Action{
			Type: "feedback",
			Args: map[string]interface{}{
				"memory_id": "last",
				"helpful":   helpful,
			},
		}
	}

	return nil
}

// AnalyzeConversations provides aggregate statistics across conversations.
func AnalyzeConversations(stats []*ConversationStats) map[string]interface{} {
	totalSessions := len(stats)
	totalContextdCalls := 0
	totalSearches := 0
	totalRecords := 0
	totalFeedbacks := 0
	totalCheckpoints := 0

	for _, s := range stats {
		totalContextdCalls += len(s.ContextdToolCalls)
		totalSearches += s.MemorySearches
		totalRecords += s.MemoryRecords
		totalFeedbacks += s.MemoryFeedbacks
		totalCheckpoints += s.CheckpointSaves + s.CheckpointResumes
	}

	avgCallsPerSession := float64(0)
	if totalSessions > 0 {
		avgCallsPerSession = float64(totalContextdCalls) / float64(totalSessions)
	}

	return map[string]interface{}{
		"total_sessions":        totalSessions,
		"total_contextd_calls":  totalContextdCalls,
		"total_searches":        totalSearches,
		"total_records":         totalRecords,
		"total_feedbacks":       totalFeedbacks,
		"total_checkpoints":     totalCheckpoints,
		"avg_calls_per_session": avgCallsPerSession,
	}
}
