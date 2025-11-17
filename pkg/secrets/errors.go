// Package secrets provides secret detection and redaction using the Gitleaks SDK.
package secrets

import "errors"

var (
	// ErrInvalidRegex indicates a regex pattern failed to compile.
	ErrInvalidRegex = errors.New("invalid regex pattern")

	// ErrInvalidTOML indicates a TOML file could not be parsed.
	ErrInvalidTOML = errors.New("invalid TOML format")

	// ErrAllowlistNotFound indicates an allowlist file was not found.
	ErrAllowlistNotFound = errors.New("allowlist file not found")
)
