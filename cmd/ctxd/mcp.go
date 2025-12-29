// Package main implements MCP server configuration commands.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(mcpCmd)
	mcpCmd.AddCommand(mcpInstallCmd)
	mcpCmd.AddCommand(mcpUninstallCmd)
	mcpCmd.AddCommand(mcpStatusCmd)
}

// mcpCmd is the parent command for MCP server configuration
var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Manage Claude Code MCP server configuration",
	Long: `Manage the contextd MCP server configuration in Claude Code settings.

Automatically configures Claude Code to use contextd as an MCP server.

Examples:
  # Install MCP server configuration
  ctxd mcp install

  # Check MCP server status
  ctxd mcp status

  # Uninstall MCP server configuration
  ctxd mcp uninstall`,
}

// mcpInstallCmd installs the MCP server configuration
var mcpInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install contextd as MCP server in Claude Code",
	Long: `Automatically configure Claude Code to use contextd as an MCP server.

This command:
- Detects your contextd installation (binary or Docker)
- Adds the MCP server configuration to Claude Code settings
- Validates the configuration
- Provides next steps

The configuration is idempotent - safe to run multiple times.

Examples:
  ctxd mcp install`,
	RunE: runMCPInstall,
}

// mcpUninstallCmd removes the MCP server configuration
var mcpUninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Remove contextd MCP server from Claude Code",
	Long: `Remove the contextd MCP server configuration from Claude Code settings.

Examples:
  ctxd mcp uninstall`,
	RunE: runMCPUninstall,
}

// mcpStatusCmd shows MCP server configuration status
var mcpStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check contextd MCP server configuration status",
	Long: `Check if contextd is configured as an MCP server in Claude Code.

Shows:
- Whether MCP server is configured
- Configuration details
- Server health status

Examples:
  ctxd mcp status`,
	RunE: runMCPStatus,
}

// runMCPInstall handles the mcp install command
func runMCPInstall(cmd *cobra.Command, args []string) error {
	fmt.Println("🚀 Installing contextd MCP server configuration...")
	fmt.Println()

	// Step 1: Detect installation type
	fmt.Println("📦 Detecting contextd installation...")
	installType, command, cmdArgs, err := detectContextdInstallation()
	if err != nil {
		return fmt.Errorf("failed to detect contextd installation: %w\n\nPlease install contextd first:\n  brew install fyrsmithlabs/tap/contextd\n  OR download from https://github.com/fyrsmithlabs/contextd/releases", err)
	}
	fmt.Printf("   Found: %s\n", installType)
	fmt.Println()

	// Step 2: Load existing settings
	settingsPath := getClaudeSettingsPath()
	fmt.Printf("📝 Updating Claude Code settings: %s\n", settingsPath)

	settings, err := loadMCPSettings(settingsPath)
	if err != nil {
		fmt.Printf("   Creating new settings file\n")
		settings = make(map[string]interface{})
	}

	// Step 3: Configure MCP server
	if settings["mcpServers"] == nil {
		settings["mcpServers"] = make(map[string]interface{})
	}

	mcpServers, ok := settings["mcpServers"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid mcpServers format in settings")
	}

	// Check if already configured
	if existing, exists := mcpServers["contextd"]; exists {
		fmt.Println("   ⚠️  contextd MCP server already configured")
		fmt.Printf("   Current config: %+v\n", existing)
		fmt.Println()
		fmt.Print("   Overwrite? (y/N): ")
		var response string
		_, _ = fmt.Scanln(&response)
		if strings.ToLower(response) != "y" {
			fmt.Println("\n✅ Keeping existing configuration")
			return nil
		}
	}

	// Add/update contextd MCP server config
	mcpServers["contextd"] = map[string]interface{}{
		"type":    "stdio",
		"command": command,
		"args":    cmdArgs,
	}

	// Step 4: Write settings back
	if err := saveMCPSettings(settingsPath, settings); err != nil {
		return fmt.Errorf("failed to save settings: %w", err)
	}

	fmt.Println("   ✅ MCP server configured")
	fmt.Println()

	// Step 5: Verify configuration
	fmt.Println("🔍 Verifying configuration...")
	if err := verifyMCPConfig(command, cmdArgs); err != nil {
		fmt.Printf("   ⚠️  Validation warning: %v\n", err)
		fmt.Println("   Configuration saved, but may need adjustment")
	} else {
		fmt.Println("   ✅ Configuration valid")
	}
	fmt.Println()

	// Step 6: Show next steps
	fmt.Println("🎉 Installation complete!")
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("  1. Restart Claude Code to load the new MCP server")
	fmt.Println("  2. Test with: `mcp__contextd__memory_search(project_id: \"test\", query: \"hello\")`")
	fmt.Println("  3. Run `ctxd mcp status` to verify the server is running")
	fmt.Println()

	return nil
}

