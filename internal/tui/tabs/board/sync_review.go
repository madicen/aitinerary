package board

import (
	"context"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	overlay "github.com/madicen/bubble-overlay"

	"github.com/madicen/aitinerary/internal/actions"
	"github.com/madicen/aitinerary/internal/jira"
	"github.com/madicen/aitinerary/internal/theme"
	"github.com/madicen/aitinerary/internal/tui/overlays"
	"github.com/madicen/aitinerary/internal/tui/styles"
	"github.com/madicen/aitinerary/internal/tui/util"
)

type syncReviewState struct {
	open    bool
	index   int
	actions []actions.Action
	edit    bool
	draft   string
}

func (m *Model) commentActions() []actions.Action {
	if m.queue == nil {
		return nil
	}
	var out []actions.Action
	for _, a := range m.queue.Pending() {
		if a.Kind == actions.KindAddComment {
			out = append(out, a)
		}
	}
	return out
}

func (m *Model) openSyncReview() {
	acts := m.commentActions()
	if len(acts) == 0 {
		return
	}
	m.syncReview = &syncReviewState{
		open:    true,
		index:   0,
		actions: acts,
		draft:   acts[0].Comment,
	}
}

func (m *Model) handleSyncReviewKey(msg tea.KeyMsg) (*Model, tea.Cmd) {
	sr := m.syncReview
	if sr == nil || !sr.open {
		return m, nil
	}
	switch {
	case key.Matches(msg, m.keys.Back):
		m.syncReview = nil
	case key.Matches(msg, m.keys.QueueEdit):
		sr.edit = !sr.edit
	case key.Matches(msg, m.keys.QueueApprove):
		return m, m.approveSyncReview()
	case key.Matches(msg, m.keys.QueueReject):
		if m.queue != nil {
			m.queue.Remove(sr.current().ID)
		}
		m.advanceSyncReview()
	default:
		if sr.edit {
			sr.draft += msg.String()
		}
	}
	return m, nil
}

func (sr *syncReviewState) current() actions.Action {
	if sr == nil || sr.index >= len(sr.actions) {
		return actions.Action{}
	}
	return sr.actions[sr.index]
}

func (m *Model) approveSyncReview() tea.Cmd {
	sr := m.syncReview
	if sr == nil || m.engine == nil {
		return nil
	}
	a := sr.current()
	a.Comment = sr.draft
	_, err := m.engine.Apply(context.Background(), a)
	if m.queue != nil {
		m.queue.Remove(a.ID)
	}
	if err != nil {
		m.errText = err.Error()
		return nil
	}
	m.advanceSyncReview()
	return refreshStoreCmd(m.store)
}

func (m *Model) advanceSyncReview() {
	sr := m.syncReview
	if sr == nil {
		return
	}
	acts := m.commentActions()
	if len(acts) == 0 {
		m.syncReview = nil
		m.statusText = "Sync-prep review complete"
		return
	}
	sr.actions = acts
	sr.index = 0
	sr.draft = acts[0].Comment
	sr.edit = false
}

func (m *Model) renderSyncReview(base string) string {
	sr := m.syncReview
	if sr == nil || !sr.open {
		return base
	}
	a := sr.current()
	var t jira.Ticket
	for _, tk := range m.allTickets {
		if tk.Key == a.TicketKey {
			t = tk
			break
		}
	}
	left := styles.BoldStyle.Render(t.Key+" · "+t.Summary) + "\n" +
		styles.DimStyle.Render("Status: "+t.Status)
	right := sr.draft
	if sr.edit {
		right = lipgloss.NewStyle().Foreground(theme.Adaptive(theme.RoleAccent)).Render("▸ ") + right + "_"
	}
	panel := overlays.ModalFrameSized(util.Clamp(m.width-6, 50, 100)).Render(
		styles.BoldStyle.Render("Sync-prep review") + "\n\n" +
			lipgloss.JoinHorizontal(lipgloss.Top, leftStyle(m.width/2-4, left), " │ ", rightStyle(m.width/2-4, right)) +
			"\n\n" + styles.DimStyle.Render("a approve · e edit · x skip · esc close"))
	top := max(1, (m.height-lipgloss.Height(panel))/2)
	leftPad := max(2, (m.width-lipgloss.Width(panel))/2)
	return overlay.OverlayView(base, panel, m.width, m.height, top, leftPad)
}

func leftStyle(w int, s string) string {
	return lipgloss.NewStyle().Width(w).Render(s)
}

func rightStyle(w int, s string) string {
	return lipgloss.NewStyle().Width(w).Render(strings.TrimSpace(s))
}

func (m *Model) renderAutoAppliedStrip(base string) string {
	if m.autoApplied == nil {
		return base
	}
	recent := m.autoApplied.Recent()
	if len(recent) == 0 {
		return base
	}
	e := recent[len(recent)-1]
	strip := styles.OkStyle.Render("Auto-applied: "+e.Action.Describe()) +
		"  " + styles.DimStyle.Render("z undo")
	return strip + "\n" + base
}

func (m *Model) undoLastAutoApplied() tea.Cmd {
	if m.autoApplied == nil || m.engine == nil {
		return nil
	}
	entry, ok := m.autoApplied.PopLast()
	if !ok {
		return nil
	}
	if err := m.engine.Undo(context.Background(), entry); err != nil {
		m.errText = err.Error()
		return nil
	}
	m.statusText = "Undid auto-applied action"
	return refreshStoreCmd(m.store)
}
