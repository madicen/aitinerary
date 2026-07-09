package settings

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	bubblepicker "github.com/madicen/bubble-color-picker"
	zone "github.com/lrstanley/bubblezone"

	"github.com/madicen/aitinerary/internal/aiconfig"
	"github.com/madicen/aitinerary/internal/config"
	"github.com/madicen/aitinerary/internal/tui/styles"
)

// Subtab identifies a settings panel.
type Subtab int

const (
	SubGeneral Subtab = iota
	SubJira
	SubAIProfiles
	SubTheme
)

const (
	zoneTabGeneral    = "zone:settings:tab:general"
	zoneTabJira       = "zone:settings:tab:jira"
	zoneTabAIProfiles = "zone:settings:tab:ai"
	zoneTabTheme      = "zone:settings:tab:theme"
)

// Opts configures the settings tab.
type Opts struct {
	AppCfg *config.Config
	AICfg  *aiconfig.Config
	Demo   bool
}

// Model is the settings tab with subtabs.
type Model struct {
	appCfg     *config.Config
	aiCfg      *aiconfig.Config
	demo       bool
	subtab     Subtab
	theme      *themePanel
	width      int
	bodyHeight int
	contentTop int
}

// New builds the settings tab.
func New(o Opts) *Model {
	if o.AppCfg == nil {
		o.AppCfg = config.Default()
	}
	if o.AICfg == nil {
		o.AICfg = aiconfig.DefaultConfig()
	}
	return &Model{
		appCfg: o.AppCfg,
		aiCfg:  o.AICfg,
		demo:   o.Demo,
		theme:  newThemePanel(),
	}
}

func (m *Model) Init() tea.Cmd { return nil }

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch typed := msg.(type) {
	case bubblepicker.ColorChangedMsg:
		if m.subtab == SubTheme && m.theme != nil {
			m.theme.applyColor(typed.Color)
			if idx := m.theme.openIndex(); idx >= 0 {
				updated, cmd := m.theme.swatches[idx].swatch.Update(typed)
				m.theme.swatches[idx].swatch = updated
				return m, cmd
			}
		}
		return m, nil
	case bubblepicker.ColorCanceledMsg:
		if m.subtab == SubTheme && m.theme != nil {
			if idx := m.theme.openIndex(); idx >= 0 {
				updated, cmd := m.theme.swatches[idx].swatch.Update(typed)
				m.theme.swatches[idx].swatch = updated
				return m, cmd
			}
		}
		return m, nil
	}

	if m.subtab == SubTheme && m.theme != nil {
		if idx := m.theme.openIndex(); idx >= 0 {
			switch msg.(type) {
			case tea.KeyMsg, tea.MouseMsg:
				updated, cmd := m.theme.swatches[idx].swatch.Update(msg)
				m.theme.swatches[idx].swatch = updated
				return m, cmd
			}
		}
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "[", "shift+tab":
			m.prevSubtab()
			return m, nil
		case "]", "tab":
			m.nextSubtab()
			return m, nil
		case "1":
			m.subtab = SubGeneral
		case "2":
			m.subtab = SubJira
		case "3":
			m.subtab = SubAIProfiles
		case "4":
			m.subtab = SubTheme
		}
		if m.subtab == SubTheme {
			return m.updateThemeKey(msg)
		}
	case tea.MouseMsg:
		if cmd, handled := m.handleMouse(msg); handled {
			return m, cmd
		}
	}
	return m, nil
}

func (m *Model) prevSubtab() {
	if m.subtab == 0 {
		m.subtab = SubTheme
		return
	}
	m.subtab--
}

func (m *Model) nextSubtab() {
	m.subtab = (m.subtab + 1) % (SubTheme + 1)
}

func (m *Model) handleMouse(msg tea.MouseMsg) (tea.Cmd, bool) {
	if msg.Action != tea.MouseActionPress || msg.Button != tea.MouseButtonLeft {
		return nil, false
	}
	type pick struct {
		zone string
		tab  Subtab
	}
	for _, p := range []pick{
		{zoneTabGeneral, SubGeneral},
		{zoneTabJira, SubJira},
		{zoneTabAIProfiles, SubAIProfiles},
		{zoneTabTheme, SubTheme},
	} {
		if z := zone.Get(p.zone); z != nil && z.InBounds(msg) {
			m.subtab = p.tab
			return nil, true
		}
	}
	if m.subtab == SubTheme && m.theme != nil {
		return m.theme.handleMouse(msg)
	}
	return nil, false
}

func (m *Model) updateThemeKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "tab", "ctrl+i", "down", "j":
		m.theme.advanceFocus(1)
	case "shift+tab", "up", "k":
		m.theme.advanceFocus(-1)
	case "enter", " ":
		return m, m.theme.openFocused()
	case "ctrl+s":
		return m, m.theme.saveCmd()
	case "r":
		m.theme.resetAll()
	}
	return m, nil
}

func (m *Model) View() string {
	var b strings.Builder
	b.WriteString(styles.BoldStyle.Render("Settings"))
	b.WriteString("\n\n")
	b.WriteString(m.renderSubtabBar())
	b.WriteString("\n\n")
	switch m.subtab {
	case SubJira:
		b.WriteString(renderJira(m.appCfg))
	case SubAIProfiles:
		b.WriteString(renderAIProfiles(m.aiCfg))
	case SubTheme:
		panel := m.theme.render(m.width)
		panel = m.theme.applyOverlays(panel, m.width, m.bodyHeight)
		b.WriteString(panel)
	default:
		b.WriteString(renderGeneral(m.appCfg, m.demo))
	}
	body := lipgloss.NewStyle().Width(m.width).Height(m.bodyHeight).Padding(0, 1).Render(b.String())
	return body
}

func (m *Model) renderSubtabBar() string {
	tab := func(label string, active bool, id string) string {
		if active {
			label = styles.TabActive.Render(label)
		} else {
			label = styles.DimStyle.Render(label)
		}
		return zone.Mark(id, label)
	}
	return tab("General", m.subtab == SubGeneral, zoneTabGeneral) + "  " +
		tab("Jira", m.subtab == SubJira, zoneTabJira) + "  " +
		tab("AI Profiles", m.subtab == SubAIProfiles, zoneTabAIProfiles) + "  " +
		tab("Theme", m.subtab == SubTheme, zoneTabTheme)
}

func (m *Model) Resize(w, h int) {
	m.width = w
	m.bodyHeight = h
}

func (m *Model) SetContentOrigin(top int) { m.contentTop = top }
