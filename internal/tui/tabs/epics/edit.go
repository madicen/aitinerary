package epics

import (
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	overlay "github.com/madicen/bubble-overlay"

	"github.com/madicen/aitinerary/internal/actions"
	"github.com/madicen/aitinerary/internal/tui/overlays"
	"github.com/madicen/aitinerary/internal/tui/styles"
	"github.com/madicen/aitinerary/internal/tui/util"
)

type editMode int

const (
	editNone editMode = iota
	editStartDate
	editDueDate
)

func (m *Model) initDateInputs() {
	m.startInput = textinput.New()
	m.startInput.Placeholder = "YYYY-MM-DD"
	m.startInput.CharLimit = 10

	m.dueInput = textinput.New()
	m.dueInput.Placeholder = "YYYY-MM-DD"
	m.dueInput.CharLimit = 10
}

func (m *Model) selectedEpic() (EpicRow, bool) {
	if m.selected < 0 || m.selected >= len(m.rows) {
		return EpicRow{}, false
	}
	return m.rows[m.selected], true
}

func (m *Model) beginEditStartDate() (*Model, tea.Cmd) {
	row, ok := m.selectedEpic()
	if !ok {
		return m, nil
	}
	m.editMode = editStartDate
	m.startInput.SetValue(row.Epic.StartDate)
	m.startInput.Focus()
	return m, textinput.Blink
}

func (m *Model) beginEditDueDate() (*Model, tea.Cmd) {
	row, ok := m.selectedEpic()
	if !ok {
		return m, nil
	}
	m.editMode = editDueDate
	m.dueInput.SetValue(row.Epic.DueDate)
	m.dueInput.Focus()
	return m, textinput.Blink
}

func (m *Model) closeEdits() {
	m.editMode = editNone
	m.startInput.Blur()
	m.dueInput.Blur()
}

func (m *Model) handleDateEditKey(msg tea.KeyMsg) (*Model, tea.Cmd) {
	var cmd tea.Cmd
	switch m.editMode {
	case editStartDate:
		m.startInput, cmd = m.startInput.Update(msg)
		if key.Matches(msg, m.keys.Back) {
			m.closeEdits()
			return m, nil
		}
		if key.Matches(msg, m.keys.ListOpen) {
			return m.applyStartDate(m.startInput.Value())
		}
	case editDueDate:
		m.dueInput, cmd = m.dueInput.Update(msg)
		if key.Matches(msg, m.keys.Back) {
			m.closeEdits()
			return m, nil
		}
		if key.Matches(msg, m.keys.ListOpen) {
			return m.applyDueDate(m.dueInput.Value())
		}
	}
	return m, cmd
}

func (m *Model) applyStartDate(iso string) (*Model, tea.Cmd) {
	row, ok := m.selectedEpic()
	if !ok || m.engine == nil {
		return m, nil
	}
	iso = strings.TrimSpace(iso)
	action := actions.Action{
		ID:        actions.NewID(actions.KindSetStartDate, time.Now()),
		Kind:      actions.KindSetStartDate,
		TicketKey: row.Epic.Key,
		Source:    actions.SourceUser,
		StartDate: iso,
		Summary:   row.Epic.Key + ": start " + iso,
	}
	return m, applyActionCmd(m.engine, action)
}

func (m *Model) applyDueDate(iso string) (*Model, tea.Cmd) {
	row, ok := m.selectedEpic()
	if !ok || m.engine == nil {
		return m, nil
	}
	iso = strings.TrimSpace(iso)
	action := actions.Action{
		ID:        actions.NewID(actions.KindSetDueDate, time.Now()),
		Kind:      actions.KindSetDueDate,
		TicketKey: row.Epic.Key,
		Source:    actions.SourceUser,
		DueDate:   iso,
		Summary:   row.Epic.Key + ": due " + iso,
	}
	return m, applyActionCmd(m.engine, action)
}

func (m *Model) renderDateEditOverlay(base string) string {
	if m.editMode == editNone {
		return base
	}
	row, ok := m.selectedEpic()
	if !ok {
		return base
	}
	label := "Start date"
	input := m.startInput.View()
	if m.editMode == editDueDate {
		label = "Due date"
		input = m.dueInput.View()
	}
	var lines []string
	lines = append(lines, styles.BoldStyle.Render(label+" for "+row.Epic.Key))
	lines = append(lines, input)
	lines = append(lines, styles.DimStyle.Render("enter apply · esc cancel"))
	modal := overlays.ModalFrameSized(util.Clamp(m.width-8, 40, 70)).Render(strings.Join(lines, "\n"))
	top := max(2, (m.height-lipgloss.Height(modal))/2)
	left := max(2, (m.width-lipgloss.Width(modal))/2)
	return overlay.OverlayView(base, modal, m.width, m.height, top, left)
}
