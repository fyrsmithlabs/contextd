// Package main provides a Temporal worker for plugin validation workflows.
//
// This worker listens for plugin validation workflows triggered by GitHub webhooks
// and executes them using Temporal's durable execution engine.
//
// Usage:
//
//	TEMPORAL_HOST=localhost:7233 \
//	GITHUB_TOKEN=ghp_xxx \
//	./plugin-validator
package main

import (
	"log"
	"os"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"

	"github.com/fyrsmithlabs/contextd/internal/workflows"
)

func main() {
	// Get Temporal server address
	temporalHost := os.Getenv("TEMPORAL_HOST")
	if temporalHost == "" {
		temporalHost = "localhost:7233"
	}

	// Create Temporal client
	c, err := client.Dial(client.Options{
		HostPort: temporalHost,
	})
	if err != nil {
		log.Fatalf("Unable to create Temporal client: %v", err)
	}
	defer c.Close()

	// Create worker
	w := worker.New(c, "plugin-validation-queue", worker.Options{})

	// Register workflow
	w.RegisterWorkflow(workflows.PluginUpdateValidationWorkflow)

	// Register activities
	w.RegisterActivity(workflows.FetchPRFilesActivity)
	w.RegisterActivity(workflows.CategorizeFilesActivity)
	w.RegisterActivity(workflows.ValidatePluginSchemasActivity)
	w.RegisterActivity(workflows.PostReminderCommentActivity)
	w.RegisterActivity(workflows.PostSuccessCommentActivity)

	// Start worker
	log.Println("Plugin validation worker starting...")
	log.Printf("Temporal server: %s", temporalHost)
	log.Printf("Task queue: plugin-validation-queue")

	err = w.Run(worker.InterruptCh())
	if err != nil {
		log.Fatalf("Unable to start worker: %v", err)
	}
}
