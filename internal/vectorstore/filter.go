// Package vectorstore provides vector storage implementations.
package vectorstore

import "errors"

// tenantFilterKeys are keys that cannot be in user filters (security).
var tenantFilterKeys = []string{"tenant_id", "team_id", "project_id"}

// ErrTenantFilterInUserFilters indicates user tried to inject tenant fields.
var ErrTenantFilterInUserFilters = errors.New("user filters cannot contain tenant fields")

// ApplyTenantFilters merges user filters with tenant filters, enforcing security.
//
// This function ensures tenant filters (from isolation layer) always win and
// rejects attempts to inject tenant fields via user filters.
//
// Parameters:
//   - userFilters: User-provided query filters (may be nil)
//   - tenantFilters: Tenant filters from isolation layer (may be nil)
//
// Returns error if userFilters contains tenant_id, team_id, or project_id.
func ApplyTenantFilters(userFilters, tenantFilters map[string]interface{}) (map[string]interface{}, error) {
	// Fast path: both nil
	if userFilters == nil && tenantFilters == nil {
		return nil, nil
	}

	// Fast path: no user filters, return tenant filters directly (no allocation)
	if userFilters == nil {
		return tenantFilters, nil
	}

	// Security check: reject user filters containing tenant fields
	for _, key := range tenantFilterKeys {
		if _, exists := userFilters[key]; exists {
			return nil, ErrTenantFilterInUserFilters
		}
	}

	// Fast path: no tenant filters, return copy of user filters
	if tenantFilters == nil {
		result := make(map[string]interface{}, len(userFilters))
		for k, v := range userFilters {
			result[k] = v
		}
		return result, nil
	}

	// Merge: start with user filters, then apply tenant filters (tenant wins)
	result := make(map[string]interface{}, len(userFilters)+len(tenantFilters))
	for k, v := range userFilters {
		result[k] = v
	}
	for k, v := range tenantFilters {
		result[k] = v
	}

	return result, nil
}

// MergeFilters combines two filter maps, with override taking precedence.
//
// Deprecated: Use ApplyTenantFilters for tenant-aware merging with security validation.
// This function is kept for backward compatibility but does not enforce tenant security.
func MergeFilters(base, override map[string]interface{}) map[string]interface{} {
	if base == nil && override == nil {
		return nil
	}
	if base == nil {
		return override
	}
	if override == nil {
		return base
	}

	result := make(map[string]interface{}, len(base)+len(override))
	for k, v := range base {
		result[k] = v
	}
	for k, v := range override {
		result[k] = v
	}

	return result
}

// FilterBuilder provides a fluent interface for building query filters.
type FilterBuilder struct {
	filters map[string]interface{}
}

// NewFilterBuilder creates a new FilterBuilder.
func NewFilterBuilder() *FilterBuilder {
	return &FilterBuilder{
		filters: make(map[string]interface{}),
	}
}

// With adds a key-value pair to the filter.
func (b *FilterBuilder) With(key string, value interface{}) *FilterBuilder {
	b.filters[key] = value
	return b
}

// WithTenant adds tenant filters from TenantInfo.
func (b *FilterBuilder) WithTenant(tenant *TenantInfo) *FilterBuilder {
	if tenant == nil {
		return b
	}
	for k, v := range tenant.TenantFilter() {
		b.filters[k] = v
	}
	return b
}

// WithMap merges an existing filter map.
func (b *FilterBuilder) WithMap(m map[string]interface{}) *FilterBuilder {
	for k, v := range m {
		b.filters[k] = v
	}
	return b
}

// Build returns the constructed filter map.
func (b *FilterBuilder) Build() map[string]interface{} {
	if len(b.filters) == 0 {
		return nil
	}
	return b.filters
}

// MetadataBuilder provides a fluent interface for building document metadata.
type MetadataBuilder struct {
	metadata map[string]interface{}
}

// NewMetadataBuilder creates a new MetadataBuilder.
func NewMetadataBuilder() *MetadataBuilder {
	return &MetadataBuilder{
		metadata: make(map[string]interface{}),
	}
}

// With adds a key-value pair to the metadata.
func (b *MetadataBuilder) With(key string, value interface{}) *MetadataBuilder {
	b.metadata[key] = value
	return b
}

// WithTenant adds tenant metadata from TenantInfo.
func (b *MetadataBuilder) WithTenant(tenant *TenantInfo) *MetadataBuilder {
	if tenant == nil {
		return b
	}
	for k, v := range tenant.TenantMetadata() {
		b.metadata[k] = v
	}
	return b
}

// WithMap merges an existing metadata map.
func (b *MetadataBuilder) WithMap(m map[string]interface{}) *MetadataBuilder {
	for k, v := range m {
		b.metadata[k] = v
	}
	return b
}

// Build returns the constructed metadata map.
func (b *MetadataBuilder) Build() map[string]interface{} {
	if len(b.metadata) == 0 {
		return nil
	}
	return b.metadata
}

// RequiredTenantFields are the fields that must be present for tenant isolation.
var RequiredTenantFields = []string{"tenant_id"}

// OptionalTenantFields are additional scope fields that may be present.
var OptionalTenantFields = []string{"team_id", "project_id"}

// ValidateFilterHasTenant checks that a filter map contains required tenant fields.
// Returns ErrMissingTenant if tenant_id is not present or not a string.
// Returns ErrInvalidTenant if tenant_id is empty.
func ValidateFilterHasTenant(filters map[string]interface{}) error {
	if filters == nil {
		return ErrMissingTenant
	}
	val, ok := filters["tenant_id"]
	if !ok {
		return ErrMissingTenant
	}
	tid, ok := val.(string)
	if !ok {
		return ErrInvalidTenant // tenant_id must be a string
	}
	if tid == "" {
		return ErrInvalidTenant
	}
	return nil
}

// ExtractTenantFromFilters creates a TenantInfo from filter map.
// Returns ErrMissingTenant if required fields are missing.
// Returns ErrInvalidTenant if tenant_id is not a valid string.
func ExtractTenantFromFilters(filters map[string]interface{}) (*TenantInfo, error) {
	if err := ValidateFilterHasTenant(filters); err != nil {
		return nil, err
	}

	// Safe type assertion - ValidateFilterHasTenant already verified this is a string
	tid, ok := filters["tenant_id"].(string)
	if !ok {
		return nil, ErrInvalidTenant
	}

	tenant := &TenantInfo{
		TenantID: tid,
	}

	if teamID, ok := filters["team_id"].(string); ok {
		tenant.TeamID = teamID
	}
	if projectID, ok := filters["project_id"].(string); ok {
		tenant.ProjectID = projectID
	}

	return tenant, nil
}
