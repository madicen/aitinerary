package board

import (
	"testing"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/madicen/aitinerary/internal/actions"
	"github.com/madicen/aitinerary/internal/config"
	"github.com/madicen/aitinerary/internal/store"
	"github.com/madicen/aitinerary/internal/tui/keys"
	"github.com/madicen/aitinerary/internal/tui/tuitest"
)

func TestFlowEpicBuilderQueuesActions(t *testing.T) {
	tuitest.ForceMonochrome(t)
	cfg := config.Default()
	st := store.New(cfg, nil, true)
	q := actions.NewQueue()
	km := keys.Default()
	m := New(st, q, cfg, km)
	m.Resize(100, 30)
	m.filters.Preset = FilterAll
	if msg := st.RefreshCmd()(); msg != nil {
		m.Update(msg)
	} else {
		m.loadData(nil)
	}

	if !key.Matches(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("E")}, km.BoardCreateEpic) {
		t.Fatal("E should match BoardCreateEpic binding")
	}

	for i, r := range m.rows {
		if r.Kind == RowTicket && r.Data.Ticket.Key == "APP-103" {
			m.selected = i
			break
		}
	}
	if m.selectedKey() != "APP-103" {
		t.Fatalf("selected = %q, want APP-103", m.selectedKey())
	}
	m.toggleMultiSelect()
	if len(m.selectedTicketKeys()) == 0 {
		t.Fatal("expected selected ticket keys after toggle")
	}

	updated, cmd := m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("E")})
	m = updated
	if cmd != nil {
		if msg := cmd(); msg != nil {
			m.Update(msg)
		}
	}
	if m.epicOverlay == nil {
		t.Fatalf("expected epic overlay open, status=%q", m.statusText)
	}

	updated, cmd = m.handleEpicOverlayKey(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated
	if cmd != nil {
		cmd()
	}

	if q.Count() < 2 {
		t.Fatalf("expected epic + set-parent actions queued, got %d", q.Count())
	}
}
