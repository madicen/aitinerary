package board

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	overlay "github.com/madicen/bubble-overlay"

	"github.com/madicen/aitinerary/internal/actions"
	"github.com/madicen/aitinerary/internal/demo"
	"github.com/madicen/aitinerary/internal/prompts"
	"github.com/madicen/aitinerary/internal/tui/overlays"
	"github.com/madicen/aitinerary/internal/tui/styles"
	"github.com/madicen/aitinerary/internal/tui/util"
)

// epicOverlayState is the epic builder form (P8.1 stub).
type epicOverlayState struct {
	open      bool
	field     int // 0 title, 1 body, 2 start, 3 due, 4 project
	title     textinput.Model
	body      textinput.Model
	startDate textinput.Model
	dueDate   textinput.Model
	project   textinput.Model
	keys      []string
}

func newEpicOverlay(keys []string, draft prompts.EpicDraftOutput, project string) *epicOverlayState {
	mk := func(val, ph string) textinput.Model {
		ti := textinput.New()
		ti.SetValue(val)
		ti.Placeholder = ph
		ti.CharLimit = 512
		return ti
	}
	e := &epicOverlayState{
		open:      true,
		keys:      append([]string(nil), keys...),
		title:     mk(draft.Title, "Epic title"),
		body:      mk(draft.Body, "Epic body (bullets)"),
		startDate: mk(draft.StartDate, "YYYY-MM-DD start"),
		dueDate:   mk(draft.DueDate, "YYYY-MM-DD due"),
		project:   mk(project, "Project key"),
	}
	e.focusField(0)
	return e
}

func (e *epicOverlayState) focusField(i int) {
	e.field = i
	for idx, ti := range []*textinput.Model{&e.title, &e.body, &e.startDate, &e.dueDate, &e.project} {
		if idx == i {
			ti.Focus()
		} else {
			ti.Blur()
		}
	}
}

func (e *epicOverlayState) activeInput() *textinput.Model {
	switch e.field {
	case 0:
		return &e.title
	case 1:
		return &e.body
	case 2:
		return &e.startDate
	case 3:
		return &e.dueDate
	default:
		return &e.project
	}
}

func (m *Model) openEpicOverlay() tea.Cmd {
	keys := m.selectedTicketKeys()
	if len(keys) == 0 {
		m.statusText = "Select tickets with space, then press E"
		return nil
	}
	draft := demo.EpicDraftForTickets(keys)
	project := demo.EpicProjectForKeys(keys)
	m.epicOverlay = newEpicOverlay(keys, draft, project)
	return textinput.Blink
}

func (m *Model) handleEpicOverlayKey(msg tea.KeyMsg) (*Model, tea.Cmd) {
	e := m.epicOverlay
	if e == nil || !e.open {
		return m, nil
	}
	switch {
	case key.Matches(msg, m.keys.Back):
		m.epicOverlay = nil
		return m, nil
	case key.Matches(msg, m.keys.ListOpen):
		return m, m.queueEpicActions()
	case msg.String() == "tab":
		e.focusField((e.field + 1) % 5)
		return m, textinput.Blink
	case msg.String() == "shift+tab":
		e.focusField((e.field + 4) % 5)
		return m, textinput.Blink
	}
	ti := e.activeInput()
	var cmd tea.Cmd
	*ti, cmd = ti.Update(msg)
	return m, cmd
}

