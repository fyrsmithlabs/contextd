// Package registry manages tenant/project registration with UUID tracking.
//
// The registry provides:
//   - Human-readable names to UUID mapping for future database migration
//   - Path sanitization for filesystem safety
//   - Registration validation and persistence
//
// Directory structure:
//
//	~/.config/contextd/vectorstore/
//	├── {tenant}/                      ← org level
//	│   ├── {project}/                 ← direct project
//	│   │   └── {collections}
//	│   ├── {team}/                    ← team level
//	│   │   └── {project}/             ← team-scoped project
//	│   │       └── {collections}
//	│   ├── memories/                  ← org-shared
//	│   └── remediations/              ← org-shared
package registry

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Errors for registry operations.
var (
	ErrTenantNotFound    = errors.New("tenant not found")
	ErrProjectNotFound   = errors.New("project not found")
	ErrTeamNotFound      = errors.New("team not found")
	ErrInvalidName       = errors.New("invalid name: must be alphanumeric with hyphens/underscores")
	ErrPathTraversal     = errors.New("path traversal detected")
	ErrRegistryCorrupted = errors.New("registry file corrupted")
)

// namePattern validates tenant/project/team names.
// Allows alphanumeric, hyphens, underscores, and dots.
var namePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]*$`)

// Entry represents a registered entity (tenant, team, or project).
type Entry struct {
	UUID      string    `json:"uuid"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

// RegistryData is the persisted registry structure.
type RegistryData struct {
	Version  int               `json:"version"`
	Tenants  map[string]*Entry `json:"tenants"`  // key: tenant name
	Teams    map[string]*Entry `json:"teams"`    // key: tenant/team
	Projects map[string]*Entry `json:"projects"` // key: tenant/project or tenant/team/project
}

// Registry manages tenant/project registration and path resolution.
type Registry struct {
	mu       sync.RWMutex
	basePath string // base vectorstore path
	data     *RegistryData
	filePath string // path to registry.json
}

// NewRegistry creates a new registry at the specified base path.
func NewRegistry(basePath string) (*Registry, error) {
	if basePath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		basePath = filepath.Join(home, ".config", "contextd", "vectorstore")
	}

	r := &Registry{
		basePath: basePath,
		filePath: filepath.Join(basePath, "registry.json"),
		data: &RegistryData{
			Version:  1,
			Tenants:  make(map[string]*Entry),
			Teams:    make(map[string]*Entry),
			Projects: make(map[string]*Entry),
		},
	}

	// Ensure base directory exists
	if err := os.MkdirAll(basePath, 0700); err != nil {
		return nil, fmt.Errorf("failed to create base directory: %w", err)
	}

	// Load existing registry
	if err := r.load(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to load registry: %w", err)
	}

	return r, nil
}

// ValidateName checks if a name is safe for filesystem paths.
func ValidateName(name string) error {
	if name == "" {
		return ErrInvalidName
	}
	if len(name) > 255 {
		return fmt.Errorf("%w: name too long (max 255)", ErrInvalidName)
	}
	if !namePattern.MatchString(name) {
		return ErrInvalidName
	}

	// Explicit path traversal checks
	// Check for . and .. (current/parent directory)
	if name == "." || name == ".." {
		return ErrPathTraversal
	}

	// Check for path separators (cross-platform)
	for _, c := range name {
		if c == '/' || c == '\\' || c == '\x00' {
			return ErrPathTraversal
		}
	}

	// Verify Clean doesn't modify the path (catches edge cases)
	if filepath.Clean(name) != name {
		return ErrPathTraversal
	}

	return nil
}

