package model

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"

	"github.com/madicen/aitinerary/internal/tui/keys"
	"github.com/madicen/aitinerary/internal/tui/mouse"
	"github.com/madicen/aitinerary/internal/tui/tabs/board"
	"github.com/madicen/aitinerary/internal/tui/styles"
	"github.com/madicen/aitinerary/internal/theme"
)

func (m *Model) View() string {
	if m.width == 0 || m.height == 0 {
		return "loading…"
	}
	header := m.renderHeader()
	body := m.renderBody()
	status := m.renderStatus()
	main := lipgloss.JoinVertical(lipgloss.Left, header, body, status)
	out := m.overlayStack.View(main, m.width, m.height)
	return zone.Scan(out)
}

func (m *Model) headerHeight() int {
	return lipgloss.Height(m.renderHeader())
}

func (m *Model) chromeBodyHeight() int {
	h := m.height - m.headerHeight() - lipgloss.Height(m.renderStatus())
	if h < 1 {
		return 1
	}
	return h
}

func (m *Model) renderHeader() string {
	tabs := m.renderTabBar()
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.Adaptive(theme.RoleOnAccent)).
		Render("aitinerary")
	row := lipgloss.JoinHorizontal(lipgloss.Top, title, "  ", tabs)
	// Background/padding only — wrapping marked tabs in a foreground style breaks
	// bubblezone markers for every tab after the first.
	return lipgloss.NewStyle().
		Width(m.width).
		Background(theme.Adaptive(theme.RoleAccent)).
		Padding(0, 1).
		Render(row)
}

type tabSpec struct {
	mode  mode
	label string
	zone  string
}

func (m *Model) tabSpecs() []tabSpec {
	return []tabSpec{
		{modeBoard, "Board", mouse.ZoneTabBoard},
		{modeQueue, "Queue", mouse.ZoneTabQueue},
		{modeEpics, "Epics", mouse.ZoneTabEpics},
		{modeSettings, "Settings", mouse.ZoneTabSettings},
		{modeHelp, "Help", mouse.ZoneTabHelp},
	}
}

func (m *Model) renderTabBar() string {
	parts := make([]string, 0, 5)
	for _, t := range m.tabSpecs() {
		label := t.label
		if m.mode == t.mode {
			label = styles.TabActive.Render(label)
		} else {
			label = styles.TabInactive.Render(label)
		}
		parts = append(parts, zone.Mark(t.zone, label))
	}
	return strings.Join(parts, "  ")
}

func (m *Model) renderBody() string {
	tab := m.tabs[m.mode]
	if tab == nil {
		return styles.AppPadding.Render("tab unavailable")
	}
	body := tab.View()
	if m.store != nil && m.store.Refreshing() && m.mode == modeBoard {
		spin := styles.AppPadding.Render(m.spinner.View() + " refreshing tickets…")
		return lipgloss.JoinVertical(lipgloss.Left, body, spin)
	}
	return body
}

type statusSeg struct {
	text string
	zone string
}

func joinStatusSegs(segs []statusSeg) string {
	parts := make([]string, 0, len(segs))
	for _, s := range segs {
		if s.text == "" {
			continue
		}
		if s.zone != "" {
			parts = append(parts, zone.Mark(s.zone, s.text))
		} else {
			parts = append(parts, s.text)
		}
	}
	return strings.Join(parts, " · ")
}

func (m *Model) statusSegs() []statusSeg {
	km := m.keys
	seg := func(b key.Binding, z string) statusSeg {
		return statusSeg{text: keys.SegText(b), zone: z}
	}
	modeLabel := m.mode.String()
	ws := m.status.workspace
	if ws == "" {
		ws = "All"
	}
	qc := m.queueCount()
	queueHint := "queue 0"
	if qc > 0 {
		queueHint = styles.BoldStyle.Render("queue ") + styles.ErrStyle.Render(fmt.Sprintf("%d", qc))
	}
	profile := "AI " + m.activeProfileLabel()

	attention := ""
	if m.mode == modeBoard {
		if tab, ok := m.tabs[modeBoard].(*tabAdapter); ok {
			if b, ok := tab.inner.(*board.Model); ok {
				attention = b.StatusText()
			}
		}
	}

	segs := []statusSeg{
		{text: modeLabel},
		{text: "ws " + ws},
		{text: "fresh " + m.freshnessLabel()},
		{text: queueHint, zone: mouse.ZoneStatusQueue},
		{text: profile},
		seg(km.Refresh, mouse.ZoneStatusRefresh),
		seg(km.Undo, mouse.ZoneStatusUndo),
		seg(km.Palette, mouse.ZoneStatusPalette),
		seg(km.Help, mouse.ZoneStatusHelp),
		seg(km.Quit, mouse.ZoneStatusQuit),
	}
	if attention != "" {
		segs = append([]statusSeg{{text: attention}}, segs...)
	}
	return segs
}

func (m *Model) renderStatus() string {
	hint := joinStatusSegs(m.statusSegs())
	return styles.StatusBar.Width(m.width).Render(hint)
}
