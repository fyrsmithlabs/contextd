package monitor

import (
	"context"
	"fmt"
	"time"

	"github.com/NimbleMarkets/ntcharts/sparkline"
	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	sparklineWidth  = 30
	sparklineHeight = 3
	historySize     = 30
)

// Model represents the BubbleTea dashboard model
type Model struct {
	vmURL      string
	interval   time.Duration
	lastUpdate time.Time
	metrics    MetricsSnapshot
	err        error
	quitting   bool

	// Progress bars
	memoryProgress  progress.Model
	requestProgress progress.Model
}

// MetricsSnapshot holds the current metrics data
type MetricsSnapshot struct {
	HTTPRate        float64
	HTTPLatencyP95  float64
	EmbeddingRate   float64
	EmbeddingTokens float64
	EmbeddingCost   float64
	Uptime          int64
	Goroutines      int
	MemoryMB        float64

	// Context monitoring metrics
	ContextTokensUsed   float64
	ContextUsagePercent float64
	Threshold70Hits     float64
	Threshold90Hits     float64
	AvgTokensSaved      float64
	AvgReductionPct     float64

	// Historical data for sparklines (last N points)
	HTTPRateHistory      []float64
	LatencyHistory       []float64
	EmbeddingRateHistory []float64
	MemoryHistory        []float64
	ContextUsageHistory  []float64
	ReductionPctHistory  []float64

	// Peak values for progress bars
	HTTPRatePeak float64
	MemoryMax    float64
}

// Lipgloss styles (k9s-inspired color scheme)
var (
	// Header style - bright cyan background, bold black text
	headerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("0")).
			Background(lipgloss.Color("51")).
			Bold(true).
			Padding(0, 1)

	// Border style - dim gray (#444444) - currently unused
	// borderStyle = lipgloss.NewStyle().
	// 		BorderForeground(lipgloss.Color("238"))

	// Section title style - bold bright cyan
	sectionStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("51")).
			Bold(true).
			MarginTop(1)

	// Label style - dim cyan
	labelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("45"))

	// Value style - bright white
	valueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("231")).
			Bold(true)

	// Dim style - for units and secondary info
	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245"))

	// Status styles with unicode symbols
	healthyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("46")).
			Bold(true)

	warningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("226")).
			Bold(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)

	// Container style - rounded border with dim gray
	containerStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("238")).
			Padding(1, 2)

	// Footer style - bright keys on dim background
	footerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")).
			MarginTop(1)

	footerKeyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("51")).
			Bold(true)

	// Sparkline container
	sparklineStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("51"))
)

// NewModel creates a new dashboard model
func NewModel(vmURL string, interval time.Duration) Model {
	// Initialize progress bars with custom gradient
	memProg := progress.New(
		progress.WithGradient("#00ff00", "#ffff00"),
		progress.WithWidth(40),
	)

	reqProg := progress.New(
		progress.WithGradient("#00ffff", "#ff00ff"),
		progress.WithWidth(40),
	)

	return Model{
		vmURL:           vmURL,
		interval:        interval,
		quitting:        false,
		memoryProgress:  memProg,
		requestProgress: reqProg,
		metrics: MetricsSnapshot{
			HTTPRateHistory:      make([]float64, 0, historySize),
			LatencyHistory:       make([]float64, 0, historySize),
			EmbeddingRateHistory: make([]float64, 0, historySize),
			MemoryHistory:        make([]float64, 0, historySize),
			ContextUsageHistory:  make([]float64, 0, historySize),
			ReductionPctHistory:  make([]float64, 0, historySize),
			HTTPRatePeak:         1.0,   // Minimum peak to avoid division by zero
			MemoryMax:            512.0, // Default max memory in MB
		},
	}
}

// getLatencyBadge returns a colored status badge based on latency
func getLatencyBadge(latencyMS float64) string {
	if latencyMS < 100 {
		return healthyStyle.Render("[✓]")
	} else if latencyMS < 500 {
		return warningStyle.Render("[⚠]")
	}
	return errorStyle.Render("[✗]")
}

// getStatusBadge returns overall system status badge
func getStatusBadge(latencyMS float64) string {
	if latencyMS < 100 {
		return healthyStyle.Render("✓ HEALTHY")
	} else if latencyMS < 500 {
		return warningStyle.Render("⚠ WARN")
	}
	return errorStyle.Render("✗ ERROR")
}

