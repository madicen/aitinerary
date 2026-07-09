package queue

import (
	"context"
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	zone "github.com/lrstanley/bubblezone"

	"github.com/madicen/aitinerary/internal/actions"
	"github.com/madicen/aitinerary/internal/digest"
	"github.com/madicen/aitinerary/internal/jira"
	"github.com/madicen/aitinerary/internal/store"
	"github.com/madicen/aitinerary/internal/tui/keys"
	"github.com/madicen/aitinerary/internal/tui/listnav"
	"github.com/madicen/aitinerary/internal/tui/mouse"
	"github.com/madicen/aitinerary/internal/tui/util"
)

type flatItem struct {
	header bool
	kind   actions.Kind
	action actions.Action
}

// Opts configures the queue tab.
type Opts struct {
	Queue  *actions.Queue
	Engine *actions.Engine
	Keys   keys.Map
	Store  *store.Store
	Demo   bool
}

// Model lists pending actions awaiting review.
type Model struct {
	queue  *actions.Queue
	engine *actions.Engine
	keys   keys.Map
	store  *store.Store
	demo   bool
	now    func() time.Time
	zm     *zone.Manager
	nav    listnav.Model

	width, height int
	originY       int

	flat     []flatItem
	selected int

	editID   string
	editText string
	confirm  bool
	confirmMsg string
	status   string

	lastEntry actions.JournalEntry
	hasUndo   bool
}

// New builds a queue tab model.
func New(o Opts) *Model {
	nowFn := time.Now
	return &Model{
		queue:  o.Queue,
		engine: o.Engine,
		keys:   o.Keys,
		store:  o.Store,
		demo:   o.Demo,
		now:    nowFn,
		zm:     zone.DefaultManager,
		nav:    listnav.New(),
	}
}

func (m *Model) Init() tea.Cmd { return nil }

func (m *Model) Resize(w, h int) {
	m.width = w
	m.height = h
}

func (m *Model) SetContentOrigin(top int) { m.originY = top }

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	m.rebuildFlat()
	switch msg := msg.(type) {
	case digest.ExportedMsg:
		if msg.Err != nil {
			m.status = "digest export: " + msg.Err.Error()
		} else {
			m.status = "digest written to " + msg.Path + " (clipboard)"
		}
		return m, nil
	case util.ClipboardCopiedMsg:
		return m, nil
	case tea.WindowSizeMsg:
		m.Resize(msg.Width, msg.Height)
	case tea.KeyMsg:
		if m.confirm {
			return m.handleConfirmKey(msg)
		}
		if m.editID != "" {
			return m.handleEditKey(msg)
		}
		switch {
		case key.Matches(msg, m.keys.QueueApprove):
			return m, m.approveSelected()
		case key.Matches(msg, m.keys.QueueReject):
			return m, m.rejectSelected()
		case key.Matches(msg, m.keys.QueueEdit):
			return m, m.beginEdit()
		case key.Matches(msg, m.keys.QueueBulkApprove):
			m.confirm = true
			m.confirmMsg = fmt.Sprintf("Approve all %d pending actions?", m.queue.Count())
			return m, nil
		case key.Matches(msg, m.keys.Undo):
			return m, m.Undo()
		case msg.String() == "j", msg.String() == "down":
			m.moveSelection(1)
			return m, nil
		case msg.String() == "k", msg.String() == "up":
			m.moveSelection(-1)
			return m, nil
		}
	case tea.MouseMsg:
		if msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft {
			if z := m.zm.Get(mouse.ZoneQueueBulkApprove); z != nil && z.InBounds(msg) {
				m.confirm = true
				m.confirmMsg = fmt.Sprintf("Approve all %d pending actions?", m.queue.Count())
				return m, nil
			}
			if cmd, ok := m.handleRowMouse(msg); ok {
				return m, cmd
			}
		}
		if m.nav.WheelScroll(msg) {
			return m, nil
		}
	}
	return m, nil
}

func (m *Model) handleConfirmKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y", "enter":
		m.confirm = false
		return m, m.approveAll()
	case "n", "N", "esc":
		m.confirm = false
		return m, nil
	}
	return m, nil
}

func (m *Model) handleEditKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Back):
		m.editID = ""
		m.editText = ""
		return m, nil
	case key.Matches(msg, m.keys.ListOpen):
		return m, m.saveEdit()
	case msg.Type == tea.KeyRunes:
		m.editText += string(msg.Runes)
		return m, nil
	case msg.Type == tea.KeyBackspace:
		if len(m.editText) > 0 {
			m.editText = m.editText[:len(m.editText)-1]
		}
		return m, nil
	}
	return m, nil
}

func (m *Model) handleRowMouse(msg tea.MouseMsg) (tea.Cmd, bool) {
	sel := selectable(m.flat)
	for i := range sel {
		if z := m.zm.Get(mouse.ZoneQueueApprove(i)); z != nil && z.InBounds(msg) {
			m.selected = i
			return m.approveSelected(), true
		}
		if z := m.zm.Get(mouse.ZoneQueueEdit(i)); z != nil && z.InBounds(msg) {
			m.selected = i
			return m.beginEdit(), true
		}
		if z := m.zm.Get(mouse.ZoneQueueReject(i)); z != nil && z.InBounds(msg) {
			m.selected = i
			return m.rejectSelected(), true
		}
		if z := m.zm.Get(mouse.ZoneQueueRow(i)); z != nil && z.InBounds(msg) {
			m.selected = i
			return nil, true
		}
	}
	return nil, false
}

