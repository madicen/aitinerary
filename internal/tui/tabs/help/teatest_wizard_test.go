package help_test

import (
	"bytes"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/madicen/aitinerary/internal/tui/model"
	"github.com/madicen/aitinerary/internal/tui/tabs/help"
	"github.com/madicen/aitinerary/internal/tui/tuitest"
)

func TestFlowSetupWizardHappyPathDemo(t *testing.T) {
	tuitest.ForceMonochrome(t)
	m := model.New(model.Options{Demo: true})
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})

	// Help tab → setup wizard.
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("5")})
	if _, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("w")}); cmd != nil {
		if msg := cmd(); msg != nil {
			m.Update(msg)
		}
	}

	view := m.View()
	if !strings.Contains(view, "Setup wizard") {
		t.Fatalf("wizard overlay missing:\n%s", view)
	}
	if !strings.Contains(view, "demo mode") {
		t.Fatalf("demo checks missing:\n%s", view)
	}

	m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if strings.Contains(m.View(), "Setup wizard") {
		t.Fatal("wizard should dismiss on esc")
	}
	if !strings.Contains(m.View(), "Keyboard shortcuts") {
		t.Fatalf("help tab should show shortcuts:\n%s", m.View())
	}
}

func TestCheckerDemoResults(t *testing.T) {
	t.Parallel()
	c := help.Checker{Demo: true}
	results := c.RunAll()
	if len(results) != 6 {
		t.Fatalf("got %d demo checks, want 6", len(results))
	}
	for _, r := range results {
		if r.Status != "ok" {
			t.Fatalf("demo check %q status = %q", r.Title, r.Status)
		}
	}
}

func TestSetupWizardHermetic(t *testing.T) {
	tuitest.ForceMonochrome(t)
	w := help.NewSetupWizard(nil, nil, true)
	w.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	view := w.View()
	if !bytes.Contains([]byte(view), []byte("Setup wizard")) {
		t.Fatalf("view missing title:\n%s", view)
	}
	if !bytes.Contains([]byte(view), []byte("demo mode")) {
		t.Fatalf("view missing demo checks:\n%s", view)
	}
}
