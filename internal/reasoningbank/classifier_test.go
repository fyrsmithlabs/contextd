package reasoningbank

import (
	"strings"
	"sync"
	"testing"
)

func TestRegexCategoryClassifier_Classify(t *testing.T) {
	classifier := NewRegexCategoryClassifier()

	tests := []struct {
		name         string
		title        string
		content      string
		tags         []string
		wantCategory MemoryCategory
		wantMinConf  float64
	}{
		// --- Operational ---
		{
			name:         "go build command",
			title:        "Building the contextd binary",
			content:      "Run go build ./cmd/contextd to build the binary",
			wantCategory: CategoryOperational,
			wantMinConf:  0.8,
		},
		{
			name:         "docker compose",
			title:        "Start development stack",
			content:      "Use docker-compose up -d to start all services",
			wantCategory: CategoryOperational,
			wantMinConf:  0.8,
		},
		{
			name:         "environment variable",
			title:        "Configure vector store",
			content:      "Set CONTEXTD_VECTORSTORE_PROVIDER=chromem to use embedded store",
			wantCategory: CategoryOperational,
			wantMinConf:  0.8,
		},
		{
			name:         "port configuration",
			title:        "Server port",
			content:      "The HTTP server listens on localhost:9090 by default",
			wantCategory: CategoryOperational,
			wantMinConf:  0.8,
		},
		{
			name:         "npm install",
			title:        "Frontend setup",
			content:      "Run npm install to install all dependencies",
			wantCategory: CategoryOperational,
			wantMinConf:  0.8,
		},
		{
			name:         "CI/CD pipeline",
			title:        "Deploy pipeline",
			content:      "The CI/CD pipeline runs tests and deploys to staging",
			wantCategory: CategoryOperational,
			wantMinConf:  0.6,
		},

		// --- Architectural ---
		{
			name:         "design decision",
			title:        "Chose payload isolation over filesystem isolation",
			content:      "The design decision was to use payload-based tenant isolation for multi-tenancy",
			wantCategory: CategoryArchitectural,
			wantMinConf:  0.8,
		},
		{
			name:         "service pattern",
			title:        "Repository pattern for data access",
			content:      "We use a repository pattern to abstract vectorstore access",
			wantCategory: CategoryArchitectural,
			wantMinConf:  0.8,
		},
		{
			name:         "module boundary",
			title:        "Module boundary for secrets",
			content:      "The module boundary between secrets and MCP ensures scrubbing happens at the right layer",
			wantCategory: CategoryArchitectural,
			wantMinConf:  0.8,
		},
		{
			name:         "refactor",
			title:        "Refactored memory storage",
			content:      "Refactor the storage layer to decouple from specific backends",
			wantCategory: CategoryArchitectural,
			wantMinConf:  0.6,
		},

		// --- Debugging ---
		{
			name:         "root cause analysis",
			title:        "Memory loss bug",
			content:      "Root cause was that metadata was lost during the delete-then-add update pattern",
			wantCategory: CategoryDebugging,
			wantMinConf:  0.8,
		},
		{
			name:         "stack trace error",
			title:        "Panic in search",
			content:      "The panic stack trace showed a nil pointer dereference when the collection was empty",
			wantCategory: CategoryDebugging,
			wantMinConf:  0.8,
		},
		{
			name:         "fix with context",
			title:        "Fix nil pointer crash",
			content:      "The error occurred because the embedder was nil when repository search was called",
			wantCategory: CategoryDebugging,
			wantMinConf:  0.6,
		},

		// --- Security ---
		{
			name:         "vulnerability fix",
			title:        "CVE-2024-1234 mitigation",
			content:      "Applied fix for CVE-2024-1234 vulnerability in the auth module",
			wantCategory: CategorySecurity,
			wantMinConf:  0.8,
		},
		{
			name:         "OWASP pattern",
			title:        "Prevent SQL injection",
			content:      "Following OWASP guidelines to prevent SQL injection in user input handling",
			wantCategory: CategorySecurity,
			wantMinConf:  0.8,
		},
		{
			name:         "path traversal",
			title:        "Path traversal protection",
			content:      "Added path traversal prevention for project_path input validation",
			wantCategory: CategorySecurity,
			wantMinConf:  0.8,
		},
		{
			name:         "authentication pattern",
			title:        "Auth middleware",
			content:      "Authentication bypass vulnerability in the tenant context middleware",
			wantCategory: CategorySecurity,
			wantMinConf:  0.8,
		},

		// --- Feature ---
		{
			name:         "implemented new feature",
			title:        "Added memory consolidation",
			content:      "Implemented the new consolidation workflow for merging similar memories",
			wantCategory: CategoryFeature,
			wantMinConf:  0.7,
		},
		{
			name:         "API endpoint",
			title:        "Checkpoint resume endpoint",
			content:      "Added a new MCP tool handler for checkpoint resume operations",
			wantCategory: CategoryFeature,
			wantMinConf:  0.7,
		},

		// --- General (fallback) ---
		{
			name:         "generic content",
			title:        "Team meeting notes",
			content:      "The team discussed upcoming priorities and resource allocation",
			wantCategory: CategoryGeneral,
			wantMinConf:  0.5,
		},

		// --- Edge cases ---
		{
			name:         "empty content",
			title:        "",
			content:      "",
			wantCategory: CategoryGeneral,
			wantMinConf:  0.5,
		},
		{
			name:         "tags influence",
			title:        "Some title",
			content:      "Some content about things",
			tags:         []string{"docker", "deploy"},
			wantCategory: CategoryOperational,
			wantMinConf:  0.6,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			category, confidence := classifier.Classify(tt.title, tt.content, tt.tags)
			if category != tt.wantCategory {
				t.Errorf("Classify() category = %q, want %q", category, tt.wantCategory)
			}
			if confidence < tt.wantMinConf {
				t.Errorf("Classify() confidence = %f, want >= %f", confidence, tt.wantMinConf)
			}
		})
	}
}

