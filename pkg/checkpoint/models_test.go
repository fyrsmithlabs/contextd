package checkpoint

import (
	"errors"
	"strings"
	"testing"
)

func TestCheckpoint_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cp      Checkpoint
		wantErr error
	}{
		{
			name: "valid checkpoint",
			cp: Checkpoint{
				ProjectPath: "/tmp/test-project",
				Summary:     "test checkpoint",
				Content:     "some content",
			},
			wantErr: nil,
		},
		{
			name: "missing project path",
			cp: Checkpoint{
				Summary: "test",
			},
			wantErr: ErrProjectPathRequired,
		},
		{
			name: "relative project path",
			cp: Checkpoint{
				ProjectPath: "relative/path",
				Summary:     "test",
			},
			wantErr: ErrProjectPathNotAbs,
		},
		{
			name: "missing summary",
			cp: Checkpoint{
				ProjectPath: "/tmp/test",
			},
			wantErr: ErrSummaryRequired,
		},
		{
			name: "summary too long",
			cp: Checkpoint{
				ProjectPath: "/tmp/test",
				Summary:     strings.Repeat("a", MaxSummaryLength+1),
			},
			wantErr: ErrSummaryTooLong,
		},
		{
			name: "content too large",
			cp: Checkpoint{
				ProjectPath: "/tmp/test",
				Summary:     "test",
				Content:     strings.Repeat("x", MaxContentSize+1),
			},
			wantErr: ErrContentTooLarge,
		},
		{
			name: "path with traversal cleaned",
			cp: Checkpoint{
				ProjectPath: "/tmp/../tmp/test",
				Summary:     "test",
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cp.Validate()
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
			if !errors.Is(err, ErrInvalidCheckpoint) && !errors.Is(err, tt.wantErr) {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
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
				MinScore:    DefaultMinScore,
			},
		},
		{
			name: "valid with custom values",
			opts: SearchOptions{
				ProjectPath: "/tmp/test",
				Limit:       20,
				MinScore:    0.8,
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
			name: "limit too small",
			opts: SearchOptions{
				ProjectPath: "/tmp/test",
				Limit:       0, // Will be set to default
			},
			wantErr: nil,
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
			name: "min score negative",
			opts: SearchOptions{
				ProjectPath: "/tmp/test",
				MinScore:    -0.1,
			},
			wantErr: ErrInvalidMinScore,
		},
		{
			name: "min score too high",
			opts: SearchOptions{
				ProjectPath: "/tmp/test",
				MinScore:    1.1,
			},
			wantErr: ErrInvalidMinScore,
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
				if tt.opts.MinScore != tt.wantOpt.MinScore {
					t.Errorf("Validate() MinScore = %f, want %f", tt.opts.MinScore, tt.wantOpt.MinScore)
				}
			}
		})
	}
}

func TestListOptions_Validate(t *testing.T) {
	tests := []struct {
		name    string
		opts    ListOptions
		wantErr error
	}{
		{
			name: "valid with defaults",
			opts: ListOptions{
				ProjectPath: "/tmp/test",
			},
			wantErr: nil,
		},
		{
			name: "valid with custom values",
			opts: ListOptions{
				ProjectPath: "/tmp/test",
				Limit:       50,
				Offset:      10,
			},
			wantErr: nil,
		},
		{
			name: "missing project path",
			opts: ListOptions{
				Limit: 10,
			},
			wantErr: ErrProjectPathRequired,
		},
		{
			name: "negative offset",
			opts: ListOptions{
				ProjectPath: "/tmp/test",
				Offset:      -1,
			},
			wantErr: errors.New("offset must be non-negative"),
		},
		{
			name: "limit too large",
			opts: ListOptions{
				ProjectPath: "/tmp/test",
				Limit:       MaxLimit + 1,
			},
			wantErr: ErrInvalidLimit,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.opts.Validate()
			if (err != nil) != (tt.wantErr != nil) {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
