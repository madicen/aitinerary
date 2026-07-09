package board

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	zone "github.com/lrstanley/bubblezone"
	overlay "github.com/madicen/bubble-overlay"

	"github.com/madicen/aitinerary/internal/actions"
	"github.com/madicen/aitinerary/internal/config"
	"github.com/madicen/aitinerary/internal/demo"
	"github.com/madicen/aitinerary/internal/gh"
	"github.com/madicen/aitinerary/internal/hygiene"
	"github.com/madicen/aitinerary/internal/jira"
	"github.com/madicen/aitinerary/internal/match"
	"github.com/madicen/aitinerary/internal/store"
	"github.com/madicen/aitinerary/internal/tui/keys"
	"github.com/madicen/aitinerary/internal/tui/listnav"
	"github.com/madicen/aitinerary/internal/tui/mousedouble"
	"github.com/madicen/aitinerary/internal/tui/util"
	"github.com/madicen/aitinerary/internal/tui/util/dropdown"
)

const longPressThreshold = 500 * time.Millisecond

// RowKind distinguishes group headers from ticket rows in the display list.
type RowKind int

const (
	RowGroup RowKind = iota
	RowTicket
)

// Row is one line in the board list (group header or ticket).
type Row struct {
	Kind  RowKind
	Group hygiene.FindingType
	Data  ticketData
}

// Model is the board tab sub-model.
type Model struct {
	listnav.Model

	cfg   *config.Config
	store *store.Store
	queue *actions.Queue
	engine *actions.Engine
	zone  *zone.Manager
	keys  keys.Map
	now   time.Time

	transitions TransitionLister
	sprints     SprintLister
	epics       EpicLister

	width, height int
	contentTop    int

	allTickets []jira.Ticket
	chips      []workspaceChip
	findings   []hygiene.Finding
	matches    []match.Match
	byKey      map[string]ticketData

	filters  FilterState
	rows     []Row
	selected int

	detailOpen       bool
	scrollToSelected bool

	search searchInput

	transitionsFor       []jira.Transition
	transitionsKey       string
	loadingTransitions   bool

	sprintDD     dropdown.Host
	epicDD       dropdown.Host
	transitionDD dropdown.Host
	dueInput     textinput.Model
	labelsInput  textinput.Model
	editMode     editMode

	contextMenu    *ContextMenuState
	rowDoubleClick mousedouble.DoubleClick
	mousePressGen  uint64

	sprintOptions []jira.Sprint
	epicOptions   []jira.Ticket

	statusText string
	errText    string
	sweepMsg   string
	sweep      *sweepState

	multiSelect map[string]bool
	rangeAnchor int
	epicOverlay *epicOverlayState
}

type editMode int

const (
	editNone editMode = iota
	editSprint
	editDueDate
	editParent
	editLabels
	editTransition
)

type searchInput struct {
	open  bool
	input textinput.Model
}

// SetZoneManager wires bubblezone for mouse hit-testing.
func (m *Model) SetZoneManager(zm *zone.Manager) { m.zone = zm }

// SetEngine attaches the actions engine for ticket writes.
func (m *Model) SetEngine(eng *actions.Engine) { m.engine = eng }

// SetListers injects all catalog listers (tests).
func (m *Model) SetListers(tr TransitionLister, sp SprintLister, ep EpicLister) {
	if tr != nil {
		m.transitions = tr
	}
	if sp != nil {
		m.sprints = sp
	}
	if ep != nil {
		m.epics = ep
	}
}

