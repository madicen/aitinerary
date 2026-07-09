package queue

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/madicen/aitinerary/internal/tui/tuitest"
)

// TestFlowMinimalRender sanity-checks the queue tab renders fixture data.
func TestFlowMinimalRender(t *testing.T) {
	m, _ := testModel(t, demoPending()...)
	if m.View() == "" {
		t.Fatal("empty view")
	}
}

// TestFlowApproveBulkUndoKeyboard drives approve → reject → bulk → undo via
// keyboard messages (hermetic alternative to teatest when alt-screen capture
// is empty in this environment).
func TestFlowApproveBulkUndoKeyboard(t *testing.T) {
	tuitest.ForceMonochrome(t)
	m, q := testModel(t, demoPending()...)

	send := func(key string) {
		t.Helper()
		updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
		if mm, ok := updated.(*Model); ok {
			m = mm
		}
		if cmd != nil {
			if msg := cmd(); msg != nil {
				updated, _ = m.Update(msg)
				if mm, ok := updated.(*Model); ok {
					m = mm
				}
			}
		}
	}

	send("a")
	if q.Count() != 3 {
		t.Fatalf("after approve: pending=%d", q.Count())
	}
	send("x")
	if q.Count() != 2 {
		t.Fatalf("after reject: pending=%d", q.Count())
	}
	send("A")
	if !m.confirm {
		t.Fatal("bulk confirm not shown")
	}
	send("y")
	if q.Count() != 0 {
		t.Fatalf("after bulk: pending=%d", q.Count())
	}
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlZ})
	if cmd != nil {
		if msg := cmd(); msg != nil {
			m.Update(msg)
		}
	}
	if m.status == "" || m.status == "select an action" {
		t.Fatalf("expected undo status, got %q", m.status)
	}
}
