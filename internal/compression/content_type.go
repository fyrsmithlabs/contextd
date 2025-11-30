package compression

import (
	"regexp"
	"strings"
)

// ContentType represents the type of content being compressed
type ContentType string

const (
	// ContentTypeCode represents code content (Go, Python, JS, etc.)
	ContentTypeCode ContentType = "code"
	// ContentTypeMarkdown represents markdown documentation
	ContentTypeMarkdown ContentType = "markdown"
	// ContentTypeConversation represents dialog/conversation
	ContentTypeConversation ContentType = "conversation"
	// ContentTypeMixed represents mixed content types
	ContentTypeMixed ContentType = "mixed"
	// ContentTypePlain represents plain text
	ContentTypePlain ContentType = "plain"
)

var (
	// Code markers
	codeBackticksRegex = regexp.MustCompile("```[a-z]*\n")
	funcKeywordRegex   = regexp.MustCompile(`\b(func|function|def|class|interface|struct|impl)\b`)
	braceSemiRegex     = regexp.MustCompile(`[{};]`)
	indentedCodeRegex  = regexp.MustCompile(`(?m)^    \w+`)

	// Markdown markers
	headerRegex     = regexp.MustCompile(`(?m)^#{1,6}\s+.+$`)
	listRegex       = regexp.MustCompile(`(?m)^[\s]*[-*+]\s+`)
	numberedRegex   = regexp.MustCompile(`(?m)^[\s]*\d+\.\s+`)
	linkRegex       = regexp.MustCompile(`\[.+\]\(.+\)`)
	boldItalicRegex = regexp.MustCompile(`(\*\*|__|_|\*)`)

	// Conversation markers
	conversationRegex = regexp.MustCompile(`(?m)^(Human|Assistant|User|Bot|AI):\s+`)
)

// detectContentType identifies the primary content type using hybrid detection
func (c *ExtractiveCompressor) detectContentType(content string) ContentType {
	// Check for mixed content first (markdown + code blocks)
	hasMarkdown := c.hasClearMarkdownMarkers(content)
	hasCode := strings.Contains(content, "```") || c.hasClearCodeMarkers(content)

	if hasMarkdown && hasCode {
		return ContentTypeMixed
	}

	// Fast path: check for clear markers
	if hasCode {
		return ContentTypeCode
	}

	if hasMarkdown {
		return ContentTypeMarkdown
	}

	if c.hasClearConversationMarkers(content) {
		return ContentTypeConversation
	}

	// Fallback to statistical analysis
	return c.detectByStatistics(content)
}

// hasClearCodeMarkers checks for obvious code indicators
func (c *ExtractiveCompressor) hasClearCodeMarkers(content string) bool {
	// Triple backticks
	if codeBackticksRegex.MatchString(content) {
		return true
	}

	// Function/class keywords
	if funcKeywordRegex.MatchString(content) {
		return true
	}

	// Indented code blocks (4+ spaces)
	if indentedCodeRegex.MatchString(content) {
		lines := strings.Split(content, "\n")
		indentedCount := 0
		for _, line := range lines {
			if len(line) >= 4 && line[:4] == "    " {
				indentedCount++
			}
		}
		// At least 3 indented lines suggests code
		if indentedCount >= 3 {
			return true
		}
	}

	// High density of braces/semicolons
	braceCount := len(braceSemiRegex.FindAllString(content, -1))
	if float64(braceCount)/float64(len(content)) > 0.03 { // 3% threshold
		return true
	}

	return false
}

// hasClearMarkdownMarkers checks for obvious markdown indicators
func (c *ExtractiveCompressor) hasClearMarkdownMarkers(content string) bool {
	// Headers
	if headerRegex.MatchString(content) {
		return true
	}

	// Lists
	if listRegex.MatchString(content) || numberedRegex.MatchString(content) {
		return true
	}

	// Links
	if linkRegex.MatchString(content) {
		return true
	}

	// Bold/italic (at least 2 occurrences)
	matches := boldItalicRegex.FindAllString(content, -1)
	return len(matches) >= 2
}

// hasClearConversationMarkers checks for dialog patterns
func (c *ExtractiveCompressor) hasClearConversationMarkers(content string) bool {
	matches := conversationRegex.FindAllString(content, -1)
	return len(matches) >= 2 // At least 2 turns
}

// detectByStatistics uses statistical analysis for ambiguous content
func (c *ExtractiveCompressor) detectByStatistics(content string) ContentType {
	scores := map[ContentType]float64{
		ContentTypeCode:         c.scoreAsCode(content),
		ContentTypeMarkdown:     c.scoreAsMarkdown(content),
		ContentTypeConversation: c.scoreAsConversation(content),
		ContentTypePlain:        c.scoreAsPlain(content),
	}

	// Find highest score
	maxScore := 0.0
	result := ContentTypePlain
	for ct, score := range scores {
		if score > maxScore {
			maxScore = score
			result = ct
		}
	}

	return result
}

