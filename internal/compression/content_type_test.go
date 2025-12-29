package compression

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDetectContentType(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected ContentType
	}{
		{
			name:     "code with triple backticks",
			content:  "Here is some code:\n```go\nfunc main() {\n    fmt.Println(\"hello\")\n}\n```",
			expected: ContentTypeCode,
		},
		{
			name:     "markdown with headers",
			content:  "# Main Header\n\n## Subheader\n\nSome content here.\n\n### Another section",
			expected: ContentTypeMarkdown,
		},
		{
			name:     "conversation with turns",
			content:  "Human: How do I implement this?\n\nAssistant: You can implement it by...\n\nHuman: Thanks!",
			expected: ContentTypeConversation,
		},
		{
			name:     "indented code block",
			content:  "    def hello():\n        print('world')\n        return True\n\n    class Foo:\n        pass",
			expected: ContentTypeCode,
		},
		{
			name:     "plain text",
			content:  "This is just plain text. No special formatting. Just sentences and paragraphs.",
			expected: ContentTypePlain,
		},
		{
			name:     "mixed content - code dominant",
			content:  "# Setup\n\nFirst install:\n```bash\nnpm install\n```\n\nThen run:\n```js\nconst x = 42;\n```",
			expected: ContentTypeMixed,
		},
		{
			name:     "markdown list",
			content:  "- Item one\n- Item two\n  - Nested item\n- Item three\n\n1. First\n2. Second",
			expected: ContentTypeMarkdown,
		},
		{
			name:     "code with function signature",
			content:  "func ProcessData(ctx context.Context, data []byte) error {\n    if len(data) == 0 {\n        return errors.New(\"empty\")\n    }\n    return nil\n}",
			expected: ContentTypeCode,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compressor := NewExtractiveCompressor(Config{})
			got := compressor.detectContentType(tt.content)
			assert.Equal(t, tt.expected, got, "content type mismatch for: %s", tt.name)
		})
	}
}

func TestHasClearCodeMarkers(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected bool
	}{
		{"triple backticks", "```go\ncode\n```", true},
		{"function keyword", "func main() {}", true},
		{"class keyword", "class Foo:", true},
		{"indented code", "    def hello():\n        pass", true},
		{"braces and semicolons", "if (x > 0) {\n    return true;\n}", true},
		{"plain text", "This is just text", false},
		{"markdown", "# Header\n\nContent", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compressor := NewExtractiveCompressor(Config{})
			got := compressor.hasClearCodeMarkers(tt.content)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestDetectByStatistics(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected ContentType
	}{
		{
			name:     "high brace density",
			content:  "function test() { if (x) { return { a: 1, b: 2 }; } }",
			expected: ContentTypeCode,
		},
		{
			name:     "markdown patterns",
			content:  "Look at [this link](http://example.com) and **bold text**",
			expected: ContentTypeMarkdown,
		},
		{
			name:     "conversation patterns",
			content:  "User: What is this?\nBot: This is a response.\nUser: Thanks",
			expected: ContentTypeConversation,
		},
		{
			name:     "plain text",
			content:  "Just normal sentences. Nothing special here. Regular words.",
			expected: ContentTypePlain,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compressor := NewExtractiveCompressor(Config{})
			got := compressor.detectByStatistics(tt.content)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestSplitPreservingStructure(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		contentType ContentType
		wantBlocks  int
		description string
	}{
		{
			name:        "code with functions",
			content:     "func foo() {\n    return 1\n}\n\nfunc bar() {\n    return 2\n}",
			contentType: ContentTypeCode,
			wantBlocks:  2,
			description: "should split into 2 function blocks",
		},
		{
			name:        "markdown sections",
			content:     "# Header 1\n\nContent 1\n\n## Header 2\n\nContent 2",
			contentType: ContentTypeMarkdown,
			wantBlocks:  2,
			description: "should split into 2 sections",
		},
		{
			name:        "conversation turns",
			content:     "Human: Question?\n\nAssistant: Answer.\n\nHuman: Thanks!",
			contentType: ContentTypeConversation,
			wantBlocks:  3,
			description: "should preserve 3 turns",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compressor := NewExtractiveCompressor(Config{})
			blocks := compressor.splitPreservingStructure(tt.content, tt.contentType)
			assert.Equal(t, tt.wantBlocks, len(blocks), tt.description)
		})
	}
}
