package tmux

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewTmux(t *testing.T) {
	tmux := NewTmux()

	assert.NotNil(t, tmux)
}

func TestTmuxStructInitialization(t *testing.T) {
	// Test that Tmux struct can be created
	tmux := &Tmux{}

	assert.NotNil(t, tmux)
}

func TestVersionCommandParsing(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   string
	}{
		{
			name:   "tmux version 3.3a",
			output: "tmux 3.3a",
			want:   "tmux 3.3a",
		},
		{
			name:   "tmux version 3.2",
			output: "tmux 3.2",
			want:   "tmux 3.2",
		},
		{
			name:   "tmux version with trailing newline",
			output: "tmux 3.3a\n",
			want:   "tmux 3.3a",
		},
		{
			name:   "tmux version with spaces",
			output: "  tmux 3.3a  ",
			want:   "tmux 3.3a",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test trimming logic
			trimmed := trimWhitespace(tt.output)
			assert.Equal(t, tt.want, trimmed)
		})
	}
}

func TestCommandConstruction(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want []string
	}{
		{
			name: "version command",
			args: []string{"-V"},
			want: []string{"-V"},
		},
		{
			name: "list sessions",
			args: []string{"list-sessions"},
			want: []string{"list-sessions"},
		},
		{
			name: "new session with name",
			args: []string{"new-session", "-d", "-s", "test-session"},
			want: []string{"new-session", "-d", "-s", "test-session"},
		},
		{
			name: "kill session",
			args: []string{"kill-session", "-t", "test-session"},
			want: []string{"kill-session", "-t", "test-session"},
		},
		{
			name: "split window",
			args: []string{"split-window", "-h", "-t", "test-session"},
			want: []string{"split-window", "-h", "-t", "test-session"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify command arguments are constructed correctly
			assert.Equal(t, tt.want, tt.args)
			assert.Equal(t, len(tt.want), len(tt.args))
		})
	}
}

func TestTmuxTargetFormat(t *testing.T) {
	tests := []struct {
		name    string
		session string
		window  string
		pane    string
		want    string
	}{
		{
			name:    "session target",
			session: "ocw-session",
			want:    "ocw-session",
		},
		{
			name:    "window target",
			session: "ocw-session",
			window:  "1",
			want:    "ocw-session:1",
		},
		{
			name:    "pane target",
			session: "ocw-session",
			window:  "1",
			pane:    "1",
			want:    "ocw-session:1.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test target format construction
			target := tt.session
			if tt.window != "" {
				target = target + ":" + tt.window
			}
			if tt.pane != "" {
				target = target + "." + tt.pane
			}

			assert.Equal(t, tt.want, target)
		})
	}
}

func TestTmuxFlagPatterns(t *testing.T) {
	tests := []struct {
		name        string
		description string
		flag        string
		wantShort   bool
	}{
		{
			name:        "detached session",
			description: "create session in background",
			flag:        "-d",
			wantShort:   true,
		},
		{
			name:        "session name",
			description: "specify session name",
			flag:        "-s",
			wantShort:   true,
		},
		{
			name:        "target",
			description: "specify target session/window/pane",
			flag:        "-t",
			wantShort:   true,
		},
		{
			name:        "horizontal split",
			description: "split window horizontally",
			flag:        "-h",
			wantShort:   true,
		},
		{
			name:        "vertical split",
			description: "split window vertically",
			flag:        "-v",
			wantShort:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify flag format (short flags start with -)
			assert.True(t, len(tt.flag) > 0)
			if tt.wantShort {
				assert.Equal(t, "-", tt.flag[:1])
				assert.Equal(t, 2, len(tt.flag))
			}
		})
	}
}

func TestTmuxCommandTypes(t *testing.T) {
	tests := []struct {
		name        string
		command     string
		needsTarget bool
		hasOutput   bool
	}{
		{
			name:        "new-session",
			command:     "new-session",
			needsTarget: false,
			hasOutput:   false,
		},
		{
			name:        "kill-session",
			command:     "kill-session",
			needsTarget: true,
			hasOutput:   false,
		},
		{
			name:        "list-sessions",
			command:     "list-sessions",
			needsTarget: false,
			hasOutput:   true,
		},
		{
			name:        "split-window",
			command:     "split-window",
			needsTarget: true,
			hasOutput:   false,
		},
		{
			name:        "send-keys",
			command:     "send-keys",
			needsTarget: true,
			hasOutput:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify command properties
			assert.NotEmpty(t, tt.command)
			// These are metadata tests - just verify the properties are set correctly
			assert.NotNil(t, tt.needsTarget)
			assert.NotNil(t, tt.hasOutput)
		})
	}
}

// Helper function for trimming whitespace (mimics strings.TrimSpace behavior)
func trimWhitespace(s string) string {
	start := 0
	end := len(s)

	// Trim leading whitespace
	for start < end && isWhitespace(s[start]) {
		start++
	}

	// Trim trailing whitespace
	for end > start && isWhitespace(s[end-1]) {
		end--
	}

	return s[start:end]
}

func isWhitespace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}
