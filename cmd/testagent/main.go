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
	analyzeConvos := flag.String("analyze", "", "Analyze conversation exports from directory")
	generateFrom := flag.String("generate", "", "Generate scenarios from conversation exports")
	outputFile := flag.String("output", "", "Output file for generated scenarios")
	flag.Parse()

	// Handle analyze mode
	if *analyzeConvos != "" {
		runAnalyze(*analyzeConvos)
		return
	}

	// Handle generate mode
	if *generateFrom != "" {
		runGenerate(*generateFrom, *outputFile)
		return
	}

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
	defer func() { _ = logger.Sync() }()

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

func runAnalyze(dir string) {
	fmt.Printf("Analyzing conversations in: %s\n", dir)

	stats, err := agent.ParseConversationsDir(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\nFound %d conversations with contextd usage\n", len(stats))
	fmt.Println(strings.Repeat("-", 60))

	// Show aggregate stats
	agg := agent.AnalyzeConversations(stats)
	fmt.Printf("\nAggregate Statistics:\n")
	fmt.Printf("  Total sessions:        %v\n", agg["total_sessions"])
	fmt.Printf("  Total contextd calls:  %v\n", agg["total_contextd_calls"])
	fmt.Printf("  Total searches:        %v\n", agg["total_searches"])
	fmt.Printf("  Total records:         %v\n", agg["total_records"])
	fmt.Printf("  Total feedbacks:       %v\n", agg["total_feedbacks"])
	fmt.Printf("  Total checkpoints:     %v\n", agg["total_checkpoints"])
	fmt.Printf("  Avg calls/session:     %.1f\n", agg["avg_calls_per_session"])

	// Show top sessions by contextd usage
	fmt.Println("\nTop sessions by contextd usage:")
	// Sort by number of calls (simple bubble sort for small N)
	for i := 0; i < len(stats)-1; i++ {
		for j := 0; j < len(stats)-i-1; j++ {
			if len(stats[j].ContextdToolCalls) < len(stats[j+1].ContextdToolCalls) {
				stats[j], stats[j+1] = stats[j+1], stats[j]
			}
		}
	}

	limit := 10
	if len(stats) < limit {
		limit = len(stats)
	}
	for i := 0; i < limit; i++ {
		s := stats[i]
		fmt.Printf("  %d. %s: %d calls (searches: %d, records: %d, feedbacks: %d)\n",
			i+1, s.SessionID[:8], len(s.ContextdToolCalls),
			s.MemorySearches, s.MemoryRecords, s.MemoryFeedbacks)
	}
}

func runGenerate(dir string, outputPath string) {
	fmt.Printf("Generating scenarios from: %s\n", dir)

	stats, err := agent.ParseConversationsDir(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Filter to sessions with meaningful activity
	var filtered []*agent.ConversationStats
	for _, s := range stats {
		if len(s.ContextdToolCalls) >= 3 {
			filtered = append(filtered, s)
		}
	}

	fmt.Printf("Found %d conversations with 3+ contextd calls\n", len(filtered))

	// Generate scenarios from top sessions
	scenarios := make([]agent.Scenario, 0)

	// Sort by number of calls
	for i := 0; i < len(filtered)-1; i++ {
		for j := 0; j < len(filtered)-i-1; j++ {
			if len(filtered[j].ContextdToolCalls) < len(filtered[j+1].ContextdToolCalls) {
				filtered[j], filtered[j+1] = filtered[j+1], filtered[j]
			}
		}
	}

	// Take top 10 sessions
	limit := 10
	if len(filtered) < limit {
		limit = len(filtered)
	}
	for i := 0; i < limit; i++ {
		scenario := agent.GenerateScenarioFromStats(filtered[i])
		if scenario != nil && len(scenario.Actions) > 0 {
			scenarios = append(scenarios, *scenario)
		}
	}

	if len(scenarios) == 0 {
		fmt.Println("No scenarios generated (no actionable data)")
		return
	}

	// Output scenarios
	scenarioFile := ScenarioFile{Scenarios: scenarios}
	data, err := json.MarshalIndent(scenarioFile, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling: %v\n", err)
		os.Exit(1)
	}

	if outputPath == "" {
		outputPath = "test/scenarios/generated.json"
	}

	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\nGenerated %d scenarios to: %s\n", len(scenarios), outputPath)
}
