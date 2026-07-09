package model

import (
	tea "github.com/charmbracelet/bubbletea"
	overlay "github.com/madicen/bubble-overlay"

	"github.com/madicen/aitinerary/internal/tui/commands"
	"github.com/madicen/aitinerary/internal/tui/overlays"
	"github.com/madicen/aitinerary/internal/tui/util"
)

func (m *Model) openHelpOverlay() tea.Cmd {
	hm := overlays.NewHelpOverlay(m.keys.Sections(), max(40, m.width-6), max(10, m.height-6))
	return tea.Batch(
		m.overlayStack.Push(hm, overlay.DefaultOverlayConfig()),
		func() tea.Msg { return tea.WindowSizeMsg{Width: m.width, Height: m.height} },
	)
}

func (m *Model) openCommandPalette() tea.Cmd {
	enabled := m.palette.Enabled(m.commandContext())
	pm := overlays.NewPaletteOverlay(enabled, max(40, m.width-6), max(10, m.height-6))
	cfg := overlay.DefaultOverlayConfig()
	cfg.CloseOnClickOutside = true
	return tea.Batch(
		m.overlayStack.Push(pm, cfg),
		func() tea.Msg { return tea.WindowSizeMsg{Width: m.width, Height: m.height} },
	)
}

func (m *Model) buildCommandRegistry() *commands.Registry {
	r := commands.New()
	km := m.keys

	r.Register(commands.Command{
		ID: "help.show", Title: "Show keyboard shortcuts", Category: "Global",
		Binding: km.Help, Run: func() tea.Cmd { return m.openHelpOverlay() },
	})
	r.Register(commands.Command{
		ID: "app.quit", Title: "Quit aitinerary", Category: "Global",
		Binding: km.Quit, Run: func() tea.Cmd { util.FlushMouse(); return tea.Quit },
	})
	r.Register(commands.Command{
		ID: "app.refresh", Title: "Refresh ticket data", Category: "Global",
		Binding: km.Refresh, Run: func() tea.Cmd { return m.store.RefreshCmd() },
	})
	r.Register(commands.Command{
		ID: "app.undo", Title: "Undo last action", Category: "Global",
		Binding: km.Undo, Run: func() tea.Cmd { return nil },
	})
	r.Register(commands.Command{
		ID: "tab.board", Title: "Go to Board tab", Category: "Tabs",
		Binding: km.TabBoard, Run: func() tea.Cmd {
			m.mode = modeBoard
			return nil
		},
	})
	r.Register(commands.Command{
		ID: "tab.queue", Title: "Go to Queue tab", Category: "Tabs",
		Binding: km.TabQueue, Run: func() tea.Cmd {
			m.mode = modeQueue
			return nil
		},
	})
	r.Register(commands.Command{
		ID: "tab.epics", Title: "Go to Epics tab", Category: "Tabs",
		Binding: km.TabEpics, Run: func() tea.Cmd {
			m.mode = modeEpics
			return nil
		},
	})
	r.Register(commands.Command{
		ID: "tab.settings", Title: "Go to Settings tab", Category: "Tabs",
		Binding: km.TabSettings, Run: func() tea.Cmd {
			m.mode = modeSettings
			return nil
		},
	})
	r.Register(commands.Command{
		ID: "tab.help", Title: "Go to Help tab", Category: "Tabs",
		Binding: km.TabHelp, Run: func() tea.Cmd {
			m.mode = modeHelp
			return nil
		},
	})
	r.Register(commands.Command{
		ID: "board.search", Title: "Search tickets", Category: "Board",
		Binding: km.ListSearch,
		Enabled: func(c commands.Context) bool { return c.Mode == "board" },
		Run:     func() tea.Cmd { return nil },
	})
	r.Register(commands.Command{
		ID: "board.open", Title: "Open selected ticket", Category: "Board",
		Binding: km.ListOpen,
		Enabled: func(c commands.Context) bool { return c.Mode == "board" },
		Run:     func() tea.Cmd { return nil },
	})
	return r
}
