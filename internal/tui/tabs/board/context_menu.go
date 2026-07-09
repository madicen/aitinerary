package board

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/madicen/aitinerary/internal/theme"
	"github.com/madicen/aitinerary/internal/tui/listnav"
	"github.com/madicen/aitinerary/internal/tui/mouse"
	"github.com/madicen/aitinerary/internal/tui/util"
)

// ContextMenuState holds the long-press context menu state for a board row.
type ContextMenuState struct {
	RowIndex  int
	MouseX    int
	MouseY    int
	PressID   int
	HoverItem int // -1 = none
}

type ctxMenuItem struct {
	Label  string
	Key    string
	Action func(m *Model) tea.Cmd
}

func contextMenuItems() []ctxMenuItem {
	return []ctxMenuItem{
		{Label: "Transition", Key: "t", Action: func(m *Model) tea.Cmd { _, cmd := m.beginEditTransition(); return cmd }},
		{Label: "Assign sprint", Key: "s", Action: func(m *Model) tea.Cmd { _, cmd := m.beginEditSprint(); return cmd }},
		{Label: "Set epic", Key: "p", Action: func(m *Model) tea.Cmd { _, cmd := m.beginEditParent(); return cmd }},
		{Label: "Open in browser", Key: "o", Action: func(m *Model) tea.Cmd { return m.openBrowserCmd() }},
		{Label: "Copy key", Key: "y", Action: func(m *Model) tea.Cmd {
			if k := m.selectedKey(); k != "" {
				return util.CopyPlainTextCmd(k)
			}
			return nil
		}},
	}
}

func (m *Model) handleLongPress(msg tea.MouseMsg) tea.Cmd {
	if m.contextMenu != nil {
		return m.trackContextMenuHover(msg)
	}
	return m.ArmLongPress(m.zone, msg, listnav.LongPressConfig{
		MenuOpen:  m.contextMenu != nil,
		ItemCount: m.visibleTicketCount(),
		RowZoneID: mouse.ZoneBoardRow,
		Threshold: longPressThreshold,
		MakeTick:  func(pressID int) tea.Msg { return LongPressTickMsg{PressID: pressID} },
	})
}

func (m *Model) trackContextMenuHover(msg tea.MouseMsg) tea.Cmd {
	switch msg.Action {
	case tea.MouseActionMotion, tea.MouseActionPress:
		if msg.Button != tea.MouseButtonLeft && msg.Action == tea.MouseActionPress {
			return nil
		}
		m.contextMenu.HoverItem = -1
		for i := range contextMenuItems() {
			if z := m.zone.Get(mouse.ZoneBoardCtxMenuItem(i)); z != nil && z.InBounds(msg) {
				m.contextMenu.HoverItem = i
				break
			}
		}
	case tea.MouseActionRelease:
		m.contextMenu = nil
	}
	return nil
}

func (m *Model) handleContextMenuZone(zoneID string, e tea.MouseMsg) (*Model, tea.Cmd) {
	items := contextMenuItems()
	row := m.selected
	if m.contextMenu != nil {
		row = m.contextMenu.RowIndex
	}
	for i, item := range items {
		if zoneID != mouse.ZoneBoardCtxMenuItem(i) {
			continue
		}
		if e.Action != tea.MouseActionRelease || e.Button != tea.MouseButtonLeft {
			return m, nil
		}
		m.contextMenu = nil
		m.selected = row
		m.detailOpen = true
		return m, item.Action(m)
	}
	if e.Action == tea.MouseActionRelease {
		m.contextMenu = nil
	}
	return m, nil
}

func (m *Model) renderContextMenu() string {
	items := contextMenuItems()
	itemStyle := lipgloss.NewStyle().Foreground(theme.Adaptive(theme.RoleFg))
	hoverStyle := lipgloss.NewStyle().
		Foreground(theme.Adaptive(theme.RoleSelectionFg)).
		Background(theme.Adaptive(theme.RoleSelectionBg))
	keyStyle := lipgloss.NewStyle().Foreground(theme.Adaptive(theme.RoleMuted))

	hover := -1
	if m.contextMenu != nil {
		hover = m.contextMenu.HoverItem
	}

	var rows []string
	for i, item := range items {
		ls, ks := itemStyle, keyStyle
		if i == hover {
			ls, ks = hoverStyle, hoverStyle
		}
		row := fmt.Sprintf("  %s  %s", ls.Render(item.Label), ks.Render(item.Key))
		rows = append(rows, m.mark(mouse.ZoneBoardCtxMenuItem(i), row))
	}

	header := ""
	if m.contextMenu != nil {
		if m.contextMenu.RowIndex >= 0 && m.contextMenu.RowIndex < len(m.rows) {
			td := m.rows[m.contextMenu.RowIndex].Data
			title := td.Ticket.Key + " " + td.Ticket.Summary
			if len(title) > 42 {
				title = title[:39] + "…"
			}
			header = lipgloss.NewStyle().Foreground(theme.Adaptive(theme.RoleSuccess)).Bold(true).Render(title)
		}
	}

	body := strings.Join(rows, "\n")
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.Adaptive(theme.RoleBorderAccent)).
		Padding(0, 1)
	if header != "" {
		return box.Render(header + "\n" + body)
	}
	return box.Render(body)
}
