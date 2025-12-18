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

	t.Run("with map merges existing metadata", func(t *testing.T) {
		existing := map[string]interface{}{"a": 1, "b": 2}
		got := NewMetadataBuilder().
			WithMap(existing).
			With("c", 3).
			Build()

		if got["a"] != 1 || got["b"] != 2 || got["c"] != 3 {
			t.Errorf("Build() = %v, want {a:1, b:2, c:3}", got)
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
		{
			name:    "tenant_id wrong type (int)",
			filters: map[string]interface{}{"tenant_id": 123},
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

	t.Run("tenant_id wrong type returns error", func(t *testing.T) {
		filters := map[string]interface{}{"tenant_id": 123}
		_, err := ExtractTenantFromFilters(filters)
		if err != ErrInvalidTenant {
			t.Errorf("ExtractTenantFromFilters() error = %v, want ErrInvalidTenant", err)
		}
	})
}

// =============================================================================
// ApplyTenantFilters Tests - Security Critical
// =============================================================================

func TestApplyTenantFilters(t *testing.T) {
	// Fast path tests
	t.Run("both nil returns nil", func(t *testing.T) {
		got, err := ApplyTenantFilters(nil, nil)
		if err != nil {
			t.Fatalf("ApplyTenantFilters() error = %v", err)
		}
		if got != nil {
			t.Errorf("ApplyTenantFilters(nil, nil) = %v, want nil", got)
		}
	})

	t.Run("nil user filters returns tenant filters directly", func(t *testing.T) {
		tenantFilters := map[string]interface{}{"tenant_id": "org-123"}
		got, err := ApplyTenantFilters(nil, tenantFilters)
		if err != nil {
			t.Fatalf("ApplyTenantFilters() error = %v", err)
		}
		// Should return the same map (no allocation)
		if got["tenant_id"] != "org-123" {
			t.Errorf("ApplyTenantFilters() tenant_id = %v, want org-123", got["tenant_id"])
		}
	})

	t.Run("nil tenant filters returns copy of user filters", func(t *testing.T) {
		userFilters := map[string]interface{}{"status": "active"}
		got, err := ApplyTenantFilters(userFilters, nil)
		if err != nil {
			t.Fatalf("ApplyTenantFilters() error = %v", err)
		}
		if got["status"] != "active" {
			t.Errorf("ApplyTenantFilters() status = %v, want active", got["status"])
		}
	})

	// Merge behavior tests
	t.Run("merges user and tenant filters", func(t *testing.T) {
		userFilters := map[string]interface{}{"status": "active", "type": "memory"}
		tenantFilters := map[string]interface{}{"tenant_id": "org-123", "team_id": "team-1"}

		got, err := ApplyTenantFilters(userFilters, tenantFilters)
		if err != nil {
			t.Fatalf("ApplyTenantFilters() error = %v", err)
		}

		// Check all fields present
		if got["status"] != "active" {
			t.Errorf("ApplyTenantFilters() status = %v, want active", got["status"])
		}
		if got["type"] != "memory" {
			t.Errorf("ApplyTenantFilters() type = %v, want memory", got["type"])
		}
		if got["tenant_id"] != "org-123" {
			t.Errorf("ApplyTenantFilters() tenant_id = %v, want org-123", got["tenant_id"])
		}
		if got["team_id"] != "team-1" {
			t.Errorf("ApplyTenantFilters() team_id = %v, want team-1", got["team_id"])
		}
	})

	// SECURITY: Tenant injection prevention tests
	t.Run("SECURITY: rejects tenant_id in user filters", func(t *testing.T) {
		userFilters := map[string]interface{}{
			"tenant_id": "attacker-org", // Attacker tries to inject tenant
			"status":    "active",
		}
		tenantFilters := map[string]interface{}{"tenant_id": "legitimate-org"}

		_, err := ApplyTenantFilters(userFilters, tenantFilters)
		if err != ErrTenantFilterInUserFilters {
			t.Errorf("SECURITY VIOLATION: ApplyTenantFilters() should reject tenant_id in user filters, got error = %v", err)
		}
	})

	t.Run("SECURITY: rejects team_id in user filters", func(t *testing.T) {
		userFilters := map[string]interface{}{
			"team_id": "attacker-team", // Attacker tries to inject team
			"status":  "active",
		}
		tenantFilters := map[string]interface{}{"tenant_id": "legitimate-org"}

		_, err := ApplyTenantFilters(userFilters, tenantFilters)
		if err != ErrTenantFilterInUserFilters {
			t.Errorf("SECURITY VIOLATION: ApplyTenantFilters() should reject team_id in user filters, got error = %v", err)
		}
	})

	t.Run("SECURITY: rejects project_id in user filters", func(t *testing.T) {
		userFilters := map[string]interface{}{
			"project_id": "attacker-project", // Attacker tries to inject project
			"status":     "active",
		}
		tenantFilters := map[string]interface{}{"tenant_id": "legitimate-org"}

		_, err := ApplyTenantFilters(userFilters, tenantFilters)
		if err != ErrTenantFilterInUserFilters {
			t.Errorf("SECURITY VIOLATION: ApplyTenantFilters() should reject project_id in user filters, got error = %v", err)
		}
	})

	t.Run("SECURITY: rejects all tenant fields even with nil tenant filters", func(t *testing.T) {
		// Even if tenant filters are nil, user should not be able to inject tenant fields
		userFilters := map[string]interface{}{
			"tenant_id": "attacker-org",
			"status":    "active",
		}

		_, err := ApplyTenantFilters(userFilters, nil)
		if err != ErrTenantFilterInUserFilters {
			t.Errorf("SECURITY VIOLATION: ApplyTenantFilters() should reject tenant_id even with nil tenant filters, got error = %v", err)
		}
	})

	t.Run("SECURITY: rejects multiple tenant fields in user filters", func(t *testing.T) {
		userFilters := map[string]interface{}{
			"tenant_id":  "attacker-org",
			"team_id":    "attacker-team",
			"project_id": "attacker-project",
		}
		tenantFilters := map[string]interface{}{"tenant_id": "legitimate-org"}

		_, err := ApplyTenantFilters(userFilters, tenantFilters)
		if err != ErrTenantFilterInUserFilters {
			t.Errorf("SECURITY VIOLATION: ApplyTenantFilters() should reject multiple tenant fields, got error = %v", err)
		}
	})

	// Edge cases
	t.Run("allows empty string values for non-tenant fields", func(t *testing.T) {
		userFilters := map[string]interface{}{"status": ""}
		tenantFilters := map[string]interface{}{"tenant_id": "org-123"}

		got, err := ApplyTenantFilters(userFilters, tenantFilters)
		if err != nil {
			t.Fatalf("ApplyTenantFilters() error = %v", err)
		}
		if got["status"] != "" {
			t.Errorf("ApplyTenantFilters() status = %v, want empty string", got["status"])
		}
	})

	t.Run("preserves various value types", func(t *testing.T) {
		userFilters := map[string]interface{}{
			"string_val": "test",
			"int_val":    42,
			"float_val":  3.14,
			"bool_val":   true,
			"slice_val":  []string{"a", "b"},
		}
		tenantFilters := map[string]interface{}{"tenant_id": "org-123"}

		got, err := ApplyTenantFilters(userFilters, tenantFilters)
		if err != nil {
			t.Fatalf("ApplyTenantFilters() error = %v", err)
		}

		if got["string_val"] != "test" {
			t.Errorf("ApplyTenantFilters() string_val = %v, want test", got["string_val"])
		}
		if got["int_val"] != 42 {
			t.Errorf("ApplyTenantFilters() int_val = %v, want 42", got["int_val"])
		}
		if got["float_val"] != 3.14 {
			t.Errorf("ApplyTenantFilters() float_val = %v, want 3.14", got["float_val"])
		}
		if got["bool_val"] != true {
			t.Errorf("ApplyTenantFilters() bool_val = %v, want true", got["bool_val"])
		}
	})
}

// TestApplyTenantFilters_DoesNotMutateInputs verifies that the function
// does not modify the input maps (important for security and correctness).
func TestApplyTenantFilters_DoesNotMutateInputs(t *testing.T) {
	userFilters := map[string]interface{}{"status": "active"}
	tenantFilters := map[string]interface{}{"tenant_id": "org-123"}

	// Store original values
	origUserLen := len(userFilters)
	origTenantLen := len(tenantFilters)

	got, err := ApplyTenantFilters(userFilters, tenantFilters)
	if err != nil {
		t.Fatalf("ApplyTenantFilters() error = %v", err)
	}

	// Verify inputs were not mutated
	if len(userFilters) != origUserLen {
		t.Errorf("userFilters was mutated: len changed from %d to %d", origUserLen, len(userFilters))
	}
	if len(tenantFilters) != origTenantLen {
		t.Errorf("tenantFilters was mutated: len changed from %d to %d", origTenantLen, len(tenantFilters))
	}

	// Verify output is independent
	got["new_key"] = "new_value"
	if _, exists := userFilters["new_key"]; exists {
		t.Error("Modifying output affected userFilters input")
	}
	if _, exists := tenantFilters["new_key"]; exists {
		t.Error("Modifying output affected tenantFilters input")
	}
}