// RegisterTenant registers a new tenant or returns existing entry.
func (r *Registry) RegisterTenant(name string) (*Entry, error) {
	if err := ValidateName(name); err != nil {
		return nil, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Return existing
	if entry, ok := r.data.Tenants[name]; ok {
		return entry, nil
	}

	// Create new
	entry := &Entry{
		UUID:      uuid.New().String(),
		Name:      name,
		CreatedAt: time.Now().UTC(),
	}
	r.data.Tenants[name] = entry

	// Create tenant directory
	tenantPath := filepath.Join(r.basePath, name)
	if err := os.MkdirAll(tenantPath, 0700); err != nil {
		return nil, fmt.Errorf("failed to create tenant directory: %w", err)
	}

	// Persist
	if err := r.save(); err != nil {
		return nil, err
	}

	return entry, nil
}

// RegisterTeam registers a new team under a tenant.
func (r *Registry) RegisterTeam(tenant, team string) (*Entry, error) {
	if err := ValidateName(tenant); err != nil {
		return nil, fmt.Errorf("tenant: %w", err)
	}
	if err := ValidateName(team); err != nil {
		return nil, fmt.Errorf("team: %w", err)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Ensure tenant exists
	if _, ok := r.data.Tenants[tenant]; !ok {
		return nil, ErrTenantNotFound
	}

	key := tenant + "/" + team

	// Return existing
	if entry, ok := r.data.Teams[key]; ok {
		return entry, nil
	}

	// Create new
	entry := &Entry{
		UUID:      uuid.New().String(),
		Name:      team,
		CreatedAt: time.Now().UTC(),
	}
	r.data.Teams[key] = entry

	// Create team directory
	teamPath := filepath.Join(r.basePath, tenant, team)
	if err := os.MkdirAll(teamPath, 0700); err != nil {
		return nil, fmt.Errorf("failed to create team directory: %w", err)
	}

	// Persist
	if err := r.save(); err != nil {
		return nil, err
	}

	return entry, nil
}

// RegisterProject registers a new project.
// If team is empty, registers under tenant directly.
// If team is set, registers under tenant/team.
func (r *Registry) RegisterProject(tenant, team, project string) (*Entry, error) {
	if err := ValidateName(tenant); err != nil {
		return nil, fmt.Errorf("tenant: %w", err)
	}
	if team != "" {
		if err := ValidateName(team); err != nil {
			return nil, fmt.Errorf("team: %w", err)
		}
	}
	if err := ValidateName(project); err != nil {
		return nil, fmt.Errorf("project: %w", err)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Ensure tenant exists
	if _, ok := r.data.Tenants[tenant]; !ok {
		return nil, ErrTenantNotFound
	}

	// Ensure team exists if specified
	if team != "" {
		teamKey := tenant + "/" + team
		if _, ok := r.data.Teams[teamKey]; !ok {
			return nil, ErrTeamNotFound
		}
	}

	// Build key
	var key string
	var projectPath string
	if team != "" {
		key = tenant + "/" + team + "/" + project
		projectPath = filepath.Join(r.basePath, tenant, team, project)
	} else {
		key = tenant + "/" + project
		projectPath = filepath.Join(r.basePath, tenant, project)
	}

	// Return existing
	if entry, ok := r.data.Projects[key]; ok {
		return entry, nil
	}

	// Create new
	entry := &Entry{
		UUID:      uuid.New().String(),
		Name:      project,
		CreatedAt: time.Now().UTC(),
	}
	r.data.Projects[key] = entry

	// Create project directory
	if err := os.MkdirAll(projectPath, 0700); err != nil {
		return nil, fmt.Errorf("failed to create project directory: %w", err)
	}

	// Persist
	if err := r.save(); err != nil {
		return nil, err
	}

	return entry, nil
}

// GetTenant returns the tenant entry by name.
func (r *Registry) GetTenant(name string) (*Entry, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entry, ok := r.data.Tenants[name]
	if !ok {
		return nil, ErrTenantNotFound
	}
	return entry, nil
}

// GetTeam returns the team entry.
func (r *Registry) GetTeam(tenant, team string) (*Entry, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	key := tenant + "/" + team
	entry, ok := r.data.Teams[key]
	if !ok {
		return nil, ErrTeamNotFound
	}
	return entry, nil
}

// GetProject returns the project entry.
func (r *Registry) GetProject(tenant, team, project string) (*Entry, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var key string
	if team != "" {
		key = tenant + "/" + team + "/" + project
	} else {
		key = tenant + "/" + project
	}

	entry, ok := r.data.Projects[key]
	if !ok {
		return nil, ErrProjectNotFound
	}
	return entry, nil
}

// GetProjectPath returns the filesystem path for a project.
func (r *Registry) GetProjectPath(tenant, team, project string) (string, error) {
	if err := ValidateName(tenant); err != nil {
		return "", fmt.Errorf("tenant: %w", err)
	}
	if team != "" {
		if err := ValidateName(team); err != nil {
			return "", fmt.Errorf("team: %w", err)
		}
	}
	if err := ValidateName(project); err != nil {
		return "", fmt.Errorf("project: %w", err)
	}

	if team != "" {
		return filepath.Join(r.basePath, tenant, team, project), nil
	}
	return filepath.Join(r.basePath, tenant, project), nil
}

// GetTeamPath returns the filesystem path for a team (for shared collections).
func (r *Registry) GetTeamPath(tenant, team string) (string, error) {
	if err := ValidateName(tenant); err != nil {
		return "", fmt.Errorf("tenant: %w", err)
	}
	if err := ValidateName(team); err != nil {
		return "", fmt.Errorf("team: %w", err)
	}
	return filepath.Join(r.basePath, tenant, team), nil
}

// GetOrgPath returns the filesystem path for an org (for org-shared collections).
func (r *Registry) GetOrgPath(tenant string) (string, error) {
	if err := ValidateName(tenant); err != nil {
		return "", fmt.Errorf("tenant: %w", err)
	}
	return filepath.Join(r.basePath, tenant), nil
}

// EnsureProjectExists registers tenant and project if they don't exist.
// This is a convenience method for auto-registration on first use.
func (r *Registry) EnsureProjectExists(tenant, team, project string) error {
	// Register tenant (idempotent)
	if _, err := r.RegisterTenant(tenant); err != nil {
		return err
	}

	// Register team if specified (idempotent)
	if team != "" {
		if _, err := r.RegisterTeam(tenant, team); err != nil {
			return err
		}
	}

	// Register project (idempotent)
	if _, err := r.RegisterProject(tenant, team, project); err != nil {
		return err
	}

	return nil
}

// ListTenants returns all registered tenant names.
func (r *Registry) ListTenants() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.data.Tenants))
	for name := range r.data.Tenants {
		names = append(names, name)
	}
	return names
}

