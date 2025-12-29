package tenant

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// Scope defines the hierarchy level for multi-tenant collections.
//
// Deprecated: With the StoreProvider architecture (database-per-project isolation),
// scope is now handled at the store level, not collection level.
// Use vectorstore.StoreProvider.GetProjectStore/GetTeamStore/GetOrgStore instead.
type Scope string

const (
	// ScopeOrg indicates organization-level data (shared across all teams).
	ScopeOrg Scope = "org"
	// ScopeTeam indicates team-level data (shared within a team).
	ScopeTeam Scope = "team"
	// ScopeProject indicates project-level data (isolated per project).
	ScopeProject Scope = "project"
)

// CollectionType represents the type of data stored in a collection.
type CollectionType string

const (
	CollectionMemories      CollectionType = "memories"
	CollectionRemediations  CollectionType = "remediations"
	CollectionCheckpoints   CollectionType = "checkpoints"
	CollectionPolicies      CollectionType = "policies"
	CollectionSkills        CollectionType = "skills"
	CollectionAgents        CollectionType = "agents"
	CollectionSessions      CollectionType = "sessions"
	CollectionCodebase      CollectionType = "codebase"
	CollectionStandards     CollectionType = "coding_standards"
	CollectionRepoStandards CollectionType = "repo_standards"
	CollectionAntiPatterns  CollectionType = "anti_patterns"
	CollectionFeedback      CollectionType = "feedback"
)

// Common errors.
var (
	ErrInvalidScope      = errors.New("invalid scope")
	ErrInvalidTenantID   = errors.New("invalid tenant ID")
	ErrInvalidTeamID     = errors.New("invalid team ID")
	ErrInvalidProjectID  = errors.New("invalid project ID")
	ErrInvalidCollection = errors.New("invalid collection type")
	ErrAccessDenied      = errors.New("access denied")
)

// CollectionRouter routes requests to the appropriate collection based on tenant scope.
//
// Deprecated: With StoreProvider architecture, collection routing is no longer needed.
// StoreProvider handles database-per-project isolation, so services use simple collection
// names ("memories", "remediations", "codebase") within their scoped store.
//
// Migration guide:
//   - Instead of: router.GetCollectionName(ScopeProject, CollectionMemories, tenant, team, project)
//   - Use: stores.GetProjectStore(ctx, tenant, team, project) then simple "memories" collection
//
// This interface is retained for backward compatibility but will be removed in a future version.
type CollectionRouter interface {
	// GetCollectionName returns the collection name for the given scope and identifiers.
	GetCollectionName(scope Scope, collectionType CollectionType, tenantID, teamID, projectID string) (string, error)

	// ValidateAccess verifies that the session has access to the specified collection.
	ValidateAccess(sessionTenantID, sessionTeamID, sessionProjectID, collectionName string) error

	// GetSearchCollections returns collections to search based on scope hierarchy.
	GetSearchCollections(scope Scope, collectionType CollectionType, tenantID, teamID, projectID string) ([]string, error)
}

// router implements CollectionRouter.
type router struct {
	strictMode bool
}

// NewRouter creates a new collection router.
//
// Deprecated: Use vectorstore.StoreProvider instead. StoreProvider provides
// database-per-project isolation which eliminates the need for collection routing.
func NewRouter(strictMode bool) CollectionRouter {
	return &router{
		strictMode: strictMode,
	}
}

// GetCollectionName returns the collection name following the spec.
func (r *router) GetCollectionName(scope Scope, collectionType CollectionType, tenantID, teamID, projectID string) (string, error) {
	// Validate inputs
	if tenantID == "" {
		return "", ErrInvalidTenantID
	}
	if !isValidIdentifier(string(collectionType)) {
		return "", ErrInvalidCollection
	}

	switch scope {
	case ScopeOrg:
		// org_{type}
		return fmt.Sprintf("org_%s", collectionType), nil

	case ScopeTeam:
		// {team}_{type}
		if teamID == "" {
			return "", ErrInvalidTeamID
		}
		if !isValidIdentifier(teamID) {
			return "", ErrInvalidTeamID
		}
		return fmt.Sprintf("%s_%s", teamID, collectionType), nil

	case ScopeProject:
		// {team}_{project}_{type}
		if teamID == "" {
			return "", ErrInvalidTeamID
		}
		if projectID == "" {
			return "", ErrInvalidProjectID
		}
		if !isValidIdentifier(teamID) {
			return "", ErrInvalidTeamID
		}
		if !isValidIdentifier(projectID) {
			return "", ErrInvalidProjectID
		}
		return fmt.Sprintf("%s_%s_%s", teamID, projectID, collectionType), nil

	default:
		return "", ErrInvalidScope
	}
}

// ValidateAccess is a stub implementation - always returns nil for now.
func (r *router) ValidateAccess(sessionTenantID, sessionTeamID, sessionProjectID, collectionName string) error {
	// Stub: Full implementation pending
	return nil
}

// GetSearchCollections returns collections to search based on scope hierarchy.
func (r *router) GetSearchCollections(scope Scope, collectionType CollectionType, tenantID, teamID, projectID string) ([]string, error) {
	var collections []string

	switch scope {
	case ScopeProject:
		// Project → Team → Org
		projColl, err := r.GetCollectionName(ScopeProject, collectionType, tenantID, teamID, projectID)
		if err == nil {
			collections = append(collections, projColl)
		}
		teamColl, err := r.GetCollectionName(ScopeTeam, collectionType, tenantID, teamID, "")
		if err == nil {
			collections = append(collections, teamColl)
		}
		orgColl, err := r.GetCollectionName(ScopeOrg, collectionType, tenantID, "", "")
		if err == nil {
			collections = append(collections, orgColl)
		}

	case ScopeTeam:
		// Team → Org
		teamColl, err := r.GetCollectionName(ScopeTeam, collectionType, tenantID, teamID, "")
		if err == nil {
			collections = append(collections, teamColl)
		}
		orgColl, err := r.GetCollectionName(ScopeOrg, collectionType, tenantID, "", "")
		if err == nil {
			collections = append(collections, orgColl)
		}

	case ScopeOrg:
		// Org only
		orgColl, err := r.GetCollectionName(ScopeOrg, collectionType, tenantID, "", "")
		if err == nil {
			collections = append(collections, orgColl)
		}

	default:
		return nil, ErrInvalidScope
	}

	return collections, nil
}

// isValidIdentifier checks if a string is a valid collection identifier.
// Must be lowercase alphanumeric with underscores only.
func isValidIdentifier(s string) bool {
	if s == "" {
		return false
	}
	// Must match: ^[a-z0-9_]+$
	validPattern := regexp.MustCompile(`^[a-z0-9_]+$`)
	return validPattern.MatchString(strings.ToLower(s))
}
