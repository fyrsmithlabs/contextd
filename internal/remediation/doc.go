// Package remediation provides error fix pattern storage and retrieval with semantic search.
//
// The package stores structured remediations (error patterns and their solutions)
// with confidence scoring and multi-tenant isolation. Remediations are searchable
// by semantic similarity, enabling AI agents to learn from past error resolutions.
//
// # Security
//
// The package implements defense-in-depth security:
//   - Multi-tenant isolation via scoped collections (org, team, project)
//   - Tenant ID validation and enforcement
//   - Confidence scoring to surface high-quality fixes
//   - Feedback-based confidence adjustment
//   - Support for both legacy single-store and StoreProvider isolation modes
//
// # Usage
//
// Basic remediation recording and search:
//
//	svc := remediation.NewService(cfg, store, logger)
//
//	// Record a new fix pattern
//	rem, err := svc.Record(ctx, &remediation.RecordRequest{
//	    Title:       "Fix nil pointer in auth handler",
//	    Problem:     "Panic when user session is nil",
//	    Symptoms:    []string{"runtime panic", "nil pointer dereference"},
//	    RootCause:   "Missing session validation before access",
//	    Solution:    "Add nil check before accessing session fields",
//	    Category:    remediation.ErrorRuntime,
//	    Scope:       remediation.ScopeProject,
//	    TenantID:    "org-123",
//	    ProjectPath: "/path/to/project",
//	})
//
//	// Search for similar errors
//	results, err := svc.Search(ctx, &remediation.SearchRequest{
//	    Query:         "panic nil pointer session",
//	    Limit:         5,
//	    MinConfidence: 0.6,
//	    TenantID:      "org-123",
//	    Scope:         remediation.ScopeProject,
//	    ProjectPath:   "/path/to/project",
//	})
//
//	// Provide feedback to adjust confidence
//	err = svc.Feedback(ctx, &remediation.FeedbackRequest{
//	    RemediationID: rem.ID,
//	    TenantID:      "org-123",
//	    Rating:        remediation.RatingHelpful,
//	})
//
// # Scoping
//
// Remediations support three scope levels:
//   - ScopeOrg: Visible across entire organization
//   - ScopeTeam: Visible to specific team
//   - ScopeProject: Visible to specific project
//
// Hierarchical search (IncludeHierarchy=true) searches parent scopes:
//   - Project scope searches: project → team → org
//   - Team scope searches: team → org
//   - Org scope searches: org only
//
// # Confidence Scoring
//
// Remediations start with default confidence (0.5) and are adjusted via feedback:
//   - RatingHelpful: +0.1 confidence
//   - RatingNotHelpful: -0.1 confidence
//   - RatingOutdated: -0.2 confidence
//
// Confidence is clamped to [0.1, 1.0] range. Use MinConfidence in search
// to filter low-quality remediations.
package remediation
