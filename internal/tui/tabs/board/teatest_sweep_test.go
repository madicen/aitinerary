package board

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/madicen/aitinerary/internal/actions"
	"github.com/madicen/aitinerary/internal/demo"
)

func runSweepSync(t *testing.T, m *Model) *Model {
	t.Helper()
	upd, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("S")})
	m = upd.(*Model)
	if cmd == nil {
		t.Fatal("expected sweep cmd")
	}
	msg := cmd()
	done, ok := msg.(SweepDoneMsg)
	if !ok {
		t.Fatalf("expected SweepDoneMsg, got %T", msg)
	}
	if done.Err != nil {
		t.Fatalf("sweep err: %v", done.Err)
	}
	if done.Cancelled {
		t.Fatal("sweep cancelled unexpectedly")
	}
	upd, _ = m.Update(done)
	return upd.(*Model)
}

func TestTeatestSweepQueuesCommentsTransitionsAndRank(t *testing.T) {
	m := demoBoardModel(100, 30)
	q := actions.NewQueue()
	m.queue = q
	m.filters.Preset = FilterAll

	m = runSweepSync(t, m)

	pending := q.Pending()
	if len(pending) == 0 {
		t.Fatal("sweep should queue actions")
	}

	var comments, transitions, ranks int
	for _, a := range pending {
		switch a.Kind {
		case actions.KindAddComment:
			comments++
		case actions.KindTransition:
			transitions++
		case actions.KindRank:
			ranks++
		}
	}
	if comments == 0 {
		t.Fatal("expected AI status comments")
	}
	if transitions == 0 {
		t.Fatal("expected merged-not-done transition proposals")
	}
	if ranks == 0 {
		t.Fatal("expected rank ordering actions")
	}
}

func TestBuildSweepActionsDemoFixtures(t *testing.T) {
	now := time.Date(2026, 6, 15, 10, 0, 0, 0, time.UTC)
	tickets := demo.AllTickets()
	matches := matchPRsForTickets(tickets, demo.MergedPRs(), []string{"APP", "HELP"})
	acts := buildSweepActions(tickets, matches, nil, "Done", now)
	if len(acts) == 0 {
		t.Fatal("expected sweep actions from demo fixtures")
	}
}
