package settings_test

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/madicen/aitinerary/internal/aiconfig"
	"github.com/madicen/aitinerary/internal/tui/tabs/settings"
	"github.com/madicen/aitinerary/internal/tui/tuitest"
)

const testSecretKey = "sk-super-secret-key-12345"

func TestTeatestAIProfilesAddProfile(t *testing.T) {
	tuitest.ForceMonochrome(t)
	cfg := aiconfig.DefaultConfig()
	cfg.Profiles[0].APIKey = testSecretKey

	m := settings.New(settings.Opts{AICfg: cfg, Demo: true})
	m.Resize(100, 30)
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("3")}) // AI profiles subtab

	before := m.View()
	if !strings.Contains(before, "AI profiles") {
		t.Fatalf("AI profiles panel missing:\n%s", before)
	}
	if strings.Contains(before, testSecretKey) {
		t.Fatalf("raw API key leaked in list view:\n%s", before)
	}

	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	after := m.View()
	if after == before {
		t.Fatalf("pressing n should add a profile and change the view:\n%s", after)
	}
	if !strings.Contains(after, "profile-") {
		t.Fatalf("new profile name missing after add:\n%s", after)
	}
	if strings.Contains(after, testSecretKey) {
		t.Fatalf("raw API key leaked after add:\n%s", after)
	}
}

func TestTeatestAIProfilesSecretsRedacted(t *testing.T) {
	tuitest.ForceMonochrome(t)
	cfg := aiconfig.DefaultConfig()
	cfg.Profiles[0].APIKey = testSecretKey

	m := settings.New(settings.Opts{AICfg: cfg, Demo: true})
	m.Resize(100, 30)
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("3")})

	view := m.View()
	if strings.Contains(view, testSecretKey) {
		t.Fatalf("raw API key must never appear in view output:\n%s", view)
	}

	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e")})
	editView := m.View()
	if strings.Contains(editView, testSecretKey) {
		t.Fatalf("raw API key must never appear in edit view:\n%s", editView)
	}
	if !strings.Contains(editView, "****") {
		t.Fatalf("edit view should show redacted key hint:\n%s", editView)
	}
}
