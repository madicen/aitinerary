package model

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/madicen/aitinerary/internal/tui/tuitest"
)

const (
	goldenTermW = 100
	goldenTermH = 30
)

func newGoldenModel(t *testing.T) *Model {
	t.Helper()
	tuitest.ForceMonochrome(t)
	m := New(Options{Demo: true})
	m.Update(tea.WindowSizeMsg{Width: goldenTermW, Height: goldenTermH})
	return m
}

func TestGoldenEmptyShell(t *testing.T) {
	tuitest.ForceMonochrome(t)
	m := newGoldenModel(t)
	tuitest.AssertGolden(t, "empty_shell", m.View())
}
