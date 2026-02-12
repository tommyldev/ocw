package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/cobra"
	"github.com/tommyzliu/ocw/internal/config"
	"github.com/tommyzliu/ocw/internal/workspace"
	"gopkg.in/yaml.v3"
)

// TaskDefinition represents a single task from the watch file
type TaskDefinition struct {
	Name   string `yaml:"name" json:"name"`
	Branch string `yaml:"branch" json:"branch"`
	Base   string `yaml:"base" json:"base"`
}

// TaskFile represents the structure of the watch file
type TaskFile struct {
	Tasks []TaskDefinition `yaml:"tasks" json:"tasks"`
}

var watchCmd = &cobra.Command{
	Use:   "watch <file>",
	Short: "Watch a file for tasks and auto-create instances",
	Long: `Watch a YAML or JSON file containing task definitions and automatically create instances.

The file should contain a list of tasks with name, branch, and base fields:

YAML format:
  tasks:
    - name: feature-1
      branch: feature/feature-1
      base: main
    - name: bugfix-2
      branch: fix/bug-2
      base: production

JSON format:
  {
    "tasks": [
      {"name": "feature-1", "branch": "feature/feature-1", "base": "main"},
      {"name": "bugfix-2", "branch": "fix/bug-2", "base": "production"}
    ]
  }

The watch command will:
1. Parse the file and create instances for each task (if not already exists)
2. Watch the file for changes
3. Create new instances when tasks are added
4. Run in foreground until interrupted with Ctrl+C`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		filePath := args[0]

		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}

		repoRoot := cwd
		for {
			if _, err := os.Stat(filepath.Join(repoRoot, ".git")); err == nil {
				break
			}
			parent := filepath.Dir(repoRoot)
			if parent == repoRoot {
				repoRoot = cwd
				break
			}
			repoRoot = parent
		}

		cfg, err := config.LoadConfig(repoRoot)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		mgr, err := workspace.NewManager(repoRoot, cfg)
		if err != nil {
			return fmt.Errorf("failed to create workspace manager: %w", err)
		}

		fmt.Printf("📋 Processing tasks from: %s\n", filePath)
		if err := processTasks(mgr, filePath, cfg); err != nil {
			return fmt.Errorf("failed to process tasks: %w", err)
		}

		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			return fmt.Errorf("failed to create file watcher: %w", err)
		}
		defer watcher.Close()

		if err := watcher.Add(filePath); err != nil {
			return fmt.Errorf("failed to watch file: %w", err)
		}

		fmt.Printf("👀 Watching %s for changes (Press Ctrl+C to stop)...\n\n", filePath)

		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return nil
				}
				if event.Op&fsnotify.Write == fsnotify.Write {
					fmt.Printf("\n📝 File changed, processing tasks...\n")
					if err := processTasks(mgr, filePath, cfg); err != nil {
						fmt.Fprintf(os.Stderr, "Error processing tasks: %v\n", err)
					}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return nil
				}
				fmt.Fprintf(os.Stderr, "Watcher error: %v\n", err)
			case <-sigChan:
				fmt.Printf("\n\n🛑 Stopping watch...\n")
				return nil
			}
		}
	},
}

// processTasks reads the task file and creates instances for any new tasks
func processTasks(mgr *workspace.Manager, filePath string, cfg *config.Config) error {
	tasks, err := parseTaskFile(filePath)
	if err != nil {
		return err
	}

	if len(tasks.Tasks) == 0 {
		fmt.Printf("⚠️  No tasks found in file\n")
		return nil
	}

	existingInstances, err := mgr.ListInstances()
	if err != nil {
		return fmt.Errorf("failed to list instances: %w", err)
	}

	existingBranches := make(map[string]bool)
	for _, inst := range existingInstances {
		existingBranches[inst.Branch] = true
	}

	created := 0
	skipped := 0
	for _, task := range tasks.Tasks {
		if existingBranches[task.Branch] {
			skipped++
			fmt.Printf("  ⏭️  Skipping %s (instance already exists for branch %s)\n", task.Name, task.Branch)
			continue
		}

		opts := workspace.CreateOpts{
			Name:       task.Name,
			Branch:     task.Branch,
			BaseBranch: task.Base,
		}

		if opts.BaseBranch == "" {
			opts.BaseBranch = cfg.Workspace.BaseBranch
		}

		instance, err := mgr.CreateInstance(opts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  ❌ Failed to create instance for %s: %v\n", task.Name, err)
			continue
		}

		created++
		fmt.Printf("  ✅ Created instance: %s\n", task.Name)
		fmt.Printf("     ID:       %s\n", instance.ID)
		fmt.Printf("     Branch:   %s\n", instance.Branch)
		fmt.Printf("     Base:     %s\n", instance.BaseBranch)
		fmt.Printf("     Worktree: %s\n", instance.WorktreePath)
	}

	fmt.Printf("\n📊 Summary: %d created, %d skipped, %d total\n", created, skipped, len(tasks.Tasks))
	return nil
}

// parseTaskFile parses a YAML or JSON task file
func parseTaskFile(filePath string) (*TaskFile, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var tasks TaskFile

	ext := filepath.Ext(filePath)
	switch ext {
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, &tasks); err != nil {
			return nil, fmt.Errorf("failed to parse YAML: %w", err)
		}
	case ".json":
		if err := json.Unmarshal(data, &tasks); err != nil {
			return nil, fmt.Errorf("failed to parse JSON: %w", err)
		}
	default:
		if err := yaml.Unmarshal(data, &tasks); err != nil {
			if jsonErr := json.Unmarshal(data, &tasks); jsonErr != nil {
				return nil, fmt.Errorf("failed to parse file as YAML or JSON: YAML error: %v, JSON error: %v", err, jsonErr)
			}
		}
	}

	for i, task := range tasks.Tasks {
		if task.Branch == "" {
			return nil, fmt.Errorf("task %d: branch field is required", i+1)
		}
		if task.Name == "" {
			tasks.Tasks[i].Name = task.Branch
		}
	}

	return &tasks, nil
}

func init() {
	rootCmd.AddCommand(watchCmd)
}
