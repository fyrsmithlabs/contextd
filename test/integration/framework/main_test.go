// Package framework provides the integration test harness for contextd.
package framework

import (
	"flag"
	"os"
	"testing"
)

// TestMain is the entry point for all tests in this package.
// It skips integration tests when running with -short flag.
func TestMain(m *testing.M) {
	// Parse flags first (required before testing.Short())
	flag.Parse()

	// Skip all integration tests when running with -short
	if testing.Short() {
		os.Exit(0)
	}

	// Run tests
	os.Exit(m.Run())
}
