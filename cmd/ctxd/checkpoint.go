package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/fyrsmithlabs/contextd/internal/checkpoint"
	"github.com/fyrsmithlabs/contextd/internal/config"
	"github.com/fyrsmithlabs/contextd/internal/embeddings"
	"github.com/fyrsmithlabs/contextd/internal/logging"
	"github.com/fyrsmithlabs/contextd/internal/vectorstore"
)

var (
	// checkpoint command flags
	cpTenantID    string
	cpTeamID      string
	cpProjectID   string
	cpProjectPath string
	cpSessionID   string
	cpAutoOnly    bool
	cpLimit       int
	cpLevel       string
	cpOutputJSON  bool
)

func init() {
	rootCmd.AddCommand(checkpointCmd)
	checkpointCmd.AddCommand(checkpointListCmd)
	checkpointCmd.AddCommand(checkpointResumeCmd)

	// Common flags for all checkpoint commands
	checkpointCmd.PersistentFlags().StringVar(&cpTenantID, "tenant-id", "", "Tenant identifier (required)")
	checkpointCmd.PersistentFlags().StringVar(&cpTeamID, "team-id", "", "Team identifier (defaults to tenant-id)")
	checkpointCmd.PersistentFlags().StringVar(&cpProjectID, "project-id", "", "Project identifier (defaults to project path basename)")
	checkpointCmd.PersistentFlags().StringVar(&cpProjectPath, "project-path", "", "Project path (defaults to current directory)")
	checkpointCmd.PersistentFlags().BoolVar(&cpOutputJSON, "json", false, "Output results as JSON")

	// List-specific flags
	checkpointListCmd.Flags().StringVar(&cpSessionID, "session-id", "", "Filter by session ID")
	checkpointListCmd.Flags().BoolVar(&cpAutoOnly, "auto-only", false, "Only show auto-created checkpoints")
	checkpointListCmd.Flags().IntVar(&cpLimit, "limit", 20, "Maximum number of checkpoints to return")

	// Resume-specific flags
	checkpointResumeCmd.Flags().StringVar(&cpLevel, "level", "context", "Resume level: summary, context, or full")
}

var checkpointCmd = &cobra.Command{
	Use:   "checkpoint",
	Short: "Manage checkpoints",
	Long: `Manage checkpoints for saving and resuming session state.

Checkpoints allow you to save the current session state and resume it later.
This is useful for preserving context across sessions or recovering from interruptions.

Examples:
  # List all checkpoints for a project
  ctxd checkpoint list --tenant-id dahendel --project-path /home/dahendel/projects/contextd

  # List checkpoints for a specific session
  ctxd checkpoint list --tenant-id dahendel --session-id sess_123

  # Resume from a checkpoint
  ctxd checkpoint resume <checkpoint-id> --tenant-id dahendel --level context`,
}

var checkpointListCmd = &cobra.Command{
	Use:   "list",
	Short: "List checkpoints",
	Long: `List checkpoints for a project or session.

Examples:
  # List all checkpoints for a project
  ctxd checkpoint list --tenant-id dahendel --project-path /home/dahendel/projects/contextd

  # List checkpoints for a specific session
  ctxd checkpoint list --tenant-id dahendel --session-id sess_123

  # List only auto-created checkpoints
  ctxd checkpoint list --tenant-id dahendel --auto-only

  # Output as JSON
  ctxd checkpoint list --tenant-id dahendel --json`,
	RunE: runCheckpointList,
}

var checkpointResumeCmd = &cobra.Command{
	Use:   "resume <checkpoint-id>",
	Short: "Resume from a checkpoint",
	Long: `Resume from a checkpoint at the specified level.

Resume levels:
  summary - Only the brief summary (minimal context)
  context - Summary + relevant context (recommended)
  full    - Complete checkpoint state

Examples:
  # Resume with context level (recommended)
  ctxd checkpoint resume ckpt_123 --tenant-id dahendel --level context

  # Resume with full state
  ctxd checkpoint resume ckpt_123 --tenant-id dahendel --level full

  # Output as JSON
  ctxd checkpoint resume ckpt_123 --tenant-id dahendel --json`,
	Args: cobra.ExactArgs(1),
	RunE: runCheckpointResume,
}

