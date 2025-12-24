// Package project provides multi-project isolation for contextd-v2.
//
// Project Representation:
//
// Each project represents a user's codebase with:
//   - Unique project ID (UUID)
//   - Project name (user-friendly)
//   - Project path (filesystem location)
//   - Isolated collections in Qdrant
//
// Collection Naming:
//
// Each project gets isolated collections:
//   - {project_id}_memories
//   - {project_id}_checkpoints
//   - {project_id}_remediations
//   - {project_id}_sessions
//
// This provides physical isolation between user projects.
//
// Manager Interface:
//
// The Manager provides CRUD operations for projects:
//   - Create: Create new project with unique ID
//   - Get: Retrieve project by ID
//   - List: List all projects
//   - Delete: Remove project and associated collections
//   - GetByPath: Find project by filesystem path
package project
