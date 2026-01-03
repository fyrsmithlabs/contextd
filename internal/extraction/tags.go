package extraction

import (
	"path/filepath"
	"strings"
)

// DefaultTagRules maps tags to keywords/patterns that indicate them.
var DefaultTagRules = map[string][]string{
	// Languages
	"golang":     {".go", "go mod", "go build", "go test", "golang"},
	"python":     {".py", "pip", "pytest", "python", "django", "flask"},
	"typescript": {".ts", ".tsx", "npm", "yarn", "node", "typescript"},
	"javascript": {".js", ".jsx", "npm", "node", "javascript"},
	"rust":       {".rs", "cargo", "rustc", "rust"},
	"java":       {".java", "maven", "gradle", "java"},

	// Infrastructure
	"kubernetes": {"kubectl", "k8s", "helm", "deployment.yaml", "service.yaml", "kubernetes"},
	"terraform":  {".tf", "terraform", "tfstate", "tfvars"},
	"docker":     {"Dockerfile", "docker-compose", "container", "image", "docker"},
	"aws":        {"aws", "s3", "ec2", "lambda", "cloudformation", "iam"},
	"gcp":        {"gcloud", "gcp", "pubsub", "bigquery", "gke"},

	// Activities
	"debugging":     {"fix", "bug", "error", "issue", "broken", "failing", "debug"},
	"documentation": {"docs", "readme", "comment", "explain", "document", "documentation"},
	"testing":       {"test", "spec", "coverage", "mock", "assert", "unittest"},
	"refactoring":   {"refactor", "cleanup", "rename", "extract", "simplify", "restructure"},
	"security":      {"auth", "secret", "credential", "permission", "encrypt", "security"},
	"performance":   {"optimize", "slow", "fast", "cache", "latency", "performance"},

	// Architecture
	"api":       {"api", "endpoint", "rest", "grpc", "graphql"},
	"database":  {"database", "sql", "postgres", "mysql", "mongodb", "redis"},
	"frontend":  {"frontend", "ui", "react", "vue", "angular", "css"},
	"backend":   {"backend", "server", "service", "handler"},
	"microservices": {"microservice", "service mesh", "istio"},
}

// TagRules represents configurable tag extraction rules.
type TagRules struct {
	Rules map[string][]string
}

// DefaultTagExtractor implements TagExtractor using keyword matching.
type DefaultTagExtractor struct {
	rules map[string][]string
}

// NewTagExtractor creates a new tag extractor with the given rules.
func NewTagExtractor(rules map[string][]string) *DefaultTagExtractor {
	if len(rules) == 0 {
		rules = DefaultTagRules
	}
	return &DefaultTagExtractor{rules: rules}
}

// ExtractTags extracts tags from content based on keyword matching.
func (t *DefaultTagExtractor) ExtractTags(content string) []string {
	content = strings.ToLower(content)
	tags := make(map[string]bool)

	for tag, keywords := range t.rules {
		for _, keyword := range keywords {
			if strings.Contains(content, strings.ToLower(keyword)) {
				tags[tag] = true
				break // Found one match, move to next tag
			}
		}
	}

	result := make([]string, 0, len(tags))
	for tag := range tags {
		result = append(result, tag)
	}
	return result
}

// ExtractTagsFromFiles extracts tags based on file extensions and paths.
func (t *DefaultTagExtractor) ExtractTagsFromFiles(paths []string) []string {
	tags := make(map[string]bool)

	for _, path := range paths {
		ext := filepath.Ext(path)
		base := filepath.Base(path)
		lowerPath := strings.ToLower(path)

		// Check file extension and name against rules
		for tag, keywords := range t.rules {
			for _, keyword := range keywords {
				keywordLower := strings.ToLower(keyword)
				if ext == keywordLower ||
					strings.Contains(lowerPath, keywordLower) ||
					strings.Contains(base, keywordLower) {
					tags[tag] = true
					break
				}
			}
		}
	}

	result := make([]string, 0, len(tags))
	for tag := range tags {
		result = append(result, tag)
	}
	return result
}

// ExtractDomain tries to determine the domain/area from tags.
func ExtractDomain(tags []string) string {
	// Priority order for domain detection
	domains := []string{
		"kubernetes", "terraform", "docker", "aws", "gcp",
		"frontend", "backend", "api", "database",
		"testing", "debugging", "security", "performance",
	}

	tagSet := make(map[string]bool)
	for _, t := range tags {
		tagSet[t] = true
	}

	for _, domain := range domains {
		if tagSet[domain] {
			return domain
		}
	}

	// Default to first tag if any
	if len(tags) > 0 {
		return tags[0]
	}

	return ""
}

// Ensure DefaultTagExtractor implements TagExtractor.
var _ TagExtractor = (*DefaultTagExtractor)(nil)
