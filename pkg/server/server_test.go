package server

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/fyrsmithlabs/contextd/pkg/config"
)

func TestNewServer(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Port:            8080,
			ShutdownTimeout: 10 * time.Second,
		},
	}

	srv := NewServer(cfg)
	if srv == nil {
		t.Fatal("NewServer() returned nil")
	}

	if srv.config.Server.Port != 8080 {
		t.Errorf("server port = %d, want 8080", srv.config.Server.Port)
	}
}

func TestServer_HealthCheck(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Port:            8081,
			ShutdownTimeout: 5 * time.Second,
		},
	}

	srv := NewServer(cfg)

	// Start server in background
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Start(ctx)
	}()

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)

	// Test health check endpoint
	resp, err := http.Get("http://localhost:8081/health")
	if err != nil {
		t.Fatalf("GET /health failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("GET /health status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	// Shutdown server
	cancel()

	// Wait for server to stop
	select {
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			t.Errorf("Start() error = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("server did not shutdown in time")
	}
}

func TestServer_GracefulShutdown(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Port:            8082,
			ShutdownTimeout: 2 * time.Second,
		},
	}

	srv := NewServer(cfg)

	ctx, cancel := context.WithCancel(context.Background())

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Start(ctx)
	}()

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)

	// Verify server is running
	resp, err := http.Get("http://localhost:8082/health")
	if err != nil {
		t.Fatalf("server not running: %v", err)
	}
	resp.Body.Close()

	// Trigger shutdown
	shutdownStart := time.Now()
	cancel()

	// Wait for server to stop
	select {
	case shutdownErr := <-errCh:
		shutdownDuration := time.Since(shutdownStart)
		if shutdownErr != nil && shutdownErr != http.ErrServerClosed {
			t.Errorf("Start() error = %v", shutdownErr)
		}
		// Verify shutdown was fast (< timeout)
		if shutdownDuration > 3*time.Second {
			t.Errorf("shutdown took %v, expected < 3s", shutdownDuration)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("server did not shutdown within timeout")
	}

	// Verify server is stopped
	checkResp, checkErr := http.Get("http://localhost:8082/health")
	if checkErr == nil {
		checkResp.Body.Close()
		t.Error("server still responding after shutdown")
	}
}

func TestServer_PortAlreadyInUse(t *testing.T) {
	port := 8083
	cfg := &config.Config{
		Server: config.ServerConfig{
			Port:            port,
			ShutdownTimeout: 2 * time.Second,
		},
	}

	// Start first server
	srv1 := NewServer(cfg)
	ctx1, cancel1 := context.WithCancel(context.Background())
	defer cancel1()

	errCh1 := make(chan error, 1)
	go func() {
		errCh1 <- srv1.Start(ctx1)
	}()

	// Wait for first server to start
	time.Sleep(100 * time.Millisecond)

	// Try to start second server on same port
	srv2 := NewServer(cfg)
	ctx2, cancel2 := context.WithCancel(context.Background())
	defer cancel2()

	err := srv2.Start(ctx2)
	if err == nil {
		t.Error("expected error when port is already in use, got nil")
	}

	// Cleanup first server
	cancel1()
	select {
	case <-errCh1:
	case <-time.After(2 * time.Second):
		t.Fatal("first server did not shutdown")
	}
}
