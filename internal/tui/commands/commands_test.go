package commands

import (
	"testing"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

func sampleRegistry() *Registry {
	r := New()
	r.Register(Command{ID: "refresh", Title: "Refresh tickets", Category: "board",
		Enabled: func(c Context) bool { return c.Mode == "board" },
		Run:     func() tea.Cmd { return nil }})
	r.Register(Command{ID: "undo", Title: "Undo last action", Category: "global",
		Run: func() tea.Cmd { return nil }})
	return r
}

func TestEnabledGating(t *testing.T) {
	r := sampleRegistry()
	board := r.Enabled(Context{Mode: "board"})
	if !hasID(board, "refresh") || !hasID(board, "undo") {
		t.Fatalf("board ctx: got %v", ids(board))
	}
	queue := r.Enabled(Context{Mode: "queue"})
	if hasID(queue, "refresh") || !hasID(queue, "undo") {
		t.Fatalf("queue ctx: got %v", ids(queue))
	}
}

func TestFilterFuzzyMatch(t *testing.T) {
	r := sampleRegistry()
	all := r.All()
	if got := Filter(all, ""); len(got) != 2 {
		t.Fatalf("empty query passthrough, got %v", ids(got))
	}
	got := Filter(all, "undo")
	if len(got) == 0 || got[0].ID != "undo" {
		t.Fatalf("query undo: got %v", ids(got))
	}
}

func TestShortcutLabel(t *testing.T) {
	withKey := Command{Binding: key.NewBinding(key.WithKeys("f"), key.WithHelp("f", "filter"))}
	if !withKey.HasBinding() || withKey.ShortcutLabel() != "f" {
		t.Fatalf("expected f, got %q", withKey.ShortcutLabel())
	}
}

func hasID(cmds []Command, id string) bool {
	for _, c := range cmds {
		if c.ID == id {
			return true
		}
	}
	return false
}

func ids(cmds []Command) []string {
	out := make([]string, len(cmds))
	for i, c := range cmds {
		out[i] = c.ID
	}
	return out
}
