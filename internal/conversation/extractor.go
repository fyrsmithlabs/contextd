package conversation

import (
	"regexp"
	"strings"
)

// Extractor extracts file references, commit SHAs, and other metadata from messages.
type Extractor struct {
	// Patterns for extracting references
	commitSHAPattern *regexp.Regexp
	filePathPattern  *regexp.Regexp
}

// NewExtractor creates a new metadata extractor.
func NewExtractor() *Extractor {
	return &Extractor{
		// Match git commit SHAs (7-40 hex characters)
		commitSHAPattern: regexp.MustCompile(`\b([a-f0-9]{7,40})\b`),
		// Match file paths (simplified - looks for paths with extensions)
		filePathPattern: regexp.MustCompile(`(?:^|[\s"'\(])([a-zA-Z0-9_\-./]+\.[a-zA-Z0-9]+)(?:$|[\s"'\):,])`),
	}
}

// ExtractFileReferences extracts file references from tool calls in a message.
func (e *Extractor) ExtractFileReferences(msg RawMessage) []FileReference {
	refs := make(map[string]FileReference) // Dedupe by path

	for _, tc := range msg.ToolCalls {
		ref := e.extractFromToolCall(tc)
		if ref != nil {
			// Update with more specific action if we have one
			if existing, ok := refs[ref.Path]; ok {
				if ref.Action != existing.Action && ref.Action != ActionRead {
					refs[ref.Path] = *ref
				}
			} else {
				refs[ref.Path] = *ref
			}
		}
	}

	// Also extract file paths from content (for references without tool calls)
	contentRefs := e.extractFilePathsFromText(msg.Content)
	for _, path := range contentRefs {
		if _, ok := refs[path]; !ok {
			refs[path] = FileReference{
				Path:   path,
				Action: ActionRead, // Default to read when just mentioned
			}
		}
	}

	result := make([]FileReference, 0, len(refs))
	for _, ref := range refs {
		result = append(result, ref)
	}
	return result
}

// extractFromToolCall extracts a file reference from a specific tool call.
func (e *Extractor) extractFromToolCall(tc ToolCall) *FileReference {
	switch tc.Name {
	case "Read":
		if path := tc.Params["file_path"]; path != "" {
			return &FileReference{
				Path:       path,
				Action:     ActionRead,
				LineRanges: extractLineRanges(tc.Params),
			}
		}
	case "Edit":
		if path := tc.Params["file_path"]; path != "" {
			return &FileReference{
				Path:   path,
				Action: ActionEdited,
			}
		}
	case "Write":
		if path := tc.Params["file_path"]; path != "" {
			return &FileReference{
				Path:   path,
				Action: ActionCreated,
			}
		}
	case "Glob", "Grep":
		// These return multiple paths in results
		// For now, just note the search was performed
		if path := tc.Params["path"]; path != "" {
			return &FileReference{
				Path:   path,
				Action: ActionRead,
			}
		}
	case "Bash":
		// Check for file operations in command
		cmd := tc.Params["command"]
		if strings.Contains(cmd, "rm ") || strings.Contains(cmd, "rm -") {
			paths := e.extractFilePathsFromText(cmd)
			if len(paths) > 0 {
				return &FileReference{
					Path:   paths[0],
					Action: ActionDeleted,
				}
			}
		}
	}
	return nil
}

// extractLineRanges extracts line range information from tool params.
func extractLineRanges(params map[string]string) []string {
	var ranges []string
	if offset := params["offset"]; offset != "" {
		if limit := params["limit"]; limit != "" {
			ranges = append(ranges, offset+"-"+limit)
		} else {
			ranges = append(ranges, offset)
		}
	}
	return ranges
}

// extractFilePathsFromText extracts file paths mentioned in text.
func (e *Extractor) extractFilePathsFromText(text string) []string {
	matches := e.filePathPattern.FindAllStringSubmatch(text, -1)
	paths := make([]string, 0, len(matches))
	seen := make(map[string]bool)

	for _, match := range matches {
		if len(match) > 1 {
			path := match[1]
			// Filter out common false positives
			if isValidFilePath(path) && !seen[path] {
				paths = append(paths, path)
				seen[path] = true
			}
		}
	}
	return paths
}

