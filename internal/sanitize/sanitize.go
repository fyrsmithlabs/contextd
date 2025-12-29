// Package sanitize provides shared identifier sanitization for collection names.
//
// Collection names in vector stores (Qdrant, chromem) must match: ^[a-z0-9_]{1,64}$
// This package ensures all identifiers conform to this requirement.
package sanitize

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

const (
	// MaxIdentifierLength is the maximum length for collection name components.
	// Qdrant and chromem require collection names to be 1-64 characters.
	MaxIdentifierLength = 64

	// HashSuffixLength is the length of the hash suffix added to truncated identifiers.
	// Format: _<8-char-hash> = 9 characters total
	HashSuffixLength = 9

	// DefaultIdentifier is used when sanitization produces an empty result.
	DefaultIdentifier = "default"
)

// Identifier sanitizes a string for use in collection names.
//
// Rules applied:
//   - Converts to lowercase
//   - Replaces invalid characters with underscores
//   - Collapses multiple underscores
//   - Trims leading/trailing underscores
//   - Truncates to MaxIdentifierLength with hash suffix if too long
//   - Returns DefaultIdentifier if result would be empty
//
// Examples:
//
//	"github.com/user" -> "github_com_user"
//	"My Project!"     -> "my_project"
//	"" or "!!!"       -> "default"
func Identifier(s string) string {
	if s == "" {
		return DefaultIdentifier
	}

	// Convert to lowercase
	s = strings.ToLower(s)

	// Replace invalid characters with underscores
	var result strings.Builder
	result.Grow(len(s))
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
			result.WriteRune(r)
		} else {
			result.WriteRune('_')
		}
	}

	// Collapse multiple underscores and trim
	sanitized := result.String()
	for strings.Contains(sanitized, "__") {
		sanitized = strings.ReplaceAll(sanitized, "__", "_")
	}
	sanitized = strings.Trim(sanitized, "_")

	// Handle empty result
	if sanitized == "" {
		return DefaultIdentifier
	}

	// Truncate with hash suffix if too long
	if len(sanitized) > MaxIdentifierLength {
		sanitized = truncateWithHash(sanitized)
	}

	return sanitized
}

// truncateWithHash truncates a string to fit within MaxIdentifierLength,
// appending a hash suffix to preserve uniqueness.
//
// Format: <truncated>_<8-char-hash>
// Example: "very_long_identifier..." -> "very_long_iden_a1b2c3d4"
func truncateWithHash(s string) string {
	// Calculate hash of original string
	hash := sha256.Sum256([]byte(s))
	hashSuffix := "_" + hex.EncodeToString(hash[:])[:8]

	// Truncate to make room for hash suffix
	maxBase := MaxIdentifierLength - HashSuffixLength
	truncated := s[:maxBase]

	// Clean up trailing underscore if present
	truncated = strings.TrimRight(truncated, "_")

	return truncated + hashSuffix
}

// CollectionName builds a collection name from tenant and project components.
//
// Format: {sanitized_tenant}_{sanitized_project}_{suffix}
// Example: CollectionName("github.com/user", "my-project", "codebase")
//
//	-> "github_com_user_my_project_codebase"
//
// The result is guaranteed to be valid for vector store collection names.
func CollectionName(tenant, project, suffix string) string {
	sanitizedTenant := Identifier(tenant)
	sanitizedProject := Identifier(project)

	var name string
	if suffix != "" {
		name = sanitizedTenant + "_" + sanitizedProject + "_" + suffix
	} else {
		name = sanitizedTenant + "_" + sanitizedProject
	}

	// Final length check on combined name
	if len(name) > MaxIdentifierLength {
		name = truncateWithHash(name)
	}

	return name
}
