package monitor

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatRate(t *testing.T) {
	tests := []struct {
		name     string
		rate     float64
		expected string
	}{
		{"normal", 45.7, "45.7 req/min"},
		{"zero", 0.0, "0.0 req/min"},
		{"large", 999.9, "999.9 req/min"},
		{"small", 0.1, "0.1 req/min"},
		{"very_large", 999999.9, "999999.9 req/min"},
		{"very_small", 0.0001, "0.0 req/min"},
		{"negative", -5.0, "-5.0 req/min"}, // Should handle gracefully
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatRate(tt.rate)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// RED PHASE: FormatLatency test
func TestFormatLatency(t *testing.T) {
	tests := []struct {
		name           string
		latencySeconds float64
		expected       string
	}{
		{"milliseconds", 0.0123, "12.3ms"},
		{"sub_millisecond", 0.0001, "0.1ms"},
		{"seconds", 1.234, "1.2s"},
		{"multiple_seconds", 5.678, "5.7s"},
		{"zero", 0.0, "0.0ms"},
		{"very_large", 123.456, "123.5s"},
		{"very_small", 0.00001, "0.0ms"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatLatency(tt.latencySeconds)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// RED PHASE: FormatCost test
func TestFormatCost(t *testing.T) {
	tests := []struct {
		name       string
		costPerMin float64
		expected   string
	}{
		{"normal", 0.0034, "$0.0034/min"},
		{"zero", 0.0, "$0.0000/min"},
		{"large", 1.2345, "$1.2345/min"},
		{"very_small", 0.00001, "$0.0000/min"},
		{"very_large", 99.9999, "$99.9999/min"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatCost(tt.costPerMin)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// RED PHASE: FormatPercentage test
func TestFormatPercentage(t *testing.T) {
	tests := []struct {
		name     string
		ratio    float64
		expected string
	}{
		{"normal", 0.985, "98.5%"},
		{"zero", 0.0, "0.0%"},
		{"one", 1.0, "100.0%"},
		{"small", 0.012, "1.2%"},
		{"very_small", 0.0003, "0.0%"},
		{"over_hundred", 1.5, "150.0%"}, // Handle edge case
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatPercentage(tt.ratio)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// RED PHASE: FormatMemory test
func TestFormatMemory(t *testing.T) {
	tests := []struct {
		name     string
		bytes    uint64
		expected string
	}{
		{"megabytes", 25690112, "24.5 MB"}, // 24.5 * 1024 * 1024
		{"kilobytes", 1024, "1.0 KB"},
		{"bytes", 512, "512 B"},
		{"gigabytes", 1610612736, "1.5 GB"}, // 1.5 * 1024 * 1024 * 1024
		{"zero", 0, "0 B"},
		{"large_gb", 5368709120, "5.0 GB"}, // Exactly 5 GB
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatMemory(tt.bytes)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// RED PHASE: FormatDuration test
func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		seconds  int64
		expected string
	}{
		{"hours_and_minutes", 8100, "2h 15m"}, // 2*3600 + 15*60
		{"only_hours", 7200, "2h 0m"},
		{"only_minutes", 900, "15m"},
		{"zero", 0, "0m"},
		{"one_minute", 60, "1m"},
		{"many_hours", 36000, "10h 0m"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatDuration(tt.seconds)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// REFACTOR PHASE: Edge cases for special float values
func TestFormatRate_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		rate     float64
		expected string
	}{
		{"nan", math.NaN(), "NaN req/min"},
		{"inf", math.Inf(1), "+Inf req/min"},
		{"neg_inf", math.Inf(-1), "-Inf req/min"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatRate(tt.rate)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatLatency_EdgeCases(t *testing.T) {
	tests := []struct {
		name           string
		latencySeconds float64
		expected       string
	}{
		{"nan", math.NaN(), "NaNs"},     // NaN >= 1.0 is false, but NaN < 1.0 is also false, so goes to seconds
		{"inf", math.Inf(1), "+Infs"},   // +Inf >= 1.0 is true
		{"negative", -1.5, "-1500.0ms"}, // -1.5 < 1.0, so converts to ms
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatLatency(tt.latencySeconds)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatCost_EdgeCases(t *testing.T) {
	tests := []struct {
		name       string
		costPerMin float64
		expected   string
	}{
		{"nan", math.NaN(), "$NaN/min"},
		{"inf", math.Inf(1), "$+Inf/min"},
		{"negative", -0.5, "$-0.5000/min"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatCost(tt.costPerMin)
			assert.Equal(t, tt.expected, result)
		})
	}
}
