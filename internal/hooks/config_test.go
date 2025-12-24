package hooks

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	configJSON := `{
		"hooks": {
			"auto_checkpoint_on_clear": true,
			"auto_resume_on_start": true,
			"checkpoint_threshold_percent": 75,
			"verify_before_clear": false
		}
	}`
	err := os.WriteFile(configPath, []byte(configJSON), 0600)
	if err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}
	config, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	if !config.AutoCheckpointOnClear {
		t.Error("Expected AutoCheckpointOnClear to be true")
	}
	if config.CheckpointThreshold != 75 {
		t.Errorf("Expected CheckpointThreshold 75, got %d", config.CheckpointThreshold)
	}
}

func TestLoadConfig_NotFound(t *testing.T) {
	config, err := LoadConfig("/nonexistent/config.json")
	if err != nil {
		t.Fatalf("LoadConfig should not error on missing file: %v", err)
	}
	if config == nil {
		t.Fatal("Expected default config, got nil")
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()
	if config == nil {
		t.Fatal("DefaultConfig returned nil")
	}
	if config.CheckpointThreshold != 70 {
		t.Errorf("Expected default threshold 70, got %d", config.CheckpointThreshold)
	}
}

func TestLoadConfigWithEnvOverride(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	configJSON := `{
		"hooks": {
			"auto_checkpoint_on_clear": false,
			"checkpoint_threshold_percent": 70
		}
	}`
	err := os.WriteFile(configPath, []byte(configJSON), 0600)
	if err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}
	os.Setenv("CONTEXTD_AUTO_CHECKPOINT_ON_CLEAR", "true")
	os.Setenv("CONTEXTD_CHECKPOINT_THRESHOLD", "80")
	defer os.Unsetenv("CONTEXTD_AUTO_CHECKPOINT_ON_CLEAR")
	defer os.Unsetenv("CONTEXTD_CHECKPOINT_THRESHOLD")
	config, err := LoadConfigWithEnvOverride(configPath)
	if err != nil {
		t.Fatalf("LoadConfigWithEnvOverride failed: %v", err)
	}
	if !config.AutoCheckpointOnClear {
		t.Error("Expected env override for AutoCheckpointOnClear")
	}
	if config.CheckpointThreshold != 80 {
		t.Errorf("Expected env override threshold 80, got %d", config.CheckpointThreshold)
	}
}

func TestConfigValidation_Integration(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	configJSON := `{"hooks": {"checkpoint_threshold_percent": 150}}`
	err := os.WriteFile(configPath, []byte(configJSON), 0600)
	if err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}
	_, err = LoadConfig(configPath)
	if err == nil {
		t.Error("Expected validation error for invalid threshold")
	}
}

func TestLoadConfig_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	err := os.WriteFile(configPath, []byte("{invalid json}"), 0600)
	if err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}
	_, err = LoadConfig(configPath)
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

func TestLoadConfig_MissingHooksSection(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	err := os.WriteFile(configPath, []byte("{}"), 0600)
	if err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}
	config, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Expected default config, got error: %v", err)
	}
	if config.CheckpointThreshold != 70 {
		t.Errorf("Expected default threshold 70, got %d", config.CheckpointThreshold)
	}
}

func TestLoadConfigWithEnvOverride_InvalidValues(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	configJSON := `{
		"hooks": {
			"auto_checkpoint_on_clear": false,
			"checkpoint_threshold_percent": 70
		}
	}`
	err := os.WriteFile(configPath, []byte(configJSON), 0600)
	if err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}
	os.Setenv("CONTEXTD_AUTO_CHECKPOINT_ON_CLEAR", "invalid")
	os.Setenv("CONTEXTD_CHECKPOINT_THRESHOLD", "invalid")
	defer os.Unsetenv("CONTEXTD_AUTO_CHECKPOINT_ON_CLEAR")
	defer os.Unsetenv("CONTEXTD_CHECKPOINT_THRESHOLD")
	config, err := LoadConfigWithEnvOverride(configPath)
	if err != nil {
		t.Fatalf("LoadConfigWithEnvOverride failed: %v", err)
	}
	if config.AutoCheckpointOnClear != false {
		t.Error("Expected file value to remain when env var invalid")
	}
	if config.CheckpointThreshold != 70 {
		t.Errorf("Expected file value 70, got %d", config.CheckpointThreshold)
	}
}

func TestLoadConfigWithEnvOverride_InvalidThreshold(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	configJSON := `{"hooks": {"checkpoint_threshold_percent": 70}}`
	err := os.WriteFile(configPath, []byte(configJSON), 0600)
	if err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	os.Setenv("CONTEXTD_CHECKPOINT_THRESHOLD", "150")
	defer os.Unsetenv("CONTEXTD_CHECKPOINT_THRESHOLD")

	_, err = LoadConfigWithEnvOverride(configPath)
	if err == nil {
		t.Error("Expected validation error for threshold > 100")
	}
	if err != nil && !contains(err.Error(), "checkpoint_threshold") {
		t.Errorf("Expected threshold validation error, got: %v", err)
	}
}

// Helper for error message checking
func contains(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
