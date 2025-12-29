// internal/logging/levels.go
package logging

import (
	"go.uber.org/zap/zapcore"
)

// TraceLevel is a custom level below Debug for ultra-verbose logging.
// Value: -2 (Debug is -1, Info is 0)
//
// Use for:
//   - Function entry/exit
//   - Wire protocol data
//   - Byte-level details
//   - Almost always filtered in production
const TraceLevel = zapcore.Level(-2)

// LevelFromString parses a string into a zapcore.Level, supporting "trace".
func LevelFromString(level string) (zapcore.Level, error) {
	if level == "trace" {
		return TraceLevel, nil
	}
	var l zapcore.Level
	if err := l.UnmarshalText([]byte(level)); err != nil {
		return zapcore.InfoLevel, err
	}
	return l, nil
}
