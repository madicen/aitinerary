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

func TestTeatestSelectAndDetail(t *testing.T) {
	tuitest.ForceMonochrome(t)
	cfg := config.Default()
	st := store.New(cfg, nil, true)
	m := New(st, actions.NewQueue(), cfg, keys.Default())
	m.Resize(100, 30)
	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(100, 30))

	st.RefreshCmd()()
	tm.Send(store.StoreRefreshedMsg{})
	waitBoardContains(t, tm, "APP-101")

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	waitBoardContains(t, tm, "▸")

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	waitBoardContains(t, tm, "▸")

	if err := tm.Quit(); err != nil {
		t.Fatalf("quit: %v", err)
	}
	tm.WaitFinished(t, teatest.WithFinalTimeout(5*time.Second))
}

func TestTeatestFilterDoneCandidates(t *testing.T) {
	tuitest.ForceMonochrome(t)
	cfg := config.Default()
	st := store.New(cfg, nil, true)
	m := New(st, actions.NewQueue(), cfg, keys.Default())
	m.Resize(100, 30)
	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(100, 30))
	tm.Send(tea.WindowSizeMsg{Width: 100, Height: 30})
	st.RefreshCmd()()

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("m")})
	waitBoardContains(t, tm, "Done candidates")

	if err := tm.Quit(); err != nil {
		t.Fatalf("quit: %v", err)
	}
	tm.WaitFinished(t, teatest.WithFinalTimeout(5*time.Second))
}

func waitBoardContains(t *testing.T, tm *teatest.TestModel, sub string) {
	t.Helper()
	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		return bytes.Contains(b, []byte(sub))
	}, teatest.WithDuration(5*time.Second))
}
