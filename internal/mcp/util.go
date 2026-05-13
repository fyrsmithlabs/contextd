package mcp

// ptrTrue returns a pointer to the bool literal true. It exists because
// several fields on mcp.ToolAnnotations (DestructiveHint, OpenWorldHint)
// are *bool so the SDK can distinguish "explicitly set" from "unset".
// Using a helper keeps the call sites at the annotation literal readable.
func ptrTrue() *bool {
	v := true
	return &v
}

// ptrFalse returns a pointer to the bool literal false. Counterpart to
// ptrTrue; see that helper for rationale.
func ptrFalse() *bool {
	v := false
	return &v
}