// scoreAsCode calculates code likelihood score
func (c *ExtractiveCompressor) scoreAsCode(content string) float64 {
	score := 0.0

	// Brace density
	braceCount := len(braceSemiRegex.FindAllString(content, -1))
	score += float64(braceCount) / float64(len(content)) * 100

	// Keyword density
	keywordCount := len(funcKeywordRegex.FindAllString(content, -1))
	score += float64(keywordCount) * 2

	// Indentation patterns
	lines := strings.Split(content, "\n")
	indented := 0
	for _, line := range lines {
		if len(line) > 0 && (line[0] == ' ' || line[0] == '\t') {
			indented++
		}
	}
	score += float64(indented) / float64(len(lines)) * 10

	return score
}

// scoreAsMarkdown calculates markdown likelihood score
func (c *ExtractiveCompressor) scoreAsMarkdown(content string) float64 {
	score := 0.0

	// Headers
	score += float64(len(headerRegex.FindAllString(content, -1))) * 3

	// Lists
	score += float64(len(listRegex.FindAllString(content, -1))) * 2
	score += float64(len(numberedRegex.FindAllString(content, -1))) * 2

	// Links
	score += float64(len(linkRegex.FindAllString(content, -1))) * 2

	// Bold/italic
	score += float64(len(boldItalicRegex.FindAllString(content, -1))) * 0.5

	return score
}

// scoreAsConversation calculates conversation likelihood score
func (c *ExtractiveCompressor) scoreAsConversation(content string) float64 {
	turnCount := len(conversationRegex.FindAllString(content, -1))
	return float64(turnCount) * 5
}

// scoreAsPlain calculates plain text likelihood score (baseline)
func (c *ExtractiveCompressor) scoreAsPlain(content string) float64 {
	// Plain text is the default, so give it a base score
	// Will be overridden by higher scores from other types
	return 1.0
}

// splitPreservingStructure splits content while preserving logical blocks
func (c *ExtractiveCompressor) splitPreservingStructure(content string, contentType ContentType) []string {
	switch contentType {
	case ContentTypeCode:
		return c.splitCodeBlocks(content)
	case ContentTypeMarkdown:
		return c.splitMarkdownSections(content)
	case ContentTypeConversation:
		return c.splitConversationTurns(content)
	default:
		// Fall back to sentence splitting
		return c.splitIntoSentences(content)
	}
}

// splitCodeBlocks splits code into logical blocks (functions, classes)
func (c *ExtractiveCompressor) splitCodeBlocks(content string) []string {
	var blocks []string
	var current strings.Builder
	braceDepth := 0

	lines := strings.Split(content, "\n")
	for i, line := range lines {
		// Track brace depth for this line
		lineOpenBraces := strings.Count(line, "{")
		lineCloseBraces := strings.Count(line, "}")

		current.WriteString(line)
		current.WriteString("\n")

		braceDepth += lineOpenBraces - lineCloseBraces

		// When we return to depth 0 after being in a block, save it
		if braceDepth == 0 && lineCloseBraces > 0 && current.Len() > 0 {
			blocks = append(blocks, strings.TrimSpace(current.String()))
			current.Reset()
			continue
		}

		// Also split on blank lines when at depth 0
		if braceDepth == 0 && strings.TrimSpace(line) == "" && current.Len() > 1 {
			block := strings.TrimSpace(current.String())
			if block != "" {
				blocks = append(blocks, block)
			}
			current.Reset()
		}

		// If we're at the last line and have content, save it
		if i == len(lines)-1 && current.Len() > 0 {
			block := strings.TrimSpace(current.String())
			if block != "" {
				blocks = append(blocks, block)
			}
		}
	}

	return blocks
}

// splitMarkdownSections splits on headers
func (c *ExtractiveCompressor) splitMarkdownSections(content string) []string {
	var blocks []string
	var current strings.Builder

	lines := strings.Split(content, "\n")
	for _, line := range lines {
		// Check if it's a header
		if headerRegex.MatchString(line) {
			// Save previous section
			if current.Len() > 0 {
				blocks = append(blocks, strings.TrimSpace(current.String()))
				current.Reset()
			}
		}

		current.WriteString(line)
		current.WriteString("\n")
	}

	// Add final section
	if current.Len() > 0 {
		blocks = append(blocks, strings.TrimSpace(current.String()))
	}

	return blocks
}

// splitConversationTurns splits on conversation turn markers
func (c *ExtractiveCompressor) splitConversationTurns(content string) []string {
	var blocks []string
	var current strings.Builder

	lines := strings.Split(content, "\n")
	for _, line := range lines {
		// Check if it's a turn marker
		if conversationRegex.MatchString(line) {
			// Save previous turn
			if current.Len() > 0 {
				blocks = append(blocks, strings.TrimSpace(current.String()))
				current.Reset()
			}
		}

		current.WriteString(line)
		current.WriteString("\n")
	}

	// Add final turn
	if current.Len() > 0 {
		blocks = append(blocks, strings.TrimSpace(current.String()))
	}

	return blocks
}
