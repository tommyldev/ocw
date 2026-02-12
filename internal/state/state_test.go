package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStore(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)

	assert.NotNil(t, store)
	assert.Equal(t, tmpDir, store.dir)
}

func TestStateJSONRoundtrip(t *testing.T) {
	tests := []struct {
		name  string
		state State
	}{
		{
			name: "empty state",
			state: State{
				Repo:        "",
				TmuxSession: "",
				Instances:   []Instance{},
			},
		},
		{
			name: "state with single instance",
			state: State{
				Repo:        "/path/to/repo",
				TmuxSession: "ocw-session",
				Instances: []Instance{
					{
						ID:            "abc123",
						Name:          "feature-1",
						Branch:        "feature/test",
						BaseBranch:    "main",
						WorktreePath:  "/path/to/worktree",
						TmuxWindow:    "1",
						PrimaryPane:   "1.1",
						SubTerminals:  []SubTerminal{},
						PID:           12345,
						Port:          8080,
						Status:        "active",
						CreatedAt:     time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
						LastActivity:  time.Date(2024, 1, 1, 13, 0, 0, 0, time.UTC),
						PRUrl:         "https://github.com/user/repo/pull/1",
						ConflictsWith: []string{},
					},
				},
			},
		},
		{
			name: "state with multiple instances",
			state: State{
				Repo:        "/path/to/repo",
				TmuxSession: "ocw-main",
				Instances: []Instance{
					{
						ID:            "abc123",
						Name:          "feature-1",
						Branch:        "feature/test-1",
						BaseBranch:    "main",
						WorktreePath:  "/path/to/worktree1",
						TmuxWindow:    "1",
						PrimaryPane:   "1.1",
						SubTerminals:  []SubTerminal{},
						PID:           12345,
						Status:        "active",
						CreatedAt:     time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
						LastActivity:  time.Date(2024, 1, 1, 13, 0, 0, 0, time.UTC),
						ConflictsWith: []string{"def456"},
					},
					{
						ID:           "def456",
						Name:         "feature-2",
						Branch:       "feature/test-2",
						BaseBranch:   "main",
						WorktreePath: "/path/to/worktree2",
						TmuxWindow:   "2",
						PrimaryPane:  "2.1",
						SubTerminals: []SubTerminal{
							{
								PaneID:    "2.2",
								Label:     "tests",
								CreatedAt: time.Date(2024, 1, 1, 12, 30, 0, 0, time.UTC),
							},
						},
						PID:           12346,
						Status:        "active",
						CreatedAt:     time.Date(2024, 1, 1, 12, 15, 0, 0, time.UTC),
						LastActivity:  time.Date(2024, 1, 1, 13, 15, 0, 0, time.UTC),
						ConflictsWith: []string{"abc123"},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal to JSON
			data, err := json.Marshal(tt.state)
			require.NoError(t, err)

			// Unmarshal back
			var decoded State
			err = json.Unmarshal(data, &decoded)
			require.NoError(t, err)

			// Compare
			assert.Equal(t, tt.state.Repo, decoded.Repo)
			assert.Equal(t, tt.state.TmuxSession, decoded.TmuxSession)
			assert.Equal(t, len(tt.state.Instances), len(decoded.Instances))

			for i := range tt.state.Instances {
				assert.Equal(t, tt.state.Instances[i].ID, decoded.Instances[i].ID)
				assert.Equal(t, tt.state.Instances[i].Name, decoded.Instances[i].Name)
				assert.Equal(t, tt.state.Instances[i].Branch, decoded.Instances[i].Branch)
				assert.Equal(t, tt.state.Instances[i].Status, decoded.Instances[i].Status)
			}
		})
	}
}

func TestStoreSaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)

	state := &State{
		Repo:        "/test/repo",
		TmuxSession: "test-session",
		Instances: []Instance{
			{
				ID:            "test123",
				Name:          "test-feature",
				Branch:        "feature/test",
				BaseBranch:    "main",
				WorktreePath:  "/test/worktree",
				TmuxWindow:    "1",
				PrimaryPane:   "1.1",
				SubTerminals:  []SubTerminal{},
				PID:           99999,
				Status:        "active",
				CreatedAt:     time.Now(),
				LastActivity:  time.Now(),
				ConflictsWith: []string{},
			},
		},
	}

	// Save state
	err := store.Save(state)
	require.NoError(t, err)

	// Verify .ocw directory was created
	ocwDir := filepath.Join(tmpDir, ".ocw")
	_, err = os.Stat(ocwDir)
	assert.NoError(t, err)

	// Verify state.json was created
	statePath := filepath.Join(ocwDir, "state.json")
	_, err = os.Stat(statePath)
	assert.NoError(t, err)

	// Load state
	loaded, err := store.Load()
	require.NoError(t, err)

	// Verify loaded state matches saved state
	assert.Equal(t, state.Repo, loaded.Repo)
	assert.Equal(t, state.TmuxSession, loaded.TmuxSession)
	assert.Equal(t, len(state.Instances), len(loaded.Instances))
	assert.Equal(t, state.Instances[0].ID, loaded.Instances[0].ID)
	assert.Equal(t, state.Instances[0].Name, loaded.Instances[0].Name)
}

