package model

import (
	tea "github.com/charmbracelet/bubbletea"
	zone "github.com/lrstanley/bubblezone"

	"github.com/madicen/aitinerary/internal/tui/mouse"
	"github.com/madicen/aitinerary/internal/tui/state"
)

func (m *Model) switchTab(next mode) (tea.Model, tea.Cmd) {
	if next == m.mode {
		return m, nil
	}
	m.mode = next
	return m, nil
}

func (m *Model) handleNavigate(t state.NavigateTarget) (tea.Model, tea.Cmd) {
	if t.Err != nil {
		return m, m.pushErrorOverlay(t.Err)
	}
	return m, nil
}

func (m *Model) statusBarY() int {
	if m.height <= 0 {
		return 0
	}
	return m.height - 1
}

func (m *Model) inHeaderMouse(msg tea.MouseMsg) bool {
	h := m.headerHeight()
	return msg.Y >= 0 && msg.Y < h
}

func (m *Model) inStatusBarMouse(msg tea.MouseMsg) bool {
	return m.height > 0 && msg.Y == m.statusBarY()
}

func (m *Model) handleTabBarMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd, bool) {
	if msg.Action != tea.MouseActionPress || msg.Button != tea.MouseButtonLeft {
		return m, nil, false
	}
	if !m.inHeaderMouse(msg) {
		return m, nil, false
	}
	for _, t := range m.tabSpecs() {
		if z := zone.Get(t.zone); z != nil && z.InBounds(msg) {
			upd, cmd := m.switchTab(t.mode)
			return upd, cmd, true
		}
	}
	return m, nil, false
}

func (m *Model) handleStatusBarMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd, bool) {
	if msg.Action != tea.MouseActionPress || msg.Button != tea.MouseButtonLeft {
		return m, nil, false
	}
	if !m.inStatusBarMouse(msg) {
		return m, nil, false
	}
	type action struct {
		zone string
		cmd  tea.Cmd
	}
	actions := []action{
		{mouse.ZoneStatusHelp, m.openHelpOverlay()},
		{mouse.ZoneStatusPalette, m.openCommandPalette()},
		{mouse.ZoneStatusRefresh, m.store.RefreshCmd()},
		{mouse.ZoneStatusQueue, nil},
		{mouse.ZoneStatusUndo, m.undoCmd()},
		{mouse.ZoneStatusQuit, tea.Quit},
	}
	for _, a := range actions {
		if z := zone.Get(a.zone); z != nil && z.InBounds(msg) {
			if a.zone == mouse.ZoneStatusQuit {
				utilFlushMouse()
			}
			if a.zone == mouse.ZoneStatusQueue {
				upd, cmd := m.switchTab(modeQueue)
				return upd, cmd, true
			}
			return m, a.cmd, true
		}
	}
	return m, nil, false
}

func utilFlushMouse() {
	FlushMouse()
}
