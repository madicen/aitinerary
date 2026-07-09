// Package model hosts the root Bubble Tea model. The public facade lives in
// internal/tui; only cmd/aitinerary/main.go should import that package.
package model

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	zone "github.com/lrstanley/bubblezone"
	overlay "github.com/madicen/bubble-overlay"

	"github.com/madicen/aitinerary/internal/actions"
	"github.com/madicen/aitinerary/internal/aiconfig"
	"github.com/madicen/aitinerary/internal/config"
	"github.com/madicen/aitinerary/internal/jira"
	"github.com/madicen/aitinerary/internal/store"
	"github.com/madicen/aitinerary/internal/tui/commands"
	"github.com/madicen/aitinerary/internal/tui/data"
	"github.com/madicen/aitinerary/internal/tui/keys"
	"github.com/madicen/aitinerary/internal/tui/overlays"
	"github.com/madicen/aitinerary/internal/tui/state"
	"github.com/madicen/aitinerary/internal/tui/tabs/board"
	"github.com/madicen/aitinerary/internal/tui/tabs/epics"
	"github.com/madicen/aitinerary/internal/tui/tabs/help"
	"github.com/madicen/aitinerary/internal/tui/tabs/queue"
	"github.com/madicen/aitinerary/internal/tui/tabs/settings"
	"github.com/madicen/aitinerary/internal/tui/util"
)

// FlushMouse resets terminal mouse-tracking state on exit.
var FlushMouse = util.FlushMouse

// New constructs the root TUI model with every tab registered.
func New(opts Options) *Model {
	zone.NewGlobal()

	if opts.Config == nil {
		opts.Config = config.Default()
	}
	if opts.AIConfig == nil {
		opts.AIConfig = aiconfig.DefaultConfig()
	}

	sp := spinner.New()
	sp.Spinner = spinner.Dot

	st := store.New(opts.Config, nil, opts.Demo)
	q := actions.NewQueue()
	autoApplied := actions.NewAutoApplied()
	var eng *actions.Engine
	var jiraClient *jira.Client
	if svc, err := data.NewServices(opts.Config, opts.Demo); err == nil {
		eng = svc.Engine
		jiraClient = svc.Jira
	} else if !opts.Demo {
		eng = actions.NewEngine(actions.NewJiraExecutor(nil), nil)
	} else {
		eng = actions.NewEngine(nil, nil)
	}
	km := keys.Default()

	boardTab := board.New(st, q, opts.Config, km)
	boardTab.SetZoneManager(zone.DefaultManager)
	boardTab.SetEngine(eng)
	boardTab.SetAutoApplied(autoApplied)
	boardTab.SetAIConfig(opts.AIConfig)

	m := &Model{
		opts:    opts,
		mode:    modeBoard,
		keys:    km,
		tabs:    map[mode]Tab{},
		spinner: sp,
		config:  opts.Config,
		store:   st,
		queue:   q,
		engine:  eng,
		demo:    opts.Demo,
		status:  statusBarState{workspace: "All"},

		jira:        jiraClient,
		autoApplied: autoApplied,
	}
	m.tabs[modeBoard] = NewTabAdapter(boardTab)
	m.tabs[modeQueue] = NewTabAdapter(queue.New(queue.Opts{
		Queue:  q,
		Engine: eng,
		Keys:   km,
		Store:  st,
		Demo:   opts.Demo,
	}))
	m.tabs[modeEpics] = NewTabAdapter(epics.New(epics.Opts{
		Store:  st,
		Keys:   km,
		Queue:  q,
		Engine: eng,
	}))
	m.tabs[modeSettings] = NewTabAdapter(settings.New(settings.Opts{
		AppCfg: opts.Config,
		AICfg:  opts.AIConfig,
		Demo:   opts.Demo,
	}))
	m.tabs[modeHelp] = NewTabAdapter(help.New(help.Opts{
		Keys:     km,
		Config:   opts.Config,
		AIConfig: opts.AIConfig,
		Demo:     opts.Demo,
	}))

	m.overlayFocus.Stack = &m.overlayStack
	m.palette = m.buildCommandRegistry()
	return m
}

func (m *Model) openSetupWizardCmd() tea.Cmd {
	w := help.NewSetupWizard(m.config, m.activeAIConfig(), m.demo)
	return tea.Batch(
		m.overlayStack.Push(w, overlay.DefaultOverlayConfig()),
		func() tea.Msg { return tea.WindowSizeMsg{Width: m.width, Height: m.height} },
	)
}

func (m *Model) Init() tea.Cmd {
	var cmds []tea.Cmd
	cmds = append(cmds, m.spinner.Tick)
	cmds = append(cmds, m.refreshAllCmd())
	for _, tab := range m.tabs {
		if tab != nil {
			cmds = append(cmds, tab.Init())
		}
	}
	if help.ShouldShowWizard(m.demo) {
		cmds = append(cmds, m.openSetupWizardCmd())
	}
	return tea.Batch(cmds...)
}

