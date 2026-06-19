package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/fyrsmithlabs/contextd/internal/checkpoint"
	"github.com/fyrsmithlabs/contextd/internal/reasoningbank"
	"github.com/fyrsmithlabs/contextd/internal/remediation"
	"github.com/fyrsmithlabs/contextd/internal/sanitize"
)

// resourceScheme is the URI scheme used for all contextd resources.
const resourceScheme = "contextd"

// Resource collection kinds used in URIs.
const (
	kindMemories     = "memories"
	kindMemory       = "memory"
	kindCheckpoints  = "checkpoints"
	kindCheckpoint   = "checkpoint"
	kindRemediation  = "remediation"
	kindRemediations = "remediations"
)

// collectionResourceURI builds a collection-level resource URI of the form
// "contextd://<projectID>/<kind>". It is reused by other files in this package.
func collectionResourceURI(projectID, kind string) string {
	return fmt.Sprintf("%s://%s/%s", resourceScheme, projectID, kind)
}

// itemResourceURI builds an item-level resource URI of the form
// "contextd://<projectID>/<kind>/<id>". It is reused by other files in this package.
func itemResourceURI(projectID, kind, id string) string {
	return fmt.Sprintf("%s://%s/%s/%s", resourceScheme, projectID, kind, id)
}

// parsedResourceURI holds the components extracted from a contextd resource URI.
type parsedResourceURI struct {
	projectID string
	kind      string
	id        string
	query     string
}

// parseResourceURI parses a "contextd://<project_id>/<kind>[/<id>][?query]" URI.
// It returns the parsed components or an error if the URI is malformed. The
// project_id is NOT validated here; callers must validate it with
// sanitize.ValidateProjectID and fail closed on error.
func parseResourceURI(raw string) (parsedResourceURI, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return parsedResourceURI{}, fmt.Errorf("invalid uri: %w", err)
	}
	if u.Scheme != resourceScheme {
		return parsedResourceURI{}, fmt.Errorf("unexpected scheme %q", u.Scheme)
	}

	// For "contextd://<project_id>/<kind>", the project_id lands in u.Host
	// and the remainder ("/<kind>[/<id>]") lands in u.Path.
	projectID := u.Host
	if projectID == "" {
		return parsedResourceURI{}, fmt.Errorf("missing project_id in uri")
	}

	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	// strings.Split on "" yields [""], normalize to empty.
	if len(parts) == 1 && parts[0] == "" {
		parts = nil
	}
	if len(parts) == 0 {
		return parsedResourceURI{}, fmt.Errorf("missing resource kind in uri")
	}

	p := parsedResourceURI{
		projectID: projectID,
		kind:      parts[0],
		query:     u.Query().Get("query"),
	}
	if len(parts) >= 2 {
		p.id = parts[1]
	}
	return p, nil
}

// registerResources registers MCP resources and resource templates that expose
// memories, checkpoints, and remediations as readable resources.
func (s *Server) registerResources() {
	// Static help resource documenting the URI scheme.
	s.mcp.AddResource(&mcp.Resource{
		Name:        "contextd-help",
		URI:         resourceScheme + "://help",
		Title:       "contextd Resource Help",
		Description: "Documentation of the contextd:// resource URI scheme",
		MIMEType:    "application/json",
	}, s.handleHelpResource)

	// Recent memories for a project.
	s.mcp.AddResourceTemplate(&mcp.ResourceTemplate{
		Name:        "contextd-memories",
		URITemplate: resourceScheme + "://{project_id}/memories",
		Title:       "Project Memories",
		Description: "Recent memories (learnings/strategies) for a project",
		MIMEType:    "application/json",
	}, s.handleResource)

	// Single memory by ID.
	s.mcp.AddResourceTemplate(&mcp.ResourceTemplate{
		Name:        "contextd-memory",
		URITemplate: resourceScheme + "://{project_id}/memory/{id}",
		Title:       "Project Memory",
		Description: "A single memory by ID for a project",
		MIMEType:    "application/json",
	}, s.handleResource)

	// Checkpoint list for a project.
	s.mcp.AddResourceTemplate(&mcp.ResourceTemplate{
		Name:        "contextd-checkpoints",
		URITemplate: resourceScheme + "://{project_id}/checkpoints",
		Title:       "Project Checkpoints",
		Description: "Saved context checkpoints for a project",
		MIMEType:    "application/json",
	}, s.handleResource)

	// Single checkpoint by ID.
	s.mcp.AddResourceTemplate(&mcp.ResourceTemplate{
		Name:        "contextd-checkpoint",
		URITemplate: resourceScheme + "://{project_id}/checkpoint/{id}",
		Title:       "Project Checkpoint",
		Description: "A single checkpoint by ID for a project",
		MIMEType:    "application/json",
	}, s.handleResource)

	// Single remediation by ID.
	s.mcp.AddResourceTemplate(&mcp.ResourceTemplate{
		Name:        "contextd-remediation",
		URITemplate: resourceScheme + "://{project_id}/remediation/{id}",
		Title:       "Project Remediation",
		Description: "A single remediation (error fix) by ID for a project",
		MIMEType:    "application/json",
	}, s.handleResource)

	// Remediation search for a project.
	s.mcp.AddResourceTemplate(&mcp.ResourceTemplate{
		Name:        "contextd-remediations",
		URITemplate: resourceScheme + "://{project_id}/remediations{?query}",
		Title:       "Project Remediations",
		Description: "Search remediations (error fixes) for a project via ?query=",
		MIMEType:    "application/json",
	}, s.handleResource)
}

