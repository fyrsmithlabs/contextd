// Package main implements statusline commands for Claude Code integration.
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/fyrsmithlabs/contextd/internal/checkpoint"
	"github.com/fyrsmithlabs/contextd/internal/config"
	"github.com/fyrsmithlabs/contextd/internal/embeddings"
	"github.com/fyrsmithlabs/contextd/internal/vectorstore"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var (
	// statuslineInterval is the polling interval for periodic updates
	statuslineInterval time.Duration
	// statuslineOnce runs once and exits (for one-shot mode)
	statuslineOnce bool
	// statuslineDirect queries database directly without HTTP
	statuslineDirect bool
)

func init() {
	rootCmd.AddCommand(statuslineCmd)
	statuslineCmd.AddCommand(statuslineRunCmd)
	statuslineCmd.AddCommand(statuslineInstallCmd)
	statuslineCmd.AddCommand(statuslineUninstallCmd)
	statuslineCmd.AddCommand(statuslineTestCmd)

	statuslineRunCmd.Flags().DurationVar(&statuslineInterval, "interval", 5*time.Second, "polling interval")
	statuslineRunCmd.Flags().BoolVar(&statuslineOnce, "once", false, "run once and exit")
	statuslineRunCmd.Flags().BoolVar(&statuslineDirect, "direct", false, "query database directly (no HTTP server needed)")

	statuslineTestCmd.Flags().BoolVar(&statuslineDirect, "direct", false, "query database directly (no HTTP server needed)")
}

// statuslineCmd is the parent command for statusline operations
var statuslineCmd = &cobra.Command{
	Use:   "statusline",
	Short: "Manage Claude Code statusline integration",
	Long: `Manage the contextd statusline integration with Claude Code.

The statusline displays key metrics in Claude Code's status bar:
  - Service health indicator
  - Memory count
  - Checkpoint count
  - Context usage percentage
  - Last confidence score
  - Compression ratio

Examples:
  # Run statusline fetcher (for Claude Code)
  ctxd statusline run

  # Install statusline script
  ctxd statusline install

  # Test statusline output
  ctxd statusline test`,
}

// statuslineRunCmd runs the statusline fetcher
var statuslineRunCmd = &cobra.Command{
	Use:   "run",
	Short: "Run the statusline fetcher for Claude Code",
	Long: `Run the statusline fetcher that polls the contextd server and outputs
formatted status for Claude Code's statusline.

In normal mode, reads JSON commands from stdin and outputs formatted status.
With --once, runs a single status check and exits.

Examples:
  # Run as Claude Code statusline script
  ctxd statusline run

  # Run once for testing
  ctxd statusline run --once`,
	RunE: runStatuslineRun,
}

// statuslineInstallCmd installs the statusline configuration
var statuslineInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install statusline configuration for Claude Code",
	Long: `Install the statusline script path into Claude Code settings.

This updates the Claude Code settings to use ctxd as the statusline script.

Examples:
  ctxd statusline install`,
	RunE: runStatuslineInstall,
}

// statuslineUninstallCmd removes the statusline configuration
var statuslineUninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Remove statusline configuration from Claude Code",
	Long: `Remove the statusline script configuration from Claude Code settings.

Examples:
  ctxd statusline uninstall`,
	RunE: runStatuslineUninstall,
}

// statuslineTestCmd tests the statusline output
var statuslineTestCmd = &cobra.Command{
	Use:   "test",
	Short: "Test statusline output without installing",
	Long: `Test the statusline output by fetching status and displaying the formatted line.

Examples:
  ctxd statusline test`,
	RunE: runStatuslineTest,
}

// StatusResponse matches internal/http/server.go StatusResponse
type StatusResponse struct {
	Status      string             `json:"status"`
	Services    map[string]string  `json:"services"`
	Counts      StatusCounts       `json:"counts"`
	Context     *ContextStatus     `json:"context,omitempty"`
	Compression *CompressionStatus `json:"compression,omitempty"`
	Memory      *MemoryStatus      `json:"memory,omitempty"`
}

// StatusCounts contains count information
type StatusCounts struct {
	Checkpoints int `json:"checkpoints"`
	Memories    int `json:"memories"`
}

