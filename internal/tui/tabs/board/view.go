package board

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/madicen/aitinerary/internal/hygiene"
	"github.com/madicen/aitinerary/internal/theme"
	"github.com/madicen/aitinerary/internal/tui/mouse"
	"github.com/madicen/aitinerary/internal/tui/styles"
)

func (m *Model) renderBoard() string {
	var header []string
	if m.detailOpen {
		header = append(header, m.renderDetail())
		header = append(header, strings.Repeat("─", m.width))
	}
	header = append(header, m.renderFilterChips())
	if m.filters.SearchQuery != "" {
		header = append(header, lipgloss.NewStyle().Foreground(theme.Adaptive(theme.RoleInfo)).
			Render("Search: "+m.filters.SearchQuery+"  (Esc to clear)"))
	}
	if m.errText != "" {
		header = append(header, styles.ErrStyle.Render(m.errText))
	}
	if m.sweepMsg != "" {
		header = append(header, styles.DimStyle.Render(m.sweepMsg))
	}
	if m.sweepMsg != "" {
		header = append(header, styles.OkStyle.Render(m.sweepMsg))
	}
	header = append(header, styles.DimStyle.Render("space select · v range · E epic · j/k nav · / search"))

	fixed := strings.Join(header, "\n")
	headerLines := strings.Count(fixed, "\n") + 1
	listHeight := m.height - headerLines
	if listHeight < 1 {
		listHeight = 1
	}

	ticketRows := m.ticketRowIndices()
	total := len(m.rows)
	if m.scrollToSelected {
		m.scrollToSelected = false
		sel := m.selected
		if sel < 0 {
			sel = 0
		}
		m.ScrollToSelected(sel, total, listHeight)
	}
	start, end := m.VisibleRange(total, listHeight)

	var listLines []string
	ticketVis := 0
	for i := start; i < end; i++ {
		r := m.rows[i]
		if r.Kind == RowGroup {
			listLines = append(listLines, m.renderGroupRow(r))
			continue
		}
		listLines = append(listLines, m.renderTicketRow(ticketVis, i, r))
		ticketVis++
	}
	_ = ticketRows
	if len(listLines) == 0 {
		listLines = []string{styles.DimStyle.Render("  No tickets match the current filters.")}
	}
	return fixed + "\n" + strings.Join(listLines, "\n")
}

func (m *Model) selectedVisibleIndex() int {
	ticketRows := m.ticketRowIndices()
	for i, idx := range ticketRows {
		if idx == m.selected {
			return i
		}
	}
	return 0
}

func (m *Model) renderFilterChips() string {
	var chips []string
	chips = append(chips, m.renderChip("All", mouse.ZoneFilterAll, m.filters.Preset == FilterAll))
	chips = append(chips, m.renderChip("Needs attention", mouse.ZoneFilterNeedsAttention, m.filters.Preset == FilterNeedsAttention))
	chips = append(chips, m.renderChip("Done candidates", mouse.ZoneFilterDoneCandidates, m.filters.Preset == FilterDoneCandidates))
	for _, ws := range m.chips {
		active := strings.EqualFold(m.filters.Workspace, ws.Chip)
		chips = append(chips, m.renderChip(ws.Chip, mouse.ZoneFilterWorkspace(ws.Chip), active))
	}
	for _, cat := range []struct{ label, key string }{
		{"To Do", "new"},
		{"In flight", "indeterminate"},
		{"Done", "done"},
	} {
		active := strings.EqualFold(m.filters.StatusCategory, cat.key)
		chips = append(chips, m.renderChip(cat.label, mouse.ZoneFilterStatus(cat.key), active))
	}
	return strings.Join(chips, " ")
}

