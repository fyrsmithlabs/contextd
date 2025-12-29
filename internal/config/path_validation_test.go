package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateConfigPath_RejectsPathTraversal(t *testing.T) {
	// Test that path traversal attempts are rejected
	tests := []struct {
		name string
		path string
	}{
		{"double dot escape", "/etc/contextd../etc/passwd"},
		{"multiple escapes", "~/.config/contextd/../../../../etc/passwd"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfigPath(tt.path)
			if err == nil {
				t.Errorf("Expected error for path traversal attempt: %s", tt.path)
			}
		})
	}
}

func TestValidateConfigPath_AllowsValidPaths(t *testing.T) {
	home := os.Getenv("HOME")
	if home == "" {
		home = "/tmp"
		os.Setenv("HOME", home)
		defer os.Unsetenv("HOME")
	}

	validPaths := []string{
		filepath.Join(home, ".config", "contextd", "config.yaml"),
		filepath.Join(home, ".config", "contextd", "subdir", "config.yaml"),
		"/etc/contextd/config.yaml",
		"/etc/contextd/production/config.yaml",
	}

	for _, path := range validPaths {
		t.Run(path, func(t *testing.T) {
			err := validateConfigPath(path)
			if err != nil {
				t.Errorf("Valid path rejected: %s, error: %v", path, err)
			}
		})
	}
}

func TestValidateConfigPath_RejectsOutsideAllowedDirs(t *testing.T) {
	invalidPaths := []string{
		"/etc/passwd",
		"/tmp/config.yaml",
		"/var/lib/contextd/config.yaml",
	}

	for _, path := range invalidPaths {
		t.Run(path, func(t *testing.T) {
			err := validateConfigPath(path)
			if err == nil {
				t.Errorf("Path outside allowed directories should be rejected: %s", path)
			}
		})
	}
}

func TestValidateConfigPath_HandlesNonExistentFiles(t *testing.T) {
	home := os.Getenv("HOME")
	if home == "" {
		home = "/tmp"
		os.Setenv("HOME", home)
		defer os.Unsetenv("HOME")
	}

	// Non-existent file in allowed directory should pass validation
	nonExistent := filepath.Join(home, ".config", "contextd", "nonexistent.yaml")
	err := validateConfigPath(nonExistent)
	if err != nil {
		t.Errorf("Non-existent file in allowed directory should pass validation: %v", err)
	}
}