// New constructs the board tab used by the root model.
func New(st *store.Store, q *actions.Queue, cfg *config.Config, km keys.Map) *Model {
	if cfg == nil {
		cfg = config.Default()
	}
	now := demoNow()
	ti := textinput.New()
	ti.Placeholder = "filter tickets"
	ti.Prompt = "/ "
	ti.CharLimit = 256

	list := demoDemoLister()
	m := &Model{
		Model:        listnav.New(),
		cfg:          cfg,
		store:        st,
		queue:        q,
		keys:         km,
		now:          now,
		transitions:  list,
		sprints:      list,
		epics:        list,
		width:        100,
		height:       30,
		filters:      DefaultFilters(),
		selected:     -1,
		search:       searchInput{input: ti},
		dueInput:     textinput.New(),
		labelsInput:  textinput.New(),
		sprintDD:     *dropdown.New("Select sprint…"),
		epicDD:       *dropdown.New("Select epic…"),
		transitionDD: *dropdown.New("Transition to…"),
	}
	m.rebuildChips()
	m.loadData(nil)
	return m
}

// NewWithOptions constructs a board tab with explicit dependencies (tests).
func NewWithOptions(opts Options) *Model {
	m := New(opts.Store, nil, opts.Config, opts.Keys)
	if opts.Engine != nil {
		m.engine = opts.Engine
	}
	if opts.Zone != nil {
		m.zone = opts.Zone
	}
	if opts.Transitions != nil {
		m.transitions = opts.Transitions
	}
	if opts.Sprints != nil {
		m.sprints = opts.Sprints
	}
	if opts.Epics != nil {
		m.epics = opts.Epics
	}
	if !opts.Now.IsZero() {
		m.now = opts.Now
	}
	m.loadData(opts.MergedPRs)
	return m
}

func demoDemoLister() *fakeLister {
	return &fakeLister{
		transitions: []jira.Transition{
			{ID: "1", Name: "Done", ToStatus: "Done"},
			{ID: "2", Name: "In Review", ToStatus: "In Review"},
		},
		sprints: []jira.Sprint{{ID: 9001, Name: "Sprint 24", State: "active"}},
		epics:   demo.Epics(),
	}
}

// legacyOptions kept for tests referencing Options fields.
type Options struct {
	Config        *config.Config
	Store         *store.Store
	Engine        *actions.Engine
	Zone          *zone.Manager
	Keys          keys.Map
	MergedPRs     []gh.PR
	Now           time.Time
	Transitions   TransitionLister
	Sprints       SprintLister
	Epics       EpicLister
}

func (m *Model) rebuildChips() {
	m.chips = nil
	for _, ws := range m.cfg.Workspaces {
		chip := ws.Project
		switch {
		case strings.EqualFold(ws.Name, "IRE"):
			chip = "IRE"
		case strings.EqualFold(ws.Project, "APP"):
			chip = "APP"
		}
		m.chips = append(m.chips, workspaceChip{Name: ws.Name, Project: ws.Project, Chip: chip})
	}
}

func (m *Model) loadData(mergedPRs []gh.PR) {
	if m.store != nil {
		m.allTickets = m.store.Tickets()
	} else {
		m.allTickets = demo.AllTickets()
	}
	if len(mergedPRs) == 0 && m.store != nil && m.store.DemoMode() {
		mergedPRs = demo.MergedPRs()
	}
	prs := make([]match.PR, 0, len(mergedPRs))
	for _, p := range mergedPRs {
		prs = append(prs, p.ToMatch())
	}
	m.matches = match.MatchPRs(m.cfg.ProjectKeys(), prs, m.allTickets)
	rules := hygiene.Rules{
		StaleInProgressDays: m.cfg.StaleInProgressDays,
		Now:                 m.now,
	}
	m.findings = hygiene.Scan(m.allTickets, m.matches, rules)
	m.byKey = indexTickets(m.allTickets, m.chips, m.findings, m.matches)
	m.rebuildRows()
	m.updateStatusText()
}

