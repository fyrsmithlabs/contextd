package prefetch

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

var (
	// ErrNotGitRepo indicates the directory is not a Git repository
	ErrNotGitRepo = errors.New("not a git repository")

	// ErrWatcherFailed indicates the filesystem watcher failed to initialize
	ErrWatcherFailed = errors.New("failed to initialize filesystem watcher")
)

// EventType represents the type of git event detected.
type EventType int

const (
	// EventTypeBranchSwitch indicates a branch switch was detected
	EventTypeBranchSwitch EventType = iota

	// EventTypeNewCommit indicates a new commit was detected
	EventTypeNewCommit
)

// GitEvent represents a detected git event.
type GitEvent struct {
	// Type is the event type (branch switch or new commit)
	Type EventType

	// ProjectPath is the absolute path to the project
	ProjectPath string

	// OldBranch is the previous branch name (for branch switch events)
	OldBranch string

	// NewBranch is the new branch name (for branch switch events)
	NewBranch string

	// CommitHash is the commit SHA (for new commit events)
	CommitHash string

	// Timestamp is when the event was detected
	Timestamp time.Time
}

// GitEventDetector detects git events using filesystem watchers.
type GitEventDetector struct {
	projectPath   string
	gitDir        string
	watcher       *fsnotify.Watcher
	events        chan GitEvent
	stop          chan struct{}
	currentBranch string
	lastCommit    string // Track last commit to detect new commits
}

// NewGitEventDetector creates a new git event detector for a project.
//
// The detector supports both main repositories and git worktrees.
// Each worktree is treated as an independent project with its own detector.
//
// Returns an error if the path is not a git repository or the watcher fails.
func NewGitEventDetector(projectPath string) (*GitEventDetector, error) {
	// Detect git directory (handles both main repo and worktrees)
	gitDir, err := DetectGitDir(projectPath)
	if err != nil {
		return nil, fmt.Errorf("detecting git directory: %w", err)
	}

	// Create filesystem watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrWatcherFailed, err)
	}

	return &GitEventDetector{
		projectPath: projectPath,
		gitDir:      gitDir,
		watcher:     watcher,
		events:      make(chan GitEvent, 10),
		stop:        make(chan struct{}),
	}, nil
}

// Start begins watching for git events.
//
// This method runs in a background goroutine and sends events to the Events() channel.
// Call Stop() to clean up resources.
func (d *GitEventDetector) Start(ctx context.Context) error {
	// Read current branch
	branch, err := d.readCurrentBranch()
	if err != nil {
		return fmt.Errorf("reading current branch: %w", err)
	}
	d.currentBranch = branch

	// Watch HEAD file for branch switches
	headFile := filepath.Join(d.gitDir, "HEAD")
	if err := d.watcher.Add(headFile); err != nil {
		return fmt.Errorf("watching HEAD file: %w", err)
	}

	// Watch logs/HEAD for new commits
	logsHeadFile := filepath.Join(d.gitDir, "logs", "HEAD")
	if _, err := os.Stat(logsHeadFile); err == nil {
		// logs/HEAD exists, watch it
		_ = d.watcher.Add(logsHeadFile)
		// Ignore errors - logs might not exist in bare repos
	}

	// Start event processing goroutine
	go d.processEvents(ctx)

	return nil
}

// Stop stops the detector and cleans up resources.
func (d *GitEventDetector) Stop() {
	select {
	case <-d.stop:
		// Already stopped
		return
	default:
		close(d.stop)
		_ = d.watcher.Close() // Best-effort cleanup, ignore error
	}
}

// Events returns the channel for receiving git events.
func (d *GitEventDetector) Events() <-chan GitEvent {
	return d.events
}

// processEvents processes filesystem events and emits git events.
func (d *GitEventDetector) processEvents(ctx context.Context) {
	for {
		select {
		case <-d.stop:
			return
		case <-ctx.Done():
			return
		case event, ok := <-d.watcher.Events:
			if !ok {
				return
			}

			// Handle file modifications
			if event.Op&fsnotify.Write == fsnotify.Write {
				d.handleFileChange(event.Name)
			}

		case err, ok := <-d.watcher.Errors:
			if !ok {
				return
			}
			// Log error but continue watching
			_ = err // In production, use structured logging
		}
	}
}

// handleFileChange handles a file change event.
func (d *GitEventDetector) handleFileChange(path string) {
	base := filepath.Base(path)

	switch base {
	case "HEAD":
		// Branch switch or detached HEAD change
		d.detectBranchSwitch()

	// Note: We could also watch logs/HEAD for commits, but for simplicity
	// and to match the design doc's focus on branch switches, we'll
	// primarily focus on HEAD changes. Commit detection can be added
	// by watching logs/HEAD similarly.
	default:
		// Check if this is logs/HEAD
		if strings.HasSuffix(path, "logs/HEAD") {
			d.detectNewCommit()
		}
	}
}

