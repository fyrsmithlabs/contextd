package project

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// Common errors.
var (
	ErrProjectNotFound     = errors.New("project not found")
	ErrProjectExists       = errors.New("project already exists")
	ErrInvalidProjectID    = errors.New("invalid project ID")
	ErrInvalidProjectName  = errors.New("invalid project name")
	ErrInvalidProjectPath  = errors.New("invalid project path")
	ErrEmptyProjectID      = errors.New("project ID cannot be empty")
	ErrEmptyProjectName    = errors.New("project name cannot be empty")
	ErrEmptyProjectPath    = errors.New("project path cannot be empty")
)

// Project represents a user's codebase with isolated collections.
type Project struct {
	// ID is the unique project identifier (UUID).
	ID string `json:"id"`

	// Name is the human-readable project name.
	Name string `json:"name"`

	// Path is the filesystem location of the project.
	Path string `json:"path"`

	// CreatedAt is when the project was created.
	CreatedAt time.Time `json:"created_at"`

	// UpdatedAt is when the project was last modified.
	UpdatedAt time.Time `json:"updated_at"`
}

// NewProject creates a new project with a generated UUID.
func NewProject(name, path string) (*Project, error) {
	if name == "" {
		return nil, ErrEmptyProjectName
	}
	if path == "" {
		return nil, ErrEmptyProjectPath
	}

	now := time.Now()
	return &Project{
		ID:        uuid.New().String(),
		Name:      name,
		Path:      path,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// Validate checks if the project has valid fields.
func (p *Project) Validate() error {
	if p.ID == "" {
		return ErrEmptyProjectID
	}
	if _, err := uuid.Parse(p.ID); err != nil {
		return ErrInvalidProjectID
	}
	if p.Name == "" {
		return ErrEmptyProjectName
	}
	if p.Path == "" {
		return ErrEmptyProjectPath
	}
	return nil
}
