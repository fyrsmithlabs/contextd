package handlers

import (
	"testing"
)

func TestRegistry_SessionTools(t *testing.T) {
	// Create registry with session tools (services.Registry provided)
	reg := NewRegistry(nil, nil, nil, nil, &mockRegistry{}, nil, nil)

	tools := reg.ListTools()

	expectedTools := []string{"session_start", "session_end", "context_threshold"}
	for _, expected := range expectedTools {
		found := false
		for _, tool := range tools {
			if tool == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected tool %s not found", expected)
		}
	}
}

func TestRegistry_SessionTools_NilRegistry(t *testing.T) {
	// Create registry without session tools (no services.Registry)
	reg := NewRegistry(nil, nil, nil, nil, nil, nil, nil)

	tools := reg.ListTools()

	// Session tools should NOT be present
	sessionTools := []string{"session_start", "session_end", "context_threshold"}
	for _, sessionTool := range sessionTools {
		for _, tool := range tools {
			if tool == sessionTool {
				t.Errorf("session tool %s should not be present when svcRegistry is nil", sessionTool)
			}
		}
	}
}
