package board

import (
	"testing"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/madicen/aitinerary/internal/demo"
	"github.com/madicen/aitinerary/internal/tui/keys"
	"github.com/madicen/aitinerary/internal/tui/tuitest"
)

func selectTicket(m *Model, key string) {
	for i, r := range m.rows {
		if r.Kind == RowTicket && r.Data.Ticket.Key == key {
			m.selected = i
			return
		}
	}
}

func runCmd(m *Model, cmd tea.Cmd) *Model {
	if cmd == nil {
		return m
	}
	msg := cmd()
	if msg == nil {
		return m
	}
	updated, next := m.Update(msg)
	if um, ok := updated.(*Model); ok {
		m = um
	}
	if next != nil {
		return runCmd(m, next)
	}
	return m
}

func editTestModel(t *testing.T) (*Model, *fakeExecutor) {
	t.Helper()
	tuitest.ForceMonochrome(t)
	fx := &fakeExecutor{ticket: demo.Tickets()[2]} // APP-103 — missing sprint/due
	m := demoBoardModelWithExecutor(100, 36, fx)
	selectTicket(m, "APP-103")
	m.detailOpen = true
	return m, fx
}

func TestFlowEditDueDateWritesViaFakeEngine(t *testing.T) {
	m, fx := editTestModel(t)
	km := keys.Default()

	updated, cmd := m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	m = updated
	if cmd != nil {
		m = runCmd(m, cmd)
	}
	if m.editMode != editDueDate {
		t.Fatalf("editMode = %v, want due date", m.editMode)
	}

	m.dueInput.SetValue("2026-06-30")
	updated, cmd = m.applyDueDate("2026-06-30")
	m = runCmd(updated, cmd)

	if fx.dueDate != "2026-06-30" {
		t.Fatalf("SetDueDate = %q, want 2026-06-30", fx.dueDate)
	}
	if m.editMode != editNone {
		t.Fatalf("editMode after apply = %v, want none", m.editMode)
	}
	if !key.Matches(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")}, km.BoardEditDueDate) {
		t.Fatal("d should match BoardEditDueDate binding")
	}
}

func TestFlowEditSprintWritesViaFakeEngine(t *testing.T) {
	m, fx := editTestModel(t)

	updated, cmd := m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	m = runCmd(updated, cmd)
	if m.editMode != editSprint {
		t.Fatalf("editMode = %v, want sprint", m.editMode)
	}
	if len(m.sprintOptions) == 0 {
		t.Fatal("sprint options not loaded")
	}

	upd, cmd := m.Update(SprintSelectedMsg{Index: 0})
	if um, ok := upd.(*Model); ok {
		m = um
	}
	m = runCmd(m, cmd)

	if fx.sprintID != 9001 {
		t.Fatalf("AssignSprint id = %d, want 9001", fx.sprintID)
	}
	if m.editMode != editNone {
		t.Fatalf("editMode after apply = %v, want none", m.editMode)
	}
}
