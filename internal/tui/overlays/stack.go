// Package overlays owns modal dialogs the root TUI stacks on top of the active screen.
package overlays

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/madicen/aitinerary/internal/theme"
	"github.com/madicen/aitinerary/internal/tui/util"
)

// DismissMsg is emitted when the user dismisses a modal overlay.
type DismissMsg struct {
	Result any
}

// ConfirmAnswer is the DismissMsg.Result payload from ConfirmOverlay.
type ConfirmAnswer struct {
	Confirm bool
}

func dismiss(result any) tea.Cmd {
	return func() tea.Msg { return DismissMsg{Result: result} }
}

// ModalChrome is the shared border + padding for every modal.
var ModalChrome = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(theme.Adaptive(theme.RoleAccent)).
	Padding(1, 2)

// ModalFrameSized returns ModalChrome sized to outerWidth.
func ModalFrameSized(outerWidth int) lipgloss.Style {
	return ModalChrome.Width(util.Clamp(outerWidth, 40, 120))
}

var (
	confirmYesKeys = key.NewBinding(key.WithKeys("y", "Y", "enter"))
	confirmNoKeys  = key.NewBinding(key.WithKeys("n", "N", "esc", "q"))
)
