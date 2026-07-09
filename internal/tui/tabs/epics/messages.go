package epics

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/madicen/aitinerary/internal/actions"
	"github.com/madicen/aitinerary/internal/store"
)

// ActionAppliedMsg reports a successful actions-engine apply.
type ActionAppliedMsg struct {
	Action actions.Action
	Err    error
}

func applyActionCmd(engine *actions.Engine, action actions.Action) tea.Cmd {
	if engine == nil {
		return nil
	}
	return func() tea.Msg {
		_, err := engine.Apply(context.Background(), action)
		return ActionAppliedMsg{Action: action, Err: err}
	}
}

func refreshStoreCmd(st *store.Store) tea.Cmd {
	if st == nil {
		return nil
	}
	return st.RefreshCmd()
}
