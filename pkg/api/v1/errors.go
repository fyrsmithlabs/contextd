package v1

import "errors"

// Common API errors.
var (
	ErrSessionRequired = errors.New("session_id is required")
	ErrInvalidRequest  = errors.New("invalid request")
	ErrNotFound        = errors.New("resource not found")
	ErrTimeout         = errors.New("operation timed out")
	ErrPermissionDenied = errors.New("permission denied")
)
