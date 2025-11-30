package compression

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/fyrsmithlabs/contextd/internal/vectorstore"
)

// RED Phase: Write failing tests first

func TestPreview_ValidResult_GeneratesOutput(t *testing.T) {
	// Setup test data
	original := `Line 1
Line 2
Line 3
Line 4
Line 5`

	compressed := `Line 1
Line 3
Line 5`

	result := &Result{
		Content:        compressed,
		ProcessingTime: 100 * time.Millisecond,
		QualityScore:   0.85,
		Metadata: vectorstore.CompressionMetadata{
			Level:            vectorstore.CompressionLevelSummary,
			Algorithm:        "extractive",
			OriginalSize:     len(original),
			CompressedSize:   len(compressed),
			CompressionRatio: float64(len(original)) / float64(len(compressed)),
		},
	}

	// Execute
	var buf bytes.Buffer
	opts := PreviewOptions{
		Width:       80,
		ShowMetrics: true,
		ColorOutput: false, // Disable for easier testing
	}

	err := Preview(&buf, original, result, opts)

	// Verify
	if err != nil {
		t.Fatalf("Preview() unexpected error: %v", err)
	}

	output := buf.String()

	// Check that output contains expected sections
	if !strings.Contains(output, "COMPRESSION PREVIEW") {
		t.Error("Output missing title")
	}
	if !strings.Contains(output, "Compression Ratio:") {
		t.Error("Output missing compression ratio")
	}
	if !strings.Contains(output, "Quality Score:") {
		t.Error("Output missing quality score")
	}
	if !strings.Contains(output, "Original") && !strings.Contains(output, "ORIGINAL") {
		t.Error("Output missing original section header")
	}
	if !strings.Contains(output, "Compressed") && !strings.Contains(output, "COMPRESSED") {
		t.Error("Output missing compressed section header")
	}
}

func TestPreview_EmptyOriginal_ReturnsError(t *testing.T) {
	result := &Result{
		Content: "some compressed content",
	}

	var buf bytes.Buffer
	opts := PreviewOptions{Width: 80}

	err := Preview(&buf, "", result, opts)

	if err == nil {
		t.Error("Expected error for empty original content, got nil")
	}
	if err != nil && !strings.Contains(err.Error(), "original content") {
		t.Errorf("Expected error about original content, got: %v", err)
	}
}

func TestPreview_NilResult_ReturnsError(t *testing.T) {
	var buf bytes.Buffer
	opts := PreviewOptions{Width: 80}

	err := Preview(&buf, "original content", nil, opts)

	if err == nil {
		t.Error("Expected error for nil result, got nil")
	}
	if err != nil && !strings.Contains(err.Error(), "result") {
		t.Errorf("Expected error about nil result, got: %v", err)
	}
}

func TestPreview_NilWriter_ReturnsError(t *testing.T) {
	result := &Result{
		Content: "compressed",
	}
	opts := PreviewOptions{Width: 80}

	err := Preview(nil, "original", result, opts)

	if err == nil {
		t.Error("Expected error for nil writer, got nil")
	}
	if err != nil && !strings.Contains(err.Error(), "writer") {
		t.Errorf("Expected error about nil writer, got: %v", err)
	}
}

func TestPreview_DifferentLengthContent_ShowsDiff(t *testing.T) {
	original := `First line
Second line
Third line`

	compressed := `First line
Third line`

	result := &Result{
		Content: compressed,
		Metadata: vectorstore.CompressionMetadata{
			OriginalSize:     len(original),
			CompressedSize:   len(compressed),
			CompressionRatio: float64(len(original)) / float64(len(compressed)),
		},
	}

	var buf bytes.Buffer
	opts := PreviewOptions{
		Width:       80,
		ColorOutput: false,
	}

	err := Preview(&buf, original, result, opts)

	if err != nil {
		t.Fatalf("Preview() unexpected error: %v", err)
	}

	output := buf.String()

	// Should show both originals and indicate removed lines
	if !strings.Contains(output, "First line") {
		t.Error("Output missing 'First line'")
	}
	if !strings.Contains(output, "Third line") {
		t.Error("Output missing 'Third line'")
	}
}

func TestPreview_MetricsDisabled_NoMetrics(t *testing.T) {
	result := &Result{
		Content: "compressed",
		Metadata: vectorstore.CompressionMetadata{
			OriginalSize:     100,
			CompressedSize:   50,
			CompressionRatio: 2.0,
		},
		QualityScore: 0.9,
	}

	var buf bytes.Buffer
	opts := PreviewOptions{
		Width:       80,
		ShowMetrics: false, // Explicitly disable
	}

	err := Preview(&buf, "original content", result, opts)

	if err != nil {
		t.Fatalf("Preview() unexpected error: %v", err)
	}

	output := buf.String()

	// Should NOT contain metrics when disabled
	if strings.Contains(output, "Compression Ratio:") {
		t.Error("Output should not contain metrics when ShowMetrics=false")
	}
}

