// Package main provides a GitHub webhook server that triggers Temporal workflows.
//
// This server receives GitHub webhook events and triggers corresponding Temporal
// workflows for plugin validation and other automation tasks.
//
// Usage:
//
//	TEMPORAL_HOST=localhost:7233 \
//	GITHUB_WEBHOOK_SECRET=your_secret \
//	PORT=8080 \
//	./github-webhook
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/google/go-github/v57/github"
	"go.temporal.io/sdk/client"

	"github.com/fyrsmithlabs/contextd/internal/workflows"
)

type WebhookServer struct {
	temporalClient client.Client
	webhookSecret  string
}

func main() {
	// Get configuration from environment
	temporalHost := os.Getenv("TEMPORAL_HOST")
	if temporalHost == "" {
		temporalHost = "localhost:7233"
	}

	webhookSecret := os.Getenv("GITHUB_WEBHOOK_SECRET")
	if webhookSecret == "" {
		log.Fatal("GITHUB_WEBHOOK_SECRET not set")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Create Temporal client
	c, err := client.Dial(client.Options{
		HostPort: temporalHost,
	})
	if err != nil {
		log.Fatalf("Unable to create Temporal client: %v", err)
	}
	defer c.Close()

	// Create webhook server
	server := &WebhookServer{
		temporalClient: c,
		webhookSecret:  webhookSecret,
	}

	// Setup routes
	http.HandleFunc("/webhook", server.handleWebhook)
	http.HandleFunc("/health", handleHealth)

	// Start server
	addr := ":" + port
	log.Printf("GitHub webhook server starting on %s", addr)
	log.Printf("Temporal server: %s", temporalHost)

	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func (s *WebhookServer) handleWebhook(w http.ResponseWriter, r *http.Request) {
	// Validate webhook signature
	payload, err := github.ValidatePayload(r, []byte(s.webhookSecret))
	if err != nil {
		log.Printf("Invalid webhook signature: %v", err)
		http.Error(w, "Invalid signature", http.StatusUnauthorized)
		return
	}

	// Parse webhook event
	event, err := github.ParseWebHook(github.WebHookType(r), payload)
	if err != nil {
		log.Printf("Failed to parse webhook: %v", err)
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	// Handle different event types
	switch e := event.(type) {
	case *github.PullRequestEvent:
		if err := s.handlePullRequestEvent(e); err != nil {
			log.Printf("Error handling PR event: %v", err)
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}

	default:
		log.Printf("Ignoring event type: %T", event)
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *WebhookServer) handlePullRequestEvent(event *github.PullRequestEvent) error {
	// Only trigger on opened, synchronize (new commits), and reopened
	action := event.GetAction()
	if action != "opened" && action != "synchronize" && action != "reopened" {
		log.Printf("Ignoring PR action: %s", action)
		return nil
	}

	pr := event.GetPullRequest()
	repo := event.GetRepo()

	log.Printf("Processing PR #%d: %s/%s - %s",
		pr.GetNumber(),
		repo.GetOwner().GetLogin(),
		repo.GetName(),
		action)

	// Create workflow config
	config := workflows.PluginUpdateValidationConfig{
		Owner:      repo.GetOwner().GetLogin(),
		Repo:       repo.GetName(),
		PRNumber:   pr.GetNumber(),
		BaseBranch: pr.GetBase().GetRef(),
		HeadBranch: pr.GetHead().GetRef(),
		HeadSHA:    pr.GetHead().GetSHA(),
	}

	// Start Temporal workflow
	workflowID := fmt.Sprintf("plugin-validation-%s-%s-pr-%d-%d",
		config.Owner,
		config.Repo,
		config.PRNumber,
		time.Now().Unix())

	options := client.StartWorkflowOptions{
		ID:        workflowID,
		TaskQueue: "plugin-validation-queue",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	we, err := s.temporalClient.ExecuteWorkflow(ctx, options, workflows.PluginUpdateValidationWorkflow, config)
	if err != nil {
		return fmt.Errorf("failed to start workflow: %w", err)
	}

	log.Printf("Started workflow: %s (RunID: %s)", we.GetID(), we.GetRunID())
	return nil
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}
