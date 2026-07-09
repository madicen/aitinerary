package demo

import (
	"testing"

	"github.com/madicen/aitinerary/internal/match"
)

func TestDemoTicketsCoverWorkspaces(t *testing.T) {
	var app, help int
	for _, t := range Tickets() {
		switch {
		case len(t.Key) >= 4 && t.Key[:4] == "APP-":
			app++
		case len(t.Key) >= 5 && t.Key[:5] == "HELP-":
			help++
		}
	}
	if app == 0 || help == 0 {
		t.Fatalf("expected both APP and HELP tickets, got app=%d help=%d", app, help)
	}
}

func TestDemoMergedPRsMatchLiveTickets(t *testing.T) {
	prs := make([]match.PR, len(MergedPRs()))
	for i, p := range MergedPRs() {
		prs[i] = p.ToMatch()
	}
	matches := match.MatchPRs([]string{"APP", "HELP"}, prs, Tickets())
	if len(matches) == 0 {
		t.Fatal("expected at least one PR/ticket match in demo fixtures")
	}
}

func TestAllTicketsIncludesEpics(t *testing.T) {
	if len(AllTickets()) <= len(Tickets()) {
		t.Fatal("AllTickets should include epics beyond board tickets")
	}
}

func TestEpicDraftForTickets(t *testing.T) {
	draft := EpicDraftForTickets([]string{"APP-103", "HELP-201"})
	if draft.Title == "" {
		t.Fatal("expected non-empty epic title")
	}
	if draft.Body == "" {
		t.Fatal("expected non-empty epic body")
	}
}

func TestWeeklyDigestNonEmpty(t *testing.T) {
	if WeeklyDigest() == "" {
		t.Fatal("expected demo digest")
	}
}
