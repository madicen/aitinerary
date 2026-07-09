package state

import tea "github.com/charmbracelet/bubbletea"

// NavigateKind identifies a navigation event emitted by a sub-model.
type NavigateKind int

const (
	// NavBack closes the current overlay or tab context and returns to the
	// previous mode. Root owns the transition; sub-models only emit intent.
	NavBack NavigateKind = iota
)

// NavigateTarget bundles parameters for a navigation request. Payload is an
// optional opaque value for the root to interpret (saved config, selection,
// etc.) without this package importing higher layers.
type NavigateTarget struct {
	Kind NavigateKind

	// Payload carries modal-specific results when non-nil.
	Payload any

	// Err, when non-nil, asks root to surface an error overlay instead of
	// changing modes.
	Err error

	// Cancelled distinguishes a dismiss-without-commit from a successful back.
	Cancelled bool
}

// NavigateMsg is the only callback from sub-models to root for view changes
// only root can perform.
type NavigateMsg struct {
	Target NavigateTarget
}

// Cmd returns a tea.Cmd that sends this navigation to the root model.
func (t NavigateTarget) Cmd() tea.Cmd {
	return func() tea.Msg { return NavigateMsg{Target: t} }
}
