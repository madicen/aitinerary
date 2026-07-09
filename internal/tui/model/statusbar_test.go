package model

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/madicen/aitinerary/internal/actions"
	"github.com/madicen/aitinerary/internal/tui/tuitest"
)

func TestStatusBarQueueBadge(t *testing.T) {
	tuitest.ForceMonochrome(t)
	m := New(Options{Demo: true})
	m.queue.Add(actions.Action{ID: "a1", Kind: actions.KindAddComment, TicketKey: "APP-1"})
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 30})
	view := m.View()
	if !strings.Contains(view, "queue") || !strings.Contains(view, "1") {
		t.Fatalf("status bar should show queue badge, got:\n%s", view)
	}
}
