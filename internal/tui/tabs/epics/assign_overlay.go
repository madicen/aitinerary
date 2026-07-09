package epics

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	overlay "github.com/madicen/bubble-overlay"

	"github.com/madicen/aitinerary/internal/actions"
	"github.com/madicen/aitinerary/internal/jira"
	"github.com/madicen/aitinerary/internal/tui/overlays"
	"github.com/madicen/aitinerary/internal/tui/styles"
	"github.com/madicen/aitinerary/internal/tui/util"
)

type assignOverlayState struct {
	open       bool
	epicKey    string
	project    string
	tickets    []jira.Ticket
	selected   int
	multiSelect map[string]bool
}

func (m *Model) openAssignOverlay() (*Model, tea.Cmd) {
	row, ok := m.selectedEpic()
	if !ok {
		return m, nil
	}
	tickets := m.unparentedTickets(row.Project)
	if len(tickets) == 0 {
		return m, nil
	}
	m.assignOverlay = &assignOverlayState{
		open:        true,
		epicKey:     row.Epic.Key,
		project:     row.Project,
		tickets:     tickets,
		selected:    0,
		multiSelect: map[string]bool{},
	}
	return m, nil
}

func (m *Model) unparentedTickets(project string) []jira.Ticket {
	if m.store == nil {
		return nil
	}
	var out []jira.Ticket
	for _, t := range m.store.Tickets() {
		if t.IsDone() || strings.EqualFold(t.Type, "Epic") || t.IsSubtask {
			continue
		}
		if projectOf(t.Key) != project {
			continue
		}
		if t.ParentKey != "" {
			continue
		}
		out = append(out, t)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out
}

func (m *Model) handleAssignOverlayKey(msg tea.KeyMsg) (*Model, tea.Cmd) {
	ao := m.assignOverlay
	if ao == nil || !ao.open {
		return m, nil
	}
	switch {
	case key.Matches(msg, m.keys.Back):
		m.assignOverlay = nil
		return m, nil
	case key.Matches(msg, m.keys.ListOpen):
		return m, m.queueAssignActions()
	case msg.String() == " ":
		if ao.selected >= 0 && ao.selected < len(ao.tickets) {
			k := ao.tickets[ao.selected].Key
			if ao.multiSelect[k] {
				delete(ao.multiSelect, k)
			} else {
				ao.multiSelect[k] = true
			}
		}
		return m, nil
	case msg.String() == "j", msg.String() == "down":
		if len(ao.tickets) > 0 {
			ao.selected = (ao.selected + 1) % len(ao.tickets)
		}
		return m, nil
	case msg.String() == "k", msg.String() == "up":
		if len(ao.tickets) > 0 {
			ao.selected = (ao.selected - 1 + len(ao.tickets)) % len(ao.tickets)
		}
		return m, nil
	}
	return m, nil
}

func (m *Model) queueAssignActions() tea.Cmd {
	ao := m.assignOverlay
	if ao == nil || m.queue == nil {
		return nil
	}
	var keys []string
	for k, on := range ao.multiSelect {
		if on {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	if len(keys) == 0 {
		return nil
	}
	now := time.Now()
	var queued []actions.Action
	for _, k := range keys {
		queued = append(queued, actions.Action{
			ID:        actions.NewID(actions.KindSetParent, now),
			Kind:      actions.KindSetParent,
			TicketKey: k,
			Source:    actions.SourceUser,
			ParentKey: ao.epicKey,
			Summary:   fmt.Sprintf("%s: set epic %s", k, ao.epicKey),
		})
		now = now.Add(time.Nanosecond)
	}
	m.queue.Add(queued...)
	m.assignOverlay = nil
	return nil
}

func (m *Model) renderAssignOverlay(base string) string {
	ao := m.assignOverlay
	if ao == nil || !ao.open {
		return base
	}
	var lines []string
	lines = append(lines, styles.BoldStyle.Render("Assign tickets to "+ao.epicKey))
	lines = append(lines, styles.DimStyle.Render("space toggle · enter queue · esc cancel"))
	for i, t := range ao.tickets {
		prefix := "  "
		if i == ao.selected {
			prefix = "▸ "
		}
		mark := "[ ]"
		if ao.multiSelect[t.Key] {
			mark = "[x]"
		}
		lines = append(lines, prefix+mark+" "+t.Key+"  "+t.Summary)
	}
	modal := overlays.ModalFrameSized(util.Clamp(m.width-8, 48, 90)).Render(strings.Join(lines, "\n"))
	top := max(2, (m.height-lipgloss.Height(modal))/2)
	left := max(2, (m.width-lipgloss.Width(modal))/2)
	return overlay.OverlayView(base, modal, m.width, m.height, top, left)
}
