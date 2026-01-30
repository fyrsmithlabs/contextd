package reasoningbank

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"
)

// SessionSummarizer aggregates flushed session buffers into session-level memories.
//
// The summarization pipeline:
//  1. Aggregate turns by outcome (success vs failure)
//  2. Run fact extraction on aggregated content
//  3. Build template-based summary incorporating extracted facts
//  4. Create Memory with session metadata (SessionID, SessionDate, Granularity)
//  5. Return memories for storage via service.Record()
type SessionSummarizer struct {
	extractor FactExtractor
	logger    *zap.Logger
}

// NewSessionSummarizer creates a new session summarizer.
func NewSessionSummarizer(extractor FactExtractor, logger *zap.Logger) (*SessionSummarizer, error) {
	if extractor == nil {
		return nil, fmt.Errorf("fact extractor cannot be nil")
	}
	if logger == nil {
		return nil, fmt.Errorf("logger cannot be nil")
	}

	return &SessionSummarizer{
		extractor: extractor,
		logger:    logger,
	}, nil
}

// outcomeGroup holds turns aggregated by outcome.
type outcomeGroup struct {
	outcome Outcome
	turns   []TurnEntry
	tags    map[string]struct{}
}

// Summarize processes a flushed session buffer and produces session-level memories.
//
// The returned memories have SessionID, SessionDate, and Granularity fields set.
// The caller is responsible for storing them via service.Record().
//
// Returns one memory per outcome group (success and/or failure) found in the buffer.
// Returns nil with no error if the buffer is empty.
func (s *SessionSummarizer) Summarize(ctx context.Context, buf *SessionBuffer) ([]*Memory, error) {
	if buf == nil {
		return nil, fmt.Errorf("session buffer cannot be nil")
	}
	if len(buf.Turns) == 0 {
		s.logger.Debug("empty session buffer, skipping summarization",
			zap.String("session_id", buf.SessionID),
			zap.String("project_id", buf.ProjectID))
		return nil, nil
	}

	s.logger.Info("summarizing session buffer",
		zap.String("session_id", buf.SessionID),
		zap.String("project_id", buf.ProjectID),
		zap.Int("turns", len(buf.Turns)))

	// Step 1: Aggregate turns by outcome
	groups := s.aggregateByOutcome(buf.Turns)

	var memories []*Memory

	for _, group := range groups {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		// Step 2: Build combined content from turns
		combinedContent := s.buildCombinedContent(group.turns)

		// Step 3: Extract facts from combined content
		referenceDate := buf.SessionDate
		if referenceDate.IsZero() {
			referenceDate = time.Now()
		}

		facts, err := s.extractor.Extract(ctx, combinedContent, referenceDate)
		if err != nil {
			s.logger.Warn("fact extraction failed for outcome group, continuing without facts",
				zap.String("session_id", buf.SessionID),
				zap.String("outcome", string(group.outcome)),
				zap.Error(err))
			// Continue without facts - they're supplementary
		}

		// Step 4: Build template-based summary
		title := s.buildTitle(buf.SessionID, group.outcome, len(group.turns))
		content := s.buildContent(group, facts)

		// Collect unique tags
		tags := make([]string, 0, len(group.tags))
		for tag := range group.tags {
			tags = append(tags, tag)
		}

		// Step 5: Create Memory with session metadata
		memory, err := NewMemory(buf.ProjectID, title, content, group.outcome, tags)
		if err != nil {
			s.logger.Error("failed to create session memory",
				zap.String("session_id", buf.SessionID),
				zap.String("outcome", string(group.outcome)),
				zap.Error(err))
			continue
		}

		// Set session-level metadata
		memory.SessionID = buf.SessionID
		sessionDate := buf.SessionDate
		memory.SessionDate = &sessionDate
		memory.Granularity = GranularitySession
		memory.Confidence = DistilledConfidence
		memory.Description = fmt.Sprintf("Session summary (%d turns, %s)",
			len(group.turns), group.outcome)

		memories = append(memories, memory)

		s.logger.Debug("created session memory",
			zap.String("session_id", buf.SessionID),
			zap.String("memory_id", memory.ID),
			zap.String("outcome", string(group.outcome)),
			zap.Int("turns", len(group.turns)),
			zap.Int("facts", len(facts)))
	}

	s.logger.Info("session summarization completed",
		zap.String("session_id", buf.SessionID),
		zap.String("project_id", buf.ProjectID),
		zap.Int("memories_created", len(memories)))

	return memories, nil
}

