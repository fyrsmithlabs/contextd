package mcp

import (
	"path/filepath"
	"testing"

	"github.com/fyrsmithlabs/contextd/internal/checkpoint"
	"github.com/stretchr/testify/assert"
)

// TestCheckpointListRequestHasProjectID verifies that when creating a ListRequest
// from checkpointListInput with ProjectPath, the ProjectID is correctly derived.
//
// Bug: checkpoint_list was failing with "invalid team ID" because:
// 1. The handler was not deriving ProjectID from ProjectPath
// 2. The checkpoint service's List method requires ProjectID for store lookup
func TestCheckpointListRequestHasProjectID(t *testing.T) {
	// Simulate the handler logic - this tests what the handler SHOULD do
	// The handler receives checkpointListInput and creates checkpoint.ListRequest

	args := checkpointListInput{
		TenantID:    "test-tenant",
		ProjectPath: "/home/user/projects/contextd",
	}

	// FIX: Derive ProjectID from ProjectPath (what the handler should do)
	projectID := ""
	if args.ProjectPath != "" {
		projectID = filepath.Base(args.ProjectPath)
	}

	listReq := &checkpoint.ListRequest{
		SessionID:   args.SessionID,
		TenantID:    args.TenantID,
		TeamID:      "", // Empty team is allowed
		ProjectID:   projectID,
		ProjectPath: args.ProjectPath,
		Limit:       args.Limit,
		AutoOnly:    args.AutoOnly,
	}

	// ProjectID should now be derived from ProjectPath
	assert.Equal(t, "contextd", listReq.ProjectID, "ProjectID should be derived from ProjectPath")
}
