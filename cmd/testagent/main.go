// Package main provides a CLI for running test agent scenarios.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fyrsmithlabs/contextd/test/agent"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// ScenarioFile represents the JSON structure for scenario files.
type ScenarioFile struct {
	Scenarios []agent.Scenario `json:"scenarios"`
}

func main() {
	// CLI flags
	scenarioPath := flag.String("scenario", "", "Path to scenario JSON file or directory")
	verbose := flag.Bool("v", false, "Verbose output")
	listScenarios := flag.Bool("list", false, "List available scenarios")
	runScenario := flag.String("run", "", "Run a specific scenario by name")
	flag.Parse()

	// Setup logging
	logLevel := zapcore.InfoLevel
	if *verbose {
		logLevel = zapcore.DebugLevel
	}

	config := zap.Config{
		Level:            zap.NewAtomicLevelAt(logLevel),
		Encoding:         "console",
		EncoderConfig:    zap.NewDevelopmentEncoderConfig(),
		OutputPaths:      []string{"stderr"},
		ErrorOutputPaths: []string{"stderr"},
	}
	logger, err := config.Build()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	// Default scenario path
	if *scenarioPath == "" {
		*scenarioPath = "test/scenarios"
	}

	// Load scenarios
	scenarios, err := loadScenarios(*scenarioPath)
	if err != nil {
		logger.Fatal("Failed to load scenarios", zap.Error(err))
	}

	if len(scenarios) == 0 {
		logger.Fatal("No scenarios found", zap.String("path", *scenarioPath))
	}

	// List mode
	if *listScenarios {
		fmt.Println("Available scenarios:")
		for _, s := range scenarios {
			fmt.Printf("  - %s: %s\n", s.Name, s.Description)
		}
		return
	}

	// Filter to specific scenario if requested
	if *runScenario != "" {
		filtered := make([]agent.Scenario, 0)
		for _, s := range scenarios {
			if s.Name == *runScenario || strings.Contains(s.Name, *runScenario) {
				filtered = append(filtered, s)
			}
		}
		if len(filtered) == 0 {
			logger.Fatal("Scenario not found", zap.String("name", *runScenario))
		}
		scenarios = filtered
	}

	// Create mock client for now (will be replaced with real client)
	client := agent.NewMockContextdClient()

	// Create runner
	runner, err := agent.NewRunner(agent.RunnerConfig{
		Client: client,
		Logger: logger,
	})
	if err != nil {
		logger.Fatal("Failed to create runner", zap.Error(err))
	}

	// Run scenarios
	ctx := context.Background()
	results, err := runner.RunScenarios(ctx, scenarios)
	if err != nil {
		logger.Fatal("Failed to run scenarios", zap.Error(err))
	}

	// Print results
	printResults(results, *verbose)

	// Exit with error if any failed
	for _, r := range results {
		if !r.Passed {
			os.Exit(1)
		}
	}
}

func loadScenarios(path string) ([]agent.Scenario, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("stat path: %w", err)
	}

	if info.IsDir() {
		return loadScenariosFromDir(path)
	}
	return loadScenariosFromFile(path)
}

func loadScenariosFromDir(dir string) ([]agent.Scenario, error) {
	scenarios := make([]agent.Scenario, 0)

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read dir: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		filePath := filepath.Join(dir, entry.Name())
		fileScenarios, err := loadScenariosFromFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("load %s: %w", filePath, err)
		}
		scenarios = append(scenarios, fileScenarios...)
	}

	return scenarios, nil
}

func loadScenariosFromFile(path string) ([]agent.Scenario, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	var file ScenarioFile
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}

	return file.Scenarios, nil
}

func printResults(results []agent.TestResult, verbose bool) {
	passed := 0
	failed := 0

	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("TEST RESULTS")
	fmt.Println(strings.Repeat("=", 60))

	for _, r := range results {
		status := "✓ PASS"
		if !r.Passed {
			status = "✗ FAIL"
			failed++
		} else {
			passed++
		}

		fmt.Printf("\n%s %s (%s)\n", status, r.Scenario, r.Duration)

		if r.Error != "" {
			fmt.Printf("  Error: %s\n", r.Error)
		}

		if verbose || !r.Passed {
			for _, ar := range r.Assertions {
				assertStatus := "  ✓"
				if !ar.Passed {
					assertStatus = "  ✗"
				}
				fmt.Printf("%s %s\n", assertStatus, ar.Assertion.Message)
				if !ar.Passed && ar.Message != "" {
					fmt.Printf("      → %s\n", ar.Message)
				}
			}
		}
	}

	fmt.Println("\n" + strings.Repeat("-", 60))
	fmt.Printf("Total: %d passed, %d failed\n", passed, failed)
	fmt.Println(strings.Repeat("=", 60))
}
