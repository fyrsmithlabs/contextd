//go:build cgo

package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestInitCmd_Exists(t *testing.T) {
	// Verify initCmd exists and is added to rootCmd
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "init" {
			found = true
			break
		}
	}
	if !found {
		t.Error("init command not found in rootCmd")
	}
}

func TestInitCmd_Help(t *testing.T) {
	// Test that help text is set
	var initCmd *cobra.Command
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "init" {
			initCmd = cmd
			break
		}
	}

	if initCmd == nil {
		t.Fatal("init command not found")
	}

	if initCmd.Short == "" {
		t.Error("init command should have Short description")
	}

	if initCmd.Long == "" {
		t.Error("init command should have Long description")
	}

	// Check the Long description mentions ONNX
	if !strings.Contains(strings.ToLower(initCmd.Long), "onnx") {
		t.Error("init command Long description should mention ONNX")
	}
}

func TestInitCmd_ForceFlag(t *testing.T) {
	var initCmd *cobra.Command
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "init" {
			initCmd = cmd
			break
		}
	}

	if initCmd == nil {
		t.Fatal("init command not found")
	}

	// Check --force flag exists
	forceFlag := initCmd.Flags().Lookup("force")
	if forceFlag == nil {
		t.Error("init command should have --force flag")
	}
}

func TestInitCmd_AlreadyInstalled(t *testing.T) {
	// Set up test environment with ONNX_PATH set
	tmpDir := t.TempDir()
	libPath := filepath.Join(tmpDir, "libonnxruntime.so")

	// Create fake library file
	if err := os.WriteFile(libPath, []byte("fake lib"), 0644); err != nil {
		t.Fatal(err)
	}

	// Set env var
	oldONNX := os.Getenv("ONNX_PATH")
	os.Setenv("ONNX_PATH", libPath)
	defer func() {
		if oldONNX != "" {
			os.Setenv("ONNX_PATH", oldONNX)
		} else {
			os.Unsetenv("ONNX_PATH")
		}
	}()

	// Run init command - should succeed without downloading
	var initCmd *cobra.Command
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "init" {
			initCmd = cmd
			break
		}
	}

	if initCmd == nil {
		t.Fatal("init command not found")
	}

	// Capture output
	var out bytes.Buffer
	initCmd.SetOut(&out)
	initCmd.SetErr(&out)

	// Run without --force
	err := initCmd.RunE(initCmd, []string{})
	if err != nil {
		t.Errorf("init command failed: %v", err)
	}

	// Output should indicate already installed
	output := out.String()
	if !strings.Contains(strings.ToLower(output), "already") {
		t.Errorf("output should indicate ONNX is already installed, got: %s", output)
	}
}
