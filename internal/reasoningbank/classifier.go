package reasoningbank

import (
	"regexp"
	"strings"
)

// categoryRule pairs a compiled regex with the category it detects and
// a base confidence score. Rules are evaluated in order; the first match wins.
// This follows the patternRule pattern from extractor.go.
type categoryRule struct {
	regex      *regexp.Regexp
	category   MemoryCategory
	confidence float64
}

// CategoryRule is the exported version of categoryRule for use in ConfigurableClassifier.
// Users construct these to define custom classification rules at org/team/project level.
type CategoryRule struct {
	// Pattern is a regex pattern string. Compiled at construction time.
	Pattern string
	// Category is the target category when the pattern matches.
	Category MemoryCategory
	// Confidence is the base confidence score (0.0-1.0) for this rule.
	Confidence float64
}

// RuleScope defines the hierarchy level a rule set applies to.
// Mirrors remediation.Scope for consistency across the codebase.
type RuleScope string

const (
	// RuleScopeProject applies rules at the project level (highest priority).
	RuleScopeProject RuleScope = "project"
	// RuleScopeTeam applies rules at the team level (medium priority).
	RuleScopeTeam RuleScope = "team"
	// RuleScopeOrg applies rules at the org level (lowest custom priority).
	RuleScopeOrg RuleScope = "org"
)

// RegexCategoryClassifier classifies memory content using ordered regex rules.
// Thread-safe: all regex patterns are compiled at construction time.
type RegexCategoryClassifier struct {
	rules []*categoryRule
}

// NewRegexCategoryClassifier creates a classifier with built-in rules.
func NewRegexCategoryClassifier() *RegexCategoryClassifier {
	return &RegexCategoryClassifier{
		rules: buildCategoryRules(),
	}
}

// buildCategoryRules returns ordered regex rules for category classification.
// More specific patterns are listed first to avoid shadowing.
// All patterns use (?i) for case-insensitive matching.
func buildCategoryRules() []*categoryRule {
	return []*categoryRule{
		// --- Security (highest priority -- overlaps with debugging) ---
		{
			regex:      regexp.MustCompile(`(?i)\b(?:vulnerab|CVE-\d|OWASP|injection|XSS|CSRF|auth(?:entication|orization)|secrets?\s+(?:leak|expos)|SQL\s+inject|command\s+inject|path\s+travers)`),
			category:   CategorySecurity,
			confidence: 0.9,
		},

		// --- Operational ---
		{
			// Build/run commands, env vars, ports, config files, Docker, k8s
			regex:      regexp.MustCompile(`(?i)\b(?:(?:go|npm|yarn|pip|cargo|make|docker|kubectl|helm|terraform)\s+(?:run|build|install|test|deploy|start|exec|apply)|localhost:\d+|port\s+\d+|\.env\b|(?:config|\.ya?ml|\.toml|\.ini|Makefile|Dockerfile|docker-compose)\b|env(?:ironment)?\s+var|CONTEXTD_|QDRANT_|OTEL_|SERVER_PORT)`),
			category:   CategoryOperational,
			confidence: 0.85,
		},

		// --- Debugging ---
		{
			regex:      regexp.MustCompile(`(?i)\b(?:(?:root\s+cause|stack\s*trace|panic|segfault|deadlock|race\s+condition|goroutine\s+leak)\b|(?:error|bug|fix(?:ed)?|workaround|regression|flaky)\b.*\b(?:when|because|caused|due\s+to)\b)`),
			category:   CategoryDebugging,
			confidence: 0.85,
		},

		// --- Architectural ---
		{
			regex:      regexp.MustCompile(`(?i)\b(?:design\s+(?:decision|pattern|choice)|(?:service|repository|factory|adapter|strategy|observer|middleware)\s+pattern|interface\s+(?:design|contract|boundary)|module\s+(?:boundary|structure)|abstraction|(?:coupling|cohesion|separation\s+of\s+concerns)|(?:domain|clean|hexagonal|layered)\s+architecture)`),
			category:   CategoryArchitectural,
			confidence: 0.85,
		},

		// --- Feature ---
		{
			regex:      regexp.MustCompile(`(?i)\b(?:implement(?:ed|ing)?\s+(?:the|a|new)\b|(?:API|REST|gRPC|MCP)\s+(?:endpoint|handler|tool)|workflow\s+(?:for|to|that)|feature\s+(?:flag|toggle|gate)|user\s+(?:story|flow|journey))`),
			category:   CategoryFeature,
			confidence: 0.8,
		},

		// Broader debugging fallback (single keywords with lower confidence)
		{
			regex:      regexp.MustCompile(`(?i)\b(?:error|bug|fix|stacktrace|root\s*cause|workaround|regression|crash|nil\s+pointer|timeout|OOM)\b`),
			category:   CategoryDebugging,
			confidence: 0.7,
		},

		// Broader operational fallback
		{
			regex:      regexp.MustCompile(`(?i)\b(?:deploy|CI/?CD|pipeline|container|image|volume|network|cluster|scaling|replicas|health\s*check|readiness|liveness)\b`),
			category:   CategoryOperational,
			confidence: 0.7,
		},

		// Broader architectural fallback
		{
			regex:      regexp.MustCompile(`(?i)\b(?:refactor|decouple|extract|migrate|split|merge|consolidat|restructur)\b`),
			category:   CategoryArchitectural,
			confidence: 0.65,
		},

		// Broader feature fallback
		{
			regex:      regexp.MustCompile(`(?i)\b(?:endpoint|handler|route|controller|service\s+method|business\s+logic)\b`),
			category:   CategoryFeature,
			confidence: 0.6,
		},
	}
}

