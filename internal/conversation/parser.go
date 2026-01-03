package conversation

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Parser implements ConversationParser for Claude Code JSONL files.
type Parser struct{}

// NewParser creates a new conversation parser.
func NewParser() *Parser {
	return &Parser{}
}

// jsonlMessage represents the raw structure of a Claude Code JSONL message.
type jsonlMessage struct {
	UUID       string          `json:"uuid"`
	ParentUUID string          `json:"parentUuid,omitempty"`
	Type       string          `json:"type"`
	Message    json.RawMessage `json:"message,omitempty"`
	Timestamp  string          `json:"timestamp,omitempty"`
	SessionID  string          `json:"sessionId,omitempty"`
	GitBranch  string          `json:"cwd,omitempty"` // Often contains git context
}

// claudeMessage represents the nested message structure.
type claudeMessage struct {
	Role    string         `json:"role"`
	Content []contentBlock `json:"content"`
}

// contentBlock represents a content block in a Claude message.
type contentBlock struct {
	Type      string          `json:"type"`
	Text      string          `json:"text,omitempty"`
	ToolUse   *toolUseBlock   `json:"tool_use,omitempty"`
	Name      string          `json:"name,omitempty"`
	Input     json.RawMessage `json:"input,omitempty"`
	ToolUseID string          `json:"tool_use_id,omitempty"`
	Content   string          `json:"content,omitempty"`
}

// toolUseBlock represents a tool use within content.
type toolUseBlock struct {
	ID    string          `json:"id"`
	Name  string          `json:"name"`
	Input json.RawMessage `json:"input"`
}

// Parse reads a JSONL file and extracts messages.
func (p *Parser) Parse(path string) ([]RawMessage, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening file: %w", err)
	}
	defer file.Close()

	var messages []RawMessage
	scanner := bufio.NewScanner(file)

	// Increase buffer size for large messages
	const maxScanTokenSize = 10 * 1024 * 1024 // 10MB
	buf := make([]byte, maxScanTokenSize)
	scanner.Buffer(buf, maxScanTokenSize)

	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}

		var jm jsonlMessage
		if err := json.Unmarshal([]byte(line), &jm); err != nil {
			// Skip malformed lines but continue parsing
			continue
		}

		// Only process user and assistant messages
		if jm.Type != "user" && jm.Type != "assistant" {
			continue
		}

		msg, err := p.parseMessage(jm, path)
		if err != nil {
			// Skip messages that can't be parsed
			continue
		}

		if msg != nil {
			messages = append(messages, *msg)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanning file: %w", err)
	}

	return messages, nil
}

// parseMessage converts a jsonlMessage to a RawMessage.
func (p *Parser) parseMessage(jm jsonlMessage, path string) (*RawMessage, error) {
	// Extract session ID from file path if not in message
	sessionID := jm.SessionID
	if sessionID == "" {
		// Try to extract from filename (often session UUID)
		sessionID = strings.TrimSuffix(filepath.Base(path), ".jsonl")
	}

	// Parse timestamp
	var timestamp time.Time
	if jm.Timestamp != "" {
		var err error
		timestamp, err = time.Parse(time.RFC3339, jm.Timestamp)
		if err != nil {
			// Try alternative formats
			timestamp, err = time.Parse("2006-01-02T15:04:05Z", jm.Timestamp)
			if err != nil {
				timestamp = time.Now() // Fallback
			}
		}
	}

	// Parse the nested message content
	var content string
	var toolCalls []ToolCall
	var role Role

	if jm.Type == "user" {
		role = RoleUser
		// User messages might be simple strings or structured
		var userContent string
		if err := json.Unmarshal(jm.Message, &userContent); err == nil {
			content = userContent
		} else {
			// Try parsing as content blocks
			var cm claudeMessage
			if err := json.Unmarshal(jm.Message, &cm); err == nil {
				content, toolCalls = p.extractContent(cm.Content)
			}
		}
	} else if jm.Type == "assistant" {
		role = RoleAssistant
		var cm claudeMessage
		if err := json.Unmarshal(jm.Message, &cm); err == nil {
			content, toolCalls = p.extractContent(cm.Content)
		}
	}

	// Skip empty messages
	if content == "" && len(toolCalls) == 0 {
		return nil, nil
	}

	return &RawMessage{
		SessionID:  sessionID,
		UUID:       jm.UUID,
		Timestamp:  timestamp,
		Role:       role,
		Content:    content,
		ToolCalls:  toolCalls,
		GitBranch:  jm.GitBranch,
		ParentUUID: jm.ParentUUID,
	}, nil
}

// extractContent extracts text content and tool calls from content blocks.
func (p *Parser) extractContent(blocks []contentBlock) (string, []ToolCall) {
	var textParts []string
	var toolCalls []ToolCall

	for _, block := range blocks {
		switch block.Type {
		case "text":
			if block.Text != "" {
				textParts = append(textParts, block.Text)
			}
		case "tool_use":
			if block.ToolUse != nil {
				tc := ToolCall{
					Name:   block.ToolUse.Name,
					Params: make(map[string]string),
				}
				// Parse input as map of strings
				var inputMap map[string]interface{}
				if err := json.Unmarshal(block.ToolUse.Input, &inputMap); err == nil {
					for k, v := range inputMap {
						tc.Params[k] = fmt.Sprintf("%v", v)
					}
				}
				toolCalls = append(toolCalls, tc)
			} else if block.Name != "" {
				// Alternative format
				tc := ToolCall{
					Name:   block.Name,
					Params: make(map[string]string),
				}
				var inputMap map[string]interface{}
				if err := json.Unmarshal(block.Input, &inputMap); err == nil {
					for k, v := range inputMap {
						tc.Params[k] = fmt.Sprintf("%v", v)
					}
				}
				toolCalls = append(toolCalls, tc)
			}
		case "tool_result":
			// Store tool results for cross-referencing
			if block.Content != "" && len(toolCalls) > 0 {
				// Associate result with previous tool call
				toolCalls[len(toolCalls)-1].Result = block.Content
			}
		}
	}

	return strings.Join(textParts, "\n"), toolCalls
}

// ParseAll reads all JSONL files in a directory and returns messages grouped by session.
func (p *Parser) ParseAll(dir string) (map[string][]RawMessage, error) {
	result := make(map[string][]RawMessage)

	// Find all JSONL files
	pattern := filepath.Join(dir, "*.jsonl")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("globbing files: %w", err)
	}

	// Also check subdirectories (Claude Code structure varies)
	subdirPattern := filepath.Join(dir, "*", "*.jsonl")
	subdirFiles, _ := filepath.Glob(subdirPattern)
	files = append(files, subdirFiles...)

	for _, file := range files {
		messages, err := p.Parse(file)
		if err != nil {
			// Log but continue with other files
			continue
		}

		if len(messages) > 0 {
			sessionID := messages[0].SessionID
			result[sessionID] = append(result[sessionID], messages...)
		}
	}

	return result, nil
}

// extractSessionID extracts the session ID from a file path.
func extractSessionID(path string) string {
	return strings.TrimSuffix(filepath.Base(path), ".jsonl")
}

// Ensure Parser implements ConversationParser.
var _ ConversationParser = (*Parser)(nil)
