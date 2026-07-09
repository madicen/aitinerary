package board

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/madicen/aitinerary/internal/actions"
)

func TestTeatestCreateTicketQueuesIREAction(t *testing.T) {
	m := demoBoardModel(100, 30)
	q := actions.NewQueue()
	m.queue = q

	upd, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	m = upd.(*Model)
	if m.createForm == nil || !m.createForm.open {
		t.Fatal("create form should open on n")
	}
	m.createForm.title.SetValue("IRE dashboard timeout")
	upd, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = upd.(*Model)

	pending := q.Pending()
	if len(pending) != 1 {
		t.Fatalf("queue len = %d, want 1", len(pending))
	}
	a := pending[0]
	if a.Kind != actions.KindCreateTicket {
		t.Fatalf("kind = %q", a.Kind)
	}
	if a.Ticket == nil || len(a.Ticket.Labels) == 0 || a.Ticket.Labels[0] != "Apps" {
		t.Fatalf("ticket payload = %+v", a.Ticket)
	}
	if a.Ticket.Project != "HELP" {
		t.Fatalf("project = %q", a.Ticket.Project)
	}
}
