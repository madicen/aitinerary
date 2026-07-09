package board

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/madicen/aitinerary/internal/actions"
	"github.com/madicen/aitinerary/internal/jira"
	"github.com/madicen/aitinerary/internal/theme"
	"github.com/madicen/aitinerary/internal/tui/mouse"
	"github.com/madicen/aitinerary/internal/tui/mousedouble"
	"github.com/madicen/aitinerary/internal/tui/styles"
	"github.com/madicen/aitinerary/internal/tui/util"
)

func (m *Model) renderDetail() string {
	t, ok := m.selectedTicket()
	if !ok {
		return styles.DimStyle.Render("Select a ticket to view details.")
	}
	td := m.byKey[t.Key]

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(theme.Adaptive(theme.RoleAccent))
	labelStyle := styles.DimStyle
	valueStyle := lipgloss.NewStyle()

	var lines []string
	lines = append(lines, titleStyle.Render(t.Key+" · "+t.Summary))
	lines = append(lines, labelStyle.Render("Status")+"  "+valueStyle.Render(t.Status))
	lines = append(lines, labelStyle.Render("Type")+"    "+valueStyle.Render(t.Type))
	if t.Priority != "" {
		lines = append(lines, labelStyle.Render("Priority")+" "+valueStyle.Render(t.Priority))
	}
	if t.Assignee != "" {
		lines = append(lines, labelStyle.Render("Assignee")+" "+valueStyle.Render(t.Assignee))
	}
	if t.ParentKey != "" {
		lines = append(lines, labelStyle.Render("Epic")+"    "+valueStyle.Render(t.ParentKey+" "+t.ParentSummary))
	}
	if t.Sprint != nil {
		lines = append(lines, labelStyle.Render("Sprint")+"  "+valueStyle.Render(t.Sprint.Name))
	}
	if t.DueDate != "" {
		lines = append(lines, labelStyle.Render("Due")+"     "+valueStyle.Render(t.DueDate))
	}
	if len(t.Labels) > 0 {
		lines = append(lines, labelStyle.Render("Labels")+"  "+valueStyle.Render(strings.Join(t.Labels, ", ")))
	}
	if t.Updated.IsZero() {
		lines = append(lines, labelStyle.Render("Updated")+" "+valueStyle.Render("—"))
	} else {
		lines = append(lines, labelStyle.Render("Updated")+" "+valueStyle.Render(t.Updated.Format("2006-01-02")))
	}

	if len(td.Findings) > 0 {
		var notes []string
		for _, f := range td.Findings {
			notes = append(notes, f.Detail)
		}
		lines = append(lines, labelStyle.Render("Hygiene")+" "+styles.ErrStyle.Render(strings.Join(notes, "; ")))
	}

	if len(td.Matches) > 0 {
		lines = append(lines, labelStyle.Render("Matched PRs"))
		for _, mt := range td.Matches {
			pr := mt.PR
			ref := fmt.Sprintf("#%d", pr.Number)
			if pr.Repo != "" {
				ref = pr.Repo + "#" + fmt.Sprint(pr.Number)
			}
			line := ref + " (" + mt.Confidence.String() + ")"
			if pr.URL != "" {
				line += " " + pr.URL
			}
			lines = append(lines, "  "+valueStyle.Render(line))
		}
	}

	if t.Description != "" {
		lines = append(lines, labelStyle.Render("Description"))
		lines = append(lines, renderDescription(t.Description, m.width-4))
	}

	lines = append(lines, m.renderEditAffordances(t)...)

	if m.loadingTransitions {
		lines = append(lines, styles.DimStyle.Render("Loading transitions…"))
	} else if len(m.transitionsFor) > 0 {
		var tr []string
		for _, trn := range m.transitionsFor {
			tr = append(tr, trn.Name+" → "+trn.ToStatus)
		}
		lines = append(lines, labelStyle.Render("Transitions")+" "+strings.Join(tr, " · "))
	}

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.Adaptive(theme.RoleBorderAccent)).
		Padding(0, 1).
		Render(strings.Join(lines, "\n"))
	return box
}

func renderDescription(desc string, width int) string {
	if width < 20 {
		width = 20
	}
	// Glamour is optional; plain text fallback keeps tests hermetic.
	return util.WrapForViewport(desc, width)
}

