package model_test

import (
	"bytes"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"

	"github.com/madicen/aitinerary/internal/tui/model"
	"github.com/madicen/aitinerary/internal/tui/mouse"
	"github.com/madicen/aitinerary/internal/tui/state"
	"github.com/madicen/aitinerary/internal/tui/tuitest"
)

const (
	flowW       = 100
	flowH       = 30
	waitTimeout = 5 * time.Second
)

func newFlowModel(t *testing.T) *teatest.TestModel {
	t.Helper()
	tuitest.ForceMonochrome(t)
	m := model.New(model.Options{Demo: true})
	return teatest.NewTestModel(t, m, teatest.WithInitialTermSize(flowW, flowH))
}

// waitForText blocks until cumulative program output contains sub.
// teatest.WaitFor drains the output buffer each call; only wait for signals
// produced after the previous wait returned.
func waitForText(t *testing.T, tm *teatest.TestModel, sub string) {
	t.Helper()
	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		return bytes.Contains(b, []byte(sub))
	}, teatest.WithDuration(waitTimeout))
}

func quit(t *testing.T, tm *teatest.TestModel) {
	t.Helper()
	if err := tm.Quit(); err != nil {
		t.Fatalf("quit program: %v", err)
	}
	tm.WaitFinished(t, teatest.WithFinalTimeout(waitTimeout))
}

func clickZone(t *testing.T, tm *teatest.TestModel, zoneID string) {
	t.Helper()
	x, y, ok := tuitest.ZoneCenter(zoneID)
	if !ok {
		t.Fatalf("zone %q not registered — send a frame first", zoneID)
	}
	tm.Send(tea.MouseMsg{
		X:      x,
		Y:      y,
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionPress,
	})
}

func TestFlowTabSwitchKeyboard(t *testing.T) {
	tm := newFlowModel(t)
	waitForText(t, tm, "APP-101")
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("2")})
	waitForText(t, tm, "Review queue")
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("b")})
	waitForText(t, tm, "APP-101")
	quit(t, tm)
}

func TestFlowTabSwitchMouse(t *testing.T) {
	tm := newFlowModel(t)
	waitForText(t, tm, "APP-101")
	clickZone(t, tm, mouse.ZoneTabQueue)
	waitForText(t, tm, "Review queue")
	clickZone(t, tm, mouse.ZoneTabBoard)
	waitForText(t, tm, "APP-101")
	quit(t, tm)
}

func TestFlowTabSwitchKeyboardSync(t *testing.T) {
	tuitest.ForceMonochrome(t)
	m := model.New(model.Options{Demo: true})
	m.Update(tea.WindowSizeMsg{Width: flowW, Height: flowH})

	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("2")})
	if m.Mode() != state.ModeQueue {
		t.Fatalf("mode after 2 = %v, want queue", m.Mode())
	}
	view := m.View()
	if !strings.Contains(view, "Review queue") {
		t.Fatalf("queue view missing header:\n%s", view)
	}

	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("b")})
	if m.Mode() != state.ModeBoard {
		t.Fatalf("mode after b = %v, want board", m.Mode())
	}
	if !strings.Contains(m.View(), "Needs attention") {
		t.Fatalf("board view missing filters:\n%s", m.View())
	}
}

func clickTabZone(t *testing.T, m *model.Model, zoneID string) {
	t.Helper()
	_ = m.View()
	x, y, ok := tuitest.ZoneCenter(zoneID)
	if !ok {
		t.Fatalf("zone %q not registered", zoneID)
	}
	m.Update(tea.MouseMsg{
		X: x, Y: y, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress,
	})
}

func TestFlowTabSwitchMouseSync(t *testing.T) {
	tuitest.ForceMonochrome(t)
	m := model.New(model.Options{Demo: true})
	m.Update(tea.WindowSizeMsg{Width: flowW, Height: flowH})

	clickTabZone(t, m, mouse.ZoneTabQueue)
	if m.Mode() != state.ModeQueue {
		t.Fatalf("click queue tab: mode=%v", m.Mode())
	}

	clickTabZone(t, m, mouse.ZoneTabBoard)
	if m.Mode() != state.ModeBoard {
		t.Fatalf("click board tab: mode=%v", m.Mode())
	}
}