// detectBranchSwitch detects if a branch switch occurred.
func (d *GitEventDetector) detectBranchSwitch() {
	newBranch, err := d.readCurrentBranch()
	if err != nil {
		// Error reading branch, skip
		return
	}

	if newBranch != d.currentBranch {
		// Branch switched
		event := GitEvent{
			Type:        EventTypeBranchSwitch,
			ProjectPath: d.projectPath,
			OldBranch:   d.currentBranch,
			NewBranch:   newBranch,
			Timestamp:   time.Now(),
		}

		// Send event (non-blocking)
		select {
		case d.events <- event:
		default:
			// Channel full, skip event
		}

		// Update current branch
		d.currentBranch = newBranch
	}
}

// detectNewCommit detects if a new commit was made.
func (d *GitEventDetector) detectNewCommit() {
	// Read logs/HEAD to get latest commit
	logsFile := filepath.Join(d.gitDir, "logs", "HEAD")
	content, err := os.ReadFile(logsFile)
	if err != nil {
		return
	}

	// Handle empty file (no commits yet)
	trimmed := strings.TrimSpace(string(content))
	if trimmed == "" {
		return
	}

	// Parse last line for commit hash
	lines := strings.Split(trimmed, "\n")
	if len(lines) == 0 {
		return
	}

	lastLine := lines[len(lines)-1]
	parts := strings.Fields(lastLine)
	if len(parts) < 2 {
		return
	}

	// Second field is the new commit hash (old hash is first field)
	commitHash := parts[1]

	// Check if this is a new commit
	if commitHash == d.lastCommit {
		return
	}

	// Update last commit
	d.lastCommit = commitHash

	// Emit event
	event := GitEvent{
		Type:        EventTypeNewCommit,
		ProjectPath: d.projectPath,
		CommitHash:  commitHash,
		Timestamp:   time.Now(),
	}

	// Send event (non-blocking)
	select {
	case d.events <- event:
	default:
		// Channel full, skip event
	}
}

// readCurrentBranch reads the current branch from HEAD.
func (d *GitEventDetector) readCurrentBranch() (string, error) {
	headFile := filepath.Join(d.gitDir, "HEAD")
	content, err := os.ReadFile(headFile)
	if err != nil {
		return "", fmt.Errorf("reading HEAD: %w", err)
	}

	head := strings.TrimSpace(string(content))

	// Empty HEAD = detached
	if head == "" {
		return "detached", nil
	}

	// Check if HEAD points to a branch (ref: refs/heads/<branch>)
	if strings.HasPrefix(head, "ref: refs/heads/") {
		branch := strings.TrimPrefix(head, "ref: refs/heads/")
		return branch, nil
	}

	// Otherwise it's a commit hash (detached HEAD)
	return "detached", nil
}

// DetectGitDir detects the git directory for a project path.
//
// Handles both main repositories (.git directory) and worktrees (.git file).
//
// For main repo:
//   - Returns /path/to/project/.git
//
// For worktree:
//   - Reads .git file containing "gitdir: /main/.git/worktrees/feature"
//   - Returns /main/.git/worktrees/feature
//
// Returns an error if the path is not a git repository.
func DetectGitDir(projectPath string) (string, error) {
	gitPath := filepath.Join(projectPath, ".git")

	info, err := os.Stat(gitPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("%w: %s", ErrNotGitRepo, projectPath)
		}
		return "", fmt.Errorf("stat .git: %w", err)
	}

	// Main repository: .git is a directory
	if info.IsDir() {
		return gitPath, nil
	}

	// Worktree: .git is a file containing "gitdir: <path>"
	content, err := os.ReadFile(gitPath)
	if err != nil {
		return "", fmt.Errorf("reading .git file: %w", err)
	}

	gitDir := parseGitDir(string(content))
	if gitDir == "" {
		return "", fmt.Errorf("%w: invalid .git file format", ErrNotGitRepo)
	}

	return gitDir, nil
}

// parseGitDir parses the .git file content to extract the gitdir path.
//
// Expected format: "gitdir: /path/to/git/directory\n"
func parseGitDir(content string) string {
	content = strings.TrimSpace(content)

	// Check for "gitdir:" prefix
	if !strings.HasPrefix(content, "gitdir:") {
		return ""
	}

	// Extract path after "gitdir:"
	path := strings.TrimSpace(strings.TrimPrefix(content, "gitdir:"))
	return path
}
