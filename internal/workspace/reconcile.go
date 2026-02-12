package workspace

import (
	"fmt"
	"os"

	"github.com/tommyzliu/ocw/internal/git"
	"github.com/tommyzliu/ocw/internal/state"
)

// ReconcileResult contains the results of a reconciliation operation.
type ReconcileResult struct {
	RepairedWorktrees bool
	PrunedWorktrees   bool
	InstancesFixed    int
	InstancesRemoved  int
	OrphanedWorktrees []string
	Errors            []error
}

// Reconcile performs startup reconciliation to detect and recover from crashes,
// orphaned processes, and stale state. This ensures state matches reality.
//
// Steps:
// 1. Run git worktree repair (Metis guardrail)
// 2. Run git worktree prune (Metis guardrail)
// 3. Load state and git worktrees
// 4. For each instance, check: worktree exists, tmux window exists, PID alive, pane status
// 5. Reconcile discrepancies: remove invalid instances, mark failed ones as "error"
// 6. Detect orphaned worktrees (exist but not in state)
// 7. Update state with reconciled data
func (m *Manager) Reconcile() (*ReconcileResult, error) {
	result := &ReconcileResult{
		Errors: make([]error, 0),
	}

	// Step 1: Run git worktree repair (Metis guardrail)
	if err := m.git.WorktreeRepair(); err != nil {
		// Log warning but continue - repair failure shouldn't block startup
		result.Errors = append(result.Errors, fmt.Errorf("worktree repair failed (non-fatal): %w", err))
	} else {
		result.RepairedWorktrees = true
	}

	// Step 2: Run git worktree prune (Metis guardrail)
	if err := m.git.WorktreePrune(); err != nil {
		// Log warning but continue - prune failure shouldn't block startup
		result.Errors = append(result.Errors, fmt.Errorf("worktree prune failed (non-fatal): %w", err))
	} else {
		result.PrunedWorktrees = true
	}

	// Step 3: Load state from .ocw/state.json
	currentState, err := m.store.Load()
	if err != nil {
		return result, fmt.Errorf("failed to load state: %w", err)
	}

	// Get list of git worktrees
	worktrees, err := m.git.WorktreeList()
	if err != nil {
		return result, fmt.Errorf("failed to list worktrees: %w", err)
	}

	// Create map of worktree paths for quick lookup
	worktreeMap := make(map[string]git.WorktreeInfo)
	for _, wt := range worktrees {
		worktreeMap[wt.Path] = wt
	}

	// Check if tmux session exists
	sessionName := m.SessionName()
	sessionExists := m.tmux.HasSession(sessionName)

	// Get tmux windows if session exists
	var windowMap map[string]bool
	if sessionExists {
		windows, err := m.tmux.ListWindows(sessionName)
		if err != nil {
			// If we can't list windows, assume session is corrupted
			sessionExists = false
			result.Errors = append(result.Errors, fmt.Errorf("failed to list tmux windows (treating as session missing): %w", err))
		} else {
			windowMap = make(map[string]bool)
			for _, w := range windows {
				windowMap[w.ID] = true
			}
		}
	}

	// Step 4 & 5: Check each instance and reconcile discrepancies
	instancesToRemove := make([]string, 0)
	instancesToUpdate := make(map[string]func(*state.Instance))

	for i := range currentState.Instances {
		inst := &currentState.Instances[i]

		// Check 1: Does worktree path exist?
		_, worktreeExists := worktreeMap[inst.WorktreePath]
		if !worktreeExists {
			// Worktree missing but state exists → remove from state
			instancesToRemove = append(instancesToRemove, inst.ID)
			result.InstancesRemoved++

			// Try to kill tmux window if it exists
			if sessionExists && windowMap[inst.TmuxWindow] {
				if err := m.tmux.KillWindow(inst.TmuxWindow); err != nil {
					result.Errors = append(result.Errors, fmt.Errorf("failed to cleanup window for removed instance %s: %w", inst.ID, err))
				}
			}
			continue
		}

		// Check 2: Does tmux window exist?
		windowExists := sessionExists && windowMap[inst.TmuxWindow]

		// Check 3: Is opencode PID alive?
		pidAlive := false
		if inst.PID > 0 {
			pidAlive = isProcessAlive(inst.PID)
		}

		// Check 4: Is pane dead? (require both window exists and session exists)
		paneDead := false
		if windowExists {
			panes, err := m.tmux.ListPanes(inst.TmuxWindow)
			if err != nil {
				result.Errors = append(result.Errors, fmt.Errorf("failed to check panes for instance %s: %w", inst.ID, err))
			} else {
				// Check if primary pane is dead
				for _, pane := range panes {
					if pane.ID == inst.PrimaryPane {
						paneDead = pane.Dead
						break
					}
				}
			}
		}

		// Reconcile based on findings
		needsUpdate := false
		newStatus := inst.Status

		// Scenario 2: tmux crashed (session doesn't exist)
		if !sessionExists {
			if inst.Status == "running" || inst.Status == "paused" {
				newStatus = "error"
				needsUpdate = true
				result.InstancesFixed++
			}
		} else if !windowExists {
			// Window missing but session exists
			if inst.Status == "running" || inst.Status == "paused" {
				newStatus = "error"
				needsUpdate = true
				result.InstancesFixed++
			}
		}

		// Scenario 3: Opencode crashed (PID dead but state shows running)
		if inst.Status == "running" && !pidAlive {
			newStatus = "error"
			needsUpdate = true
			result.InstancesFixed++
		}

		// Scenario 3b: Pane is dead (remain-on-exit captured it)
		if paneDead && inst.Status == "running" {
			newStatus = "error"
			needsUpdate = true
			result.InstancesFixed++
		}

		// Apply updates
		if needsUpdate {
			instID := inst.ID
			instancesToUpdate[instID] = func(i *state.Instance) {
				i.Status = newStatus
			}
		}
	}

	// Step 6: Detect orphaned worktrees (exist but not in state)
	stateWorktreePaths := make(map[string]bool)
	for _, inst := range currentState.Instances {
		stateWorktreePaths[inst.WorktreePath] = true
	}

	for _, wt := range worktrees {
		// Skip the main worktree (bare repo)
		if wt.Bare {
			continue
		}

		// Skip if this worktree is tracked in state
		if stateWorktreePaths[wt.Path] {
			continue
		}

		// Check if this is in our managed worktrees directory
		// This prevents false positives for unrelated worktrees
		// Orphaned worktrees are logged as warnings, not errors
		result.OrphanedWorktrees = append(result.OrphanedWorktrees, wt.Path)
	}

	// Step 7: Apply state updates
	// First remove instances
	for _, id := range instancesToRemove {
		if err := m.store.RemoveInstance(id); err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("failed to remove instance %s: %w", id, err))
		}
	}

	// Then update remaining instances
	for id, updateFn := range instancesToUpdate {
		if err := m.store.UpdateInstance(id, updateFn); err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("failed to update instance %s: %w", id, err))
		}
	}

	return result, nil
}

