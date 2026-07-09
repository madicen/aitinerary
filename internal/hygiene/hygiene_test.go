package hygiene

import (
	"testing"
	"time"

	"github.com/madicen/aitinerary/internal/jira"
	"github.com/madicen/aitinerary/internal/match"
)

var now = time.Date(2026, 7, 8, 12, 0, 0, 0, time.UTC)

func rules() Rules { return Rules{StaleInProgressDays: 10, Now: now} }

// findingsFor returns the set of finding types produced for a single ticket.
func findingsFor(t jira.Ticket, matches []match.Match, r Rules) map[FindingType]Finding {
	out := map[FindingType]Finding{}
	for _, f := range Scan([]jira.Ticket{t}, matches, r) {
		out[f.Type] = f
	}
	return out
}

func TestDoneTicketsSkipped(t *testing.T) {
	tk := jira.Ticket{Key: "APP-1", Type: "Story", StatusCategory: "done"}
	if got := Scan([]jira.Ticket{tk}, nil, rules()); len(got) != 0 {
		t.Fatalf("done ticket should produce no findings, got %+v", got)
	}
}

func TestMissingEpicSprintDueDate(t *testing.T) {
	tk := jira.Ticket{Key: "APP-1", Type: "Story", StatusCategory: "indeterminate", Updated: now}
	got := findingsFor(tk, nil, rules())
	for _, want := range []FindingType{MissingEpic, MissingSprint, MissingDueDate} {
		if _, ok := got[want]; !ok {
			t.Errorf("expected finding %s, got %+v", want, got)
		}
	}
	// Not stale (updated now), so no StaleInProgress.
	if _, ok := got[StaleInProgress]; ok {
		t.Errorf("should not be stale")
	}
}

func TestSubtaskSkipsEpicCheck(t *testing.T) {
	tk := jira.Ticket{Key: "APP-2", Type: "Sub-task", IsSubtask: true, StatusCategory: "new", DueDate: "2026-08-01"}
	// Give it a sprint so only the epic check could fire.
	tk.Sprint = &jira.Sprint{State: "active"}
	got := findingsFor(tk, nil, rules())
	if _, ok := got[MissingEpic]; ok {
		t.Fatalf("subtasks must skip the epic check, got %+v", got)
	}
}

func TestMissingSprintSkippedForKanban(t *testing.T) {
	tk := jira.Ticket{Key: "HELP-5", Type: "Task", StatusCategory: "new", DueDate: "2026-08-01", ParentKey: "HELP-1"}
	r := rules()
	r.KanbanProjects = map[string]bool{"HELP": true}
	got := findingsFor(tk, nil, r)
	if _, ok := got[MissingSprint]; ok {
		t.Fatalf("kanban project should skip MissingSprint, got %+v", got)
	}
}

func TestActiveSprintSatisfiesSprintCheck(t *testing.T) {
	tk := jira.Ticket{Key: "APP-3", Type: "Story", StatusCategory: "new", DueDate: "2026-08-01", ParentKey: "APP-1"}
	tk.Sprint = &jira.Sprint{Name: "S7", State: "active"}
	got := findingsFor(tk, nil, rules())
	if _, ok := got[MissingSprint]; ok {
		t.Fatalf("ticket with active sprint should not flag MissingSprint: %+v", got)
	}
	// Closed sprint should NOT satisfy the check.
	tk.Sprint = &jira.Sprint{Name: "S6", State: "closed"}
	got = findingsFor(tk, nil, rules())
	if _, ok := got[MissingSprint]; !ok {
		t.Fatalf("closed sprint should still flag MissingSprint: %+v", got)
	}
}

func TestStaleInProgress(t *testing.T) {
	tk := jira.Ticket{
		Key: "APP-9", Type: "Story", StatusCategory: "indeterminate",
		ParentKey: "APP-1", DueDate: "2026-08-01",
		Updated: now.Add(-15 * 24 * time.Hour),
	}
	tk.Sprint = &jira.Sprint{State: "active"}
	got := findingsFor(tk, nil, rules())
	f, ok := got[StaleInProgress]
	if !ok {
		t.Fatalf("expected StaleInProgress, got %+v", got)
	}
	if f.Severity != High {
		t.Fatalf("stale should be High severity")
	}
}

func TestEpicMissingDates(t *testing.T) {
	epic := jira.Ticket{Key: "APP-100", Type: "Epic", StatusCategory: "indeterminate"}
	got := findingsFor(epic, nil, rules())
	if _, ok := got[EpicMissingDates]; !ok {
		t.Fatalf("epic without dates should flag EpicMissingDates: %+v", got)
	}
	// Epics must NOT get the missing-epic / missing-sprint / due-date detectors.
	for _, unwanted := range []FindingType{MissingEpic, MissingSprint, MissingDueDate} {
		if _, ok := got[unwanted]; ok {
			t.Fatalf("epic should not get %s", unwanted)
		}
	}
	// A fully-dated epic produces nothing.
	epic.StartDate, epic.DueDate = "2026-07-01", "2026-08-01"
	if len(Scan([]jira.Ticket{epic}, nil, rules())) != 0 {
		t.Fatalf("dated epic should be clean")
	}
}

func TestMergedNotDoneAutoFixableOnlyForHighConfidence(t *testing.T) {
	tickets := []jira.Ticket{
		{Key: "APP-1", Type: "Story", StatusCategory: "indeterminate", ParentKey: "APP-9", DueDate: "2026-08-01"},
		{Key: "APP-2", Type: "Story", StatusCategory: "indeterminate", ParentKey: "APP-9", DueDate: "2026-08-01"},
	}
	tickets[0].Sprint = &jira.Sprint{State: "active"}
	tickets[1].Sprint = &jira.Sprint{State: "active"}
	matches := []match.Match{
		{Key: "APP-1", Confidence: match.High, PR: match.PR{Repo: "o/r", Number: 12}},
		{Key: "APP-2", Confidence: match.Ambiguous, PR: match.PR{Repo: "o/r", Number: 13}},
	}
	findings := Scan(tickets, matches, rules())
	var high, amb *Finding
	for i := range findings {
		if findings[i].Type != MergedNotDone {
			continue
		}
		switch findings[i].TicketKey {
		case "APP-1":
			high = &findings[i]
		case "APP-2":
			amb = &findings[i]
		}
	}
	if high == nil || !high.AutoFixable {
		t.Fatalf("high-confidence merge should be auto-fixable: %+v", high)
	}
	if amb == nil || amb.AutoFixable {
		t.Fatalf("ambiguous merge must not be auto-fixable: %+v", amb)
	}
}

func TestMergedNotDoneSkipsDoneTicket(t *testing.T) {
	tickets := []jira.Ticket{{Key: "APP-1", Type: "Story", StatusCategory: "done"}}
	matches := []match.Match{{Key: "APP-1", Confidence: match.High}}
	for _, f := range Scan(tickets, matches, rules()) {
		if f.Type == MergedNotDone {
			t.Fatalf("done ticket should not get MergedNotDone")
		}
	}
}
