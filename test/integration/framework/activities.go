// Package framework provides the integration test framework for contextd.
package framework

import (
	"context"
	"fmt"
)

// Activity input/output types

// RecordMemoryInput is the input for RecordMemoryActivity.
type RecordMemoryInput struct {
	ContextdHandle ContextdHandle
	Memory         MemoryRecord
}

// SearchMemoryInput is the input for SearchMemoryActivity.
type SearchMemoryInput struct {
	ContextdHandle ContextdHandle
	Query          string
	Limit          int
}

// CheckpointSaveInput is the input for CheckpointSaveActivity.
type CheckpointSaveInput struct {
	ContextdHandle ContextdHandle
	Summary        string
}

// CheckpointResumeInput is the input for CheckpointResumeActivity.
type CheckpointResumeInput struct {
	ContextdHandle ContextdHandle
	CheckpointID   string
}

// Activities encapsulates all test framework activities.
// This allows proper dependency injection for testing.
type Activities struct {
	// sharedStore is used for cross-developer scenarios
	sharedStore *SharedStore
	// developers tracks active developer instances by contextd handle ID
	developers map[string]*Developer
}

// NewActivities creates a new Activities instance.
func NewActivities(sharedStore *SharedStore) *Activities {
	return &Activities{
		sharedStore: sharedStore,
		developers:  make(map[string]*Developer),
	}
}

// StartContextdActivity starts a contextd instance for a developer.
func (a *Activities) StartContextdActivity(ctx context.Context, config DeveloperConfig) (ContextdHandle, error) {
	var dev *Developer
	var err error

	if a.sharedStore != nil {
		dev, err = NewDeveloperWithStore(config, a.sharedStore)
	} else {
		dev, err = NewDeveloper(config)
	}
	if err != nil {
		return ContextdHandle{}, fmt.Errorf("creating developer: %w", err)
	}

	if err := dev.StartContextd(ctx); err != nil {
		return ContextdHandle{}, fmt.Errorf("starting contextd: %w", err)
	}

	handle := ContextdHandle{
		ID:        fmt.Sprintf("ctx-%s", config.ID),
		Developer: config,
	}
	a.developers[handle.ID] = dev

	return handle, nil
}

// StopContextdActivity stops a contextd instance.
func (a *Activities) StopContextdActivity(ctx context.Context, handle ContextdHandle) error {
	dev, ok := a.developers[handle.ID]
	if !ok {
		return fmt.Errorf("developer not found: %s", handle.ID)
	}

	if err := dev.StopContextd(ctx); err != nil {
		return fmt.Errorf("stopping contextd: %w", err)
	}

	delete(a.developers, handle.ID)
	return nil
}

// RecordMemoryActivity records a memory via contextd.
func (a *Activities) RecordMemoryActivity(ctx context.Context, input RecordMemoryInput) (string, error) {
	dev, ok := a.developers[input.ContextdHandle.ID]
	if !ok {
		return "", fmt.Errorf("developer not found: %s", input.ContextdHandle.ID)
	}

	memoryID, err := dev.RecordMemory(ctx, input.Memory)
	if err != nil {
		return "", fmt.Errorf("recording memory: %w", err)
	}

	return memoryID, nil
}

// SearchMemoryActivity searches for memories via contextd.
func (a *Activities) SearchMemoryActivity(ctx context.Context, input SearchMemoryInput) ([]MemoryResult, error) {
	dev, ok := a.developers[input.ContextdHandle.ID]
	if !ok {
		return nil, fmt.Errorf("developer not found: %s", input.ContextdHandle.ID)
	}

	results, err := dev.SearchMemory(ctx, input.Query, input.Limit)
	if err != nil {
		return nil, fmt.Errorf("searching memory: %w", err)
	}

	return results, nil
}

// CheckpointSaveActivity saves a checkpoint.
func (a *Activities) CheckpointSaveActivity(ctx context.Context, input CheckpointSaveInput) (string, error) {
	// TODO: Implement checkpoint save via Developer
	// For now, return a mock checkpoint ID
	return fmt.Sprintf("ckpt-%s", input.ContextdHandle.ID), nil
}

// CheckpointResumeActivity resumes from a checkpoint.
func (a *Activities) CheckpointResumeActivity(ctx context.Context, input CheckpointResumeInput) error {
	// TODO: Implement checkpoint resume via Developer
	return nil
}

// ClearContextActivity clears the current context (simulates /clear).
func (a *Activities) ClearContextActivity(ctx context.Context, handle ContextdHandle) error {
	// TODO: Implement context clear via Developer
	return nil
}

// GiveFeedbackActivity gives feedback on a memory.
func (a *Activities) GiveFeedbackActivity(ctx context.Context, handle ContextdHandle, memoryID string, helpful bool, reasoning string) error {
	dev, ok := a.developers[handle.ID]
	if !ok {
		return fmt.Errorf("developer not found: %s", handle.ID)
	}

	return dev.GiveFeedback(ctx, memoryID, helpful, reasoning)
}

// Package-level activity functions for workflow registration.
// These are thin wrappers that will be bound to an Activities instance at runtime.

// StartContextdActivity is the activity function signature for starting contextd.
func StartContextdActivity(ctx context.Context, config DeveloperConfig) (ContextdHandle, error) {
	// This is a placeholder - actual implementation uses Activities struct
	return ContextdHandle{}, fmt.Errorf("activity not registered")
}

// StopContextdActivity is the activity function signature for stopping contextd.
func StopContextdActivity(ctx context.Context, handle ContextdHandle) error {
	return fmt.Errorf("activity not registered")
}

// RecordMemoryActivity is the activity function signature for recording memory.
func RecordMemoryActivity(ctx context.Context, input RecordMemoryInput) (string, error) {
	return "", fmt.Errorf("activity not registered")
}

// SearchMemoryActivity is the activity function signature for searching memory.
func SearchMemoryActivity(ctx context.Context, input SearchMemoryInput) ([]MemoryResult, error) {
	return nil, fmt.Errorf("activity not registered")
}

// CheckpointSaveActivity is the activity function signature for saving checkpoint.
func CheckpointSaveActivity(ctx context.Context, input CheckpointSaveInput) (string, error) {
	return "", fmt.Errorf("activity not registered")
}

// CheckpointResumeActivity is the activity function signature for resuming checkpoint.
func CheckpointResumeActivity(ctx context.Context, input CheckpointResumeInput) error {
	return fmt.Errorf("activity not registered")
}

// ClearContextActivity is the activity function signature for clearing context.
func ClearContextActivity(ctx context.Context, handle ContextdHandle) error {
	return fmt.Errorf("activity not registered")
}
