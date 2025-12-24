package folding

import (
	"context"
	"testing"
)

func TestPermissiveSessionValidator(t *testing.T) {
	v := &PermissiveSessionValidator{}
	ctx := context.Background()

	tests := []struct {
		name      string
		sessionID string
		callerID  string
	}{
		{
			name:      "empty caller ID allowed",
			sessionID: "session_123",
			callerID:  "",
		},
		{
			name:      "any caller allowed",
			sessionID: "session_123",
			callerID:  "user_456",
		},
		{
			name:      "mismatched session and caller allowed",
			sessionID: "user_abc_session",
			callerID:  "user_xyz",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateSession(ctx, tt.sessionID, tt.callerID)
			if err != nil {
				t.Errorf("PermissiveSessionValidator.ValidateSession() = %v, want nil", err)
			}
		})
	}
}

func TestStrictSessionValidator(t *testing.T) {
	v := &StrictSessionValidator{}
	ctx := context.Background()

	tests := []struct {
		name        string
		sessionID   string
		callerID    string
		expectError bool
	}{
		{
			name:        "empty caller ID rejected",
			sessionID:   "session_123",
			callerID:    "",
			expectError: true,
		},
		{
			name:        "exact match allowed",
			sessionID:   "user_123",
			callerID:    "user_123",
			expectError: false,
		},
		{
			name:        "session starts with caller ID and underscore allowed",
			sessionID:   "user_123_session_456",
			callerID:    "user_123",
			expectError: false,
		},
		{
			name:        "session starts with caller ID but no underscore rejected",
			sessionID:   "user_123extra",
			callerID:    "user_123",
			expectError: true,
		},
		{
			name:        "mismatched caller rejected",
			sessionID:   "user_abc_session",
			callerID:    "user_xyz",
			expectError: true,
		},
		{
			name:        "caller longer than session rejected",
			sessionID:   "user",
			callerID:    "user_long_id",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateSession(ctx, tt.sessionID, tt.callerID)
			if tt.expectError {
				if err == nil {
					t.Error("StrictSessionValidator.ValidateSession() = nil, want error")
				}
				if err != ErrSessionUnauthorized {
					t.Errorf("StrictSessionValidator.ValidateSession() = %v, want ErrSessionUnauthorized", err)
				}
			} else {
				if err != nil {
					t.Errorf("StrictSessionValidator.ValidateSession() = %v, want nil", err)
				}
			}
		})
	}
}

func TestSessionValidator_Interface(t *testing.T) {
	// Ensure both implementations satisfy the interface
	var _ SessionValidator = &PermissiveSessionValidator{}
	var _ SessionValidator = &StrictSessionValidator{}
}
