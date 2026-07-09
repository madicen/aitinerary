package mousedouble

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func TestOverlapRelease(t *testing.T) {
	var o OverlapRelease
	var pressGen uint64 = 1
	e := tea.MouseMsg{Button: tea.MouseButtonLeft, Action: tea.MouseActionRelease, X: 3, Y: 4}
	if o.ShouldSkipOverlappingRelease(e, pressGen) {
		t.Fatal("first release should not skip")
	}
	if !o.ShouldSkipOverlappingRelease(e, pressGen) {
		t.Fatal("second overlapping release should skip")
	}
}

func TestDoubleClick(t *testing.T) {
	var d DoubleClick
	t0 := time.Unix(1000, 0)
	e := tea.MouseMsg{Button: tea.MouseButtonLeft, Action: tea.MouseActionRelease}
	if d.ObserveLeftRelease("row:1", e, t0, DefaultDoubleClickWindow) {
		t.Fatal("first click should not be double")
	}
	if !d.ObserveLeftRelease("row:1", e, t0.Add(100*time.Millisecond), DefaultDoubleClickWindow) {
		t.Fatal("second release within window should be double")
	}
}

func TestObserveLeftReleaseEmptyKey(t *testing.T) {
	var d DoubleClick
	e := tea.MouseMsg{Button: tea.MouseButtonLeft, Action: tea.MouseActionRelease}
	if d.ObserveLeftRelease("", e, time.Now(), DefaultDoubleClickWindow) {
		t.Fatal("empty key should not count as double")
	}
}
