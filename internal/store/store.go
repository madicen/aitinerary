// Package store is the async ticket cache feeding the board and hygiene layer.
// It merges tickets from every configured workspace, tracks per-workspace
// freshness, and exposes a BubbleTea refresh command.
package store

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/madicen/aitinerary/internal/config"
	"github.com/madicen/aitinerary/internal/demo"
	"github.com/madicen/aitinerary/internal/jira"
)

// JiraClient is the subset of jira.Client the store needs. Tests inject fakes.
type JiraClient interface {
	SearchWorkspace(ctx context.Context, q jira.WorkspaceQuery) ([]jira.Ticket, error)
}

// StoreRefreshedMsg is emitted when a refresh completes (success or partial failure).
type StoreRefreshedMsg struct {
	Err             error
	WorkspaceErrors map[string]error
}

// Store holds merged tickets and per-workspace freshness timestamps.
type Store struct {
	cfg  *config.Config
	jira JiraClient
	demo bool

	mu          sync.RWMutex
	tickets     []jira.Ticket
	freshness   map[string]time.Time
	lastRefresh time.Time
	lastErr     error
	refreshing  bool
}

// New constructs a Store. jira may be nil only in demo mode.
func New(cfg *config.Config, jiraClient JiraClient, demoMode bool) *Store {
	if cfg == nil {
		cfg = config.Default()
	}
	return &Store{
		cfg:       cfg,
		jira:      jiraClient,
		demo:      demoMode,
		freshness: make(map[string]time.Time),
	}
}

// DemoMode reports whether the store serves canned demo data.
func (s *Store) DemoMode() bool { return s.demo }

// Refreshing reports whether a refresh is in flight.
func (s *Store) Refreshing() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.refreshing
}

// Tickets returns a copy of the merged ticket list.
func (s *Store) Tickets() []jira.Ticket {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]jira.Ticket, len(s.tickets))
	copy(out, s.tickets)
	return out
}

// Freshness returns when each workspace was last refreshed successfully.
func (s *Store) Freshness() map[string]time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]time.Time, len(s.freshness))
	for k, v := range s.freshness {
		out[k] = v
	}
	return out
}

// LastRefresh returns the most recent successful merge time.
func (s *Store) LastRefresh() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastRefresh
}

// LastError returns the most recent refresh error, if any.
func (s *Store) LastError() error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastErr
}

// RefreshCmd returns a BubbleTea command that refreshes tickets asynchronously.
func (s *Store) RefreshCmd() tea.Cmd {
	return func() tea.Msg {
		msg := s.refreshSync(context.Background())
		return msg
	}
}

func (s *Store) refreshSync(ctx context.Context) StoreRefreshedMsg {
	s.mu.Lock()
	s.refreshing = true
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.refreshing = false
		s.mu.Unlock()
	}()

	msg := s.fetch(ctx)

	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastErr = msg.Err
	if msg.Err == nil {
		s.lastRefresh = time.Now()
	}
	return msg
}

func (s *Store) fetch(ctx context.Context) StoreRefreshedMsg {
	if s.demo {
		return s.loadDemo()
	}
	if s.jira == nil {
		return StoreRefreshedMsg{Err: fmt.Errorf("store: jira client required when not in demo mode")}
	}

	type result struct {
		name    string
		tickets []jira.Ticket
		err     error
	}

	workspaces := s.cfg.Workspaces
	ch := make(chan result, len(workspaces))
	var wg sync.WaitGroup
	for _, ws := range workspaces {
		wg.Add(1)
		go func(ws config.Workspace) {
			defer wg.Done()
			q := workspaceQuery(s.cfg, ws)
			tickets, err := s.jira.SearchWorkspace(ctx, q)
			ch <- result{name: ws.Name, tickets: tickets, err: err}
		}(ws)
	}
	wg.Wait()
	close(ch)

	merged := map[string]jira.Ticket{}
	freshness := make(map[string]time.Time, len(workspaces))
	var wsErrors map[string]error
	now := time.Now()

	for r := range ch {
		if r.err != nil {
			if wsErrors == nil {
				wsErrors = make(map[string]error)
			}
			wsErrors[r.name] = r.err
			continue
		}
		freshness[r.name] = now
		for _, t := range r.tickets {
			merged[strings.ToUpper(t.Key)] = t
		}
	}

	tickets := make([]jira.Ticket, 0, len(merged))
	for _, t := range merged {
		tickets = append(tickets, t)
	}
	sort.Slice(tickets, func(i, j int) bool {
		return tickets[i].Updated.After(tickets[j].Updated)
	})

	s.mu.Lock()
	s.tickets = tickets
	s.freshness = freshness
	s.mu.Unlock()

	var err error
	if len(wsErrors) > 0 {
		if len(freshness) == 0 {
			err = fmt.Errorf("all workspaces failed to refresh")
		} else {
			err = fmt.Errorf("%d workspace(s) failed to refresh", len(wsErrors))
		}
	}
	return StoreRefreshedMsg{Err: err, WorkspaceErrors: wsErrors}
}

func (s *Store) loadDemo() StoreRefreshedMsg {
	tickets := demo.AllTickets()
	now := time.Now()
	freshness := make(map[string]time.Time, len(s.cfg.Workspaces))
	for _, ws := range s.cfg.Workspaces {
		if ws.Name != "" {
			freshness[ws.Name] = now
		}
	}

	s.mu.Lock()
	s.tickets = tickets
	s.freshness = freshness
	s.mu.Unlock()

	return StoreRefreshedMsg{}
}

func workspaceQuery(cfg *config.Config, ws config.Workspace) jira.WorkspaceQuery {
	return jira.WorkspaceQuery{
		Project:             ws.Project,
		Labels:              ws.Labels,
		AssigneeCurrentUser: true,
		ExcludedStatuses:    cfg.ExcludedStatuses,
		JQLExtra:            ws.JQLExtra,
	}
}

// TicketByKey looks up a ticket by key (case-insensitive).
func (s *Store) TicketByKey(key string) (jira.Ticket, bool) {
	key = strings.ToUpper(strings.TrimSpace(key))
	for _, t := range s.Tickets() {
		if strings.ToUpper(t.Key) == key {
			return t, true
		}
	}
	return jira.Ticket{}, false
}

// LiveTickets returns tickets that are not Done.
func (s *Store) LiveTickets() []jira.Ticket {
	var out []jira.Ticket
	for _, t := range s.Tickets() {
		if !t.IsDone() {
			out = append(out, t)
		}
	}
	return out
}
