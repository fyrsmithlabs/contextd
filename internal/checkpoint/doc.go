// Package checkpoint provides session state persistence and resumption.
//
// Saves/restores Claude session context with tiered resume levels
// (summary → context → full). Auto-checkpoint at configurable thresholds.
//
// See CLAUDE.md for checkpoint schema and resume levels.
package checkpoint