func TestLoadNonexistentState(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)

	// Load from nonexistent file should return empty state
	state, err := store.Load()
	require.NoError(t, err)

	assert.NotNil(t, state)
	assert.Equal(t, "", state.Repo)
	assert.Equal(t, "", state.TmuxSession)
	assert.NotNil(t, state.Instances)
	assert.Equal(t, 0, len(state.Instances))
}

func TestAddInstance(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)

	instance := Instance{
		ID:            "test123",
		Name:          "test-feature",
		Branch:        "feature/test",
		BaseBranch:    "main",
		WorktreePath:  "/test/worktree",
		Status:        "active",
		ConflictsWith: []string{},
	}

	// Add instance
	err := store.AddInstance(instance)
	require.NoError(t, err)

	// Load and verify
	state, err := store.Load()
	require.NoError(t, err)

	assert.Equal(t, 1, len(state.Instances))
	assert.Equal(t, "test123", state.Instances[0].ID)
	assert.Equal(t, "test-feature", state.Instances[0].Name)
}

func TestRemoveInstance(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)

	// Create initial state with two instances
	state := &State{
		Instances: []Instance{
			{ID: "inst1", Name: "feature-1"},
			{ID: "inst2", Name: "feature-2"},
		},
	}
	err := store.Save(state)
	require.NoError(t, err)

	// Remove one instance
	err = store.RemoveInstance("inst1")
	require.NoError(t, err)

	// Load and verify
	loaded, err := store.Load()
	require.NoError(t, err)

	assert.Equal(t, 1, len(loaded.Instances))
	assert.Equal(t, "inst2", loaded.Instances[0].ID)
}

func TestUpdateInstance(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)

	// Create initial state
	state := &State{
		Instances: []Instance{
			{ID: "inst1", Name: "feature-1", Status: "active"},
		},
	}
	err := store.Save(state)
	require.NoError(t, err)

	// Update instance
	err = store.UpdateInstance("inst1", func(inst *Instance) {
		inst.Status = "completed"
		inst.PRUrl = "https://github.com/test/repo/pull/1"
	})
	require.NoError(t, err)

	// Load and verify
	loaded, err := store.Load()
	require.NoError(t, err)

	assert.Equal(t, 1, len(loaded.Instances))
	assert.Equal(t, "completed", loaded.Instances[0].Status)
	assert.Equal(t, "https://github.com/test/repo/pull/1", loaded.Instances[0].PRUrl)
}

func TestUpdateInstanceNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)

	// Create empty state
	state := &State{Instances: []Instance{}}
	err := store.Save(state)
	require.NoError(t, err)

	// Try to update nonexistent instance
	err = store.UpdateInstance("nonexistent", func(inst *Instance) {
		inst.Status = "completed"
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestGenerateID(t *testing.T) {
	// Generate multiple IDs
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id, err := GenerateID()
		require.NoError(t, err)

		// Should be 6 characters (3 bytes * 2 hex chars)
		assert.Equal(t, 6, len(id))

		// Should be unique
		assert.False(t, ids[id], "Generated duplicate ID: %s", id)
		ids[id] = true

		// Should only contain hex characters
		for _, c := range id {
			assert.True(t, (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f'),
				"ID contains non-hex character: %c", c)
		}
	}
}

func TestAtomicWrite(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)

	state := &State{
		Repo:      "/test/repo",
		Instances: []Instance{{ID: "test1"}},
	}

	// Save state
	err := store.Save(state)
	require.NoError(t, err)

	// Verify temp file was cleaned up
	tmpPath := filepath.Join(tmpDir, ".ocw", "state.json.tmp")
	_, err = os.Stat(tmpPath)
	assert.True(t, os.IsNotExist(err), "Temp file should be removed after save")
}

func TestStatePathsCorrect(t *testing.T) {
	tmpDir := "/test/dir"
	store := NewStore(tmpDir)

	expectedStatePath := filepath.Join(tmpDir, ".ocw", "state.json")
	expectedLockPath := filepath.Join(tmpDir, ".ocw", "state.json.lock")

	assert.Equal(t, expectedStatePath, store.statePath())
	assert.Equal(t, expectedLockPath, store.lockPath())
}
