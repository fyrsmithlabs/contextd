package secrets

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/BurntSushi/toml"
)

// Allowlist contains path and content regex patterns to exclude from secret detection.
type Allowlist struct {
	Paths   []string // File path regex patterns to ignore
	Regexes []string // Content regex patterns to ignore
}

// LoadAllowlists loads and merges project and user allowlists using union (OR) logic.
// Missing files are silently ignored. Invalid TOML or regex patterns return errors.
//
// projectPath: Directory containing .gitleaks.toml (empty string to skip)
// userPath: Full path to user allowlist.toml file (empty string to skip)
func LoadAllowlists(projectPath, userPath string) (*Allowlist, error) {
	merged := &Allowlist{
		Paths:   []string{},
		Regexes: []string{},
	}

	// Load project allowlist
	if projectPath != "" {
		projectFile := filepath.Join(projectPath, ".gitleaks.toml")
		if project, err := loadTOML(projectFile); err != nil {
			// Only return error if file exists but is invalid
			if !os.IsNotExist(err) {
				return nil, err
			}
			// File doesn't exist - OK, skip it
		} else {
			merged.Paths = append(merged.Paths, project.Paths...)
			merged.Regexes = append(merged.Regexes, project.Regexes...)
		}
	}

	// Load user allowlist
	if userPath != "" {
		if user, err := loadTOML(userPath); err != nil {
			// Only return error if file exists but is invalid
			if !os.IsNotExist(err) {
				return nil, err
			}
			// File doesn't exist - OK, skip it
		} else {
			merged.Paths = append(merged.Paths, user.Paths...)
			merged.Regexes = append(merged.Regexes, user.Regexes...)
		}
	}

	return merged, nil
}

// loadTOML loads and validates a single allowlist file.
func loadTOML(path string) (*Allowlist, error) {
	var config struct {
		Allowlist struct {
			Paths   []string
			Regexes []string
		}
	}

	// Check if file exists
	if _, err := os.Stat(path); err != nil {
		return nil, err // os.IsNotExist can identify this
	}

	// Parse TOML
	if _, err := toml.DecodeFile(path, &config); err != nil {
		return nil, fmt.Errorf("%w: %s: %v", ErrInvalidTOML, path, err)
	}

	// Validate path regex patterns (fail-fast)
	for _, pattern := range config.Allowlist.Paths {
		if _, err := regexp.Compile(pattern); err != nil {
			return nil, fmt.Errorf("%w: invalid path pattern '%s' in %s: %v",
				ErrInvalidRegex, pattern, path, err)
		}
	}

	// Validate content regex patterns (fail-fast)
	for _, pattern := range config.Allowlist.Regexes {
		if _, err := regexp.Compile(pattern); err != nil {
			return nil, fmt.Errorf("%w: invalid content pattern '%s' in %s: %v",
				ErrInvalidRegex, pattern, path, err)
		}
	}

	return &Allowlist{
		Paths:   config.Allowlist.Paths,
		Regexes: config.Allowlist.Regexes,
	}, nil
}
