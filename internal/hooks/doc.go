// Package hooks provides lifecycle hook management for contextd sessions.
//
// Supports session_start, session_end, before_clear, after_clear, and
// context_threshold events. Enables auto-checkpoint at configurable thresholds
// and auto-resume on session start.
//
// See CLAUDE.md for hook types and configuration options.
package hooks
