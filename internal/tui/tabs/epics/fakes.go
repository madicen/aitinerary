package epics

import (
	"context"

	"github.com/madicen/aitinerary/internal/actions"
	"github.com/madicen/aitinerary/internal/config"
	"github.com/madicen/aitinerary/internal/demo"
	"github.com/madicen/aitinerary/internal/jira"
	"github.com/madicen/aitinerary/internal/store"
	"github.com/madicen/aitinerary/internal/tui/keys"
)

type fakeExecutor struct {
	ticket    jira.Ticket
	startDate string
	dueDate   string
}

func (f *fakeExecutor) GetTicket(_ context.Context, key string) (jira.Ticket, error) {
	t := f.ticket
	t.Key = key
	return t, nil
}
func (f *fakeExecutor) Transition(context.Context, string, string) error { return nil }
func (f *fakeExecutor) SetParent(context.Context, string, string) error  { return nil }
func (f *fakeExecutor) AssignSprint(context.Context, int, []string) error {
	return nil
}
func (f *fakeExecutor) SetDueDate(_ context.Context, _, iso string) error {
	f.dueDate = iso
	f.ticket.DueDate = iso
	return nil
}
func (f *fakeExecutor) SetStartDate(_ context.Context, _, iso string) error {
	f.startDate = iso
	f.ticket.StartDate = iso
	return nil
}
func (f *fakeExecutor) SetLabels(context.Context, string, []string, []string) error { return nil }
func (f *fakeExecutor) AddComment(context.Context, string, string) error            { return nil }
func (f *fakeExecutor) CreateEpic(context.Context, jira.EpicSpec) (string, error) {
	return "APP-500", nil
}
func (f *fakeExecutor) CreateTicket(context.Context, jira.CreateIssueSpec) (string, error) {
	return "APP-999", nil
}
func (f *fakeExecutor) RankIssues(context.Context, []string, string) error { return nil }
func (f *fakeExecutor) ListTransitions(context.Context, string) ([]jira.Transition, error) {
	return nil, nil
}

func demoEpicsModel(width, height int) (*Model, *actions.Queue, *fakeExecutor) {
	cfg := config.Default()
	st := store.New(cfg, nil, true)
	_ = st.RefreshCmd()()
	q := actions.NewQueue()
	fx := &fakeExecutor{ticket: demo.Epics()[0]}
	m := New(Opts{
		Store:  st,
		Keys:   keys.Default(),
		Queue:  q,
		Engine: actions.NewEngine(fx, nil),
	})
	m.Resize(width, height)
	m.SetContentOrigin(2)
	m.SetNow(demo.Epics()[0].Updated)
	return m, q, fx
}
