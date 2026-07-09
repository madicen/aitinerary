// Package styles owns lipgloss styles sourced from internal/theme.
package styles

import (
	"github.com/charmbracelet/lipgloss"

	"github.com/madicen/aitinerary/internal/theme"
)

var (
	DimStyle = lipgloss.NewStyle().Foreground(theme.Adaptive(theme.RoleMuted))
	BoldStyle = lipgloss.NewStyle().Bold(true)
	ErrStyle  = lipgloss.NewStyle().Foreground(theme.Adaptive(theme.RoleError)).Bold(true)
	OkStyle   = lipgloss.NewStyle().Foreground(theme.Adaptive(theme.RoleSuccess))
	WarnStyle = lipgloss.NewStyle().Foreground(theme.Adaptive(theme.RoleWarning))
	ModalButtonStyle = lipgloss.NewStyle().
		Foreground(theme.Adaptive(theme.RoleOnSurface)).
		Background(theme.Adaptive(theme.RoleSurface)).
		Padding(0, 1).
		Bold(true)
)
