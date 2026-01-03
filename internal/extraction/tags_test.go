package extraction

import (
	"sort"
	"testing"
)

func TestDefaultTagExtractor_ExtractTags(t *testing.T) {
	extractor := NewTagExtractor(nil) // Use defaults

	tests := []struct {
		name     string
		content  string
		wantTags []string
	}{
		{
			name:     "golang content",
			content:  "Let me run go test to verify this change.",
			wantTags: []string{"golang", "testing"},
		},
		{
			name:     "kubernetes deployment",
			content:  "Apply this kubectl deployment to the k8s cluster.",
			wantTags: []string{"kubernetes"},
		},
		{
			name:     "debugging session",
			content:  "There's a bug in the error handling code.",
			wantTags: []string{"debugging"},
		},
		{
			name:     "multiple activities",
			content:  "After refactoring, let's add some tests to verify the fix.",
			wantTags: []string{"refactoring", "testing", "debugging"},
		},
		{
			name:     "no matching tags",
			content:  "Hello, how are you today?",
			wantTags: []string{},
		},
		{
			name:     "case insensitive",
			content:  "Using DOCKER to create the container.",
			wantTags: []string{"docker"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractor.ExtractTags(tt.content)

			// Sort for comparison
			sort.Strings(got)
			sort.Strings(tt.wantTags)

			if len(got) != len(tt.wantTags) {
				t.Errorf("ExtractTags() got %v, want %v", got, tt.wantTags)
				return
			}

			for i := range got {
				if got[i] != tt.wantTags[i] {
					t.Errorf("ExtractTags() got %v, want %v", got, tt.wantTags)
					return
				}
			}
		})
	}
}

func TestDefaultTagExtractor_ExtractTagsFromFiles(t *testing.T) {
	extractor := NewTagExtractor(nil) // Use defaults

	tests := []struct {
		name     string
		paths    []string
		wantTags []string
	}{
		{
			name:     "go files",
			paths:    []string{"/path/to/main.go", "/path/to/handler.go"},
			wantTags: []string{"backend", "golang"}, // handler matches backend
		},
		{
			name:     "typescript files",
			paths:    []string{"/src/app.ts", "/src/types.tsx"},
			wantTags: []string{"typescript"},
		},
		{
			name:     "dockerfile only",
			paths:    []string{"Dockerfile"},
			wantTags: []string{"docker"},
		},
		{
			name:     "kubernetes manifests",
			paths:    []string{"deployment.yaml", "pod.yaml"},
			wantTags: []string{"kubernetes"},
		},
		{
			name:     "terraform files",
			paths:    []string{"main.tf", "variables.tf"},
			wantTags: []string{"terraform"},
		},
		{
			name:     "unknown files",
			paths:    []string{"notes.txt", "data.csv"},
			wantTags: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractor.ExtractTagsFromFiles(tt.paths)

			// Sort for comparison
			sort.Strings(got)
			sort.Strings(tt.wantTags)

			if len(got) != len(tt.wantTags) {
				t.Errorf("ExtractTagsFromFiles() got %v, want %v", got, tt.wantTags)
				return
			}

			for i := range got {
				if got[i] != tt.wantTags[i] {
					t.Errorf("ExtractTagsFromFiles() got %v, want %v", got, tt.wantTags)
					return
				}
			}
		})
	}
}

func TestNewTagExtractor_CustomRules(t *testing.T) {
	customRules := map[string][]string{
		"custom_tag": {"custom_keyword", "another_keyword"},
	}

	extractor := NewTagExtractor(customRules)

	// Should match custom rule
	tags := extractor.ExtractTags("This contains a custom_keyword.")
	if len(tags) != 1 || tags[0] != "custom_tag" {
		t.Errorf("ExtractTags() = %v, want [custom_tag]", tags)
	}

	// Should not match default rules
	tags = extractor.ExtractTags("go test ./...")
	if len(tags) != 0 {
		t.Errorf("ExtractTags() = %v, want [] (no default rules)", tags)
	}
}

func TestExtractDomain(t *testing.T) {
	tests := []struct {
		name   string
		tags   []string
		want   string
	}{
		{
			name: "kubernetes is priority",
			tags: []string{"golang", "testing", "kubernetes"},
			want: "kubernetes",
		},
		{
			name: "api over language",
			tags: []string{"golang", "api"},
			want: "api",
		},
		{
			name: "first tag when no priority",
			tags: []string{"golang", "python"},
			want: "golang", // First non-priority tag
		},
		{
			name: "empty tags",
			tags: []string{},
			want: "",
		},
		{
			name: "testing comes before security in priority",
			tags: []string{"testing", "security"},
			want: "testing", // testing is earlier in the priority list
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractDomain(tt.tags)
			if got != tt.want {
				t.Errorf("ExtractDomain() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDefaultTagRules(t *testing.T) {
	// Verify default rules exist and have content
	if len(DefaultTagRules) == 0 {
		t.Error("DefaultTagRules is empty")
	}

	// Check some expected categories exist
	expectedCategories := []string{"golang", "python", "docker", "kubernetes", "debugging", "testing"}
	for _, cat := range expectedCategories {
		if _, ok := DefaultTagRules[cat]; !ok {
			t.Errorf("DefaultTagRules missing category %q", cat)
		}
	}
}
