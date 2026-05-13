package mcp

// ptrTrue returns a pointer to a true bool value.
//
// Used when populating mcp.ToolAnnotations pointer fields (DestructiveHint,
// OpenWorldHint) where omission would be ambiguous to clients. The
// HANDLER-GUIDE requires explicit pointers so the JSON wire format always
// includes the hint.
func ptrTrue() *bool {
	v := true
	return &v
}

// ptrFalse returns a pointer to a false bool value. See ptrTrue.
func ptrFalse() *bool {
	v := false
	return &v
}
