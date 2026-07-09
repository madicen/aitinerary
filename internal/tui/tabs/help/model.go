package help

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/madicen/aitinerary/internal/aiconfig"
	"github.com/madicen/aitinerary/internal/config"
	"github.com/madicen/aitinerary/internal/tui/keys"
	"github.com/madicen/aitinerary/internal/tui/styles"
)

// Opts configures the help tab.
type Opts struct {
	Keys     keys.Map
	Config   *config.Config
	AIConfig *aiconfig.Config
	Demo     bool
}

// Model renders keyboard shortcuts from keys.Map.
type Model struct {
	keys       keys.Map
	cfg        *config.Config
	aiCfg      *aiconfig.Config
	demo       bool
	width      int
	bodyHeight int
	contentTop int
}

// New builds the help tab.
func New(o Opts) *Model {
	return &Model{
		keys:  o.Keys,
		cfg:   o.Config,
		aiCfg: o.AIConfig,
		demo:  o.Demo,
	}
}

func (m *Model) Init() tea.Cmd { return nil }

// OpenSetupWizardMsg requests the root model push the setup wizard overlay.
type OpenSetupWizardMsg struct{}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if key.Matches(msg, m.keys.HelpSetupWizard) {
			return m, func() tea.Msg { return OpenSetupWizardMsg{} }
		}
	}
	return m, nil
}

func (m *Model) View() string {
	var b strings.Builder
	b.WriteString(styles.BoldStyle.Render("Keyboard shortcuts"))
	b.WriteString("\n")
	b.WriteString(styles.DimStyle.Render("w — run setup wizard (Jira, gh, Ollama checks)"))
	b.WriteString("\n\n")
	keyW := 0
	for _, s := range m.keys.Sections() {
		for _, bnd := range s.Bindings {
			if k := bnd.Help().Key; len(k) > keyW {
				keyW = len(k)
			}
		}
	}
	for i, s := range m.keys.Sections() {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString(styles.BoldStyle.Render(s.Title))
		b.WriteString("\n")
		for _, bnd := range s.Bindings {
			h := bnd.Help()
			if h.Key == "" {
				continue
			}
			pad := strings.Repeat(" ", keyW-len(h.Key))
			b.WriteString("  " + styles.BoldStyle.Render(h.Key) + pad + "  " + styles.DimStyle.Render(h.Desc) + "\n")
		}
	}
	return lipgloss.NewStyle().Width(m.width).Height(m.bodyHeight).Padding(0, 1).Render(b.String())
}

func (m *Model) Resize(w, h int) {
	m.width = w
	m.bodyHeight = h
}

func (m *Model) SetContentOrigin(top int) { m.contentTop = top }
