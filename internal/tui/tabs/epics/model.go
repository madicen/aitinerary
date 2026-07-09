package epics

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	zone "github.com/lrstanley/bubblezone"

	"github.com/madicen/aitinerary/internal/actions"
	"github.com/madicen/aitinerary/internal/jira"
	"github.com/madicen/aitinerary/internal/store"
	"github.com/madicen/aitinerary/internal/tui/keys"
	"github.com/madicen/aitinerary/internal/tui/listnav"
	"github.com/madicen/aitinerary/internal/tui/mouse"
)

// EpicRow is one epic with derived child stats for the list pane.
type EpicRow struct {
	Epic     jira.Ticket
	Project  string
	Children int
	Done     int
	Undated  bool
}

// Opts configures the epics tab.
type Opts struct {
	Store  *store.Store
	Queue  *actions.Queue
	Engine *actions.Engine
	Keys   keys.Map
	Now    time.Time
}

// Model is the epics + timeline tab (P8.2/P8.3 basics).
type Model struct {
	store  *store.Store
	queue  *actions.Queue
	engine *actions.Engine
	keys   keys.Map
	zm     *zone.Manager
	nav    listnav.Model
	rows   []EpicRow
	width  int
	height int
	originY int

	selected    int
	timelineOff int
	now         time.Time

	editMode     editMode
	startInput   textinput.Model
	dueInput     textinput.Model
	assignOverlay *assignOverlayState
}

// New builds an epics tab model.
func New(o Opts) *Model {
	now := o.Now
	if now.IsZero() {
		now = time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)
	}
	m := &Model{
		store:  o.Store,
		queue:  o.Queue,
		engine: o.Engine,
		keys:   o.Keys,
		zm:     zone.DefaultManager,
		nav:    listnav.New(),
		now:    now,
	}
	m.initDateInputs()
	return m
}

func (m *Model) Init() tea.Cmd { return nil }

func (m *Model) Resize(w, h int) {
	m.width = w
	m.height = h
}

func (m *Model) SetContentOrigin(top int) { m.originY = top }

func (m *Model) rebuild() {
	if m.store == nil {
		m.rows = nil
		return
	}
	tickets := m.store.Tickets()
	children := map[string][]jira.Ticket{}
	epics := map[string]jira.Ticket{}

	for _, t := range tickets {
		if strings.EqualFold(t.Type, "Epic") {
			epics[strings.ToUpper(t.Key)] = t
			continue
		}
		if t.ParentKey != "" {
			children[strings.ToUpper(t.ParentKey)] = append(children[strings.ToUpper(t.ParentKey)], t)
		}
	}

	m.rows = m.rows[:0]
	for key, epic := range epics {
		project := projectOf(epic.Key)
		kids := children[key]
		done := 0
		for _, c := range kids {
			if c.IsDone() {
				done++
			}
		}
		m.rows = append(m.rows, EpicRow{
			Epic:     epic,
			Project:  project,
			Children: len(kids),
			Done:     done,
			Undated:  epic.StartDate == "" || epic.DueDate == "",
		})
	}
	sort.Slice(m.rows, func(i, j int) bool {
		if m.rows[i].Project != m.rows[j].Project {
			return m.rows[i].Project < m.rows[j].Project
		}
		return m.rows[i].Epic.Key < m.rows[j].Epic.Key
	})
	if m.selected >= len(m.rows) {
		m.selected = max(0, len(m.rows)-1)
	}
}

func projectOf(key string) string {
	if i := strings.IndexByte(key, '-'); i > 0 {
		return key[:i]
	}
	return key
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Resize(msg.Width, msg.Height)
	case store.StoreRefreshedMsg:
		m.rebuild()
	case ActionAppliedMsg:
		if msg.Err != nil {
			return m, refreshStoreCmd(m.store)
		}
		m.closeEdits()
		return m, refreshStoreCmd(m.store)
	case tea.KeyMsg:
		if m.assignOverlay != nil && m.assignOverlay.open {
			return m.handleAssignOverlayKey(msg)
		}
		if m.editMode != editNone {
			return m.handleDateEditKey(msg)
		}
		switch {
		case key.Matches(msg, m.keys.EpicsEditStartDate):
			return m.beginEditStartDate()
		case key.Matches(msg, m.keys.EpicsEditDueDate):
			return m.beginEditDueDate()
		case key.Matches(msg, m.keys.EpicsAssign):
			return m.openAssignOverlay()
		}
		switch msg.String() {
		case "j", "down":
			m.move(1)
		case "k", "up":
			m.move(-1)
		case "h", "left":
			m.timelineOff--
		case "l", "right":
			m.timelineOff++
		}
	case tea.MouseMsg:
		if m.assignOverlay != nil && m.assignOverlay.open {
			return m, nil
		}
		if m.editMode != editNone {
			return m, nil
		}
		if m.nav.WheelScroll(msg) {
			return m, nil
		}
		if msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft {
			for i := range m.rows {
				if z := m.zm.Get(mouse.ZoneEpicsRow(i)); z != nil && z.InBounds(msg) {
					m.selected = i
					return m, nil
				}
			}
		}
	}
	m.rebuild()
	return m, nil
}

func (m *Model) move(delta int) {
	n := len(m.rows)
	if n == 0 {
		m.selected = 0
		return
	}
	m.selected = (m.selected + delta + n) % n
	listH := max(1, m.height-m.originY-10)
	m.nav.ScrollToSelected(m.selected, n, listH)
}

// FormatProgress renders done/total for an epic row.
func FormatProgress(row EpicRow) string {
	if row.Children == 0 {
		return "—"
	}
	return fmt.Sprintf("%d/%d", row.Done, row.Children)
}

// KeyMatchesTimeline reports whether msg scrolls the timeline per keys.Map.
func KeyMatchesTimeline(km keys.Map, msg tea.KeyMsg) bool {
	return key.Matches(msg, km.EpicsTimelineScroll) ||
		msg.String() == "h" || msg.String() == "l" ||
		msg.String() == "left" || msg.String() == "right"
}

// SetNow pins "today" for deterministic golden tests.
func (m *Model) SetNow(t time.Time) { m.now = t }

// TimelineOffset exposes horizontal scroll for tests.
func (m *Model) TimelineOffset() int { return m.timelineOff }

// AssignOverlayOpen reports whether the assign picker is visible (tests).
func (m *Model) AssignOverlayOpen() bool {
	return m.assignOverlay != nil && m.assignOverlay.open
}
