package model

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	zone "github.com/lrstanley/bubblezone"

	"github.com/madicen/aitinerary/internal/tui/mouse"
	"github.com/madicen/aitinerary/internal/tui/tuitest"
)

func TestHandleTabBarMouseSwitch(t *testing.T) {
	tuitest.ForceMonochrome(t)
	m := New(Options{Demo: true})
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	_ = m.View()

	qx, qy, ok := tuitest.ZoneCenter(mouse.ZoneTabQueue)
	if !ok {
		t.Fatal("queue zone missing")
	}
	mm := tea.MouseMsg{X: qx, Y: qy, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress}
	if !m.overlayFocus.InteractiveToBase(mm) {
		t.Fatal("InteractiveToBase false before queue click")
	}
	_, _, handled := m.handleTabBarMouse(mm)
	if !handled || m.mode != modeQueue {
		t.Fatalf("queue click handled=%v mode=%v", handled, m.mode)
	}

	_ = m.View()
	bx, by, ok := tuitest.ZoneCenter(mouse.ZoneTabBoard)
	if !ok {
		t.Fatal("board zone missing")
	}
	bm := tea.MouseMsg{X: bx, Y: by, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress}
	if !m.overlayFocus.InteractiveToBase(bm) {
		t.Fatal("InteractiveToBase false before board click")
	}
	if _, _, handled := m.handleStatusBarMouse(bm); handled {
		t.Fatal("status bar stole board tab click")
	}
	z := zone.Get(mouse.ZoneTabBoard)
	if z == nil || !z.InBounds(bm) {
		t.Fatalf("board zone miss: z=%+v", z)
	}
	_, _, handled = m.handleTabBarMouse(bm)
	if !handled || m.mode != modeBoard {
		t.Fatalf("board click handled=%v mode=%v", handled, m.mode)
	}
}
