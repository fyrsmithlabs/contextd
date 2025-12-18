package vectorstore

import (
	"context"
	"testing"
)

func TestPayloadIsolation_InjectFilter(t *testing.T) {
	iso := NewPayloadIsolation()

	t.Run("injects tenant filter", func(t *testing.T) {
		ctx := ContextWithTenant(context.Background(), &TenantInfo{
			TenantID:  "org-123",
			TeamID:    "team-1",
			ProjectID: "proj-1",
		})

		got, err := iso.InjectFilter(ctx, nil)
		if err != nil {
			t.Fatalf("InjectFilter() error = %v", err)
		}
		if got["tenant_id"] != "org-123" {
			t.Errorf("InjectFilter() tenant_id = %v, want org-123", got["tenant_id"])
		}
		if got["team_id"] != "team-1" {
			t.Errorf("InjectFilter() team_id = %v, want team-1", got["team_id"])
		}
		if got["project_id"] != "proj-1" {
			t.Errorf("InjectFilter() project_id = %v, want proj-1", got["project_id"])
		}
	})

	t.Run("merges with existing filters", func(t *testing.T) {
		ctx := ContextWithTenant(context.Background(), &TenantInfo{TenantID: "org-123"})
		existing := map[string]interface{}{"status": "active"}

		got, err := iso.InjectFilter(ctx, existing)
		if err != nil {
			t.Fatalf("InjectFilter() error = %v", err)
		}
		if got["tenant_id"] != "org-123" {
			t.Errorf("InjectFilter() tenant_id = %v, want org-123", got["tenant_id"])
		}
		if got["status"] != "active" {
			t.Errorf("InjectFilter() status = %v, want active", got["status"])
		}
	})

	t.Run("tenant filter overrides existing tenant_id", func(t *testing.T) {
		ctx := ContextWithTenant(context.Background(), &TenantInfo{TenantID: "org-123"})
		// Attacker tries to inject different tenant
		malicious := map[string]interface{}{"tenant_id": "victim-org"}

		got, err := iso.InjectFilter(ctx, malicious)
		if err != nil {
			t.Fatalf("InjectFilter() error = %v", err)
		}
		// Security: context tenant wins over filter tenant
		if got["tenant_id"] != "org-123" {
			t.Errorf("SECURITY VIOLATION: tenant_id = %v, want org-123 (context tenant should win)", got["tenant_id"])
		}
	})

	t.Run("fails without tenant context", func(t *testing.T) {
		_, err := iso.InjectFilter(context.Background(), nil)
		if err != ErrMissingTenant {
			t.Errorf("InjectFilter() error = %v, want ErrMissingTenant", err)
		}
	})

	t.Run("fails with invalid tenant", func(t *testing.T) {
		ctx := ContextWithTenant(context.Background(), &TenantInfo{TenantID: ""})
		_, err := iso.InjectFilter(ctx, nil)
		if err != ErrInvalidTenant {
			t.Errorf("InjectFilter() error = %v, want ErrInvalidTenant", err)
		}
	})
}

func TestPayloadIsolation_InjectMetadata(t *testing.T) {
	iso := NewPayloadIsolation()

	t.Run("injects tenant metadata", func(t *testing.T) {
		ctx := ContextWithTenant(context.Background(), &TenantInfo{
			TenantID:  "org-123",
			TeamID:    "team-1",
			ProjectID: "proj-1",
		})

		docs := []Document{{ID: "doc1", Content: "test"}}
		err := iso.InjectMetadata(ctx, docs)
		if err != nil {
			t.Fatalf("InjectMetadata() error = %v", err)
		}
		if docs[0].Metadata["tenant_id"] != "org-123" {
			t.Errorf("InjectMetadata() tenant_id = %v, want org-123", docs[0].Metadata["tenant_id"])
		}
	})

	t.Run("overwrites existing tenant_id for security", func(t *testing.T) {
		ctx := ContextWithTenant(context.Background(), &TenantInfo{TenantID: "org-123"})

		docs := []Document{{
			ID:       "doc1",
			Content:  "test",
			Metadata: map[string]interface{}{"tenant_id": "victim-org"},
		}}

		err := iso.InjectMetadata(ctx, docs)
		if err != nil {
			t.Fatalf("InjectMetadata() error = %v", err)
		}
		// Security: context tenant overwrites any provided tenant_id
		if docs[0].Metadata["tenant_id"] != "org-123" {
			t.Errorf("SECURITY VIOLATION: tenant_id = %v, want org-123", docs[0].Metadata["tenant_id"])
		}
	})

	t.Run("preserves existing metadata", func(t *testing.T) {
		ctx := ContextWithTenant(context.Background(), &TenantInfo{TenantID: "org-123"})

		docs := []Document{{
			ID:       "doc1",
			Content:  "test",
			Metadata: map[string]interface{}{"custom": "value"},
		}}

		err := iso.InjectMetadata(ctx, docs)
		if err != nil {
			t.Fatalf("InjectMetadata() error = %v", err)
		}
		if docs[0].Metadata["custom"] != "value" {
			t.Errorf("InjectMetadata() custom = %v, want value", docs[0].Metadata["custom"])
		}
	})

	t.Run("fails without tenant context", func(t *testing.T) {
		docs := []Document{{ID: "doc1"}}
		err := iso.InjectMetadata(context.Background(), docs)
		if err != ErrMissingTenant {
			t.Errorf("InjectMetadata() error = %v, want ErrMissingTenant", err)
		}
	})
}

