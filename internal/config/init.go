package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/tommyzliu/ocw/internal/state"
)

// InitWorkspace creates the .ocw directory with default config.toml and empty state.json
func InitWorkspace(repoRoot string) error {
	ocwDir := filepath.Join(repoRoot, ".ocw")

	// Create .ocw directory
	if err := os.MkdirAll(ocwDir, 0755); err != nil {
		return fmt.Errorf("failed to create .ocw directory: %w", err)
	}

	// Write default config.toml
	defaultConfig := DefaultConfig()
	if err := SaveConfig(repoRoot, defaultConfig); err != nil {
		return fmt.Errorf("failed to save default config: %w", err)
	}

	// Write empty state.json
	emptyState := &state.State{
		Instances: []state.Instance{},
	}

	statePath := filepath.Join(ocwDir, "state.json")
	stateData, err := json.MarshalIndent(emptyState, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal empty state: %w", err)
	}

	if err := os.WriteFile(statePath, stateData, 0644); err != nil {
		return fmt.Errorf("failed to write state.json: %w", err)
	}

	return nil
}
