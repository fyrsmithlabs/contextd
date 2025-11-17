package secrets

import (
	"regexp"

	gitleaksConfig "github.com/zricethezav/gitleaks/v8/config"
	"github.com/zricethezav/gitleaks/v8/detect"
	gitleaksRegexp "github.com/zricethezav/gitleaks/v8/regexp"
)

// Finding represents a detected secret with location information.
type Finding struct {
	RuleID   string // Gitleaks rule ID (e.g., "github-pat")
	RuleDesc string // Human-readable description
	Line     int    // Line number where secret was found
	StartCol int    // Start column (0-indexed)
	EndCol   int    // End column (0-indexed)
	Match    string // The actual secret value
}

// Detect scans content for secrets using the Gitleaks SDK.
// Returns findings with position information for redaction.
//
// allowlist: Optional allowlist to exclude patterns (nil to skip)
func Detect(content string, allowlist *Allowlist) ([]Finding, error) {
	// Create detector with default Gitleaks config (800+ patterns)
	detector, err := detect.NewDetectorDefaultConfig()
	if err != nil {
		return nil, err
	}

	// Apply allowlist if provided
	if allowlist != nil {
		applyAllowlist(&detector.Config, allowlist)
	}

	// Scan content string directly
	gitleaksFindings := detector.DetectString(content)

	// Convert Gitleaks findings to our Finding type
	result := make([]Finding, 0, len(gitleaksFindings))
	for _, f := range gitleaksFindings {
		result = append(result, Finding{
			RuleID:   f.RuleID,
			RuleDesc: f.Description,
			Line:     f.StartLine,
			StartCol: f.StartColumn,
			EndCol:   f.EndColumn,
			Match:    f.Secret,
		})
	}

	return result, nil
}

// applyAllowlist merges allowlist patterns into Gitleaks config.
func applyAllowlist(cfg *gitleaksConfig.Config, allowlist *Allowlist) {
	// Create a new global allowlist entry
	globalAllowlist := &gitleaksConfig.Allowlist{
		Description: "Contextd user/project allowlist",
	}

	// Compile and add path patterns
	// Note: Patterns are pre-validated in loadTOML(). If compilation fails here,
	// it indicates a programming error (validation was bypassed).
	for _, pattern := range allowlist.Paths {
		re, err := regexp.Compile(pattern)
		if err != nil {
			// This should never happen - patterns are validated before reaching this code
			panic("BUG: pre-validated regex pattern failed to compile: " + pattern + ": " + err.Error())
		}
		globalAllowlist.Paths = append(globalAllowlist.Paths, (*gitleaksRegexp.Regexp)(re))
	}

	// Compile and add content regex patterns
	for _, pattern := range allowlist.Regexes {
		re, err := regexp.Compile(pattern)
		if err != nil {
			// This should never happen - patterns are validated before reaching this code
			panic("BUG: pre-validated regex pattern failed to compile: " + pattern + ": " + err.Error())
		}
		globalAllowlist.Regexes = append(globalAllowlist.Regexes, (*gitleaksRegexp.Regexp)(re))
	}

	// Add stopwords for allowlisted patterns
	globalAllowlist.StopWords = append(globalAllowlist.StopWords, allowlist.Regexes...)

	// Append to config allowlists
	cfg.Allowlists = append(cfg.Allowlists, globalAllowlist)
}