func (m *Model) renderEditAffordances(t jira.Ticket) []string {
	if m.editMode == editDueDate {
		return []string{m.dueInput.View()}
	}
	if m.editMode == editLabels {
		return []string{m.labelsInput.View()}
	}
	var lines []string
	sprintLine := "s sprint"
	if m.editMode == editSprint {
		sprintLine = m.sprintDD.TriggerView()
	}
	lines = append(lines, m.mark(mouse.ZoneBoardDetailSprint, sprintLine))

	dueVal := t.DueDate
	if dueVal == "" {
		dueVal = "—"
	}
	lines = append(lines, m.mark(mouse.ZoneBoardDetailDueDate, "d due "+dueVal))

	parent := t.ParentKey
	if parent == "" {
		parent = "—"
	}
	epicLine := "p epic " + parent
	if m.editMode == editParent {
		epicLine = m.epicDD.TriggerView()
	}
	lines = append(lines, m.mark(mouse.ZoneBoardDetailParent, epicLine))

	labels := strings.Join(t.Labels, ", ")
	if labels == "" {
		labels = "—"
	}
	lines = append(lines, m.mark(mouse.ZoneBoardDetailLabels, "l labels "+labels))

	trLine := "t transition"
	if m.editMode == editTransition {
		trLine = m.transitionDD.TriggerView()
	}
	lines = append(lines, m.mark(mouse.ZoneBoardDetailTransition, trLine))
	return lines
}

func (m *Model) handleZone(zoneID string, e tea.MouseMsg) (*Model, tea.Cmd) {
	if m.contextMenu != nil {
		return m.handleContextMenuZone(zoneID, e)
	}

	switch zoneID {
	case mouse.ZoneFilterAll:
		m.filters.Preset = FilterAll
		m.rebuildRows()
		m.updateStatusText()
		return m, nil
	case mouse.ZoneFilterNeedsAttention:
		m.filters.Preset = FilterNeedsAttention
		m.rebuildRows()
		m.updateStatusText()
		return m, nil
	case mouse.ZoneFilterDoneCandidates:
		m.filters.Preset = FilterDoneCandidates
		m.rebuildRows()
		m.updateStatusText()
		return m, nil
	}
	for _, ws := range m.chips {
		if zoneID == mouse.ZoneFilterWorkspace(ws.Chip) {
			if strings.EqualFold(m.filters.Workspace, ws.Chip) {
				m.filters.Workspace = ""
			} else {
				m.filters.Workspace = ws.Chip
			}
			m.rebuildRows()
			return m, nil
		}
	}
	for _, cat := range []string{"new", "indeterminate", "done"} {
		if zoneID == mouse.ZoneFilterStatus(cat) {
			if strings.EqualFold(m.filters.StatusCategory, cat) {
				m.filters.StatusCategory = ""
			} else {
				m.filters.StatusCategory = cat
			}
			m.rebuildRows()
			return m, nil
		}
	}

	if m.detailOpen {
		switch zoneID {
		case mouse.ZoneBoardDetailSprint:
			return m.beginEditSprint()
		case mouse.ZoneBoardDetailDueDate:
			return m.beginEditDueDate()
		case mouse.ZoneBoardDetailParent:
			return m.beginEditParent()
		case mouse.ZoneBoardDetailLabels:
			return m.beginEditLabels()
		case mouse.ZoneBoardDetailTransition:
			return m.beginEditTransition()
		}
	}

	ticketRows := m.ticketRowIndices()
	for vi, rowIdx := range ticketRows {
		if zoneID != mouse.ZoneBoardRow(vi) {
			continue
		}
		if e.Action == tea.MouseActionPress && e.Button == tea.MouseButtonLeft {
			mousedouble.OnLeftPress(&m.mousePressGen)
		}
		if e.Action == tea.MouseActionRelease && e.Button == tea.MouseButtonLeft {
			m.selected = rowIdx
			m.detailOpen = true
			m.scrollToSelected = true
			key := fmt.Sprintf("board:%d", vi)
			if m.rowDoubleClick.ObserveLeftRelease(key, e, time.Now(), mousedouble.DefaultDoubleClickWindow) {
				return m, m.openBrowserCmd()
			}
			return m, m.loadDetailCmds()
		}
	}
	return m, nil
}

func (m *Model) beginEditSprint() (*Model, tea.Cmd) {
	t, ok := m.selectedTicket()
	if !ok {
		return m, nil
	}
	proj := projectOf(t.Key)
	m.editMode = editSprint
	m.sprintDD.OnSelect = func(i int) tea.Cmd {
		return func() tea.Msg { return SprintSelectedMsg{Index: i} }
	}
	return m, loadSprintsCmd(m.sprints, proj)
}

func (m *Model) beginEditDueDate() (*Model, tea.Cmd) {
	t, ok := m.selectedTicket()
	if !ok {
		return m, nil
	}
	m.editMode = editDueDate
	m.dueInput.SetValue(t.DueDate)
	m.dueInput.Focus()
	return m, textinput.Blink
}

func (m *Model) beginEditParent() (*Model, tea.Cmd) {
	t, ok := m.selectedTicket()
	if !ok {
		return m, nil
	}
	proj := projectOf(t.Key)
	m.editMode = editParent
	m.epicDD.OnSelect = func(i int) tea.Cmd {
		return func() tea.Msg { return EpicSelectedMsg{Index: i} }
	}
	return m, loadEpicsCmd(m.epics, proj)
}

