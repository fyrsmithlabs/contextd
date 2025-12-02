// Package ignore provides gitignore-style file parsing for repository indexing.
package ignore

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// Parser reads and parses gitignore-style files.
type Parser struct {
	// IgnoreFiles is the list of ignore file names to look for.
	IgnoreFiles []string

	// FallbackPatterns are returned when no ignore files are found.
	FallbackPatterns []string
}

// NewParser creates a new ignore file parser with the given configuration.
func NewParser(ignoreFiles, fallbackPatterns []string) *Parser {
	return &Parser{
		IgnoreFiles:      ignoreFiles,
		FallbackPatterns: fallbackPatterns,
	}
}

// ParseProject reads all ignore files from the project root and returns
// combined exclude patterns. If no ignore files are found, returns fallback patterns.
func (p *Parser) ParseProject(projectRoot string) ([]string, error) {
	var patterns []string
	foundAny := false

	for _, ignoreFile := range p.IgnoreFiles {
		path := filepath.Join(projectRoot, ignoreFile)
		filePatterns, err := p.parseFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		patterns = append(patterns, filePatterns...)
		foundAny = true
	}

	if !foundAny {
		return p.FallbackPatterns, nil
	}

	// Deduplicate patterns
	return deduplicate(patterns), nil
}

// parseFile reads a single gitignore-style file and returns patterns.
func (p *Parser) parseFile(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var patterns []string
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		pattern := parseLine(line)
		if pattern != "" {
			patterns = append(patterns, pattern)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return patterns, nil
}

// parseLine parses a single line from a gitignore file.
// Returns empty string for comments and blank lines.
func parseLine(line string) string {
	// Trim trailing whitespace (unless escaped, but we'll keep it simple)
	line = strings.TrimRight(line, " \t")

	// Skip empty lines
	if line == "" {
		return ""
	}

	// Skip comments
	if strings.HasPrefix(line, "#") {
		return ""
	}

	// Skip negation patterns (we don't support them for simplicity)
	if strings.HasPrefix(line, "!") {
		return ""
	}

	// Convert to glob pattern suitable for doublestar matching
	pattern := toGlobPattern(line)

	return pattern
}

// toGlobPattern converts a gitignore pattern to a glob pattern.
func toGlobPattern(pattern string) string {
	// Remove leading slash (absolute path in gitignore means relative to root)
	pattern = strings.TrimPrefix(pattern, "/")

	// If pattern ends with /, it's a directory - add **
	if strings.HasSuffix(pattern, "/") {
		pattern = pattern + "**"
	}

	// If pattern doesn't contain /, it can match anywhere - prefix with **/
	if !strings.Contains(pattern, "/") && !strings.HasPrefix(pattern, "**/") {
		// But only if it's not already a glob pattern that starts with *
		if !strings.HasPrefix(pattern, "*") {
			pattern = "**/" + pattern
		}
	}

	// Ensure directory patterns have /** suffix for recursive matching
	// e.g., "node_modules" should become "**/node_modules/**"
	if !strings.HasSuffix(pattern, "/**") && !strings.HasSuffix(pattern, "/*") && !strings.Contains(pattern, ".") {
		// Looks like a directory name, add /** for recursive match
		pattern = pattern + "/**"
	}

	return pattern
}

// deduplicate removes duplicate patterns while preserving order.
func deduplicate(patterns []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(patterns))

	for _, p := range patterns {
		if !seen[p] {
			seen[p] = true
			result = append(result, p)
		}
	}

	return result
}