func runCheckpointList(cmd *cobra.Command, args []string) error {
	// Validate required flags
	if cpTenantID == "" {
		return fmt.Errorf("--tenant-id is required")
	}

	// Set defaults
	if cpTeamID == "" {
		cpTeamID = cpTenantID
	}
	if cpProjectPath == "" {
		var err error
		cpProjectPath, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
	}
	if cpProjectID == "" {
		cpProjectID = getProjectIDFromPath(cpProjectPath)
	}

	// Initialize services
	svc, err := initCheckpointService()
	if err != nil {
		return err
	}
	defer svc.Close()

	// Create list request
	req := &checkpoint.ListRequest{
		TenantID:    cpTenantID,
		TeamID:      cpTeamID,
		ProjectID:   cpProjectID,
		ProjectPath: cpProjectPath,
		SessionID:   cpSessionID,
		AutoOnly:    cpAutoOnly,
		Limit:       cpLimit,
	}

	// Call service
	checkpoints, err := svc.List(context.Background(), req)
	if err != nil {
		return fmt.Errorf("failed to list checkpoints: %w", err)
	}

	// Output results
	if cpOutputJSON {
		return outputJSON(checkpoints)
	}

	// Human-readable table output
	if len(checkpoints) == 0 {
		fmt.Println("No checkpoints found")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tSESSION\tCREATED\tAUTO\tTOKENS")
	for _, cp := range checkpoints {
		autoStr := ""
		if cp.AutoCreated {
			autoStr = "yes"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%d\n",
			truncate(cp.ID, 12),
			truncate(cp.Name, 30),
			truncate(cp.SessionID, 12),
			cp.CreatedAt.Format("2006-01-02 15:04"),
			autoStr,
			cp.TokenCount,
		)
	}
	w.Flush()

	return nil
}

func runCheckpointResume(cmd *cobra.Command, args []string) error {
	checkpointID := args[0]

	// Validate required flags
	if cpTenantID == "" {
		return fmt.Errorf("--tenant-id is required")
	}

	// Validate level
	validLevels := map[string]checkpoint.ResumeLevel{
		"summary": checkpoint.ResumeSummary,
		"context": checkpoint.ResumeContext,
		"full":    checkpoint.ResumeFull,
	}
	resumeLevel, ok := validLevels[cpLevel]
	if !ok {
		return fmt.Errorf("invalid level: %s (valid: summary, context, full)", cpLevel)
	}

	// Set defaults
	if cpTeamID == "" {
		cpTeamID = cpTenantID
	}
	if cpProjectPath == "" {
		var err error
		cpProjectPath, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
	}
	if cpProjectID == "" {
		cpProjectID = getProjectIDFromPath(cpProjectPath)
	}

	// Initialize services
	svc, err := initCheckpointService()
	if err != nil {
		return err
	}
	defer svc.Close()

	// Create resume request
	req := &checkpoint.ResumeRequest{
		CheckpointID: checkpointID,
		TenantID:     cpTenantID,
		TeamID:       cpTeamID,
		ProjectID:    cpProjectID,
		Level:        resumeLevel,
	}

	// Call service
	resp, err := svc.Resume(context.Background(), req)
	if err != nil {
		return fmt.Errorf("failed to resume checkpoint: %w", err)
	}

	// Output results
	if cpOutputJSON {
		return outputJSON(resp)
	}

	// Human-readable output
	fmt.Printf("Checkpoint: %s\n", resp.Checkpoint.Name)
	fmt.Printf("Description: %s\n", resp.Checkpoint.Description)
	fmt.Printf("Created: %s\n", resp.Checkpoint.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("Session: %s\n", resp.Checkpoint.SessionID)
	fmt.Printf("Token Count: %d\n", resp.TokenCount)
	fmt.Printf("\n--- Content (%s level) ---\n\n", cpLevel)
	fmt.Println(resp.Content)

	return nil
}

// Helper functions

func initCheckpointService() (checkpoint.Service, error) {
	// Load configuration (try file first, fallback to env vars)
	cfg, err := config.LoadWithFile("")
	if err != nil {
		// Fall back to environment-only config
		cfg = config.Load()
	}

	// Initialize logger
	logCfg := logging.NewDefaultConfig()
	logger, err := logging.NewLogger(logCfg, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create logger: %w", err)
	}

	// Initialize embeddings provider
	embCfg := embeddings.ProviderConfig{
		Provider: cfg.Embeddings.Provider,
		Model:    cfg.Embeddings.Model,
		BaseURL:  cfg.Embeddings.BaseURL,
		CacheDir: cfg.Embeddings.CacheDir,
	}
	embProvider, err := embeddings.NewProvider(embCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create embeddings provider: %w", err)
	}

	// Get provider dimension and update config
	providerDim := embProvider.Dimension()
	cfg.VectorStore.Chromem.VectorSize = providerDim

	// Initialize vector store
	store, err := vectorstore.NewStore(cfg, embProvider, logger.Underlying())
	if err != nil {
		return nil, fmt.Errorf("failed to create vectorstore: %w", err)
	}

	// Initialize checkpoint service (using legacy adapter for single store)
	cpCfg := checkpoint.DefaultServiceConfig()
	cpCfg.VectorSize = uint64(providerDim)
	svc, err := checkpoint.NewServiceWithStore(cpCfg, store, logger.Underlying())
	if err != nil {
		return nil, fmt.Errorf("failed to create checkpoint service: %w", err)
	}

	return svc, nil
}

func getProjectIDFromPath(path string) string {
	// Simple implementation: use basename of path
	// For more sophisticated logic, could parse git remote, etc.
	if path == "" {
		return "default"
	}
	base := path
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' || path[i] == '\\' {
			base = path[i+1:]
			break
		}
	}
	if base == "" {
		return "default"
	}
	return base
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func outputJSON(v interface{}) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
