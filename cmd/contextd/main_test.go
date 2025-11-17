package main

import (
	"context"
	"net/http"
	"os"
	"testing"
	"time"
)

func TestMainIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Set test port to avoid conflicts
	os.Setenv("SERVER_PORT", "8084")
	defer os.Unsetenv("SERVER_PORT")

	// Create context with timeout for the test
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Start server in goroutine
	errCh := make(chan error, 1)
	go func() {
		errCh <- run(ctx)
	}()

	// Wait for server to start
	time.Sleep(200 * time.Millisecond)

	// Test health check endpoint
	resp, err := http.Get("http://localhost:8084/health")
	if err != nil {
		t.Fatalf("GET /health failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("GET /health status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	// Cancel context to shutdown server
	cancel()

	// Wait for server to stop
	select {
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			t.Errorf("run() error = %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("server did not shutdown in time")
	}
}
