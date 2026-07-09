package listnav

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	zone "github.com/lrstanley/bubblezone"
)

func TestNewResetsLongPress(t *testing.T) {
	m := New()
	if m.LongPressItemIndex != -1 {
		t.Errorf("LongPressItemIndex = %d, want -1", m.LongPressItemIndex)
	}
}

func TestWheelScroll(t *testing.T) {
	m := New()
	m.YOffset = 5
	if !m.WheelScroll(tea.MouseMsg{Button: tea.MouseButtonWheelDown}) {
		t.Fatal("wheel down should be handled")
	}
	if m.YOffset != 8 {
		t.Errorf("YOffset = %d, want 8", m.YOffset)
	}
}

func TestVisibleRange(t *testing.T) {
	m := New()
	m.YOffset = 500
	start, end := m.VisibleRange(100, 10)
	if start != 90 || end != 100 {
		t.Errorf("VisibleRange = (%d,%d), want (90,100)", start, end)
	}
}

func TestScrollToSelected(t *testing.T) {
	m := New()
	m.YOffset = 20
	m.ScrollToSelected(5, 100, 10)
	if m.YOffset != 5 {
		t.Errorf("YOffset = %d, want 5", m.YOffset)
	}
}

func TestArmLongPress_ReleaseResets(t *testing.T) {
	m := New()
	m.LongPressItemIndex = 2
	cfg := LongPressConfig{RowZoneID: func(i int) string { return "row" }}
	m.ArmLongPress(zone.New(), tea.MouseMsg{Action: tea.MouseActionRelease}, cfg)
	if m.LongPressItemIndex != -1 {
		t.Errorf("release should reset index, got %d", m.LongPressItemIndex)
	}
}

func TestArmLongPress_MenuOpenGuard(t *testing.T) {
	m := New()
	cfg := LongPressConfig{
		MenuOpen:  true,
		ItemCount: 3,
		RowZoneID: func(i int) string { return "row" },
		Threshold: time.Millisecond,
	}
	press := tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonLeft}
	if cmd := m.ArmLongPress(zone.New(), press, cfg); cmd != nil {
		t.Error("press while menu open should not arm")
	}
}
