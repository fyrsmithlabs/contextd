package secrets

import (
	"sort"
	"strings"
	"sync"
	"time"
)

// Scrubber detects and redacts secrets from content.
type Scrubber interface {
	// Scrub redacts secrets from the content.
	Scrub(content string) *Result

	// ScrubBytes redacts secrets from byte content.
	ScrubBytes(content []byte) *Result

	// Check detects secrets without redacting.
	Check(content string) *Result

	// IsEnabled returns whether scrubbing is enabled.
	IsEnabled() bool
}

// scrubber is the default implementation using regexp patterns.
type scrubber struct {
	config *Config
	mu     sync.RWMutex
}

// redaction tracks a position to redact.
type redaction struct {
	start, end int
	ruleID     string
}

// New creates a new Scrubber with the given configuration.
// If config is nil, DefaultConfig() is used.
func New(cfg *Config) (Scrubber, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &scrubber{
		config: cfg,
	}, nil
}

// MustNew creates a new Scrubber, panicking on error.
func MustNew(cfg *Config) Scrubber {
	s, err := New(cfg)
	if err != nil {
		panic(err)
	}
	return s
}

// Scrub redacts secrets from the content.
func (s *scrubber) Scrub(content string) *Result {
	start := time.Now()
	result := &Result{
		Original: content,
		Scrubbed: content,
		Findings: make([]Finding, 0),
		ByRule:   make(map[string]int),
	}

	if !s.config.Enabled {
		result.Duration = time.Since(start)
		return result
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	// Track positions to redact (to handle overlapping matches)
	redactions := make([]redaction, 0)

	// Check each rule
	for _, rule := range s.config.compiledRules {
		// If rule has keywords, check if any are present
		if len(rule.keywords) > 0 {
			hasKeyword := false
			for _, kw := range rule.keywords {
				if kw.MatchString(content) {
					hasKeyword = true
					break
				}
			}
			if !hasKeyword {
				continue
			}
		}

		// Find all matches
		matches := rule.pattern.FindAllStringIndex(content, -1)
		for _, match := range matches {
			matchStr := content[match[0]:match[1]]

			// Check against allow list
			if s.isAllowed(matchStr) {
				continue
			}

			// Calculate line number
			line := strings.Count(content[:match[0]], "\n") + 1

			finding := Finding{
				RuleID:      rule.ID,
				Description: rule.Description,
				Severity:    rule.Severity,
				StartIndex:  match[0],
				EndIndex:    match[1],
				Line:        line,
			}

			result.Findings = append(result.Findings, finding)
			result.ByRule[rule.ID]++

			redactions = append(redactions, redaction{
				start:  match[0],
				end:    match[1],
				ruleID: rule.ID,
			})
		}
	}

	result.TotalFindings = len(result.Findings)

	// Apply redactions (merge overlapping, then apply in reverse order)
	if len(redactions) > 0 {
		// Sort by start position ascending first
		sortRedactionsAsc(redactions)

		// Merge overlapping redactions
		merged := mergeRedactions(redactions)

		// Sort by start position descending for safe replacement
		sortRedactions(merged)

		scrubbed := content
		for _, r := range merged {
			if r.start >= 0 && r.end <= len(scrubbed) && r.start < r.end {
				scrubbed = scrubbed[:r.start] + s.config.RedactionString + scrubbed[r.end:]
			}
		}
		result.Scrubbed = scrubbed
	}

	result.Duration = time.Since(start)
	return result
}

// ScrubBytes redacts secrets from byte content.
func (s *scrubber) ScrubBytes(content []byte) *Result {
	return s.Scrub(string(content))
}

// Check detects secrets without redacting.
func (s *scrubber) Check(content string) *Result {
	result := s.Scrub(content)
	// Restore original content (check-only mode)
	result.Scrubbed = result.Original
	return result
}

// IsEnabled returns whether scrubbing is enabled.
func (s *scrubber) IsEnabled() bool {
	return s.config.Enabled
}

// isAllowed checks if the match is in the allow list.
func (s *scrubber) isAllowed(match string) bool {
	for _, pattern := range s.config.compiledAllowList {
		if pattern.MatchString(match) {
			return true
		}
	}
	return false
}

// sortRedactions sorts redactions by start position descending.
func sortRedactions(redactions []redaction) {
	sort.Slice(redactions, func(i, j int) bool {
		return redactions[i].start > redactions[j].start
	})
}

// sortRedactionsAsc sorts redactions by start position ascending.
func sortRedactionsAsc(redactions []redaction) {
	sort.Slice(redactions, func(i, j int) bool {
		return redactions[i].start < redactions[j].start
	})
}

// mergeRedactions merges overlapping or adjacent redactions.
func mergeRedactions(redactions []redaction) []redaction {
	if len(redactions) == 0 {
		return redactions
	}

	merged := []redaction{redactions[0]}

	for i := 1; i < len(redactions); i++ {
		last := &merged[len(merged)-1]
		curr := redactions[i]

		// If current overlaps with or is adjacent to last, merge them
		if curr.start <= last.end {
			if curr.end > last.end {
				last.end = curr.end
			}
		} else {
			merged = append(merged, curr)
		}
	}

	return merged
}

// NoopScrubber is a scrubber that does nothing (for testing or disabled mode).
type NoopScrubber struct{}

// Scrub returns content unchanged.
func (n *NoopScrubber) Scrub(content string) *Result {
	return &Result{
		Original:      content,
		Scrubbed:      content,
		Findings:      make([]Finding, 0),
		ByRule:        make(map[string]int),
		TotalFindings: 0,
	}
}

// ScrubBytes returns content unchanged.
func (n *NoopScrubber) ScrubBytes(content []byte) *Result {
	return n.Scrub(string(content))
}

// Check returns content unchanged.
func (n *NoopScrubber) Check(content string) *Result {
	return n.Scrub(content)
}

// IsEnabled returns false.
func (n *NoopScrubber) IsEnabled() bool {
	return false
}

// Compile-time check that scrubber implements Scrubber.
var _ Scrubber = (*scrubber)(nil)
var _ Scrubber = (*NoopScrubber)(nil)
