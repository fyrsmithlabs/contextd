package config

import (
	"os"
	"testing"
)

func TestProductionConfig_Defaults(t *testing.T) {
	// Clean up environment on test completion
	defer os.Unsetenv("CONTEXTD_PRODUCTION_MODE")
	defer os.Unsetenv("CONTEXTD_LOCAL_MODE")
	// Ensure clean starting state
	os.Unsetenv("CONTEXTD_PRODUCTION_MODE")
	os.Unsetenv("CONTEXTD_LOCAL_MODE")

	cfg := Load()

	if cfg.Production.Enabled {
		t.Error("Production.Enabled = true, want false (disabled by default)")
	}
}

func TestProductionConfig_EnabledViaEnv(t *testing.T) {
	defer os.Unsetenv("CONTEXTD_PRODUCTION_MODE")
	os.Setenv("CONTEXTD_PRODUCTION_MODE", "1")

	cfg := Load()

	if !cfg.Production.Enabled {
		t.Error("Production.Enabled = false, want true when CONTEXTD_PRODUCTION_MODE=1")
	}
}
func TestProductionConfig_Validate_BlocksNoIsolation(t *testing.T) {
	cfg := &ProductionConfig{
		Enabled:          true,
		AllowNoIsolation: true,
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("Expected error when AllowNoIsolation=true in production, got nil")
	}
	if err != nil && err.Error() != "SECURITY: NoIsolation mode cannot be enabled in production" {
		t.Errorf("Wrong error message: %v", err)
	}
}

func TestProductionConfig_Validate_RequiresAuth(t *testing.T) {
	cfg := &ProductionConfig{
		Enabled:                  true,
		RequireAuthentication:    true,
		AuthenticationConfigured: false,
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("Expected error when RequireAuthentication=true but not configured, got nil")
	}
}

func TestProductionConfig_Validate_PassesWhenValid(t *testing.T) {
	cfg := &ProductionConfig{
		Enabled:                  true,
		RequireAuthentication:    true,
		AuthenticationConfigured: true,
		RequireTLS:               true,
		AllowNoIsolation:         false,
	}

	err := cfg.Validate()
	if err != nil {
		t.Errorf("Expected no error for valid production config, got: %v", err)
	}
}

func TestProductionConfig_Validate_PassesWhenDisabled(t *testing.T) {
	cfg := &ProductionConfig{
		Enabled:          false,
		AllowNoIsolation: true, // Should be ignored when disabled
	}

	err := cfg.Validate()
	if err != nil {
		t.Errorf("Expected no error when production disabled, got: %v", err)
	}
}

func TestProductionConfig_LocalModeBypassesAuthAndTLS(t *testing.T) {
	defer os.Unsetenv("CONTEXTD_PRODUCTION_MODE")
	defer os.Unsetenv("CONTEXTD_LOCAL_MODE")
	
	os.Setenv("CONTEXTD_PRODUCTION_MODE", "1")
	os.Setenv("CONTEXTD_LOCAL_MODE", "1")

	cfg := Load()

	if !cfg.Production.Enabled {
		t.Error("Production.Enabled should be true when CONTEXTD_PRODUCTION_MODE=1")
	}
	if !cfg.Production.LocalModeAcknowledged {
		t.Error("LocalModeAcknowledged should be true when CONTEXTD_LOCAL_MODE=1")
	}
	if cfg.Production.RequireAuthentication {
		t.Error("RequireAuthentication should be false when LocalMode is enabled")
	}
	if cfg.Production.RequireTLS {
		t.Error("RequireTLS should be false when LocalMode is enabled")
	}
}

func TestProductionConfig_LoadWithFile_PreservesYAMLConfig(t *testing.T) {
	defer os.Unsetenv("CONTEXTD_PRODUCTION_MODE")
	defer os.Unsetenv("CONTEXTD_LOCAL_MODE")
	
	// Set HOME for test
	home := os.Getenv("HOME")
	if home == "" {
		home = "/tmp"
		os.Setenv("HOME", home)
		defer os.Unsetenv("HOME")
	}
	tmpDir := home + "/.config/contextd"
	os.MkdirAll(tmpDir, 0700)
	configPath := tmpDir + "/test_config.yaml"
	defer os.Remove(configPath)
	
	yamlContent := `production:
  enabled: true
  require_authentication: false
  require_tls: false
  allow_no_isolation: false
`
	
	if err := os.WriteFile(configPath, []byte(yamlContent), 0600); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	cfg, err := LoadWithFile(configPath)
	if err != nil {
		t.Fatalf("LoadWithFile failed: %v", err)
	}

	if !cfg.Production.Enabled {
		t.Error("Production.Enabled should be true from YAML")
	}
	if cfg.Production.RequireAuthentication {
		t.Error("RequireAuthentication should be false as set in YAML, not overridden")
	}
	if cfg.Production.RequireTLS {
		t.Error("RequireTLS should be false as set in YAML, not overridden")
	}
}

func TestProductionConfig_EnvOverridesYAML(t *testing.T) {
	defer os.Unsetenv("CONTEXTD_PRODUCTION_MODE")
	defer os.Unsetenv("CONTEXTD_LOCAL_MODE")
	
	// Set HOME for test
	home := os.Getenv("HOME")
	if home == "" {
		home = "/tmp"
		os.Setenv("HOME", home)
		defer os.Unsetenv("HOME")
	}
	tmpDir := home + "/.config/contextd"
	os.MkdirAll(tmpDir, 0700)
	configPath := tmpDir + "/test_config2.yaml"
	defer os.Remove(configPath)
	
	yamlContent := `production:
  enabled: false
`
	
	if err := os.WriteFile(configPath, []byte(yamlContent), 0600); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// But enable via environment variable
	defer os.Unsetenv("CONTEXTD_PRODUCTION_MODE")
	os.Setenv("CONTEXTD_PRODUCTION_MODE", "1")
	os.Setenv("CONTEXTD_LOCAL_MODE", "1")
	
	cfg, err := LoadWithFile(configPath)
	if err != nil {
		t.Fatalf("LoadWithFile failed: %v", err)
	}

	// Environment should override YAML
	if !cfg.Production.Enabled {
		t.Error("Production.Enabled should be true from environment, overriding YAML")
	}
}