// aggregateByOutcome groups turns by their outcome.
func (s *SessionSummarizer) aggregateByOutcome(turns []TurnEntry) []outcomeGroup {
	groupMap := make(map[Outcome]*outcomeGroup)

	for _, turn := range turns {
		outcome := turn.Outcome
		// Default to success if outcome is empty
		if outcome == "" {
			outcome = OutcomeSuccess
		}

		g, ok := groupMap[outcome]
		if !ok {
			g = &outcomeGroup{
				outcome: outcome,
				turns:   make([]TurnEntry, 0),
				tags:    make(map[string]struct{}),
			}
			groupMap[outcome] = g
		}

		g.turns = append(g.turns, turn)
		for _, tag := range turn.Tags {
			g.tags[tag] = struct{}{}
		}
	}

	// Convert map to slice with deterministic ordering (success before failure)
	groups := make([]outcomeGroup, 0, len(groupMap))
	if g, ok := groupMap[OutcomeSuccess]; ok {
		groups = append(groups, *g)
	}
	if g, ok := groupMap[OutcomeFailure]; ok {
		groups = append(groups, *g)
	}

	return groups
}

// buildCombinedContent concatenates turn contents for fact extraction.
func (s *SessionSummarizer) buildCombinedContent(turns []TurnEntry) string {
	var b strings.Builder
	for i, turn := range turns {
		if i > 0 {
			b.WriteString(". ")
		}
		if turn.Title != "" {
			b.WriteString(turn.Title)
			b.WriteString(": ")
		}
		b.WriteString(turn.Content)
	}
	return b.String()
}

// buildTitle generates a title for the session memory.
func (s *SessionSummarizer) buildTitle(sessionID string, outcome Outcome, turnCount int) string {
	prefix := "Session"
	switch outcome {
	case OutcomeSuccess:
		prefix = "Success"
	case OutcomeFailure:
		prefix = "Anti-pattern"
	}

	// Truncate session ID for readability
	shortID := sessionID
	if len(shortID) > 8 {
		shortID = shortID[:8]
	}

	return fmt.Sprintf("%s: Session %s (%d turns)", prefix, shortID, turnCount)
}

// buildContent generates the memory content from turns and extracted facts.
func (s *SessionSummarizer) buildContent(group outcomeGroup, facts []Fact) string {
	var b strings.Builder

	// Section: Overview
	b.WriteString("## Overview\n")
	b.WriteString(fmt.Sprintf("Session with %d turns (outcome: %s).\n\n", len(group.turns), group.outcome))

	// Section: Key Activities
	b.WriteString("## Key Activities\n")
	for _, turn := range group.turns {
		if turn.Title != "" {
			b.WriteString(fmt.Sprintf("- %s\n", turn.Title))
		}
	}
	if b.Len() > 0 {
		b.WriteString("\n")
	}

	// Section: Details
	b.WriteString("## Details\n")
	for _, turn := range group.turns {
		b.WriteString(turn.Content)
		b.WriteString("\n")
	}
	b.WriteString("\n")

	// Section: Extracted Facts (if any)
	if len(facts) > 0 {
		b.WriteString("## Extracted Facts\n")
		for _, fact := range facts {
			b.WriteString(fmt.Sprintf("- %s %s %s", fact.Subject, fact.Predicate, fact.Object))
			if !fact.Timestamp.IsZero() {
				b.WriteString(fmt.Sprintf(" (%s)", fact.Timestamp.Format("2006-01-02")))
			}
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	// Section: Tags
	if len(group.tags) > 0 {
		b.WriteString("## Tags\n")
		tags := make([]string, 0, len(group.tags))
		for tag := range group.tags {
			tags = append(tags, tag)
		}
		b.WriteString(strings.Join(tags, ", "))
		b.WriteString("\n")
	}

	return b.String()
}
