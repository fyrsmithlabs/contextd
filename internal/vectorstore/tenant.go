// Package vectorstore provides vector storage implementations.
package vectorstore

import (
	"context"
	"errors"
)

// Tenant isolation error types - fail closed security model.
var (
	// ErrMissingTenant is returned when tenant info is missing from context.
	// This triggers "fail closed" behavior - no empty results, just errors.
	ErrMissingTenant = errors.New("tenant info missing from context")

	// ErrInvalidTenant is returned when tenant identifier is invalid.
	ErrInvalidTenant = errors.New("invalid tenant identifier")
)

// tenantContextKey is the context key for TenantInfo.
type tenantContextKey struct{}

// TenantInfo holds tenant context for filtering and isolation.
//
// Multi-tenancy hierarchy:
//   - TenantID (required): Organization or user identifier
//   - TeamID (optional): Team scope within tenant
//   - ProjectID (optional): Project scope within team
//
// Security: All fields are validated before use in queries.
type TenantInfo struct {
	// TenantID is the organization/user identifier (required).
	TenantID string

	// TeamID is the team identifier (optional).
	TeamID string

	// ProjectID is the project identifier (optional).
	ProjectID string
}

// Validate checks that required fields are present and valid.
func (t *TenantInfo) Validate() error {
	if t.TenantID == "" {
		return ErrInvalidTenant
	}
	return nil
}

// ContextWithTenant adds TenantInfo to a context.
func ContextWithTenant(ctx context.Context, tenant *TenantInfo) context.Context {
	return context.WithValue(ctx, tenantContextKey{}, tenant)
}

// TenantFromContext extracts TenantInfo from a context.
// Returns ErrMissingTenant if not present - fail closed.
func TenantFromContext(ctx context.Context) (*TenantInfo, error) {
	val := ctx.Value(tenantContextKey{})
	if val == nil {
		return nil, ErrMissingTenant
	}
	tenant, ok := val.(*TenantInfo)
	if !ok || tenant == nil {
		return nil, ErrMissingTenant
	}
	return tenant, nil
}

// MustTenantFromContext extracts TenantInfo from context or panics.
// Use only when tenant presence is guaranteed by middleware.
func MustTenantFromContext(ctx context.Context) *TenantInfo {
	tenant, err := TenantFromContext(ctx)
	if err != nil {
		panic("tenant info required but missing from context")
	}
	return tenant
}

// HasTenant checks if TenantInfo is present in context without error.
func HasTenant(ctx context.Context) bool {
	_, err := TenantFromContext(ctx)
	return err == nil
}

// TenantMetadata returns tenant info as a metadata map for document storage.
func (t *TenantInfo) TenantMetadata() map[string]interface{} {
	meta := map[string]interface{}{
		"tenant_id": t.TenantID,
	}
	if t.TeamID != "" {
		meta["team_id"] = t.TeamID
	}
	if t.ProjectID != "" {
		meta["project_id"] = t.ProjectID
	}
	return meta
}

// TenantFilter returns filter conditions for queries.
// Returns conditions that match this tenant's scope.
func (t *TenantInfo) TenantFilter() map[string]interface{} {
	filter := map[string]interface{}{
		"tenant_id": t.TenantID,
	}
	// Only add team/project filters if specified
	if t.TeamID != "" {
		filter["team_id"] = t.TeamID
	}
	if t.ProjectID != "" {
		filter["project_id"] = t.ProjectID
	}
	return filter
}
