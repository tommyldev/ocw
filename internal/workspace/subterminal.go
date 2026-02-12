package workspace

import (
	"fmt"
	"time"

	"github.com/tommyzliu/ocw/internal/state"
)

// CreateSubTerminal creates a new sub-terminal pane for an instance.
// First sub-terminal: horizontal split (-v flag, 70/30 ratio)
// Second sub-terminal: vertical split (-h flag in bottom area)
// Returns the new pane ID.
func (m *Manager) CreateSubTerminal(instanceID, label string) (string, error) {
	inst, err := m.GetInstance(instanceID)
	if err != nil {
		return "", fmt.Errorf("failed to get instance: %w", err)
	}

	if inst == nil {
		return "", fmt.Errorf("instance %q not found", instanceID)
	}

	count := len(inst.SubTerminals)

	const maxPanes = 6
	if count >= maxPanes {
		return "", fmt.Errorf("maximum sub-terminals reached (%d/%d)\n\nToo many panes can make the terminal difficult to use.\n\nTo fix:\n  1. Close unused sub-terminals first\n  2. Or use a larger terminal window\n  3. Consider using tmux windows instead of panes", count, maxPanes)
	}

	// Determine split direction and percentage
	var split string
	var percentage int

	if count == 0 {
		// First sub-terminal: horizontal split (vertical in tmux terms, -v flag)
		// This splits the window top/bottom, with new pane taking 30% at bottom
		split = "vertical"
		percentage = 30
	} else if count == 1 {
		// Second sub-terminal: vertical split (-h flag)
		// This splits the bottom pane left/right
		split = "horizontal"
		percentage = 50
	} else {
		// For additional sub-terminals, alternate or warn
		// For now, use horizontal split with 50%
		split = "horizontal"
		percentage = 50
	}

	// Get primary pane target (window ID)
	target := inst.TmuxWindow

	// Split the window to create new pane
	newPaneID, err := m.tmux.SplitWindow(target, inst.WorktreePath, split, percentage)
	if err != nil {
		return "", fmt.Errorf("failed to split window: %w", err)
	}

	// Send init command if configured
	if m.config.Workspace.SubTerminalInitCommand != "" {
		if err := m.tmux.SendKeys(newPaneID, m.config.Workspace.SubTerminalInitCommand); err != nil {
			// Log error but continue - don't fail the sub-terminal creation
			fmt.Printf("warning: failed to send init command to sub-terminal: %v\n", err)
		}
	}

	// Update state with new sub-terminal
	err = m.store.UpdateInstance(instanceID, func(i *state.Instance) {
		i.SubTerminals = append(i.SubTerminals, state.SubTerminal{
			PaneID:    newPaneID,
			Label:     label,
			CreatedAt: time.Now(),
		})
	})
	if err != nil {
		// Try to clean up the pane if state update fails
		_ = m.tmux.KillPane(newPaneID)
		return "", fmt.Errorf("failed to update state: %w", err)
	}

	return newPaneID, nil
}

// ListSubTerminals returns all sub-terminals for an instance.
func (m *Manager) ListSubTerminals(instanceID string) ([]state.SubTerminal, error) {
	inst, err := m.GetInstance(instanceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get instance: %w", err)
	}

	if inst == nil {
		return nil, fmt.Errorf("instance %q not found", instanceID)
	}

	return inst.SubTerminals, nil
}

// KillSubTerminal kills a specific sub-terminal pane and removes it from state.
func (m *Manager) KillSubTerminal(instanceID, paneID string) error {
	// Kill the pane
	if err := m.tmux.KillPane(paneID); err != nil {
		return fmt.Errorf("failed to kill pane: %w", err)
	}

	// Update state to remove the sub-terminal
	err := m.store.UpdateInstance(instanceID, func(i *state.Instance) {
		// Filter out the sub-terminal with the specified pane ID
		filtered := make([]state.SubTerminal, 0, len(i.SubTerminals))
		for _, st := range i.SubTerminals {
			if st.PaneID != paneID {
				filtered = append(filtered, st)
			}
		}
		i.SubTerminals = filtered
	})
	if err != nil {
		return fmt.Errorf("failed to update state: %w", err)
	}

	return nil
}

// KillAllSubTerminals kills all sub-terminal panes for an instance.
func (m *Manager) KillAllSubTerminals(instanceID string) error {
	inst, err := m.GetInstance(instanceID)
	if err != nil {
		return fmt.Errorf("failed to get instance: %w", err)
	}

	if inst == nil {
		return fmt.Errorf("instance %q not found", instanceID)
	}

	// Kill all sub-terminal panes
	for _, st := range inst.SubTerminals {
		if err := m.tmux.KillPane(st.PaneID); err != nil {
			// Log error but continue killing other panes
			fmt.Printf("warning: failed to kill pane %q: %v\n", st.PaneID, err)
		}
	}

	// Clear sub-terminals from state
	err = m.store.UpdateInstance(instanceID, func(i *state.Instance) {
		i.SubTerminals = []state.SubTerminal{}
	})
	if err != nil {
		return fmt.Errorf("failed to update state: %w", err)
	}

	return nil
}

// SubTerminalCount returns the number of sub-terminals for an instance.
func (m *Manager) SubTerminalCount(instanceID string) int {
	inst, err := m.GetInstance(instanceID)
	if err != nil {
		return 0
	}

	if inst == nil {
		return 0
	}

	return len(inst.SubTerminals)
}
