package overlays

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	zone "github.com/lrstanley/bubblezone"

	"github.com/madicen/aitinerary/internal/tui/commands"
	"github.com/madicen/aitinerary/internal/tui/mouse"
	"github.com/madicen/aitinerary/internal/tui/styles"
	"github.com/madicen/aitinerary/internal/tui/util"
)

// PaletteChoice is the DismissMsg.Result payload from PaletteOverlay.
type PaletteChoice struct {
	Cmd commands.Command
}

const paletteMaxRows = 12

// PaletteOverlay is the ctrl+k fuzzy command palette.
type PaletteOverlay struct {
	input    textinput.Model
	all      []commands.Command
	filtered []commands.Command
	cursor   int
	width    int
}

// NewPaletteOverlay builds the palette over enabled commands.
func NewPaletteOverlay(cmds []commands.Command, w, h int) *PaletteOverlay {
	ti := textinput.New()
	ti.Prompt = "› "
	ti.Placeholder = "type to filter commands…"
	ti.CharLimit = 120
	ti.Width = util.Clamp(w-14, 20, 80)
	ti.Focus()
	m := &PaletteOverlay{
		input: ti,
		all:   cmds,
		width: util.Clamp(w-8, 40, 96),
	}
	m.refilter()
	_ = h
	return m
}

func (m *PaletteOverlay) Init() tea.Cmd { return textinput.Blink }

func (m *PaletteOverlay) refilter() {
	m.filtered = commands.Filter(m.all, m.input.Value())
	if m.cursor >= len(m.filtered) {
		m.cursor = max0(len(m.filtered) - 1)
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}

func (m *PaletteOverlay) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.input.Width = util.Clamp(msg.Width-14, 20, 80)
		m.width = util.Clamp(msg.Width-8, 40, 96)
		return m, nil
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, paletteCloseKeys):
			return m, dismiss(nil)
		case key.Matches(msg, paletteDownKeys):
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
			}
			return m, nil
		case key.Matches(msg, paletteUpKeys):
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil
		case key.Matches(msg, paletteRunKeys):
			return m, m.runSelected()
		}
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		m.refilter()
		return m, cmd
	case tea.MouseMsg:
		if msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft {
			for i := range m.visible() {
				if z := zone.Get(mouse.PaletteRow(i)); z != nil && z.InBounds(msg) {
					m.cursor = m.windowStart() + i
					return m, m.runSelected()
				}
			}
		}
		return m, nil
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	m.refilter()
	return m, cmd
}

func (m *PaletteOverlay) runSelected() tea.Cmd {
	if m.cursor < 0 || m.cursor >= len(m.filtered) {
		return dismiss(nil)
	}
	return dismiss(PaletteChoice{Cmd: m.filtered[m.cursor]})
}

func (m *PaletteOverlay) windowStart() int {
	if len(m.filtered) <= paletteMaxRows {
		return 0
	}
	start := m.cursor - paletteMaxRows/2
	if start < 0 {
		start = 0
	}
	if start > len(m.filtered)-paletteMaxRows {
		start = len(m.filtered) - paletteMaxRows
	}
	return start
}

func (m *PaletteOverlay) visible() []commands.Command {
	start := m.windowStart()
	end := start + paletteMaxRows
	if end > len(m.filtered) {
		end = len(m.filtered)
	}
	return m.filtered[start:end]
}

func (m *PaletteOverlay) View() string {
	var b strings.Builder
	b.WriteString(styles.BoldStyle.Render("Command palette"))
	b.WriteString("\n\n")
	b.WriteString(m.input.View())
	b.WriteString("\n\n")

	if len(m.filtered) == 0 {
		b.WriteString(styles.DimStyle.Render("no matching commands"))
	} else {
		start := m.windowStart()
		for i, c := range m.visible() {
			absolute := start + i
			row := m.renderRow(c, absolute == m.cursor)
			b.WriteString(zone.Mark(mouse.PaletteRow(i), row))
			b.WriteString("\n")
		}
	}
	b.WriteString("\n")
	b.WriteString(styles.DimStyle.Render("↑/↓ move · enter run · esc close"))
	return ModalFrameSized(m.width).Render(b.String())
}

func (m *PaletteOverlay) renderRow(c commands.Command, selected bool) string {
	caret := "  "
	title := c.Title
	if selected {
		caret = styles.OkStyle.Render("› ")
		title = styles.BoldStyle.Render(title)
	}
	line := caret + title
	if sc := c.ShortcutLabel(); sc != "" {
		line += "  " + styles.DimStyle.Render("["+sc+"]")
	}
	if c.Category != "" {
		line += "  " + styles.DimStyle.Render(c.Category)
	}
	return line
}

var (
	paletteCloseKeys = key.NewBinding(key.WithKeys("esc", "ctrl+k"))
	paletteDownKeys  = key.NewBinding(key.WithKeys("down", "ctrl+n"))
	paletteUpKeys    = key.NewBinding(key.WithKeys("up", "ctrl+p"))
	paletteRunKeys   = key.NewBinding(key.WithKeys("enter"))
)
