package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/tommyzliu/ocw/internal/config"
	"github.com/tommyzliu/ocw/internal/tui"
	"github.com/tommyzliu/ocw/internal/workspace"
)

var rootCmd = &cobra.Command{
	Use:   "ocw",
	Short: "OCW - Open Code Workspace",
	Long:  "OCW is a terminal-based workspace manager for open source development",
	Run: func(cmd *cobra.Command, args []string) {
		if err := runTUI(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

// runTUI launches the Bubbletea TUI application
func runTUI() error {
	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Find git repository root
	repoRoot := cwd
	for {
		if _, err := os.Stat(filepath.Join(repoRoot, ".git")); err == nil {
			break
		}
		parent := filepath.Dir(repoRoot)
		if parent == repoRoot {
			// Reached filesystem root without finding .git
			repoRoot = cwd
			break
		}
		repoRoot = parent
	}

	// Load configuration
	cfg, err := config.LoadConfig(repoRoot)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Create workspace manager
	mgr, err := workspace.NewManager(repoRoot, cfg)
	if err != nil {
		// If manager creation fails, create a minimal context for demo
		// This allows the TUI to run even without a full workspace setup
		ctx := tui.NewContext(cfg, nil)
		app := tui.NewApp(ctx)
		p := tea.NewProgram(app, tea.WithAltScreen())
		app.SetProgram(p)
		if _, err := p.Run(); err != nil {
			return fmt.Errorf("failed to run TUI: %w", err)
		}
		return nil
	}

	result, err := mgr.Reconcile()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Reconciliation failed: %v\n", err)
	} else if result != nil {
		if result.InstancesFixed > 0 || result.InstancesRemoved > 0 || len(result.OrphanedWorktrees) > 0 {
			fmt.Fprintf(os.Stderr, "Startup Reconciliation:\n")
			if result.InstancesFixed > 0 {
				fmt.Fprintf(os.Stderr, "  - Marked %d instance(s) as error (crashed/stopped)\n", result.InstancesFixed)
			}
			if result.InstancesRemoved > 0 {
				fmt.Fprintf(os.Stderr, "  - Removed %d instance(s) (missing worktrees)\n", result.InstancesRemoved)
			}
			if len(result.OrphanedWorktrees) > 0 {
				fmt.Fprintf(os.Stderr, "  - Found %d orphaned worktree(s) (not in state)\n", len(result.OrphanedWorktrees))
			}
			if len(result.Errors) > 0 {
				fmt.Fprintf(os.Stderr, "  - %d warning(s) during reconciliation\n", len(result.Errors))
			}
			fmt.Fprintf(os.Stderr, "\n")
		}
	}

	// Create TUI context
	ctx := tui.NewContext(cfg, mgr)

	// Create and run app
	app := tui.NewApp(ctx)
	p := tea.NewProgram(app, tea.WithAltScreen())
	app.SetProgram(p)
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("failed to run TUI: %w", err)
	}

	return nil
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		panic(err)
	}
}
