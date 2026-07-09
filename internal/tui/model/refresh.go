package model

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/madicen/aitinerary/internal/tui/data"
)

func (m *Model) refreshAllCmd() tea.Cmd {
	return data.RefreshAllCmd(m.config, m.demo, m.store, m.jira, m.engine, m.autoApplied)
}

func (m *Model) refreshStoreOnlyCmd() tea.Cmd {
	return m.store.RefreshCmd()
}

// RefreshAllCmdForTest exposes refresh for tests.
func (m *Model) RefreshAllCmdForTest() tea.Cmd { return m.refreshAllCmd() }

// HandleRefreshMsg routes refresh messages to tabs (tests).
func (m *Model) HandleRefreshMsg(msg tea.Msg) (*Model, tea.Cmd) {
	upd, cmd := m.Update(msg)
	return upd.(*Model), cmd
}
