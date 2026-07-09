// Package keys is the single source of truth for every key binding the root
// TUI responds to. Handlers match against these bindings (key.Matches); the
// status bar, help overlay, and command palette read labels from the same
// Binding.Help() so hints never drift from handlers.
package keys

import "github.com/charmbracelet/bubbles/key"

// Map holds every binding grouped by context. Default() is stored on the root
// model as a value so a future user-remap feature has a clean seam.
type Map struct {
	// Global — available from every tab.
	Help    key.Binding // ? — toggle help overlay
	Palette key.Binding // ctrl+k — command palette
	Refresh key.Binding // ctrl+r — refresh ticket data
	Undo    key.Binding // ctrl+z — undo last action
	Back    key.Binding // esc — back / dismiss
	Quit    key.Binding // ctrl+c — quit

	// Tab navigation (number row + mnemonic letters).
	TabBoard    key.Binding // 1 / b
	TabQueue    key.Binding // 2 / q
	TabEpics    key.Binding // 3 / e
	TabSettings key.Binding // 4 / ,
	TabHelp     key.Binding // 5 / ?

	// Board list — display-only and functional bindings for help/status.
	ListNav         key.Binding // j/k, ↑/↓
	ListClick       key.Binding // click
	ListOpenClick   key.Binding // double-click
	ListSearch      key.Binding // /
	ListOpen        key.Binding // enter
	BoardSyncPrep   key.Binding // S — sync-prep sweep
	BoardFilterAll            key.Binding // 0
	BoardFilterNeedsAttention key.Binding // N
	BoardFilterDoneCandidates key.Binding // m
	BoardEditSprint           key.Binding // s
	BoardEditDueDate          key.Binding // d
	BoardEditParent           key.Binding // p
	BoardEditLabels           key.Binding // l
	BoardEditTransition       key.Binding // t
	BoardOpenBrowser          key.Binding // o
	BoardCopyKey              key.Binding // y
	BoardMultiSelect          key.Binding // space
	BoardRangeSelect          key.Binding // v
	BoardCreateEpic           key.Binding // E
	BoardCreateTicket         key.Binding // n — IRE quick capture
	BoardUndoAuto             key.Binding // z — undo last auto-applied
	BoardSyncReview           key.Binding // R — open sync-prep review

	// Help tab.
	HelpSetupWizard key.Binding // w

	// Queue tab.
	QueueNav        key.Binding // j/k, ↑/↓
	QueueApprove    key.Binding // a
	QueueEdit       key.Binding // e
	QueueReject     key.Binding // x
	QueueBulkApprove key.Binding // A

	// Epics tab.
	EpicsNav            key.Binding // j/k, ↑/↓
	EpicsTimelineScroll key.Binding // h/l
	EpicsEditStartDate  key.Binding // s
	EpicsEditDueDate    key.Binding // d
	EpicsAssign         key.Binding // a
	EpicsMultiSelect    key.Binding // space
}

