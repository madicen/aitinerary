package epics

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"

	"github.com/madicen/aitinerary/internal/theme"
	"github.com/madicen/aitinerary/internal/tui/mouse"
	"github.com/madicen/aitinerary/internal/tui/styles"
	"github.com/madicen/aitinerary/internal/tui/util"
)

const timelineHeight = 5

// View renders the epics list and read-only timeline strip.
func (m *Model) View() string {
	m.rebuild()
	if m.width == 0 {
		m.width = 100
	}
	w := m.width

	title := styles.BoldStyle.Render("Epics")
	list := m.renderList(w)
	timeline := m.renderTimeline(w)
	v := lipgloss.JoinVertical(lipgloss.Left, title, "", list, "", timeline)
	v = m.renderDateEditOverlay(v)
	v = m.renderAssignOverlay(v)
	return v
}

func (m *Model) renderList(w int) string {
	if len(m.rows) == 0 {
		return styles.DimStyle.Render("  (no epics)")
	}
	listH := max(1, m.height-m.originY-timelineHeight-6)
	start, end := m.nav.VisibleRange(len(m.rows), listH)

	var lines []string
	curProject := ""
	for i, row := range m.rows {
		if row.Project != curProject {
			curProject = row.Project
			lines = append(lines, styles.BoldStyle.Foreground(theme.Adaptive(theme.RoleAccent)).
				Render("▸ "+curProject))
		}
		if i < start || i >= end {
			continue
		}
		style := lipgloss.NewStyle()
		if i == m.selected {
			style = style.Background(theme.Adaptive(theme.RoleSurface)).Bold(true)
		}
		dates := row.Epic.StartDate
		if row.Epic.DueDate != "" {
			if dates != "" {
				dates += " → "
			}
			dates += row.Epic.DueDate
		}
		if dates == "" {
			dates = "no dates"
		}
		line := fmt.Sprintf("  %s  %s  children %s  %s",
			row.Epic.Key, row.Epic.Summary, FormatProgress(row), dates)
		lines = append(lines, m.zm.Mark(mouse.ZoneEpicsRow(i), style.Render(line)))
	}
	return strings.Join(lines, "\n")
}

func (m *Model) renderTimeline(w int) string {
	innerW := util.Clamp(w-4, 40, 200)
	startMonth := monthStart(m.now).AddDate(0, m.timelineOff, 0)
	months := 6
	colW := innerW / months
	if colW < 8 {
		colW = 8
	}

	headerCells := make([]string, months)
	for i := 0; i < months; i++ {
		m := startMonth.AddDate(0, i, 0)
		headerCells[i] = lipgloss.NewStyle().Width(colW).Render(m.Format("Jan 2006"))
	}
	header := strings.Join(headerCells, "")

	rangeStart := startMonth
	rangeEnd := startMonth.AddDate(0, months, 0)
	totalDays := int(rangeEnd.Sub(rangeStart).Hours() / 24)

	barLines := make([]string, 0, len(m.rows)+2)
	gutter := styles.DimStyle.Render("no dates:")
	undated := []string{}
	for i, row := range m.rows {
		if row.Undated {
			undated = append(undated, row.Epic.Key)
			continue
		}
		bar := renderBar(row, rangeStart, totalDays, innerW, i, m.zm)
		barLines = append(barLines, bar)
	}
	if len(undated) > 0 {
		barLines = append(barLines, gutter+" "+strings.Join(undated, ", "))
	}

	todayX := dayOffset(rangeStart, m.now, totalDays, innerW)
	todayMark := strings.Repeat(" ", todayX) + styles.BoldStyle.Render("│")

	body := strings.Join(barLines, "\n")
	if body == "" {
		body = styles.DimStyle.Render("  (no dated epics)")
	}

	frame := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, false, true, false).
		BorderForeground(theme.Adaptive(theme.RoleMuted)).
		Width(innerW).
		Render(header + "\n" + body + "\n" + todayMark)

	return m.zm.Mark(mouse.ZoneEpicsTimeline, frame)
}

func monthStart(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
}

func dayOffset(rangeStart time.Time, day time.Time, totalDays, width int) int {
	if totalDays <= 0 {
		return 0
	}
	off := int(day.Sub(rangeStart).Hours() / 24)
	if off < 0 {
		off = 0
	}
	if off > totalDays {
		off = totalDays
	}
	return off * width / totalDays
}

func renderBar(row EpicRow, rangeStart time.Time, totalDays, width, idx int, zm *zone.Manager) string {
	start, err1 := time.Parse("2006-01-02", row.Epic.StartDate)
	end, err2 := time.Parse("2006-01-02", row.Epic.DueDate)
	if err1 != nil || err2 != nil {
		return ""
	}
	x0 := dayOffset(rangeStart, start, totalDays, width)
	x1 := dayOffset(rangeStart, end, totalDays, width)
	if x1 < x0 {
		x1 = x0
	}
	barW := x1 - x0
	if barW < 1 {
		barW = 1
	}
	pad := strings.Repeat(" ", x0)
	bar := lipgloss.NewStyle().
		Foreground(theme.Adaptive(theme.RoleOnSurface)).
		Background(theme.Adaptive(theme.RoleAccent)).
		Render(strings.Repeat(" ", barW))
	label := styles.DimStyle.Render(row.Epic.Key)
	line := pad + bar + " " + label
	return zm.Mark(mouse.ZoneEpicsTimelineBar(idx), line)
}
