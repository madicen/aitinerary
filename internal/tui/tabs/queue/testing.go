package queue

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/madicen/aitinerary/internal/actions"
	"github.com/madicen/aitinerary/internal/jira"
	"github.com/madicen/aitinerary/internal/tui/keys"
	"github.com/madicen/aitinerary/internal/tui/tuitest"
)

type testExecutor struct {
	ticket      jira.Ticket
	transition  string
	comments    []string
	failTransit bool
}

func (f *testExecutor) GetTicket(_ context.Context, _ string) (jira.Ticket, error) {
	return f.ticket, nil
}
func (f *testExecutor) Transition(_ context.Context, _, to string) error {
	if f.failTransit {
		return errors.New("no path")
	}
	f.transition = to
	return nil
}
func (f *testExecutor) SetParent(context.Context, string, string) error     { return nil }
func (f *testExecutor) AssignSprint(context.Context, int, []string) error   { return nil }
func (f *testExecutor) SetDueDate(context.Context, string, string) error    { return nil }
func (f *testExecutor) SetStartDate(context.Context, string, string) error  { return nil }
func (f *testExecutor) SetLabels(context.Context, string, []string, []string) error {
	return nil
}
func (f *testExecutor) AddComment(_ context.Context, _, text string) error {
	f.comments = append(f.comments, text)
	return nil
}
func (f *testExecutor) CreateEpic(context.Context, jira.EpicSpec) (string, error) {
	return "APP-500", nil
}
func (f *testExecutor) ListTransitions(context.Context, string) ([]jira.Transition, error) {
	return nil, nil
}

func testModel(t *testing.T, pending ...actions.Action) (*Model, *actions.Queue) {
	t.Helper()
	tuitest.ForceMonochrome(t)
	fx := &testExecutor{ticket: jira.Ticket{Key: "APP-1", Status: "In Progress"}}
	eng := actions.NewEngine(fx, nil)
	q := actions.NewQueue()
	q.Add(pending...)
	m := New(Opts{Queue: q, Engine: eng, Keys: keys.Default(), Demo: true})
	m.rebuildFlat()
	m.Resize(100, 30)
	m.SetContentOrigin(2)
	return m, q
}

func demoPending() []actions.Action {
	now := time.Date(2026, 6, 15, 10, 0, 0, 0, time.UTC)
	return []actions.Action{
		{
			ID: "t1", Kind: actions.KindTransition, TicketKey: "APP-101",
			Source: actions.SourceAuto, Confidence: actions.ConfAmbiguous,
			ToStatus: "Done", Summary: "APP-101 → Done",
		},
		{
			ID: "c1", Kind: actions.KindAddComment, TicketKey: "APP-102",
			Source: actions.SourceAI, Summary: "status comment",
			Comment: "Merged webhook retry PR; ready for review.",
		},
		{
			ID: "c2", Kind: actions.KindAddComment, TicketKey: "HELP-201",
			Source: actions.SourceAI, Summary: "IRE status",
			Comment: "Investigating dashboard timeout.",
		},
		{
			ID:        actions.NewID(actions.KindSetDueDate, now),
			Kind:      actions.KindSetDueDate,
			TicketKey: "APP-103",
			Source:    actions.SourceUser,
			DueDate:   "2026-06-30",
			Summary:   "APP-103: due 2026-06-30",
		},
	}
}
