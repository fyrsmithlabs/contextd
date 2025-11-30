package secrets

import (
	"fmt"
	"regexp"
)

// Config configures the scrubber.
type Config struct {
	// Enabled controls whether scrubbing is active (default: true)
	Enabled bool `koanf:"enabled"`

	// Rules defines the detection rules
	Rules []Rule `koanf:"rules"`

	// RedactionString is the replacement for detected secrets (default: "[REDACTED]")
	RedactionString string `koanf:"redaction_string"`

	// AllowList contains patterns to skip during scrubbing
	AllowList []string `koanf:"allow_list"`

	// compiled patterns (populated by Validate)
	compiledRules     []*compiledRule
	compiledAllowList []*regexp.Regexp
}

// Rule defines a secret detection rule.
type Rule struct {
	// ID is the unique identifier for this rule
	ID string `koanf:"id"`

	// Description explains what this rule detects
	Description string `koanf:"description"`

	// Pattern is the regex pattern to match secrets
	Pattern string `koanf:"pattern"`

	// Keywords are optional keywords that must be present for the rule to apply
	Keywords []string `koanf:"keywords"`

	// Severity indicates the importance (high, medium, low)
	Severity string `koanf:"severity"`

	// Entropy is the minimum entropy threshold (0 to disable)
	Entropy float64 `koanf:"entropy"`
}

// compiledRule holds a compiled rule.
type compiledRule struct {
	Rule
	pattern  *regexp.Regexp
	keywords []*regexp.Regexp
}

// DefaultConfig returns a configuration with standard secret detection rules.
func DefaultConfig() *Config {
	cfg := &Config{
		Enabled:         true,
		RedactionString: "[REDACTED]",
		Rules:           DefaultRules(),
		AllowList:       []string{},
	}
	return cfg
}

// Validate validates and compiles the configuration.
func (c *Config) Validate() error {
	if !c.Enabled {
		return nil
	}

	if c.RedactionString == "" {
		c.RedactionString = "[REDACTED]"
	}

	// Compile rules
	c.compiledRules = make([]*compiledRule, 0, len(c.Rules))
	for i, rule := range c.Rules {
		if rule.ID == "" {
			return fmt.Errorf("rule %d: ID is required", i)
		}
		if rule.Pattern == "" {
			return fmt.Errorf("rule %s: pattern is required", rule.ID)
		}

		pattern, err := regexp.Compile(rule.Pattern)
		if err != nil {
			return fmt.Errorf("rule %s: invalid pattern: %w", rule.ID, err)
		}

		compiled := &compiledRule{
			Rule:     rule,
			pattern:  pattern,
			keywords: make([]*regexp.Regexp, 0, len(rule.Keywords)),
		}

		// Compile keywords (case-insensitive)
		for _, kw := range rule.Keywords {
			kwPattern, err := regexp.Compile("(?i)" + regexp.QuoteMeta(kw))
			if err != nil {
				return fmt.Errorf("rule %s: invalid keyword %q: %w", rule.ID, kw, err)
			}
			compiled.keywords = append(compiled.keywords, kwPattern)
		}

		c.compiledRules = append(c.compiledRules, compiled)
	}

	// Compile allow list
	c.compiledAllowList = make([]*regexp.Regexp, 0, len(c.AllowList))
	for i, pattern := range c.AllowList {
		compiled, err := regexp.Compile(pattern)
		if err != nil {
			return fmt.Errorf("allow_list %d: invalid pattern: %w", i, err)
		}
		c.compiledAllowList = append(c.compiledAllowList, compiled)
	}

	return nil
}