func (m *Model) queueEpicActions() tea.Cmd {
	e := m.epicOverlay
	if e == nil {
		return nil
	}
	project := strings.TrimSpace(e.project.Value())
	title := strings.TrimSpace(e.title.Value())
	if project == "" || title == "" {
		m.errText = "Epic needs a project and title"
		return nil
	}
	bullets := prompts.EpicDraftOutput{Body: e.body.Value()}.Bullets()
	now := time.Now()
	create := actions.Action{
		ID:      actions.NewID(actions.KindCreateEpic, now),
		Kind:    actions.KindCreateEpic,
		Source:  actions.SourceUser,
		Summary: fmt.Sprintf("create epic %q in %s", title, project),
		Epic: &actions.EpicPayload{
			Project:     project,
			Summary:     title,
			BodyBullets: bullets,
			StartDate:   strings.TrimSpace(e.startDate.Value()),
			DueDate:     strings.TrimSpace(e.dueDate.Value()),
			SetParentOn: append([]string(nil), e.keys...),
		},
	}
	now = now.Add(time.Nanosecond)
	var queued []actions.Action
	queued = append(queued, create)
	for _, k := range e.keys {
		queued = append(queued, actions.Action{
			ID:        actions.NewID(actions.KindSetParent, now),
			Kind:      actions.KindSetParent,
			TicketKey: k,
			Source:    actions.SourceUser,
			Summary:   fmt.Sprintf("%s: set parent (after epic create)", k),
			ParentKey: "(pending epic)",
		})
		now = now.Add(time.Nanosecond)
	}
	if m.queue != nil {
		m.queue.Add(queued...)
	}
	m.epicOverlay = nil
	m.multiSelect = nil
	m.statusText = fmt.Sprintf("Queued epic + %d parent actions", len(e.keys))
	return nil
}

func (m *Model) renderEpicOverlay(base string) string {
	e := m.epicOverlay
	if e == nil || !e.open {
		return base
	}
	labels := []string{"Title", "Body", "Start", "Due", "Project"}
	inputs := []string{e.title.View(), e.body.View(), e.startDate.View(), e.dueDate.View(), e.project.View()}
	var lines []string
	lines = append(lines, styles.BoldStyle.Render("Create epic from selection"))
	lines = append(lines, styles.DimStyle.Render(fmt.Sprintf("%d tickets: %s", len(e.keys), strings.Join(e.keys, ", "))))
	for i, label := range labels {
		prefix := "  "
		if i == e.field {
			prefix = "▸ "
		}
		lines = append(lines, prefix+label+": "+inputs[i])
	}
	lines = append(lines, "")
	lines = append(lines, styles.DimStyle.Render("tab field · enter queue · esc cancel"))
	modal := overlays.ModalFrameSized(util.Clamp(m.width-8, 48, 90)).Render(strings.Join(lines, "\n"))
	top := max(2, (m.height-lipgloss.Height(modal))/2)
	left := max(2, (m.width-lipgloss.Width(modal))/2)
	return overlay.OverlayView(base, modal, m.width, m.height, top, left)
}

func (m *Model) toggleMultiSelect() {
	key := m.selectedKey()
	if key == "" {
		return
	}
	if m.multiSelect == nil {
		m.multiSelect = map[string]bool{}
	}
	if m.multiSelect[key] {
		delete(m.multiSelect, key)
	} else {
		m.multiSelect[key] = true
		m.rangeAnchor = m.selected
	}
}

func (m *Model) rangeMultiSelect() {
	if m.rangeAnchor < 0 {
		m.rangeAnchor = m.selected
	}
	if m.multiSelect == nil {
		m.multiSelect = map[string]bool{}
	}
	lo, hi := m.rangeAnchor, m.selected
	if lo > hi {
		lo, hi = hi, lo
	}
	for i := lo; i <= hi; i++ {
		if i < 0 || i >= len(m.rows) || m.rows[i].Kind != RowTicket {
			continue
		}
		m.multiSelect[m.rows[i].Data.Ticket.Key] = true
	}
}

func (m *Model) selectedTicketKeys() []string {
	if len(m.multiSelect) == 0 {
		if k := m.selectedKey(); k != "" {
			return []string{k}
		}
		return nil
	}
	var out []string
	for k, on := range m.multiSelect {
		if on {
			out = append(out, k)
		}
	}
	sortStrings(out)
	return out
}

func sortStrings(ss []string) {
	for i := 0; i < len(ss); i++ {
		for j := i + 1; j < len(ss); j++ {
			if ss[j] < ss[i] {
				ss[i], ss[j] = ss[j], ss[i]
			}
		}
	}
}

func (m *Model) isMultiSelected(key string) bool {
	return m.multiSelect != nil && m.multiSelect[key]
}
