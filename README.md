# OCW - Open Code Workspace

A terminal-based workspace manager for managing parallel development workflows across isolated git worktrees. OCW orchestrates multiple work-in-progress branches using tmux and git worktrees, providing a keyboard-driven TUI dashboard for monitoring, switching, and merging your work.

## Features

- **Parallel Development**: Work on multiple branches simultaneously using git worktrees
- **Tmux Integration**: Each workspace gets its own tmux window with dedicated sub-terminal panes
- **Interactive TUI Dashboard**: Monitor all workspaces, view status, and navigate with keyboard shortcuts
- **Conflict Detection**: Automatically detect overlapping changes across instances
- **IDE Integration**: Launch your preferred IDE/editor for each workspace
- **PR Creation**: Create pull requests directly from the merge view (supports `gh` and `glab`)
- **State Management**: Persistent state tracking with automatic reconciliation on startup
- **Sub-terminals**: Create and manage multiple terminal panes per workspace instance

## Prerequisites

OCW requires the following tools to be installed:

- **Git** (2.5+): For git worktree support
- **Tmux** (2.0+): For terminal multiplexing
- **Go** (1.21+): For building from source
- **GitHub CLI** (`gh`) or **GitLab CLI** (`glab`): Optional, for PR creation

## Installation

### From Source

```bash
# Clone the repository
git clone https://github.com/tommyzliu/ocw.git
cd ocw

# Build and install
go install

# Or build locally
go build -o ocw
```

### Using Go Install

```bash
go install github.com/tommyzliu/ocw@latest
```

## Quick Start

1. **Initialize OCW in your git repository**:
   ```bash
   cd /path/to/your/repo
   ocw init
   ```

2. **Launch the TUI dashboard**:
   ```bash
   ocw
   ```

3. **Create a new workspace** (from dashboard or CLI):
   ```bash
   # Via CLI
   ocw new feature/my-feature

   # Or press 'n' in the dashboard
   ```

4. **Focus on a workspace**:
   ```bash
   # Via CLI
   ocw focus feature/my-feature

   # Or select in dashboard and press Enter
   ```

## Usage

### Commands

OCW provides both a TUI interface and CLI commands:

#### Initialization
```bash
ocw init              # Initialize OCW in current git repository
```

#### Instance Management
```bash
ocw new <branch>      # Create new workspace instance
ocw new <branch> -b <base-branch>  # Create from specific base branch
ocw list              # List all workspace instances
ocw delete <id>       # Delete a workspace instance
ocw status <id>       # Show detailed status of an instance
ocw kill <id>         # Force kill an instance and its processes
```

#### Navigation
```bash
ocw                   # Launch TUI dashboard (or reattach if session exists)
ocw focus <id>        # Focus on a specific workspace (attach to tmux window)
ocw term <id>         # Open sub-terminal for an instance
ocw edit <id>         # Launch IDE for instance
```

#### Code Management
```bash
ocw diff <id>         # View diff against base branch
ocw merge <id>        # Merge workspace (creates PR)
```

#### Configuration
```bash
ocw config            # Edit OCW configuration
```

### TUI Dashboard

The dashboard provides an interactive interface with the following views:

- **Dashboard View**: Overview of all workspace instances
- **Create View**: Form for creating new instances
- **Focus View**: Attach to a workspace's tmux window
- **Diff View**: View changes in a workspace
- **Merge View**: Review changes and create pull requests

#### Keyboard Shortcuts

**Dashboard View**:
- `n` - Create new instance
- `Enter` - Focus on selected instance
- `d` - Show diff for selected instance
- `m` - Merge selected instance
- `x` - Delete selected instance
- `r` - Refresh view
- `q` - Quit
- `?` - Show help

**Navigation**:
- `↑/↓` or `j/k` - Move selection
- `Esc` - Return to previous view
- `Ctrl+C` - Quit

## Configuration

OCW stores its configuration in `.ocw/config.toml` at the repository root. After running `ocw init`, you can customize:

```toml
[workspace]
base_branch = "main"          # Default base branch for new instances
worktree_dir = ".worktrees"   # Directory for git worktrees

[tmux]
session_prefix = "ocw"        # Prefix for tmux session names
default_shell = "zsh"         # Shell for new panes

[editor]
command = "code"              # IDE/editor command (e.g., "code", "nvim", "emacs")
args = ["."]                  # Arguments passed to editor

[git]
pr_tool = "gh"                # PR tool: "gh" or "glab"
```

### State Management

OCW maintains workspace state in `.ocw/state.json`. This file tracks:
- Active instances and their statuses
- Worktree paths and branch information
- Tmux window/pane associations
- Sub-terminal configurations

**Note**: The `.ocw` directory should be added to `.gitignore` as it contains local workspace state.

## Architecture

OCW is built with:

- **Go**: Primary language
- **Cobra**: CLI framework
- **Bubbletea**: Terminal UI framework
- **Lipgloss**: Terminal styling
- **Tmux**: Terminal multiplexing
- **Git Worktrees**: Isolated working directories

### Project Structure

```
ocw/
├── cmd/              # CLI command implementations
├── internal/
│   ├── config/       # Configuration management
│   ├── deps/         # Dependency checking
│   ├── git/          # Git operations (worktree, diff, merge)
│   ├── ide/          # IDE launcher
│   ├── state/        # State persistence
│   ├── tmux/         # Tmux integration
│   ├── tui/          # Terminal UI (Bubbletea)
│   └── workspace/    # Workspace management
└── main.go
```

## How It Works

1. **Initialization**: `ocw init` creates the `.ocw` directory with configuration and state files
2. **Instance Creation**: When you create a new instance, OCW:
   - Creates a git worktree for the branch
   - Creates a tmux window within the OCW session
   - Launches your configured IDE
   - Tracks the instance in state
3. **Focus**: Attach to an instance's tmux window to work on that branch
4. **Sub-terminals**: Create additional terminal panes within an instance for running tests, servers, etc.
5. **Merge**: Push your branch and create a PR using GitHub or GitLab CLI
6. **Cleanup**: Delete an instance to remove the worktree, tmux window, and state entry

## Troubleshooting

### Instance shows as "error" status
OCW performs startup reconciliation to detect crashed or stopped instances. You can:
- Kill the instance: `ocw kill <id>`
- Delete and recreate: `ocw delete <id> && ocw new <branch>`

### Orphaned worktrees
If you manually delete worktrees outside of OCW, run the dashboard to trigger reconciliation, or use:
```bash
git worktree prune
```

### Tmux session not found
If the tmux session was killed manually, simply run `ocw` again to create a new session.

## Development

### Building
```bash
go build -o ocw
```

### Running Tests
```bash
go test ./...
```

### Dependencies
```bash
go mod tidy
```

## Contributing

Contributions are welcome! Please ensure:
- Code follows Go best practices
- Commands include helpful error messages
- Edge cases are handled gracefully

## License

[Specify your license here]

## Acknowledgments

OCW draws inspiration from:
- [claude-squad](https://github.com/smtg-ai/claude-squad) - Similar concept for Claude Code
- [lazygit](https://github.com/jesseduffield/lazygit) - Git worktree patterns
- [sesh](https://github.com/joshmedeski/sesh) - Tmux session management
- [gh-dash](https://github.com/dlvhdr/gh-dash) - Bubbletea TUI patterns