func (m *Model) rebuildRows() {
	m.rows = nil
	var tickets []ticketData
	for _, t := range m.allTickets {
		td := m.byKey[t.Key]
		if !passesPreset(td, m.filters.Preset) {
			continue
		}
		if !passesWorkspace(td, m.filters.Workspace) {
			continue
		}
		if !passesStatus(td, m.filters.StatusCategory) {
			continue
		}
		if !ticketMatchesSearch(td.Ticket, m.filters.SearchQuery) {
			continue
		}
		tickets = append(tickets, td)
	}

	if m.filters.Preset == FilterNeedsAttention {
		types := groupFindingTypes(m.findings)
		for _, ft := range types {
			var group []ticketData
			for _, td := range tickets {
				for _, f := range td.Findings {
					if f.Type == ft {
						group = append(group, td)
						break
					}
				}
			}
			if len(group) == 0 {
				continue
			}
			sort.Slice(group, func(i, j int) bool { return group[i].Ticket.Key < group[j].Ticket.Key })
			m.rows = append(m.rows, Row{Kind: RowGroup, Group: ft})
			for _, td := range group {
				m.rows = append(m.rows, Row{Kind: RowTicket, Data: td})
			}
		}
	} else {
		sort.Slice(tickets, func(i, j int) bool {
			return tickets[i].Ticket.Updated.After(tickets[j].Ticket.Updated)
		})
		for _, td := range tickets {
			m.rows = append(m.rows, Row{Kind: RowTicket, Data: td})
		}
	}

	if len(m.rows) == 0 {
		m.selected = -1
		m.detailOpen = false
		return
	}
	if m.selected >= len(m.rows) {
		m.selected = len(m.rows) - 1
	}
	for m.selected >= 0 && m.rows[m.selected].Kind == RowGroup {
		m.selected--
	}
	if m.selected < 0 {
		for i, r := range m.rows {
			if r.Kind == RowTicket {
				m.selected = i
				break
			}
		}
	}
}

func (m *Model) updateStatusText() {
	counts := countFindingsByType(m.findings)
	total := len(m.findings)
	switch m.filters.Preset {
	case FilterNeedsAttention:
		if total == 0 {
			m.statusText = "Needs attention: none"
			return
		}
		parts := []string{fmt.Sprintf("Needs attention: %d", total)}
		for _, ft := range groupFindingTypes(m.findings) {
			if c := counts[ft]; c > 0 {
				parts = append(parts, fmt.Sprintf("%s %d", findingTypeLabel(ft), c))
			}
		}
		m.statusText = strings.Join(parts, " · ")
	case FilterDoneCandidates:
		c := counts[hygiene.MergedNotDone]
		m.statusText = fmt.Sprintf("Done candidates: %d", c)
	default:
		m.statusText = fmt.Sprintf("All tickets: %d", len(m.allTickets))
	}
}

// StatusText returns the board summary for the root status bar (P6.2).
func (m *Model) StatusText() string { return m.statusText }

// Init triggers an initial refresh when a store is attached.
func (m *Model) Init() tea.Cmd {
	if m.store != nil && len(m.allTickets) == 0 {
		return m.store.RefreshCmd()
	}
	return nil
}

// Resize sets the content area dimensions.
func (m *Model) Resize(w, h int) {
	m.width = w
	m.height = h
}

// SetContentOrigin records the absolute terminal row where tab content begins.
func (m *Model) SetContentOrigin(top int) {
	m.contentTop = top
	m.sprintDD.ContentTop = top
	m.epicDD.ContentTop = top
	m.transitionDD.ContentTop = top
}

