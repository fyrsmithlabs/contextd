package reflection

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"

	"github.com/fyrsmithlabs/contextd/internal/reasoningbank"
)

// DefaultAnalyzer implements pattern analysis using the ReasoningBank.
type DefaultAnalyzer struct {
	memorySvc *reasoningbank.Service
}

// NewAnalyzer creates a new pattern analyzer.
func NewAnalyzer(memorySvc *reasoningbank.Service) *DefaultAnalyzer {
	return &DefaultAnalyzer{
		memorySvc: memorySvc,
	}
}

// Analyze identifies patterns in memories for a project.
func (a *DefaultAnalyzer) Analyze(ctx context.Context, opts AnalyzeOptions) ([]Pattern, error) {
	if opts.ProjectID == "" {
		return nil, fmt.Errorf("project_id is required")
	}

	// Set defaults
	if opts.MinConfidence == 0 {
		opts.MinConfidence = 0.3
	}
	if opts.MinFrequency == 0 {
		opts.MinFrequency = 2
	}
	if opts.MaxPatterns == 0 {
		opts.MaxPatterns = 20
	}

	// Retrieve all memories for the project
	rawMemories, err := a.memorySvc.Search(ctx, opts.ProjectID, "", 1000)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve memories: %w", err)
	}

	if len(rawMemories) == 0 {
		return []Pattern{}, nil
	}

	// Convert to pointer slice for filtering
	memories := make([]*reasoningbank.Memory, len(rawMemories))
	for i := range rawMemories {
		memories[i] = &rawMemories[i]
	}

	// Filter by period if specified
	if opts.Period != nil {
		memories = filterByPeriod(memories, opts.Period)
	}

	// Filter by tags if specified
	if len(opts.IncludeTags) > 0 || len(opts.ExcludeTags) > 0 {
		memories = filterByTags(memories, opts.IncludeTags, opts.ExcludeTags)
	}

	// Extract patterns
	patterns := a.extractPatterns(memories, opts)

	// Sort by frequency and confidence
	sort.Slice(patterns, func(i, j int) bool {
		// Primary sort by frequency, secondary by confidence
		if patterns[i].Frequency != patterns[j].Frequency {
			return patterns[i].Frequency > patterns[j].Frequency
		}
		return patterns[i].Confidence > patterns[j].Confidence
	})

	// Limit results
	if len(patterns) > opts.MaxPatterns {
		patterns = patterns[:opts.MaxPatterns]
	}

	return patterns, nil
}

// extractPatterns identifies patterns from memories.
func (a *DefaultAnalyzer) extractPatterns(memories []*reasoningbank.Memory, opts AnalyzeOptions) []Pattern {
	// Group memories by various dimensions
	byTag := make(map[string][]*reasoningbank.Memory)
	byOutcome := make(map[reasoningbank.Outcome][]*reasoningbank.Memory)

	for _, m := range memories {
		// Group by outcome
		byOutcome[m.Outcome] = append(byOutcome[m.Outcome], m)

		// Group by tag
		for _, tag := range m.Tags {
			byTag[tag] = append(byTag[tag], m)
		}
	}

	var patterns []Pattern

	// Create success/failure patterns
	for outcome, mems := range byOutcome {
		if len(mems) < opts.MinFrequency {
			continue
		}

		category := PatternSuccess
		if outcome == reasoningbank.OutcomeFailure {
			category = PatternFailure
		}

		avgConfidence := calculateAverageConfidence(mems)
		if avgConfidence < opts.MinConfidence {
			continue
		}

		patterns = append(patterns, Pattern{
			ID:          uuid.New().String(),
			Category:    category,
			Description: fmt.Sprintf("%s outcomes with %d occurrences", outcome, len(mems)),
			Tags:        extractAllTags(mems),
			Frequency:   len(mems),
			Confidence:  avgConfidence,
			MemoryIDs:   extractMemoryIDs(mems),
			FirstSeen:   findEarliestTime(mems),
			LastSeen:    findLatestTime(mems),
		})
	}

	// Create tag-based patterns
	for tag, mems := range byTag {
		if len(mems) < opts.MinFrequency {
			continue
		}

		avgConfidence := calculateAverageConfidence(mems)
		if avgConfidence < opts.MinConfidence {
			continue
		}

		// Determine if improving or declining based on confidence trend
		category := PatternRecurring
		trend := calculateConfidenceTrend(mems)
		if trend > 0.1 {
			category = PatternImproving
		} else if trend < -0.1 {
			category = PatternDeclining
		}

		patterns = append(patterns, Pattern{
			ID:          uuid.New().String(),
			Category:    category,
			Description: fmt.Sprintf("Recurring pattern in '%s' with %d occurrences", tag, len(mems)),
			Tags:        []string{tag},
			Frequency:   len(mems),
			Confidence:  avgConfidence,
			MemoryIDs:   extractMemoryIDs(mems),
			FirstSeen:   findEarliestTime(mems),
			LastSeen:    findLatestTime(mems),
		})
	}

	return patterns
}

