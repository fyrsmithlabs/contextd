package config

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// setupTestHome creates a temporary home directory for testing.
// Returns the home dir path and a cleanup function.
func setupTestHome(t *testing.T) (string, func()) {
	t.Helper()

	// Create temp dir for fake home
	tmpHome := t.TempDir()

	// Save original HOME
	originalHome := os.Getenv("HOME")

	// Set HOME to temp dir
	os.Setenv("HOME", tmpHome)

	// Return cleanup function
	cleanup := func() {
		if originalHome != "" {
			os.Setenv("HOME", originalHome)
		} else {
			os.Unsetenv("HOME")
		}
	}

	return tmpHome, cleanup
}

// TestLoadWithFile_ValidYAML tests loading configuration from a valid YAML file.
func TestLoadWithFile_ValidYAML(t *testing.T) {
	// Setup test home directory
	home, cleanup := setupTestHome(t)
	defer cleanup()

	// Create config directory in allowed location
	configDir := filepath.Join(home, ".config", "contextd")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	configPath := filepath.Join(configDir, "config.yaml")

	yamlContent := `server:
  http_port: 9090
  http_host: 127.0.0.1

observability:
  enable_telemetry: true
  service_name: contextd-test
`

	if err := os.WriteFile(configPath, []byte(yamlContent), 0600); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Test: Load configuration from YAML
	cfg, err := LoadWithFile(configPath)
	if err != nil {
		t.Fatalf("LoadWithFile() error = %v, want nil", err)
	}

	// Verify configuration values from YAML
	if cfg.Server.Port != 9090 {
		t.Errorf("Server.Port = %d, want 9090", cfg.Server.Port)
	}

	if cfg.Observability.ServiceName != "contextd-test" {
		t.Errorf("Observability.ServiceName = %q, want %q", cfg.Observability.ServiceName, "contextd-test")
	}

	if !cfg.Observability.EnableTelemetry {
		t.Error("Observability.EnableTelemetry = false, want true")
	}
}

// TestLoadWithFile_EnvironmentOverride tests that environment variables override YAML.
func TestLoadWithFile_EnvironmentOverride(t *testing.T) {
	// Setup test home directory
	home, cleanup := setupTestHome(t)
	defer cleanup()

	// Create config directory in allowed location
	configDir := filepath.Join(home, ".config", "contextd")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	configPath := filepath.Join(configDir, "config.yaml")

	yamlContent := `server:
  http_port: 9090
  shutdown_timeout: 10s

observability:
  enable_telemetry: false
  service_name: yaml-service
`

	if err := os.WriteFile(configPath, []byte(yamlContent), 0600); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Set environment variables (should override YAML)
	os.Setenv("SERVER_HTTP_PORT", "7777")
	os.Setenv("OBSERVABILITY_SERVICE_NAME", "env-service")
	defer os.Unsetenv("SERVER_HTTP_PORT")
	defer os.Unsetenv("OBSERVABILITY_SERVICE_NAME")

	// Load config
	cfg, err := LoadWithFile(configPath)
	if err != nil {
		t.Fatalf("LoadWithFile() error = %v, want nil", err)
	}

	// Verify environment variables override YAML
	if cfg.Server.Port != 7777 {
		t.Errorf("Server.Port = %d, want 7777 (from env override)", cfg.Server.Port)
	}

	if cfg.Observability.ServiceName != "env-service" {
		t.Errorf("Observability.ServiceName = %q, want %q (from env override)", cfg.Observability.ServiceName, "env-service")
	}
}

// TestLoadWithFile_DefaultPath tests using default config path.
func TestLoadWithFile_DefaultPath(t *testing.T) {
	// Create config directory
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("Cannot get home directory")
	}

	configDir := filepath.Join(home, ".config", "contextd")
	configPath := filepath.Join(configDir, "config.yaml")

	// Check if config file exists (real file from user)
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Skip("No config file at default path")
	}

	// Test: Load with empty path (should use default)
	cfg, err := LoadWithFile("")
	if err != nil {
		t.Fatalf("LoadWithFile(\"\") error = %v, want nil", err)
	}

	// Just verify it loaded without error and has valid port
	if cfg.Server.Port < 1 || cfg.Server.Port > 65535 {
		t.Errorf("Server.Port = %d, want valid port (1-65535)", cfg.Server.Port)
	}
}

