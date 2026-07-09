package settings

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"
	bubblepicker "github.com/madicen/bubble-color-picker"

	"github.com/madicen/aitinerary/internal/theme"
	"github.com/madicen/aitinerary/internal/tui/styles"
)

type colorSlot struct {
	slot   string
	label  string
	swatch *bubblepicker.SwatchPicker
	row    int
	col    int
	zoneID string
}

type themePanel struct {
	draft    *theme.Theme
	swatches []colorSlot
	focus    int
}

func newThemePanel() *themePanel {
	draft := theme.Current()
	p := &themePanel{draft: draft.Clone(), focus: 0}
	slots := []struct {
		slot, label string
	}{
		{"primary", "Primary"},
		{"secondary", "Secondary"},
		{"muted", "Muted"},
	}
	for i, s := range slots {
		sw := bubblepicker.NewSwatchPicker(draft.Color(s.slot), s.label)
		sw.SetZoneManager(zone.DefaultManager)
		sw.SetPickerOptions(bubblepicker.WithAutoDismiss(true))
		if i == 0 {
			sw.SetFocused(true)
		}
		p.swatches = append(p.swatches, colorSlot{
			slot:   s.slot,
			label:  s.label,
			swatch: sw,
			zoneID: fmt.Sprintf("zone:settings:theme:%s", s.slot),
		})
	}
	return p
}

func (p *themePanel) openIndex() int {
	for i, sw := range p.swatches {
		if sw.swatch.Open() {
			return i
		}
	}
	return -1
}

func (p *themePanel) advanceFocus(delta int) {
	if len(p.swatches) == 0 {
		return
	}
	p.swatches[p.focus].swatch.SetFocused(false)
	p.focus = (p.focus + delta + len(p.swatches)) % len(p.swatches)
	p.swatches[p.focus].swatch.SetFocused(true)
}

func (p *themePanel) openFocused() tea.Cmd {
	if len(p.swatches) == 0 {
		return nil
	}
	sw := p.swatches[p.focus]
	press := tea.MouseMsg{Button: tea.MouseButtonLeft, Action: tea.MouseActionPress}
	updated, cmd := sw.swatch.Update(press)
	sw.swatch = updated
	p.swatches[p.focus] = sw
	return cmd
}

func (p *themePanel) applyColor(hex string) {
	idx := p.openIndex()
	if idx < 0 {
		idx = p.focus
	}
	if idx < 0 || idx >= len(p.swatches) {
		return
	}
	slot := p.swatches[idx].slot
	p.draft.Set(slot, hex)
	p.swatches[idx].swatch.SetColor(p.draft.Color(slot))
}

func (p *themePanel) resetAll() {
	p.draft = theme.Default()
	for i, sw := range p.swatches {
		sw.swatch.SetColor(p.draft.Color(sw.slot))
		p.swatches[i] = sw
	}
}

func (p *themePanel) saveCmd() tea.Cmd {
	return func() tea.Msg {
		_ = theme.Save(p.draft, "")
		theme.Apply(p.draft)
		return nil
	}
}

func (p *themePanel) handleMouse(msg tea.MouseMsg) (tea.Cmd, bool) {
	for i := range p.swatches {
		sw := &p.swatches[i]
		if z := zone.Get(sw.zoneID); z != nil && z.InBounds(msg) {
			p.focus = i
			for j := range p.swatches {
				p.swatches[j].swatch.SetFocused(j == i)
			}
			updated, cmd := sw.swatch.Update(msg)
			sw.swatch = updated
			return cmd, true
		}
	}
	return nil, false
}

func (p *themePanel) render(width int) string {
	const labelW = 12
	var b strings.Builder
	b.WriteString(styles.DimStyle.Render("Theme colours · click swatch or press enter"))
	b.WriteString("\n\n")
	row := 2
	for i, sw := range p.swatches {
		hex := p.draft.Color(sw.slot)
		sw.swatch.SetColor(hex)
		sw.row = row
		sw.col = labelW + 2
		sw.swatch.SetBounds(sw.row, sw.col, 0, 0)
		p.swatches[i] = sw

		focus := "  "
		if i == p.focus {
			focus = styles.OkStyle.Render("▶ ")
		}
		line := fmt.Sprintf("%s%-12s  %s  %s", focus, sw.label, sw.swatch.SwatchView(), hex)
		b.WriteString(zone.Mark(sw.zoneID, line))
		b.WriteString("\n")
		row++
	}
	b.WriteString("\n")
	b.WriteString(styles.DimStyle.Render("ctrl+s save · r reset · tab next swatch"))
	_ = width
	return b.String()
}

func (p *themePanel) applyOverlays(panel string, width, height int) string {
	for _, sw := range p.swatches {
		if sw.swatch.Open() {
			panel = sw.swatch.ViewWithOverlay(panel, width, height)
		}
	}
	return panel
}

func padRight(s string, w int) string {
	if lipgloss.Width(s) >= w {
		return s
	}
	return s + strings.Repeat(" ", w-lipgloss.Width(s))
}
