package demo

import (
	"fmt"
	"strings"

	"github.com/madicen/aitinerary/internal/jira"
	"github.com/madicen/aitinerary/internal/match"
	"github.com/madicen/aitinerary/internal/prompts"
)

// EpicDraftForTickets returns a stub AI epic draft for the given ticket keys.
func EpicDraftForTickets(keys []string) prompts.EpicDraftOutput {
	keySet := map[string]bool{}
	for _, k := range keys {
		keySet[k] = true
	}
	var selected []jira.Ticket
	for _, t := range Tickets() {
		if keySet[t.Key] {
			selected = append(selected, t)
		}
	}
	if len(selected) == 0 {
		return prompts.EpicDraftOutput{
			Title:     "Grouped work",
			Body:      "- Scope from selected tickets",
			StartDate: "2026-06-01",
			DueDate:   "2026-06-30",
		}
	}
	var bullets []string
	earliestDue := ""
	latestDue := ""
	for _, t := range selected {
		bullets = append(bullets, "- "+t.Key+" — "+t.Summary)
		if t.DueDate != "" {
			if earliestDue == "" || t.DueDate < earliestDue {
				earliestDue = t.DueDate
			}
			if latestDue == "" || t.DueDate > latestDue {
				latestDue = t.DueDate
			}
		}
	}
	title := "Retroactive epic for " + selected[0].Key
	if len(selected) > 1 {
		title = fmt.Sprintf("Group %d tickets", len(selected))
	}
	return prompts.EpicDraftOutput{
		Title:     title,
		Body:      strings.Join(bullets, "\n"),
		StartDate: coalesce(earliestDue, "2026-06-01"),
		DueDate:   coalesce(latestDue, "2026-06-30"),
	}
}

// EpicProjectForKeys infers the Jira project for selected ticket keys.
func EpicProjectForKeys(keys []string) string {
	for _, k := range keys {
		if strings.HasPrefix(k, "HELP-") {
			return "HELP"
		}
	}
	return "APP"
}

func coalesce(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

// StatusCommentForTicket returns a deterministic sync-prep status comment stub
// using ticket fields and any matched merged PRs (≤2 sentences).
func StatusCommentForTicket(t jira.Ticket, matches []match.Match) string {
	var prRefs []string
	for _, m := range matches {
		if m.Key != t.Key {
			continue
		}
		ref := fmt.Sprintf("#%d", m.PR.Number)
		if m.PR.Repo != "" {
			ref = m.PR.Repo + "#" + fmt.Sprint(m.PR.Number)
		}
		prRefs = append(prRefs, ref)
	}
	if len(prRefs) > 0 {
		return fmt.Sprintf("%s (%s): merged %s — ready to mark Done after verification.",
			t.Key, t.Summary, strings.Join(prRefs, ", "))
	}
	switch strings.ToLower(t.StatusCategory) {
	case "indeterminate":
		return fmt.Sprintf("%s (%s): still %s; no merged PR detected this sweep.",
			t.Key, t.Summary, t.Status)
	default:
		return fmt.Sprintf("%s (%s): queued as %s for sync review.",
			t.Key, t.Summary, t.Status)
	}
}

// WeeklyDigest returns a deterministic weekly digest markdown for demo/VHS.
func WeeklyDigest() string {
	return `# Weekly digest (demo)

## APP
### Done this week
- APP-99 Already done

### In flight
- APP-101 Widget export drops rows on large CSVs
- APP-102 Add retry to webhook delivery
- APP-103 Stale ticket missing sprint and epic

### Blocked
- (none)

### Next
- APP-102 due 2026-06-20

## HELP (IRE)
### In flight
- HELP-201 IRE: Apps dashboard timeout in prod
`
}

// DemoConfigJSON is the config.json written by setup-demo-fixtures.sh.
const DemoConfigJSON = `{
  "jira_url": "https://stackadapt.atlassian.net",
  "jira_user": "demo@example.com",
  "jira_token_env": "JIRA_TOKEN",
  "workspaces": [
    {"name": "Applications", "project": "APP"},
    {"name": "IRE", "project": "HELP", "labels": ["Apps"]}
  ],
  "board_ids": {"APP": 42, "HELP": 43},
  "excluded_statuses": ["Won't Fix", "Will Not Do"],
  "done_status_name": "Done",
  "auto_done_on_merge": true,
  "stale_in_progress_days": 10,
  "github_token_source": "gh_cli",
  "github_repos": ["stackadapt/applications"],
  "theme_primary": "#7E00AF",
  "theme_secondary": "#50FA7B",
  "theme_muted": "#6272A4"
}
`

// DemoAIConfigJSON is the ai.json written by setup-demo-fixtures.sh.
const DemoAIConfigJSON = `{
  "profiles": [
    {
      "name": "default",
      "provider": "ollama",
      "base_url": "http://127.0.0.1:11434/v1",
      "model": "qwen2.5-coder:7b"
    }
  ],
  "active_profile": "default"
}
`
