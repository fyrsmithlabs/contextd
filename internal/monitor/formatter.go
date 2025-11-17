package monitor

import "fmt"

// FormatRate formats a rate value as "X.X req/min"
func FormatRate(rate float64) string {
	return fmt.Sprintf("%.1f req/min", rate)
}

// FormatLatency formats latency in seconds as "X.Xms" or "X.Xs"
func FormatLatency(latencySeconds float64) string {
	if latencySeconds < 1.0 {
		// Convert to milliseconds
		ms := latencySeconds * 1000
		return fmt.Sprintf("%.1fms", ms)
	}
	return fmt.Sprintf("%.1fs", latencySeconds)
}

// FormatCost formats cost per minute
func FormatCost(costPerMin float64) string {
	return fmt.Sprintf("$%.4f/min", costPerMin)
}

// FormatPercentage formats a ratio (0-1) as percentage
func FormatPercentage(ratio float64) string {
	return fmt.Sprintf("%.1f%%", ratio*100)
}

// FormatMemory formats memory in bytes as "X.X MB" or "X.X GB" or "X B"
func FormatMemory(bytes uint64) string {
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.1f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.1f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

// FormatUptime formats uptime in seconds to "Xh Ym" or "Xm"
func FormatUptime(seconds int64) string {
	return FormatDuration(seconds)
}

// FormatDuration formats duration in seconds to "Xh Ym" or "Xm"
func FormatDuration(seconds int64) string {
	hours := seconds / 3600
	minutes := (seconds % 3600) / 60

	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	return fmt.Sprintf("%dm", minutes)
}
