// Package main implements the ctxd CLI for manual operations against contextd HTTP server.
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/spf13/cobra"
)

var (
	// serverURL is the base URL for the contextd HTTP server
	serverURL string
	// version information
	version = "dev"
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "ctxd",
	Short: "CLI for contextd HTTP server operations",
	Long: `ctxd is a command-line interface for interacting with the contextd HTTP server.

Available commands:
  health   Check contextd server health status

Use "ctxd [command] --help" for more information about a command.
Use --server to specify a custom server URL (default: http://localhost:9090).`,
	Version: version,
}

func init() {
	rootCmd.PersistentFlags().StringVar(&serverURL, "server", "http://localhost:9090", "contextd server URL")
	rootCmd.AddCommand(healthCmd)
}

// healthCmd checks server health
var healthCmd = &cobra.Command{
	Use:   "health",
	Short: "Check contextd server health",
	Long: `Check the health status of the contextd HTTP server.

Examples:
  # Check health
  ctxd health

  # Check health on a different server
  ctxd health --server http://localhost:8080`,
	RunE: runHealth,
}

// HealthResponse matches internal/http/server.go HealthResponse
type HealthResponse struct {
	Status string `json:"status"`
}

// runHealth handles the health command
func runHealth(cmd *cobra.Command, args []string) error {
	url := fmt.Sprintf("%s/health", serverURL)

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to connect to %s: %v\n\n", url, err)
		fmt.Fprintf(os.Stderr, "Hint: The contextd HTTP server is not running.\n")
		fmt.Fprintf(os.Stderr, "      Start contextd with HTTP enabled:\n")
		fmt.Fprintf(os.Stderr, "        contextd           (HTTP mode, default port 9090)\n")
		fmt.Fprintf(os.Stderr, "      Or if using MCP mode, check your Claude Code MCP server status.\n")
		fmt.Fprintln(os.Stderr)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return fmt.Errorf("server returned status %d (failed to read response body: %w)", resp.StatusCode, readErr)
		}
		return fmt.Errorf("server returned status %d: %s", resp.StatusCode, string(body))
	}

	var healthResp HealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&healthResp); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	fmt.Printf("Server Status: %s\n", healthResp.Status)
	fmt.Printf("Server URL: %s\n", serverURL)

	return nil
}