// filterByPeriod filters memories to those within the specified period.
func filterByPeriod(memories []*reasoningbank.Memory, period *ReportPeriod) []*reasoningbank.Memory {
	filtered := make([]*reasoningbank.Memory, 0)
	for _, m := range memories {
		if m.CreatedAt.After(period.Start) && m.CreatedAt.Before(period.End) {
			filtered = append(filtered, m)
		}
	}
	return filtered
}

// filterByTags filters memories by inclusion/exclusion tags.
func filterByTags(memories []*reasoningbank.Memory, include, exclude []string) []*reasoningbank.Memory {
	includeSet := make(map[string]bool)
	excludeSet := make(map[string]bool)
	for _, t := range include {
		includeSet[t] = true
	}
	for _, t := range exclude {
		excludeSet[t] = true
	}

	filtered := make([]*reasoningbank.Memory, 0)
	for _, m := range memories {
		// Check exclusions first
		excluded := false
		for _, tag := range m.Tags {
			if excludeSet[tag] {
				excluded = true
				break
			}
		}
		if excluded {
			continue
		}

		// Check inclusions (if specified)
		if len(includeSet) > 0 {
			included := false
			for _, tag := range m.Tags {
				if includeSet[tag] {
					included = true
					break
				}
			}
			if !included {
				continue
			}
		}

		filtered = append(filtered, m)
	}
	return filtered
}

// calculateAverageConfidence calculates the average confidence of memories.
func calculateAverageConfidence(memories []*reasoningbank.Memory) float64 {
	if len(memories) == 0 {
		return 0
	}
	total := 0.0
	for _, m := range memories {
		total += m.Confidence
	}
	return total / float64(len(memories))
}

// calculateConfidenceTrend calculates whether confidence is trending up or down.
// Returns positive for improving, negative for declining, near-zero for stable.
func calculateConfidenceTrend(memories []*reasoningbank.Memory) float64 {
	if len(memories) < 2 {
		return 0
	}

	// Sort by creation time
	sorted := make([]*reasoningbank.Memory, len(memories))
	copy(sorted, memories)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].CreatedAt.Before(sorted[j].CreatedAt)
	})

	// Compare first half average to second half average
	mid := len(sorted) / 2
	firstHalf := calculateAverageConfidence(sorted[:mid])
	secondHalf := calculateAverageConfidence(sorted[mid:])

	return secondHalf - firstHalf
}

// extractAllTags extracts unique tags from memories.
func extractAllTags(memories []*reasoningbank.Memory) []string {
	tagSet := make(map[string]bool)
	for _, m := range memories {
		for _, tag := range m.Tags {
			tagSet[tag] = true
		}
	}
	tags := make([]string, 0, len(tagSet))
	for tag := range tagSet {
		tags = append(tags, tag)
	}
	return tags
}

// extractMemoryIDs extracts IDs from memories.
func extractMemoryIDs(memories []*reasoningbank.Memory) []string {
	ids := make([]string, len(memories))
	for i, m := range memories {
		ids[i] = m.ID
	}
	return ids
}

// findEarliestTime finds the earliest creation time.
func findEarliestTime(memories []*reasoningbank.Memory) time.Time {
	if len(memories) == 0 {
		return time.Time{}
	}
	earliest := memories[0].CreatedAt
	for _, m := range memories[1:] {
		if m.CreatedAt.Before(earliest) {
			earliest = m.CreatedAt
		}
	}
	return earliest
}

// findLatestTime finds the latest creation time.
func findLatestTime(memories []*reasoningbank.Memory) time.Time {
	if len(memories) == 0 {
		return time.Time{}
	}
	latest := memories[0].CreatedAt
	for _, m := range memories[1:] {
		if m.CreatedAt.After(latest) {
			latest = m.CreatedAt
		}
	}
	return latest
}

// Ensure DefaultAnalyzer implements Analyzer.
var _ Analyzer = (*DefaultAnalyzer)(nil)