// jsonResource marshals v to JSON and wraps it in a ReadResourceResult for uri.
func jsonResource(uri string, v interface{}) (*mcp.ReadResourceResult, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("marshal resource: %w", err)
	}
	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{
			{
				URI:      uri,
				MIMEType: "application/json",
				Text:     string(data),
			},
		},
	}, nil
}

// handleHelpResource returns a JSON object documenting the URI scheme.
func (s *Server) handleHelpResource(_ context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	uri := req.Params.URI
	help := map[string]interface{}{
		"scheme":      resourceScheme,
		"description": "contextd exposes memories, checkpoints, and remediations as MCP resources. Replace {project_id} with your project identifier.",
		"resources": []map[string]string{
			{"uri": resourceScheme + "://help", "description": "This help document."},
			{"uri": resourceScheme + "://{project_id}/memories", "description": "Recent memories for a project (up to 20)."},
			{"uri": resourceScheme + "://{project_id}/memory/{id}", "description": "A single memory by ID."},
			{"uri": resourceScheme + "://{project_id}/checkpoints", "description": "Checkpoints for a project."},
			{"uri": resourceScheme + "://{project_id}/checkpoint/{id}", "description": "A single checkpoint by ID."},
			{"uri": resourceScheme + "://{project_id}/remediation/{id}", "description": "A single remediation by ID."},
			{"uri": resourceScheme + "://{project_id}/remediations{?query}", "description": "Search remediations; provide ?query=<text>."},
		},
	}
	return jsonResource(uri, help)
}

// handleResource dispatches a templated resource read based on the parsed URI.
// It validates the project_id (fail-closed) and routes to the appropriate
// service-backed handler.
func (s *Server) handleResource(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	uri := req.Params.URI

	parsed, err := parseResourceURI(uri)
	if err != nil {
		return nil, mcp.ResourceNotFoundError(uri)
	}

	// Validate project_id (fail-closed): invalid project -> not found.
	if err := sanitize.ValidateProjectID(parsed.projectID); err != nil {
		return nil, mcp.ResourceNotFoundError(uri)
	}

	// Establish tenant context: tenantID = projectID, teamID = "", projectID = projectID.
	ctx, err = withTenantContext(ctx, parsed.projectID, "", parsed.projectID)
	if err != nil {
		return nil, mcp.ResourceNotFoundError(uri)
	}

	switch parsed.kind {
	case kindMemories:
		return s.readMemories(ctx, uri, parsed.projectID)
	case kindMemory:
		if parsed.id == "" {
			return nil, mcp.ResourceNotFoundError(uri)
		}
		return s.readMemory(ctx, uri, parsed.projectID, parsed.id)
	case kindCheckpoints:
		return s.readCheckpoints(ctx, uri, parsed.projectID)
	case kindCheckpoint:
		if parsed.id == "" {
			return nil, mcp.ResourceNotFoundError(uri)
		}
		return s.readCheckpoint(ctx, uri, parsed.projectID, parsed.id)
	case kindRemediation:
		if parsed.id == "" {
			return nil, mcp.ResourceNotFoundError(uri)
		}
		return s.readRemediation(ctx, uri, parsed.projectID, parsed.id)
	case kindRemediations:
		return s.readRemediations(ctx, uri, parsed.projectID, parsed.query)
	default:
		return nil, mcp.ResourceNotFoundError(uri)
	}
}

// readMemories returns recent memories for a project (limit 20, offset 0).
func (s *Server) readMemories(ctx context.Context, uri, projectID string) (*mcp.ReadResourceResult, error) {
	memories, err := s.reasoningbankSvc.ListMemories(ctx, projectID, 20, 0)
	if err != nil {
		return nil, mcp.ResourceNotFoundError(uri)
	}

	out := make([]map[string]interface{}, 0, len(memories))
	for i := range memories {
		out = append(out, s.memoryToMap(&memories[i]))
	}
	return jsonResource(uri, map[string]interface{}{
		"project_id": projectID,
		"count":      len(out),
		"memories":   out,
	})
}

// readMemory returns a single memory by ID.
func (s *Server) readMemory(ctx context.Context, uri, projectID, id string) (*mcp.ReadResourceResult, error) {
	mem, err := s.reasoningbankSvc.GetByProjectID(ctx, projectID, id)
	if err != nil || mem == nil {
		return nil, mcp.ResourceNotFoundError(uri)
	}
	return jsonResource(uri, s.memoryToMap(mem))
}