// ContextStatus contains context usage information
type ContextStatus struct {
	UsagePercent     int  `json:"usage_percent"`
	ThresholdWarning bool `json:"threshold_warning"`
}

// CompressionStatus contains compression metrics
type CompressionStatus struct {
	LastRatio       float64 `json:"last_ratio"`
	LastQuality     float64 `json:"last_quality"`
	OperationsTotal int64   `json:"operations_total"`
}

// MemoryStatus contains memory/reasoning bank metrics
type MemoryStatus struct {
	LastConfidence float64 `json:"last_confidence"`
}

// runStatuslineRun handles the statusline run command
func runStatuslineRun(cmd *cobra.Command, args []string) error {
	if statuslineOnce {
		return outputStatusline()
	}

	// Claude Code statusline protocol: read JSON from stdin, output formatted line
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		// Claude Code sends JSON commands, we respond with formatted statusline
		if err := outputStatusline(); err != nil {
			// Output error indicator but don't exit
			fmt.Println("\033[31m\u26a0\ufe0f contextd error\033[0m")
		}
	}

	return scanner.Err()
}

// outputStatusline fetches status and outputs formatted line
func outputStatusline() error {
	var status *StatusResponse
	var err error

	if statuslineDirect {
		status, err = fetchStatusDirect()
	} else {
		status, err = fetchStatusHTTP()
	}

	if err != nil {
		return err
	}

	line := formatStatusline(status)
	fmt.Println(line)
	return nil
}

// fetchStatusHTTP fetches status from the contextd HTTP server
func fetchStatusHTTP() (*StatusResponse, error) {
	url := fmt.Sprintf("%s/api/v1/status", serverURL)

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server returned status %d: %s", resp.StatusCode, string(body))
	}

	var status StatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &status, nil
}

