package compression

import (
	"fmt"
	"io"
	"strings"
)

// ANSI color codes for diff highlighting
const (
	colorReset  = "\x1b[0m"
	colorRed    = "\x1b[31m" // Removed lines
	colorGreen  = "\x1b[32m" // Added/compressed lines
	colorYellow = "\x1b[33m" // Modified lines
	colorCyan   = "\x1b[36m" // Headers
	colorBold   = "\x1b[1m"  // Bold text
)

// PreviewOptions configures the preview output
type PreviewOptions struct {
	// Width of the output (terminal columns)
	Width int

	// ShowMetrics displays compression metrics at the top
	ShowMetrics bool

	// ColorOutput enables ANSI color codes for diff highlighting
	ColorOutput bool
}

// Preview generates a side-by-side comparison of original vs compressed content
// with diff highlighting and optional quality metrics display.
//
// The preview shows:
// - Quality metrics (compression ratio, quality score, processing time)
// - Side-by-side comparison with removed lines highlighted
// - Diff indicators showing what was removed or kept
//
// Parameters:
//   - w: Output writer (e.g., os.Stdout, bytes.Buffer)
//   - original: Original uncompressed content
//   - result: Compression result containing compressed content and metadata
//   - opts: Display options (width, colors, metrics)
//
// Returns error if validation fails or writing fails.
func Preview(w io.Writer, original string, result *Result, opts PreviewOptions) error {
	// Validate inputs
	if w == nil {
		return fmt.Errorf("writer cannot be nil")
	}
	if strings.TrimSpace(original) == "" {
		return fmt.Errorf("original content cannot be empty")
	}
	if result == nil {
		return fmt.Errorf("result cannot be nil")
	}

	// Apply defaults
	if opts.Width == 0 {
		opts.Width = 100 // Default terminal width
	}

	// Generate preview output
	builder := &strings.Builder{}

	// Header
	writeHeader(builder, opts)

	// Metrics section (if enabled)
	if opts.ShowMetrics {
		writeMetrics(builder, result, opts)
	}

	// Side-by-side comparison
	writeSideBySide(builder, original, result.Content, opts)

	// Write to output
	_, err := w.Write([]byte(builder.String()))
	return err
}

// writeHeader writes the preview title
func writeHeader(b *strings.Builder, opts PreviewOptions) {
	// Calculate header width (respect custom width)
	headerWidth := opts.Width
	if headerWidth < 20 {
		headerWidth = 20
	}

	if opts.ColorOutput {
		b.WriteString(colorBold + colorCyan)
	}
	b.WriteString(strings.Repeat("═", headerWidth) + "\n")
	title := "COMPRESSION PREVIEW"
	padding := (headerWidth - len(title)) / 2
	if padding < 0 {
		padding = 0
	}
	b.WriteString(strings.Repeat(" ", padding) + title + strings.Repeat(" ", headerWidth-padding-len(title)) + "\n")
	b.WriteString(strings.Repeat("═", headerWidth) + "\n")
	if opts.ColorOutput {
		b.WriteString(colorReset)
	}
	b.WriteString("\n")
}

// writeMetrics writes compression quality metrics
func writeMetrics(b *strings.Builder, result *Result, opts PreviewOptions) {
	if opts.ColorOutput {
		b.WriteString(colorBold)
	}
	b.WriteString("Compression Metrics:\n")
	if opts.ColorOutput {
		b.WriteString(colorReset)
	}

	b.WriteString(fmt.Sprintf("  Algorithm:         %s\n", result.Metadata.Algorithm))
	b.WriteString(fmt.Sprintf("  Compression Ratio: %.2fx\n", result.Metadata.CompressionRatio))
	b.WriteString(fmt.Sprintf("  Quality Score:     %.2f%%\n", result.QualityScore*100))
	b.WriteString(fmt.Sprintf("  Original Size:     %d bytes\n", result.Metadata.OriginalSize))
	b.WriteString(fmt.Sprintf("  Compressed Size:   %d bytes\n", result.Metadata.CompressedSize))
	if result.ProcessingTime > 0 {
		b.WriteString(fmt.Sprintf("  Processing Time:   %v\n", result.ProcessingTime))
	}
	b.WriteString("\n")
}

