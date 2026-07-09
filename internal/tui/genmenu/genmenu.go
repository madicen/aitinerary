// Package genmenu provides a reusable long-press popover for AI generate
// buttons across modals. Hold 500ms over a generate-button zone to pick a
// one-shot AI profile without changing the active default.
package genmenu

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"

	"github.com/madicen/aitinerary/internal/aiconfig"
	"github.com/madicen/aitinerary/internal/theme"
	"github.com/madicen/aitinerary/internal/tui/longpress"
	"github.com/madicen/aitinerary/internal/tui/mouse"
)

// LongPressThreshold is how long a left-button press must be held before the
// profile picker opens.
const LongPressThreshold = 500 * time.Millisecond

// MotionSlack re-exports longpress.MotionSlack for callers and tests.
const MotionSlack = longpress.MotionSlack

// TickMsg fires after LongPressThreshold to open the menu.
type TickMsg struct {
	Owner   string
	PressID int
}

// State holds long-press + menu visibility for one generate button.
type State struct {
	pressID     int
	pressActive bool
	pressOwner  string
	mouseX      int
	mouseY      int
	shown       bool
	hoverIdx    int
}

// IsShown reports whether the popover is visible.
func (s *State) IsShown() bool { return s != nil && s.shown }

// IsArmed reports whether a press is pending (before tick or release).
func (s *State) IsArmed() bool { return s != nil && s.pressActive }

// MouseAnchor returns the terminal coordinate for the popover anchor.
func (s *State) MouseAnchor() (int, int) {
	if s == nil {
		return 0, 0
	}
	return s.mouseX, s.mouseY
}

// HoverIndex returns the hovered row index, or -1.
func (s *State) HoverIndex() int {
	if s == nil {
		return -1
	}
	return s.hoverIdx
}

// Reset clears press and shown state.
func (s *State) Reset() {
	if s == nil {
		return
	}
	s.pressActive = false
	s.pressOwner = ""
	s.shown = false
	s.hoverIdx = -1
}

// BeginPress records a press and schedules TickMsg.
func (s *State) BeginPress(zoneID string, msg tea.MouseMsg) tea.Cmd {
	if s == nil {
		return nil
	}
	s.pressID++
	s.pressActive = true
	s.pressOwner = zoneID
	s.mouseX = msg.X
	s.mouseY = msg.Y
	s.hoverIdx = -1
	pressID := s.pressID
	owner := zoneID
	return tea.Tick(LongPressThreshold, func(time.Time) tea.Msg {
		return TickMsg{Owner: owner, PressID: pressID}
	})
}

// CancelPress drops a pending tick when not yet shown.
func (s *State) CancelPress() {
	if s == nil || s.shown {
		return
	}
	s.pressActive = false
	s.pressOwner = ""
}

// OnMotion keeps the press armed within slack tolerance.
func (s *State) OnMotion(zoneManager *zone.Manager, msg tea.MouseMsg) {
	if s == nil || s.shown || !s.pressActive {
		return
	}
	if longpress.StillArmed(zoneManager, s.pressOwner, s.mouseX, s.mouseY, msg) {
		return
	}
	s.pressActive = false
	s.pressOwner = ""
}

// OpenIfMatches opens the menu when the tick matches the active press.
func (s *State) OpenIfMatches(t TickMsg) bool {
	if s == nil {
		return false
	}
	if !s.pressActive || s.shown {
		return false
	}
	if t.Owner != s.pressOwner || t.PressID != s.pressID {
		return false
	}
	s.shown = true
	s.hoverIdx = -1
	return true
}

// UpdateHover hit-tests menu rows.
func (s *State) UpdateHover(zoneManager *zone.Manager, msg tea.MouseMsg, count int) {
	if s == nil || !s.shown || zoneManager == nil {
		return
	}
	hit := -1
	for i := 0; i < count; i++ {
		z := zoneManager.Get(mouse.ZoneGenMenuItem(i))
		if z != nil && z.InBounds(msg) {
			hit = i
			break
		}
	}
	s.hoverIdx = hit
}

// HitTestRelease returns the released row index and closes the menu.
func (s *State) HitTestRelease(zoneManager *zone.Manager, msg tea.MouseMsg, count int) int {
	if s == nil || !s.shown {
		return -1
	}
	hit := -1
	if zoneManager != nil {
		for i := 0; i < count; i++ {
			z := zoneManager.Get(mouse.ZoneGenMenuItem(i))
			if z != nil && z.InBounds(msg) {
				hit = i
				break
			}
		}
	}
	s.shown = false
	s.hoverIdx = -1
	s.pressActive = false
	s.pressOwner = ""
	return hit
}

// Close dismisses the popover (e.g. Esc).
func (s *State) Close() bool {
	if s == nil || !s.shown {
		return false
	}
	s.shown = false
	s.hoverIdx = -1
	s.pressActive = false
	s.pressOwner = ""
	return true
}

// Render returns the styled, zone-marked popover string.
func Render(zoneManager *zone.Manager, profiles []aiconfig.Profile, activeName string, hoverIdx int) string {
	menuBorder := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.Adaptive(theme.RoleAccent)).
		Padding(0, 1)

	rowStyle := lipgloss.NewStyle().Foreground(theme.Adaptive(theme.RoleOnSurface))
	dimStyle := lipgloss.NewStyle().Foreground(theme.Adaptive(theme.RoleMuted))
	hoverStyle := lipgloss.NewStyle().
		Foreground(theme.Adaptive(theme.RoleOnSurface)).
		Background(theme.Adaptive(theme.RoleAccent))
	hoverDim := lipgloss.NewStyle().
		Foreground(theme.Adaptive(theme.RoleMuted)).
		Background(theme.Adaptive(theme.RoleAccent))
	activeMark := lipgloss.NewStyle().Foreground(theme.Adaptive(theme.RoleSuccess))

	header := lipgloss.NewStyle().
		Foreground(theme.Adaptive(theme.RoleAccent)).
		Bold(true).
		Render("AI profile")

	rows := make([]string, 0, len(profiles))
	for i, p := range profiles {
		mark := "  "
		if strings.EqualFold(p.Name, activeName) {
			mark = activeMark.Render("● ")
		}
		isHover := i == hoverIdx
		nameStyle := rowStyle
		summaryStyle := dimStyle
		if isHover {
			nameStyle = hoverStyle
			summaryStyle = hoverDim
		}
		name := nameStyle.Render(fmt.Sprintf("%s%s", mark, p.Name))
		summary := summaryStyle.Render("  " + p.Summary())
		row := name + summary
		if zoneManager != nil {
			row = zoneManager.Mark(mouse.ZoneGenMenuItem(i), row)
		}
		rows = append(rows, row)
	}
	if len(rows) == 0 {
		empty := dimStyle.Render("  (no profiles configured)")
		return menuBorder.Render(header + "\n" + empty)
	}
	hint := dimStyle.Render("hold to open · click to use once")
	return menuBorder.Render(header + "\n" + strings.Join(rows, "\n") + "\n" + hint)
}
