package mcp

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"
)

// notifyCollectionUpdated tells subscribed swarm members that a project's
// resource collection changed (kind ∈ "memories","checkpoints","remediations").
//
// This powers the agent-swarm coordination mechanism: when several agents
// (MCP clients) share a single contextd server over the Streamable HTTP
// transport (stateful sessions), one agent recording shared knowledge should
// prompt the others to re-read the affected collection. Agents express
// interest by calling resources/subscribe on a collection URI; the go-sdk
// tracks those subscriptions per server session, and ResourceUpdated fans the
// notifications/resources/updated message out to exactly the sessions
// subscribed to the matching URI.
//
// Callers (wired separately by the record handlers, not in this file):
//   - memory_record      → notifyCollectionUpdated(ctx, projectID, "memories")
//   - remediation_record → notifyCollectionUpdated(ctx, projectID, "remediations")
//   - checkpoint_save     → notifyCollectionUpdated(ctx, projectID, "checkpoints")
//
// Tenant isolation: the URI embeds the project_id, so only agents subscribed
// to that project's collection are notified — cross-tenant agents never see
// another tenant's updates.
//
// Delivery is best-effort: notification failures are logged but never returned
// to (or surfaced as an error for) the caller, because a failed re-read prompt
// must not fail the underlying record operation. The function returns nothing
// and never panics; a nil underlying MCP server is treated as a no-op.
func (s *Server) notifyCollectionUpdated(ctx context.Context, projectID, kind string) {
	if s == nil || s.mcp == nil {
		return
	}

	// NOTE: build the URI inline. A collectionResourceURI helper is defined in
	// another file in this package; duplicating it here would fail to compile.
	uri := fmt.Sprintf("contextd://%s/%s", projectID, kind)

	if err := s.mcp.ResourceUpdated(ctx, &mcp.ResourceUpdatedNotificationParams{URI: uri}); err != nil {
		// Best-effort: log and move on. The record already succeeded; failing
		// to notify subscribers should not propagate to the caller.
		if s.logger != nil {
			s.logger.Warn("failed to notify swarm of collection update",
				zap.String("uri", uri),
				zap.String("project_id", projectID),
				zap.String("kind", kind),
				zap.Error(err),
			)
		}
		return
	}

	if s.logger != nil {
		s.logger.Debug("notified swarm of collection update",
			zap.String("uri", uri),
			zap.String("project_id", projectID),
			zap.String("kind", kind),
		)
	}
}