func TestPreview_CustomWidth_RespectsWidth(t *testing.T) {
	original := "This is a very long line that should be wrapped or truncated based on the specified width"
	compressed := "This is a long line"

	result := &Result{
		Content: compressed,
		Metadata: vectorstore.CompressionMetadata{
			OriginalSize:     len(original),
			CompressedSize:   len(compressed),
			CompressionRatio: float64(len(original)) / float64(len(compressed)),
		},
	}

	var buf bytes.Buffer
	opts := PreviewOptions{
		Width:       40, // Narrow width
		ShowMetrics: false,
		ColorOutput: false,
	}

	err := Preview(&buf, original, result, opts)

	if err != nil {
		t.Fatalf("Preview() unexpected error: %v", err)
	}

	output := buf.String()
	lines := strings.Split(output, "\n")

	// Check data lines (skip headers/separators which may be exactly opts.Width)
	dataLineCount := 0
	for _, line := range lines {
		// Strip ANSI codes for length calculation
		cleanLine := stripANSI(line)

		// Skip empty lines and separator lines (made of repeated chars)
		if cleanLine == "" || isRepeatedChar(cleanLine) {
			continue
		}

		// Data lines should respect width (with some tolerance for separator " │ ")
		if len(cleanLine) > opts.Width+10 { // Allow reasonable buffer for formatting
			t.Errorf("Data line exceeds reasonable width: %d > %d: %q", len(cleanLine), opts.Width+10, cleanLine)
		}
		dataLineCount++
	}

	if dataLineCount == 0 {
		t.Error("Expected some data lines in output")
	}
}

// Helper to check if a string is made of repeated characters (separators/borders)
func isRepeatedChar(s string) bool {
	if len(s) == 0 {
		return false
	}
	// Check if string is mostly one character (separators like "═", "─", "┼")
	charCounts := make(map[rune]int)
	for _, ch := range s {
		if ch != ' ' {
			charCounts[ch]++
		}
	}
	// If there are very few unique non-space characters, it's a separator
	return len(charCounts) <= 2
}

func TestCompressionExample_Integration(t *testing.T) {
	// Integration test: Compress and preview
	ctx := context.Background()

	config := Config{
		DefaultAlgorithm: AlgorithmExtractive,
		TargetRatio:      2.0,
	}

	service, err := NewService(config)
	if err != nil {
		t.Fatalf("NewService() error: %v", err)
	}

	original := `This is a sample document with multiple lines.
It contains various information that we want to compress.
Some lines are important, while others are less critical.
The compression algorithm will decide what to keep.
This demonstrates the preview functionality.`

	result, err := service.Compress(ctx, original, AlgorithmExtractive, 2.0)
	if err != nil {
		t.Fatalf("Compress() error: %v", err)
	}

	var buf bytes.Buffer
	opts := PreviewOptions{
		Width:       100,
		ShowMetrics: true,
		ColorOutput: false,
	}

	err = Preview(&buf, original, result, opts)
	if err != nil {
		t.Fatalf("Preview() error: %v", err)
	}

	output := buf.String()

	// Verify output contains key elements
	if len(output) == 0 {
		t.Error("Preview generated empty output")
	}

	if !strings.Contains(output, "Compression Ratio:") {
		t.Error("Missing compression ratio in preview")
	}
}

func TestPreview_ZeroWidth_UsesDefault(t *testing.T) {
	result := &Result{
		Content: "compressed",
		Metadata: vectorstore.CompressionMetadata{
			OriginalSize:   100,
			CompressedSize: 50,
		},
	}

	var buf bytes.Buffer
	opts := PreviewOptions{
		Width: 0, // Should use default
	}

	err := Preview(&buf, "original", result, opts)

	if err != nil {
		t.Fatalf("Preview() should handle zero width: %v", err)
	}

	// Should not panic and should generate output
	if buf.Len() == 0 {
		t.Error("Expected output even with zero width")
	}
}

func TestPreview_ColorOutput_ContainsANSICodes(t *testing.T) {
	original := `Line 1
Line 2 removed
Line 3`

	compressed := `Line 1
Line 3`

	result := &Result{
		Content:        compressed,
		ProcessingTime: 50 * time.Millisecond,
		QualityScore:   0.92,
		Metadata: vectorstore.CompressionMetadata{
			Level:            vectorstore.CompressionLevelSummary,
			Algorithm:        "hybrid",
			OriginalSize:     len(original),
			CompressedSize:   len(compressed),
			CompressionRatio: float64(len(original)) / float64(len(compressed)),
		},
	}

	var buf bytes.Buffer
	opts := PreviewOptions{
		Width:       80,
		ShowMetrics: true,
		ColorOutput: true, // Enable color output
	}

	err := Preview(&buf, original, result, opts)

	if err != nil {
		t.Fatalf("Preview() unexpected error: %v", err)
	}

	output := buf.String()

	// Check for ANSI escape codes
	if !strings.Contains(output, "\x1b[") {
		t.Error("Expected ANSI color codes in output when ColorOutput=true")
	}

	// Should contain color codes for header (cyan, bold)
	if !strings.Contains(output, "\x1b[36m") { // Cyan
		t.Error("Expected cyan color code for header")
	}

	// Should contain color codes for removed lines (red)
	if !strings.Contains(output, "\x1b[31m") { // Red
		t.Error("Expected red color code for removed lines")
	}

	// Should contain reset codes
	if !strings.Contains(output, "\x1b[0m") {
		t.Error("Expected reset color codes")
	}
}

// Helper function to strip ANSI color codes for testing
func stripANSI(s string) string {
	// Simple ANSI code stripper for testing
	result := ""
	inEscape := false
	for _, r := range s {
		if r == '\x1b' {
			inEscape = true
			continue
		}
		if inEscape {
			if r == 'm' {
				inEscape = false
			}
			continue
		}
		result += string(r)
	}
	return result
}
