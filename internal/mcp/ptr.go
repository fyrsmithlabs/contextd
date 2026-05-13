package mcp

// ptrTrue returns a pointer to a true boolean. Used for MCP tool annotation
// hints (DestructiveHint, OpenWorldHint) where the field type is *bool and
// omission means "unknown" to the client. See docs/spec/mcp/HANDLER-GUIDE.md §2.
func ptrTrue() *bool {
	v := true
	return &v
}

// ptrFalse returns a pointer to a false boolean. Used for MCP tool annotation
// hints (DestructiveHint, OpenWorldHint) where the field type is *bool and
// omission means "unknown" to the client. See docs/spec/mcp/HANDLER-GUIDE.md §2.
func ptrFalse() *bool {
	v := false
	return &v
}