func (m *Model) handleZone(id string) (tea.Cmd, bool) {
	if id == mouse.ConfirmYes {
		m.confirm = false
		return m.approveAll(), true
	}
	if id == mouse.ConfirmNo {
		m.confirm = false
		return nil, true
	}
	return nil, false
}

func (m *Model) moveSelection(delta int) {
	sel := selectable(m.flat)
	if len(sel) == 0 {
		m.selected = 0
		return
	}
	m.selected += delta
	if m.selected < 0 {
		m.selected = 0
	}
	if m.selected >= len(sel) {
		m.selected = len(sel) - 1
	}
}

func (m *Model) selectedAction() (actions.Action, bool) {
	sel := selectable(m.flat)
	if m.selected < 0 || m.selected >= len(sel) {
		return actions.Action{}, false
	}
	return sel[m.selected], true
}

func (m *Model) approveSelected() tea.Cmd {
	a, ok := m.selectedAction()
	if !ok {
		return nil
	}
	return m.applyOne(a)
}

func (m *Model) rejectSelected() tea.Cmd {
	a, ok := m.selectedAction()
	if !ok {
		return nil
	}
	return func() tea.Msg {
		m.queue.Remove(a.ID)
		m.status = "rejected " + a.Describe()
		return nil
	}
}

func (m *Model) approveAll() tea.Cmd {
	pending := m.queue.Pending()
	if len(pending) == 0 {
		return nil
	}
	return func() tea.Msg {
		hadComments := false
		for _, a := range pending {
			if a.Kind == actions.KindAddComment {
				hadComments = true
			}
			m.applySync(a)
		}
		m.status = fmt.Sprintf("approved %d actions", len(pending))
		if hadComments {
			return m.exportDigestMsg()
		}
		return nil
	}
}

func (m *Model) exportDigestMsg() tea.Msg {
	tickets := m.ticketsForDigest()
	cmd := digest.ExportCmd(tickets, m.demo, m.now())
	return cmd()
}

func (m *Model) ticketsForDigest() []jira.Ticket {
	if m.store != nil {
		return m.store.Tickets()
	}
	return nil
}

func (m *Model) applyOne(a actions.Action) tea.Cmd {
	return func() tea.Msg {
		m.applySync(a)
		return nil
	}
}

func (m *Model) applySync(a actions.Action) {
	if m.engine != nil {
		entry, err := m.engine.Apply(context.Background(), a)
		if err != nil {
			m.status = "apply failed: " + err.Error()
			return
		}
		m.lastEntry = entry
		m.hasUndo = true
	}
	m.queue.Remove(a.ID)
	m.status = "applied " + a.Describe()
}

func (m *Model) beginEdit() tea.Cmd {
	a, ok := m.selectedAction()
	if !ok {
		return nil
	}
	m.editID = a.ID
	switch a.Kind {
	case actions.KindAddComment:
		m.editText = a.Comment
	default:
		m.editText = a.Summary
	}
	return nil
}

func (m *Model) saveEdit() tea.Cmd {
	if m.editID == "" {
		return nil
	}
	pending := m.queue.Pending()
	for i, a := range pending {
		if a.ID != m.editID {
			continue
		}
		switch a.Kind {
		case actions.KindAddComment:
			a.Comment = m.editText
		default:
			a.Summary = m.editText
		}
		pending[i] = a
		break
	}
	m.queue.RemoveAll()
	m.queue.Add(pending...)
	m.editID = ""
	m.editText = ""
	m.status = "updated pending action"
	return nil
}

// Undo reverses the last applied action from this tab session.
func (m *Model) Undo() tea.Cmd {
	if !m.hasUndo || m.engine == nil {
		return nil
	}
	entry := m.lastEntry
	return func() tea.Msg {
		if err := m.engine.Undo(context.Background(), entry); err != nil {
			m.status = "undo failed: " + err.Error()
			return nil
		}
		m.hasUndo = false
		m.status = "undid " + entry.Action.Describe()
		return nil
	}
}

func (m *Model) rebuildFlat() {
	byKind := map[actions.Kind][]actions.Action{}
	order := []actions.Kind{}
	for _, a := range m.queue.Pending() {
		if _, seen := byKind[a.Kind]; !seen {
			order = append(order, a.Kind)
		}
		byKind[a.Kind] = append(byKind[a.Kind], a)
	}
	m.flat = nil
	for _, k := range order {
		m.flat = append(m.flat, flatItem{header: true, kind: k})
		for _, a := range byKind[k] {
			m.flat = append(m.flat, flatItem{action: a})
		}
	}
	sel := selectable(m.flat)
	if len(sel) == 0 {
		m.selected = 0
	} else if m.selected >= len(sel) {
		m.selected = len(sel) - 1
	}
}

func selectable(flat []flatItem) []actions.Action {
	var out []actions.Action
	for _, it := range flat {
		if !it.header {
			out = append(out, it.action)
		}
	}
	return out
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