// isValidFilePath checks if a string looks like a valid file path.
func isValidFilePath(path string) bool {
	// Must contain at least one directory separator or be a simple filename
	if len(path) < 3 {
		return false
	}

	// Filter out URLs
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return false
	}

	// Filter out version numbers (e.g., "v1.0.0")
	if strings.HasPrefix(path, "v") && regexp.MustCompile(`^v\d+\.\d+`).MatchString(path) {
		return false
	}

	// Filter out common false positives
	falsePositives := []string{
		"0.0.0", "1.0.0", "2.0.0", // Version-like
		"e.g.", "i.e.", "etc.",    // Abbreviations
	}
	for _, fp := range falsePositives {
		if path == fp {
			return false
		}
	}

	// Should have a reasonable extension
	parts := strings.Split(path, ".")
	if len(parts) < 2 {
		return false
	}
	ext := parts[len(parts)-1]
	if len(ext) > 10 || len(ext) < 1 {
		return false
	}

	return true
}

// ExtractCommitReferences extracts git commit SHAs from messages.
func (e *Extractor) ExtractCommitReferences(msg RawMessage) []CommitReference {
	var refs []CommitReference
	seen := make(map[string]bool)

	// Check tool results for git output
	for _, tc := range msg.ToolCalls {
		if tc.Name == "Bash" {
			cmd := tc.Params["command"]
			if strings.Contains(cmd, "git") {
				commits := e.extractCommitsFromGitOutput(tc.Result, cmd)
				for _, c := range commits {
					if !seen[c.SHA] {
						refs = append(refs, c)
						seen[c.SHA] = true
					}
				}
			}
		}
	}

	return refs
}

// extractCommitsFromGitOutput extracts commit info from git command output.
func (e *Extractor) extractCommitsFromGitOutput(output, cmd string) []CommitReference {
	var refs []CommitReference

	// For git commit output, look for the commit SHA
	if strings.Contains(cmd, "git commit") {
		// Pattern: [branch SHA] message
		if matches := regexp.MustCompile(`\[[\w\-/]+\s+([a-f0-9]{7,40})\]\s+(.+)`).FindStringSubmatch(output); len(matches) > 2 {
			refs = append(refs, CommitReference{
				SHA:     matches[1],
				Message: strings.TrimSpace(matches[2]),
			})
		}
	}

	// For git log output
	if strings.Contains(cmd, "git log") {
		// Pattern: commit SHA or short SHA at line start
		lines := strings.Split(output, "\n")
		for i, line := range lines {
			if strings.HasPrefix(line, "commit ") {
				sha := strings.TrimPrefix(line, "commit ")
				sha = strings.TrimSpace(sha)
				// Try to get message from next non-empty line after author/date
				var message string
				for j := i + 1; j < len(lines) && j < i+5; j++ {
					l := strings.TrimSpace(lines[j])
					if l != "" && !strings.HasPrefix(l, "Author:") && !strings.HasPrefix(l, "Date:") {
						message = l
						break
					}
				}
				refs = append(refs, CommitReference{
					SHA:     sha,
					Message: message,
				})
			}
		}
	}

	// Generic SHA extraction as fallback
	if len(refs) == 0 {
		matches := e.commitSHAPattern.FindAllString(output, -1)
		for _, sha := range matches {
			// Filter out common false positives (like color codes, test data)
			if len(sha) >= 7 && !isCommonHexValue(sha) {
				refs = append(refs, CommitReference{SHA: sha})
			}
		}
	}

	return refs
}

// isCommonHexValue filters out hex strings that are unlikely to be commits.
func isCommonHexValue(s string) bool {
	common := []string{
		"0000000", "fffffff", "1234567", "abcdefg",
	}
	lower := strings.ToLower(s)
	for _, c := range common {
		if strings.HasPrefix(lower, c) {
			return true
		}
	}
	return false
}

// ExtractMetadata extracts all metadata from a message.
func (e *Extractor) ExtractMetadata(msg RawMessage) (files []FileReference, commits []CommitReference) {
	files = e.ExtractFileReferences(msg)
	commits = e.ExtractCommitReferences(msg)
	return
}
