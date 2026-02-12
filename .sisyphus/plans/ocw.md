# OCW — OpenCode Workspace Manager

## TL;DR

> **Quick Summary**: Build a Go TUI multiplexer (Bubbletea + Lipgloss + Cobra) that orchestrates parallel OpenCode instances across isolated git worktrees via tmux, with sub-terminal panes, PR-based merging, and a keyboard-driven dashboard.
> 
> **Deliverables**:
> - `ocw` Go binary with CLI subcommands and TUI dashboard
> - Workspace manager: git worktree lifecycle + tmux session/window/pane orchestration
> - Dashboard view, Instance/Focus view, Create view, Diff view, Merge view
> - Sub-terminal pane management per instance
> - IDE launcher, PR creation (gh/glab), conflict detection
> - State persistence (.ocw/state.json) and config (.ocw/config.toml)
> - Post-MVP: output streaming, send prompts, named sub-terminals, plugins, ocw watch
> 
> **Estimated Effort**: XL (30+ tasks across MVP and post-MVP)
> **Parallel Execution**: YES — 6 waves
> **Critical Path**: Task 1 (project scaffold) → Task 2 (state/config) → Task 4 (tmux) → Task 5 (worktree) → Task 6 (dashboard TUI) → Task 8 (instance creation) → Task 12 (focus view)

---

## Context

### Original Request
Build OCW — a TUI multiplexer for managing parallel OpenCode instances across isolated git worktrees. Full spec provided covering architecture, workflow, hotkeys, state management, configuration, CLI interface, and edge cases.

### Interview Summary
**Key Discussions**:
- Spec v2 replaces port management with sub-terminals (N tmux panes per instance)
- Full spec scope: MVP + all post-MVP features in a single plan
- Go + Bubbletea + Lipgloss + Cobra confirmed as stack
- Shell out to git CLI (not go-git) — lazygit pattern for reliability
- Tests after implementation, not TDD
- claude-squad deeply studied as reference (adapted for OpenCode, not Claude Code)

**Research Findings**:
- **claude-squad** (smtg-ai/claude-squad): AGPL-3.0, near-identical concept. Key patterns: Instance struct with Status enum, SHA256 hash-based activity detection, tmux+PTY for attach/detach, JSON state persistence, Bubbletea state machine TUI
- **lazygit**: WorktreeCommands pattern, `git worktree list --porcelain` parsing, worktree health checks
- **sesh** (1.6k stars): Clean tmux connector pattern — abstraction layer behind interface
- **overmind** (3.4k stars): tmux as process supervisor, `#{pane_dead}` for crash detection
- **gh-dash**: Production Cobra+Bubbletea, multi-view state machine, shared ProgramContext
- **Glow**: Official tea.Suspend/tea.ResumeMsg (but Metis warns against using tea.Suspend directly)
- **gofrs/flock**: File locking for concurrent state access, kernel auto-releases on crash

### Metis Review
**Identified Gaps** (addressed):
- **tea.Suspend has silent failure mode**: ReleaseTerminal() fails silently, suspendProcess() blocks indefinitely. RESOLVED: Use explicit ReleaseTerminal/RestoreTerminal with error handling instead of tea.Suspend
- **Pane PID ≠ command PID**: tmux pane PID is the shell, not opencode. RESOLVED: Store opencode PID explicitly, use `remain-on-exit on` for pane death detection
- **tmux crash orphans processes**: All sessions lost, processes reparented to PID 1. RESOLVED: Store opencode PIDs in state.json, attempt recovery on startup by checking worktree dirs and running processes
- **git merge-tree needs explicit base**: Multiple merge bases produce nondeterministic results. RESOLVED: Always use `--merge-base=$(git merge-base main <branch>)` explicitly
- **git worktree repair**: Run `git worktree repair` + `git worktree prune` during startup reconciliation
- **No file locking in claude-squad**: They use atomic writes only. RESOLVED: Use gofrs/flock for proper read/write locking in OCW

---

## Work Objectives

### Core Objective
Build a complete, production-quality Go TUI application that automates the workflow of managing multiple parallel OpenCode instances across isolated git worktrees, orchestrated via tmux, with a keyboard-driven dashboard for monitoring, switching, and merging.

### Concrete Deliverables
- `ocw` binary installable via `go install github.com/tommyzliu/ocw@latest`
- CLI subcommands: `init`, `new`, `list`, `focus`, `term`, `edit`, `diff`, `merge`, `delete`, `status`, `kill`, `config`
- TUI views: Dashboard, Instance/Focus, Create, Diff, Merge, Help
- Workspace manager: worktree + tmux + opencode process lifecycle
- Sub-terminal pane management with default splits
- State persistence and config management
- Dependency checking (tmux, gh/glab) with clear error messages

### Definition of Done
- [ ] `ocw init` creates .ocw/ directory with config.toml and state.json
- [ ] `ocw new feat/test` creates worktree, tmux window, launches opencode
- [ ] Dashboard lists all instances with status, elapsed time, sub-terminal count
- [ ] Focus view attaches to instance's tmux window
- [ ] Sub-terminals can be created, listed, and auto-cleaned on instance delete
- [ ] Diff view shows git diff --stat against base branch
- [ ] Merge flow pushes branch and creates PR via gh/glab
- [ ] `ocw delete` cleanly removes worktree, tmux window, all panes, and state entry
- [ ] `ocw` re-attaches to existing session on relaunch
- [ ] All edge cases handled with clear error messages

### Must Have
- tmux and git as runtime dependencies with clear error messages if missing
- Graceful handling of tmux/opencode crashes with recovery on restart
- Conflict detection across instances (overlapping file modifications)
- Clean teardown of all resources (worktrees, tmux windows, panes)
- Atomic state writes with file locking

### Must NOT Have (Guardrails)
- NO go-git library — shell out to git CLI only
- NO tea.Suspend for focus view — use explicit ReleaseTerminal/RestoreTerminal
- NO hardcoded tmux session names — always use configurable prefix
- NO port management or .env.local writing (removed from spec v2)
- NO daemon mode in MVP (claude-squad has it, but it's not in OCW spec)
- NO AI-generated comments or documentation beyond what's needed for clarity
- NO over-abstraction — keep interfaces minimal (tmux, git, state)
- NO tests that mock the entire TUI — test underlying logic only

---

## Verification Strategy

> **UNIVERSAL RULE: ZERO HUMAN INTERVENTION**
>
> ALL tasks in this plan MUST be verifiable WITHOUT any human action.

### Test Decision
- **Infrastructure exists**: NO (greenfield)
- **Automated tests**: YES (after implementation — separate test tasks)
- **Framework**: Go standard `testing` package + `testify/assert`

### Agent-Executed QA Scenarios (MANDATORY — ALL tasks)

**Verification Tool by Deliverable Type:**

| Type | Tool | How Agent Verifies |
|------|------|-------------------|
| **Go compilation** | Bash | `go build ./...` exits 0 |
| **CLI commands** | Bash | Run command, check exit code, parse stdout |
| **TUI interaction** | interactive_bash (tmux) | Launch in tmux, send keys, capture pane |
| **tmux operations** | Bash | `tmux list-sessions`, `tmux list-windows`, `tmux list-panes` |
| **Git operations** | Bash | `git worktree list`, `git branch`, `git diff --stat` |
| **State files** | Bash | `cat .ocw/state.json | jq .`, validate structure |
| **Config files** | Bash | `cat .ocw/config.toml`, validate TOML syntax |

---

## Execution Strategy

### Parallel Execution Waves

```
Wave 1 (Foundation — Start Immediately):
├── Task 1: Project scaffold (go mod, directories, .gitignore)
└── Task 3: Git abstraction layer

Wave 2 (Core Infrastructure — After Wave 1):
├── Task 2: State + Config management (depends: 1)
├── Task 4: tmux abstraction layer (depends: 1)
└── Task 3 continues if not done

Wave 3 (Workspace Engine — After Wave 2):
├── Task 5: Worktree manager (depends: 2, 3, 4)
├── Task 7: IDE launcher (depends: 1)
└── Task 6a: TUI scaffold + Dashboard skeleton (depends: 2)

Wave 4 (TUI Views — After Wave 3):
├── Task 6: Dashboard view complete (depends: 5, 6a)
├── Task 8: Instance creation flow (depends: 5, 6a)
├── Task 9: Sub-terminal management (depends: 4, 5)
├── Task 10: Diff view (depends: 3, 6a)
└── Task 11: Conflict detection (depends: 3, 5)

Wave 5 (Integration — After Wave 4):
├── Task 12: Focus view with ReleaseTerminal (depends: 4, 6)
├── Task 13: Merge view + PR creation (depends: 3, 6, 10)
├── Task 14: Instance deletion flow (depends: 5, 6, 9)
├── Task 15: CLI subcommands (depends: 5, 6, 8, 9)
└── Task 16: Startup reconciliation + crash recovery (depends: 2, 4, 5)

Wave 6 (Polish + Post-MVP):
├── Task 17: Edge cases + error handling polish
├── Task 18: Tests for core logic
├── Tasks 19-30: Post-MVP features
└── Task 31: README + documentation
```

### Dependency Matrix

| Task | Depends On | Blocks | Can Parallelize With |
|------|------------|--------|---------------------|
| 1 | None | 2, 4, 6a, 7 | 3 |
| 2 | 1 | 5, 6a, 16 | 3, 4 |
| 3 | 1 | 5, 10, 11, 13 | 2, 4 |
| 4 | 1 | 5, 9, 12, 16 | 2, 3 |
| 5 | 2, 3, 4 | 6, 8, 9, 11, 14, 16 | 7, 6a |
| 6a | 2 | 6, 8, 10, 12, 13, 14 | 5, 7 |
| 6 | 5, 6a | 12, 13, 14, 15 | 8, 9, 10, 11 |
| 7 | 1 | 15 | 5, 6a |
| 8 | 5, 6a | 15 | 6, 9, 10, 11 |
| 9 | 4, 5 | 14, 15 | 6, 8, 10, 11 |
| 10 | 3, 6a | 13 | 6, 8, 9, 11 |
| 11 | 3, 5 | 6, 17 | 8, 9, 10 |
| 12 | 4, 6 | 15 | 13, 14 |
| 13 | 3, 6, 10 | 15 | 12, 14 |
| 14 | 5, 6, 9 | 15 | 12, 13 |
| 15 | 5, 6, 7, 8, 9, 12, 13, 14 | 17 | 16 |
| 16 | 2, 4, 5 | 17 | 15 |
| 17 | 15, 16 | 18 | None |
| 18 | 17 | None | 19-30 |

### Agent Dispatch Summary

| Wave | Tasks | Recommended Dispatch |
|------|-------|---------------------|
| 1 | 1, 3 | Two parallel agents: quick for scaffold, unspecified-high for git layer |
| 2 | 2, 4 | Two parallel agents: unspecified-high each |
| 3 | 5, 6a, 7 | Three parallel agents |
| 4 | 6, 8, 9, 10, 11 | Five parallel agents (all independent after wave 3) |
| 5 | 12, 13, 14, 15, 16 | Five parallel agents |
| 6 | 17, 18, 19-30, 31 | Sequential within MVP polish, parallel for post-MVP |

---

## TODOs

### WAVE 1: Foundation

- [x] 1. Project Scaffold & Dependencies

  **What to do**:
  - Initialize Go module: `go mod init github.com/tommyzliu/ocw`
  - Create directory structure:
    ```
    ocw/
    ├── cmd/
    │   └── root.go          # Cobra root command
    ├── internal/
    │   ├── config/          # TOML config loading
    │   ├── state/           # JSON state persistence
    │   ├── git/             # Git CLI abstraction
    │   ├── tmux/            # tmux CLI abstraction
    │   ├── workspace/       # Worktree + process lifecycle
    │   ├── ide/             # IDE launcher
    │   └── tui/
    │       ├── app.go       # Root Bubbletea model
    │       ├── context.go   # Shared context
    │       ├── keys.go      # Key bindings
    │       ├── styles.go    # Lipgloss styles
    │       └── views/       # View sub-models
    │           ├── dashboard.go
    │           ├── instance.go
    │           ├── create.go
    │           ├── diff.go
    │           ├── merge.go
    │           └── help.go
    ├── main.go              # Entry point
    ├── .gitignore
    └── go.mod
    ```
  - Install dependencies:
    ```
    go get github.com/charmbracelet/bubbletea
    go get github.com/charmbracelet/lipgloss
    go get github.com/charmbracelet/bubbles
    go get github.com/charmbracelet/huh
    go get github.com/spf13/cobra
    go get github.com/BurntSushi/toml
    go get github.com/gofrs/flock
    ```
  - Create `.gitignore` (ocw binary, .ocw/, .worktrees/, *.test)
  - Create minimal `main.go` that calls `cmd.Execute()`
  - Create `cmd/root.go` with Cobra root command that prints "OCW v0.1.0-dev"
  - Verify: `go build -o ocw . && ./ocw` prints version

  **Must NOT do**:
  - Do not implement any real functionality yet — scaffold only
  - Do not add test files yet
  - Do not create README.md

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []
    - Pure Go project setup, no special domain knowledge needed

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Task 3)
  - **Blocks**: Tasks 2, 4, 6a, 7
  - **Blocked By**: None

  **References**:

  **Pattern References**:
  - gh-dash directory structure: `cmd/root.go` + `internal/` layout. Follow this convention.
  - Glow entry point: Cobra rootCmd.RunE launches Bubbletea. Pattern: `main.go` → `cmd.Execute()` → `cmd/root.go`

  **External References**:
  - Cobra getting started: https://github.com/spf13/cobra-cli/blob/main/README.md
  - Bubbletea basics: https://github.com/charmbracelet/bubbletea#tutorial

  **Acceptance Criteria**:
  - [ ] `go build -o ocw .` exits 0 with no errors
  - [ ] `./ocw` prints version string containing "ocw"
  - [ ] All directories in the structure above exist
  - [ ] `go mod tidy` exits 0 (all dependencies resolve)
  - [ ] `.gitignore` contains: ocw, .ocw/, .worktrees/

  **Agent-Executed QA Scenarios**:
  ```
  Scenario: Go project builds successfully
    Tool: Bash
    Preconditions: Go 1.25.6 installed
    Steps:
      1. Run: go build -o ocw .
      2. Assert: exit code 0
      3. Run: ./ocw
      4. Assert: stdout contains "ocw"
      5. Run: go mod tidy
      6. Assert: exit code 0
    Expected Result: Binary builds and runs
    Evidence: Build output captured

  Scenario: Directory structure is correct
    Tool: Bash
    Preconditions: Project scaffold created
    Steps:
      1. Run: find . -type d | sort | grep -v .git | grep -v .sisyphus
      2. Assert: output contains cmd/, internal/config/, internal/state/, internal/git/,
                 internal/tmux/, internal/workspace/, internal/ide/, internal/tui/,
                 internal/tui/views/
      3. Run: cat .gitignore
      4. Assert: contains "ocw" and ".ocw/" and ".worktrees/"
    Expected Result: All directories and files present
    Evidence: find output captured
  ```

  **Commit**: YES
  - Message: `feat: initialize project scaffold with Go module and directory structure`
  - Files: `go.mod, go.sum, main.go, cmd/root.go, .gitignore, all internal/ dirs`
  - Pre-commit: `go build -o ocw .`