// Default returns the built-in keymap for aitinerary (PLAN P2.1).
func Default() Map {
	b := func(help, desc string, keys ...string) key.Binding {
		return key.NewBinding(key.WithKeys(keys...), key.WithHelp(help, desc))
	}
	disp := func(help, desc string) key.Binding {
		return key.NewBinding(key.WithHelp(help, desc), key.WithDisabled())
	}
	return Map{
		Help:    b("?", "help", "?"),
		Palette: b("ctrl+k", "commands", "ctrl+k"),
		Refresh: b("ctrl+r", "refresh", "ctrl+r"),
		Undo:    b("ctrl+z", "undo", "ctrl+z"),
		Back:    b("esc", "back", "esc"),
		Quit:    b("ctrl+c", "quit", "ctrl+c"),

		TabBoard:    b("1/b", "board", "1", "b"),
		TabQueue:    b("2/q", "queue", "2", "q"),
		TabEpics:    b("3/e", "epics", "3", "e"),
		TabSettings: b("4/,", "settings", "4", ","),
		TabHelp:     b("5/?", "help tab", "5"),

		ListNav:       disp("j/k ↑/↓", "nav"),
		ListClick:     disp("click", "select"),
		ListOpenClick: disp("double-click", "open"),
		ListSearch:    b("/", "search", "/"),
		ListOpen:      b("enter", "open", "enter"),
		BoardSyncPrep: b("S", "sync-prep sweep", "S"),
		BoardFilterAll:            b("0", "all tickets", "0"),
		BoardFilterNeedsAttention: b("N", "needs attention", "N"),
		BoardFilterDoneCandidates: b("m", "done candidates", "m"),
		BoardEditSprint:           b("s", "sprint", "s"),
		BoardEditDueDate:          b("d", "due date", "d"),
		BoardEditParent:           b("p", "parent epic", "p"),
		BoardEditLabels:           b("l", "labels", "l"),
		BoardEditTransition:       b("t", "transition", "t"),
		BoardOpenBrowser:          b("o", "open in browser", "o"),
		BoardCopyKey:              b("y", "copy key", "y"),
		BoardMultiSelect:          b("space", "toggle multi-select", " "),
		BoardRangeSelect:          b("v", "range select", "v"),
		BoardCreateEpic:           b("E", "create epic from selection", "E"),
		BoardCreateTicket:         b("n", "IRE quick capture", "n"),
		BoardUndoAuto:             b("z", "undo auto-applied", "z"),
		BoardSyncReview:           b("R", "sync-prep review", "R"),

		HelpSetupWizard: b("w", "setup wizard", "w"),

		QueueNav:         disp("j/k ↑/↓", "nav"),
		QueueApprove:     b("a", "approve", "a"),
		QueueEdit:        b("e", "edit", "e"),
		QueueReject:      b("x", "reject", "x"),
		QueueBulkApprove: b("A", "approve all", "A"),

		EpicsNav:            disp("j/k ↑/↓", "nav"),
		EpicsTimelineScroll: disp("h/l", "timeline scroll"),
		EpicsEditStartDate:  b("s", "start date", "s"),
		EpicsEditDueDate:    b("d", "due date", "d"),
		EpicsAssign:         b("a", "assign tickets", "a"),
		EpicsMultiSelect:    b("space", "toggle ticket", " "),
	}
}

// SegText renders a binding as a compact status-bar segment: "key desc".
func SegText(bnd key.Binding) string {
	h := bnd.Help()
	if h.Desc == "" {
		return h.Key
	}
	return h.Key + " " + h.Desc
}

// Section is one titled group in the help overlay.
type Section struct {
	Title    string
	Bindings []key.Binding
}

// Sections returns ordered binding groups for the help overlay.
func (m Map) Sections() []Section {
	return []Section{
		{Title: "Global", Bindings: []key.Binding{
			m.Help, m.Palette, m.Refresh, m.Undo, m.Back, m.Quit,
		}},
		{Title: "Tabs", Bindings: []key.Binding{
			m.TabBoard, m.TabQueue, m.TabEpics, m.TabSettings, m.TabHelp,
		}},
		{Title: "Board", Bindings: []key.Binding{
			m.ListNav, m.ListClick, m.ListOpenClick, m.ListSearch, m.ListOpen, m.BoardSyncPrep,
			m.BoardFilterAll, m.BoardFilterNeedsAttention, m.BoardFilterDoneCandidates,
			m.BoardEditSprint, m.BoardEditDueDate, m.BoardEditParent,
			m.BoardEditLabels, m.BoardEditTransition, m.BoardOpenBrowser, m.BoardCopyKey,
			m.BoardMultiSelect, m.BoardRangeSelect, m.BoardCreateEpic,
			m.BoardCreateTicket, m.BoardUndoAuto, m.BoardSyncReview,
		}},
		{Title: "Queue", Bindings: []key.Binding{
			m.QueueNav, m.QueueApprove, m.QueueEdit, m.QueueReject, m.QueueBulkApprove,
		}},
		{Title: "Epics", Bindings: []key.Binding{
			m.EpicsNav, m.EpicsTimelineScroll,
			m.EpicsEditStartDate, m.EpicsEditDueDate, m.EpicsAssign, m.EpicsMultiSelect, m.ListOpen,
		}},
		{Title: "Help", Bindings: []key.Binding{
			m.HelpSetupWizard,
		}},
	}
}
