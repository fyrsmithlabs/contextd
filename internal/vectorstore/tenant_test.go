package vectorstore

import (
	"context"
	"testing"
)

func TestTenantInfo_Validate(t *testing.T) {
	tests := []struct {
		name    string
		tenant  *TenantInfo
		wantErr error
	}{
		{
			name:    "valid with tenant only",
			tenant:  &TenantInfo{TenantID: "org-123"},
			wantErr: nil,
		},
		{
			name:    "valid with all fields",
			tenant:  &TenantInfo{TenantID: "org-123", TeamID: "team-1", ProjectID: "proj-1"},
			wantErr: nil,
		},
		{
			name:    "invalid empty tenant",
			tenant:  &TenantInfo{TenantID: ""},
			wantErr: ErrInvalidTenant,
		},
		{
			name:    "invalid zero value",
			tenant:  &TenantInfo{},
			wantErr: ErrInvalidTenant,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.tenant.Validate()
			if err != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTenantContext(t *testing.T) {
	t.Run("round trip", func(t *testing.T) {
		tenant := &TenantInfo{
			TenantID:  "org-123",
			TeamID:    "team-1",
			ProjectID: "proj-1",
		}

		ctx := ContextWithTenant(context.Background(), tenant)
		got, err := TenantFromContext(ctx)
		if err != nil {
			t.Fatalf("TenantFromContext() error = %v", err)
		}
		if got.TenantID != tenant.TenantID || got.TeamID != tenant.TeamID || got.ProjectID != tenant.ProjectID {
			t.Errorf("TenantFromContext() = %+v, want %+v", got, tenant)
		}
	})

	t.Run("missing tenant returns error", func(t *testing.T) {
		ctx := context.Background()
		_, err := TenantFromContext(ctx)
		if err != ErrMissingTenant {
			t.Errorf("TenantFromContext() error = %v, want ErrMissingTenant", err)
		}
	})

	t.Run("nil tenant returns error", func(t *testing.T) {
		ctx := ContextWithTenant(context.Background(), nil)
		_, err := TenantFromContext(ctx)
		if err != ErrMissingTenant {
			t.Errorf("TenantFromContext() error = %v, want ErrMissingTenant", err)
		}
	})

	t.Run("HasTenant returns true when present", func(t *testing.T) {
		ctx := ContextWithTenant(context.Background(), &TenantInfo{TenantID: "test"})
		if !HasTenant(ctx) {
			t.Error("HasTenant() = false, want true")
		}
	})

	t.Run("HasTenant returns false when absent", func(t *testing.T) {
		if HasTenant(context.Background()) {
			t.Error("HasTenant() = true, want false")
		}
	})
}

func TestMustTenantFromContext(t *testing.T) {
	t.Run("returns tenant when present", func(t *testing.T) {
		tenant := &TenantInfo{TenantID: "org-123"}
		ctx := ContextWithTenant(context.Background(), tenant)
		got := MustTenantFromContext(ctx)
		if got.TenantID != tenant.TenantID {
			t.Errorf("MustTenantFromContext() = %+v, want %+v", got, tenant)
		}
	})

	t.Run("panics when absent", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("MustTenantFromContext() did not panic")
			}
		}()
		MustTenantFromContext(context.Background())
	})
}

func TestTenantInfo_TenantMetadata(t *testing.T) {
	tests := []struct {
		name   string
		tenant *TenantInfo
		want   map[string]interface{}
	}{
		{
			name:   "tenant only",
			tenant: &TenantInfo{TenantID: "org-123"},
			want:   map[string]interface{}{"tenant_id": "org-123"},
		},
		{
			name:   "with team",
			tenant: &TenantInfo{TenantID: "org-123", TeamID: "team-1"},
			want:   map[string]interface{}{"tenant_id": "org-123", "team_id": "team-1"},
		},
		{
			name:   "all fields",
			tenant: &TenantInfo{TenantID: "org-123", TeamID: "team-1", ProjectID: "proj-1"},
			want:   map[string]interface{}{"tenant_id": "org-123", "team_id": "team-1", "project_id": "proj-1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.tenant.TenantMetadata()
			if len(got) != len(tt.want) {
				t.Errorf("TenantMetadata() len = %d, want %d", len(got), len(tt.want))
			}
			for k, v := range tt.want {
				if got[k] != v {
					t.Errorf("TenantMetadata()[%s] = %v, want %v", k, got[k], v)
				}
			}
		})
	}
}

func TestTenantInfo_TenantFilter(t *testing.T) {
	tests := []struct {
		name   string
		tenant *TenantInfo
		want   map[string]interface{}
	}{
		{
			name:   "tenant only",
			tenant: &TenantInfo{TenantID: "org-123"},
			want:   map[string]interface{}{"tenant_id": "org-123"},
		},
		{
			name:   "with team",
			tenant: &TenantInfo{TenantID: "org-123", TeamID: "team-1"},
			want:   map[string]interface{}{"tenant_id": "org-123", "team_id": "team-1"},
		},
		{
			name:   "all fields",
			tenant: &TenantInfo{TenantID: "org-123", TeamID: "team-1", ProjectID: "proj-1"},
			want:   map[string]interface{}{"tenant_id": "org-123", "team_id": "team-1", "project_id": "proj-1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.tenant.TenantFilter()
			if len(got) != len(tt.want) {
				t.Errorf("TenantFilter() len = %d, want %d", len(got), len(tt.want))
			}
			for k, v := range tt.want {
				if got[k] != v {
					t.Errorf("TenantFilter()[%s] = %v, want %v", k, got[k], v)
				}
			}
		})
	}
}
