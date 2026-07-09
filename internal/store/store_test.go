package store

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/madicen/aitinerary/internal/config"
	"github.com/madicen/aitinerary/internal/demo"
	"github.com/madicen/aitinerary/internal/jira"
)

type fakeJira struct {
	mu        sync.Mutex
	byProject map[string][]jira.Ticket
	errFor    map[string]error
	calls     []jira.WorkspaceQuery
}

func (f *fakeJira) SearchWorkspace(ctx context.Context, q jira.WorkspaceQuery) ([]jira.Ticket, error) {
	f.mu.Lock()
	f.calls = append(f.calls, q)
	err := f.errFor[q.Project]
	tickets := append([]jira.Ticket(nil), f.byProject[q.Project]...)
	f.mu.Unlock()
	if err != nil {
		return nil, err
	}
	return tickets, nil
}

func TestRefreshMergesWorkspaces(t *testing.T) {
	cfg := config.Default()
	fj := &fakeJira{
		byProject: map[string][]jira.Ticket{
			"APP": {
				{Key: "APP-1", Summary: "app ticket", Updated: time.Unix(100, 0)},
			},
			"HELP": {
				{Key: "HELP-1", Summary: "ire ticket", Updated: time.Unix(200, 0)},
			},
		},
	}
	s := New(cfg, fj, false)

	msg := s.refreshSync(context.Background())
	if msg.Err != nil {
		t.Fatalf("refresh: %v", msg.Err)
	}
	if len(fj.calls) != 2 {
		fj.mu.Lock()
		n := len(fj.calls)
		fj.mu.Unlock()
		t.Fatalf("calls = %d, want 2", n)
	}

	tickets := s.Tickets()
	if len(tickets) != 2 {
		t.Fatalf("len(tickets) = %d, want 2", len(tickets))
	}
	if tickets[0].Key != "HELP-1" {
		t.Fatalf("sort order: first = %s, want HELP-1", tickets[0].Key)
	}

	fresh := s.Freshness()
	if len(fresh) != 2 {
		t.Fatalf("freshness workspaces = %d, want 2", len(fresh))
	}
	if fresh["Applications"].IsZero() || fresh["IRE"].IsZero() {
		t.Fatalf("expected freshness for both workspaces: %+v", fresh)
	}
}

func TestRefreshPartialFailure(t *testing.T) {
	cfg := config.Default()
	fj := &fakeJira{
		byProject: map[string][]jira.Ticket{
			"APP": {{Key: "APP-1", Updated: time.Now()}},
		},
		errFor: map[string]error{
			"HELP": errors.New("jira down"),
		},
	}
	s := New(cfg, fj, false)
	msg := s.refreshSync(context.Background())
	if msg.Err == nil {
		t.Fatal("expected partial failure error")
	}
	if msg.WorkspaceErrors["IRE"] == nil {
		t.Fatalf("expected IRE error, got %+v", msg.WorkspaceErrors)
	}
	if len(s.Tickets()) != 1 {
		t.Fatalf("expected 1 merged ticket, got %d", len(s.Tickets()))
	}
	if _, ok := s.Freshness()["Applications"]; !ok {
		t.Fatal("APP workspace should be fresh")
	}
	if _, ok := s.Freshness()["IRE"]; ok {
		t.Fatal("IRE workspace should not be fresh after error")
	}
}

func TestRefreshAllWorkspacesFail(t *testing.T) {
	cfg := config.Default()
	fj := &fakeJira{
		errFor: map[string]error{
			"APP":  errors.New("fail app"),
			"HELP": errors.New("fail help"),
		},
	}
	s := New(cfg, fj, false)
	msg := s.refreshSync(context.Background())
	if msg.Err == nil {
		t.Fatal("expected error")
	}
	if len(s.Tickets()) != 0 {
		t.Fatalf("expected no tickets, got %d", len(s.Tickets()))
	}
}

func TestDemoModeLoadsFixtures(t *testing.T) {
	s := New(config.Default(), nil, true)
	msg := s.refreshSync(context.Background())
	if msg.Err != nil {
		t.Fatalf("demo refresh: %v", msg.Err)
	}
	tickets := s.Tickets()
	if len(tickets) != len(demo.AllTickets()) {
		t.Fatalf("len = %d, want %d", len(tickets), len(demo.AllTickets()))
	}
	if _, ok := s.TicketByKey("APP-101"); !ok {
		t.Fatal("expected APP-101 in demo data")
	}
	if _, ok := s.TicketByKey("APP-100"); !ok {
		t.Fatal("expected epic APP-100 in demo data")
	}
}

func TestRefreshCmdReturnsMsg(t *testing.T) {
	s := New(config.Default(), nil, true)
	cmd := s.RefreshCmd()
	msg := cmd()
	refreshed, ok := msg.(StoreRefreshedMsg)
	if !ok {
		t.Fatalf("msg type = %T, want StoreRefreshedMsg", msg)
	}
	if refreshed.Err != nil {
		t.Fatalf("demo refresh err: %v", refreshed.Err)
	}
}

func TestLiveTicketsSkipsDone(t *testing.T) {
	s := New(config.Default(), nil, true)
	s.refreshSync(context.Background())
	for _, ticket := range s.LiveTickets() {
		if ticket.IsDone() {
			t.Fatalf("live ticket %s is Done", ticket.Key)
		}
	}
}

func TestWorkspaceQueryUsesConfig(t *testing.T) {
	cfg := config.Default()
	q := workspaceQuery(cfg, cfg.Workspaces[1])
	if q.Project != "HELP" {
		t.Fatalf("project = %q", q.Project)
	}
	if len(q.Labels) != 1 || q.Labels[0] != "Apps" {
		t.Fatalf("labels = %v", q.Labels)
	}
	if !q.AssigneeCurrentUser {
		t.Fatal("expected assignee = currentUser()")
	}
}

func TestRefreshRequiresJiraWhenNotDemo(t *testing.T) {
	s := New(config.Default(), nil, false)
	msg := s.refreshSync(context.Background())
	if msg.Err == nil {
		t.Fatal("expected error without jira client")
	}
	if msg.Err.Error() != "store: jira client required when not in demo mode" {
		t.Fatalf("unexpected err: %v", msg.Err)
	}
}