// Update handles Bubble Tea messages for the board tab.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case SweepDoneMsg:
		m.sweep = nil
		if msg.Cancelled {
			m.sweepMsg = "sweep cancelled"
			return m, nil
		}
		if msg.Err != nil {
			m.sweepMsg = fmt.Sprintf("sweep failed: %v", msg.Err)
			return m, nil
		}
		if m.queue != nil && len(msg.Actions) > 0 {
			m.queue.Add(msg.Actions...)
		}
		if msg.Queued > 0 {
			m.sweepMsg = fmt.Sprintf("sweep queued %d actions", msg.Queued)
		}
		m.loadData(nil)
		return m, nil
	case store.StoreRefreshedMsg:
		m.loadData(nil)
		if msg.Err != nil {
			m.errText = msg.Err.Error()
		}
	case LongPressTickMsg:
		if msg.PressID == m.LongPressPressID && m.LongPressItemIndex >= 0 {
			idx := m.rowIndexFromVisible(m.LongPressItemIndex)
			if idx >= 0 {
				m.contextMenu = &ContextMenuState{
					RowIndex: idx,
					MouseX:   m.LongPressMouseX,
					MouseY:   m.LongPressMouseY,
					PressID:  msg.PressID,
					HoverItem: -1,
				}
				m.selected = idx
				m.detailOpen = true
				m.scrollToSelected = true
			}
		}
	case TransitionsLoadedMsg:
		m.loadingTransitions = false
		if msg.Err != nil {
			m.errText = msg.Err.Error()
			return m, nil
		}
		if msg.Key == m.selectedKey() {
			m.transitionsFor = msg.Transitions
			m.transitionsKey = msg.Key
			m.rebuildTransitionDropdown()
		}
	case SprintsLoadedMsg:
		if msg.Err != nil {
			m.errText = msg.Err.Error()
			return m, nil
		}
		m.rebuildSprintDropdown(msg.Sprints)
	case EpicsLoadedMsg:
		if msg.Err != nil {
			m.errText = msg.Err.Error()
			return m, nil
		}
		m.rebuildEpicDropdown(msg.Epics)
	case ActionAppliedMsg:
		if msg.Err != nil {
			m.errText = msg.Err.Error()
			return m, refreshStoreCmd(m.store)
		}
		m.closeEdits()
		cmds = append(cmds, refreshStoreCmd(m.store))
	case SprintSelectedMsg:
		return m, m.applySprintSelection(msg.Index)
	case EpicSelectedMsg:
		return m, m.applyEpicSelection(msg.Index)
	case TransitionSelectedMsg:
		return m, m.applyTransitionSelection(msg.Index)
	case util.ClipboardCopiedMsg:
		if msg.Success {
			m.statusText = "Copied to clipboard"
		} else if msg.Err != nil {
			m.errText = msg.Err.Error()
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		if m.epicOverlay != nil && m.epicOverlay.open {
			return m.handleEpicOverlayKey(msg)
		}
		if m.sweep != nil && m.sweep.active {
			if key.Matches(msg, m.keys.Back) || msg.String() == "esc" {
				return m, m.cancelSweep()
			}
			return m, nil
		}
		if m.search.open {
			return m.handleSearchKey(msg)
		}
		if m.editMode == editDueDate {
			var cmd tea.Cmd
			m.dueInput, cmd = m.dueInput.Update(msg)
			if key.Matches(msg, m.keys.Back) {
				m.closeEdits()
				return m, nil
			}
			if key.Matches(msg, m.keys.ListOpen) {
				return m.applyDueDate(m.dueInput.Value())
			}
			return m, cmd
		}
		if m.editMode == editLabels {
			var cmd tea.Cmd
			m.labelsInput, cmd = m.labelsInput.Update(msg)
			if key.Matches(msg, m.keys.Back) {
				m.closeEdits()
				return m, nil
			}
			if key.Matches(msg, m.keys.ListOpen) {
				return m.applyLabels(m.labelsInput.Value())
			}
			return m, cmd
		}
		if cmd := m.forwardDropdown(msg); cmd != nil {
			return m, cmd
		}
		updated, cmd := m.handleKey(msg)
		cmds = append(cmds, cmd)
		return updated, tea.Batch(cmds...)
	case zone.MsgZoneInBounds:
		if msg.Event.Action != tea.MouseActionRelease || msg.Event.Button != tea.MouseButtonLeft {
			return m, nil
		}
		zid := m.resolveClickedZone(msg)
		if zid == "" {
			return m, nil
		}
		updated, cmd := m.handleZone(zid, msg.Event)
		cmds = append(cmds, cmd)
		return updated, tea.Batch(cmds...)
	case tea.MouseMsg:
		if m.search.open {
			return m, nil
		}
		if m.WheelScroll(msg) {
			return m, nil
		}
		if cmd := m.handleLongPress(msg); cmd != nil {
			return m, cmd
		}
	}

	if cmd := m.forwardDropdown(msg); cmd != nil {
		cmds = append(cmds, cmd)
	}
	return m, tea.Batch(cmds...)
}

