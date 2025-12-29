package hooks

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
)

// ConfigFile represents the structure of the config file
type ConfigFile struct {
	Hooks *Config `json:"hooks"`
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		AutoCheckpointOnClear: false, // Prompt by default (false = prompt)
		AutoResumeOnStart:     true,  // Enable auto-resume by default
		CheckpointThreshold:   70,    // 70% context threshold
		VerifyBeforeClear:     true,  // Verify by default for safety
	}
}

// LoadConfig loads configuration from a JSON file
// Returns default config if file doesn't exist
func LoadConfig(path string) (*Config, error) {
	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		// Return default config if file doesn't exist
		return DefaultConfig(), nil
	}

	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse JSON
	var configFile ConfigFile
	if err := json.Unmarshal(data, &configFile); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Use default if hooks section missing
	if configFile.Hooks == nil {
		return DefaultConfig(), nil
	}

	// Validate config
	if err := configFile.Hooks.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return configFile.Hooks, nil
}

// LoadConfigWithEnvOverride loads config from file and applies environment variable overrides
func LoadConfigWithEnvOverride(path string) (*Config, error) {
	// Load base config
	config, err := LoadConfig(path)
	if err != nil {
		return nil, err
	}

	// Apply environment overrides
	if val := os.Getenv("CONTEXTD_AUTO_CHECKPOINT_ON_CLEAR"); val != "" {
		if b, err := strconv.ParseBool(val); err == nil {
			config.AutoCheckpointOnClear = b
		}
	}

	if val := os.Getenv("CONTEXTD_AUTO_RESUME_ON_START"); val != "" {
		if b, err := strconv.ParseBool(val); err == nil {
			config.AutoResumeOnStart = b
		}
	}

	if val := os.Getenv("CONTEXTD_CHECKPOINT_THRESHOLD"); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			config.CheckpointThreshold = i
		}
	}

	if val := os.Getenv("CONTEXTD_VERIFY_BEFORE_CLEAR"); val != "" {
		if b, err := strconv.ParseBool(val); err == nil {
			config.VerifyBeforeClear = b
		}
	}

	// Validate after env overrides
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config after env override: %w", err)
	}

	return config, nil
}