func TestRegexCategoryClassifier_LongInput(t *testing.T) {
	classifier := NewRegexCategoryClassifier()

	// Create input longer than maxQueryLength
	longContent := strings.Repeat("some generic content about things ", 200)
	category, confidence := classifier.Classify("title", longContent, nil)

	// Should not panic and should return general
	if category != CategoryGeneral {
		t.Errorf("expected CategoryGeneral for long generic input, got %q", category)
	}
	if confidence != 0.5 {
		t.Errorf("expected 0.5 confidence for general fallback, got %f", confidence)
	}
}

func TestRegexCategoryClassifier_ConcurrentSafety(t *testing.T) {
	classifier := NewRegexCategoryClassifier()

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			title := "test"
			content := "go build ./cmd/contextd"
			if n%2 == 0 {
				content = "CVE-2024-1234 vulnerability found"
			}
			category, _ := classifier.Classify(title, content, nil)
			if n%2 == 0 {
				if category != CategorySecurity {
					t.Errorf("goroutine %d: expected security, got %s", n, category)
				}
			} else {
				if category != CategoryOperational {
					t.Errorf("goroutine %d: expected operational, got %s", n, category)
				}
			}
		}(i)
	}
	wg.Wait()
}

// ===== ConfigurableClassifier Tests =====

func TestConfigurableClassifier_ProjectOverridesBuiltIn(t *testing.T) {
	// Project rule classifies "temporal" content as operational (project-specific term)
	classifier := NewConfigurableClassifier(
		WithScopedRules(RuleScopeProject, []CategoryRule{
			{Pattern: `(?i)\btemporal\s+workflow`, Category: CategoryOperational, Confidence: 0.95},
		}),
	)

	// "temporal workflow" would normally be classified as feature (mentions "workflow")
	// but project rule should override
	cat, conf := classifier.Classify("Temporal workflow setup", "How to run the temporal workflow locally", nil)
	if cat != CategoryOperational {
		t.Errorf("expected operational (project override), got %q", cat)
	}
	if conf < 0.9 {
		t.Errorf("expected confidence >= 0.9, got %f", conf)
	}
}

