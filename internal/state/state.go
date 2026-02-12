package state

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/gofrs/flock"
)

// State represents the complete OCW state
type State struct {
	Repo        string     `json:"repo"`
	TmuxSession string     `json:"tmux_session"`
	Instances   []Instance `json:"instances"`
}

// Instance represents a single OCW instance
type Instance struct {
	ID            string        `json:"id"`
	Name          string        `json:"name"`
	Branch        string        `json:"branch"`
	BaseBranch    string        `json:"base_branch"`
	WorktreePath  string        `json:"worktree_path"`
	TmuxWindow    string        `json:"tmux_window"`
	PrimaryPane   string        `json:"primary_pane"`
	SubTerminals  []SubTerminal `json:"sub_terminals"`
	PID           int           `json:"pid"`
	Port          int           `json:"port,omitempty"`
	Status        string        `json:"status"`
	CreatedAt     time.Time     `json:"created_at"`
	LastActivity  time.Time     `json:"last_activity"`
	PRUrl         string        `json:"pr_url,omitempty"`
	ConflictsWith []string      `json:"conflicts_with"`
	DependsOn     []string      `json:"depends_on"`
}

// SubTerminal represents a sub-terminal within an instance
type SubTerminal struct {
	PaneID    string    `json:"pane_id"`
	Label     string    `json:"label"`
	CreatedAt time.Time `json:"created_at"`
}

// Store manages state persistence with file locking
type Store struct {
	dir string
}

// NewStore creates a new Store for the specified directory
func NewStore(dir string) *Store {
	return &Store{dir: dir}
}

func (s *Store) statePath() string {
	return filepath.Join(s.dir, ".ocw", "state.json")
}

func (s *Store) lockPath() string {
	return filepath.Join(s.dir, ".ocw", "state.json.lock")
}

// Load reads the state from state.json with read lock
func (s *Store) Load() (*State, error) {
	statePath := s.statePath()
	lockPath := s.lockPath()

	// Ensure .ocw directory exists (needed for lock file)
	ocwDir := filepath.Join(s.dir, ".ocw")
	if err := os.MkdirAll(ocwDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create .ocw directory: %w", err)
	}

	// Create lock
	lock := flock.New(lockPath)
	if err := lock.RLock(); err != nil {
		return nil, fmt.Errorf("failed to acquire read lock: %w", err)
	}
	defer lock.Unlock()

	// Check if state file exists
	if _, err := os.Stat(statePath); os.IsNotExist(err) {
		// Return empty state if file doesn't exist
		return &State{
			Instances: []Instance{},
		}, nil
	}

	// Read file
	data, err := os.ReadFile(statePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	// Parse JSON
	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to parse state file: %w", err)
	}

	// Ensure instances is not nil
	if state.Instances == nil {
		state.Instances = []Instance{}
	}

	return &state, nil
}

// Save writes the state to state.json with write lock and atomic write
func (s *Store) Save(state *State) error {
	statePath := s.statePath()
	lockPath := s.lockPath()

	// Ensure .ocw directory exists
	ocwDir := filepath.Join(s.dir, ".ocw")
	if err := os.MkdirAll(ocwDir, 0755); err != nil {
		return fmt.Errorf("failed to create .ocw directory: %w", err)
	}

	// Create lock
	lock := flock.New(lockPath)
	if err := lock.Lock(); err != nil {
		return fmt.Errorf("failed to acquire write lock: %w", err)
	}
	defer lock.Unlock()

	// Marshal to JSON with indentation
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	// Write to temp file first (atomic write)
	tmpPath := statePath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	// Rename temp file to actual file (atomic operation)
	if err := os.Rename(tmpPath, statePath); err != nil {
		os.Remove(tmpPath) // Clean up temp file on error
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}

// AddInstance adds a new instance to the state
func (s *Store) AddInstance(inst Instance) error {
	state, err := s.Load()
	if err != nil {
		return fmt.Errorf("failed to load state: %w", err)
	}

	state.Instances = append(state.Instances, inst)

	if err := s.Save(state); err != nil {
		return fmt.Errorf("failed to save state: %w", err)
	}

	return nil
}

// RemoveInstance removes an instance from the state by ID
func (s *Store) RemoveInstance(id string) error {
	state, err := s.Load()
	if err != nil {
		return fmt.Errorf("failed to load state: %w", err)
	}

	// Filter out the instance with the specified ID
	filtered := make([]Instance, 0, len(state.Instances))
	for _, inst := range state.Instances {
		if inst.ID != id {
			filtered = append(filtered, inst)
		}
	}

	state.Instances = filtered

	if err := s.Save(state); err != nil {
		return fmt.Errorf("failed to save state: %w", err)
	}

	return nil
}

// UpdateInstance updates an instance by ID using the provided function
func (s *Store) UpdateInstance(id string, fn func(*Instance)) error {
	state, err := s.Load()
	if err != nil {
		return fmt.Errorf("failed to load state: %w", err)
	}

	// Find and update the instance
	found := false
	for i := range state.Instances {
		if state.Instances[i].ID == id {
			fn(&state.Instances[i])
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("instance with ID %s not found", id)
	}

	if err := s.Save(state); err != nil {
		return fmt.Errorf("failed to save state: %w", err)
	}

	return nil
}

// GenerateID generates a unique 6-character hex ID
func GenerateID() (string, error) {
	bytes := make([]byte, 3) // 3 bytes = 6 hex characters
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random ID: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}
