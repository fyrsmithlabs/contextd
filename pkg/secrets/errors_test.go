package secrets

import (
	"errors"
	"testing"
)

func TestErrors(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		wantMsg string
	}{
		{
			name:    "invalid regex error",
			err:     ErrInvalidRegex,
			wantMsg: "invalid regex pattern",
		},
		{
			name:    "invalid toml error",
			err:     ErrInvalidTOML,
			wantMsg: "invalid TOML format",
		},
		{
			name:    "allowlist not found error",
			err:     ErrAllowlistNotFound,
			wantMsg: "allowlist file not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Error() != tt.wantMsg {
				t.Errorf("error message = %q, want %q", tt.err.Error(), tt.wantMsg)
			}
		})
	}
}

func TestErrorWrapping(t *testing.T) {
	baseErr := errors.New("base error")
	wrappedErr := errors.Join(ErrInvalidRegex, baseErr)

	if !errors.Is(wrappedErr, ErrInvalidRegex) {
		t.Error("wrapped error should be identifiable as ErrInvalidRegex")
	}
}
