package board

import (
	"fmt"
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

type createTicketState struct {
	open      bool
	issueType string
	title     textinput.Model
	body      textinput.Model
	field     int
}

func (m *Model) openCreateTicket() tea.Cmd {
	ti := textinput.New()
	ti.Placeholder = "Summary"
	ti.Focus()
	bi := textinput.New()
	bi.Placeholder = "Description (optional)"
	m.createForm = &createTicketState{
		open:      true,
		issueType: "Task",
		title:     ti,
		body:      bi,
	}
	return textinput.Blink
}

func (m *Model) handleCreateTicketKey(msg tea.KeyMsg) (*Model, tea.Cmd) {
	cf := m.createForm
	if cf == nil || !cf.open {
		return m, nil
	}
	switch {
	case key.Matches(msg, m.keys.Back):
		m.createForm = nil
	case key.Matches(msg, m.keys.ListOpen):
		return m, m.queueCreateTicket()
	case msg.String() == "tab":
		cf.field = (cf.field + 1) % 2
		if cf.field == 0 {
			cf.title.Focus()
		} else {
			cf.body.Focus()
		}
		return m, textinput.Blink
	default:
		var cmd tea.Cmd
		if cf.field == 0 {
			cf.title, cmd = cf.title.Update(msg)
		} else {
			cf.body, cmd = cf.body.Update(msg)
		}
		return m, cmd
	}
	return m, nil
}

func (m *Model) queueCreateTicket() tea.Cmd {
	cf := m.createForm
	if cf == nil || m.queue == nil {
		return nil
	}
	title := strings.TrimSpace(cf.title.Value())
	if title == "" {
		m.errText = "Title required"
		return nil
	}
	now := time.Now()
	a := actions.Action{
		ID:     actions.NewID(actions.KindCreateTicket, now),
		Kind:   actions.KindCreateTicket,
		Source: actions.SourceUser,
		Summary: fmt.Sprintf("create HELP ticket %q", title),
		Ticket: &actions.TicketPayload{
			Project:     "HELP",
			IssueType:   cf.issueType,
			Summary:     title,
			Description: strings.TrimSpace(cf.body.Value()),
			Labels:      []string{"Apps"},
		},
	}
	m.queue.Add(a)
	m.createForm = nil
	m.statusText = "Queued IRE ticket create"
	return nil
}

func (m *Model) renderCreateTicket(base string) string {
	cf := m.createForm
	if cf == nil || !cf.open {
		return base
	}
	lines := []string{
		styles.BoldStyle.Render("Quick capture (IRE)"),
		styles.DimStyle.Render("Project HELP · label Apps"),
		"Type: " + cf.issueType,
		"Title: " + cf.title.View(),
		"Body:  " + cf.body.View(),
		styles.DimStyle.Render("enter queue · esc cancel"),
	}
	modal := overlays.ModalFrameSized(util.Clamp(m.width-8, 44, 80)).Render(strings.Join(lines, "\n"))
	top := max(2, (m.height-lipgloss.Height(modal))/2)
	left := max(2, (m.width-lipgloss.Width(modal))/2)
	return overlay.OverlayView(base, modal, m.width, m.height, top, left)
}
