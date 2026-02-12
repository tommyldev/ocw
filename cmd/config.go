package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Open or display config file",
	Long:  "Open the config.toml file in $EDITOR, or display its path with --path",
	RunE: func(cmd *cobra.Command, args []string) error {
		showPath, _ := cmd.Flags().GetBool("path")

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
				return fmt.Errorf("not in a git repository")
			}
			repoRoot = parent
		}

		// Check if .ocw exists
		ocwDir := filepath.Join(repoRoot, ".ocw")
		if _, err := os.Stat(ocwDir); os.IsNotExist(err) {
			return fmt.Errorf(".ocw directory not found; run 'ocw init' first")
		}

		configPath := filepath.Join(ocwDir, "config.toml")

		// If --path flag, just print the path
		if showPath {
			fmt.Println(configPath)
			return nil
		}

		// Try to open in $EDITOR
		editor := os.Getenv("EDITOR")
		if editor == "" {
			// Fallback: cat the file
			content, err := os.ReadFile(configPath)
			if err != nil {
				return fmt.Errorf("failed to read config file: %w", err)
			}
			fmt.Println(string(content))
			return nil
		}

		// Open in editor
		editorCmd := exec.Command(editor, configPath)
		editorCmd.Stdin = os.Stdin
		editorCmd.Stdout = os.Stdout
		editorCmd.Stderr = os.Stderr

		if err := editorCmd.Run(); err != nil {
			return fmt.Errorf("failed to open editor: %w", err)
		}

		return nil
	},
}

func init() {
	configCmd.Flags().Bool("path", false, "Print config file path instead of opening")
	rootCmd.AddCommand(configCmd)
}