// fetchStatusDirect queries the database directly without HTTP
func fetchStatusDirect() (*StatusResponse, error) {
	ctx := context.Background()

	// Load config
	cfg := config.Load()

	// Create a silent logger for statusline (no output)
	logger := zap.NewNop()

	// Initialize embedder
	embedder, err := embeddings.NewProvider(embeddings.ProviderConfig{
		Provider: cfg.Embeddings.Provider,
		Model:    cfg.Embeddings.Model,
		CacheDir: cfg.Embeddings.CacheDir,
		BaseURL:  cfg.Embeddings.BaseURL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create embedder: %w", err)
	}

	// Initialize vectorstore
	store, err := vectorstore.NewStore(cfg, embedder, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create vectorstore: %w", err)
	}
	defer store.Close()

	// Initialize checkpoint service
	checkpointSvc, err := checkpoint.NewService(nil, store, logger) // nil config uses defaults
	if err != nil {
		return nil, fmt.Errorf("failed to create checkpoint service: %w", err)
	}

	// Build status response
	status := &StatusResponse{
		Status:   "ok",
		Services: make(map[string]string),
		Counts:   StatusCounts{},
	}

	// All services are "ok" in direct mode (we have access)
	status.Services["checkpoint"] = "ok"
	status.Services["memory"] = "ok"
	status.Services["vectorstore"] = "ok"

	// Get checkpoint count
	checkpoints, err := checkpointSvc.List(ctx, &checkpoint.ListRequest{Limit: 1000})
	if err == nil {
		status.Counts.Checkpoints = len(checkpoints)
	}

	// Memory count would require project_id, leave as 0 for now
	status.Counts.Memories = 0

	return status, nil
}

// formatStatusline formats the status response as a statusline string
func formatStatusline(status *StatusResponse) string {
	var parts []string

	// Service health indicator
	healthIcon := getHealthIcon(status)
	parts = append(parts, healthIcon)

	// Memory count
	parts = append(parts, fmt.Sprintf("\U0001f9e0%d", status.Counts.Memories))

	// Checkpoint count
	parts = append(parts, fmt.Sprintf("\U0001f4be%d", status.Counts.Checkpoints))

	// Context usage (if available)
	if status.Context != nil {
		contextIcon := "\U0001f4ca"
		if status.Context.ThresholdWarning {
			contextIcon = "\033[33m\U0001f4ca\033[0m" // Yellow warning
		}
		parts = append(parts, fmt.Sprintf("%s%d%%", contextIcon, status.Context.UsagePercent))
	}

	// Confidence (if available)
	if status.Memory != nil && status.Memory.LastConfidence > 0 {
		parts = append(parts, fmt.Sprintf("C:%.2f", status.Memory.LastConfidence))
	}

	// Compression ratio (if available)
	if status.Compression != nil && status.Compression.LastRatio > 0 {
		parts = append(parts, fmt.Sprintf("F:%.1fx", status.Compression.LastRatio))
	}

	return strings.Join(parts, " \u2502 ")
}

// getHealthIcon returns the appropriate health indicator icon
func getHealthIcon(status *StatusResponse) string {
	if status.Status != "ok" {
		return "\033[31m\U0001f534\033[0m" // Red circle
	}

	// Check if any service is unavailable
	for _, svcStatus := range status.Services {
		if svcStatus == "unavailable" {
			return "\033[33m\U0001f7e1\033[0m" // Yellow circle
		}
	}

	return "\033[32m\U0001f7e2\033[0m" // Green circle
}

// runStatuslineInstall handles the statusline install command
func runStatuslineInstall(cmd *cobra.Command, args []string) error {
	// Find ctxd binary path
	ctxdPath, err := exec.LookPath("ctxd")
	if err != nil {
		// Try to use the current executable
		ctxdPath, err = os.Executable()
		if err != nil {
			return fmt.Errorf("could not find ctxd binary: %w", err)
		}
	}

	// Get absolute path
	ctxdPath, err = filepath.Abs(ctxdPath)
	if err != nil {
		return fmt.Errorf("could not resolve ctxd path: %w", err)
	}

	// Build the statusline command - use --direct by default (no HTTP server needed)
	var statuslineScript string
	if serverURL != "http://localhost:9090" {
		// User specified a custom server, use HTTP mode
		statuslineScript = fmt.Sprintf("%s statusline run --server %s", ctxdPath, serverURL)
	} else {
		// Default to direct mode (queries database directly)
		statuslineScript = fmt.Sprintf("%s statusline run --direct", ctxdPath)
	}

	// Get Claude Code settings path
	settingsPath := getClaudeSettingsPath()

	// Read existing settings
	var settings map[string]interface{}
	if data, err := os.ReadFile(settingsPath); err == nil {
		if err := json.Unmarshal(data, &settings); err != nil {
			settings = make(map[string]interface{})
		}
	} else {
		settings = make(map[string]interface{})
	}

	// Check for existing statusline and append if needed
	existingStatusLine := ""

	// Handle both string and object formats for statusLine
	if existing, ok := settings["statusLine"].(string); ok {
		existingStatusLine = existing
	} else if statusLineObj, ok := settings["statusLine"].(map[string]interface{}); ok {
		if cmd, ok := statusLineObj["command"].(string); ok {
			existingStatusLine = cmd
		}
	}

	// Auto-detect common statusline script locations if no statusLine configured
	if existingStatusLine == "" {
		homeDir, _ := os.UserHomeDir()
		commonPaths := []string{
			filepath.Join(homeDir, ".claude", "statusline.sh"),
			filepath.Join(homeDir, ".claude", "statusline"),
			filepath.Join(homeDir, ".config", "claude", "statusline.sh"),
		}
		for _, path := range commonPaths {
			if _, err := os.Stat(path); err == nil {
				existingStatusLine = path
				fmt.Printf("Auto-detected existing statusline script: %s\n", path)
				break
			}
		}
	}

	// Show current statusline before modifying
	if existingStatusLine != "" {
		fmt.Printf("Current statusline: %s\n", existingStatusLine)
	}

	// If there's an existing statusline that doesn't contain ctxd, append our script
	if existingStatusLine != "" && !strings.Contains(existingStatusLine, "ctxd") {
		// Create a combined script that runs both
		statuslineScript = fmt.Sprintf("%s; echo -n ' │ '; %s", existingStatusLine, statuslineScript)
	} else if existingStatusLine != "" && strings.Contains(existingStatusLine, "ctxd") {
		// Replace existing ctxd command but preserve any prefix
		// Find where ctxd starts and replace from there
		if idx := strings.Index(existingStatusLine, "ctxd"); idx > 0 {
			prefix := existingStatusLine[:idx]
			// Check if there's a semicolon before ctxd (another command)
			if lastSemi := strings.LastIndex(prefix, ";"); lastSemi >= 0 {
				prefix = strings.TrimSpace(existingStatusLine[:lastSemi+1])
				statuslineScript = fmt.Sprintf("%s %s", prefix, statuslineScript)
			}
		}
	}

	// Update statusline setting
	settings["statusLine"] = statuslineScript

	// Write settings back
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0755); err != nil {
		return fmt.Errorf("failed to create settings directory: %w", err)
	}

	if err := os.WriteFile(settingsPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write settings: %w", err)
	}

	fmt.Printf("Installed statusline script: %s\n", statuslineScript)
	fmt.Printf("Settings updated: %s\n", settingsPath)
	fmt.Println("\nRestart Claude Code to apply changes.")

	return nil
}

