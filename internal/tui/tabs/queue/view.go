package queue

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"

	"github.com/madicen/aitinerary/internal/actions"
	"github.com/madicen/aitinerary/internal/theme"
	"github.com/madicen/aitinerary/internal/tui/keys"
	"github.com/madicen/aitinerary/internal/tui/mouse"
	"github.com/madicen/aitinerary/internal/tui/overlays"
	"github.com/madicen/aitinerary/internal/tui/styles"
)

func (m *Model) View() string {
	m.rebuildFlat()
	if m.width == 0 {
		m.width = 80
	}
	if m.height == 0 {
		m.height = 24
	}

	title := styles.BoldStyle.Render("Review queue")
	count := fmt.Sprintf("(%d pending)", m.queue.Count())
	header := lipgloss.JoinHorizontal(lipgloss.Left,
		title, " ", styles.DimStyle.Render(count), "  ",
		m.zm.Mark(mouse.ZoneQueueBulkApprove, styles.OkStyle.Render("[A] approve all")),
	)

	hints := styles.DimStyle.Render(keys.SegText(m.keys.QueueApprove) + " · " +
		keys.SegText(m.keys.QueueEdit) + " · " + keys.SegText(m.keys.QueueReject))

	body := m.renderList()
	if m.editID != "" {
		body = m.renderEditOverlay(body)
	}
	if m.confirm {
		body = overlays.ModalFrameSized(m.width/2).Render(m.confirmMsg+"\n\n"+
			zone.Mark(mouse.ConfirmYes, styles.OkStyle.Render(" Yes "))+"  "+
			zone.Mark(mouse.ConfirmNo, styles.ErrStyle.Render(" No "))) + "\n\n" + body
	}

	status := m.status
	if status == "" {
		status = "select an action"
	}
	footer := styles.DimStyle.Render(status)

	return lipgloss.JoinVertical(lipgloss.Left, header, hints, "", body, "", footer)
}

func (m *Model) renderList() string {
	sel := selectable(m.flat)
	if len(sel) == 0 {
		return styles.DimStyle.Render("  (queue empty)")
	}
	listH := max(1, m.height-m.originY-8)
	start, end := m.nav.VisibleRange(len(sel), listH)

	var lines []string
	selIdx := 0
	for _, it := range m.flat {
		if it.header {
			lines = append(lines, styles.BoldStyle.Foreground(theme.Adaptive(theme.RoleAccent)).
				Render("▸ "+actions.KindLabel(it.kind)))
			continue
		}
		rowStyle := lipgloss.NewStyle()
		if selIdx == m.selected {
			rowStyle = rowStyle.Background(theme.Adaptive(theme.RoleSurface)).Bold(true)
		}
		desc := it.action.Describe()
		if it.action.Source == actions.SourceAI {
			desc = "✦ " + desc
		}
		controls := lipgloss.JoinHorizontal(lipgloss.Left,
			m.zm.Mark(mouse.ZoneQueueApprove(selIdx), styles.OkStyle.Render(" a ")),
			m.zm.Mark(mouse.ZoneQueueEdit(selIdx), styles.DimStyle.Render(" e ")),
			m.zm.Mark(mouse.ZoneQueueReject(selIdx), styles.ErrStyle.Render(" x ")),
		)
		line := rowStyle.Render(fmt.Sprintf("  %s  %s", desc, controls))
		line = m.zm.Mark(mouse.ZoneQueueRow(selIdx), line)
		if selIdx >= start && selIdx < end {
			lines = append(lines, line)
		}
		selIdx++
	}
	if len(lines) == 0 {
		return styles.DimStyle.Render("  …")
	}
	return strings.Join(lines, "\n")
}

func (m *Model) renderEditOverlay(body string) string {
	prompt := styles.BoldStyle.Render("Edit pending action")
	field := "> " + m.editText + "_"
	return lipgloss.JoinVertical(lipgloss.Left,
		prompt, field, styles.DimStyle.Render("enter save · esc cancel"), "", body)
}
