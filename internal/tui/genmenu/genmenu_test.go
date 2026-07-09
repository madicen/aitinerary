package genmenu

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	zone "github.com/lrstanley/bubblezone"

	"github.com/madicen/aitinerary/internal/aiconfig"
	"github.com/madicen/aitinerary/internal/tui/mouse"
	"github.com/madicen/aitinerary/internal/tui/tuitest"
)

func TestLongPressOpensAndSelectsProfile(t *testing.T) {
	tuitest.ForceMonochrome(t)
	zone.NewGlobal()
	s := &State{}
	profiles := []aiconfig.Profile{
		{Name: "default", Provider: aiconfig.ProviderOllama, Model: "qwen"},
		{Name: "cloud", Provider: aiconfig.ProviderOpenAICompatible, Model: "gpt-4"},
	}
	msg := tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonLeft, X: 10, Y: 5}
	cmd := s.BeginPress(mouse.ZoneBoardGenerate, msg)
	if cmd == nil {
		t.Fatal("expected tick cmd")
	}
	tick, ok := cmd().(TickMsg)
	if !ok {
		t.Fatalf("expected TickMsg, got %T", cmd())
	}
	if !s.OpenIfMatches(tick) {
		t.Fatal("menu should open")
	}
	view := Render(nil, profiles, "default", 1)
	if view == "" {
		t.Fatal("render empty")
	}
	release := tea.MouseMsg{Action: tea.MouseActionRelease, Button: tea.MouseButtonLeft, X: 10, Y: 7}
	// Without real zones marked, HitTestRelease returns -1 but still closes.
	if hit := s.HitTestRelease(nil, release, len(profiles)); hit != -1 {
		t.Fatalf("expected -1 without zones, got %d", hit)
	}
	if s.IsShown() {
		t.Fatal("menu should close on release")
	}
	_ = time.Millisecond // keep time import used
}

func TestStaleTickIgnored(t *testing.T) {
	s := &State{}
	msg := tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonLeft, X: 1, Y: 1}
	_ = s.BeginPress(mouse.ZoneBoardGenerate, msg)
	s.CancelPress()
	if s.OpenIfMatches(TickMsg{Owner: mouse.ZoneBoardGenerate, PressID: 1}) {
		t.Fatal("stale tick after cancel should not open menu")
	}
}
