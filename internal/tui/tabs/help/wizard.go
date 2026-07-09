package help

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	zone "github.com/lrstanley/bubblezone"

	"github.com/madicen/aitinerary/internal/aiconfig"
	"github.com/madicen/aitinerary/internal/config"
	"github.com/madicen/aitinerary/internal/tui/mouse"
	"github.com/madicen/aitinerary/internal/tui/overlays"
	"github.com/madicen/aitinerary/internal/tui/styles"
	"github.com/madicen/aitinerary/internal/tui/util"
)

// ChecksDoneMsg delivers async setup check results to the wizard overlay.
type ChecksDoneMsg struct {
	Results []CheckResult
}

// SetupWizard is the first-run setup overlay (P10.1).
type SetupWizard struct {
	checker  Checker
	results  []CheckResult
	width    int
	height   int
	demo     bool
	markDone bool
}

// NewSetupWizard builds the setup wizard overlay.
func NewSetupWizard(cfg *config.Config, aiCfg *aiconfig.Config, demo bool) SetupWizard {
	w := SetupWizard{
		checker:  Checker{Config: cfg, AIConfig: aiCfg, Demo: demo},
		demo:     demo,
		markDone: !demo,
	}
	if demo {
		w.results = w.checker.RunAll()
	}
	return w
}

func (m SetupWizard) Init() tea.Cmd {
	if len(m.results) > 0 {
		return nil
	}
	return m.runChecks()
}

func (m SetupWizard) runChecks() tea.Cmd {
	ch := m.checker
	return func() tea.Msg {
		return ChecksDoneMsg{Results: ch.RunAll()}
	}
}

func (m SetupWizard) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case ChecksDoneMsg:
		m.results = msg.Results
		return m, nil
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.KeyMsg:
		if key.Matches(msg, wizardDismissKeys) {
			return m, m.dismissCmd()
		}
	case tea.MouseMsg:
		if msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft {
			if z := zone.Get(mouse.SetupWizardDismiss); z != nil && z.InBounds(msg) {
				return m, m.dismissCmd()
			}
		}
	}
	return m, nil
}

func (m SetupWizard) dismissCmd() tea.Cmd {
	mark := m.markDone
	return func() tea.Msg {
		if mark {
			_ = MarkWizardComplete()
		}
		return overlays.DismissMsg{}
	}
}

func (m SetupWizard) View() string {
	var b strings.Builder
	b.WriteString(styles.BoldStyle.Render("Setup wizard"))
	b.WriteString("\n\n")
	if len(m.results) == 0 {
		b.WriteString(styles.DimStyle.Render("Running checks…"))
	} else {
		for i, r := range m.results {
			if i > 0 {
				b.WriteString("\n")
			}
			b.WriteString(m.renderResult(r))
		}
	}
	b.WriteString("\n\n")
	b.WriteString(styles.DimStyle.Render("Board ids of 0 auto-discover on refresh. "))
	b.WriteString(zone.Mark(mouse.SetupWizardDismiss, styles.DimStyle.Render("Continue (enter / esc)")))
	outerW := util.Clamp(m.width-6, 48, 100)
	return overlays.ModalFrameSized(outerW).Render(b.String())
}

func (m SetupWizard) renderResult(r CheckResult) string {
	icon := "·"
	style := styles.DimStyle
	switch r.Status {
	case "ok":
		icon = "✓"
		style = styles.OkStyle
	case "warn":
		icon = "!"
		style = styles.WarnStyle
	case "fail":
		icon = "✗"
		style = styles.ErrStyle
	}
	line := style.Render(icon+" "+r.Title+": ") + r.Detail
	if r.Hint != "" && r.Status != "ok" {
		line += "\n    " + styles.DimStyle.Render(r.Hint)
	}
	return line
}

var wizardDismissKeys = key.NewBinding(key.WithKeys("enter", "esc", " "))
