// Package hooks provides lifecycle hook management for contextd
package hooks

import (
	"context"
	"fmt"
)

// HookType represents different lifecycle hooks
type HookType string

const (
	// HookSessionStart is called when a new session starts
	HookSessionStart HookType = "session_start"

	// HookSessionEnd is called when a session ends
	HookSessionEnd HookType = "session_end"

	// HookBeforeClear is called before /clear command
	HookBeforeClear HookType = "before_clear"

	// HookAfterClear is called after /clear command
	HookAfterClear HookType = "after_clear"

	// HookContextThreshold is called when context threshold reached
	HookContextThreshold HookType = "context_threshold"
)

// Config holds hook configuration
type Config struct {
	// AutoCheckpointOnClear enables automatic checkpoint before /clear
	AutoCheckpointOnClear bool `json:"auto_checkpoint_on_clear"`

	// AutoResumeOnStart enables automatic resume on session start
	AutoResumeOnStart bool `json:"auto_resume_on_start"`

	// CheckpointThreshold is the context percentage to trigger checkpoint (70-95)
	CheckpointThreshold int `json:"checkpoint_threshold_percent"`

	// VerifyBeforeClear enables verification before clearing
	VerifyBeforeClear bool `json:"verify_before_clear"`
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Threshold must be < 100 to ensure checkpoint happens before context is completely full
	// Valid range: 1-99 (e.g., 70 means checkpoint at 70% context usage)
	if c.CheckpointThreshold < 1 || c.CheckpointThreshold >= 100 {
		return fmt.Errorf("checkpoint_threshold must be between 1 and 99, got %d", c.CheckpointThreshold)
	}
	return nil
}

// HookHandler is a function that handles a hook event
type HookHandler func(ctx context.Context, data map[string]interface{}) error

// HookManager manages lifecycle hooks
type HookManager struct {
	config   *Config
	handlers map[HookType][]HookHandler
}

// NewHookManager creates a new hook manager
func NewHookManager(config *Config) *HookManager {
	return &HookManager{
		config:   config,
		handlers: make(map[HookType][]HookHandler),
	}
}

// RegisterHandler registers a handler for a hook type
func (h *HookManager) RegisterHandler(hookType HookType, handler HookHandler) {
	h.handlers[hookType] = append(h.handlers[hookType], handler)
}

// Execute executes all handlers for the given hook type
func (h *HookManager) Execute(ctx context.Context, hookType HookType, data map[string]interface{}) error {
	handlers, ok := h.handlers[hookType]
	if !ok {
		// No handlers registered - not an error
		return nil
	}

	for _, handler := range handlers {
		if err := handler(ctx, data); err != nil {
			return fmt.Errorf("hook %s failed: %w", hookType, err)
		}
	}

	return nil
}

// Config returns the hook configuration
func (h *HookManager) Config() *Config {
	return h.config
}
