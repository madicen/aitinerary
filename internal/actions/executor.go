package actions

import (
	"context"

	"github.com/madicen/aitinerary/internal/jira"
)

// Executor is the Jira-facing surface the engine drives. It is an interface so
// the engine can be unit-tested with a fake, per the hermetic-tests convention.
type Executor interface {
	// GetTicket fetches current state, used to capture prior values for undo.
	GetTicket(ctx context.Context, key string) (jira.Ticket, error)

	Transition(ctx context.Context, key, toStatus string) error
	SetParent(ctx context.Context, key, parentKey string) error
	AssignSprint(ctx context.Context, sprintID int, keys []string) error
	SetDueDate(ctx context.Context, key, isoDate string) error
	SetStartDate(ctx context.Context, key, isoDate string) error
	SetLabels(ctx context.Context, key string, add, remove []string) error
	AddComment(ctx context.Context, key, text string) error
	CreateEpic(ctx context.Context, spec jira.EpicSpec) (string, error)

	// ListTransitions backs undo: it lets the engine find a reverse transition
	// to the prior status.
	ListTransitions(ctx context.Context, key string) ([]jira.Transition, error)
}

// JiraExecutor adapts a *jira.Client to the Executor interface.
type JiraExecutor struct{ C *jira.Client }

// NewJiraExecutor wraps c.
func NewJiraExecutor(c *jira.Client) *JiraExecutor { return &JiraExecutor{C: c} }

func (e *JiraExecutor) GetTicket(ctx context.Context, key string) (jira.Ticket, error) {
	t, err := e.C.GetIssue(ctx, key)
	if err != nil {
		return jira.Ticket{}, err
	}
	return *t, nil
}

func (e *JiraExecutor) Transition(ctx context.Context, key, toStatus string) error {
	return e.C.TransitionTo(ctx, key, toStatus)
}

func (e *JiraExecutor) SetParent(ctx context.Context, key, parentKey string) error {
	_, err := e.C.SetParent(ctx, key, parentKey)
	return err
}

func (e *JiraExecutor) AssignSprint(ctx context.Context, sprintID int, keys []string) error {
	return e.C.AssignToSprint(ctx, sprintID, keys)
}

func (e *JiraExecutor) SetDueDate(ctx context.Context, key, isoDate string) error {
	return e.C.SetDueDate(ctx, key, isoDate)
}

func (e *JiraExecutor) SetStartDate(ctx context.Context, key, isoDate string) error {
	return e.C.SetStartDate(ctx, key, isoDate)
}

func (e *JiraExecutor) SetLabels(ctx context.Context, key string, add, remove []string) error {
	return e.C.SetLabels(ctx, key, add, remove)
}

func (e *JiraExecutor) AddComment(ctx context.Context, key, text string) error {
	return e.C.AddComment(ctx, key, jira.TextToADF(text))
}

func (e *JiraExecutor) CreateEpic(ctx context.Context, spec jira.EpicSpec) (string, error) {
	return e.C.CreateEpic(ctx, spec)
}

func (e *JiraExecutor) ListTransitions(ctx context.Context, key string) ([]jira.Transition, error) {
	return e.C.ListTransitions(ctx, key)
}
