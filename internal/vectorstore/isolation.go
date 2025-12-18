// Package vectorstore provides vector storage implementations.
package vectorstore

import (
	"context"
	"fmt"
)

// IsolationMode defines how tenant isolation is enforced in vector stores.
//
// Implementations determine whether isolation is via:
//   - Payload filtering (single collection, metadata-based filtering)
//   - Filesystem isolation (separate databases per tenant/project)
//   - Hybrid approaches (tiered isolation)
//
// Security: All implementations must enforce fail-closed behavior.
type IsolationMode interface {
	// InjectFilter adds tenant filtering to search options.
	// Must fail with ErrMissingTenant if tenant context is absent.
	InjectFilter(ctx context.Context, filters map[string]interface{}) (map[string]interface{}, error)

	// InjectMetadata adds tenant metadata to documents before storage.
	// Must fail with ErrMissingTenant if tenant context is absent.
	InjectMetadata(ctx context.Context, docs []Document) error

	// ValidateTenant checks that tenant context is present and valid.
	// Returns nil if valid, ErrMissingTenant or ErrInvalidTenant otherwise.
	ValidateTenant(ctx context.Context) error

	// Mode returns the isolation mode name for logging/debugging.
	Mode() string
}

// PayloadIsolation implements IsolationMode using metadata filtering.
//
// In this mode:
//   - All documents in a single collection per type (e.g., "memories")
//   - tenant_id, team_id, project_id stored as document metadata
//   - All queries automatically filtered by tenant context
//   - Missing tenant context = error (fail closed)
//
// Security guarantees:
//   - Mandatory filter injection on all queries
//   - No bypass possible - private methods enforce filtering
//   - Audit-friendly - tenant always in context
type PayloadIsolation struct{}

// NewPayloadIsolation creates a new PayloadIsolation mode.
func NewPayloadIsolation() *PayloadIsolation {
	return &PayloadIsolation{}
}

// InjectFilter adds tenant filters to existing query filters.
func (p *PayloadIsolation) InjectFilter(ctx context.Context, filters map[string]interface{}) (map[string]interface{}, error) {
	tenant, err := TenantFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if err := tenant.Validate(); err != nil {
		return nil, err
	}

	// Merge tenant filters with existing filters
	result := MergeFilters(filters, tenant.TenantFilter())
	return result, nil
}

// InjectMetadata adds tenant metadata to all documents.
func (p *PayloadIsolation) InjectMetadata(ctx context.Context, docs []Document) error {
	tenant, err := TenantFromContext(ctx)
	if err != nil {
		return err
	}
	if err := tenant.Validate(); err != nil {
		return err
	}

	tenantMeta := tenant.TenantMetadata()
	for i := range docs {
		if docs[i].Metadata == nil {
			docs[i].Metadata = make(map[string]interface{})
		}
		// Inject tenant metadata (overwrites if present for security)
		for k, v := range tenantMeta {
			docs[i].Metadata[k] = v
		}
	}
	return nil
}

// ValidateTenant checks tenant context is present and valid.
func (p *PayloadIsolation) ValidateTenant(ctx context.Context) error {
	tenant, err := TenantFromContext(ctx)
	if err != nil {
		return err
	}
	return tenant.Validate()
}

// Mode returns "payload" for this isolation mode.
func (p *PayloadIsolation) Mode() string {
	return "payload"
}

// FilesystemIsolation implements IsolationMode using separate stores.
//
// In this mode:
//   - Each tenant/team/project gets its own database directory
//   - Physical filesystem isolation provides security boundary
//   - No metadata filtering needed (isolation is structural)
//
// Note: This is the legacy mode used by ChromemStoreProvider.
// Consider migration to PayloadIsolation for simpler operations.
type FilesystemIsolation struct{}

// NewFilesystemIsolation creates a new FilesystemIsolation mode.
func NewFilesystemIsolation() *FilesystemIsolation {
	return &FilesystemIsolation{}
}

// InjectFilter is a no-op for filesystem isolation (isolation is structural).
func (f *FilesystemIsolation) InjectFilter(ctx context.Context, filters map[string]interface{}) (map[string]interface{}, error) {
	// Still validate tenant context for consistency
	if err := f.ValidateTenant(ctx); err != nil {
		return nil, err
	}
	// Return original filters unchanged - no metadata filtering needed
	return filters, nil
}

// InjectMetadata is optional for filesystem isolation but adds tenant info for auditability.
func (f *FilesystemIsolation) InjectMetadata(ctx context.Context, docs []Document) error {
	// Still validate tenant context for consistency
	tenant, err := TenantFromContext(ctx)
	if err != nil {
		return err
	}
	if err := tenant.Validate(); err != nil {
		return err
	}

	// Add tenant metadata for audit purposes (not for filtering)
	tenantMeta := tenant.TenantMetadata()
	for i := range docs {
		if docs[i].Metadata == nil {
			docs[i].Metadata = make(map[string]interface{})
		}
		for k, v := range tenantMeta {
			docs[i].Metadata[k] = v
		}
	}
	return nil
}

// ValidateTenant checks tenant context is present and valid.
func (f *FilesystemIsolation) ValidateTenant(ctx context.Context) error {
	tenant, err := TenantFromContext(ctx)
	if err != nil {
		return err
	}
	return tenant.Validate()
}

// Mode returns "filesystem" for this isolation mode.
func (f *FilesystemIsolation) Mode() string {
	return "filesystem"
}

// NoIsolation provides no tenant isolation - for testing only.
//
// WARNING: This mode provides no security guarantees.
// Use only in tests where tenant isolation is not relevant.
type NoIsolation struct{}

// NewNoIsolation creates a new NoIsolation mode (testing only).
func NewNoIsolation() *NoIsolation {
	return &NoIsolation{}
}

// InjectFilter passes through filters unchanged.
func (n *NoIsolation) InjectFilter(ctx context.Context, filters map[string]interface{}) (map[string]interface{}, error) {
	return filters, nil
}

// InjectMetadata is a no-op.
func (n *NoIsolation) InjectMetadata(ctx context.Context, docs []Document) error {
	return nil
}

// ValidateTenant always succeeds.
func (n *NoIsolation) ValidateTenant(ctx context.Context) error {
	return nil
}

// Mode returns "none" for this isolation mode.
func (n *NoIsolation) Mode() string {
	return "none"
}

// Ensure implementations satisfy IsolationMode interface.
var (
	_ IsolationMode = (*PayloadIsolation)(nil)
	_ IsolationMode = (*FilesystemIsolation)(nil)
	_ IsolationMode = (*NoIsolation)(nil)
)

// IsolationModeFromString creates an IsolationMode from a string name.
func IsolationModeFromString(mode string) (IsolationMode, error) {
	switch mode {
	case "payload":
		return NewPayloadIsolation(), nil
	case "filesystem":
		return NewFilesystemIsolation(), nil
	case "none":
		return NewNoIsolation(), nil
	default:
		return nil, fmt.Errorf("unknown isolation mode: %s", mode)
	}
}