// Classify returns the best-matching category and confidence for the given content.
// If no rule matches, returns CategoryGeneral with 0.5 confidence.
func (c *RegexCategoryClassifier) Classify(title, content string, tags []string) (MemoryCategory, float64) {
	// Combine inputs for matching (title is weighted by appearing first)
	combined := title + " " + content
	if len(tags) > 0 {
		combined += " " + strings.Join(tags, " ")
	}

	// Truncate to prevent ReDoS on extremely long inputs
	if len(combined) > maxQueryLength {
		combined = combined[:maxQueryLength]
	}

	for _, rule := range c.rules {
		if rule.regex.MatchString(combined) {
			return rule.category, rule.confidence
		}
	}

	return CategoryGeneral, 0.5
}

// scopedRuleSet holds compiled rules for a single scope level.
type scopedRuleSet struct {
	scope RuleScope
	rules []*categoryRule
}

// ConfigurableClassifier extends RegexCategoryClassifier with hierarchical
// custom rules at org, team, and project levels. Rules are evaluated in
// priority order: project → team → org → built-in.
//
// This mirrors the remediation service's scope hierarchy pattern where
// project-level rules override team, team overrides org.
//
// Thread-safe: all rules are compiled at construction time and immutable.
type ConfigurableClassifier struct {
	// scopedRules holds custom rule sets ordered by priority (project first).
	scopedRules []scopedRuleSet
	// fallback is the built-in RegexCategoryClassifier used when no custom rules match.
	fallback *RegexCategoryClassifier
}

// ConfigurableClassifierOption configures a ConfigurableClassifier.
type ConfigurableClassifierOption func(*ConfigurableClassifier)

// WithScopedRules adds a set of custom rules at a specific scope level.
// Rules are compiled at construction time; invalid patterns cause an error.
func WithScopedRules(scope RuleScope, rules []CategoryRule) ConfigurableClassifierOption {
	return func(c *ConfigurableClassifier) {
		compiled := make([]*categoryRule, 0, len(rules))
		for _, r := range rules {
			re, err := regexp.Compile(r.Pattern)
			if err != nil {
				// Skip invalid patterns (logged at a higher level)
				continue
			}
			compiled = append(compiled, &categoryRule{
				regex:      re,
				category:   r.Category,
				confidence: r.Confidence,
			})
		}
		if len(compiled) > 0 {
			c.scopedRules = append(c.scopedRules, scopedRuleSet{
				scope: scope,
				rules: compiled,
			})
		}
	}
}

// NewConfigurableClassifier creates a classifier with hierarchical custom rules.
// Custom rules are evaluated in scope priority order (project → team → org),
// falling back to built-in regex rules if no custom rule matches.
func NewConfigurableClassifier(opts ...ConfigurableClassifierOption) *ConfigurableClassifier {
	c := &ConfigurableClassifier{
		fallback: NewRegexCategoryClassifier(),
	}
	for _, opt := range opts {
		opt(c)
	}
	// Sort scoped rules by priority: project > team > org
	sortScopedRules(c.scopedRules)
	return c
}

// sortScopedRules orders scoped rule sets by priority (project first).
func sortScopedRules(sets []scopedRuleSet) {
	priority := map[RuleScope]int{
		RuleScopeProject: 0,
		RuleScopeTeam:    1,
		RuleScopeOrg:     2,
	}
	for i := 1; i < len(sets); i++ {
		for j := i; j > 0 && priority[sets[j].scope] < priority[sets[j-1].scope]; j-- {
			sets[j], sets[j-1] = sets[j-1], sets[j]
		}
	}
}

// Classify evaluates custom rules in scope priority order, then falls back
// to built-in rules. Returns the best-matching category and confidence.
//
// Evaluation order: project rules → team rules → org rules → built-in rules.
// Within each scope, first-match wins (same as RegexCategoryClassifier).
func (c *ConfigurableClassifier) Classify(title, content string, tags []string) (MemoryCategory, float64) {
	combined := title + " " + content
	if len(tags) > 0 {
		combined += " " + strings.Join(tags, " ")
	}
	if len(combined) > maxQueryLength {
		combined = combined[:maxQueryLength]
	}

	// Evaluate scoped rules in priority order (project → team → org)
	for _, ruleSet := range c.scopedRules {
		for _, rule := range ruleSet.rules {
			if rule.regex.MatchString(combined) {
				return rule.category, rule.confidence
			}
		}
	}

	// Fall back to built-in rules
	return c.fallback.Classify(title, content, tags)
}
