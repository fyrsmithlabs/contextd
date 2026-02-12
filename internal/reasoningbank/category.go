package reasoningbank

// MemoryCategory classifies the domain of a memory's content.
// Categories enable gap detection (e.g., missing operational knowledge)
// and category-aware search boosting.
type MemoryCategory string

const (
	// CategoryOperational covers build commands, environment setup, ports,
	// config files, Docker/k8s commands, and other "how to run" knowledge.
	CategoryOperational MemoryCategory = "operational"

	// CategoryArchitectural covers design decisions, patterns, interfaces,
	// module boundaries, and structural choices.
	CategoryArchitectural MemoryCategory = "architectural"

	// CategoryDebugging covers error fixes, bug analysis, stack traces,
	// root cause analysis, and workarounds.
	CategoryDebugging MemoryCategory = "debugging"

	// CategorySecurity covers vulnerabilities, CVEs, OWASP patterns,
	// authentication, authorization, and injection prevention.
	CategorySecurity MemoryCategory = "security"

	// CategoryFeature covers feature implementations, API endpoints,
	// handlers, workflows, and business logic.
	CategoryFeature MemoryCategory = "feature"

	// CategoryGeneral is the fallback when no specific category matches.
	CategoryGeneral MemoryCategory = "general"
)

// ValidCategories maps valid category strings to their typed values.
var ValidCategories = map[string]MemoryCategory{
	"operational":   CategoryOperational,
	"architectural": CategoryArchitectural,
	"debugging":     CategoryDebugging,
	"security":      CategorySecurity,
	"feature":       CategoryFeature,
	"general":       CategoryGeneral,
}

// IsValidCategory returns true if the string is a recognized category.
func IsValidCategory(s string) bool {
	_, ok := ValidCategories[s]
	return ok
}

// CategoryClassifier assigns a category and confidence to memory content.
type CategoryClassifier interface {
	// Classify returns the best-matching category and a confidence score (0.0-1.0).
	Classify(title, content string, tags []string) (MemoryCategory, float64)
}
