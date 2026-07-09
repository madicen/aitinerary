package board

import (
	"context"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/madicen/aitinerary/internal/actions"
	"github.com/madicen/aitinerary/internal/demo"
	"github.com/madicen/aitinerary/internal/hygiene"
	"github.com/madicen/aitinerary/internal/jira"
	"github.com/madicen/aitinerary/internal/match"
	"github.com/madicen/aitinerary/internal/store"
)

// SweepDoneMsg reports a completed sync-prep sweep (P9.1 stub).
type SweepDoneMsg struct {
	Queued    int
	Actions   []actions.Action
	Cancelled bool
	Err       error
}

// SweepProgressMsg updates the sweep overlay step label.
type SweepProgressMsg struct {
	Step string
}

type sweepState struct {
	active bool
	step   string
	cancel context.CancelFunc
}

func (m *Model) startSweep() tea.Cmd {
	if m.sweep != nil && m.sweep.active {
		return nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	m.sweep = &sweepState{active: true, step: "starting", cancel: cancel}
	return runSweepCmd(ctx, m.store, m.cfg.StaleInProgressDays, m.now)
}

func (m *Model) cancelSweep() tea.Cmd {
	if m.sweep != nil && m.sweep.cancel != nil {
		m.sweep.cancel()
	}
	m.sweep = nil
	m.sweepMsg = "sweep cancelled"
	return nil
}

func runSweepCmd(ctx context.Context, st *store.Store, staleDays int, now time.Time) tea.Cmd {
	return func() tea.Msg {
		tickets := demo.AllTickets()
		steps := []struct {
			name string
			run  func([]jira.Ticket) ([]jira.Ticket, []actions.Action, error)
		}{
			{"refresh tickets", func(cur []jira.Ticket) ([]jira.Ticket, []actions.Action, error) {
				if st != nil {
					if msg, ok := st.RefreshCmd()().(store.StoreRefreshedMsg); ok {
						if msg.Err != nil {
							return cur, nil, msg.Err
						}
						return st.Tickets(), nil, nil
					}
				}
				return cur, nil, nil
			}},
			{"match merged PRs", func(cur []jira.Ticket) ([]jira.Ticket, []actions.Action, error) {
				_ = matchPRsForTickets(cur)
				return cur, nil, nil
			}},
			{"hygiene scan", func(cur []jira.Ticket) ([]jira.Ticket, []actions.Action, error) {
				rules := hygiene.Rules{StaleInProgressDays: staleDays, Now: now}
				_ = hygiene.Scan(cur, matchPRsForTickets(cur), rules)
				return cur, nil, nil
			}},
			{"draft status comments", func(cur []jira.Ticket) ([]jira.Ticket, []actions.Action, error) {
				return cur, stubSweepActions(cur), nil
			}},
		}

		var queued []actions.Action
		for _, step := range steps {
			select {
			case <-ctx.Done():
				return SweepDoneMsg{Cancelled: true}
			default:
			}
			next, acts, err := step.run(tickets)
			if err != nil {
				return SweepDoneMsg{Queued: len(queued), Actions: queued, Err: err}
			}
			tickets = next
			queued = append(queued, acts...)
		}
		return SweepDoneMsg{Queued: len(queued), Actions: queued}
	}
}

func matchPRsForTickets(tickets []jira.Ticket) []match.Match {
	prs := demo.MergedPRs()
	matchPRs := make([]match.PR, 0, len(prs))
	for _, p := range prs {
		matchPRs = append(matchPRs, p.ToMatch())
	}
	keys := make([]string, 0, len(tickets))
	seen := map[string]bool{}
	for _, t := range tickets {
		if i := indexOfDash(t.Key); i > 0 {
			pk := t.Key[:i]
			if !seen[pk] {
				seen[pk] = true
				keys = append(keys, pk)
			}
		}
	}
	return match.MatchPRs(keys, matchPRs, tickets)
}

func indexOfDash(s string) int {
	for i, c := range s {
		if c == '-' {
			return i
		}
	}
	return -1
}

func stubSweepActions(tickets []jira.Ticket) []actions.Action {
	now := time.Now()
	var out []actions.Action
	for _, t := range tickets {
		if t.IsDone() {
			continue
		}
		out = append(out, actions.Action{
			ID:        actions.NewID(actions.KindAddComment, now),
			Kind:      actions.KindAddComment,
			TicketKey: t.Key,
			Source:    actions.SourceAI,
			Summary:   "AI status update",
			Comment:   fmt.Sprintf("%s: progress update from sync-prep stub.", t.Key),
		})
		now = now.Add(time.Nanosecond)
	}
	return out
}

// renderSweepOverlay draws the cancellable progress overlay.
func (m *Model) renderSweepOverlay(base string) string {
	if m.sweep == nil || !m.sweep.active {
		return base
	}
	step := m.sweep.step
	if step == "" {
		step = "…"
	}
	overlay := fmt.Sprintf("Sync-prep sweep\n\nStep: %s\n\n[cancel esc]", step)
	return overlay + "\n\n" + base
}
