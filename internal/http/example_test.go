package http_test

import (
	"context"
	"fmt"
	"time"

	httpserver "github.com/fyrsmithlabs/contextd/internal/http"
	"github.com/fyrsmithlabs/contextd/internal/secrets"
	"go.uber.org/zap"
)

// ExampleServer demonstrates how to create and start the HTTP server.
func ExampleServer() {
	// Create a scrubber with default configuration
	scrubber, err := secrets.New(nil)
	if err != nil {
		panic(err)
	}

	// Create logger
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	// Configure the server
	cfg := &httpserver.Config{
		Host: "localhost",
		Port: 9090,
	}

	// Create the server
	server, err := httpserver.NewServer(scrubber, logger, cfg)
	if err != nil {
		panic(err)
	}

	// Start server in background
	go func() {
		if err := server.Start(); err != nil {
			logger.Error("server error", zap.Error(err))
		}
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("shutdown error", zap.Error(err))
	}

	fmt.Println("Server started and stopped successfully")
	// Output: Server started and stopped successfully
}
