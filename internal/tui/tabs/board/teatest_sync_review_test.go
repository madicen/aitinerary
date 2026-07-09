package board

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/madicen/aitinerary/internal/actions"
)

func seedCommentQueue(q *actions.Queue) {
	q.Add(actions.Action{
		ID: "c1", Kind: actions.KindAddComment, TicketKey: "APP-101",
		Source: actions.SourceAI, Comment: "APP-101 progress update.",
	})
	q.Add(actions.Action{
		ID: "c2", Kind: actions.KindAddComment, TicketKey: "APP-102",
		Source: actions.SourceAI, Comment: "APP-102 progress update.",
	})
}

func TestTeatestSyncReviewOpensAndSkips(t *testing.T) {
	m := demoBoardModel(100, 30)
	q := actions.NewQueue()
	seedCommentQueue(q)
	m.queue = q

	m.openSyncReview()
	if m.syncReview == nil || !m.syncReview.open {
		t.Fatal("sync review should open")
	}
	view := m.View()
	if !strings.Contains(view, "Sync-prep review") {
		t.Fatalf("review overlay missing:\n%s", view)
	}

	upd, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	m = upd.(*Model)
	if len(q.Pending()) != 1 {
		t.Fatalf("skip should leave 1 comment, got %d", len(q.Pending()))
	}

	upd, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	m = upd.(*Model)
	if m.syncReview != nil && m.syncReview.open {
		t.Fatal("review should close after last skip")
	}
}

func TestTeatestSyncReviewOpensViaKey(t *testing.T) {
	m := demoBoardModel(100, 30)
	q := actions.NewQueue()
	seedCommentQueue(q)
	m.queue = q

	upd, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("R")})
	m = upd.(*Model)
	if m.syncReview == nil || !m.syncReview.open {
		t.Fatal("R should open sync review when comments are queued")
	}
}
