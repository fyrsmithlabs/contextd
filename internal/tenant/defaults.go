package tenant

import (
	"os"
	"strings"

	"github.com/go-git/go-git/v5/config"
)

// GetDefaultTenantID returns the default tenant ID for single-tenant mode.
// Uses git user.name from global config if available, falls back to OS username.
func GetDefaultTenantID() string {
	// Try git global config user.name
	cfg, err := config.LoadConfig(config.GlobalScope)
	if err == nil && cfg.User.Name != "" {
		name := strings.ToLower(cfg.User.Name)
		name = strings.ReplaceAll(name, " ", "_")
		return sanitizeIdentifier(name)
	}

	// Fall back to OS username
	if user := os.Getenv("USER"); user != "" {
		return sanitizeIdentifier(strings.ToLower(user))
	}

	return "local"
}

// sanitizeIdentifier ensures the ID is valid for Qdrant collection names.
// Keeps only lowercase alphanumeric characters and underscores.
func sanitizeIdentifier(s string) string {
	var result strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
			result.WriteRune(r)
		}
	}
	if result.Len() == 0 {
		return "local"
	}
	return result.String()
}
