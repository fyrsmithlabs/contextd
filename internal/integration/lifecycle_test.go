//go:build integration

package integration

import (
	"testing"
)

func TestSessionLifecycle_EndToEnd(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Test plan:
	// 1. session_start returns primed memories
	// 2. session_end calls distiller, creates memory
	// 3. next session_start finds the new memory

	t.Run("full_lifecycle", func(t *testing.T) {
		// For now, just verify the test file compiles
		// Full integration requires Qdrant running
		t.Log("Integration test placeholder - requires live Qdrant")
	})
}