// runStatuslineUninstall handles the statusline uninstall command
func runStatuslineUninstall(cmd *cobra.Command, args []string) error {
	settingsPath := getClaudeSettingsPath()

	// Read existing settings
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("No Claude Code settings found, nothing to uninstall.")
			return nil
		}
		return fmt.Errorf("failed to read settings: %w", err)
	}

	var settings map[string]interface{}
	if err := json.Unmarshal(data, &settings); err != nil {
		return fmt.Errorf("failed to parse settings: %w", err)
	}

	// Remove statusline setting
	if _, exists := settings["statusLine"]; !exists {
		fmt.Println("No statusline configuration found, nothing to uninstall.")
		return nil
	}

	delete(settings, "statusLine")

	// Write settings back
	data, err = json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	if err := os.WriteFile(settingsPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write settings: %w", err)
	}

	fmt.Printf("Removed statusline configuration from: %s\n", settingsPath)
	fmt.Println("\nRestart Claude Code to apply changes.")

	return nil
}

// runStatuslineTest handles the statusline test command
func runStatuslineTest(cmd *cobra.Command, args []string) error {
	var status *StatusResponse
	var err error

	if statuslineDirect {
		status, err = fetchStatusDirect()
	} else {
		status, err = fetchStatusHTTP()
	}
	if err != nil {
		return fmt.Errorf("failed to fetch status: %w", err)
	}

	// Show raw status
	fmt.Println("=== Raw Status ===")
	fmt.Printf("Status: %s\n", status.Status)
	fmt.Printf("Services: %v\n", status.Services)
	fmt.Printf("Counts: memories=%d, checkpoints=%d\n", status.Counts.Memories, status.Counts.Checkpoints)

	if status.Context != nil {
		fmt.Printf("Context: usage=%d%%, warning=%v\n", status.Context.UsagePercent, status.Context.ThresholdWarning)
	}
	if status.Memory != nil {
		fmt.Printf("Memory: lastConfidence=%.2f\n", status.Memory.LastConfidence)
	}
	if status.Compression != nil {
		fmt.Printf("Compression: ratio=%.2f, quality=%.2f, ops=%d\n",
			status.Compression.LastRatio, status.Compression.LastQuality, status.Compression.OperationsTotal)
	}

	// Show formatted line
	fmt.Println("\n=== Formatted Statusline ===")
	fmt.Println(formatStatusline(status))

	return nil
}

// getClaudeSettingsPath returns the path to Claude Code settings
func getClaudeSettingsPath() string {
	var configDir string

	switch runtime.GOOS {
	case "darwin":
		home, _ := os.UserHomeDir()
		configDir = filepath.Join(home, "Library", "Application Support", "Claude")
	case "linux":
		if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
			configDir = filepath.Join(xdg, "claude")
		} else {
			home, _ := os.UserHomeDir()
			configDir = filepath.Join(home, ".config", "claude")
		}
	case "windows":
		configDir = filepath.Join(os.Getenv("APPDATA"), "Claude")
	default:
		home, _ := os.UserHomeDir()
		configDir = filepath.Join(home, ".claude")
	}

	return filepath.Join(configDir, "settings.json")
}
