package board

import (
	"context"

	"github.com/madicen/aitinerary/internal/actions"
	"github.com/madicen/aitinerary/internal/config"
	"github.com/madicen/aitinerary/internal/demo"
	"github.com/madicen/aitinerary/internal/jira"
	"github.com/madicen/aitinerary/internal/store"
	"github.com/madicen/aitinerary/internal/tui/keys"
)

type fakeLister struct {
	transitions []jira.Transition
	sprints     []jira.Sprint
	epics       []jira.Ticket
}

func (f *fakeLister) ListTransitions(_ context.Context, _ string) ([]jira.Transition, error) {
	return f.transitions, nil
}

func (f *fakeLister) ListSprints(_ context.Context, _ string) ([]jira.Sprint, error) {
	return f.sprints, nil
}

func (f *fakeLister) ListEpics(_ context.Context, _ string) ([]jira.Ticket, error) {
	return f.epics, nil
}

type fakeExecutor struct {
	ticket      jira.Ticket
	transition  string
	dueDate     string
	parent      string
	sprintID    int
	labelsAdd   []string
	transitions []jira.Transition
}

func (f *fakeExecutor) GetTicket(_ context.Context, _ string) (jira.Ticket, error) {
	return f.ticket, nil
}
func (f *fakeExecutor) Transition(_ context.Context, _, to string) error {
	f.transition = to
	f.ticket.Status = to
	return nil
}
func (f *fakeExecutor) SetParent(_ context.Context, _, parent string) error {
	f.parent = parent
	f.ticket.ParentKey = parent
	return nil
}
func (f *fakeExecutor) AssignSprint(_ context.Context, sprintID int, _ []string) error {
	f.sprintID = sprintID
	return nil
}
func (f *fakeExecutor) SetDueDate(_ context.Context, _, iso string) error {
	f.dueDate = iso
	f.ticket.DueDate = iso
	return nil
}
func (f *fakeExecutor) SetStartDate(_ context.Context, _, _ string) error { return nil }
func (f *fakeExecutor) SetLabels(_ context.Context, _ string, add, _ []string) error {
	f.labelsAdd = add
	f.ticket.Labels = append([]string(nil), add...)
	return nil
}
func (f *fakeExecutor) AddComment(_ context.Context, _, _ string) error { return nil }
func (f *fakeExecutor) CreateEpic(_ context.Context, _ jira.EpicSpec) (string, error) {
	return "APP-500", nil
}
func (f *fakeExecutor) CreateTicket(_ context.Context, _ jira.CreateIssueSpec) (string, error) {
	return "APP-999", nil
}
func (f *fakeExecutor) RankIssues(_ context.Context, _ []string, _ string) error { return nil }
func (f *fakeExecutor) ListTransitions(_ context.Context, _ string) ([]jira.Transition, error) {
	return f.transitions, nil
}

func demoBoardModel(width, height int) *Model {
	fx := &fakeExecutor{
		ticket: demo.Tickets()[0],
		transitions: []jira.Transition{
			{ID: "1", Name: "Done", ToStatus: "Done"},
			{ID: "2", Name: "Review", ToStatus: "In Review"},
		},
	}
	return demoBoardModelWithExecutor(width, height, fx)
}

func demoBoardModelWithExecutor(width, height int, fx *fakeExecutor) *Model {
	cfg := config.Default()
	st := store.New(cfg, nil, true)
	_ = st.RefreshCmd()()
	if fx == nil {
		fx = &fakeExecutor{ticket: demo.Tickets()[0]}
	}
	if fx.transitions == nil {
		fx.transitions = []jira.Transition{
			{ID: "1", Name: "Done", ToStatus: "Done"},
			{ID: "2", Name: "Review", ToStatus: "In Review"},
		}
	}
	list := &fakeLister{
		transitions: fx.transitions,
		sprints:     []jira.Sprint{{ID: 9001, Name: "Sprint 24", State: "active"}},
		epics:       demo.Epics(),
	}
	m := NewWithOptions(Options{
		Config:      cfg,
		Store:       st,
		Engine:      actions.NewEngine(fx, nil),
		Keys:        keys.Default(),
		MergedPRs:   demo.MergedPRs(),
		Transitions: list,
		Sprints:     list,
		Epics:       list,
	})
	m.Resize(width, height)
	m.detailOpen = true
	for i, r := range m.rows {
		if r.Kind == RowTicket {
			m.selected = i
			break
		}
	}
	return m
}