func TestConfigurableClassifier_ScopePriorityOrder(t *testing.T) {
	// Org says "bazel" is operational, team says "bazel" is architectural,
	// project says "bazel" is debugging.
	classifier := NewConfigurableClassifier(
		WithScopedRules(RuleScopeOrg, []CategoryRule{
			{Pattern: `(?i)\bbazel\b`, Category: CategoryOperational, Confidence: 0.8},
		}),
		WithScopedRules(RuleScopeTeam, []CategoryRule{
			{Pattern: `(?i)\bbazel\b`, Category: CategoryArchitectural, Confidence: 0.85},
		}),
		WithScopedRules(RuleScopeProject, []CategoryRule{
			{Pattern: `(?i)\bbazel\b`, Category: CategoryDebugging, Confidence: 0.9},
		}),
	)

	// Project should win regardless of insertion order
	cat, _ := classifier.Classify("Bazel build issue", "The bazel cache was corrupted", nil)
	if cat != CategoryDebugging {
		t.Errorf("expected debugging (project priority), got %q", cat)
	}
}

func TestConfigurableClassifier_FallsBackToBuiltIn(t *testing.T) {
	// Custom rules don't match, should fall back to built-in
	classifier := NewConfigurableClassifier(
		WithScopedRules(RuleScopeProject, []CategoryRule{
			{Pattern: `(?i)\bmy-custom-tool\b`, Category: CategoryOperational, Confidence: 0.95},
		}),
	)

	// Content matches built-in security rules, not custom
	cat, _ := classifier.Classify("CVE fix", "Fixed CVE-2024-5678 vulnerability", nil)
	if cat != CategorySecurity {
		t.Errorf("expected security (built-in fallback), got %q", cat)
	}
}

func TestConfigurableClassifier_NoCustomRules(t *testing.T) {
	// Should behave identically to RegexCategoryClassifier
	classifier := NewConfigurableClassifier()

	cat, conf := classifier.Classify("go build", "Run go build ./cmd/contextd", nil)
	if cat != CategoryOperational {
		t.Errorf("expected operational, got %q", cat)
	}
	if conf < 0.8 {
		t.Errorf("expected confidence >= 0.8, got %f", conf)
	}
}

func TestConfigurableClassifier_InvalidPatternSkipped(t *testing.T) {
	// Invalid regex should be skipped, not panic
	classifier := NewConfigurableClassifier(
		WithScopedRules(RuleScopeProject, []CategoryRule{
			{Pattern: `(?i)[invalid`, Category: CategoryOperational, Confidence: 0.9}, // bad regex
			{Pattern: `(?i)\bvalid-tool\b`, Category: CategoryDebugging, Confidence: 0.85},
		}),
	)

	// The valid rule should still work
	cat, _ := classifier.Classify("valid-tool issue", "The valid-tool is crashing", nil)
	if cat != CategoryDebugging {
		t.Errorf("expected debugging (valid rule), got %q", cat)
	}
}

func TestConfigurableClassifier_ConcurrentSafety(t *testing.T) {
	classifier := NewConfigurableClassifier(
		WithScopedRules(RuleScopeProject, []CategoryRule{
			{Pattern: `(?i)\bcustom-cmd\b`, Category: CategoryOperational, Confidence: 0.9},
		}),
	)

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			if n%2 == 0 {
				cat, _ := classifier.Classify("custom-cmd", "Run custom-cmd deploy", nil)
				if cat != CategoryOperational {
					t.Errorf("goroutine %d: expected operational, got %s", n, cat)
				}
			} else {
				cat, _ := classifier.Classify("CVE fix", "CVE-2024-1234 found", nil)
				if cat != CategorySecurity {
					t.Errorf("goroutine %d: expected security, got %s", n, cat)
				}
			}
		}(i)
	}
	wg.Wait()
}

func TestIsValidCategory(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"operational", true},
		{"architectural", true},
		{"debugging", true},
		{"security", true},
		{"feature", true},
		{"general", true},
		{"", false},
		{"invalid", false},
		{"OPERATIONAL", false}, // case-sensitive
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := IsValidCategory(tt.input); got != tt.want {
				t.Errorf("IsValidCategory(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
