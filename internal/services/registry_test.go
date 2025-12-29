package services

import (
	"testing"

	"github.com/fyrsmithlabs/contextd/internal/checkpoint"
	"github.com/fyrsmithlabs/contextd/internal/hooks"
	"github.com/fyrsmithlabs/contextd/internal/reasoningbank"
	"github.com/fyrsmithlabs/contextd/internal/remediation"
	"github.com/fyrsmithlabs/contextd/internal/repository"
	"github.com/fyrsmithlabs/contextd/internal/secrets"
	"github.com/fyrsmithlabs/contextd/internal/troubleshoot"
)

func TestNewRegistry(t *testing.T) {
	// This will fail because Registry doesn't exist yet
	var _ Registry = (*registry)(nil)
}

func TestRegistryAccessors(t *testing.T) {
	// Create registry with nil services - just testing interface
	reg := NewRegistry(Options{})

	// Test that accessors return what was passed
	if reg.Checkpoint() != nil {
		t.Error("expected nil checkpoint service")
	}
	if reg.Hooks() != nil {
		t.Error("expected nil hooks manager")
	}
	if reg.Memory() != nil {
		t.Error("expected nil memory service")
	}
	if reg.Repository() != nil {
		t.Error("expected nil repository service")
	}
	if reg.Troubleshoot() != nil {
		t.Error("expected nil troubleshoot service")
	}
	if reg.Distiller() != nil {
		t.Error("expected nil distiller")
	}
	if reg.Scrubber() != nil {
		t.Error("expected nil scrubber")
	}
	if reg.Remediation() != nil {
		t.Error("expected nil remediation service")
	}
}

func TestRegistryWithServices(t *testing.T) {
	// Create mock services
	var mockCheckpoint checkpoint.Service
	var mockRemediation remediation.Service
	var mockMemory *reasoningbank.Service
	var mockRepository *repository.Service
	var mockTroubleshoot *troubleshoot.Service
	var mockHooks *hooks.HookManager
	var mockDistiller *reasoningbank.Distiller
	var mockScrubber secrets.Scrubber

	// Create registry with services
	reg := NewRegistry(Options{
		Checkpoint:   mockCheckpoint,
		Remediation:  mockRemediation,
		Memory:       mockMemory,
		Repository:   mockRepository,
		Troubleshoot: mockTroubleshoot,
		Hooks:        mockHooks,
		Distiller:    mockDistiller,
		Scrubber:     mockScrubber,
	})

	// Test that accessors return the same instances
	if reg.Checkpoint() != mockCheckpoint {
		t.Error("checkpoint service mismatch")
	}
	if reg.Remediation() != mockRemediation {
		t.Error("remediation service mismatch")
	}
	if reg.Memory() != mockMemory {
		t.Error("memory service mismatch")
	}
	if reg.Repository() != mockRepository {
		t.Error("repository service mismatch")
	}
	if reg.Troubleshoot() != mockTroubleshoot {
		t.Error("troubleshoot service mismatch")
	}
	if reg.Hooks() != mockHooks {
		t.Error("hooks manager mismatch")
	}
	if reg.Distiller() != mockDistiller {
		t.Error("distiller mismatch")
	}
	if reg.Scrubber() != mockScrubber {
		t.Error("scrubber mismatch")
	}
}
