package board

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	zone "github.com/lrstanley/bubblezone"
	overlay "github.com/madicen/bubble-overlay"

	"github.com/madicen/aitinerary/internal/aiconfig"
	"github.com/madicen/aitinerary/internal/demo"
	"github.com/madicen/aitinerary/internal/prompts"
	"github.com/madicen/aitinerary/internal/tui/genmenu"
	"github.com/madicen/aitinerary/internal/tui/mouse"
	"github.com/madicen/aitinerary/internal/tui/styles"
)

func (m *Model) aiProfiles() []aiconfig.Profile {
	if m.aiCfg != nil {
		return m.aiCfg.ProfileList()
	}
	return aiconfig.DefaultConfig().ProfileList()
}

func (m *Model) activeAIProfileName() string {
	if m.aiCfg != nil {
		return m.aiCfg.Active().Name
	}
	return aiconfig.DefaultProfileName
}

func (m *Model) regenerateEpicDraft(profileName string) tea.Cmd {
	if m.epicOverlay == nil || !m.epicOverlay.open {
		return nil
	}
	m.genReqID++
	reqID := m.genReqID
	keys := append([]string(nil), m.epicOverlay.keys...)
	return func() tea.Msg {
		draft := demo.EpicDraftForTickets(keys)
		return EpicDraftGeneratedMsg{ReqID: reqID, Profile: profileName, Draft: draft}
	}
}

func (m *Model) applyEpicDraft(draft prompts.EpicDraftOutput) {
	e := m.epicOverlay
	if e == nil {
		return
	}
	e.title.SetValue(draft.Title)
	e.body.SetValue(draft.Body)
	e.startDate.SetValue(draft.StartDate)
	e.dueDate.SetValue(draft.DueDate)
}

func (m *Model) handleEpicGenMenuMsg(msg tea.Msg) (*Model, tea.Cmd) {
	switch msg := msg.(type) {
	case genmenu.TickMsg:
		if m.epicOverlay == nil || !m.epicOverlay.open {
			return m, nil
		}
		if m.genMenu.OpenIfMatches(msg) {
			return m, nil
		}
	case EpicDraftGeneratedMsg:
		if m.epicOverlay == nil || msg.ReqID != m.genReqID {
			return m, nil
		}
		m.applyEpicDraft(msg.Draft)
		m.statusText = fmt.Sprintf("Epic draft regenerated (%s, one-shot)", msg.Profile)
		return m, nil
	}
	return m, nil
}

func (m *Model) handleEpicGenMenuMouse(msg tea.MouseMsg) (*Model, tea.Cmd) {
	if m.epicOverlay == nil || !m.epicOverlay.open {
		return m, nil
	}
	profiles := m.aiProfiles()
	if m.genMenu.IsShown() {
		m.genMenu.UpdateHover(m.zone, msg, len(profiles))
		if msg.Action == tea.MouseActionRelease && msg.Button == tea.MouseButtonLeft {
			if hit := m.genMenu.HitTestRelease(m.zone, msg, len(profiles)); hit >= 0 && hit < len(profiles) {
				name := profiles[hit].Name
				m.statusText = fmt.Sprintf("Regenerating epic draft (%s, one-shot)", name)
				return m, m.regenerateEpicDraft(name)
			}
		}
		return m, nil
	}
	if msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft {
		if z := m.zone.Get(mouse.ZoneBoardGenerate); z != nil && z.InBounds(msg) {
			return m, m.genMenu.BeginPress(mouse.ZoneBoardGenerate, msg)
		}
	}
	if m.genMenu.IsArmed() {
		m.genMenu.OnMotion(m.zone, msg)
	}
	if msg.Action == tea.MouseActionRelease && msg.Button == tea.MouseButtonLeft && m.genMenu.IsArmed() {
		if z := m.zone.Get(mouse.ZoneBoardGenerate); z != nil && z.InBounds(msg) {
			name := m.activeAIProfileName()
			m.genMenu.CancelPress()
			m.statusText = fmt.Sprintf("Regenerating epic draft (%s, one-shot)", name)
			return m, m.regenerateEpicDraft(name)
		}
		m.genMenu.CancelPress()
	}
	return m, nil
}

func (m *Model) renderEpicGenMenu(base string) string {
	if m.epicOverlay == nil || !m.epicOverlay.open || !m.genMenu.IsShown() {
		return base
	}
	profiles := m.aiProfiles()
	menu := genmenu.Render(m.zone, profiles, m.activeAIProfileName(), m.genMenu.HoverIndex())
	mx, my := m.genMenu.MouseAnchor()
	return overlay.OverlayViewAtPoint(base, menu, m.width, m.height, my, mx)
}

func (m *Model) markGenerateButton(label string) string {
	line := styles.DimStyle.Render(label)
	if m.zone != nil {
		line = m.zone.Mark(mouse.ZoneBoardGenerate, line)
	}
	return line
}

func (m *Model) epicOverlayZoneIDs() []string {
	if m.epicOverlay == nil || !m.epicOverlay.open {
		return nil
	}
	ids := []string{mouse.ZoneBoardGenerate}
	for i := range m.aiProfiles() {
		if m.genMenu.IsShown() {
			ids = append(ids, mouse.ZoneGenMenuItem(i))
		}
	}
	return ids
}

func (m *Model) resolveEpicOverlayZone(msg zone.MsgZoneInBounds) string {
	if msg.Zone == nil || m.zone == nil {
		return ""
	}
	for _, id := range m.epicOverlayZoneIDs() {
		if z := m.zone.Get(id); z != nil && z.InBounds(msg.Event) {
			return id
		}
	}
	return ""
}
