package data

import (
	"context"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/madicen/aitinerary/internal/actions"
	"github.com/madicen/aitinerary/internal/config"
	"github.com/madicen/aitinerary/internal/demo"
	"github.com/madicen/aitinerary/internal/gh"
	"github.com/madicen/aitinerary/internal/hygiene"
	"github.com/madicen/aitinerary/internal/jira"
	"github.com/madicen/aitinerary/internal/match"
	"github.com/madicen/aitinerary/internal/store"
)

// RefreshAllMsg is emitted after a full data refresh (store + PRs + auto-apply).
type RefreshAllMsg struct {
	Store           store.StoreRefreshedMsg
	MergedPRs       []gh.PR
	KanbanProjects  map[string]bool
	AutoApplied     []actions.JournalEntry
	AutoApplyErrors []error
}

// RefreshAllCmd refreshes tickets, merged PRs, kanban board types, and applies
// safe auto-done transitions.
func RefreshAllCmd(cfg *config.Config, demoMode bool, st *store.Store, jc *jira.Client, eng *actions.Engine, auto *actions.AutoApplied) tea.Cmd {
	return func() tea.Msg {
		storeMsg := st.RefreshCmd()().(store.StoreRefreshedMsg)

		var merged []gh.PR
		kanban := map[string]bool{}
		if demoMode {
			merged = demo.MergedPRs()
			kanban["HELP"] = true
		} else {
			var err error
			merged, err = gh.ListMergedPRs(context.Background(), cfg.GitHubRepos, 14)
			if err != nil {
				storeMsg.Err = err
			}
			if jc != nil {
				for _, ws := range cfg.Workspaces {
					typ, err := jc.BoardTypeForProject(context.Background(), ws.Project)
					if err == nil && strings.EqualFold(typ, "kanban") {
						kanban[ws.Project] = true
					}
				}
			}
		}

		tickets := st.Tickets()
		if demoMode && len(tickets) == 0 {
			tickets = demo.AllTickets()
		}
		prs := make([]match.PR, 0, len(merged))
		for _, p := range merged {
			prs = append(prs, p.ToMatch())
		}
		matches := match.MatchPRs(cfg.ProjectKeys(), prs, tickets)
		rules := hygiene.Rules{
			StaleInProgressDays: cfg.StaleInProgressDays,
			KanbanProjects:      kanban,
			Now:                 time.Now(),
		}
		_ = hygiene.Scan(tickets, matches, rules)

		policy := actions.Policy{AutoDoneOnMerge: cfg.AutoDoneOnMerge, DoneStatusName: cfg.DoneStatusName}
		var autoActions []actions.Action
		doneName := cfg.DoneStatusName
		if doneName == "" {
			doneName = "Done"
		}
		for _, mt := range matches {
			if mt.Confidence != match.High {
				continue
			}
			a := actions.Action{
				ID:         actions.NewID(actions.KindTransition, time.Now()),
				Kind:       actions.KindTransition,
				TicketKey:  mt.Key,
				Source:     actions.SourceAuto,
				Confidence: actions.ConfHigh,
				ToStatus:   doneName,
				PRRef:      mt.PR.Repo + "#" + fmtInt(mt.PR.Number),
				Summary:    mt.Key + " → " + doneName + " (merged PR)",
			}
			if policy.AutoApplyable(a) {
				autoActions = append(autoActions, a)
			}
		}

		var applied []actions.JournalEntry
		var applyErrs []error
		if eng != nil && jc != nil {
			for _, a := range autoActions {
				entry, err := eng.Apply(context.Background(), a)
				if err != nil {
					applyErrs = append(applyErrs, err)
					continue
				}
				applied = append(applied, entry)
				if auto != nil {
					auto.Record(entry)
				}
			}
		}

		return RefreshAllMsg{
			Store:           storeMsg,
			MergedPRs:       merged,
			KanbanProjects:  kanban,
			AutoApplied:     applied,
			AutoApplyErrors: applyErrs,
		}
	}
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
