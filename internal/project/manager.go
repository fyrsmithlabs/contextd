package project

import (
	"context"
	"fmt"
	"sync"
)

// Manager provides CRUD operations for projects.
type Manager interface {
	// Create creates a new project with the given name and path.
	Create(ctx context.Context, name, path string) (*Project, error)

	// Get retrieves a project by ID.
	Get(ctx context.Context, id string) (*Project, error)

	// List returns all projects.
	List(ctx context.Context) ([]*Project, error)

	// Delete removes a project by ID.
	Delete(ctx context.Context, id string) error

	// GetByPath finds a project by its filesystem path.
	GetByPath(ctx context.Context, path string) (*Project, error)
}

// manager implements Manager with in-memory storage.
type manager struct {
	mu       sync.RWMutex
	projects map[string]*Project // id -> project
	byPath   map[string]*Project // path -> project
}

// NewManager creates a new project manager with in-memory storage.
func NewManager() Manager {
	return &manager{
		projects: make(map[string]*Project),
		byPath:   make(map[string]*Project),
	}
}

// Create creates a new project.
func (m *manager) Create(ctx context.Context, name, path string) (*Project, error) {
	if name == "" {
		return nil, ErrInvalidProjectName
	}
	if path == "" {
		return nil, ErrInvalidProjectPath
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if project with this path already exists
	if existing, ok := m.byPath[path]; ok {
		return nil, fmt.Errorf("%w: project %s already exists at path %s", ErrProjectExists, existing.ID, path)
	}

	// Create new project
	project, err := NewProject(name, path)
	if err != nil {
		return nil, fmt.Errorf("failed to create project: %w", err)
	}

	// Store in maps
	m.projects[project.ID] = project
	m.byPath[project.Path] = project

	return project, nil
}

// Get retrieves a project by ID.
func (m *manager) Get(ctx context.Context, id string) (*Project, error) {
	if id == "" {
		return nil, ErrInvalidProjectID
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	project, ok := m.projects[id]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrProjectNotFound, id)
	}

	return project, nil
}

// List returns all projects.
func (m *manager) List(ctx context.Context) ([]*Project, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	projects := make([]*Project, 0, len(m.projects))
	for _, p := range m.projects {
		projects = append(projects, p)
	}

	return projects, nil
}

// Delete removes a project by ID.
func (m *manager) Delete(ctx context.Context, id string) error {
	if id == "" {
		return ErrInvalidProjectID
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	project, ok := m.projects[id]
	if !ok {
		return fmt.Errorf("%w: %s", ErrProjectNotFound, id)
	}

	// Remove from both maps
	delete(m.projects, id)
	delete(m.byPath, project.Path)

	return nil
}

// GetByPath finds a project by its filesystem path.
func (m *manager) GetByPath(ctx context.Context, path string) (*Project, error) {
	if path == "" {
		return nil, ErrInvalidProjectPath
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	project, ok := m.byPath[path]
	if !ok {
		return nil, fmt.Errorf("%w: no project found at path %s", ErrProjectNotFound, path)
	}

	return project, nil
}
