// Package mcp provides a simplified MCP server that calls internal packages directly.
//
// This implementation uses the MCP SDK (github.com/modelcontextprotocol/go-sdk/mcp)
// and registers tools for checkpoint, remediation, repository, troubleshoot, memory,
// context-folding, conversation, and reflection services. All output is scrubbed for
// secrets before returning to clients.
//
// See CLAUDE.md for MCP tool descriptions and integration patterns.
package mcp
