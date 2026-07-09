package actions

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/madicen/aitinerary/internal/jira"
)

// fakeExecutor is an in-memory Executor recording state so undo can be verified.
type fakeExecutor struct {
	ticket       jira.Ticket
	transitions  []jira.Transition
	transitionTo string // last status transitioned to.
	dueDate      string
	startDate    string
	parent       string
	labelsAdd    []string
	labelsRemove []string
	sprintID     int
	comments     []string
	createdEpic  string
	failTransit  bool // when true, Transition errors (no path).
}

func (f *fakeExecutor) GetTicket(_ context.Context, _ string) (jira.Ticket, error) {
	return f.ticket, nil
}
func (f *fakeExecutor) Transition(_ context.Context, _, toStatus string) error {
	if f.failTransit {
		return errors.New("no transition path")
	}
	f.transitionTo = toStatus
	return nil
}
func (f *fakeExecutor) SetParent(_ context.Context, _, parentKey string) error {
	f.parent = parentKey
	return nil
}
func (f *fakeExecutor) AssignSprint(_ context.Context, sprintID int, _ []string) error {
	f.sprintID = sprintID
	return nil
}
func (f *fakeExecutor) SetDueDate(_ context.Context, _, iso string) error {
	f.dueDate = iso
	return nil
}
func (f *fakeExecutor) SetStartDate(_ context.Context, _, iso string) error {
	f.startDate = iso
	return nil
}
func (f *fakeExecutor) SetLabels(_ context.Context, _ string, add, remove []string) error {
	f.labelsAdd = add
	f.labelsRemove = remove
	return nil
}
func (f *fakeExecutor) AddComment(_ context.Context, _, text string) error {
	f.comments = append(f.comments, text)
	return nil
}
func (f *fakeExecutor) CreateEpic(_ context.Context, spec jira.EpicSpec) (string, error) {
	f.createdEpic = spec.Summary
	return "APP-500", nil
}
func (f *fakeExecutor) ListTransitions(_ context.Context, _ string) ([]jira.Transition, error) {
	return f.transitions, nil
}

func TestApplyTransitionCapturesPriorAndJournals(t *testing.T) {
	fx := &fakeExecutor{ticket: jira.Ticket{Key: "APP-1", Status: "In Progress"}}
	eng := NewEngine(fx, nil)
	a := Action{ID: "a1", Kind: KindTransition, TicketKey: "APP-1", Source: SourceAuto, Confidence: ConfHigh, ToStatus: "Done"}

	entry, err := eng.Apply(context.Background(), a)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if fx.transitionTo != "Done" {
		t.Fatalf("expected transition to Done, got %q", fx.transitionTo)
	}
	if entry.Prior.Status != "In Progress" {
		t.Fatalf("prior status not captured: %+v", entry.Prior)
	}
}

func TestUndoTransitionRestoresPriorStatus(t *testing.T) {
	fx := &fakeExecutor{ticket: jira.Ticket{Key: "APP-1", Status: "In Progress"}}
	eng := NewEngine(fx, nil)
	entry, err := eng.Apply(context.Background(), Action{Kind: KindTransition, TicketKey: "APP-1", ToStatus: "Done"})
	if err != nil {
		t.Fatal(err)
	}
	if err := eng.Undo(context.Background(), entry); err != nil {
		t.Fatalf("Undo: %v", err)
	}
	if fx.transitionTo != "In Progress" {
		t.Fatalf("undo should transition back to In Progress, got %q", fx.transitionTo)
	}
}

func TestUndoTransitionUnavailable(t *testing.T) {
	fx := &fakeExecutor{ticket: jira.Ticket{Key: "APP-1", Status: "In Progress"}}
	eng := NewEngine(fx, nil)
	entry, _ := eng.Apply(context.Background(), Action{Kind: KindTransition, TicketKey: "APP-1", ToStatus: "Done"})
	fx.failTransit = true // no path back
	err := eng.Undo(context.Background(), entry)
	var ua *ErrUndoUnavailable
	if !errors.As(err, &ua) {
		t.Fatalf("expected ErrUndoUnavailable, got %v", err)
	}
}

func TestUndoLabelsReversesAddRemove(t *testing.T) {
	fx := &fakeExecutor{ticket: jira.Ticket{Key: "APP-1", Labels: []string{"old"}}}
	eng := NewEngine(fx, nil)
	entry, err := eng.Apply(context.Background(), Action{
		Kind: KindSetLabels, TicketKey: "APP-1", Labels: []string{"Apps"}, RemoveLabels: []string{"old"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := eng.Undo(context.Background(), entry); err != nil {
		t.Fatal(err)
	}
	// Undo must remove "Apps" and re-add "old".
	if len(fx.labelsAdd) != 1 || fx.labelsAdd[0] != "old" || len(fx.labelsRemove) != 1 || fx.labelsRemove[0] != "Apps" {
		t.Fatalf("undo labels wrong: add=%v remove=%v", fx.labelsAdd, fx.labelsRemove)
	}
}

func TestUndoSetParentWithoutPriorIsUnavailable(t *testing.T) {
	fx := &fakeExecutor{ticket: jira.Ticket{Key: "APP-1", ParentKey: ""}}
	eng := NewEngine(fx, nil)
	entry, _ := eng.Apply(context.Background(), Action{Kind: KindSetParent, TicketKey: "APP-1", ParentKey: "APP-100"})
	err := eng.Undo(context.Background(), entry)
	var ua *ErrUndoUnavailable
	if !errors.As(err, &ua) {
		t.Fatalf("expected ErrUndoUnavailable when there was no prior parent, got %v", err)
	}
}

func TestCreateEpicReturnsKey(t *testing.T) {
	fx := &fakeExecutor{}
	eng := NewEngine(fx, nil)
	entry, err := eng.Apply(context.Background(), Action{
		Kind: KindCreateEpic, Source: SourceUser,
		Epic: &EpicPayload{Project: "APP", Summary: "Retro epic", BodyBullets: []string{"x"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if entry.Result != "APP-500" {
		t.Fatalf("created epic key = %q", entry.Result)
	}
	if fx.createdEpic != "Retro epic" {
		t.Fatalf("epic summary = %q", fx.createdEpic)
	}
}

func TestJournalRoundTrip(t *testing.T) {
	dir := t.TempDir()
	j := NewJournal(dir + "/journal.jsonl")
	in := JournalEntry{
		Time:   time.Date(2026, 7, 8, 10, 0, 0, 0, time.UTC),
		Action: Action{ID: "x", Kind: KindTransition, TicketKey: "APP-1", ToStatus: "Done"},
		Prior:  PriorState{Status: "In Progress"},
		Result: "",
	}
	if err := j.Append(in); err != nil {
		t.Fatal(err)
	}
	if err := j.Append(JournalEntry{Action: Action{Kind: KindSetDueDate, TicketKey: "APP-2", DueDate: "2026-08-01"}}); err != nil {
		t.Fatal(err)
	}
	got, err := j.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("want 2 entries, got %d", len(got))
	}
	if got[0].Prior.Status != "In Progress" || got[0].Action.ToStatus != "Done" {
		t.Fatalf("entry 0 round-trip wrong: %+v", got[0])
	}
}

func TestJournalLoadMissingFile(t *testing.T) {
	j := NewJournal(t.TempDir() + "/nope.jsonl")
	got, err := j.Load()
	if err != nil || got != nil {
		t.Fatalf("missing journal should load empty, got %v err=%v", got, err)
	}
}