// readCheckpoints returns the checkpoint list for a project.
func (s *Server) readCheckpoints(ctx context.Context, uri, projectID string) (*mcp.ReadResourceResult, error) {
	checkpoints, err := s.checkpointSvc.List(ctx, &checkpoint.ListRequest{
		TenantID:    projectID,
		ProjectID:   projectID,
		ProjectPath: projectID,
		Limit:       20,
	})
	if err != nil {
		return nil, mcp.ResourceNotFoundError(uri)
	}

	out := make([]map[string]interface{}, 0, len(checkpoints))
	for _, cp := range checkpoints {
		out = append(out, s.checkpointToMap(cp))
	}
	return jsonResource(uri, map[string]interface{}{
		"project_id":  projectID,
		"count":       len(out),
		"checkpoints": out,
	})
}

// readCheckpoint returns a single checkpoint by ID.
func (s *Server) readCheckpoint(ctx context.Context, uri, projectID, id string) (*mcp.ReadResourceResult, error) {
	cp, err := s.checkpointSvc.Get(ctx, projectID, "", projectID, id)
	if err != nil || cp == nil {
		return nil, mcp.ResourceNotFoundError(uri)
	}
	return jsonResource(uri, s.checkpointToMap(cp))
}

// readRemediation returns a single remediation by ID (project scope).
func (s *Server) readRemediation(ctx context.Context, uri, projectID, id string) (*mcp.ReadResourceResult, error) {
	rem, err := s.remediationSvc.Get(ctx, projectID, id)
	if err != nil || rem == nil {
		return nil, mcp.ResourceNotFoundError(uri)
	}
	return jsonResource(uri, s.remediationToMap(rem))
}

// readRemediations searches remediations for a project. If query is empty, it
// returns an empty contents list with an explanatory JSON note rather than
// calling Search with an empty query.
func (s *Server) readRemediations(ctx context.Context, uri, projectID, query string) (*mcp.ReadResourceResult, error) {
	if strings.TrimSpace(query) == "" {
		return jsonResource(uri, map[string]interface{}{
			"project_id":   projectID,
			"count":        0,
			"note":         "provide a non-empty ?query= parameter to search remediations",
			"remediations": []interface{}{},
		})
	}

	results, err := s.remediationSvc.Search(ctx, &remediation.SearchRequest{
		Query:       query,
		TenantID:    projectID,
		Scope:       remediation.ScopeProject,
		ProjectPath: projectID,
		Limit:       20,
	})
	if err != nil {
		return nil, mcp.ResourceNotFoundError(uri)
	}

	out := make([]map[string]interface{}, 0, len(results))
	for _, sr := range results {
		m := s.remediationToMap(&sr.Remediation)
		m["score"] = sr.Score
		out = append(out, m)
	}
	return jsonResource(uri, map[string]interface{}{
		"project_id":   projectID,
		"query":        query,
		"count":        len(out),
		"remediations": out,
	})
}

// memoryToMap converts a memory to a JSON-serializable map with scrubbed free text.
func (s *Server) memoryToMap(m *reasoningbank.Memory) map[string]interface{} {
	return map[string]interface{}{
		"id":          m.ID,
		"title":       m.Title,
		"description": s.scrubber.Scrub(m.Description).Scrubbed,
		"content":     s.scrubber.Scrub(m.Content).Scrubbed,
		"outcome":     string(m.Outcome),
		"confidence":  m.Confidence,
		"tags":        m.Tags,
		"created_at":  m.CreatedAt,
	}
}

// checkpointToMap converts a checkpoint to a JSON-serializable map with scrubbed free text.
func (s *Server) checkpointToMap(cp *checkpoint.Checkpoint) map[string]interface{} {
	return map[string]interface{}{
		"id":          cp.ID,
		"session_id":  cp.SessionID,
		"name":        cp.Name,
		"description": s.scrubber.Scrub(cp.Description).Scrubbed,
		"summary":     s.scrubber.Scrub(cp.Summary).Scrubbed,
		"context":     s.scrubber.Scrub(cp.Context).Scrubbed,
		"token_count": cp.TokenCount,
		"created_at":  cp.CreatedAt,
	}
}

// remediationToMap converts a remediation to a JSON-serializable map with scrubbed free text.
func (s *Server) remediationToMap(r *remediation.Remediation) map[string]interface{} {
	return map[string]interface{}{
		"id":         r.ID,
		"title":      r.Title,
		"problem":    s.scrubber.Scrub(r.Problem).Scrubbed,
		"root_cause": s.scrubber.Scrub(r.RootCause).Scrubbed,
		"solution":   s.scrubber.Scrub(r.Solution).Scrubbed,
		"category":   string(r.Category),
		"confidence": r.Confidence,
		"tags":       r.Tags,
	}
}