func (m *Model) routeToActiveTab(msg tea.Msg) (tea.Cmd, bool) {
	tab := m.tabs[m.mode]
	if tab == nil {
		return nil, false
	}
	if km, ok := msg.(tea.KeyMsg); ok && km.String() == "ctrl+c" {
		util.FlushMouse()
		return tea.Quit, true
	}
	switch msg.(type) {
	case tea.KeyMsg, tea.MouseMsg:
		if !m.overlayFocus.InteractiveToBase(msg) {
			return m.overlayStack.Update(msg), true
		}
	}
	updated, cmd := tab.Update(msg)
	m.tabs[m.mode] = updated
	return cmd, true
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		bodyH := m.chromeBodyHeight()
		for _, tab := range m.tabs {
			if tab != nil {
				tab.Resize(m.width, bodyH)
				tab.SetContentOrigin(m.headerHeight())
			}
		}
		return m, m.overlayStack.Update(msg)

	case overlays.DismissMsg:
		popped, popCmd := m.overlayStack.Pop()
		switch popped.(type) {
		case *overlays.PaletteOverlay:
			if choice, ok := msg.Result.(overlays.PaletteChoice); ok && choice.Cmd.Run != nil {
				return m, tea.Sequence(popCmd, choice.Cmd.Run())
			}
		}
		return m, popCmd

	case state.NavigateMsg:
		return m.handleNavigate(msg.Target)

	case data.RefreshAllMsg:
		var cmds []tea.Cmd
		for tabMode, tab := range m.tabs {
			if tab == nil {
				continue
			}
			updated, cmd := tab.Update(msg)
			m.tabs[tabMode] = updated
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
		if msg.Store.Err != nil {
			return m, tea.Batch(append(cmds, m.pushErrorOverlay(msg.Store.Err))...)
		}
		for _, err := range msg.AutoApplyErrors {
			if err != nil {
				cmds = append(cmds, m.pushErrorOverlay(err))
			}
		}
		return m, tea.Batch(cmds...)

	case store.StoreRefreshedMsg:
		var cmds []tea.Cmd
		for tabMode, tab := range m.tabs {
			if tab == nil {
				continue
			}
			updated, cmd := tab.Update(msg)
			m.tabs[tabMode] = updated
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
		if msg.Err != nil {
			return m, tea.Batch(append(cmds, m.pushErrorOverlay(msg.Err))...)
		}
		return m, tea.Batch(cmds...)

	case help.OpenSetupWizardMsg:
		return m, m.openSetupWizardCmd()

	case spinner.TickMsg:
		if m.overlayStack.Top() != nil {
			return m, m.overlayStack.Update(msg)
		}
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	switch msg.(type) {
	case tea.KeyMsg, tea.MouseMsg:
		if mm, ok := msg.(tea.MouseMsg); ok && m.overlayFocus.InteractiveToBase(msg) {
			if model, cmd, handled := m.handleStatusBarMouse(mm); handled {
				return model, cmd
			}
			if model, cmd, handled := m.handleTabBarMouse(mm); handled {
				return model, cmd
			}
		}
		if km, ok := msg.(tea.KeyMsg); ok {
			prevMode := m.mode
			hadOverlay := m.overlayStack.Top() != nil
			upd, cmd := m.handleKey(km)
			um := upd.(*Model)
			if cmd != nil || um.mode != prevMode || (um.overlayStack.Top() != nil) != hadOverlay {
				return upd, cmd
			}
		}
		if cmd, handled := m.routeToActiveTab(msg); handled {
			return m, cmd
		}
		if !m.overlayFocus.InteractiveToBase(msg) {
			return m, m.overlayStack.Update(msg)
		}
	}

	if cmd, handled := m.routeToActiveTab(msg); handled {
		return m, cmd
	}
	return m, nil
}

func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	km := m.keys
	switch {
	case key.Matches(msg, km.Quit):
		util.FlushMouse()
		return m, tea.Quit
	case key.Matches(msg, km.Help):
		return m, m.openHelpOverlay()
	case key.Matches(msg, km.Palette):
		return m, m.openCommandPalette()
	case key.Matches(msg, km.Refresh):
		return m, m.refreshAllCmd()
	case key.Matches(msg, km.Undo):
		return m, m.undoCmd()
	case key.Matches(msg, km.Back):
		if m.overlayStack.Top() != nil {
			return m, m.overlayStack.Update(msg)
		}
		return m, nil
	}
	// Settings subtabs own 1–4, e, j/k, etc.; only allow non-conflicting tab jumps.
	if m.mode == modeSettings {
		switch {
		case key.Matches(msg, km.TabBoard):
			return m.switchTab(modeBoard)
		case key.Matches(msg, km.TabQueue):
			return m.switchTab(modeQueue)
		case key.Matches(msg, km.TabHelp):
			return m.switchTab(modeHelp)
		default:
			return m, nil
		}
	}
	switch {
	case key.Matches(msg, km.TabBoard):
		return m.switchTab(modeBoard)
	case key.Matches(msg, km.TabQueue):
		return m.switchTab(modeQueue)
	case key.Matches(msg, km.TabEpics):
		return m.switchTab(modeEpics)
	case key.Matches(msg, km.TabSettings):
		return m.switchTab(modeSettings)
	case key.Matches(msg, km.TabHelp):
		return m.switchTab(modeHelp)
	}
	return m, nil
}

func (m *Model) undoCmd() tea.Cmd {
	if tab, ok := m.tabs[modeQueue].(*tabAdapter); ok {
		if qm, ok := tab.inner.(*queue.Model); ok {
			return qm.Undo()
		}
	}
	return nil
}

func (m *Model) pushErrorOverlay(err error) tea.Cmd {
	if err == nil {
		return nil
	}
	em := overlays.NewErrorOverlay(err.Error(), max(40, m.width-6), max(8, m.height-8))
	return tea.Batch(
		m.overlayStack.Push(em, overlay.DefaultOverlayConfig()),
		func() tea.Msg { return tea.WindowSizeMsg{Width: m.width, Height: m.height} },
	)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (m *Model) commandContext() commands.Context {
	return commands.Context{Mode: m.mode.String()}
}

// QueueCount exposes pending action count for tests.
func (m *Model) QueueCount() int { return m.queueCount() }

// SetMode switches the active tab (tests).
func (m *Model) SetMode(mode state.ViewMode) { m.mode = mode }

// Mode returns the active tab.
func (m *Model) Mode() state.ViewMode { return m.mode }