func TestPayloadIsolation_ValidateTenant(t *testing.T) {
	iso := NewPayloadIsolation()

	t.Run("valid tenant", func(t *testing.T) {
		ctx := ContextWithTenant(context.Background(), &TenantInfo{TenantID: "org-123"})
		if err := iso.ValidateTenant(ctx); err != nil {
			t.Errorf("ValidateTenant() error = %v, want nil", err)
		}
	})

	t.Run("missing tenant", func(t *testing.T) {
		if err := iso.ValidateTenant(context.Background()); err != ErrMissingTenant {
			t.Errorf("ValidateTenant() error = %v, want ErrMissingTenant", err)
		}
	})

	t.Run("invalid tenant", func(t *testing.T) {
		ctx := ContextWithTenant(context.Background(), &TenantInfo{TenantID: ""})
		if err := iso.ValidateTenant(ctx); err != ErrInvalidTenant {
			t.Errorf("ValidateTenant() error = %v, want ErrInvalidTenant", err)
		}
	})
}

func TestFilesystemIsolation_InjectFilter(t *testing.T) {
	iso := NewFilesystemIsolation()

	t.Run("passes filters through unchanged", func(t *testing.T) {
		ctx := ContextWithTenant(context.Background(), &TenantInfo{TenantID: "org-123"})
		existing := map[string]interface{}{"status": "active"}

		got, err := iso.InjectFilter(ctx, existing)
		if err != nil {
			t.Fatalf("InjectFilter() error = %v", err)
		}
		// Filesystem isolation doesn't inject tenant filters
		if _, ok := got["tenant_id"]; ok {
			t.Error("InjectFilter() should not add tenant_id for filesystem isolation")
		}
		if got["status"] != "active" {
			t.Errorf("InjectFilter() status = %v, want active", got["status"])
		}
	})

	t.Run("still requires tenant context", func(t *testing.T) {
		_, err := iso.InjectFilter(context.Background(), nil)
		if err != ErrMissingTenant {
			t.Errorf("InjectFilter() error = %v, want ErrMissingTenant", err)
		}
	})
}

func TestFilesystemIsolation_InjectMetadata(t *testing.T) {
	iso := NewFilesystemIsolation()

	t.Run("adds tenant metadata for audit purposes", func(t *testing.T) {
		ctx := ContextWithTenant(context.Background(), &TenantInfo{
			TenantID:  "org-123",
			TeamID:    "team-1",
			ProjectID: "proj-1",
		})

		docs := []Document{{ID: "doc1", Content: "test"}}
		err := iso.InjectMetadata(ctx, docs)
		if err != nil {
			t.Fatalf("InjectMetadata() error = %v", err)
		}

		// Filesystem isolation still adds metadata for audit purposes
		if docs[0].Metadata["tenant_id"] != "org-123" {
			t.Errorf("InjectMetadata() tenant_id = %v, want org-123", docs[0].Metadata["tenant_id"])
		}
		if docs[0].Metadata["team_id"] != "team-1" {
			t.Errorf("InjectMetadata() team_id = %v, want team-1", docs[0].Metadata["team_id"])
		}
	})
}

func TestNoIsolation(t *testing.T) {
	iso := NewNoIsolation()

	t.Run("allows everything", func(t *testing.T) {
		// No tenant context required
		_, err := iso.InjectFilter(context.Background(), nil)
		if err != nil {
			t.Errorf("InjectFilter() error = %v, want nil", err)
		}

		if err := iso.ValidateTenant(context.Background()); err != nil {
			t.Errorf("ValidateTenant() error = %v, want nil", err)
		}

		docs := []Document{{ID: "doc1"}}
		if err := iso.InjectMetadata(context.Background(), docs); err != nil {
			t.Errorf("InjectMetadata() error = %v, want nil", err)
		}
	})

	t.Run("mode is none", func(t *testing.T) {
		if got := iso.Mode(); got != "none" {
			t.Errorf("Mode() = %v, want none", got)
		}
	})
}

func TestIsolationModeFromString(t *testing.T) {
	tests := []struct {
		name    string
		mode    string
		wantErr bool
	}{
		{"payload", "payload", false},
		{"filesystem", "filesystem", false},
		{"none", "none", false},
		{"unknown", "unknown", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := IsolationModeFromString(tt.mode)
			if (err != nil) != tt.wantErr {
				t.Errorf("IsolationModeFromString() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
