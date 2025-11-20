package stdio

// TODO: Port type definitions from old-contextd/pkg/mcp/types.go
//
// Types to port:
// - Input/Output structs for each MCP tool
// - Rate limiter configuration
// - Metrics types
// - Service interfaces
//
// Example from old implementation:
// type CheckpointSaveInput struct {
//     Summary     string `json:"summary" jsonschema:"required,description=Brief summary of checkpoint"`
//     ProjectPath string `json:"project_path" jsonschema:"required,description=Absolute path to project directory"`
//     Content     string `json:"content,omitempty" jsonschema:"description=Full checkpoint content"`
// }
