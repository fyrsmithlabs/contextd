package mcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/fyrsmithlabs/contextd/internal/checkpoint"
)

// clientLog sends a best-effort log message to the connected client over the
// MCP logging channel. It is nil-safe (no-op if there is no session) and never
// returns an error: client-facing logs must never break a tool call. The SDK
// only forwards messages once the client has set a logging level, so this is
// silently dropped when the client has not opted in.
func (s *Server) clientLog(ctx context.Context, sess *mcp.ServerSession, level mcp.LoggingLevel, msg string) {
	if sess == nil {
		return
	}
	_ = sess.Log(ctx, &mcp.LoggingMessageParams{
		Level:  level,
		Logger: "contextd",
		Data:   msg,
	})
}

// chooseCheckpointViaElicit resolves which checkpoint to resume when the caller
// did not supply a checkpoint_id.
//
// Return contract:
//   - (id, "", nil)        → use this checkpoint id
//   - ("", listMsg, nil)   → could not auto-resolve; caller should present
//     listMsg so the user/agent can re-invoke with an explicit checkpoint_id
//   - ("", "", err)        → hard error (nothing to resume / cannot list)
//
// When multiple checkpoints exist it asks the client to choose via MCP
// elicitation; if the client does not support elicitation (or declines), it
// falls back to returning a human-readable list.
func (s *Server) chooseCheckpointViaElicit(ctx context.Context, sess *mcp.ServerSession, tenantID string) (string, string, error) {
	cps, err := s.checkpointSvc.List(ctx, &checkpoint.ListRequest{TenantID: tenantID, Limit: 20})
	if err != nil {
		return "", "", fmt.Errorf("checkpoint_id is required and checkpoints could not be listed: %w", err)
	}
	if len(cps) == 0 {
		return "", "", fmt.Errorf("no checkpoints found for tenant %q; nothing to resume", tenantID)
	}
	if len(cps) == 1 {
		return cps[0].ID, "", nil
	}

	listMsg := s.formatCheckpointList(cps)

	// No session → cannot elicit; return the list for manual selection.
	if sess == nil {
		return "", listMsg, nil
	}

	ids := make([]any, 0, len(cps))
	for _, cp := range cps {
		ids = append(ids, cp.ID)
	}
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"checkpoint_id": map[string]any{
				"type":        "string",
				"enum":        ids,
				"description": "ID of the checkpoint to resume",
			},
		},
		"required": []string{"checkpoint_id"},
	}

	res, err := sess.Elicit(ctx, &mcp.ElicitParams{
		Message:         "Multiple checkpoints exist. Choose one to resume:\n" + listMsg,
		RequestedSchema: schema,
	})
	if err != nil {
		// Client does not support elicitation — fall back to the list.
		return "", listMsg, nil
	}
	if res.Action != "accept" || res.Content == nil {
		return "", listMsg, nil
	}
	chosen, _ := res.Content["checkpoint_id"].(string)
	if chosen == "" {
		return "", listMsg, nil
	}
	return chosen, "", nil
}

// formatCheckpointList renders a compact, scrubbed list of checkpoints for the
// user to choose from.
func (s *Server) formatCheckpointList(cps []*checkpoint.Checkpoint) string {
	var b strings.Builder
	b.WriteString("Available checkpoints (re-run checkpoint_resume with one of these checkpoint_id values):\n")
	for _, cp := range cps {
		summary := s.scrubber.Scrub(cp.Summary).Scrubbed
		if len(summary) > 80 {
			summary = summary[:77] + "..."
		}
		fmt.Fprintf(&b, "- %s  (%s)  %s\n", cp.ID, cp.CreatedAt.Format("2006-01-02 15:04"), summary)
	}
	return b.String()
}