// runMCPUninstall handles the mcp uninstall command
func runMCPUninstall(cmd *cobra.Command, args []string) error {
	settingsPath := getClaudeSettingsPath()

	settings, err := loadMCPSettings(settingsPath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("No Claude Code settings found, nothing to uninstall.")
			return nil
		}
		return fmt.Errorf("failed to load settings: %w", err)
	}

	mcpServers, ok := settings["mcpServers"].(map[string]interface{})
	if !ok || mcpServers["contextd"] == nil {
		fmt.Println("contextd MCP server not configured, nothing to uninstall.")
		return nil
	}

	delete(mcpServers, "contextd")

	if err := saveMCPSettings(settingsPath, settings); err != nil {
		return fmt.Errorf("failed to save settings: %w", err)
	}

	fmt.Printf("Removed contextd MCP server from: %s\n", settingsPath)
	fmt.Println("\nRestart Claude Code to apply changes.")

	return nil
}

// runMCPStatus handles the mcp status command
func runMCPStatus(cmd *cobra.Command, args []string) error {
	settingsPath := getClaudeSettingsPath()

	settings, err := loadMCPSettings(settingsPath)
	if err != nil {
		fmt.Printf("❌ Could not load Claude Code settings from: %s\n", settingsPath)
		if os.IsNotExist(err) {
			fmt.Println("   Settings file does not exist")
		}
		return nil
	}

	fmt.Println("📊 MCP Server Configuration Status")
	fmt.Println()

	mcpServers, ok := settings["mcpServers"].(map[string]interface{})
	if !ok {
		fmt.Println("❌ No MCP servers configured in Claude Code")
		return nil
	}

	contextdConfig, exists := mcpServers["contextd"]
	if !exists {
		fmt.Println("❌ contextd MCP server not configured")
		fmt.Println()
		fmt.Println("Run `ctxd mcp install` to configure it.")
		return nil
	}

	fmt.Println("✅ contextd MCP server is configured")
	fmt.Println()
	fmt.Println("Configuration:")

	configMap, ok := contextdConfig.(map[string]interface{})
	if ok {
		if serverType, ok := configMap["type"].(string); ok {
			fmt.Printf("  Type: %s\n", serverType)
		}
		if command, ok := configMap["command"].(string); ok {
			fmt.Printf("  Command: %s\n", command)
		}
		if args, ok := configMap["args"].([]interface{}); ok {
			fmt.Printf("  Args: %v\n", args)
		}
	}

	return nil
}

// detectContextdInstallation detects how contextd is installed
func detectContextdInstallation() (string, string, []string, error) {
	// Try to find contextd binary
	if path, err := exec.LookPath("contextd"); err == nil {
		return "Binary", path, []string{}, nil
	}

	// Check if Docker is available
	if _, err := exec.LookPath("docker"); err == nil {
		// Test if contextd image exists
		cmd := exec.Command("docker", "image", "inspect", "ghcr.io/fyrsmithlabs/contextd:latest")
		if err := cmd.Run(); err == nil {
			homeDir, _ := os.UserHomeDir()
			configPath := filepath.Join(homeDir, ".config", "contextd")
			return "Docker",
				"docker",
				[]string{
					"run",
					"-i",
					"--rm",
					"-v", fmt.Sprintf("%s:/root/.config/contextd", configPath),
					"ghcr.io/fyrsmithlabs/contextd:latest",
				},
				nil
		}
	}

	return "", "", nil, fmt.Errorf("contextd not found (tried binary and Docker)")
}

// loadMCPSettings loads Claude Code settings.json
func loadMCPSettings(path string) (map[string]interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var settings map[string]interface{}
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, fmt.Errorf("invalid JSON in settings file: %w", err)
	}

	return settings, nil
}

// saveMCPSettings saves Claude Code settings.json
func saveMCPSettings(path string, settings map[string]interface{}) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create settings directory: %w", err)
	}

	// Marshal with indentation
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	// Write with secure permissions
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write settings: %w", err)
	}

	return nil
}

// verifyMCPConfig verifies the MCP server configuration can run
func verifyMCPConfig(command string, args []string) error {
	// For binary installs, check if executable exists and is accessible
	if command != "docker" {
		if _, err := exec.LookPath(command); err != nil {
			return fmt.Errorf("command not found in PATH: %s", command)
		}
		// Try to run with --version to verify it works
		cmd := exec.Command(command, "--version")
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("command exists but failed to run: %w", err)
		}
	}

	return nil
}
