package settings_test

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/madicen/aitinerary/internal/tui/model"
	"github.com/madicen/aitinerary/internal/tui/tuitest"
)

func TestTeatestThemeSwatchOpensPicker(t *testing.T) {
	tuitest.ForceMonochrome(t)
	m := model.New(model.Options{Demo: true})
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("4")}) // settings tab
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("4")}) // theme subtab

	before := m.View()
	if !strings.Contains(before, "Theme colours") {
		t.Fatalf("theme panel missing:\n%s", before)
	}

	x, y, ok := tuitest.ZoneCenter("zone:settings:theme:primary")
	if !ok {
		t.Fatal("primary swatch zone not registered")
	}
	m.Update(tea.MouseMsg{
		X:      x,
		Y:      y,
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionPress,
	})

	after := m.View()
	if after == before {
		t.Fatalf("clicking swatch should change view (picker open):\n%s", after)
	}
}