func (m *Model) handleKey(msg tea.KeyMsg) (*Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.BoardSyncPrep):
		return m, m.startSweep()
	case key.Matches(msg, m.keys.ListSearch):
		return m.beginSearch()
	case key.Matches(msg, m.keys.Back):
		if m.sweep != nil && m.sweep.active {
			return m, m.cancelSweep()
		}
		if m.contextMenu != nil {
			m.contextMenu = nil
			return m, nil
		}
		if m.detailOpen {
			m.detailOpen = false
			m.closeEdits()
			return m, nil
		}
	case key.Matches(msg, m.keys.BoardFilterAll):
		m.filters.Preset = FilterAll
		m.rebuildRows()
		m.updateStatusText()
		return m, nil
	case key.Matches(msg, m.keys.BoardFilterNeedsAttention):
		m.filters.Preset = FilterNeedsAttention
		m.rebuildRows()
		m.updateStatusText()
		return m, nil
	case key.Matches(msg, m.keys.BoardFilterDoneCandidates):
		m.filters.Preset = FilterDoneCandidates
		m.rebuildRows()
		m.updateStatusText()
		return m, nil
	case key.Matches(msg, m.keys.ListOpen):
		if m.selectedKey() != "" {
			m.detailOpen = true
			return m, m.loadDetailCmds()
		}
	case key.Matches(msg, m.keys.BoardOpenBrowser):
		return m, m.openBrowserCmd()
	case key.Matches(msg, m.keys.BoardCopyKey):
		if k := m.selectedKey(); k != "" {
			return m, util.CopyPlainTextCmd(k)
		}
	case key.Matches(msg, m.keys.BoardEditSprint):
		return m.beginEditSprint()
	case key.Matches(msg, m.keys.BoardEditDueDate):
		return m.beginEditDueDate()
	case key.Matches(msg, m.keys.BoardEditParent):
		return m.beginEditParent()
	case key.Matches(msg, m.keys.BoardEditLabels):
		return m.beginEditLabels()
	case key.Matches(msg, m.keys.BoardEditTransition):
		return m.beginEditTransition()
	case key.Matches(msg, m.keys.BoardMultiSelect):
		m.toggleMultiSelect()
		return m, nil
	case key.Matches(msg, m.keys.BoardRangeSelect):
		m.rangeMultiSelect()
		return m, nil
	case key.Matches(msg, m.keys.BoardCreateEpic):
		return m, m.openEpicOverlay()
	case msg.String() == "j", msg.String() == "down":
		return m.moveSelection(1)
	case msg.String() == "k", msg.String() == "up":
		return m.moveSelection(-1)
	}
	return m, nil
}

func (m *Model) moveSelection(delta int) (*Model, tea.Cmd) {
	if len(m.rows) == 0 {
		return m, nil
	}
	idx := m.selected
	if idx < 0 {
		idx = 0
	} else {
		idx += delta
	}
	for idx >= 0 && idx < len(m.rows) && m.rows[idx].Kind == RowGroup {
		idx += delta
	}
	if idx < 0 || idx >= len(m.rows) {
		return m, nil
	}
	m.selected = idx
	m.detailOpen = true
	m.scrollToSelected = true
	return m, m.loadDetailCmds()
}

