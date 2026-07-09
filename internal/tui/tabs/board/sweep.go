package board

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/madicen/aitinerary/internal/actions"
	"github.com/madicen/aitinerary/internal/demo"
	"github.com/madicen/aitinerary/internal/gh"
	"github.com/madicen/aitinerary/internal/hygiene"
	"github.com/madicen/aitinerary/internal/jira"
	"github.com/madicen/aitinerary/internal/match"
	"github.com/madicen/aitinerary/internal/store"
)

// SweepDoneMsg reports a completed sync-prep sweep (P9.1).
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

type sweepInput struct {
	store          *store.Store
	staleDays      int
	now            time.Time
	mergedPRs      []gh.PR
	kanbanProjects map[string]bool
	projectKeys    []string
	doneStatus     string
}

func (m *Model) startSweep() tea.Cmd {
	if m.sweep != nil && m.sweep.active {
		return nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	m.sweep = &sweepState{active: true, step: "starting", cancel: cancel}
	doneStatus := m.cfg.DoneStatusName
	if doneStatus == "" {
		doneStatus = "Done"
	}
	return runSweepCmd(ctx, sweepInput{
		store:          m.store,
		staleDays:      m.cfg.StaleInProgressDays,
		now:            m.now,
		mergedPRs:      m.mergedPRs,
		kanbanProjects: m.kanbanProjects,
		projectKeys:    m.cfg.ProjectKeys(),
		doneStatus:     doneStatus,
	})
}

func (m *Model) cancelSweep() tea.Cmd {
	if m.sweep != nil && m.sweep.cancel != nil {
		m.sweep.cancel()
	}
	m.sweep = nil
	m.sweepMsg = "sweep cancelled"
	return nil
}

func runSweepCmd(ctx context.Context, in sweepInput) tea.Cmd {
	return func() tea.Msg {
		tickets := loadSweepTickets(in.store)
		var matches []match.Match
		var findings []hygiene.Finding

		steps := []struct {
			name string
			run  func([]jira.Ticket) ([]jira.Ticket, error)
		}{
			{"refresh tickets", func(cur []jira.Ticket) ([]jira.Ticket, error) {
				if in.store != nil {
					if msg, ok := in.store.RefreshCmd()().(store.StoreRefreshedMsg); ok {
						if msg.Err != nil {
							return cur, msg.Err
						}
						return loadSweepTickets(in.store), nil
					}
				}
				return cur, nil
			}},
			{"match merged PRs", func(cur []jira.Ticket) ([]jira.Ticket, error) {
				matches = matchPRsForTickets(cur, in.mergedPRs, in.projectKeys)
				return cur, nil
			}},
			{"hygiene scan", func(cur []jira.Ticket) ([]jira.Ticket, error) {
				rules := hygiene.Rules{
					StaleInProgressDays: in.staleDays,
					KanbanProjects:      in.kanbanProjects,
					Now:                 in.now,
				}
				findings = hygiene.Scan(cur, matches, rules)
				return cur, nil
			}},
			{"draft status comments", func(cur []jira.Ticket) ([]jira.Ticket, error) {
				return cur, nil
			}},
		}

		for _, step := range steps {
			select {
			case <-ctx.Done():
				return SweepDoneMsg{Cancelled: true}
			default:
			}
			next, err := step.run(tickets)
			if err != nil {
				return SweepDoneMsg{Err: err}
			}
			tickets = next
		}

		queued := buildSweepActions(tickets, matches, findings, in.doneStatus, in.now)
		return SweepDoneMsg{Queued: len(queued), Actions: queued}
	}
}

func loadSweepTickets(st *store.Store) []jira.Ticket {
	if st != nil {
		if t := st.Tickets(); len(t) > 0 {
			return t
		}
		if st.DemoMode() {
			return demo.AllTickets()
		}
	}
	return demo.AllTickets()
}

func matchPRsForTickets(tickets []jira.Ticket, mergedPRs []gh.PR, projectKeys []string) []match.Match {
	if len(mergedPRs) == 0 {
		mergedPRs = demo.MergedPRs()
	}
	if len(projectKeys) == 0 {
		projectKeys = projectKeysFromTickets(tickets)
	}
	matchPRs := make([]match.PR, 0, len(mergedPRs))
	for _, p := range mergedPRs {
		matchPRs = append(matchPRs, p.ToMatch())
	}
	return match.MatchPRs(projectKeys, matchPRs, tickets)
}

func projectKeysFromTickets(tickets []jira.Ticket) []string {
	seen := map[string]bool{}
	var keys []string
	for _, t := range tickets {
		if i := indexOfDash(t.Key); i > 0 {
			pk := t.Key[:i]
			if !seen[pk] {
				seen[pk] = true
				keys = append(keys, pk)
			}
		}
	}
	return keys
}

func indexOfDash(s string) int {
	for i, c := range s {
		if c == '-' {
			return i
		}
	}
	return -1
}

func buildSweepActions(tickets []jira.Ticket, matches []match.Match, findings []hygiene.Finding, doneStatus string, now time.Time) []actions.Action {
	if now.IsZero() {
		now = time.Now()
	}
	matchByKey := indexMatchesByKey(matches)
	var out []actions.Action

	for _, t := range tickets {
		if t.IsDone() || strings.EqualFold(t.Type, "Epic") {
			continue
		}
		if !sweepCommentEligible(t, matchByKey[t.Key]) {
			continue
		}
		out = append(out, actions.Action{
			ID:        actions.NewID(actions.KindAddComment, now),
			Kind:      actions.KindAddComment,
			TicketKey: t.Key,
			Source:    actions.SourceAI,
			Summary:   t.Key + ": AI status update",
			Comment:   demo.StatusCommentForTicket(t, matches),
		})
		now = now.Add(time.Nanosecond)
	}

	for _, f := range findings {
		if f.Type != hygiene.MergedNotDone || !f.AutoFixable {
			continue
		}
		prRef := ""
		if f.PR != nil {
			prRef = f.PR.Repo + "#" + fmtInt(f.PR.Number)
		}
		out = append(out, actions.Action{
			ID:         actions.NewID(actions.KindTransition, now),
			Kind:       actions.KindTransition,
			TicketKey:  f.TicketKey,
			Source:     actions.SourceAI,
			Confidence: actions.ConfHigh,
			ToStatus:   doneStatus,
			PRRef:      prRef,
			Summary:    f.TicketKey + " → " + doneStatus + " (merged PR)",
		})
		now = now.Add(time.Nanosecond)
	}

	ordered := sweepRankOrder(tickets)
	for i := 0; i < len(ordered)-1; i++ {
		out = append(out, actions.Action{
			ID:         actions.NewID(actions.KindRank, now),
			Kind:       actions.KindRank,
			TicketKey:  ordered[i].Key,
			Source:     actions.SourceAI,
			RankBefore: ordered[i+1].Key,
			Summary:    fmt.Sprintf("%s: rank above %s", ordered[i].Key, ordered[i+1].Key),
		})
		now = now.Add(time.Nanosecond)
	}
	return out
}

func sweepCommentEligible(t jira.Ticket, hasMatch bool) bool {
	if hasMatch {
		return true
	}
	switch t.StatusCategory {
	case "indeterminate", "new":
		return true
	default:
		return false
	}
}

func indexMatchesByKey(matches []match.Match) map[string]bool {
	out := make(map[string]bool, len(matches))
	for _, m := range matches {
		out[m.Key] = true
	}
	return out
}

func sweepRankOrder(tickets []jira.Ticket) []jira.Ticket {
	var candidates []jira.Ticket
	for _, t := range tickets {
		if t.IsDone() || strings.EqualFold(t.Type, "Epic") {
			continue
		}
		candidates = append(candidates, t)
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		pi, pj := sweepRankPriority(candidates[i]), sweepRankPriority(candidates[j])
		if pi != pj {
			return pi < pj
		}
		if pi == 3 {
			di, dj := candidates[i].DueDate, candidates[j].DueDate
			if di != "" && dj != "" && di != dj {
				return di < dj
			}
			if di != "" && dj == "" {
				return true
			}
			if di == "" && dj != "" {
				return false
			}
		}
		return candidates[i].Key < candidates[j].Key
	})
	return candidates
}

func sweepRankPriority(t jira.Ticket) int {
	if strings.Contains(strings.ToLower(t.Status), "block") {
		return 2
	}
	if t.StatusCategory == "indeterminate" {
		return 1
	}
	return 3
}

func fmtInt(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
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
