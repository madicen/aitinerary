// Package hygiene runs the ticket-hygiene detectors described in PLAN.md §7
// P6.1. Given the current tickets and the merged-PR matches, it produces a list
// of Findings — each naming a problem and proposing the action that fixes it.
//
// It is pure logic over jira.Ticket / match.Match and the standard library, so
// every detector is table-testable without a network.
package hygiene

import (
	"sort"
	"strings"
	"time"

	"github.com/madicen/aitinerary/internal/jira"
	"github.com/madicen/aitinerary/internal/match"
)

// FindingType enumerates the hygiene problems aitinerary detects.
type FindingType string

const (
	// MergedNotDone: a merged PR maps to this ticket but it isn't Done.
	MergedNotDone FindingType = "merged_not_done"
	// MissingEpic: a non-subtask, non-epic, non-done ticket has no parent epic.
	MissingEpic FindingType = "missing_epic"
	// MissingSprint: a non-done ticket has no active/future sprint (scrum only).
	MissingSprint FindingType = "missing_sprint"
	// MissingDueDate: a non-done ticket has no due date (soon mandatory).
	MissingDueDate FindingType = "missing_due_date"
	// StaleInProgress: an In Progress ticket hasn't been updated in N days.
	StaleInProgress FindingType = "stale_in_progress"
	// EpicMissingDates: an epic lacks a start or due date — breaks the timeline.
	EpicMissingDates FindingType = "epic_missing_dates"
)

// Severity orders findings for display.
type Severity int

const (
	Info Severity = iota
	Warn
	High
)

// Finding is one detected problem on one ticket.
type Finding struct {
	Type      FindingType
	TicketKey string
	Severity  Severity
	Detail    string    // short, human-facing explanation.
	PR        *match.PR // the merged PR, for MergedNotDone.
	// AutoFixable is true for MergedNotDone with a High-confidence single-PR
	// match — the only finding the actions engine may auto-apply (P5.2).
	AutoFixable bool
}

// Rules configures the detectors. KanbanProjects lists project keys whose board
// is Kanban, for which MissingSprint is skipped (PLAN.md §9). Now is injectable
// so staleness is deterministic in tests.
type Rules struct {
	StaleInProgressDays int
	KanbanProjects      map[string]bool
	Now                 time.Time
}

// projectOf returns the project key portion of a ticket key ("APP-12" → "APP").
func projectOf(key string) string {
	if i := strings.IndexByte(key, '-'); i > 0 {
		return key[:i]
	}
	return key
}

func isEpic(t jira.Ticket) bool { return strings.EqualFold(t.Type, "Epic") }

// Scan runs every detector over tickets (plus the PR matches) and returns the
// findings, ordered by severity (high first) then ticket key for stable output.
func Scan(tickets []jira.Ticket, matches []match.Match, rules Rules) []Finding {
	if rules.Now.IsZero() {
		rules.Now = time.Now()
	}
	if rules.StaleInProgressDays <= 0 {
		rules.StaleInProgressDays = 10
	}

	byKey := make(map[string]jira.Ticket, len(tickets))
	for _, t := range tickets {
		byKey[t.Key] = t
	}

	var findings []Finding

	// MergedNotDone — from the PR matcher. High-confidence single matches are
	// auto-fixable; ambiguous ones still surface but must be reviewed.
	for _, m := range matches {
		t, ok := byKey[m.Key]
		if !ok || t.IsDone() {
			continue
		}
		pr := m.PR
		findings = append(findings, Finding{
			Type:        MergedNotDone,
			TicketKey:   m.Key,
			Severity:    High,
			Detail:      mergedDetail(pr),
			PR:          &pr,
			AutoFixable: m.Confidence == match.High,
		})
	}

	// Per-ticket detectors.
	for _, t := range tickets {
		if t.IsDone() {
			continue // Done tickets are skipped by every field detector.
		}

		if isEpic(t) {
			// Epics get the timeline-dates detector instead of the epic/sprint ones.
			if t.StartDate == "" || t.DueDate == "" {
				findings = append(findings, Finding{
					Type:      EpicMissingDates,
					TicketKey: t.Key,
					Severity:  Warn,
					Detail:    epicDatesDetail(t),
				})
			}
			continue
		}

		// MissingEpic — subtasks are parented to their story, so they skip this.
		if !t.IsSubtask && t.ParentKey == "" {
			findings = append(findings, Finding{
				Type: MissingEpic, TicketKey: t.Key, Severity: Warn,
				Detail: "no parent epic",
			})
		}

		// MissingSprint — skipped for kanban projects (no sprints there).
		if !rules.KanbanProjects[projectOf(t.Key)] && !hasLiveSprint(t) {
			findings = append(findings, Finding{
				Type: MissingSprint, TicketKey: t.Key, Severity: Warn,
				Detail: "no active or future sprint",
			})
		}

		// MissingDueDate.
		if t.DueDate == "" {
			findings = append(findings, Finding{
				Type: MissingDueDate, TicketKey: t.Key, Severity: Warn,
				Detail: "no due date",
			})
		}

		// StaleInProgress.
		if t.StatusCategory == "indeterminate" && !t.Updated.IsZero() {
			age := rules.Now.Sub(t.Updated)
			if age >= time.Duration(rules.StaleInProgressDays)*24*time.Hour {
				findings = append(findings, Finding{
					Type: StaleInProgress, TicketKey: t.Key, Severity: High,
					Detail: staleDetail(int(age.Hours() / 24)),
				})
			}
		}
	}

	sort.SliceStable(findings, func(i, j int) bool {
		if findings[i].Severity != findings[j].Severity {
			return findings[i].Severity > findings[j].Severity
		}
		if findings[i].TicketKey != findings[j].TicketKey {
			return findings[i].TicketKey < findings[j].TicketKey
		}
		return findings[i].Type < findings[j].Type
	})
	return findings
}

// hasLiveSprint reports whether the ticket is in an active or future sprint.
func hasLiveSprint(t jira.Ticket) bool {
	if t.Sprint == nil {
		return false
	}
	switch strings.ToLower(t.Sprint.State) {
	case "active", "future":
		return true
	default:
		return false
	}
}

func mergedDetail(pr match.PR) string {
	if pr.Repo != "" && pr.Number > 0 {
		return "merged PR " + pr.Repo + "#" + itoa(pr.Number) + " — should be Done"
	}
	return "a merged PR maps here — should be Done"
}

func epicDatesDetail(t jira.Ticket) string {
	switch {
	case t.StartDate == "" && t.DueDate == "":
		return "epic missing start and due dates (won't place on timeline)"
	case t.StartDate == "":
		return "epic missing start date (timeline placement)"
	default:
		return "epic missing due date (timeline placement)"
	}
}

func staleDetail(days int) string {
	return "In Progress, no update in " + itoa(days) + " days"
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
