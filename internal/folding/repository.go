package folding

import (
	"context"
	"sync"
)

// MemoryBranchRepository is an in-memory implementation of BranchRepository.
// It is thread-safe and suitable for single-instance deployments.
type MemoryBranchRepository struct {
	mu       sync.RWMutex
	branches map[string]*Branch
	// Index for fast session lookups
	bySession map[string][]string // sessionID -> []branchID
	// Index for fast parent lookups
	byParent map[string][]string // parentID -> []childBranchID
}

// NewMemoryBranchRepository creates a new in-memory branch repository.
func NewMemoryBranchRepository() *MemoryBranchRepository {
	return &MemoryBranchRepository{
		branches:  make(map[string]*Branch),
		bySession: make(map[string][]string),
		byParent:  make(map[string][]string),
	}
}

// Create stores a new branch.
func (r *MemoryBranchRepository) Create(ctx context.Context, branch *Branch) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.branches[branch.ID]; exists {
		return ErrBranchAlreadyExists
	}

	// Store a copy to prevent external mutation
	stored := *branch
	r.branches[branch.ID] = &stored

	// Update session index
	r.bySession[branch.SessionID] = append(r.bySession[branch.SessionID], branch.ID)

	// Update parent index
	if branch.ParentID != nil {
		r.byParent[*branch.ParentID] = append(r.byParent[*branch.ParentID], branch.ID)
	}

	return nil
}

// Get retrieves a branch by ID.
func (r *MemoryBranchRepository) Get(ctx context.Context, id string) (*Branch, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	branch, exists := r.branches[id]
	if !exists {
		return nil, ErrBranchNotFound
	}

	// Return a copy to prevent external mutation
	result := *branch
	return &result, nil
}

// Update modifies an existing branch.
func (r *MemoryBranchRepository) Update(ctx context.Context, branch *Branch) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.branches[branch.ID]; !exists {
		return ErrBranchNotFound
	}

	// Store a copy
	stored := *branch
	r.branches[branch.ID] = &stored

	return nil
}

// Delete removes a branch.
func (r *MemoryBranchRepository) Delete(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	branch, exists := r.branches[id]
	if !exists {
		return ErrBranchNotFound
	}

	// Remove from session index
	r.bySession[branch.SessionID] = removeFromSlice(r.bySession[branch.SessionID], id)

	// Remove from parent index
	if branch.ParentID != nil {
		r.byParent[*branch.ParentID] = removeFromSlice(r.byParent[*branch.ParentID], id)
	}

	delete(r.branches, id)
	return nil
}

// ListBySession returns all branches for a session.
func (r *MemoryBranchRepository) ListBySession(ctx context.Context, sessionID string) ([]*Branch, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ids := r.bySession[sessionID]
	result := make([]*Branch, 0, len(ids))

	for _, id := range ids {
		if branch, exists := r.branches[id]; exists {
			copy := *branch
			result = append(result, &copy)
		}
	}

	return result, nil
}

// ListByParent returns all child branches of a parent.
func (r *MemoryBranchRepository) ListByParent(ctx context.Context, parentID string) ([]*Branch, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ids := r.byParent[parentID]
	result := make([]*Branch, 0, len(ids))

	for _, id := range ids {
		if branch, exists := r.branches[id]; exists {
			copy := *branch
			result = append(result, &copy)
		}
	}

	return result, nil
}

// GetActiveBySession returns the currently active branch for a session.
// Returns the deepest active branch (highest depth) to correctly calculate
// depth when creating nested branches. Ties are broken by most recent CreatedAt.
// Returns nil if no active branch exists.
func (r *MemoryBranchRepository) GetActiveBySession(ctx context.Context, sessionID string) (*Branch, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var deepest *Branch
	ids := r.bySession[sessionID]
	for _, id := range ids {
		if branch, exists := r.branches[id]; exists && branch.Status == BranchStatusActive {
			if deepest == nil || branch.Depth > deepest.Depth ||
				(branch.Depth == deepest.Depth && branch.CreatedAt.After(deepest.CreatedAt)) {
				deepest = branch
			}
		}
	}

	if deepest == nil {
		return nil, nil
	}

	copy := *deepest
	return &copy, nil
}

// CountActiveBySession returns the count of active branches in a session.
func (r *MemoryBranchRepository) CountActiveBySession(ctx context.Context, sessionID string) (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	count := 0
	ids := r.bySession[sessionID]
	for _, id := range ids {
		if branch, exists := r.branches[id]; exists && branch.Status == BranchStatusActive {
			count++
		}
	}

	return count, nil
}

// removeFromSlice removes an element from a slice (helper).
func removeFromSlice(slice []string, item string) []string {
	result := make([]string, 0, len(slice))
	for _, s := range slice {
		if s != item {
			result = append(result, s)
		}
	}
	return result
}
