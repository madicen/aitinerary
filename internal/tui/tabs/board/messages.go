package board

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/aitinerary/internal/actions"
	"github.com/madicen/aitinerary/internal/jira"
	"github.com/madicen/aitinerary/internal/store"
)

// SprintSelectedMsg is emitted when the sprint dropdown picks an option.
type SprintSelectedMsg struct{ Index int }

// EpicSelectedMsg is emitted when the epic dropdown picks an option.
type EpicSelectedMsg struct{ Index int }

// TransitionSelectedMsg is emitted when the transition dropdown picks an option.
type TransitionSelectedMsg struct{ Index int }

// LongPressTickMsg fires after the long-press threshold to show the row context menu.
type LongPressTickMsg struct {
	PressID int
}

// TransitionsLoadedMsg delivers available workflow transitions for the selected ticket.
type TransitionsLoadedMsg struct {
	Key          string
	Transitions  []jira.Transition
	Err          error
}

// ActionAppliedMsg reports a successful actions-engine apply.
type ActionAppliedMsg struct {
	Action actions.Action
	Err    error
}

// SprintsLoadedMsg delivers sprint options for a project board.
type SprintsLoadedMsg struct {
	Project string
	Sprints []jira.Sprint
	Err     error
}

// EpicsLoadedMsg delivers epic options for a project.
type EpicsLoadedMsg struct {
	Project string
	Epics   []jira.Ticket
	Err     error
}

// StatusTextMsg lets the root model read the board status-bar summary.
type StatusTextMsg struct {
	Text string
}

// TransitionLister loads workflow transitions for a ticket key.
type TransitionLister interface {
	ListTransitions(ctx context.Context, key string) ([]jira.Transition, error)
}

// SprintLister loads active/future sprints for a project.
type SprintLister interface {
	ListSprints(ctx context.Context, project string) ([]jira.Sprint, error)
}

// EpicLister loads epics for a project.
type EpicLister interface {
	ListEpics(ctx context.Context, project string) ([]jira.Ticket, error)
}

func loadTransitionsCmd(lister TransitionLister, key string) tea.Cmd {
	if lister == nil || key == "" {
		return nil
	}
	return func() tea.Msg {
		tr, err := lister.ListTransitions(context.Background(), key)
		return TransitionsLoadedMsg{Key: key, Transitions: tr, Err: err}
	}
}

func loadSprintsCmd(lister SprintLister, project string) tea.Cmd {
	if lister == nil || project == "" {
		return nil
	}
	return func() tea.Msg {
		sp, err := lister.ListSprints(context.Background(), project)
		return SprintsLoadedMsg{Project: project, Sprints: sp, Err: err}
	}
}

func loadEpicsCmd(lister EpicLister, project string) tea.Cmd {
	if lister == nil || project == "" {
		return nil
	}
	return func() tea.Msg {
		ep, err := lister.ListEpics(context.Background(), project)
		return EpicsLoadedMsg{Project: project, Epics: ep, Err: err}
	}
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

func demoNow() time.Time {
	return time.Date(2026, 6, 15, 10, 0, 0, 0, time.UTC)
}
