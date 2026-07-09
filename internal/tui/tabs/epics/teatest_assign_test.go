package epics

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/madicen/aitinerary/internal/actions"
)

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

func selectEpic(m *Model, key string) {
	for i, row := range m.rows {
		if row.Epic.Key == key {
			m.selected = i
			return
		}
	}
}

func TestFlowAssignUnparentedQueuesSetParent(t *testing.T) {
	m, q, _ := demoEpicsModel(100, 30)
	m.rebuild()
	selectEpic(m, "APP-100")

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	m = updated.(*Model)
	if cmd != nil {
		m = runCmd(m, cmd)
	}
	if !m.AssignOverlayOpen() {
		t.Fatal("assign overlay should be open after a")
	}

	// APP-101 and APP-103 are unparented APP tickets in demo fixtures.
	for _, key := range []string{"APP-101", "APP-103"} {
		for m.assignOverlay.tickets[m.assignOverlay.selected].Key != key {
			upd, c := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
			m = upd.(*Model)
			if c != nil {
				m = runCmd(m, c)
			}
		}
		upd, c := m.Update(tea.KeyMsg{Type: tea.KeySpace})
		m = upd.(*Model)
		if c != nil {
			m = runCmd(m, c)
		}
	}

	upd, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = runCmd(upd.(*Model), cmd)

	if m.AssignOverlayOpen() {
		t.Fatal("assign overlay should close after enter")
	}

	pending := q.Pending()
	if len(pending) != 2 {
		t.Fatalf("queue len = %d, want 2 SetParent actions", len(pending))
	}
	wantKeys := map[string]bool{"APP-101": true, "APP-103": true}
	for _, a := range pending {
		if a.Kind != actions.KindSetParent {
			t.Fatalf("action kind = %q, want set_parent", a.Kind)
		}
		if a.ParentKey != "APP-100" {
			t.Fatalf("ParentKey = %q, want APP-100", a.ParentKey)
		}
		if !wantKeys[a.TicketKey] {
			t.Fatalf("unexpected ticket key %q in queue", a.TicketKey)
		}
	}
}

func TestFlowAssignEscClosesWithoutQueueing(t *testing.T) {
	m, q, _ := demoEpicsModel(100, 30)
	m.rebuild()
	selectEpic(m, "APP-100")

	upd, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	m = upd.(*Model)
	upd, _ = m.Update(tea.KeyMsg{Type: tea.KeySpace})
	m = upd.(*Model)
	upd, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = upd.(*Model)

	if m.AssignOverlayOpen() {
		t.Fatal("assign overlay should close on esc")
	}
	if len(q.Pending()) != 0 {
		t.Fatalf("queue len = %d, want 0 after esc", len(q.Pending()))
	}
}
