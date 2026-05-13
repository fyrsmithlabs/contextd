package mcp

// ptrTrue returns a pointer to a bool with value true. Used for
// mcp.ToolAnnotations fields (DestructiveHint, OpenWorldHint) where the
// pointer-bool distinguishes "explicit value" from "unknown default" on the
// wire. See docs/spec/mcp/HANDLER-GUIDE.md §2.2.
func ptrTrue() *bool {
	v := true
	return &v
}

// ptrFalse returns a pointer to a bool with value false. Used for
// mcp.ToolAnnotations fields (DestructiveHint, OpenWorldHint) where the
// pointer-bool distinguishes "explicit value" from "unknown default" on the
// wire. See docs/spec/mcp/HANDLER-GUIDE.md §2.2.
func ptrFalse() *bool {
	v := false
	return &v
}
