package styles

import (
	"github.com/charmbracelet/lipgloss"

	"github.com/madicen/aitinerary/internal/theme"
)

var (
	AppPadding = lipgloss.NewStyle().Padding(0, 1)

	HeaderBar = lipgloss.NewStyle().
			Bold(true).
			Foreground(theme.Adaptive(theme.RoleOnAccent)).
			Background(theme.Adaptive(theme.RoleAccent)).
			Padding(0, 1)

	StatusBar = lipgloss.NewStyle().
			Foreground(theme.Adaptive(theme.RoleMuted)).
			Padding(0, 1)

	TabActive = lipgloss.NewStyle().
			Bold(true).
			Foreground(theme.Adaptive(theme.RoleOnAccent)).
			Underline(true)

	TabInactive = lipgloss.NewStyle().
			Foreground(theme.Adaptive(theme.RoleOnAccent))
)
