package model_test

import (
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/madicen/aitinerary/internal/store"
	"github.com/madicen/aitinerary/internal/tui/model"
	"github.com/madicen/aitinerary/internal/tui/state"
	"github.com/madicen/aitinerary/internal/tui/tuitest"
)

func drainModelCmd(t *testing.T, m *model.Model, cmd tea.Cmd) *model.Model {
	t.Helper()
	for cmd != nil {
		msg := cmd()
		if msg == nil {
			break
		}
		updated, next := m.Update(msg)
		if um, ok := updated.(*model.Model); ok {
			m = um
		}
		cmd = next
	}
	return m
}

func seedQueueViaSweep(t *testing.T, m *model.Model) int {
	t.Helper()
	updated, cmd := m.Update(tea.WindowSizeMsg{Width: flowW, Height: flowH})
	if um, ok := updated.(*model.Model); ok {
		m = um
	}
	m = drainModelCmd(t, m, cmd)

	updated, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("S")})
	if um, ok := updated.(*model.Model); ok {
		m = um
	}
	m = drainModelCmd(t, m, cmd)

	if m.QueueCount() == 0 {
		t.Fatal("sync-prep sweep should queue demo actions")
	}
	return m.QueueCount()
}

func TestFlowQueueBulkConfirmTrapsKeys(t *testing.T) {
	tuitest.ForceMonochrome(t)
	m := model.New(model.Options{Demo: true})
	before := seedQueueViaSweep(t, m)

	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("2")})
	view := m.View()
	if !strings.Contains(view, "Review queue") {
		t.Fatalf("queue tab missing:\n%s", view)
	}

	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("A")})
	view = m.View()
	if !strings.Contains(view, "Approve all") || !strings.Contains(view, "Yes") {
		t.Fatalf("bulk confirm modal missing:\n%s", view)
	}

	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	if m.QueueCount() != before {
		t.Fatalf("approve key leaked through confirm: pending=%d want %d", m.QueueCount(), before)
	}
	if !strings.Contains(m.View(), "Approve all") {
		t.Fatal("confirm dismissed unexpectedly on unrelated key")
	}

	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	if strings.Contains(m.View(), "Approve all") {
		t.Fatalf("confirm should dismiss on n:\n%s", m.View())
	}
	if m.QueueCount() != before {
		t.Fatalf("n should not approve: pending=%d want %d", m.QueueCount(), before)
	}
}

func TestFlowErrorOverlayDismissesOnEsc(t *testing.T) {
	tuitest.ForceMonochrome(t)
	m := model.New(model.Options{Demo: true})
	updated, sizeCmd := m.Update(tea.WindowSizeMsg{Width: flowW, Height: flowH})
	if um, ok := updated.(*model.Model); ok {
		m = um
	}
	m = drainModelCmd(t, m, sizeCmd)

	updated, navCmd := m.Update(state.NavigateMsg{
		Target: state.NavigateTarget{Err: errors.New("jira unreachable")},
	})
	if um, ok := updated.(*model.Model); ok {
		m = um
	}
	m = drainModelCmd(t, m, navCmd)
	view := m.View()
	if !strings.Contains(view, "Error") || !strings.Contains(view, "jira unreachable") {
		t.Fatalf("error overlay missing:\n%s", view)
	}
	if !strings.Contains(view, "Dismiss") {
		t.Fatalf("error overlay chrome missing:\n%s", view)
	}

	modeBefore := m.Mode()
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	if m.Mode() != modeBefore {
		t.Fatalf("y leaked through error overlay: mode=%v", m.Mode())
	}
	if !strings.Contains(m.View(), "jira unreachable") {
		t.Fatal("error overlay dismissed on y")
	}

	updated, escCmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if um, ok := updated.(*model.Model); ok {
		m = um
	}
	m = drainModelCmd(t, m, escCmd)
	if strings.Contains(m.View(), "Dismiss (Esc)") {
		t.Fatalf("error overlay should dismiss on esc:\n%s", m.View())
	}
}

func TestTeatestErrorOverlayEsc(t *testing.T) {
	tm := newFlowModel(t)
	waitForText(t, tm, "APP-101")

	tm.Send(store.StoreRefreshedMsg{Err: errors.New("teatest overlay failure")})
	waitForText(t, tm, "teatest overlay failure")

	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})
	waitForText(t, tm, "APP-101")

	quit(t, tm)
}

func TestTeatestQueueBulkConfirmEsc(t *testing.T) {
	tm := newFlowModel(t)
	waitForText(t, tm, "APP-101")

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("S")})
	waitForText(t, tm, "sweep queued")

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("2")})
	waitForText(t, tm, "Review queue")

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("A")})
	waitForText(t, tm, "Approve all")

	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})
	waitForText(t, tm, "▸ Comments")

	quit(t, tm)
}
