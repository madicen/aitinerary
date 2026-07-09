package queue

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/madicen/aitinerary/internal/tui/tuitest"
)

func TestGoldenQueueGrouped(t *testing.T) {
	tuitest.ForceMonochrome(t)
	m, _ := testModel(t, demoPending()...)
	got := m.View()
	tuitest.AssertGolden(t, "queue_grouped", got)
}

func TestApproveRejectBulkUndoKeyboard(t *testing.T) {
	m, q := testModel(t, demoPending()...)
	if q.Count() != 4 {
		t.Fatalf("setup pending = %d", q.Count())
	}

	// approve first selectable (transition)
	m.selected = 0
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	if cmd != nil {
		msg := cmd()
		m.Update(msg)
	}
	if q.Count() != 3 {
		t.Fatalf("after approve expected 3 pending, got %d", q.Count())
	}

	// reject next
	m.selected = 0
	_, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	if cmd != nil {
		m.Update(cmd())
	}
	if q.Count() != 2 {
		t.Fatalf("after reject expected 2 pending, got %d", q.Count())
	}

	// bulk approve remainder
	m.confirm = false
	_, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("A")})
	if !m.confirm {
		t.Fatal("bulk approve should open confirm")
	}
	_, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	if cmd != nil {
		m.Update(cmd())
	}
	if q.Count() != 0 {
		t.Fatalf("after bulk approve expected empty queue, got %d", q.Count())
	}

	// undo last applied via Undo()
	cmd = m.Undo()
	if cmd != nil {
		m.Update(cmd())
	}
}
