package board

import (
	"bytes"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"

	"github.com/madicen/aitinerary/internal/actions"
	"github.com/madicen/aitinerary/internal/config"
	"github.com/madicen/aitinerary/internal/store"
	"github.com/madicen/aitinerary/internal/tui/keys"
	"github.com/madicen/aitinerary/internal/tui/tuitest"
)

func TestFlowSweepQueuesActions(t *testing.T) {
	tuitest.ForceMonochrome(t)
	cfg := config.Default()
	st := store.New(cfg, nil, true)
	q := actions.NewQueue()
	m := New(st, q, cfg, keys.Default())
	m.Resize(100, 30)
	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(100, 30))

	st.RefreshCmd()()
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("S")})
	waitBoard(t, tm, "sweep queued")

	if q.Count() == 0 {
		t.Fatal("sweep should queue comment actions for live tickets")
	}

	if err := tm.Quit(); err != nil {
		t.Fatalf("quit: %v", err)
	}
	tm.WaitFinished(t, teatest.WithFinalTimeout(5*time.Second))
}

func waitBoard(t *testing.T, tm *teatest.TestModel, sub string) {
	t.Helper()
	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		return bytes.Contains(b, []byte(sub))
	}, teatest.WithDuration(5*time.Second))
}
