// Package services provides centralized service registry for contextd.
//
// Registry pattern for accessing all core services (checkpoint, memory,
// remediation, repository, troubleshoot, hooks, compression, vectorstore).
// Use NewRegistry() to create a registry with service instances, then
// accessor methods to retrieve individual services.
//
// See CLAUDE.md Architecture section for service layer overview.
package services
