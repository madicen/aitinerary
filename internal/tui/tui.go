// Package tui is the public entry point for the aitinerary terminal UI.
package tui

import (
	"github.com/madicen/aitinerary/internal/tui/model"
	"github.com/madicen/aitinerary/internal/tui/util"
)

// Model is the root Bubble Tea model.
type Model = model.Model

// Options configures the root TUI model.
type Options = model.Options

// New constructs a new root TUI model.
func New(opts Options) *Model { return model.New(opts) }

// FlushMouse resets terminal mouse-tracking state on exit.
var FlushMouse = util.FlushMouse
