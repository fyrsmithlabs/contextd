package sanitize

import (
	"errors"
	"testing"
)

func TestValidatePath(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		allowedRoot string
		wantErr     error
	}{
		{
			name:    "empty path",
			path:    "",
			wantErr: ErrEmptyPath,
		},
		{
			name:    "simple relative path",
			path:    "foo/bar",
			wantErr: nil,
		},
		{
			name:    "simple absolute path",
			path:    "/tmp/test",
			wantErr: nil,
		},
		{
			name:    "traversal attack - simple",
			path:    "../etc/passwd",
			wantErr: ErrPathTraversal,
		},
		{
			name:    "traversal attack - middle",
			path:    "foo/../../../etc/passwd",
			wantErr: ErrPathTraversal,
		},
		{
			name:    "traversal attack - encoded still contains dots",
			path:    "foo/..%2f..%2fetc/passwd",
			wantErr: ErrPathTraversal, // Contains literal ".." even if slashes are encoded
		},
		{
			name:    "traversal attack - double dots at end",
			path:    "foo/bar/..",
			wantErr: ErrPathTraversal,
		},
		{
			name:        "path within root",
			path:        "/tmp/test/subdir",
			allowedRoot: "/tmp/test",
			wantErr:     nil,
		},
		{
			name:        "path escapes root",
			path:        "/tmp/test/../other",
			allowedRoot: "/tmp/test",
			wantErr:     ErrPathTraversal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ValidatePath(tt.path, tt.allowedRoot)
			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("ValidatePath() expected error containing %v, got nil", tt.wantErr)
					return
				}
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("ValidatePath() error = %v, want %v", err, tt.wantErr)
				}
			} else {
				if err != nil {
					t.Errorf("ValidatePath() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestSafeBasename(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		wantBase string
		wantErr  error
	}{
		{
			name:     "simple path",
			path:     "/foo/bar/baz",
			wantBase: "baz",
			wantErr:  nil,
		},
		{
			name:     "single component",
			path:     "file.txt",
			wantBase: "file.txt",
			wantErr:  nil,
		},
		{
			name:    "empty path",
			path:    "",
			wantErr: ErrEmptyPath,
		},
		{
			name:    "traversal attack",
			path:    "/foo/../bar",
			wantErr: ErrPathTraversal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SafeBasename(tt.path)
			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("SafeBasename() expected error, got nil")
					return
				}
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("SafeBasename() error = %v, want %v", err, tt.wantErr)
				}
			} else {
				if err != nil {
					t.Errorf("SafeBasename() unexpected error = %v", err)
					return
				}
				if got != tt.wantBase {
					t.Errorf("SafeBasename() = %q, want %q", got, tt.wantBase)
				}
			}
		})
	}
}

func TestValidateTenantID(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		wantErr error
	}{
		{
			name:    "valid lowercase",
			id:      "mytenant",
			wantErr: nil,
		},
		{
			name:    "valid with underscore",
			id:      "my_tenant_123",
			wantErr: nil,
		},
		{
			name:    "valid single char",
			id:      "a",
			wantErr: nil,
		},
		{
			name:    "empty",
			id:      "",
			wantErr: ErrInvalidTenantID,
		},
		{
			name:    "contains slash",
			id:      "tenant/bad",
			wantErr: ErrInvalidTenantID,
		},
		{
			name:    "contains backslash",
			id:      "tenant\\bad",
			wantErr: ErrInvalidTenantID,
		},
		{
			name:    "contains dots",
			id:      "tenant..bad",
			wantErr: ErrInvalidTenantID,
		},
		{
			name:    "uppercase",
			id:      "MyTenant",
			wantErr: ErrInvalidTenantID,
		},
		{
			name:    "starts with underscore",
			id:      "_tenant",
			wantErr: ErrInvalidTenantID,
		},
		{
			name:    "contains special chars",
			id:      "tenant@bad!",
			wantErr: ErrInvalidTenantID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTenantID(tt.id)
			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("ValidateTenantID() expected error, got nil")
					return
				}
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("ValidateTenantID() error = %v, want %v", err, tt.wantErr)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateTenantID() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestValidateTeamID(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		wantErr error
	}{
		{
			name:    "empty is allowed",
			id:      "",
			wantErr: nil,
		},
		{
			name:    "valid",
			id:      "platform",
			wantErr: nil,
		},
		{
			name:    "contains slash",
			id:      "team/bad",
			wantErr: ErrInvalidTeamID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTeamID(tt.id)
			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("ValidateTeamID() expected error, got nil")
					return
				}
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("ValidateTeamID() error = %v, want %v", err, tt.wantErr)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateTeamID() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestValidateGlobPattern(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		wantErr error
	}{
		{
			name:    "empty is allowed",
			pattern: "",
			wantErr: nil,
		},
		{
			name:    "simple glob",
			pattern: "*.go",
			wantErr: nil,
		},
		{
			name:    "recursive glob",
			pattern: "**/*.go",
			wantErr: nil,
		},
		{
			name:    "directory glob",
			pattern: "vendor/**",
			wantErr: nil,
		},
		{
			name:    "contains traversal",
			pattern: "../**/*.go",
			wantErr: ErrInvalidPattern,
		},
		{
			name:    "contains shell injection semicolon",
			pattern: "*.go; rm -rf /",
			wantErr: ErrInvalidPattern,
		},
		{
			name:    "contains pipe",
			pattern: "*.go | cat",
			wantErr: ErrInvalidPattern,
		},
		{
			name:    "contains backtick",
			pattern: "*.`whoami`",
			wantErr: ErrInvalidPattern,
		},
		{
			name:    "excessive wildcards",
			pattern: "*****.go",
			wantErr: ErrInvalidPattern,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateGlobPattern(tt.pattern)
			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("ValidateGlobPattern() expected error, got nil")
					return
				}
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("ValidateGlobPattern() error = %v, want %v", err, tt.wantErr)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateGlobPattern() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestSanitizeAndValidateTenantID(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		want     string
		wantErr  bool
	}{
		{
			name:    "valid lowercase",
			input:   "mytenant",
			want:    "mytenant",
			wantErr: false,
		},
		{
			name:    "uppercase gets sanitized",
			input:   "MyTenant",
			want:    "mytenant",
			wantErr: false,
		},
		{
			name:    "special chars get sanitized",
			input:   "my-tenant.com",
			want:    "my_tenant_com",
			wantErr: false,
		},
		{
			name:    "spaces get sanitized",
			input:   "My Tenant Name",
			want:    "my_tenant_name",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SanitizeAndValidateTenantID(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("SanitizeAndValidateTenantID() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("SanitizeAndValidateTenantID() unexpected error = %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("SanitizeAndValidateTenantID() = %q, want %q", got, tt.want)
			}
		})
	}
}