// getContextBadge returns context usage badge based on percentage
func getContextBadge(usagePercent float64) string {
	if usagePercent < 70 {
		return healthyStyle.Render("[✓]")
	} else if usagePercent < 90 {
		return warningStyle.Render("[⚠]")
	}
	return errorStyle.Render("[✗]")
}

// appendToHistory appends a value to history, maintaining max size
func appendToHistory(history []float64, value float64) []float64 {
	history = append(history, value)
	if len(history) > historySize {
		history = history[1:]
	}
	return history
}

// createSparkline creates a sparkline chart from historical data
func createSparkline(data []float64) string {
	if len(data) == 0 {
		return dimStyle.Render(fmt.Sprintf("%*s", sparklineWidth, "no data"))
	}

	spark := sparkline.New(sparklineWidth, sparklineHeight)
	for _, v := range data {
		spark.Push(v)
	}

	return sparklineStyle.Render(spark.View())
}

// Message types
type tickMsg time.Time
type metricsMsg MetricsSnapshot
type errMsg error

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		tick(m.interval),
		fetchMetrics(m.vmURL),
	)
}

// tick creates a tick command for auto-refresh
func tick(interval time.Duration) tea.Cmd {
	return tea.Tick(interval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// fetchMetrics fetches metrics from VictoriaMetrics
func fetchMetrics(vmURL string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		client := NewMetricsClient(vmURL)

		// Fetch all metrics
		httpRate, err := client.QueryHTTPRate(ctx)
		if err != nil {
			return errMsg(err)
		}

		httpLatency, err := client.QueryHTTPLatencyP95(ctx)
		if err != nil {
			return errMsg(err)
		}

		embeddingRate, err := client.QueryEmbeddingRate(ctx)
		if err != nil {
			return errMsg(err)
		}

		// Embedding tokens rate
		tokensResult, err := client.Query(ctx, "rate(contextd_embedding_tokens_total[1m])")
		if err != nil {
			return errMsg(err)
		}
		embeddingTokens, _ := extractFloatValue(tokensResult)

		// Embedding cost rate
		costResult, err := client.Query(ctx, "rate(contextd_embedding_cost_USD_total[1m])")
		if err != nil {
			return errMsg(err)
		}
		embeddingCost, _ := extractFloatValue(costResult)

		// Context monitoring metrics
		contextTokensUsed, err := client.QueryContextTokensUsed(ctx)
		if err != nil {
			contextTokensUsed = 0 // Graceful fallback
		}

		contextUsagePercent, err := client.QueryContextUsagePercent(ctx)
		if err != nil {
			contextUsagePercent = 0
		}

		threshold70Hits, err := client.QueryContext70ThresholdHits(ctx)
		if err != nil {
			threshold70Hits = 0
		}

		threshold90Hits, err := client.QueryContext90ThresholdHits(ctx)
		if err != nil {
			threshold90Hits = 0
		}

		avgTokensSaved, err := client.QueryAvgTokensSaved(ctx)
		if err != nil {
			avgTokensSaved = 0
		}

		avgReductionPct, err := client.QueryAvgReductionPct(ctx)
		if err != nil {
			avgReductionPct = 0
		}

		// System metrics (TODO: add process_exporter for real values)
		goroutines := 42   // Placeholder
		memoryMB := 24.5   // Placeholder
		uptime := int64(0) // Placeholder

		return metricsMsg{
			HTTPRate:            httpRate,
			HTTPLatencyP95:      httpLatency,
			EmbeddingRate:       embeddingRate,
			EmbeddingTokens:     embeddingTokens,
			EmbeddingCost:       embeddingCost,
			Uptime:              uptime,
			Goroutines:          goroutines,
			MemoryMB:            memoryMB,
			ContextTokensUsed:   contextTokensUsed,
			ContextUsagePercent: contextUsagePercent,
			Threshold70Hits:     threshold70Hits,
			Threshold90Hits:     threshold90Hits,
			AvgTokensSaved:      avgTokensSaved,
			AvgReductionPct:     avgReductionPct,
		}
	}
}

// Update handles messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		case "r":
			return m, fetchMetrics(m.vmURL)
		}

	case tickMsg:
		// Auto-refresh triggered
		return m, tea.Batch(
			tick(m.interval),
			fetchMetrics(m.vmURL),
		)

	case metricsMsg:
		// Metrics successfully fetched - update with history
		newMetrics := MetricsSnapshot(msg)

		// Preserve historical data and update ring buffers
		newMetrics.HTTPRateHistory = appendToHistory(m.metrics.HTTPRateHistory, newMetrics.HTTPRate)
		newMetrics.LatencyHistory = appendToHistory(m.metrics.LatencyHistory, newMetrics.HTTPLatencyP95*1000) // Convert to ms
		newMetrics.EmbeddingRateHistory = appendToHistory(m.metrics.EmbeddingRateHistory, newMetrics.EmbeddingRate)
		newMetrics.MemoryHistory = appendToHistory(m.metrics.MemoryHistory, newMetrics.MemoryMB)
		newMetrics.ContextUsageHistory = appendToHistory(m.metrics.ContextUsageHistory, newMetrics.ContextUsagePercent)
		newMetrics.ReductionPctHistory = appendToHistory(m.metrics.ReductionPctHistory, newMetrics.AvgReductionPct)

		// Update peaks
		newMetrics.HTTPRatePeak = m.metrics.HTTPRatePeak
		if newMetrics.HTTPRate > newMetrics.HTTPRatePeak {
			newMetrics.HTTPRatePeak = newMetrics.HTTPRate
		}
		newMetrics.MemoryMax = m.metrics.MemoryMax

		m.metrics = newMetrics
		m.lastUpdate = time.Now()
		m.err = nil
		return m, nil

	case errMsg:
		// Error occurred
		m.err = error(msg)
		return m, nil
	}

	return m, nil
}