// TestLoadWithFile_MissingFile tests handling of missing config file.
func TestLoadWithFile_MissingFile(t *testing.T) {
	// Setup test home directory
	home, cleanup := setupTestHome(t)
	defer cleanup()

	// Test with path in allowed directory (but file doesn't exist)
	configPath := filepath.Join(home, ".config", "contextd", "config.yaml")

	cfg, err := LoadWithFile(configPath)
	if err != nil {
		t.Fatalf("LoadWithFile() should not error on missing file, got: %v", err)
	}

	// Should have default values (need to set defaults in loader)
	if cfg == nil {
		t.Error("LoadWithFile() returned nil config for missing file")
	}
}

// TestLoadWithFile_InvalidYAML tests handling of malformed YAML.
func TestLoadWithFile_InvalidYAML(t *testing.T) {
	// Create temporary invalid YAML file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	invalidYAML := `server:
  http_port: not-a-number
  invalid syntax here
`

	if err := os.WriteFile(configPath, []byte(invalidYAML), 0600); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Test: Load invalid YAML (should return error)
	_, err := LoadWithFile(configPath)
	if err == nil {
		t.Error("LoadWithFile() should error on invalid YAML, got nil")
	}
}

// TestLoadWithFile_Validation tests configuration validation.
func TestLoadWithFile_Validation(t *testing.T) {
	// Create temporary YAML with invalid port
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	yamlContent := `server:
  http_port: 99999

observability:
  service_name: test
`

	if err := os.WriteFile(configPath, []byte(yamlContent), 0600); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Test: Load config with invalid port (should fail validation)
	_, err := LoadWithFile(configPath)
	if err == nil {
		t.Error("LoadWithFile() should error on invalid port, got nil")
	}
}

// TestLoadWithFile_PathTraversal tests path traversal attack prevention.
func TestLoadWithFile_PathTraversal(t *testing.T) {
	// Setup test home directory
	_, cleanup := setupTestHome(t)
	defer cleanup()

	// Test: Reject ../../../../etc/passwd
	_, err := LoadWithFile("../../../../etc/passwd")
	if err == nil {
		t.Error("Expected error for path traversal, got nil")
	}
	if !strings.Contains(err.Error(), "must be in ~/.config/contextd/ or /etc/contextd/") {
		t.Errorf("Expected path validation error, got: %v", err)
	}
}

// TestLoadWithFile_InsecurePermissions tests file permission enforcement.
func TestLoadWithFile_InsecurePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping permission test on Windows")
	}

	// Setup test home directory
	home, cleanup := setupTestHome(t)
	defer cleanup()

	// Create config directory in allowed location
	configDir := filepath.Join(home, ".config", "contextd")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	configPath := filepath.Join(configDir, "config.yaml")

	yamlContent := `server:
  http_port: 9090
`

	// Write with insecure permissions (0644 - world readable)
	if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	_, err := LoadWithFile(configPath)
	if err == nil {
		t.Error("Expected error for insecure permissions, got nil")
	}
	if !strings.Contains(err.Error(), "insecure") && !strings.Contains(err.Error(), "permissions") {
		t.Errorf("Expected 'insecure permissions' error, got: %v", err)
	}
}

// TestLoadWithFile_SecurePermissions tests that 0600 permissions are accepted.
func TestLoadWithFile_SecurePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping permission test on Windows")
	}

	// Setup test home directory
	home, cleanup := setupTestHome(t)
	defer cleanup()

	// Create config directory in allowed location
	configDir := filepath.Join(home, ".config", "contextd")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	configPath := filepath.Join(configDir, "config.yaml")

	yamlContent := `server:
  http_port: 9090
`

	// Write with secure permissions (0600)
	if err := os.WriteFile(configPath, []byte(yamlContent), 0600); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	cfg, err := LoadWithFile(configPath)
	if err != nil {
		t.Fatalf("LoadWithFile() should succeed with 0600 permissions, got error: %v", err)
	}

	if cfg.Server.Port != 9090 {
		t.Errorf("Server.Port = %d, want 9090", cfg.Server.Port)
	}
}

// TestLoadWithFile_FileTooLarge tests file size limit enforcement.
func TestLoadWithFile_FileTooLarge(t *testing.T) {
	// Setup test home directory
	home, cleanup := setupTestHome(t)
	defer cleanup()

	// Create config directory in allowed location
	configDir := filepath.Join(home, ".config", "contextd")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	configPath := filepath.Join(configDir, "config.yaml")

	// Create 2MB file (exceeds 1MB limit)
	largeContent := bytes.Repeat([]byte("# comment line\n"), 150000)
	if err := os.WriteFile(configPath, largeContent, 0600); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	_, err := LoadWithFile(configPath)
	if err == nil {
		t.Error("Expected error for large file, got nil")
	}
	if !strings.Contains(err.Error(), "too large") {
		t.Errorf("Expected 'too large' error, got: %v", err)
	}
}
