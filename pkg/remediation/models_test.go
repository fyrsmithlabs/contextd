package remediation

import (
	"errors"
	"strings"
	"testing"
)

func TestRemediation_Validate(t *testing.T) {
	tests := []struct {
		name    string
		rem     Remediation
		wantErr error
	}{
		{
			name: "valid remediation",
			rem: Remediation{
				ProjectPath: "/tmp/test-project",
				ErrorMsg:    "connection refused",
				Solution:    "start the server",
			},
			wantErr: nil,
		},
		{
			name: "missing project path",
			rem: Remediation{
				ErrorMsg: "error",
				Solution: "fix",
			},
			wantErr: ErrProjectPathRequired,
		},
		{
			name: "relative project path",
			rem: Remediation{
				ProjectPath: "relative/path",
				ErrorMsg:    "error",
				Solution:    "fix",
			},
			wantErr: ErrProjectPathNotAbs,
		},
		{
			name: "missing error message",
			rem: Remediation{
				ProjectPath: "/tmp/test",
				Solution:    "fix",
			},
			wantErr: ErrErrorMsgRequired,
		},
		{
			name: "error message too long",
			rem: Remediation{
				ProjectPath: "/tmp/test",
				ErrorMsg:    strings.Repeat("a", MaxErrorMsgLength+1),
				Solution:    "fix",
			},
			wantErr: ErrErrorMsgTooLong,
		},
		{
			name: "missing solution",
			rem: Remediation{
				ProjectPath: "/tmp/test",
				ErrorMsg:    "error",
			},
			wantErr: ErrSolutionRequired,
		},
		{
			name: "solution too long",
			rem: Remediation{
				ProjectPath: "/tmp/test",
				ErrorMsg:    "error",
				Solution:    strings.Repeat("x", MaxSolutionLength+1),
			},
			wantErr: ErrSolutionTooLong,
		},
		{
			name: "context too large",
			rem: Remediation{
				ProjectPath: "/tmp/test",
				ErrorMsg:    "error",
				Solution:    "fix",
				Context:     strings.Repeat("y", MaxContextSize+1),
			},
			wantErr: ErrContextTooLarge,
		},
		{
			name: "path with traversal cleaned",
			rem: Remediation{
				ProjectPath: "/tmp/../tmp/test",
				ErrorMsg:    "error",
				Solution:    "fix",
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.rem.Validate()
			if tt.wantErr == nil {
				if err != nil {
					t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}

			// Check error is present and contains expected error
			if err == nil {
				t.Errorf("Validate() error = nil, wantErr %v", tt.wantErr)
				return
			}
			if !errors.Is(err, ErrInvalidRemediation) && !errors.Is(err, tt.wantErr) {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestExtractPatterns(t *testing.T) {
	tests := []struct {
		name     string
		errorMsg string
		want     []string
	}{
		{
			name:     "connection refused",
			errorMsg: "dial tcp 127.0.0.1:8080: connect: connection refused",
			want:     []string{"connection refused"},
		},
		{
			name:     "file not found",
			errorMsg: "open /path/to/file.txt: no such file or directory",
			want:     []string{"file not found"},
		},
		{
			name:     "permission denied",
			errorMsg: "open /var/log/app.log: permission denied",
			want:     []string{"permission denied"},
		},
		{
			name:     "timeout error",
			errorMsg: "context deadline exceeded: timeout waiting for response",
			want:     []string{"timeout"},
		},
		{
			name:     "go file path",
			errorMsg: "/home/user/project/pkg/auth/auth.go:42: undefined: token",
			want:     []string{"file_path_error"},
		},
		{
			name:     "multiple patterns",
			errorMsg: "/app/server.go:100: dial tcp :8080: connection refused",
			want:     []string{"file_path_error", "connection refused"},
		},
		{
			name:     "port normalization",
			errorMsg: "failed to connect to port 3000",
			want:     []string{},
		},
		{
			name:     "no patterns",
			errorMsg: "something went wrong",
			want:     []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractPatterns(tt.errorMsg)

			if len(got) != len(tt.want) {
				t.Errorf("ExtractPatterns() = %v, want %v", got, tt.want)
				return
			}

			// Check each expected pattern is present
			for _, wantPattern := range tt.want {
				found := false
				for _, gotPattern := range got {
					if gotPattern == wantPattern {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("ExtractPatterns() missing pattern %q, got %v", wantPattern, got)
				}
			}
		})
	}
}

func TestSearchOptions_Validate(t *testing.T) {
	tests := []struct {
		name    string
		opts    SearchOptions
		wantErr error
		wantOpt SearchOptions // Expected after validation (for defaults)
	}{
		{
			name: "valid with defaults",
			opts: SearchOptions{
				ProjectPath: "/tmp/test",
			},
			wantErr: nil,
			wantOpt: SearchOptions{
				ProjectPath: "/tmp/test",
				Limit:       DefaultLimit,
				Threshold:   DefaultThreshold,
			},
		},
		{
			name: "valid with custom values",
			opts: SearchOptions{
				ProjectPath: "/tmp/test",
				Limit:       20,
				Threshold:   0.7,
			},
			wantErr: nil,
		},
		{
			name: "missing project path",
			opts: SearchOptions{
				Limit: 10,
			},
			wantErr: ErrProjectPathRequired,
		},
		{
			name: "relative project path",
			opts: SearchOptions{
				ProjectPath: "relative",
			},
			wantErr: ErrProjectPathNotAbs,
		},
		{
			name: "limit too large",
			opts: SearchOptions{
				ProjectPath: "/tmp/test",
				Limit:       MaxLimit + 1,
			},
			wantErr: ErrInvalidLimit,
		},
		{
			name: "threshold negative",
			opts: SearchOptions{
				ProjectPath: "/tmp/test",
				Threshold:   -0.1,
			},
			wantErr: ErrInvalidThreshold,
		},
		{
			name: "threshold too high",
			opts: SearchOptions{
				ProjectPath: "/tmp/test",
				Threshold:   1.1,
			},
			wantErr: ErrInvalidThreshold,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.opts.Validate()
			if (err != nil) != (tt.wantErr != nil) {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Check defaults were applied
			if tt.wantErr == nil && tt.wantOpt.Limit != 0 {
				if tt.opts.Limit != tt.wantOpt.Limit {
					t.Errorf("Validate() Limit = %d, want %d", tt.opts.Limit, tt.wantOpt.Limit)
				}
				if tt.opts.Threshold != tt.wantOpt.Threshold {
					t.Errorf("Validate() Threshold = %f, want %f", tt.opts.Threshold, tt.wantOpt.Threshold)
				}
			}
		})
	}
}
