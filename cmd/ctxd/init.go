//go:build cgo

package main

import (
	"context"
	"fmt"

	"github.com/fyrsmithlabs/contextd/internal/embeddings"
	"github.com/spf13/cobra"
)

var (
	forceDownload bool
)

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().BoolVarP(&forceDownload, "force", "f", false, "Force re-download even if ONNX runtime exists")
}

// initCmd initializes contextd dependencies
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize contextd dependencies",
	Long: `Initialize contextd by downloading required dependencies.

Currently this downloads the ONNX runtime library required for local
embeddings with FastEmbed. The library is installed to:
  ~/.config/contextd/lib/

If ONNX_PATH environment variable is set, that path takes precedence.

Examples:
  # Initialize contextd (download ONNX runtime)
  ctxd init

  # Force re-download even if already installed
  ctxd init --force`,
	RunE: runInit,
}

func runInit(cmd *cobra.Command, args []string) error {
	// Check if already installed (unless --force)
	if !forceDownload {
		if path := embeddings.GetONNXLibraryPath(); path != "" {
			cmd.Printf("ONNX runtime already installed at: %s\n", path)
			cmd.Println("Use --force to re-download.")
			return nil
		}
	}

	cmd.Printf("Downloading ONNX runtime v%s...\n", embeddings.DefaultONNXRuntimeVersion)

	// Use the download function from embeddings package
	if err := embeddings.DownloadONNXRuntime(context.Background(), ""); err != nil {
		return fmt.Errorf("failed to download ONNX runtime: %w", err)
	}

	// Verify installation
	path := embeddings.GetONNXLibraryPath()
	if path == "" {
		return fmt.Errorf("download completed but library not found")
	}

	cmd.Printf("Successfully installed ONNX runtime to: %s\n", path)
	return nil
}