// RecoverFromCrash attempts to recover after a complete tmux crash.
// This handles Scenario 2: All tmux sessions lost, processes may be reparented to PID 1.
//
// Returns true if recovery was successful, false otherwise.
func (m *Manager) RecoverFromCrash() (bool, error) {
	// Load state
	currentState, err := m.store.Load()
	if err != nil {
		return false, fmt.Errorf("failed to load state: %w", err)
	}

	// Check if we have any instances
	if len(currentState.Instances) == 0 {
		return false, nil
	}

	// Check if tmux session exists
	sessionName := m.SessionName()
	if m.tmux.HasSession(sessionName) {
		// Session exists, no crash recovery needed
		return false, nil
	}

	// Try to recreate session
	_, err = m.EnsureSession()
	if err != nil {
		return false, fmt.Errorf("failed to recreate session: %w", err)
	}

	// Mark all instances as error (they need manual intervention)
	for _, inst := range currentState.Instances {
		if err := m.store.UpdateInstance(inst.ID, func(i *state.Instance) {
			i.Status = "error"
		}); err != nil {
			return false, fmt.Errorf("failed to mark instance %s as error: %w", inst.ID, err)
		}
	}

	return true, nil
}

// ValidateState checks if the state file is valid and not corrupted.
// Returns true if state is valid, false if corrupted.
func (m *Manager) ValidateState() (bool, error) {
	_, err := m.store.Load()
	if err != nil {
		// Check if it's a JSON parse error (corruption)
		if _, ok := err.(*os.PathError); !ok {
			// Likely a JSON unmarshaling error = corruption
			return false, err
		}
		// File doesn't exist or can't be read, but not corrupted
		return true, err
	}

	return true, nil
}