// writeSideBySide writes the side-by-side comparison with diff highlighting
func writeSideBySide(b *strings.Builder, original, compressed string, opts PreviewOptions) {
	// Calculate column width (split screen in half, accounting for separator)
	colWidth := (opts.Width - 7) / 2 // 7 chars for "│" separator and padding
	if colWidth < 20 {
		colWidth = 20 // Minimum readable width
	}

	// Split content into lines
	originalLines := strings.Split(original, "\n")
	compressedLines := strings.Split(compressed, "\n")

	// Build lookup for compressed lines (for diff detection)
	compressedSet := make(map[string]bool)
	for _, line := range compressedLines {
		compressedSet[strings.TrimSpace(line)] = true
	}

	// Write column headers
	if opts.ColorOutput {
		b.WriteString(colorBold + colorCyan)
	}
	b.WriteString(fmt.Sprintf("%-*s │ %s\n", colWidth, "ORIGINAL", "COMPRESSED"))
	b.WriteString(strings.Repeat("─", colWidth) + "─┼─" + strings.Repeat("─", colWidth) + "\n")
	if opts.ColorOutput {
		b.WriteString(colorReset)
	}

	// Track position in compressed lines
	compIdx := 0

	// Write line-by-line comparison
	for i, origLine := range originalLines {
		trimmedOrig := strings.TrimSpace(origLine)

		// Truncate if too long
		displayOrig := truncate(origLine, colWidth)

		// Check if this line exists in compressed output
		lineKept := compressedSet[trimmedOrig]

		// Format original column
		var origCol string
		if !lineKept {
			// Line was removed
			if opts.ColorOutput {
				origCol = colorRed + displayOrig + colorReset
			} else {
				origCol = displayOrig + " [REMOVED]"
			}
		} else {
			origCol = displayOrig
		}

		// Format compressed column
		var compCol string
		if lineKept && compIdx < len(compressedLines) {
			// Check if the next compressed line matches
			if strings.TrimSpace(compressedLines[compIdx]) == trimmedOrig {
				compCol = truncate(compressedLines[compIdx], colWidth)
				if opts.ColorOutput {
					compCol = colorGreen + compCol + colorReset
				}
				compIdx++
			}
		}

		// Write the row
		b.WriteString(fmt.Sprintf("%-*s │ %s\n",
			colWidth+colorPadding(origCol, opts.ColorOutput),
			origCol,
			compCol))

		// Add visual separator every 5 lines for readability
		if (i+1)%5 == 0 && i < len(originalLines)-1 {
			if opts.ColorOutput {
				b.WriteString(colorCyan)
			}
			b.WriteString(strings.Repeat("·", colWidth) + "·┼·" + strings.Repeat("·", colWidth) + "\n")
			if opts.ColorOutput {
				b.WriteString(colorReset)
			}
		}
	}

	// Write summary
	b.WriteString("\n")
	if opts.ColorOutput {
		b.WriteString(colorBold)
	}
	b.WriteString(fmt.Sprintf("Summary: %d lines → %d lines (%.1f%% reduction)\n",
		len(originalLines),
		len(compressedLines),
		(1.0-float64(len(compressedLines))/float64(len(originalLines)))*100))
	if opts.ColorOutput {
		b.WriteString(colorReset)
	}
}

// truncate truncates a string to maxLen, adding "..." if truncated
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen < 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// colorPadding calculates extra padding needed for ANSI color codes
func colorPadding(s string, colorEnabled bool) int {
	if !colorEnabled {
		return 0
	}
	// Count ANSI escape sequences (they don't contribute to visible width)
	padding := 0
	i := 0
	for i < len(s) {
		if i < len(s)-1 && s[i] == '\x1b' && s[i+1] == '[' {
			// Found ANSI escape sequence
			j := i + 2
			for j < len(s) && s[j] != 'm' {
				j++
			}
			if j < len(s) {
				padding += (j - i + 1) // +1 for the 'm'
				i = j + 1
				continue
			}
		}
		i++
	}
	return padding
}