func (m *Model) selectedKey() string {
	if m.selected < 0 || m.selected >= len(m.rows) {
		return ""
	}
	if m.rows[m.selected].Kind != RowTicket {
		return ""
	}
	return m.rows[m.selected].Data.Ticket.Key
}

func (m *Model) selectedTicket() (jira.Ticket, bool) {
	if m.selected < 0 || m.selected >= len(m.rows) {
		return jira.Ticket{}, false
	}
	if m.rows[m.selected].Kind != RowTicket {
		return jira.Ticket{}, false
	}
	return m.rows[m.selected].Data.Ticket, true
}

func (m *Model) loadDetailCmds() tea.Cmd {
	key := m.selectedKey()
	if key == "" {
		return nil
	}
	m.loadingTransitions = true
	return loadTransitionsCmd(m.transitions, key)
}

func (m *Model) rowIndexFromVisible(visible int) int {
	ticketRows := m.ticketRowIndices()
	if visible < 0 || visible >= len(ticketRows) {
		return -1
	}
	return ticketRows[visible]
}

func (m *Model) ticketRowIndices() []int {
	var out []int
	for i, r := range m.rows {
		if r.Kind == RowTicket {
			out = append(out, i)
		}
	}
	return out
}

func (m *Model) visibleTicketCount() int {
	return len(m.ticketRowIndices())
}

// View renders the board tab.
func (m *Model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading…"
	}
	v := m.renderBoard()
	v = m.renderSweepOverlay(v)
	if m.contextMenu != nil {
		menu := m.renderContextMenu()
		v = overlay.OverlayViewAtPoint(v, menu, m.width, m.height, m.contextMenu.MouseY, m.contextMenu.MouseX)
	}
	if m.search.open {
		v = m.renderSearchOverlay(v)
	}
	if m.sprintDD.Open() {
		v = m.sprintDD.Composite(v, m.dropdownRow(), 2, m.width, m.height)
	}
	if m.epicDD.Open() {
		v = m.epicDD.Composite(v, m.dropdownRow(), 2, m.width, m.height)
	}
	if m.transitionDD.Open() {
		v = m.transitionDD.Composite(v, m.dropdownRow(), 2, m.width, m.height)
	}
	v = m.renderEpicOverlay(v)
	return v
}

func (m *Model) dropdownRow() int {
	if !m.detailOpen {
		return 8
	}
	return 10
}

func (m *Model) beginSearch() (*Model, tea.Cmd) {
	m.search.open = true
	m.search.input.SetValue(m.filters.SearchQuery)
	m.search.input.Focus()
	return m, textinput.Blink
}

func (m *Model) handleSearchKey(msg tea.KeyMsg) (*Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Back):
		m.search.open = false
		m.search.input.Blur()
		return m, nil
	case key.Matches(msg, m.keys.ListOpen):
		m.filters.SearchQuery = strings.TrimSpace(m.search.input.Value())
		m.search.open = false
		m.search.input.Blur()
		m.rebuildRows()
		return m, nil
	}
	var cmd tea.Cmd
	m.search.input, cmd = m.search.input.Update(msg)
	return m, cmd
}

func (m *Model) closeEdits() {
	m.editMode = editNone
	m.sprintDD.Clear()
	m.epicDD.Clear()
	m.transitionDD.Clear()
	m.dueInput.Blur()
	m.labelsInput.Blur()
}

func (m *Model) forwardDropdown(msg tea.Msg) tea.Cmd {
	switch m.editMode {
	case editSprint:
		return m.sprintDD.Forward(msg)
	case editParent:
		return m.epicDD.Forward(msg)
	case editTransition:
		return m.transitionDD.Forward(msg)
	}
	return nil
}

func (m *Model) openBrowserCmd() tea.Cmd {
	k := m.selectedKey()
	if k == "" || m.cfg.JiraURL == "" {
		return nil
	}
	url := strings.TrimRight(m.cfg.JiraURL, "/") + "/browse/" + k
	return util.OpenURL(url)
}
