package ignore

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseLine(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		expected string
	}{
		{"empty line", "", ""},
		{"whitespace only", "   ", ""},
		{"comment", "# this is a comment", ""},
		{"negation skipped", "!important.txt", ""},
		{"simple file glob", "*.log", "*.log"},
		{"simple directory", "node_modules", "**/node_modules/**"},
		{"directory with slash", "node_modules/", "node_modules/**"},
		{"nested path", "vendor/cache", "vendor/cache/**"},
		{"absolute path", "/dist", "**/dist/**"},
		{"glob pattern", "*.pyc", "*.pyc"},
		{"double star pattern", "**/build", "**/build/**"},
		{"file with extension", "file.txt", "**/file.txt"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseLine(tt.line)
			if result != tt.expected {
				t.Errorf("parseLine(%q) = %q, want %q", tt.line, result, tt.expected)
			}
		})
	}
}

func TestParseProject(t *testing.T) {
	// Create temp directory with ignore files
	tmpDir := t.TempDir()

	// Create .gitignore
	gitignore := `# Build outputs
dist/
build/

# Dependencies
node_modules/

# Python
*.pyc
__pycache__/
`
	if err := os.WriteFile(filepath.Join(tmpDir, ".gitignore"), []byte(gitignore), 0644); err != nil {
		t.Fatal(err)
	}

	// Create .dockerignore with some overlap
	dockerignore := `node_modules/
.git/
*.log
`
	if err := os.WriteFile(filepath.Join(tmpDir, ".dockerignore"), []byte(dockerignore), 0644); err != nil {
		t.Fatal(err)
	}

	parser := NewParser(
		[]string{".gitignore", ".dockerignore"},
		[]string{"fallback/**"},
	)

	patterns, err := parser.ParseProject(tmpDir)
	if err != nil {
		t.Fatalf("ParseProject failed: %v", err)
	}

	// Check we got patterns (exact patterns depend on conversion logic)
	if len(patterns) == 0 {
		t.Error("expected patterns, got none")
	}

	// Check deduplication worked (node_modules appears in both files)
	count := 0
	for _, p := range patterns {
		if p == "**/node_modules/**" {
			count++
		}
	}
	if count > 1 {
		t.Errorf("expected node_modules pattern once, got %d times", count)
	}
}

func TestParseProject_NoIgnoreFiles(t *testing.T) {
	tmpDir := t.TempDir()

	fallback := []string{".git/**", "node_modules/**"}
	parser := NewParser(
		[]string{".gitignore", ".dockerignore"},
		fallback,
	)

	patterns, err := parser.ParseProject(tmpDir)
	if err != nil {
		t.Fatalf("ParseProject failed: %v", err)
	}

	// Should return fallback patterns
	if len(patterns) != len(fallback) {
		t.Errorf("expected %d fallback patterns, got %d", len(fallback), len(patterns))
	}

	for i, p := range patterns {
		if p != fallback[i] {
			t.Errorf("pattern[%d] = %q, want %q", i, p, fallback[i])
		}
	}
}

func TestDeduplicate(t *testing.T) {
	input := []string{"a", "b", "a", "c", "b", "d"}
	expected := []string{"a", "b", "c", "d"}

	result := deduplicate(input)

	if len(result) != len(expected) {
		t.Fatalf("got %d items, want %d", len(result), len(expected))
	}

	for i, v := range result {
		if v != expected[i] {
			t.Errorf("result[%d] = %q, want %q", i, v, expected[i])
		}
	}
}
