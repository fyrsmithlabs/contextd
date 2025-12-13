package folding

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestBranchJSONRoundtrip(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	parentID := "parent_123"
	result := "found the config"

	branch := &Branch{
		ID:                "br_abc123",
		SessionID:         "sess_xyz",
		ParentID:          &parentID,
		Depth:             1,
		Description:       "Find database config",
		Prompt:            "Search for DB connection settings",
		BudgetTotal:       8192,
		BudgetUsed:        1024,
		TimeoutSeconds:    300,
		Status:            BranchStatusActive,
		Result:            &result,
		InjectedMemoryIDs: []string{"mem_001", "mem_002"},
		CreatedAt:         now,
	}

	// Marshal
	data, err := json.Marshal(branch)
	if err != nil {
		t.Fatalf("failed to marshal branch: %v", err)
	}

	// Unmarshal
	var decoded Branch
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal branch: %v", err)
	}

	// Verify
	if decoded.ID != branch.ID {
		t.Errorf("ID mismatch: got %s, want %s", decoded.ID, branch.ID)
	}
	if decoded.ParentID == nil || *decoded.ParentID != parentID {
		t.Errorf("ParentID mismatch")
	}
	if decoded.Status != BranchStatusActive {
		t.Errorf("Status mismatch: got %s, want %s", decoded.Status, BranchStatusActive)
	}
	if len(decoded.InjectedMemoryIDs) != 2 {
		t.Errorf("InjectedMemoryIDs length mismatch: got %d, want 2", len(decoded.InjectedMemoryIDs))
	}
}

func TestBranchStatusTransitions(t *testing.T) {
	tests := []struct {
		name   string
		from   BranchStatus
		to     BranchStatus
		valid  bool
	}{
		{"created to active", BranchStatusCreated, BranchStatusActive, true},
		{"created to completed", BranchStatusCreated, BranchStatusCompleted, false},
		{"active to completed", BranchStatusActive, BranchStatusCompleted, true},
		{"active to timeout", BranchStatusActive, BranchStatusTimeout, true},
		{"active to failed", BranchStatusActive, BranchStatusFailed, true},
		{"active to created", BranchStatusActive, BranchStatusCreated, false},
		{"completed to active", BranchStatusCompleted, BranchStatusActive, false},
		{"timeout to active", BranchStatusTimeout, BranchStatusActive, false},
		{"failed to active", BranchStatusFailed, BranchStatusActive, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.from.CanTransitionTo(tt.to)
			if result != tt.valid {
				t.Errorf("CanTransitionTo(%s, %s) = %v, want %v", tt.from, tt.to, result, tt.valid)
			}
		})
	}
}

func TestBranchStatusIsTerminal(t *testing.T) {
	tests := []struct {
		status   BranchStatus
		terminal bool
	}{
		{BranchStatusCreated, false},
		{BranchStatusActive, false},
		{BranchStatusCompleted, true},
		{BranchStatusTimeout, true},
		{BranchStatusFailed, true},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			if tt.status.IsTerminal() != tt.terminal {
				t.Errorf("IsTerminal(%s) = %v, want %v", tt.status, tt.status.IsTerminal(), tt.terminal)
			}
		})
	}
}

func TestBranchBudgetRemaining(t *testing.T) {
	branch := &Branch{
		BudgetTotal: 8192,
		BudgetUsed:  3000,
	}

	if branch.BudgetRemaining() != 5192 {
		t.Errorf("BudgetRemaining() = %d, want 5192", branch.BudgetRemaining())
	}
}

func TestBranchRequestValidation(t *testing.T) {
	tests := []struct {
		name    string
		req     BranchRequest
		wantErr error
	}{
		{
			name:    "valid request",
			req:     BranchRequest{SessionID: "sess_1", Description: "test", Prompt: "do something"},
			wantErr: nil,
		},
		{
			name:    "empty session_id",
			req:     BranchRequest{Description: "test", Prompt: "do something"},
			wantErr: ErrEmptySessionID,
		},
		{
			name:    "empty description",
			req:     BranchRequest{SessionID: "sess_1", Prompt: "do something"},
			wantErr: ErrEmptyDescription,
		},
		{
			name:    "description too long",
			req:     BranchRequest{SessionID: "sess_1", Description: strings.Repeat("a", MaxDescriptionLength+1), Prompt: "test"},
			wantErr: ErrDescriptionTooLong,
		},
		{
			name:    "empty prompt",
			req:     BranchRequest{SessionID: "sess_1", Description: "test"},
			wantErr: ErrEmptyPrompt,
		},
		{
			name:    "prompt too long",
			req:     BranchRequest{SessionID: "sess_1", Description: "test", Prompt: strings.Repeat("a", MaxPromptLength+1)},
			wantErr: ErrPromptTooLong,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if err != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestBranchRequestApplyDefaults(t *testing.T) {
	req := &BranchRequest{
		SessionID:   "sess_1",
		Description: "test",
		Prompt:      "do something",
	}

	req.ApplyDefaults()

	if req.Budget != DefaultBudget {
		t.Errorf("Budget = %d, want %d", req.Budget, DefaultBudget)
	}
	if req.TimeoutSeconds != DefaultTimeout {
		t.Errorf("TimeoutSeconds = %d, want %d", req.TimeoutSeconds, DefaultTimeout)
	}
}

func TestReturnRequestValidation(t *testing.T) {
	tests := []struct {
		name    string
		req     ReturnRequest
		wantErr error
	}{
		{
			name:    "valid request",
			req:     ReturnRequest{BranchID: "br_123", Message: "done"},
			wantErr: nil,
		},
		{
			name:    "empty branch_id",
			req:     ReturnRequest{Message: "done"},
			wantErr: ErrEmptyBranchID,
		},
		{
			name:    "message too long",
			req:     ReturnRequest{BranchID: "br_123", Message: strings.Repeat("a", MaxReturnMsgLength+1)},
			wantErr: ErrMessageTooLong,
		},
		{
			name:    "empty message is allowed",
			req:     ReturnRequest{BranchID: "br_123", Message: ""},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if err != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
