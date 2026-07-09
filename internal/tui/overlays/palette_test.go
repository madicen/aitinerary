package overlays

import (
	"testing"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/madicen/aitinerary/internal/tui/commands"
)

func paletteCmds() []commands.Command {
	return []commands.Command{
		{ID: "refresh", Title: "Refresh tickets", Category: "board",
			Binding: key.NewBinding(key.WithKeys("ctrl+r"), key.WithHelp("ctrl+r", "refresh"))},
		{ID: "undo", Title: "Undo last action", Category: "global"},
		{ID: "board", Title: "Go to board", Category: "nav"},
	}
}

func typeRunes(p *PaletteOverlay, s string) *PaletteOverlay {
	for _, r := range s {
		out, _ := p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		p = out.(*PaletteOverlay)
	}
	return p
}

func TestPaletteFiltersFuzzy(t *testing.T) {
	p := NewPaletteOverlay(paletteCmds(), 100, 40)
	if len(p.filtered) != 3 {
		t.Fatalf("initial palette should show all 3 commands, got %d", len(p.filtered))
	}
	p = typeRunes(p, "undo")
	if len(p.filtered) == 0 || p.filtered[0].ID != "undo" {
		t.Fatalf("query undo should surface undo command, got %v", ids(p.filtered))
	}
}

func TestPaletteEnterEmitsChoice(t *testing.T) {
	p := NewPaletteOverlay(paletteCmds(), 100, 40)
	p = typeRunes(p, "board")
	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("enter produced no dismiss cmd")
	}
	dm, ok := cmd().(DismissMsg)
	if !ok {
		t.Fatalf("enter should emit DismissMsg, got %T", cmd())
	}
	choice, ok := dm.Result.(PaletteChoice)
	if !ok || choice.Cmd.ID != "board" {
		t.Fatalf("dismiss should carry board choice, got %v", dm.Result)
	}
}

func TestPaletteEscCloses(t *testing.T) {
	p := NewPaletteOverlay(paletteCmds(), 100, 40)
	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatal("esc produced no dismiss cmd")
	}
	dm, ok := cmd().(DismissMsg)
	if !ok || dm.Result != nil {
		t.Fatalf("esc dismiss should carry no choice, got %v", dm.Result)
	}
}

func ids(cmds []commands.Command) []string {
	out := make([]string, len(cmds))
	for i, c := range cmds {
		out[i] = c.ID
	}
	return out
}