// View renders the dashboard
func (m Model) View() string {
	if m.quitting {
		return ""
	}

	// Display error state if error exists
	if m.err != nil {
		return m.renderError()
	}

	return m.renderDashboard()
}

// renderError renders the error view
func (m Model) renderError() string {
	header := headerStyle.Render("contextd Metrics Dashboard")

	var content string
	content += "\n"
	content += errorStyle.Render("⚠ Cannot connect to VictoriaMetrics") + "\n"
	content += "\n"
	content += dimStyle.Render("URL: ") + valueStyle.Render(m.vmURL) + "\n"
	content += dimStyle.Render("Error: ") + errorStyle.Render(m.err.Error()) + "\n"
	content += "\n"
	content += dimStyle.Render("Please ensure:") + "\n"
	content += dimStyle.Render("  1. docker-compose up -d victoriametrics") + "\n"
	content += dimStyle.Render("  2. VictoriaMetrics is running on :8428") + "\n"
	content += "\n"
	content += footerStyle.Render("[q] quit  [r] retry") + "\n"

	box := containerStyle.Render(header + "\n" + content)
	return box
}

// renderDashboard renders the main dashboard view with sparklines and progress bars
func (m Model) renderDashboard() string {
	var content string

	// Header with status badge
	lastUpdateStr := "Never"
	if !m.lastUpdate.IsZero() {
		lastUpdateStr = m.lastUpdate.Format("3:04:05 PM")
	}
	uptimeStr := FormatUptime(m.metrics.Uptime)
	latencyMS := m.metrics.HTTPLatencyP95 * 1000

	header := headerStyle.Render(" contextd Monitor ")
	statusBadge := getStatusBadge(latencyMS)
	headerLine := fmt.Sprintf("%s   %s   %s   %s",
		statusBadge,
		dimStyle.Render("Uptime:"),
		valueStyle.Render(uptimeStr),
		dimStyle.Render(lastUpdateStr))

	content += header + "\n"
	content += headerLine + "\n"

	// HTTP Requests section with sparkline and progress
	content += "\n" + sectionStyle.Render("┃ HTTP Requests") + "\n"

	// Rate with sparkline
	rateSparkline := createSparkline(m.metrics.HTTPRateHistory)
	rateBadge := getLatencyBadge(latencyMS)
	content += labelStyle.Render("  Rate: ") +
		valueStyle.Render(FormatRate(m.metrics.HTTPRate)) +
		" " + rateBadge +
		"   " + rateSparkline + "\n"

	// Latency with sparkline
	latencySparkline := createSparkline(m.metrics.LatencyHistory)
	content += labelStyle.Render("  Latency (p95): ") +
		valueStyle.Render(FormatLatency(m.metrics.HTTPLatencyP95)) +
		" " + rateBadge +
		"   " + latencySparkline + "\n"

	// Request rate progress bar
	ratePercent := 0.0
	if m.metrics.HTTPRatePeak > 0 {
		ratePercent = m.metrics.HTTPRate / m.metrics.HTTPRatePeak
		if ratePercent > 1.0 {
			ratePercent = 1.0
		}
	}
	content += labelStyle.Render("  Load: ") +
		m.requestProgress.ViewAs(ratePercent) +
		" " + dimStyle.Render(fmt.Sprintf("%.0f%%", ratePercent*100)) + "\n"

	// Embeddings section with sparkline
	content += "\n" + sectionStyle.Render("┃ Embeddings") + "\n"

	// Embedding rate with sparkline
	embeddingSparkline := createSparkline(m.metrics.EmbeddingRateHistory)
	content += labelStyle.Render("  Ops: ") +
		valueStyle.Render(FormatRate(m.metrics.EmbeddingRate)) +
		"                " + embeddingSparkline + "\n"

	// Tokens and cost
	content += labelStyle.Render("  Tokens: ") +
		valueStyle.Render(fmt.Sprintf("%.0f/min", m.metrics.EmbeddingTokens)) +
		"  " +
		labelStyle.Render("Cost: ") +
		valueStyle.Render(FormatCost(m.metrics.EmbeddingCost)) + "\n"

	// Context Window section
	content += "\n" + sectionStyle.Render("┃ Context Window") + "\n"

	// Context usage with progress bar and sparkline
	contextSparkline := createSparkline(m.metrics.ContextUsageHistory)
	contextBadge := getContextBadge(m.metrics.ContextUsagePercent)
	content += labelStyle.Render("  Usage: ") +
		valueStyle.Render(fmt.Sprintf("%.0fK / 200K tokens", m.metrics.ContextTokensUsed/1000)) +
		" " + contextBadge +
		"   " + contextSparkline + "\n"

	// Create context usage progress bar
	contextPercent := m.metrics.ContextUsagePercent / 100.0
	if contextPercent > 1.0 {
		contextPercent = 1.0
	}
	contextProgress := progress.New(
		progress.WithGradient("#00ff00", "#ff0000"), // Green to red gradient
		progress.WithWidth(40),
	)
	content += labelStyle.Render("  Progress: ") +
		contextProgress.ViewAs(contextPercent) +
		" " + dimStyle.Render(fmt.Sprintf("%.0f%%", m.metrics.ContextUsagePercent)) + "\n"

	// Threshold violations
	threshold70 := fmt.Sprintf("%.1f hits/5m", m.metrics.Threshold70Hits)
	threshold90 := fmt.Sprintf("%.1f hits/5m", m.metrics.Threshold90Hits)
	content += labelStyle.Render("  Thresholds: ") +
		dimStyle.Render("70%=") + valueStyle.Render(threshold70) +
		dimStyle.Render("  90%=") + valueStyle.Render(threshold90) + "\n"

	// Checkpoint effectiveness section
	content += "\n" + sectionStyle.Render("┃ Checkpoint Effectiveness") + "\n"

	// Average reduction with sparkline
	reductionSparkline := createSparkline(m.metrics.ReductionPctHistory)
	content += labelStyle.Render("  Avg Reduction: ") +
		valueStyle.Render(fmt.Sprintf("%.0f%%", m.metrics.AvgReductionPct)) +
		"            " + reductionSparkline + "\n"

	// Average tokens saved
	content += labelStyle.Render("  Tokens Saved: ") +
		valueStyle.Render(fmt.Sprintf("%.0fK avg", m.metrics.AvgTokensSaved/1000)) + "\n"

	// System section with memory progress
	content += "\n" + sectionStyle.Render("┃ System") + "\n"

	// Memory with progress bar
	memoryPercent := m.metrics.MemoryMB / m.metrics.MemoryMax
	if memoryPercent > 1.0 {
		memoryPercent = 1.0
	}
	content += labelStyle.Render("  Memory: ") +
		m.memoryProgress.ViewAs(memoryPercent) +
		" " + dimStyle.Render(fmt.Sprintf("%.1f%%", memoryPercent*100)) + "\n"

	// Goroutines
	content += labelStyle.Render("  Goroutines: ") +
		valueStyle.Render(fmt.Sprintf("%d", m.metrics.Goroutines)) + "\n"

	// Footer with keyboard shortcuts
	footer := footerKeyStyle.Render("[q]") + footerStyle.Render(" quit  ") +
		footerKeyStyle.Render("[r]") + footerStyle.Render(" refresh  ") +
		footerKeyStyle.Render("[w]") + footerStyle.Render(" worktrees  ") +
		footerStyle.Render(fmt.Sprintf("Auto: %v", m.interval))

	content += "\n" + footer

	// Wrap in container
	return containerStyle.Render(content)
}