func (m *Model) beginEditLabels() (*Model, tea.Cmd) {
	t, ok := m.selectedTicket()
	if !ok {
		return m, nil
	}
	m.editMode = editLabels
	m.labelsInput.SetValue(strings.Join(t.Labels, ", "))
	m.labelsInput.Focus()
	return m, textinput.Blink
}

func (m *Model) beginEditTransition() (*Model, tea.Cmd) {
	if m.selectedKey() == "" {
		return m, nil
	}
	m.editMode = editTransition
	m.transitionDD.OnSelect = func(i int) tea.Cmd {
		return func() tea.Msg { return TransitionSelectedMsg{Index: i} }
	}
	if len(m.transitionsFor) == 0 {
		return m, m.loadDetailCmds()
	}
	m.rebuildTransitionDropdown()
	return m, nil
}

func (m *Model) rebuildSprintDropdown(sprints []jira.Sprint) {
	labels := make([]string, len(sprints))
	for i, s := range sprints {
		labels[i] = s.Name
	}
	m.sprintDD.Rebuild(labels, 0)
	m.sprintOptions = sprints
}

func (m *Model) rebuildEpicDropdown(epics []jira.Ticket) {
	labels := make([]string, len(epics))
	for i, e := range epics {
		labels[i] = e.Key + " " + e.Summary
	}
	m.epicDD.Rebuild(labels, 0)
	m.epicOptions = epics
}

func (m *Model) rebuildTransitionDropdown() {
	labels := make([]string, len(m.transitionsFor))
	for i, tr := range m.transitionsFor {
		labels[i] = tr.Name + " → " + tr.ToStatus
	}
	m.transitionDD.Rebuild(labels, 0)
}

func (m *Model) applySprintSelection(i int) tea.Cmd {
	if i < 0 || i >= len(m.sprintOptions) {
		return nil
	}
	key := m.selectedKey()
	if key == "" || m.engine == nil {
		return nil
	}
	sp := m.sprintOptions[i]
	action := actions.Action{
		ID:        actions.NewID(actions.KindSetSprint, time.Now()),
		Kind:      actions.KindSetSprint,
		TicketKey: key,
		Source:    actions.SourceUser,
		SprintID:  sp.ID,
		Summary:   key + ": assign sprint " + sp.Name,
	}
	return applyActionCmd(m.engine, action)
}

func (m *Model) applyEpicSelection(i int) tea.Cmd {
	if i < 0 || i >= len(m.epicOptions) {
		return nil
	}
	key := m.selectedKey()
	if key == "" || m.engine == nil {
		return nil
	}
	ep := m.epicOptions[i]
	action := actions.Action{
		ID:        actions.NewID(actions.KindSetParent, time.Now()),
		Kind:      actions.KindSetParent,
		TicketKey: key,
		Source:    actions.SourceUser,
		ParentKey: ep.Key,
		Summary:   key + ": set epic " + ep.Key,
	}
	return applyActionCmd(m.engine, action)
}

func (m *Model) applyTransitionSelection(i int) tea.Cmd {
	if i < 0 || i >= len(m.transitionsFor) {
		return nil
	}
	key := m.selectedKey()
	if key == "" || m.engine == nil {
		return nil
	}
	tr := m.transitionsFor[i]
	action := actions.Action{
		ID:        actions.NewID(actions.KindTransition, time.Now()),
		Kind:      actions.KindTransition,
		TicketKey: key,
		Source:    actions.SourceUser,
		ToStatus:  tr.ToStatus,
		Summary:   key + " → " + tr.ToStatus,
	}
	return applyActionCmd(m.engine, action)
}

func (m *Model) applyDueDate(iso string) (*Model, tea.Cmd) {
	key := m.selectedKey()
	if key == "" || m.engine == nil {
		return m, nil
	}
	action := actions.Action{
		ID:        actions.NewID(actions.KindSetDueDate, time.Now()),
		Kind:      actions.KindSetDueDate,
		TicketKey: key,
		Source:    actions.SourceUser,
		DueDate:   strings.TrimSpace(iso),
		Summary:   key + ": due " + iso,
	}
	return m, applyActionCmd(m.engine, action)
}

func (m *Model) applyLabels(raw string) (*Model, tea.Cmd) {
	key := m.selectedKey()
	if key == "" || m.engine == nil {
		return m, nil
	}
	var add []string
	for _, p := range strings.Split(raw, ",") {
		if p = strings.TrimSpace(p); p != "" {
			add = append(add, p)
		}
	}
	action := actions.Action{
		ID:        actions.NewID(actions.KindSetLabels, time.Now()),
		Kind:      actions.KindSetLabels,
		TicketKey: key,
		Source:    actions.SourceUser,
		Labels:    add,
		Summary:   key + ": set labels",
	}
	return m, applyActionCmd(m.engine, action)
}

func projectOf(key string) string {
	if i := strings.IndexByte(key, '-'); i > 0 {
		return key[:i]
	}
	return key
}