---

- [x] 3. Git Abstraction Layer

  **What to do**:
  - Create `internal/git/git.go` — core Git command runner:
    - `type Git struct { repoPath string }` with `NewGit(repoPath string) *Git`
    - `func (g *Git) run(args ...string) (string, error)` — executes `git -C <repoPath> <args>`, captures stdout/stderr
    - `func (g *Git) IsGitRepo() bool` — checks if path is a git repo
    - `func (g *Git) GetDefaultBranch() (string, error)` — detects main/master
    - `func (g *Git) GetCurrentBranch() (string, error)` — `git rev-parse --abbrev-ref HEAD`
    - `func (g *Git) GetHeadSHA() (string, error)` — `git rev-parse HEAD`
    - `func (g *Git) BranchExists(branch string) bool` — `git show-ref --verify refs/heads/<branch>`
    - `func (g *Git) GetRemotes() ([]string, error)` — `git remote`
  - Create `internal/git/worktree.go` — worktree operations:
    - `func (g *Git) WorktreeAdd(path, branch, base string) error` — `git worktree add <path> -b <branch> <base>`
    - `func (g *Git) WorktreeAddExisting(path, branch string) error` — `git worktree add <path> <branch>`
    - `func (g *Git) WorktreeRemove(path string, force bool) error` — `git worktree remove [-f] <path>`
    - `func (g *Git) WorktreeList() ([]WorktreeInfo, error)` — parse `git worktree list --porcelain`
    - `func (g *Git) WorktreeRepair() error` — `git worktree repair`
    - `func (g *Git) WorktreePrune() error` — `git worktree prune`
    - `type WorktreeInfo struct { Path, Branch, Head string; Bare, Detached bool }`
  - Create `internal/git/diff.go` — diff operations:
    - `func (g *Git) DiffStat(base string) (DiffStat, error)` — `git diff --stat <base>` with worktree path as CWD
    - `func (g *Git) DiffStatBranch(branch, base string) (DiffStat, error)` — `git diff --stat <base>..<branch>`
    - `func (g *Git) DiffFiles(base string) ([]DiffFile, error)` — parse `git diff --name-status <base>`
    - `type DiffStat struct { FilesChanged, Insertions, Deletions int; Summary string }`
    - `type DiffFile struct { Status string; Path string }` (M/A/D/R)
  - Create `internal/git/merge.go` — merge/conflict operations:
    - `func (g *Git) MergeBase(branch1, branch2 string) (string, error)` — `git merge-base <b1> <b2>`
    - `func (g *Git) MergeTree(base, branch1, branch2 string) (MergeResult, error)` — `git merge-tree --write-tree --merge-base=<base> <b1> <b2>`
    - `func (g *Git) HasConflicts(branch, baseBranch string) (bool, []string, error)` — combines merge-base + merge-tree
    - `type MergeResult struct { Clean bool; ConflictFiles []string }`
  - Create `internal/git/push.go` — remote operations:
    - `func (g *Git) Push(remote, branch string) error` — `git push <remote> <branch>`
    - `func (g *Git) DeleteRemoteBranch(remote, branch string) error`
    - `func (g *Git) DeleteLocalBranch(branch string, force bool) error`

  **Must NOT do**:
  - Do not use go-git library — shell out only
  - Do not implement PR creation here (that's merge manager, Task 13)
  - Do not add worktree health checking (that's Task 16)

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []
    - System-level Go code with os/exec, output parsing. No special framework knowledge needed.

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Task 1)
  - **Blocks**: Tasks 5, 10, 11, 13
  - **Blocked By**: Task 1 (needs go.mod)

  **References**:

  **Pattern References**:
  - lazygit `WorktreeCommands`: https://github.com/jesseduffield/lazygit/blob/master/pkg/commands/git_commands/worktree.go — Clean worktree abstraction, `NewWorktreeOpts` pattern
  - lazygit `worktree_loader.go`: https://github.com/jesseduffield/lazygit/blob/master/pkg/commands/git_commands/worktree_loader.go — Parsing `git worktree list --porcelain`
  - claude-squad `git/worktree.go`: https://github.com/smtg-ai/claude-squad/blob/main/session/git/worktree.go — GitWorktree struct with repoPath, worktreePath, branchName, baseCommitSHA
  - claude-squad `git/diff.go`: https://github.com/smtg-ai/claude-squad/blob/main/session/git/diff.go — DiffStats computation with line counting

  **Acceptance Criteria**:
  - [ ] All files compile: `go build ./internal/git/...` exits 0
  - [ ] Git struct can detect a git repo: `IsGitRepo()` returns true for current dir
  - [ ] WorktreeList() parses `git worktree list --porcelain` output correctly
  - [ ] DiffStat parses `git diff --stat` output correctly
  - [ ] MergeTree correctly passes `--merge-base` flag

  **Agent-Executed QA Scenarios**:
  ```
  Scenario: Git abstraction compiles and basic operations work
    Tool: Bash
    Preconditions: Go project scaffold exists, inside a git repo
    Steps:
      1. Run: go build ./internal/git/...
      2. Assert: exit code 0
      3. Create a small Go test program in /tmp that imports internal/git and calls IsGitRepo()
      4. Run the test program pointing at the ocw repo
      5. Assert: returns true
    Expected Result: Git package compiles and works
    Evidence: Build output and test output captured

  Scenario: WorktreeList parses porcelain output
    Tool: Bash
    Preconditions: Git repo exists
    Steps:
      1. Run: git worktree list --porcelain (to verify format)
      2. Assert: output contains "worktree" line
    Expected Result: Porcelain format is parseable
    Evidence: Output captured
  ```

  **Commit**: YES
  - Message: `feat(git): add git CLI abstraction layer for worktrees, diffs, and merge operations`
  - Files: `internal/git/*.go`
  - Pre-commit: `go build ./...`

---

### WAVE 2: Core Infrastructure

- [x] 2. State & Config Management

  **What to do**:
  - Create `internal/config/config.go`:
    - Define `Config` struct matching `.ocw/config.toml` schema from spec:
      ```go
      type Config struct {
          Workspace WorkspaceConfig
          OpenCode  OpenCodeConfig
          Editor    EditorConfig
          Merge     MergeConfig
          Tmux      TmuxConfig
          UI        UIConfig
      }
      ```
    - `func LoadConfig(dir string) (*Config, error)` — reads .ocw/config.toml, fills defaults
    - `func DefaultConfig() *Config` — sensible defaults per spec
    - `func SaveConfig(dir string, cfg *Config) error` — writes config.toml
    - Use `BurntSushi/toml` for parsing
  - Create `internal/state/state.go`:
    - Define `State` struct matching `.ocw/state.json` schema:
      ```go
      type State struct {
          Repo         string     `json:"repo"`
          TmuxSession  string     `json:"tmux_session"`
          Instances    []Instance `json:"instances"`
      }
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
      }
      type SubTerminal struct {
          PaneID    string    `json:"pane_id"`
          Label     string    `json:"label"`
          CreatedAt time.Time `json:"created_at"`
      }
      ```
    - `func NewStore(dir string) *Store` — creates store with .ocw/ path
    - `func (s *Store) Load() (*State, error)` — reads state.json with flock RLock
    - `func (s *Store) Save(state *State) error` — writes state.json with flock Lock + atomic write (temp file + rename)
    - `func (s *Store) AddInstance(inst Instance) error` — load, append, save
    - `func (s *Store) RemoveInstance(id string) error` — load, filter, save
    - `func (s *Store) UpdateInstance(id string, fn func(*Instance)) error` — load, find, modify, save
    - Use `gofrs/flock` for file locking
    - Generate unique IDs with `crypto/rand` (6-char hex)
  - Create `internal/config/init.go`:
    - `func InitWorkspace(repoRoot string) error` — creates .ocw/ dir, writes default config.toml, creates empty state.json

  **Must NOT do**:
  - No global state store (per-repo only, in .ocw/)
  - No sync.Mutex for in-process locking (file lock is sufficient for single-process OCW)
  - Do not add migration logic for state format changes

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []
    - File I/O, JSON/TOML parsing, file locking. Standard Go patterns.

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2 (with Tasks 3, 4)
  - **Blocks**: Tasks 5, 6a, 16
  - **Blocked By**: Task 1

  **References**:

  **Pattern References**:
  - claude-squad `config/state.go`: https://github.com/smtg-ai/claude-squad/blob/main/config/state.go — State struct with JSON persistence, LoadState/SaveState pattern
  - claude-squad `session/storage.go`: https://github.com/smtg-ai/claude-squad/blob/main/session/storage.go — InstanceData serialization, DiffStatsData, GitWorktreeData
  - gh-dash config: Uses koanf + YAML. We use BurntSushi/toml instead, but same pattern (load file, fill defaults, return struct)

  **External References**:
  - gofrs/flock usage: https://pkg.go.dev/github.com/gofrs/flock — RLock/Lock/Unlock, .lock file pattern
  - BurntSushi/toml: https://pkg.go.dev/github.com/BurntSushi/toml — Decode/Encode

  **Acceptance Criteria**:
  - [ ] `go build ./internal/config/... ./internal/state/...` exits 0
  - [ ] `InitWorkspace()` creates .ocw/ with config.toml and state.json
  - [ ] LoadConfig reads valid TOML and returns correct defaults
  - [ ] Store.Save writes valid JSON readable by `jq`
  - [ ] Store.Load reads the JSON back correctly
  - [ ] File locking works (no corruption on concurrent access)

  **Agent-Executed QA Scenarios**:
  ```
  Scenario: Init creates valid config and state files
    Tool: Bash
    Preconditions: Project builds, tmp directory available
    Steps:
      1. Create temp git repo: mkdir /tmp/test-ocw && cd /tmp/test-ocw && git init
      2. Write small Go program that calls InitWorkspace("/tmp/test-ocw")
      3. Run: cat /tmp/test-ocw/.ocw/config.toml
      4. Assert: contains [workspace], [opencode], [editor], [merge], [tmux], [ui] sections
      5. Run: cat /tmp/test-ocw/.ocw/state.json | jq .
      6. Assert: valid JSON with "repo", "tmux_session", "instances" keys
    Expected Result: Valid config and state files created
    Evidence: File contents captured

  Scenario: State persistence roundtrip
    Tool: Bash
    Steps:
      1. Write Go program that: creates Store, adds Instance, saves, loads, verifies fields match
      2. Run program
      3. Assert: exit code 0, all fields roundtrip correctly
    Expected Result: JSON serialization/deserialization works
    Evidence: Program output captured
  ```

  **Commit**: YES
  - Message: `feat(config): add state persistence and TOML config management with file locking`
  - Files: `internal/config/*.go, internal/state/*.go`
  - Pre-commit: `go build ./...`

---

- [x] 4. tmux Abstraction Layer

  **What to do**:
  - Create `internal/tmux/tmux.go`:
    - `type Tmux struct { }` — stateless command runner
    - `func NewTmux() *Tmux`
    - `func (t *Tmux) IsInstalled() bool` — `which tmux`
    - `func (t *Tmux) Version() (string, error)` — `tmux -V`
    - `func (t *Tmux) run(args ...string) (string, error)` — executes tmux commands, captures output
    - `func (t *Tmux) runAttached(args ...string) error` — executes tmux with stdin/stdout/stderr attached
  - Create `internal/tmux/session.go`:
    - `func (t *Tmux) NewSession(name, dir string) error` — `tmux new-session -d -s <name> -c <dir>`
    - `func (t *Tmux) HasSession(name string) bool` — `tmux has-session -t <name>`
    - `func (t *Tmux) KillSession(name string) error` — `tmux kill-session -t <name>`
    - `func (t *Tmux) ListSessions() ([]string, error)` — `tmux list-sessions -F "#{session_name}"`
    - `func (t *Tmux) AttachSession(name string) error` — `tmux attach-session -t <name>` (attached to terminal)
  - Create `internal/tmux/window.go`:
    - `func (t *Tmux) NewWindow(session, name, dir string) (string, error)` — `tmux new-window -t <session> -n <name> -c <dir>` returns window ID
    - `func (t *Tmux) KillWindow(target string) error` — `tmux kill-window -t <target>`
    - `func (t *Tmux) ListWindows(session string) ([]WindowInfo, error)` — parse `tmux list-windows -t <session> -F "#{window_id}:#{window_name}:#{window_active}"`
    - `func (t *Tmux) SelectWindow(target string) error` — `tmux select-window -t <target>`
    - `func (t *Tmux) SendKeys(target, keys string) error` — `tmux send-keys -t <target> <keys> Enter`
    - `func (t *Tmux) RunInWindow(session, name, dir, command string) (string, error)` — creates window and runs command
    - `type WindowInfo struct { ID, Name string; Active bool }`
  - Create `internal/tmux/pane.go`:
    - `func (t *Tmux) SplitWindow(target, dir, split string, percentage int) (string, error)` — `tmux split-window -t <target> [-h|-v] -p <pct> -c <dir>` returns pane ID
    - `func (t *Tmux) KillPane(target string) error` — `tmux kill-pane -t <target>`
    - `func (t *Tmux) ListPanes(window string) ([]PaneInfo, error)` — parse `tmux list-panes -t <window> -F "#{pane_id}:#{pane_pid}:#{pane_dead}:#{pane_current_command}"`
    - `func (t *Tmux) CapturePaneContent(target string) (string, error)` — `tmux capture-pane -p -e -J -t <target>`
    - `func (t *Tmux) SetOption(target, option, value string) error` — `tmux set-option -t <target> <option> <value>`
    - `func (t *Tmux) SetRemainOnExit(target string, on bool) error` — sets `remain-on-exit` for pane death detection
    - `type PaneInfo struct { ID string; PID int; Dead bool; Command string }`

  **Must NOT do**:
  - No PTY management (unlike claude-squad — we use ReleaseTerminal/tmux attach pattern instead)
  - No status monitoring here (that's workspace manager, Task 5)
  - No session naming logic here (workspace manager decides names)

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []
    - os/exec subprocess management, tmux CLI familiarity. Reference multiclaude and sesh patterns.

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2 (with Tasks 2, 3)
  - **Blocks**: Tasks 5, 9, 12, 16
  - **Blocked By**: Task 1

  **References**:

  **Pattern References**:
  - multiclaude `pkg/tmux/client.go`: https://github.com/dlorenc/multiclaude/blob/main/pkg/tmux/client.go — Clean tmux client with NewClient(), IsTmuxAvailable(), session management
  - sesh `connector/tmux.go`: https://github.com/joshmedeski/sesh/blob/main/connector/tmux.go — tmux connector pattern with clean abstraction
  - claude-squad `session/tmux/tmux.go`: https://github.com/smtg-ai/claude-squad/blob/main/session/tmux/tmux.go — TmuxSession with session naming, Start/Close/Attach/Detach
  - overmind `start/tmux.go`: https://github.com/DarthSim/overmind/blob/master/start/tmux.go — tmux as process supervisor, pane death detection

  **Acceptance Criteria**:
  - [ ] `go build ./internal/tmux/...` exits 0
  - [ ] Tmux.IsInstalled() returns correct value based on system
  - [ ] All functions construct correct tmux command strings
  - [ ] PaneInfo correctly parses tmux format strings
  - [ ] SplitWindow correctly handles -h (horizontal) and -v (vertical) flags

  **Agent-Executed QA Scenarios**:
  ```
  Scenario: tmux package compiles
    Tool: Bash
    Steps:
      1. Run: go build ./internal/tmux/...
      2. Assert: exit code 0
    Expected Result: Package compiles
    Evidence: Build output

  Scenario: tmux command construction is correct
    Tool: Bash
    Steps:
      1. Run: go vet ./internal/tmux/...
      2. Assert: exit code 0 (no issues)
    Expected Result: Code is valid Go
    Evidence: Vet output
  ```

  **Commit**: YES
  - Message: `feat(tmux): add tmux CLI abstraction for sessions, windows, and panes`
  - Files: `internal/tmux/*.go`
  - Pre-commit: `go build ./...`

---

### WAVE 3: Workspace Engine

- [x] 5. Workspace Manager (Instance Lifecycle)

  **What to do**:
  - Create `internal/workspace/manager.go`:
    - `type Manager struct { git *git.Git; tmux *tmux.Tmux; store *state.Store; config *config.Config; repoRoot string }`
    - `func NewManager(repoRoot string, cfg *config.Config) (*Manager, error)`
    - This is the core orchestration layer that coordinates git, tmux, and state
  - Create `internal/workspace/instance.go`:
    - `type CreateOpts struct { Name, Branch, BaseBranch string }`
    - `func (m *Manager) CreateInstance(opts CreateOpts) (*state.Instance, error)`:
      1. Validate branch name (no spaces, valid git ref)
      2. Check if branch already exists → error with suggestion
      3. Determine worktree path: `<worktree_dir>/<sanitized-branch>`
      4. `git worktree add .worktrees/<branch> -b <branch> <base>`
      5. Create tmux window in OCW session: `tmux new-window -t <session> -n <name> -c <worktree_path>`
      6. Set `remain-on-exit on` for the primary pane
      7. Launch opencode in the window: `tmux send-keys -t <window> "opencode" Enter`
      8. Capture the opencode PID (via `tmux list-panes -F "#{pane_pid}"` → get child PID)
      9. Register instance in state store
    - `func (m *Manager) DeleteInstance(id string, force bool) error`:
      1. Load instance from state
      2. Kill all sub-terminal panes
      3. Kill the tmux window (kills opencode too)
      4. `git worktree remove <path>` (with force if dirty + confirmed)
      5. Optionally delete branch
      6. Remove from state
    - `func (m *Manager) PauseInstance(id string) error` — SIGSTOP to opencode PID
    - `func (m *Manager) ResumeInstance(id string) error` — SIGCONT to opencode PID
    - `func (m *Manager) GetInstanceStatus(id string) (string, error)` — check if PID alive, pane dead, etc.
    - `func (m *Manager) ListInstances() ([]state.Instance, error)`
    - `func (m *Manager) GetInstance(id string) (*state.Instance, error)`
  - Create `internal/workspace/session.go`:
    - `func (m *Manager) EnsureSession() error` — creates OCW tmux session if it doesn't exist
    - `func (m *Manager) SessionName() string` — `<prefix>-<reponame>`
    - `func (m *Manager) SessionExists() bool`
    - `func (m *Manager) KillSession() error` — kills everything

  **Must NOT do**:
  - No TUI code here — this is pure business logic
  - No sub-terminal management here (Task 9)
  - No crash recovery logic here (Task 16)
  - No conflict detection here (Task 11)

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []
    - Core orchestration logic. Requires understanding of git, tmux, and state packages.

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 3 (with Tasks 6a, 7)
  - **Blocks**: Tasks 6, 8, 9, 11, 14, 16
  - **Blocked By**: Tasks 2, 3, 4

  **References**:

  **Pattern References**:
  - claude-squad `session/instance.go`: https://github.com/smtg-ai/claude-squad/blob/main/session/instance.go — Instance struct with Start/Kill/Pause/Resume, status transitions, activity detection via SHA256
  - claude-squad lifecycle: `Ready → Start → Running ↔ Ready → Pause → Paused → Resume → Running → Kill`

  **Acceptance Criteria**:
  - [ ] `go build ./internal/workspace/...` exits 0
  - [ ] Manager correctly sanitizes branch names for worktree paths
  - [ ] CreateInstance creates worktree + tmux window + state entry
  - [ ] DeleteInstance cleans up all resources
  - [ ] SessionName follows `<prefix>-<reponame>` pattern

  **Agent-Executed QA Scenarios**:
  ```
  Scenario: Workspace manager compiles and types are correct
    Tool: Bash
    Steps:
      1. Run: go build ./internal/workspace/...
      2. Assert: exit code 0
      3. Run: go vet ./internal/workspace/...
      4. Assert: exit code 0
    Expected Result: Package compiles with no issues
    Evidence: Build output
  ```

  **Commit**: YES
  - Message: `feat(workspace): add instance lifecycle manager for worktree + tmux orchestration`
  - Files: `internal/workspace/*.go`
  - Pre-commit: `go build ./...`

---

- [x] 6a. TUI Scaffold & Dashboard Skeleton

  **What to do**:
  - Create `internal/tui/app.go` — root Bubbletea model:
    - State machine with view enum: `viewDashboard`, `viewInstance`, `viewCreate`, `viewDiff`, `viewMerge`, `viewHelp`
    - Embed sub-models for each view
    - Route Update/View calls based on current view state
    - Handle window resize events
    - Implement `tea.WindowSizeMsg` handler to propagate sizes to sub-views
  - Create `internal/tui/context.go`:
    - `type Context struct { Config *config.Config; Manager *workspace.Manager; Width, Height int }`
    - Shared context passed to all sub-views
  - Create `internal/tui/keys.go`:
    - Define key bindings per the spec's hotkey table
    - Use `bubbles/key` for key binding definitions
  - Create `internal/tui/styles.go`:
    - Lipgloss styles for: header, instance list items (active/idle/paused/error/merged/done), status indicators, borders, help bar
  - Create `internal/tui/views/dashboard.go` — skeleton:
    - Instance list using `bubbles/list` (custom item delegate for status icons, elapsed time, sub-terminal count)
    - Header bar showing "OCW Dashboard" + instance count
    - Footer bar showing hotkey hints
    - j/k navigation, 1-9 jump
    - Placeholder actions for hotkeys (n, d, e, f, m, t, q, Q)
  - Wire up in `cmd/root.go`:
    - Root command's RunE creates Manager, loads state, creates TUI Context, launches `tea.NewProgram()`
    - Fullscreen alternate screen mode

  **Must NOT do**:
  - No real hotkey implementations yet (those come in later tasks)
  - No instance creation/deletion flows
  - No real data — use mock instances for visual testing
  - No focus view yet

  **Recommended Agent Profile**:
  - **Category**: `visual-engineering`
  - **Skills**: [`frontend-ui-ux`]
    - TUI layout, styling, Bubbletea model architecture. Visual design matters.

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 3 (with Tasks 5, 7)
  - **Blocks**: Tasks 6, 8, 10, 12, 13, 14
  - **Blocked By**: Task 2 (needs config/state types)

  **References**:

  **Pattern References**:
  - gh-dash `internal/tui/ui.go`: Root model with state machine, multi-view routing, shared ProgramContext
  - claude-squad `app/app.go`: home struct with state enum (stateDefault/stateNew/statePrompt/stateHelp/stateConfirm), Update loop, handleKeyPress
  - claude-squad `ui/list.go`: Instance list rendering with status icons and diff stats display
  - Bubbletea composable-views example: https://github.com/charmbracelet/bubbletea/blob/main/examples/composable-views/main.go

  **External References**:
  - Bubbles list component: https://github.com/charmbracelet/bubbles/tree/master/list
  - Lipgloss layout: https://github.com/charmbracelet/lipgloss#layout

  **Acceptance Criteria**:
  - [ ] `go build -o ocw .` exits 0
  - [ ] `./ocw` (in a git repo with .ocw/) launches TUI with dashboard
  - [ ] Dashboard shows header, instance list area, footer with hotkeys
  - [ ] j/k keys navigate the instance list
  - [ ] q quits the TUI cleanly
  - [ ] Window resize is handled without panic

  **Agent-Executed QA Scenarios**:
  ```
  Scenario: TUI launches and displays dashboard
    Tool: interactive_bash (tmux)
    Preconditions: Binary built, .ocw/ initialized in test repo
    Steps:
      1. tmux new-session -d -s test-ocw
      2. tmux send-keys -t test-ocw "./ocw" Enter
      3. Sleep 2s
      4. tmux capture-pane -t test-ocw -p
      5. Assert: output contains "OCW" or "Dashboard"
      6. Assert: output contains hotkey hints ("n", "d", "q")
      7. tmux send-keys -t test-ocw "q"
      8. Sleep 1s
      9. Assert: process exited (pane shows shell prompt)
    Expected Result: Dashboard renders and q quits
    Evidence: Pane capture saved
  ```

  **Commit**: YES
  - Message: `feat(tui): add Bubbletea scaffold with dashboard view skeleton and key bindings`
  - Files: `internal/tui/*.go, internal/tui/views/dashboard.go, cmd/root.go`
  - Pre-commit: `go build -o ocw .`

---

- [x] 7. IDE Launcher

  **What to do**:
  - Create `internal/ide/launcher.go`:
    - `type Launcher struct { config config.EditorConfig; tmux *tmux.Tmux }`
    - `func NewLauncher(cfg config.EditorConfig, tmux *tmux.Tmux) *Launcher`
    - `func (l *Launcher) DetectEditor() string` — auto-detect order: config → $EDITOR → cursor → code → zed → nvim → vim → vi
    - `func (l *Launcher) IsTerminalEditor(editor string) bool` — checks against `terminal_editors` list
    - `func (l *Launcher) Open(worktreePath, tmuxTarget string) error`:
      - If GUI editor: `exec.Command(editor, worktreePath).Start()` (detached)
      - If terminal editor: `tmux split-window` in instance's window, run editor in new pane
    - `func (l *Launcher) DetectHeadless() bool` — check $DISPLAY, $WAYLAND_DISPLAY, SSH_TTY

  **Must NOT do**:
  - No TUI integration (that's wired in Task 15)
  - No complex editor configuration

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []
    - Simple utility code, editor detection. Small scope.

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 3 (with Tasks 5, 6a)
  - **Blocks**: Task 15
  - **Blocked By**: Task 1

  **References**:

  **Pattern References**:
  - Spec section "Open in IDE" — auto-detection order, GUI vs terminal editor handling

  **Acceptance Criteria**:
  - [ ] `go build ./internal/ide/...` exits 0
  - [ ] DetectEditor() returns a valid editor command on the system
  - [ ] IsTerminalEditor correctly identifies nvim, vim, nano, emacs

  **Agent-Executed QA Scenarios**:
  ```
  Scenario: IDE package compiles
    Tool: Bash
    Steps:
      1. Run: go build ./internal/ide/...
      2. Assert: exit code 0
    Expected Result: Compiles
    Evidence: Output captured
  ```

  **Commit**: YES (groups with Task 5 or 6a if convenient)
  - Message: `feat(ide): add editor auto-detection and launcher for GUI and terminal editors`
  - Files: `internal/ide/*.go`
  - Pre-commit: `go build ./...`

---

### WAVE 4: TUI Views & Features

- [x] 6. Dashboard View (Complete)

  **What to do**:
  - Enhance `internal/tui/views/dashboard.go` from skeleton (Task 6a) to use real data:
    - Wire instance list to actual `workspace.Manager.ListInstances()`
    - Custom list item delegate showing: index, name, status icon (● active, ○ idle, ⏸ paused, ✗ error, ✓ merged, ✔ done), elapsed time, sub-terminal count, conflict indicator (⚠)
    - Status icons with lipgloss colors (green=active, yellow=idle, gray=paused, red=error, blue=merged, cyan=done)
    - Auto-refresh tick (every 1-2 seconds) to update statuses, elapsed times
    - Periodic conflict check (every 30s) using `git.HasConflicts()` across instances
  - Implement hotkey dispatch from dashboard:
    - `n` → switch to Create view
    - `d` → confirmation overlay → delete
    - `e` → call IDE launcher
    - `t` → create sub-terminal
    - `T` → list sub-terminals
    - `f` → switch to Diff view
    - `m` → switch to Merge view
    - `Enter` → switch to Instance/Focus view
    - `1-9` → jump to instance N and focus
    - `p` → pause/resume selected instance
    - `r` → rename overlay
    - `?` → switch to Help view
    - `q` → quit (detach)
    - `Q` → quit and kill all

  **Must NOT do**:
  - No implementation of the views themselves (those are separate tasks)
  - Dashboard dispatches to views; it doesn't implement them

  **Recommended Agent Profile**:
  - **Category**: `visual-engineering`
  - **Skills**: [`frontend-ui-ux`]
    - Visual polish, status icons, color scheme, layout refinement

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 4 (with Tasks 8, 9, 10, 11)
  - **Blocks**: Tasks 12, 13, 14, 15
  - **Blocked By**: Tasks 5, 6a

  **References**:

  **Pattern References**:
  - claude-squad `ui/list.go`: Instance list rendering with status icons and diff stats
  - claude-squad `app/app.go:147`: Layout calculation — list 30% width, preview 70% width
  - Spec dashboard mockup: The ASCII art showing instance list with status, time, sub-terminal count

  **Acceptance Criteria**:
  - [ ] Dashboard renders real instances from state
  - [ ] Status icons show correct colors for each status
  - [ ] Elapsed time updates on tick
  - [ ] Sub-terminal count displays next to each instance
  - [ ] All hotkeys dispatch to correct actions/views
  - [ ] j/k, ↑/↓, 1-9 navigation works
  - [ ] q quits cleanly, Q kills all then quits

  **Agent-Executed QA Scenarios**:
  ```
  Scenario: Dashboard displays real instances with status
    Tool: interactive_bash (tmux)
    Preconditions: ocw binary built, test repo initialized, tmux available, at least 1 instance created
    Steps:
      1. tmux new-session -d -s test-dash
      2. tmux send-keys -t test-dash "./ocw" Enter
      3. Sleep 2s
      4. tmux capture-pane -t test-dash -p
      5. Assert: output contains instance name
      6. Assert: output contains status indicator (●, ○, etc.)
      7. tmux send-keys -t test-dash "q"
    Expected Result: Real instance data displayed
    Evidence: Pane capture saved
  ```

  **Commit**: YES
  - Message: `feat(tui): complete dashboard view with real instance data, status icons, and hotkey dispatch`
  - Files: `internal/tui/views/dashboard.go`
  - Pre-commit: `go build -o ocw .`

---

- [x] 8. Instance Creation Flow (Create View)

  **What to do**:
  - Create `internal/tui/views/create.go`:
    - Multi-step form using `charmbracelet/huh`:
      1. Text input: Branch name (with validation — no spaces, valid git ref chars)
      2. Text input: Base branch (default: config.Workspace.BaseBranch, or detect main/master)
    - On submit: call `workspace.Manager.CreateInstance(opts)`
    - Show spinner during creation (worktree + tmux window + opencode launch)
    - On success: switch to dashboard with new instance selected
    - On error: show error message, allow retry
  - Wire `n` hotkey in dashboard to switch to create view
  - Also implement `ocw new <branch>` CLI command (non-interactive):
    - Accept branch name as arg, base branch as flag (--base)
    - Call same CreateInstance logic
    - Print instance ID and status on success

  **Must NOT do**:
  - No template branches (post-MVP)
  - No auto-naming from prompts

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []
    - Form handling with huh, workspace manager integration

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 4 (with Tasks 6, 9, 10, 11)
  - **Blocks**: Task 15
  - **Blocked By**: Tasks 5, 6a

  **References**:

  **Pattern References**:
  - claude-squad instance creation: `app/app.go` stateNew flow — text input for name → Start(true) → statePrompt
  - huh forms: https://github.com/charmbracelet/huh — NewForm with NewGroup, NewInput, validation

  **Acceptance Criteria**:
  - [ ] `n` from dashboard opens create view
  - [ ] Branch name input validates (rejects spaces, empty)
  - [ ] Base branch defaults to configured value
  - [ ] Creating instance shows spinner then returns to dashboard
  - [ ] New instance appears in dashboard list
  - [ ] `ocw new test-branch` creates instance from CLI
  - [ ] Error on duplicate branch name shows clear message

  **Agent-Executed QA Scenarios**:
  ```
  Scenario: Create instance via TUI
    Tool: interactive_bash (tmux)
    Preconditions: ocw running in tmux, test repo initialized
    Steps:
      1. tmux send-keys -t test "n"
      2. Sleep 1s
      3. tmux capture-pane -t test -p
      4. Assert: shows branch name input
      5. tmux send-keys -t test "test/feature-1" Enter
      6. Sleep 3s (worktree creation)
      7. tmux capture-pane -t test -p
      8. Assert: dashboard shows "test/feature-1" in list
    Expected Result: Instance created and visible
    Evidence: Pane captures saved

  Scenario: Create instance via CLI
    Tool: Bash
    Steps:
      1. Run: ./ocw new test/cli-feature --base master
      2. Assert: exit code 0
      3. Assert: stdout contains instance ID
      4. Run: ./ocw list
      5. Assert: output contains "test/cli-feature"
    Expected Result: CLI creation works
    Evidence: Output captured
  ```

  **Commit**: YES
  - Message: `feat(tui): add instance creation flow with branch input and worktree setup`
  - Files: `internal/tui/views/create.go, cmd/new.go`
  - Pre-commit: `go build -o ocw .`

---

- [x] 9. Sub-Terminal Management

  **What to do**:
  - Create `internal/workspace/subterminal.go`:
    - `func (m *Manager) CreateSubTerminal(instanceID, label string) (string, error)`:
      1. Load instance from state
      2. Count existing sub-terminals
      3. Determine split direction: first = horizontal (-v flag in tmux, splits below), second = vertical (-h flag, splits right in bottom area)
      4. Get primary pane ratio from config (default 70)
      5. `tmux split-window -t <primary_pane> -v -p 30 -c <worktree_path>` (first sub-terminal)
      6. Return new pane ID
      7. Update state with new SubTerminal entry
    - `func (m *Manager) ListSubTerminals(instanceID string) ([]state.SubTerminal, error)`
    - `func (m *Manager) KillSubTerminal(instanceID, paneID string) error` — kill pane, update state
    - `func (m *Manager) KillAllSubTerminals(instanceID string) error` — kill all sub-terminal panes for instance
    - `func (m *Manager) SubTerminalCount(instanceID string) int`
  - Wire into TUI:
    - `t` hotkey in dashboard creates sub-terminal for selected instance
    - `T` hotkey shows sub-terminal list overlay, allows focusing one
  - Handle edge cases:
    - Too many panes (>6): warn user
    - Sub-terminal dies: detect via `#{pane_dead}` on tick, remove from state

  **Must NOT do**:
  - No named/labeled sub-terminals with quick-switch (post-MVP)
  - No auto-run commands on sub-terminal creation (post-MVP)
  - No configurable layouts beyond defaults

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []
    - tmux pane management, state updates, TUI overlay

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 4 (with Tasks 6, 8, 10, 11)
  - **Blocks**: Tasks 14, 15
  - **Blocked By**: Tasks 4, 5

  **References**:

  **Pattern References**:
  - Spec "Sub-Terminals" section: Layout diagram, split behavior, hotkeys
  - Spec "tmux Layout Per Instance" section: Pane numbering, default splits

  **Acceptance Criteria**:
  - [ ] `t` creates sub-terminal pane in selected instance's window
  - [ ] First sub-terminal splits horizontally (70/30)
  - [ ] Sub-terminal is cd'd to worktree path
  - [ ] Sub-terminal count updates on dashboard
  - [ ] `T` lists sub-terminals for selected instance
  - [ ] Dead sub-terminals are detected and cleaned from state

  **Agent-Executed QA Scenarios**:
  ```
  Scenario: Create and verify sub-terminal
    Tool: Bash
    Preconditions: Instance exists with tmux window
    Steps:
      1. Count panes: tmux list-panes -t <window> | wc -l
      2. Assert: 1 pane (opencode only)
      3. Create sub-terminal via manager
      4. Count panes again
      5. Assert: 2 panes
      6. Verify new pane working directory
    Expected Result: Sub-terminal created with correct CWD
    Evidence: tmux list-panes output
  ```

  **Commit**: YES
  - Message: `feat(workspace): add sub-terminal pane management with default split layout`
  - Files: `internal/workspace/subterminal.go, internal/tui/views/dashboard.go (hotkey wiring)`
  - Pre-commit: `go build -o ocw .`

---

- [x] 10. Diff View

  **What to do**:
  - Create `internal/tui/views/diff.go`:
    - Shows git diff --stat for selected instance against its base branch
    - Layout:
      ```
      Header: "Diff: <branch> → <base_branch>"
      Summary: "4 files changed, +127 -23"
      File list with status icons:
        M  src/middleware/auth.ts
        A  src/utils/jwt.ts
        D  src/old/legacy.ts
      ```
    - Use `bubbles/viewport` for scrollable file list
    - Color-coded: green for additions (+), red for deletions (-), status letters colored (M=yellow, A=green, D=red, R=blue)
    - `Esc` returns to dashboard
    - Auto-refresh on entry (fetch latest diff)
  - Wire `f` hotkey in dashboard to switch to diff view for selected instance
  - Also implement `ocw diff <id|name>` CLI command:
    - Non-TUI output of `git diff --stat`

  **Must NOT do**:
  - No file-level expand/collapse (keep it simple: --stat only)
  - No inline diff content (just stat summary)

  **Recommended Agent Profile**:
  - **Category**: `visual-engineering`
  - **Skills**: [`frontend-ui-ux`]
    - Visual diff display, color coding, viewport scrolling

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 4 (with Tasks 6, 8, 9, 11)
  - **Blocks**: Task 13
  - **Blocked By**: Tasks 3, 6a

  **References**:

  **Pattern References**:
  - claude-squad `ui/tabbed_window.go`: Tab switching between Preview/Diff with scroll handling
  - Spec "Diff View" section and merge flow mockup showing file list format

  **Acceptance Criteria**:
  - [ ] `f` shows diff view for selected instance
  - [ ] File list shows correct status icons (M/A/D/R)
  - [ ] Summary line shows file count, insertions, deletions
  - [ ] Viewport scrolls for long file lists
  - [ ] Esc returns to dashboard
  - [ ] `ocw diff <name>` prints diff stat to stdout

  **Agent-Executed QA Scenarios**:
  ```
  Scenario: Diff view shows file changes
    Tool: interactive_bash (tmux)
    Preconditions: Instance exists with some file changes
    Steps:
      1. Launch ocw in tmux
      2. Select instance with changes
      3. Send "f" key
      4. Sleep 1s
      5. Capture pane
      6. Assert: output contains "files changed"
      7. Assert: output contains file path with M/A/D prefix
      8. Send Escape
      9. Assert: back on dashboard
    Expected Result: Diff stats displayed correctly
    Evidence: Pane captures
  ```

  **Commit**: YES
  - Message: `feat(tui): add diff view showing git diff --stat with color-coded file list`
  - Files: `internal/tui/views/diff.go, cmd/diff.go`
  - Pre-commit: `go build -o ocw .`

---

- [x] 11. Conflict Detection

  **What to do**:
  - Create `internal/workspace/conflicts.go`:
    - `func (m *Manager) DetectConflicts() (map[string][]string, error)`:
      1. For each pair of active instances on different branches:
      2. Get modified files for each: `git diff --name-only <base>..<branch>`
      3. Compute intersection of modified file sets
      4. If overlap → record conflict: `instance1.ConflictsWith = append(instance1.ConflictsWith, instance2.ID)`
      5. Update state with conflict info
      6. Return map of instanceID → []conflicting_instance_IDs
    - `func (m *Manager) CheckMergeConflicts(instanceID string) (bool, []string, error)`:
      1. Get instance's branch and base branch
      2. `git merge-base <base> <branch>` — get merge base explicitly
      3. `git merge-tree --write-tree --merge-base=<merge_base> <base> <branch>`
      4. Parse output for conflict markers
      5. Return whether conflicts exist and which files
  - Wire into dashboard:
    - Run `DetectConflicts()` on a periodic tick (every 30 seconds)
    - Show ⚠ icon next to instances with overlapping file modifications
    - Show conflict details in instance status area

  **Must NOT do**:
  - No conflict resolution (post-MVP)
  - No blocking merge on file overlap (that's just a warning — merge-tree conflicts DO block)

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []
    - Git diff parsing, set intersection logic

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 4 (with Tasks 6, 8, 9, 10)
  - **Blocks**: Task 6 (enriches dashboard), Task 17
  - **Blocked By**: Tasks 3, 5

  **References**:

  **Pattern References**:
  - Spec "Conflict detection" section: `git merge-tree` usage, periodic checks, ⚠ indicator
  - Metis finding: Always use `--merge-base=$(git merge-base main <branch>)` explicitly

  **Acceptance Criteria**:
  - [ ] DetectConflicts finds overlapping file modifications between instances
  - [ ] CheckMergeConflicts uses explicit merge-base (Metis guardrail)
  - [ ] Dashboard shows ⚠ for instances with overlapping modifications
  - [ ] Conflict data stored in state.json

  **Agent-Executed QA Scenarios**:
  ```
  Scenario: Conflict detection finds overlapping files
    Tool: Bash
    Preconditions: Two instances modifying the same file
    Steps:
      1. Create two instances
      2. In each worktree, modify the same file
      3. Run DetectConflicts
      4. Assert: both instances show in each other's conflicts_with
    Expected Result: Overlapping modifications detected
    Evidence: State.json conflicts_with populated
  ```

  **Commit**: YES
  - Message: `feat(workspace): add cross-instance conflict detection using git merge-tree`
  - Files: `internal/workspace/conflicts.go`
  - Pre-commit: `go build ./...`

---

### WAVE 5: Integration

- [x] 12. Focus View (Instance View with ReleaseTerminal)

  **What to do**:
  - Create `internal/tui/views/instance.go`:
    - When user presses Enter on an instance, OCW must:
      1. Call `program.ReleaseTerminal()` (NOT tea.Suspend — Metis guardrail)
      2. Execute `tmux attach-session -t <session> \; select-window -t <window>` as a subprocess with stdin/stdout/stderr attached
      3. When user presses `Ctrl+b d` (tmux detach) or `Esc` binding, subprocess exits
      4. Call `program.RestoreTerminal()`
      5. Return to dashboard with proper error handling
    - Handle errors:
      - If ReleaseTerminal fails: show error message, stay on dashboard
      - If tmux attach fails: show error, RestoreTerminal, return to dashboard
      - If RestoreTerminal fails: fatal error with clear message
    - Also support `Ctrl+n` / `Ctrl+p` to cycle instances while focused:
      - These would be tmux key bindings set up in the OCW session, not Bubbletea keys
  - Wire hotkeys:
    - `Enter` from dashboard → focus selected instance
    - `1-9` from dashboard → jump to instance N and focus

  **Must NOT do**:
  - Do NOT use tea.Suspend (silent failure mode — Metis guardrail)
  - Do NOT use PTY management (claude-squad pattern — too complex for OCW)
  - No output streaming in dashboard (post-MVP)

  **Recommended Agent Profile**:
  - **Category**: `deep`
  - **Skills**: []
    - Complex terminal lifecycle management. ReleaseTerminal/RestoreTerminal + subprocess. Needs careful error handling.

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 5 (with Tasks 13, 14, 15, 16)
  - **Blocks**: Task 15
  - **Blocked By**: Tasks 4, 6

  **References**:

  **Pattern References**:
  - Glow suspend pattern: https://github.com/charmbracelet/glow/blob/master/ui/ui.go — tea.Suspend usage (we DON'T use this, but reference for understanding)
  - Bubbletea Program: `ReleaseTerminal()` and `RestoreTerminal()` methods on `*tea.Program`
  - Metis finding: "Do NOT use tea.Suspend() directly. Use ReleaseTerminal/RestoreTerminal imperatively with explicit error handling."
  - claude-squad `session/tmux/tmux.go:269`: Attach() method — goroutines for I/O, context cancellation (different approach but shows the complexity)

  **Acceptance Criteria**:
  - [ ] Enter on instance attaches to tmux window
  - [ ] User can interact with opencode and sub-terminals in the window
  - [ ] Detaching from tmux (Ctrl+b d) returns to OCW dashboard
  - [ ] ReleaseTerminal errors are handled gracefully (no hang)
  - [ ] Terminal state is fully restored after focus (colors, cursor, alternate screen)

  **Agent-Executed QA Scenarios**:
  ```
  Scenario: Focus and return to dashboard
    Tool: interactive_bash (tmux)
    Preconditions: OCW running with at least 1 instance
    Steps:
      1. Capture dashboard pane
      2. Assert: dashboard visible
      3. Send Enter to focus instance
      4. Sleep 2s
      5. Capture pane
      6. Assert: tmux window content visible (opencode or shell)
      7. Send Ctrl+b then d (detach)
      8. Sleep 2s
      9. Capture pane
      10. Assert: dashboard visible again
    Expected Result: Clean focus/return cycle
    Evidence: Three pane captures (before, during, after)
  ```

  **Commit**: YES
  - Message: `feat(tui): add focus view using ReleaseTerminal/RestoreTerminal for tmux attach`
  - Files: `internal/tui/views/instance.go`
  - Pre-commit: `go build -o ocw .`

---

- [x] 13. Merge View & PR Creation

  **What to do**:
  - Create `internal/tui/views/merge.go`:
    - Merge flow UI matching spec mockup:
      ```
      Header: "Merge: <branch> → <base>"
      Changes: diff stat summary
      File list: M/A/D with paths
      PR Title: text input (default: branch name formatted)
      PR Body: text area (optional, auto-generated from diff)
      Conflict status: "No conflicts ✓" or "⚠ Conflicts in: <files>"
      Footer: [Enter] Push & Create PR    [Esc] Cancel
      ```
    - On submit:
      1. Run conflict check (Task 11's CheckMergeConflicts)
      2. If conflicts → block with error, show conflicting files
      3. If clean → `git push origin <branch>`
      4. Create PR: `gh pr create --title "<title>" --body "<body>" --base <base>` (or `glab mr create`)
      5. Show PR URL on success
      6. Update instance status to "merged" in state
      7. Prompt: delete worktree now or keep?
  - Create `internal/workspace/merge.go`:
    - `func (m *Manager) PushBranch(instanceID string) error`
    - `func (m *Manager) CreatePR(instanceID, title, body string) (string, error)` — returns PR URL
    - `func (m *Manager) DetectPRTool() (string, error)` — checks for `gh` or `glab`
  - Wire `m` hotkey in dashboard to switch to merge view
  - Implement `ocw merge <id|name>` CLI command

  **Must NOT do**:
  - No auto-generated PR descriptions from OpenCode activity (post-MVP)
  - No PR template support (post-MVP feature, but config field exists)
  - No draft PR support in MVP (config field exists for later)

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []
    - Git push, gh CLI integration, TUI form, conflict checking

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 5 (with Tasks 12, 14, 15, 16)
  - **Blocks**: Task 15
  - **Blocked By**: Tasks 3, 6, 10

  **References**:

  **Pattern References**:
  - Spec "Merge (PR-Based)" section and merge flow mockup
  - claude-squad PR creation: Uses `gh` CLI for push + PR

  **Acceptance Criteria**:
  - [ ] `m` opens merge view with diff summary and PR form
  - [ ] Conflict check blocks merge when conflicts exist
  - [ ] Clean merge pushes branch and creates PR via gh/glab
  - [ ] PR URL displayed after creation
  - [ ] Instance status updated to "merged"
  - [ ] Missing gh/glab shows clear install instructions error
  - [ ] `ocw merge <name>` works from CLI

  **Agent-Executed QA Scenarios**:
  ```
  Scenario: Merge view shows diff and creates PR
    Tool: interactive_bash (tmux)
    Preconditions: Instance with changes, gh CLI installed, remote configured
    Steps:
      1. Select instance, press "m"
      2. Assert: merge view shows diff summary
      3. Assert: PR title field pre-filled
      4. Press Enter to confirm
      5. Assert: PR URL shown or push succeeds
    Expected Result: PR creation flow works end-to-end
    Evidence: Pane captures and PR URL

  Scenario: Missing gh CLI shows error
    Tool: Bash
    Preconditions: gh not installed
    Steps:
      1. Run: ./ocw merge test-instance
      2. Assert: stderr contains "gh" and "install"
    Expected Result: Clear error message
    Evidence: Output captured
  ```

  **Commit**: YES
  - Message: `feat(tui): add merge view with PR creation via gh/glab CLI`
  - Files: `internal/tui/views/merge.go, internal/workspace/merge.go, cmd/merge.go`
  - Pre-commit: `go build -o ocw .`

---

- [x] 14. Instance Deletion Flow

  **What to do**:
  - Create confirmation overlay in dashboard:
    - On `d` hotkey: show "Delete <instance-name>? All sub-terminals will be killed. (y/n)"
    - If dirty worktree: show additional warning "Worktree has uncommitted changes!"
    - On confirm: call `workspace.Manager.DeleteInstance(id, force)`
    - Deletion sequence:
      1. Kill all sub-terminal panes
      2. Kill tmux window
      3. `git worktree remove <path>` (force if confirmed)
      4. Prompt: "Also delete branch <branch>? (y/n)"
      5. If yes: `git branch -D <branch>`
      6. Remove from state
    - On cancel: return to dashboard
  - Implement `ocw delete <id|name>` CLI command with `--force` flag
  - Handle edge case: instance already dead (tmux window gone) — clean up state only

  **Must NOT do**:
  - No batch delete
  - No undo/restore

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []
    - Destruction flow, confirmation UI, cleanup logic

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 5 (with Tasks 12, 13, 15, 16)
  - **Blocks**: Task 15
  - **Blocked By**: Tasks 5, 6, 9

  **References**:

  **Pattern References**:
  - Spec "Tear Down Instance" section: Step-by-step deletion flow
  - claude-squad `Kill()` method in `instance.go`: Close tmux session + cleanup git worktree

  **Acceptance Criteria**:
  - [ ] `d` shows confirmation before deleting
  - [ ] Dirty worktree shows additional warning
  - [ ] All sub-terminals killed before instance
  - [ ] Worktree removed from filesystem
  - [ ] State updated (instance removed)
  - [ ] Optional branch deletion works
  - [ ] `ocw delete <name>` works from CLI
  - [ ] `ocw delete <name> --force` skips confirmation

  **Agent-Executed QA Scenarios**:
  ```
  Scenario: Delete instance cleans up all resources
    Tool: Bash
    Preconditions: Instance exists with sub-terminals
    Steps:
      1. Run: tmux list-windows -t <session> (count before)
      2. Run: git worktree list (count before)
      3. Run: ./ocw delete <instance> --force
      4. Assert: exit code 0
      5. Run: tmux list-windows -t <session> (count after)
      6. Assert: window count decreased by 1
      7. Run: git worktree list
      8. Assert: worktree no longer listed
      9. Run: ./ocw list
      10. Assert: instance no longer listed
    Expected Result: Complete cleanup
    Evidence: Before/after outputs
  ```

  **Commit**: YES
  - Message: `feat(tui): add instance deletion with confirmation, cleanup, and branch removal`
  - Files: `internal/tui/views/dashboard.go (overlay), cmd/delete.go`
  - Pre-commit: `go build -o ocw .`

---

- [x] 15. CLI Subcommands (All Remaining)

  **What to do**:
  - Create cobra commands for all remaining CLI subcommands:
    - `cmd/init.go`: `ocw init` — calls `config.InitWorkspace()`, creates tmux session
    - `cmd/list.go`: `ocw list` — tabular output of instances (name, branch, status, created_at)
    - `cmd/focus.go`: `ocw focus <id|name>` — attaches to instance's tmux window directly
    - `cmd/term.go`: `ocw term <id|name>` — creates sub-terminal in instance
    - `cmd/edit.go`: `ocw edit <id|name>` — opens worktree in IDE
    - `cmd/status.go`: `ocw status` — JSON dump of full state
    - `cmd/kill.go`: `ocw kill` — kills all instances, tmux session, cleans up
    - `cmd/config.go`: `ocw config` — opens config.toml in $EDITOR
  - Note: `cmd/new.go`, `cmd/diff.go`, `cmd/merge.go`, `cmd/delete.go` were created in Tasks 8, 10, 13, 14
  - Default command (no subcommand): Launch TUI dashboard or re-attach to existing session
  - Re-attach logic: if OCW tmux session exists, attach to it; if not, create new and launch TUI

  **Must NOT do**:
  - No complex flag parsing beyond what's needed
  - No shell completions (post-MVP)

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []
    - Cobra CLI commands, wiring to workspace manager

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 5 (with Tasks 12, 13, 14, 16)
  - **Blocks**: Task 17
  - **Blocked By**: Tasks 5, 6, 7, 8, 9, 12, 13, 14

  **References**:

  **Pattern References**:
  - Spec "CLI Interface" section: Full command list
  - Glow main.go: Cobra subcommands + default TUI mode pattern

  **Acceptance Criteria**:
  - [ ] All CLI commands listed in spec are implemented
  - [ ] `ocw init` creates .ocw/ directory
  - [ ] `ocw list` shows tabular output
  - [ ] `ocw status` outputs valid JSON
  - [ ] `ocw kill` destroys all instances and tmux session
  - [ ] `ocw` with no args launches TUI or re-attaches
  - [ ] `ocw` in non-git-repo shows clear error

  **Agent-Executed QA Scenarios**:
  ```
  Scenario: All CLI commands execute without crash
    Tool: Bash
    Preconditions: Test repo initialized with .ocw/
    Steps:
      1. Run: ./ocw init (in fresh repo)
      2. Assert: .ocw/ created
      3. Run: ./ocw list
      4. Assert: exit code 0 (even with no instances)
      5. Run: ./ocw status
      6. Assert: valid JSON output
      7. Run: ./ocw config (with EDITOR=cat)
      8. Assert: config.toml content printed
    Expected Result: All commands work
    Evidence: Outputs captured

  Scenario: Re-attach on relaunch
    Tool: Bash
    Steps:
      1. Run: ./ocw init (creates tmux session)
      2. Verify: tmux has-session -t ocw-<repo> (should exist)
      3. Run: ./ocw (should attach to existing session)
    Expected Result: Re-attachment works
    Evidence: tmux session list
  ```

  **Commit**: YES
  - Message: `feat(cli): add all remaining CLI subcommands (init, list, focus, term, edit, status, kill, config)`
  - Files: `cmd/*.go`
  - Pre-commit: `go build -o ocw .`

---

- [x] 16. Startup Reconciliation & Crash Recovery

  **What to do**:
  - Create `internal/workspace/recovery.go`:
    - `func (m *Manager) Reconcile() error` — called on every OCW startup:
      1. Run `git worktree repair` (fixes broken worktree references)
      2. Run `git worktree prune` (removes stale worktree entries)
      3. Load state.json
      4. For each instance in state:
         a. Check if worktree directory exists on disk
         b. Check if tmux window exists: `tmux has-session -t <session>` + `tmux list-windows`
         c. Check if opencode process alive: `kill -0 <pid>` (signal 0 = check existence)
         d. Reconcile state vs reality:
            - Worktree exists + tmux exists + process alive → status = active/idle (check pane content)
            - Worktree exists + tmux gone + process alive → recreate tmux window, adopt process
            - Worktree exists + tmux gone + process dead → status = error, offer restart
            - Worktree gone + tmux exists → kill tmux window, mark error
            - Worktree gone + tmux gone → remove from state (clean gone)
      5. Remove orphaned state entries
      6. Save reconciled state
    - Handle case: tmux session itself is gone (tmux crash)
      - Recreate OCW session
      - For each instance with living worktree: recreate window, optionally relaunch opencode

  **Must NOT do**:
  - No automatic relaunch of opencode (prompt user)
  - No state migration between versions

  **Recommended Agent Profile**:
  - **Category**: `deep`
  - **Skills**: []
    - Complex state reconciliation logic. Many edge cases. Needs careful reasoning.

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 5 (with Tasks 12, 13, 14, 15)
  - **Blocks**: Task 17
  - **Blocked By**: Tasks 2, 4, 5

  **References**:

  **Pattern References**:
  - Metis findings: "Store opencode PID (not tmux pane PID)", "Run git worktree repair + prune on startup", "If state says instances exist but tmux session doesn't, attempt recovery"
  - lazygit worktree_loader.go: Worktree health checking, detecting orphaned worktrees
  - claude-squad LoadInstances: Attempts to restore instances on startup, handles paused vs running

  **Acceptance Criteria**:
  - [ ] Reconcile() runs on every OCW startup without error
  - [ ] Orphaned state entries (worktree+tmux gone) are cleaned up
  - [ ] Worktrees with dead processes show "error" status
  - [ ] git worktree repair runs before reconciliation
  - [ ] Recreates tmux session if it was killed
  - [ ] Does not auto-relaunch opencode (just marks status)

  **Agent-Executed QA Scenarios**:
  ```
  Scenario: Recover from killed tmux session
    Tool: Bash
    Preconditions: OCW initialized with instances, worktrees exist on disk
    Steps:
      1. Kill tmux session: tmux kill-session -t ocw-<repo>
      2. Run: ./ocw
      3. Assert: OCW starts (recreates session)
      4. Run: ./ocw list
      5. Assert: instances listed with "error" status (process dead)
    Expected Result: Graceful recovery from tmux crash
    Evidence: List output showing error status
  ```

  **Commit**: YES
  - Message: `feat(workspace): add startup reconciliation and crash recovery for orphaned instances`
  - Files: `internal/workspace/recovery.go`
  - Pre-commit: `go build -o ocw .`

---

### WAVE 6: Polish & Post-MVP

- [x] 17. Edge Cases & Error Handling Polish

  **What to do**:
  - Implement all edge cases from spec:
    - Dirty worktree on delete: warn user, require confirmation
    - Branch already exists: offer to check out existing branch
    - tmux not installed: clear error with install instructions
    - gh/glab not installed: error on merge with install instructions
    - OpenCode crashes: detect exit, show error status, offer restart option
    - Repo not git: error on init with clear message
    - No remote configured: error on merge, suggest `git remote add`
    - Nested worktrees: prevent creating OCW inside a worktree
    - Too many panes (>6): warn user
    - IDE not found: fall back to $EDITOR → vi
    - GUI editor from SSH: detect headless, default to terminal editor
    - Sub-terminal dies: detect, update state, show on dashboard
  - Add dependency checking on startup:
    - Check tmux installed and version
    - Check git installed and version
    - Warn if gh/glab not installed (only error on merge)
  - Improve error messages throughout: every error should tell the user what to do next

  **Must NOT do**:
  - No new features — polish only
  - No refactoring of working code

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []
    - Error handling, edge cases, defensive programming

  **Parallelization**:
  - **Can Run In Parallel**: NO
  - **Parallel Group**: Sequential (depends on all MVP tasks)
  - **Blocks**: Task 18
  - **Blocked By**: Tasks 15, 16

  **References**:

  **Pattern References**:
  - Spec "Edge Cases & Considerations" section — full list of edge cases

  **Acceptance Criteria**:
  - [ ] Every edge case from spec has a handler
  - [ ] Error messages include actionable next steps
  - [ ] Dependency check runs on startup
  - [ ] No panics on unexpected input

  **Agent-Executed QA Scenarios**:
  ```
  Scenario: Missing tmux shows clear error
    Tool: Bash
    Preconditions: tmux not in PATH
    Steps:
      1. PATH=/usr/bin ./ocw
      2. Assert: stderr contains "tmux" and "install"
      3. Assert: exit code non-zero
    Expected Result: Clear install instructions
    Evidence: Error output

  Scenario: Non-git directory shows error
    Tool: Bash
    Steps:
      1. Run: cd /tmp && ./ocw init
      2. Assert: stderr contains "git repository"
    Expected Result: Clear error
    Evidence: Output captured
  ```

  **Commit**: YES
  - Message: `fix: comprehensive edge case handling and improved error messages`
  - Files: Multiple files across internal/
  - Pre-commit: `go build -o ocw .`

---

- [x] 18. Tests for Core Logic

  **What to do**:
  - Add Go tests for core business logic (NOT TUI):
    - `internal/git/git_test.go`: Test WorktreeList parsing, DiffStat parsing, command construction
    - `internal/git/merge_test.go`: Test MergeTree output parsing, MergeBase
    - `internal/state/state_test.go`: Test roundtrip serialization, AddInstance, RemoveInstance, UpdateInstance, file locking
    - `internal/config/config_test.go`: Test TOML parsing, DefaultConfig, InitWorkspace
    - `internal/workspace/conflicts_test.go`: Test file overlap detection logic
    - `internal/tmux/tmux_test.go`: Test command construction (not execution — no tmux in CI)
    - `internal/ide/launcher_test.go`: Test editor detection, IsTerminalEditor
  - Use standard `testing` package + `testify/assert` for assertions
  - Add `go get github.com/stretchr/testify` to dependencies
  - Tests should be runnable without tmux/git (mock command output where needed)

  **Must NOT do**:
  - No TUI tests (too brittle, agent QA covers this)
  - No integration tests requiring tmux (unit tests only)
  - No snapshot tests

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []
    - Go testing patterns, mocking, testify

  **Parallelization**:
  - **Can Run In Parallel**: YES (can parallelize with post-MVP tasks)
  - **Parallel Group**: Wave 6
  - **Blocks**: None
  - **Blocked By**: Task 17

  **References**:

  **Pattern References**:
  - claude-squad `ui/preview_test.go`: Test setup with tmux and git (reference for test structure)
  - lazygit test patterns: `pkg/commands/git_commands/` tests

  **Acceptance Criteria**:
  - [ ] `go test ./...` passes
  - [ ] Core parsing functions have test coverage
  - [ ] State roundtrip tested
  - [ ] Config defaults tested
  - [ ] Tests run without tmux/git binaries (mocked)

  **Agent-Executed QA Scenarios**:
  ```
  Scenario: All tests pass
    Tool: Bash
    Steps:
      1. Run: go test ./... -v
      2. Assert: exit code 0
      3. Assert: output shows PASS for each package
    Expected Result: All tests green
    Evidence: Test output captured
  ```

  **Commit**: YES
  - Message: `test: add unit tests for core logic (git parsing, state, config, conflict detection)`
  - Files: `internal/*_test.go`
  - Pre-commit: `go test ./...`

---

### POST-MVP FEATURES

- [x] 19. Help View

  **What to do**:
  - Create `internal/tui/views/help.go`:
    - Full-screen help showing all hotkeys in a formatted table
    - Grouped by context (Dashboard, Focus View)
    - `?` toggles help, `Esc` returns to previous view
    - Use lipgloss for clean formatting

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: [`frontend-ui-ux`]

  **Parallelization**: Independent, can run anytime after Task 6

  **Commit**: YES — `feat(tui): add help view with hotkey reference table`

---

- [x] 20. Instance Rename

  **What to do**:
  - Add rename overlay in dashboard:
    - `r` hotkey opens text input pre-filled with current name
    - On submit: update state.Instance.Name
    - Does NOT rename git branch (just the display name)

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **Parallelization**: Independent, can run after Task 6

  **Commit**: YES — `feat(tui): add instance rename overlay`

---

- [x] 21. Send Prompt to Instance (`s` hotkey)

  **What to do**:
  - Add text input overlay for sending prompts:
    - `s` hotkey opens multi-line text input
    - On submit: `tmux send-keys -t <primary_pane> "<text>" Enter`
    - Show feedback: "Prompt sent to <instance>"
  - Requires sending keys to the correct pane (opencode pane, not sub-terminals)

  **Recommended Agent Profile**:
  - **Category**: `unspecified-low`
  - **Skills**: []

  **Parallelization**: Independent, can run after Task 6

  **Commit**: YES — `feat(tui): add send prompt overlay for instances`

---

- [x] 22. View Instance Log/Output (`l` hotkey)

  **What to do**:
  - Add log viewer in TUI:
    - `l` hotkey captures pane content via `tmux capture-pane -p -S - -E -`
    - Display in scrollable viewport (bubbles/viewport)
    - Auto-scroll to bottom
    - `Esc` to return to dashboard
  - Shows full scrollback history of the opencode pane

  **Recommended Agent Profile**:
  - **Category**: `unspecified-low`
  - **Skills**: []

  **Parallelization**: Independent, can run after Task 6

  **Commit**: YES — `feat(tui): add instance log viewer with scrollback history`

---

- [ ] 23. Instance Output Preview/Streaming in Dashboard

  **What to do**:
  - Add a preview pane on the right side of dashboard (like claude-squad's tabbed preview):
    - Shows last N lines of selected instance's opencode output
    - Auto-refreshes on tick (every 100-500ms)
    - Uses SHA256 hash comparison to detect changes (claude-squad pattern)
  - Layout: list 40% | preview 60% (configurable)
  - Tab between Preview/Diff tabs

  **Recommended Agent Profile**:
  - **Category**: `visual-engineering`
  - **Skills**: [`frontend-ui-ux`]

  **Parallelization**: After Task 6

  **Commit**: YES — `feat(tui): add live output preview pane in dashboard`

---

- [x] 24. Named/Labeled Sub-Terminals with Quick-Switch

  **What to do**:
  - Enhance sub-terminal creation to accept optional label
  - `T` hotkey shows list of sub-terminals with labels
  - Quick-switch: `Ctrl+1`, `Ctrl+2` etc. to jump to specific sub-terminal pane
  - Show labels in instance detail view

  **Recommended Agent Profile**:
  - **Category**: `unspecified-low`
  - **Skills**: []

  **Parallelization**: After Task 9

  **Commit**: YES — `feat(workspace): add named sub-terminals with quick-switch`

---

- [x] 25. Auto-Run Commands on Sub-Terminal Creation

  **What to do**:
  - Add config option `[workspace] sub_terminal_init_command = ""` (e.g., `"npm install"`)
  - When creating sub-terminal, optionally run init command
  - Per-instance override possible

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **Parallelization**: After Task 9

  **Commit**: YES — `feat(config): add auto-run command for new sub-terminals`

---

- [x] 26. Auto-Generate PR Descriptions from OpenCode Activity

  **What to do**:
  - Capture opencode pane history on merge
  - Extract key actions/messages
  - Generate PR body from the activity log
  - User can edit before submitting

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []

  **Parallelization**: After Task 13

  **Commit**: YES — `feat(merge): auto-generate PR descriptions from opencode activity`

---

- [x] 27. Conflict Resolution Assistant

  **What to do**:
  - When conflicts detected, offer to launch OpenCode in the conflicting worktree with a prompt to fix conflicts
  - Automated flow: focus instance → send conflict resolution prompt

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []

  **Parallelization**: After Tasks 11, 21

  **Commit**: YES — `feat(workspace): add conflict resolution assistant via OpenCode`

---

- [x] 28. Template Branches

  **What to do**:
  - Config option for template branches: predefined starting points
  - `ocw new --template <name>` applies a template
  - Templates can specify: base branch, initial files, auto-run commands

  **Recommended Agent Profile**:
  - **Category**: `unspecified-low`
  - **Skills**: []

  **Parallelization**: After Task 8

  **Commit**: YES — `feat(config): add template branches for pre-configured starting points`

---

- [ ] 29. Stack-Based Merge Ordering

  **What to do**:
  - Define dependencies between instances (A depends on B)
  - Merge in correct order: B before A
  - Show dependency graph in TUI
  - Block merge if dependency not yet merged

  **Recommended Agent Profile**:
  - **Category**: `ultrabrain`
  - **Skills**: []
    - Dependency graph logic, topological sort

  **Parallelization**: After Task 13

  **Commit**: YES — `feat(merge): add stack-based merge ordering for dependent features`

---

- [ ] 30. `ocw watch` — Auto-Create from Task Queue

  **What to do**:
  - `ocw watch <file>` — reads a YAML/JSON file of tasks
  - Auto-creates instances for each task
  - Optionally sends initial prompt to each instance
  - Monitors completion

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []

  **Parallelization**: After Tasks 8, 21

  **Commit**: YES — `feat(cli): add ocw watch for auto-creating instances from task queue`

---

- [x] 31. README & Documentation

  **What to do**:
  - Create README.md with:
    - Feature overview with dashboard screenshot/GIF
    - Installation instructions (go install, brew, binary release)
    - Quick start guide
    - Full CLI reference
    - Configuration reference
    - Hotkey reference
    - Architecture overview
    - Requirements (tmux, git, gh/glab)

  **Recommended Agent Profile**:
  - **Category**: `writing`
  - **Skills**: []

  **Parallelization**: After Task 17

  **Commit**: YES — `docs: add comprehensive README with installation, usage, and configuration guide`

---

## Commit Strategy

| After Task | Message | Key Files | Verification |
|------------|---------|-----------|--------------|
| 1 | `feat: initialize project scaffold with Go module and directory structure` | go.mod, main.go, cmd/root.go | `go build -o ocw .` |
| 2 | `feat(config): add state persistence and TOML config management` | internal/config/, internal/state/ | `go build ./...` |
| 3 | `feat(git): add git CLI abstraction layer` | internal/git/ | `go build ./...` |
| 4 | `feat(tmux): add tmux CLI abstraction` | internal/tmux/ | `go build ./...` |
| 5 | `feat(workspace): add instance lifecycle manager` | internal/workspace/ | `go build ./...` |
| 6a | `feat(tui): add Bubbletea scaffold with dashboard skeleton` | internal/tui/, cmd/root.go | `go build -o ocw .` |
| 6 | `feat(tui): complete dashboard view with real data` | internal/tui/views/dashboard.go | `go build -o ocw .` |
| 7 | `feat(ide): add editor auto-detection and launcher` | internal/ide/ | `go build ./...` |
| 8 | `feat(tui): add instance creation flow` | internal/tui/views/create.go, cmd/new.go | `go build -o ocw .` |
| 9 | `feat(workspace): add sub-terminal pane management` | internal/workspace/subterminal.go | `go build -o ocw .` |
| 10 | `feat(tui): add diff view` | internal/tui/views/diff.go, cmd/diff.go | `go build -o ocw .` |
| 11 | `feat(workspace): add conflict detection` | internal/workspace/conflicts.go | `go build ./...` |
| 12 | `feat(tui): add focus view with ReleaseTerminal` | internal/tui/views/instance.go | `go build -o ocw .` |
| 13 | `feat(tui): add merge view with PR creation` | internal/tui/views/merge.go, cmd/merge.go | `go build -o ocw .` |
| 14 | `feat(tui): add instance deletion flow` | cmd/delete.go | `go build -o ocw .` |
| 15 | `feat(cli): add all remaining CLI subcommands` | cmd/*.go | `go build -o ocw .` |
| 16 | `feat(workspace): add startup reconciliation` | internal/workspace/recovery.go | `go build -o ocw .` |
| 17 | `fix: comprehensive edge case handling` | Multiple | `go build -o ocw .` |
| 18 | `test: add unit tests for core logic` | *_test.go | `go test ./...` |
| 19-31 | Post-MVP features and docs | Various | `go build -o ocw . && go test ./...` |

---

## Success Criteria

### Verification Commands
```bash
go build -o ocw .                    # Expected: exits 0, binary created
go test ./...                        # Expected: all tests pass
go vet ./...                         # Expected: no issues
./ocw --help                         # Expected: shows all subcommands
./ocw init                           # Expected: creates .ocw/ directory
./ocw new test-branch                # Expected: creates worktree + tmux window + opencode
./ocw list                           # Expected: shows instance table
./ocw status | jq .                  # Expected: valid JSON
./ocw delete test-branch --force     # Expected: cleans up everything
./ocw kill                           # Expected: destroys all instances
```

### Final Checklist
- [ ] All 10 MVP features implemented and working
- [ ] All CLI subcommands from spec implemented
- [ ] All hotkeys from spec implemented
- [ ] All edge cases from spec handled
- [ ] State persistence works across restarts
- [ ] Crash recovery works (tmux killed, opencode crashed)
- [ ] No panics on unexpected input
- [ ] Clean shutdown on q and Q
- [ ] Re-attach on relaunch works
- [ ] Post-MVP features implemented per plan
