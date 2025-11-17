package prefetch

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// Rule defines the interface for pre-fetch rules.
type Rule interface {
	// Execute runs the rule for a given git event and returns a result.
	// Returns an error if the rule fails or times out.
	Execute(ctx context.Context, event GitEvent) (*PreFetchResult, error)

	// Name returns the rule name (e.g., "branch_diff")
	Name() string

	// Trigger returns the event type that triggers this rule
	Trigger() EventType
}

// BranchDiffRule executes git diff when a branch switch occurs.
type BranchDiffRule struct {
	projectPath string
	maxSizeKB   int
	timeout     time.Duration
}

// NewBranchDiffRule creates a new branch diff rule.
func NewBranchDiffRule(projectPath string, maxSizeKB int, timeout time.Duration) *BranchDiffRule {
	return &BranchDiffRule{
		projectPath: projectPath,
		maxSizeKB:   maxSizeKB,
		timeout:     timeout,
	}
}

// Name returns the rule name.
func (r *BranchDiffRule) Name() string {
	return "branch_diff"
}

// Trigger returns the event type that triggers this rule.
func (r *BranchDiffRule) Trigger() EventType {
	return EventTypeBranchSwitch
}

// Execute runs the git diff command.
func (r *BranchDiffRule) Execute(ctx context.Context, event GitEvent) (*PreFetchResult, error) {
	// Create context with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	// Execute git diff --stat
	cmd := exec.CommandContext(timeoutCtx, "git", "diff", "--stat",
		fmt.Sprintf("%s..%s", event.OldBranch, event.NewBranch))
	cmd.Dir = r.projectPath

	output, err := cmd.CombinedOutput()
	if err != nil {
		// Check if it's a timeout
		if timeoutCtx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("git diff timeout after %v", r.timeout)
		}
		return nil, fmt.Errorf("git diff failed: %w (output: %s)", err, string(output))
	}

	// Check size limit
	if len(output) > r.maxSizeKB*1024 {
		output = output[:r.maxSizeKB*1024]
	}

	summary := string(output)

	// Parse changed files
	changedFiles := parseChangedFiles(summary)

	// Create result
	result := &PreFetchResult{
		Type: r.Name(),
		Data: map[string]interface{}{
			"summary":       summary,
			"changed_files": changedFiles,
		},
		Metadata: map[string]string{
			"old_branch": event.OldBranch,
			"new_branch": event.NewBranch,
		},
		Confidence: 1.0,
	}

	return result, nil
}

// parseChangedFiles extracts file paths from git diff --stat output.
func parseChangedFiles(diffOutput string) []string {
	var files []string
	lines := strings.Split(diffOutput, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		// Git diff --stat format: " filename | changes"
		parts := strings.Split(trimmed, "|")
		if len(parts) >= 1 {
			filename := strings.TrimSpace(parts[0])
			if filename != "" && !strings.Contains(filename, "file") {
				// Skip summary line like "3 files changed"
				files = append(files, filename)
			}
		}
	}
	return files
}

// RecentCommitRule fetches commit information when a new commit is detected.
type RecentCommitRule struct {
	projectPath string
	maxSizeKB   int
	timeout     time.Duration
}

// NewRecentCommitRule creates a new recent commit rule.
func NewRecentCommitRule(projectPath string, maxSizeKB int, timeout time.Duration) *RecentCommitRule {
	return &RecentCommitRule{
		projectPath: projectPath,
		maxSizeKB:   maxSizeKB,
		timeout:     timeout,
	}
}

// Name returns the rule name.
func (r *RecentCommitRule) Name() string {
	return "recent_commit"
}

// Trigger returns the event type that triggers this rule.
func (r *RecentCommitRule) Trigger() EventType {
	return EventTypeNewCommit
}

// Execute runs the git show command.
func (r *RecentCommitRule) Execute(ctx context.Context, event GitEvent) (*PreFetchResult, error) {
	// Create context with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	// Execute git show --stat
	cmd := exec.CommandContext(timeoutCtx, "git", "show", "--stat", event.CommitHash)
	cmd.Dir = r.projectPath

	output, err := cmd.CombinedOutput()
	if err != nil {
		// Check if it's a timeout
		if timeoutCtx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("git show timeout after %v", r.timeout)
		}
		return nil, fmt.Errorf("git show failed: %w (output: %s)", err, string(output))
	}

	// Check size limit
	if len(output) > r.maxSizeKB*1024 {
		output = output[:r.maxSizeKB*1024]
	}

	// Parse commit info
	commitInfo := parseCommitInfo(string(output))

	// Create result
	result := &PreFetchResult{
		Type: r.Name(),
		Data: commitInfo,
		Metadata: map[string]string{
			"commit_hash": event.CommitHash,
		},
		Confidence: 1.0,
	}

	return result, nil
}

// parseCommitInfo extracts commit information from git show output.
func parseCommitInfo(output string) map[string]interface{} {
	lines := strings.Split(output, "\n")
	info := make(map[string]interface{})

	var messageBuf bytes.Buffer
	inMessage := false

	for _, line := range lines {
		// Extract commit hash
		if strings.HasPrefix(line, "commit ") {
			hash := strings.TrimSpace(strings.TrimPrefix(line, "commit "))
			info["hash"] = hash
			continue
		}

		// Extract author
		if strings.HasPrefix(line, "Author: ") {
			author := strings.TrimSpace(strings.TrimPrefix(line, "Author: "))
			info["author"] = author
			continue
		}

		// Extract date
		if strings.HasPrefix(line, "Date: ") {
			date := strings.TrimSpace(strings.TrimPrefix(line, "Date: "))
			info["date"] = date
			inMessage = true
			continue
		}

		// Extract commit message
		if inMessage && strings.TrimSpace(line) != "" && !strings.Contains(line, "|") {
			messageBuf.WriteString(strings.TrimSpace(line))
			messageBuf.WriteString(" ")
		}

		// Stop at file changes section
		if strings.Contains(line, "|") {
			inMessage = false
		}
	}

	message := strings.TrimSpace(messageBuf.String())
	if message != "" {
		info["message"] = message
	}

	info["full_output"] = output

	return info
}

// RuleRegistry manages available pre-fetch rules.
type RuleRegistry struct {
	rules []Rule
}

// NewRuleRegistry creates a new rule registry with default rules.
func NewRuleRegistry(projectPath string) *RuleRegistry {
	return &RuleRegistry{
		rules: []Rule{
			NewBranchDiffRule(projectPath, 50, 1*time.Second),
			NewRecentCommitRule(projectPath, 20, 500*time.Millisecond),
			// RelatedFilesRule would be added here when vector store integration is ready
		},
	}
}

// GetRulesForEvent returns all rules that should be executed for an event type.
func (r *RuleRegistry) GetRulesForEvent(eventType EventType) []Rule {
	var matchingRules []Rule
	for _, rule := range r.rules {
		if rule.Trigger() == eventType {
			matchingRules = append(matchingRules, rule)
		}
	}
	return matchingRules
}

// AddRule adds a custom rule to the registry.
func (r *RuleRegistry) AddRule(rule Rule) {
	r.rules = append(r.rules, rule)
}
