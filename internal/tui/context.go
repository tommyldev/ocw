package tui

import (
	"github.com/tommyzliu/ocw/internal/config"
	"github.com/tommyzliu/ocw/internal/workspace"
)

// Context holds the TUI application context
type Context struct {
	Config  *config.Config
	Manager *workspace.Manager
	Width   int
	Height  int
}

// NewContext creates a new TUI context
func NewContext(cfg *config.Config, mgr *workspace.Manager) *Context {
	return &Context{
		Config:  cfg,
		Manager: mgr,
		Width:   80,
		Height:  24,
	}
}