// ListProjects returns all registered projects for a tenant.
func (r *Registry) ListProjects(tenant string) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	prefix := tenant + "/"
	var projects []string
	for key := range r.data.Projects {
		if len(key) > len(prefix) && key[:len(prefix)] == prefix {
			projects = append(projects, key)
		}
	}
	return projects
}

// BasePath returns the base vectorstore path.
func (r *Registry) BasePath() string {
	return r.basePath
}

// load reads the registry from disk.
func (r *Registry) load() error {
	data, err := os.ReadFile(r.filePath)
	if err != nil {
		return err
	}

	var rd RegistryData
	if err := json.Unmarshal(data, &rd); err != nil {
		return fmt.Errorf("%w: %v", ErrRegistryCorrupted, err)
	}

	// Initialize maps if nil (for version upgrades)
	if rd.Tenants == nil {
		rd.Tenants = make(map[string]*Entry)
	}
	if rd.Teams == nil {
		rd.Teams = make(map[string]*Entry)
	}
	if rd.Projects == nil {
		rd.Projects = make(map[string]*Entry)
	}

	r.data = &rd
	return nil
}

// save writes the registry to disk.
func (r *Registry) save() error {
	data, err := json.MarshalIndent(r.data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal registry: %w", err)
	}

	// Write atomically
	tmpPath := r.filePath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write registry: %w", err)
	}

	if err := os.Rename(tmpPath, r.filePath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to rename registry: %w", err)
	}

	return nil
}