func (m *Model) renderChip(label, zoneID string, active bool) string {
	style := lipgloss.NewStyle().Padding(0, 1)
	if active {
		style = style.Foreground(theme.Adaptive(theme.RoleOnAccent)).Background(theme.Adaptive(theme.RoleAccent)).Bold(true)
	} else {
		style = style.Foreground(theme.Adaptive(theme.RoleMuted))
	}
	return m.mark(zoneID, style.Render(label))
}

func (m *Model) renderGroupRow(r Row) string {
	label := findingTypeLabel(r.Group)
	style := lipgloss.NewStyle().Bold(true).Foreground(theme.Adaptive(theme.RoleWarning))
	return "  " + style.Render("── "+label+" ──")
}

func (m *Model) renderTicketRow(visibleIdx, rowIdx int, r Row) string {
	td := r.Data
	t := td.Ticket
	selected := rowIdx == m.selected
	lineStyle := lipgloss.NewStyle()
	if selected {
		lineStyle = lineStyle.Foreground(theme.Adaptive(theme.RoleSelectionFg)).Background(theme.Adaptive(theme.RoleSelectionBg))
	}

	chipStyle := lipgloss.NewStyle().Foreground(theme.Adaptive(theme.RoleInfo)).Bold(true)
	keyStyle := lipgloss.NewStyle().Foreground(theme.Adaptive(theme.RoleAccent)).Bold(true)
	statusStyle := lipgloss.NewStyle().Foreground(theme.Adaptive(theme.RoleMuted))

	parts := []string{
		chipStyle.Render(fmt.Sprintf("[%s]", td.Workspace)),
		statusStyle.Render(truncate(t.Status, 12)),
		keyStyle.Render(t.Key),
		truncate(t.Summary, m.width-40),
	}
	if m.isMultiSelected(t.Key) {
		parts = append([]string{styles.OkStyle.Render("✓")}, parts...)
	}
	parts = append(parts, m.renderBadges(td)...)
	line := strings.Join(parts, " ")
	if selected {
		line = "▸ " + line
	} else {
		line = "  " + line
	}
	return m.mark(mouse.ZoneBoardRow(visibleIdx), lineStyle.Render(line))
}

func (m *Model) renderBadges(td ticketData) []string {
	var badges []string
	has := map[hygiene.FindingType]bool{}
	for _, f := range td.Findings {
		has[f.Type] = true
	}
	warn := lipgloss.NewStyle().Foreground(theme.Adaptive(theme.RoleWarning))
	high := lipgloss.NewStyle().Foreground(theme.Adaptive(theme.RoleError))
	ok := lipgloss.NewStyle().Foreground(theme.Adaptive(theme.RoleSuccess))

	if has[hygiene.MissingEpic] {
		badges = append(badges, warn.Render("⬡E"))
	}
	if has[hygiene.MissingSprint] {
		badges = append(badges, warn.Render("⬡S"))
	}
	if has[hygiene.MissingDueDate] {
		badges = append(badges, warn.Render("⬡D"))
	}
	if has[hygiene.StaleInProgress] {
		badges = append(badges, high.Render("⏱"))
	}
	if has[hygiene.MergedNotDone] {
		badges = append(badges, ok.Render("PR✓"))
	}
	if has[hygiene.EpicMissingDates] {
		badges = append(badges, warn.Render("⬡📅"))
	}
	return badges
}

func (m *Model) renderSearchOverlay(base string) string {
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.Adaptive(theme.RoleBorderAccent)).
		Padding(0, 1).
		Width(m.width - 4).
		Render(m.search.input.View())
	lines := strings.Split(base, "\n")
	if len(lines) > 2 {
		lines[2] = box
		return strings.Join(lines, "\n")
	}
	return base + "\n" + box
}

func (m *Model) mark(id, s string) string {
	if m.zone == nil {
		return s
	}
	return m.zone.Mark(id, s)
}

func truncate(s string, max int) string {
	if max < 4 {
		return s
	}
	if ansi.StringWidth(s) <= max {
		return s
	}
	return ansi.Truncate(s, max-1, "…")
}
