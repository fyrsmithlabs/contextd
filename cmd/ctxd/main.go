// Package main implements the ctxd CLI for manual operations against contextd HTTP server.
package main

import (
	"bytes"
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
It provides commands for scrubbing secrets and checking server health.`,
	Version: version,
}

func init() {
	rootCmd.PersistentFlags().StringVar(&serverURL, "server", "http://localhost:9090", "contextd server URL")
	rootCmd.AddCommand(scrubCmd)
	rootCmd.AddCommand(healthCmd)
}

// scrubCmd scrubs secrets from files or stdin
var scrubCmd = &cobra.Command{
	Use:   "scrub [file]",
	Short: "Scrub secrets from a file or stdin",
	Long: `Scrub secrets from a file or stdin using the contextd server.

Examples:
  # Scrub a file
  ctxd scrub .env

  # Scrub from stdin
  cat output.log | ctxd scrub -

  # Use a different server
  ctxd scrub --server http://localhost:8080 .env`,
	Args: cobra.MaximumNArgs(1),
	RunE: runScrub,
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

// ScrubRequest matches internal/http/server.go ScrubRequest
type ScrubRequest struct {
	Content string `json:"content"`
}

// ScrubResponse matches internal/http/server.go ScrubResponse
type ScrubResponse struct {
	Content       string `json:"content"`
	FindingsCount int    `json:"findings_count"`
}

// HealthResponse matches internal/http/server.go HealthResponse
type HealthResponse struct {
	Status string `json:"status"`
}

// runScrub handles the scrub command
func runScrub(cmd *cobra.Command, args []string) error {
	var content []byte
	var err error

	// Read input from file or stdin
	if len(args) == 0 || args[0] == "-" {
		// Read from stdin
		content, err = io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("failed to read from stdin: %w", err)
		}
	} else {
		// Read from file
		content, err = os.ReadFile(args[0])
		if err != nil {
			return fmt.Errorf("failed to read file %s: %w", args[0], err)
		}
	}

	if len(content) == 0 {
		return fmt.Errorf("no content to scrub")
	}

	// Prepare request
	reqBody := ScrubRequest{
		Content: string(content),
	}

	reqJSON, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	// Make HTTP request
	url := fmt.Sprintf("%s/api/v1/scrub", serverURL)
	httpReq, err := http.NewRequest("POST", url, bytes.NewReader(reqJSON))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send request to %s: %w", url, err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return fmt.Errorf("server returned status %d (failed to read response body: %w)", resp.StatusCode, readErr)
		}
		return fmt.Errorf("server returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var scrubResp ScrubResponse
	if err := json.NewDecoder(resp.Body).Decode(&scrubResp); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	// Output scrubbed content to stdout
	fmt.Print(scrubResp.Content)

	// If findings were made, log to stderr
	if scrubResp.FindingsCount > 0 {
		fmt.Fprintf(os.Stderr, "\n[ctxd] Scrubbed %d secret(s)\n", scrubResp.FindingsCount)
	}

	return nil
}

// runHealth handles the health command
func runHealth(cmd *cobra.Command, args []string) error {
	url := fmt.Sprintf("%s/health", serverURL)

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to connect to %s: %v\n", url, err)
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
