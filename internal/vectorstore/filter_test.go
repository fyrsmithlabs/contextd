package vectorstore

import (
	"testing"
)

func TestMergeFilters(t *testing.T) {
	tests := []struct {
		name     string
		base     map[string]interface{}
		override map[string]interface{}
		want     map[string]interface{}
	}{
		{
			name:     "both nil",
			base:     nil,
			override: nil,
			want:     nil,
		},
		{
			name:     "base only",
			base:     map[string]interface{}{"a": 1},
			override: nil,
			want:     map[string]interface{}{"a": 1},
		},
		{
			name:     "override only",
			base:     nil,
			override: map[string]interface{}{"b": 2},
			want:     map[string]interface{}{"b": 2},
		},
		{
			name:     "merge without conflict",
			base:     map[string]interface{}{"a": 1},
			override: map[string]interface{}{"b": 2},
			want:     map[string]interface{}{"a": 1, "b": 2},
		},
		{
			name:     "override wins on conflict",
			base:     map[string]interface{}{"a": 1, "b": "old"},
			override: map[string]interface{}{"b": "new"},
			want:     map[string]interface{}{"a": 1, "b": "new"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MergeFilters(tt.base, tt.override)
			if tt.want == nil {
				if got != nil {
					t.Errorf("MergeFilters() = %v, want nil", got)
				}
				return
			}
			if len(got) != len(tt.want) {
				t.Errorf("MergeFilters() len = %d, want %d", len(got), len(tt.want))
			}
			for k, v := range tt.want {
				if got[k] != v {
					t.Errorf("MergeFilters()[%s] = %v, want %v", k, got[k], v)
				}
			}
		})
	}
}

func TestFilterBuilder(t *testing.T) {
	t.Run("builds filter", func(t *testing.T) {
		got := NewFilterBuilder().
			With("status", "active").
			With("type", "memory").
			Build()

		if got["status"] != "active" {
			t.Errorf("Build() status = %v, want active", got["status"])
		}
		if got["type"] != "memory" {
			t.Errorf("Build() type = %v, want memory", got["type"])
		}
	})

	t.Run("with tenant", func(t *testing.T) {
		tenant := &TenantInfo{TenantID: "org-123", TeamID: "team-1"}
		got := NewFilterBuilder().
			WithTenant(tenant).
			Build()

		if got["tenant_id"] != "org-123" {
			t.Errorf("Build() tenant_id = %v, want org-123", got["tenant_id"])
		}
		if got["team_id"] != "team-1" {
			t.Errorf("Build() team_id = %v, want team-1", got["team_id"])
		}
	})

	t.Run("with nil tenant", func(t *testing.T) {
		got := NewFilterBuilder().
			WithTenant(nil).
			With("other", "value").
			Build()

		if got["other"] != "value" {
			t.Errorf("Build() other = %v, want value", got["other"])
		}
		if _, ok := got["tenant_id"]; ok {
			t.Error("Build() should not have tenant_id with nil tenant")
		}
	})

	t.Run("with map", func(t *testing.T) {
		existing := map[string]interface{}{"a": 1, "b": 2}
		got := NewFilterBuilder().
			WithMap(existing).
			With("c", 3).
			Build()

		if got["a"] != 1 || got["b"] != 2 || got["c"] != 3 {
			t.Errorf("Build() = %v, want {a:1, b:2, c:3}", got)
		}
	})

	t.Run("empty builder returns nil", func(t *testing.T) {
		got := NewFilterBuilder().Build()
		if got != nil {
			t.Errorf("Build() = %v, want nil", got)
		}
	})
}

func TestMetadataBuilder(t *testing.T) {
	t.Run("builds metadata", func(t *testing.T) {
		got := NewMetadataBuilder().
			With("title", "Test").
			With("score", 0.95).
			Build()

		if got["title"] != "Test" {
			t.Errorf("Build() title = %v, want Test", got["title"])
		}
		if got["score"] != 0.95 {
			t.Errorf("Build() score = %v, want 0.95", got["score"])
		}
	})

	t.Run("with tenant", func(t *testing.T) {
		tenant := &TenantInfo{TenantID: "org-123", ProjectID: "proj-1"}
		got := NewMetadataBuilder().
			WithTenant(tenant).
			Build()

		if got["tenant_id"] != "org-123" {
			t.Errorf("Build() tenant_id = %v, want org-123", got["tenant_id"])
		}
		if got["project_id"] != "proj-1" {
			t.Errorf("Build() project_id = %v, want proj-1", got["project_id"])
		}
	})
}

func TestValidateFilterHasTenant(t *testing.T) {
	tests := []struct {
		name    string
		filters map[string]interface{}
		wantErr error
	}{
		{
			name:    "valid",
			filters: map[string]interface{}{"tenant_id": "org-123"},
			wantErr: nil,
		},
		{
			name:    "nil filters",
			filters: nil,
			wantErr: ErrMissingTenant,
		},
		{
			name:    "missing tenant_id",
			filters: map[string]interface{}{"other": "value"},
			wantErr: ErrMissingTenant,
		},
		{
			name:    "empty tenant_id",
			filters: map[string]interface{}{"tenant_id": ""},
			wantErr: ErrInvalidTenant,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFilterHasTenant(tt.filters)
			if err != tt.wantErr {
				t.Errorf("ValidateFilterHasTenant() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestExtractTenantFromFilters(t *testing.T) {
	t.Run("extracts all fields", func(t *testing.T) {
		filters := map[string]interface{}{
			"tenant_id":  "org-123",
			"team_id":    "team-1",
			"project_id": "proj-1",
		}

		got, err := ExtractTenantFromFilters(filters)
		if err != nil {
			t.Fatalf("ExtractTenantFromFilters() error = %v", err)
		}
		if got.TenantID != "org-123" || got.TeamID != "team-1" || got.ProjectID != "proj-1" {
			t.Errorf("ExtractTenantFromFilters() = %+v, want {org-123, team-1, proj-1}", got)
		}
	})

	t.Run("missing tenant_id returns error", func(t *testing.T) {
		filters := map[string]interface{}{"other": "value"}
		_, err := ExtractTenantFromFilters(filters)
		if err != ErrMissingTenant {
			t.Errorf("ExtractTenantFromFilters() error = %v, want ErrMissingTenant", err)
		}
	})

	t.Run("optional fields can be missing", func(t *testing.T) {
		filters := map[string]interface{}{"tenant_id": "org-123"}

		got, err := ExtractTenantFromFilters(filters)
		if err != nil {
			t.Fatalf("ExtractTenantFromFilters() error = %v", err)
		}
		if got.TenantID != "org-123" {
			t.Errorf("ExtractTenantFromFilters() TenantID = %v, want org-123", got.TenantID)
		}
		if got.TeamID != "" || got.ProjectID != "" {
			t.Errorf("ExtractTenantFromFilters() optional fields should be empty")
		}
	})
}
