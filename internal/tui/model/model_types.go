package model

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	overlay "github.com/madicen/bubble-overlay"

	"github.com/madicen/aitinerary/internal/actions"
	"github.com/madicen/aitinerary/internal/aiconfig"
	"github.com/madicen/aitinerary/internal/config"
	"github.com/madicen/aitinerary/internal/store"
	"github.com/madicen/aitinerary/internal/tui/commands"
	"github.com/madicen/aitinerary/internal/tui/keys"
	"github.com/madicen/aitinerary/internal/tui/state"
)

type mode = state.ViewMode

const (
	modeBoard    = state.ModeBoard
	modeQueue    = state.ModeQueue
	modeEpics    = state.ModeEpics
	modeSettings = state.ModeSettings
	modeHelp     = state.ModeHelp
)

// statusBarState holds display fields for the bottom status line.
type statusBarState struct {
	workspace string
}

// Model is the root Bubble Tea model.
type Model struct {
	opts Options
	mode mode

	keys    keys.Map
	palette *commands.Registry

	tabs map[mode]Tab

	width  int
	height int

	overlayStack overlay.OverlayStack
	overlayFocus overlay.FocusTrap

	spinner spinner.Model

	config *config.Config
	store  *store.Store
	queue  *actions.Queue
	engine *actions.Engine

	demo bool

	status statusBarState
}

func (m *Model) activeAIConfig() *aiconfig.Config {
	if m.opts.AIConfig != nil {
		return m.opts.AIConfig
	}
	return aiconfig.DefaultConfig()
}

func (m *Model) freshnessLabel() string {
	if m.store == nil {
		return "never"
	}
	if m.store.Refreshing() {
		return "refreshing…"
	}
	t := m.store.LastRefresh()
	if t.IsZero() {
		return "never"
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}

func (m *Model) queueCount() int {
	if m.queue == nil {
		return 0
	}
	return m.queue.Count()
}

func (m *Model) activeProfileLabel() string {
	cfg := m.activeAIConfig()
	if cfg == nil {
		return "default"
	}
	name := cfg.ActiveProfile
	if name == "" {
		name = aiconfig.DefaultProfileName
	}
	return name
}
