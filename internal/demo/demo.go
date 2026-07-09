// Package demo provides canned Jira tickets, GitHub PRs, and epics for demo
// mode and VHS recordings. The TUI's --demo flag and internal/store load this
// data instead of calling live APIs.
package demo

import (
	"time"

	"github.com/madicen/aitinerary/internal/gh"
	"github.com/madicen/aitinerary/internal/jira"
)

var demoNow = time.Date(2026, 6, 15, 10, 0, 0, 0, time.UTC)

// Tickets returns fixture board tickets spanning both workspaces (APP + IRE).
func Tickets() []jira.Ticket {
	activeSprint := &jira.Sprint{ID: 9001, Name: "Sprint 24", State: "active"}
	return []jira.Ticket{
		{
			Key:            "APP-101",
			Summary:        "Widget export drops rows on large CSVs",
			Status:         "In Progress",
			StatusCategory: "indeterminate",
			Type:           "Story",
			Priority:       "Medium",
			Labels:         []string{"backend"},
			Assignee:       "Michael Madicen",
			Updated:        demoNow.Add(-2 * 24 * time.Hour),
			Description:    "Export truncates after 10k rows.",
		},
		{
			Key:            "APP-102",
			Summary:        "Add retry to webhook delivery",
			Status:         "In Review",
			StatusCategory: "indeterminate",
			Type:           "Task",
			Priority:       "High",
			ParentKey:      "APP-100",
			ParentSummary:  "Q2 reliability push",
			Sprint:         activeSprint,
			DueDate:        "2026-06-20",
			Assignee:       "Michael Madicen",
			Updated:        demoNow.Add(-24 * time.Hour),
		},
		{
			Key:            "APP-103",
			Summary:        "Stale ticket missing sprint and epic",
			Status:         "In Progress",
			StatusCategory: "indeterminate",
			Type:           "Bug",
			Priority:       "Low",
			Assignee:       "Michael Madicen",
			Updated:        demoNow.Add(-14 * 24 * time.Hour),
		},
		{
			Key:            "HELP-201",
			Summary:        "IRE: Apps dashboard timeout in prod",
			Status:         "To Do",
			StatusCategory: "new",
			Type:           "Bug",
			Priority:       "High",
			Labels:         []string{"Apps"},
			DueDate:        "2026-06-18",
			Assignee:       "Michael Madicen",
			Updated:        demoNow.Add(-3 * 24 * time.Hour),
		},
		{
			Key:            "APP-99",
			Summary:        "Already done — should not match merged PR",
			Status:         "Done",
			StatusCategory: "done",
			Type:           "Task",
			Assignee:       "Michael Madicen",
			Updated:        demoNow.Add(-30 * 24 * time.Hour),
		},
	}
}

// Epics returns fixture epics for the epic builder / timeline tab.
func Epics() []jira.Ticket {
	return []jira.Ticket{
		{
			Key:            "APP-100",
			Summary:        "Q2 reliability push",
			Status:         "In Progress",
			StatusCategory: "indeterminate",
			Type:           "Epic",
			StartDate:      "2026-05-01",
			DueDate:        "2026-06-30",
			Assignee:       "Michael Madicen",
			Updated:        demoNow.Add(-5 * 24 * time.Hour),
		},
		{
			Key:            "APP-110",
			Summary:        "Undated epic for timeline gutter",
			Status:         "To Do",
			StatusCategory: "new",
			Type:           "Epic",
			Assignee:       "Michael Madicen",
			Updated:        demoNow.Add(-7 * 24 * time.Hour),
		},
	}
}

// MergedPRs returns fixture merged PRs for match / hygiene demo flows.
func MergedPRs() []gh.PR {
	merged := demoNow.Add(-48 * time.Hour)
	return []gh.PR{
		{
			Number:     501,
			Title:      "APP-101 Fix CSV export truncation",
			URL:        "https://github.com/stackadapt/applications/pull/501",
			Repository: "stackadapt/applications",
			Body:       "Fixes APP-101 by streaming rows.",
			HeadRef:    "feature/APP-101-csv-export",
			MergedAt:   merged,
		},
		{
			Number:     502,
			Title:      "Retry webhook delivery",
			URL:        "https://github.com/stackadapt/applications/pull/502",
			Repository: "stackadapt/applications",
			Body:       "Implements exponential backoff for HELP-201 and APP-102.",
			HeadRef:    "APP-102-webhook-retry",
			MergedAt:   merged.Add(24 * time.Hour),
		},
	}
}

// AllTickets returns board tickets plus epics (deduped by key).
func AllTickets() []jira.Ticket {
	seen := map[string]bool{}
	var out []jira.Ticket
	for _, t := range append(Tickets(), Epics()...) {
		if seen[t.Key] {
			continue
		}
		seen[t.Key] = true
		out = append(out, t)
	}
	return out
}
