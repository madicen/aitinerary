package keys

import (
	"testing"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

func keyMsg(s string) tea.KeyMsg {
	switch s {
	case " ":
		return tea.KeyMsg{Type: tea.KeySpace}
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "tab":
		return tea.KeyMsg{Type: tea.KeyTab}
	case "up":
		return tea.KeyMsg{Type: tea.KeyUp}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	}
	return parseKey(s)
}

func parseKey(s string) tea.KeyMsg {
	if len(s) > 5 && s[:5] == "ctrl+" {
		r := s[5]
		switch r {
		case 'k':
			return tea.KeyMsg{Type: tea.KeyCtrlK}
		case 'r':
			return tea.KeyMsg{Type: tea.KeyCtrlR}
		case 'z':
			return tea.KeyMsg{Type: tea.KeyCtrlZ}
		case 'c':
			return tea.KeyMsg{Type: tea.KeyCtrlC}
		}
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func TestEveryFunctionalBindingResolves(t *testing.T) {
	km := Default()
	bindings := map[string]key.Binding{
		"Help": km.Help, "Palette": km.Palette, "Refresh": km.Refresh,
		"Undo": km.Undo, "Back": km.Back, "Quit": km.Quit,
		"TabBoard": km.TabBoard, "TabQueue": km.TabQueue, "TabEpics": km.TabEpics,
		"TabSettings": km.TabSettings, "TabHelp": km.TabHelp,
		"ListSearch": km.ListSearch, "ListOpen": km.ListOpen,
	}
	for name, b := range bindings {
		ks := b.Keys()
		if len(ks) == 0 {
			t.Errorf("binding %s has no keys", name)
			continue
		}
		for _, k := range ks {
			if !key.Matches(keyMsg(k), b) {
				t.Errorf("binding %s: key %q does not resolve via key.Matches", name, k)
			}
		}
	}
}

func TestGlobalBindingsRegistered(t *testing.T) {
	km := Default()
	for name, b := range map[string]key.Binding{
		"Palette": km.Palette, "Refresh": km.Refresh, "Undo": km.Undo,
	} {
		if len(b.Keys()) == 0 {
			t.Errorf("binding %s has no keys", name)
		}
		if b.Help().Key == "" {
			t.Errorf("binding %s has no help key", name)
		}
	}
}

func TestSegText(t *testing.T) {
	km := Default()
	if got := SegText(km.TabBoard); got != "1/b board" {
		t.Fatalf("TabBoard seg = %q, want %q", got, "1/b board")
	}
	if got := SegText(km.ListNav); got != "j/k ↑/↓ nav" {
		t.Fatalf("ListNav seg = %q, want %q", got, "j/k ↑/↓ nav")
	}
}

func TestSectionsCoverContexts(t *testing.T) {
	km := Default()
	secs := km.Sections()
	wantTitles := []string{"Global", "Tabs", "Board", "Queue", "Epics", "Help"}
	if len(secs) != len(wantTitles) {
		t.Fatalf("got %d sections, want %d", len(secs), len(wantTitles))
	}
	for i, want := range wantTitles {
		if secs[i].Title != want {
			t.Errorf("section %d title = %q, want %q", i, secs[i].Title, want)
		}
		if len(secs[i].Bindings) == 0 {
			t.Errorf("section %q has no bindings", want)
		}
	}
}

func TestDisplayOnlyBindingsDisabled(t *testing.T) {
	km := Default()
	for name, b := range map[string]key.Binding{
		"ListNav": km.ListNav, "ListClick": km.ListClick, "ListOpenClick": km.ListOpenClick,
	} {
		if len(b.Keys()) != 0 {
			t.Errorf("display-only binding %s should have no functional keys, got %v", name, b.Keys())
		}
	}
}
