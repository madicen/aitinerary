// Package dropdown wraps madicen/bubble-dropdown in a Host that owns rebuild-on-
// options-change, Y-rebasing for mouse hit-tests, OnSelect, and overlay compositing.
package dropdown

import (
	tea "github.com/charmbracelet/bubbletea"
	zone "github.com/lrstanley/bubblezone"
	bubbledropdown "github.com/madicen/bubble-dropdown"
)

// Host owns one bubble-dropdown pointer plus integration glue.
type Host struct {
	dd          *bubbledropdown.Dropdown
	placeholder string

	// ContentTop is the absolute terminal row where the owning body begins.
	ContentTop int

	// OnSelect runs after the selection index changes (keyboard or mouse).
	OnSelect func(index int) tea.Cmd
}

// New returns a Host that builds dropdowns with the given placeholder.
func New(placeholder string) *Host { return &Host{placeholder: placeholder} }

// Rebuild recreates the dropdown from labels with selected pre-selected.
func (h *Host) Rebuild(labels []string, selected int) {
	if h == nil || (h.dd != nil && h.dd.Open()) {
		return
	}
	if selected < 0 || selected >= len(labels) {
		selected = 0
	}
	d := bubbledropdown.New(
		bubbledropdown.WithOptions(labels),
		bubbledropdown.WithInitialIndex(selected),
		bubbledropdown.WithPlaceholder(h.placeholder),
	)
	d.SetZoneManager(zone.DefaultManager)
	h.dd = d
}

// Clear drops the dropdown.
func (h *Host) Clear() {
	if h != nil {
		h.dd = nil
	}
}

// Built reports whether a dropdown has been created.
func (h *Host) Built() bool { return h != nil && h.dd != nil }

// Open reports whether the panel is currently displayed.
func (h *Host) Open() bool { return h != nil && h.dd != nil && h.dd.Open() }

// SelectedIndex returns the selected option index (0 when not built).
func (h *Host) SelectedIndex() int {
	if h == nil || h.dd == nil {
		return 0
	}
	return h.dd.SelectedIndex()
}

// SetSelectedIndex updates the displayed selection.
func (h *Host) SetSelectedIndex(i int) {
	if h != nil && h.dd != nil {
		h.dd.SetSelectedIndex(i)
	}
}

// SetFocused toggles the trigger's focused indicator.
func (h *Host) SetFocused(f bool) {
	if h != nil && h.dd != nil {
		h.dd.SetFocused(f)
	}
}

// TriggerView renders the closed-state trigger.
func (h *Host) TriggerView() string {
	if h == nil || h.dd == nil {
		return ""
	}
	return h.dd.TriggerView()
}

// Forward routes msg to the dropdown, rebasing mouse Y by ContentTop.
func (h *Host) Forward(msg tea.Msg) tea.Cmd {
	if h == nil || h.dd == nil {
		return nil
	}
	if mm, ok := msg.(tea.MouseMsg); ok && h.ContentTop != 0 {
		mm.Y -= h.ContentTop
		msg = mm
	}
	prev := h.dd.SelectedIndex()
	updated, cmd := h.dd.Update(msg)
	h.dd = updated
	if h.OnSelect != nil {
		if sel := h.dd.SelectedIndex(); sel != prev {
			if c := h.OnSelect(sel); c != nil {
				cmd = tea.Batch(cmd, c)
			}
		}
	}
	return cmd
}

// Composite draws the open panel over main at (row, col).
func (h *Host) Composite(main string, row, col, viewW, viewH int) string {
	if h == nil || h.dd == nil || !h.dd.Open() {
		return main
	}
	tw, th := h.dd.TriggerSize()
	h.dd.SetBounds(row, col, tw, th)
	return h.dd.ViewWithOverlay(main, viewW, viewH)
}
