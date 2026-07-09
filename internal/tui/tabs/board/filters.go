package board

import (
	"sort"
	"strings"

	"github.com/madicen/aitinerary/internal/hygiene"
	"github.com/madicen/aitinerary/internal/jira"
	"github.com/madicen/aitinerary/internal/match"
)

// FilterPreset is the active board filter chip selection.
type FilterPreset int

const (
	FilterNeedsAttention FilterPreset = iota
	FilterAll
	FilterDoneCandidates
)

// FilterState holds chip filters, optional workspace/status narrowing, and search.
type FilterState struct {
	Preset         FilterPreset
	Workspace      string // empty = all workspaces
	StatusCategory string // empty = all; "new"|"indeterminate"|"done"
	SearchQuery    string
}

// DefaultFilters returns the P6.2 default: needs-attention on launch.
func DefaultFilters() FilterState {
	return FilterState{Preset: FilterNeedsAttention}
}

func (f FilterState) clone() FilterState { return f }

func ticketMatchesSearch(t jira.Ticket, q string) bool {
	q = strings.ToLower(strings.TrimSpace(q))
	if q == "" {
		return true
	}
	hay := strings.ToLower(t.Key + " " + t.Summary + " " + t.Status + " " + strings.Join(t.Labels, " "))
	return strings.Contains(hay, q)
}

func ticketWorkspace(project string, workspaces []workspaceChip) string {
	for _, w := range workspaces {
		if strings.EqualFold(w.Project, project) {
			return w.Chip
		}
	}
	if i := strings.IndexByte(project, '-'); i > 0 {
		return project[:i]
	}
	return project
}

type workspaceChip struct {
	Name    string
	Project string
	Chip    string
}

// ticketIndex maps ticket key → source index in the all-tickets slice.
type ticketData struct {
	Ticket    jira.Ticket
	Workspace string
	Findings  []hygiene.Finding
	Matches   []match.Match
}

func indexTickets(tickets []jira.Ticket, chips []workspaceChip, findings []hygiene.Finding, matches []match.Match) map[string]ticketData {
	byKey := make(map[string]ticketData, len(tickets))
	findByKey := map[string][]hygiene.Finding{}
	for _, f := range findings {
		findByKey[f.TicketKey] = append(findByKey[f.TicketKey], f)
	}
	matchByKey := map[string][]match.Match{}
	for _, m := range matches {
		matchByKey[m.Key] = append(matchByKey[m.Key], m)
	}
	for _, t := range tickets {
		proj := t.Key
		if i := strings.IndexByte(t.Key, '-'); i > 0 {
			proj = t.Key[:i]
		}
		byKey[t.Key] = ticketData{
			Ticket:    t,
			Workspace: ticketWorkspace(proj, chips),
			Findings:  findByKey[t.Key],
			Matches:   matchByKey[t.Key],
		}
	}
	return byKey
}

func passesPreset(td ticketData, preset FilterPreset) bool {
	switch preset {
	case FilterAll:
		return true
	case FilterDoneCandidates:
		for _, f := range td.Findings {
			if f.Type == hygiene.MergedNotDone {
				return true
			}
		}
		return false
	default: // FilterNeedsAttention
		return len(td.Findings) > 0
	}
}

func passesWorkspace(td ticketData, chip string) bool {
	if chip == "" {
		return true
	}
	return strings.EqualFold(td.Workspace, chip)
}

func passesStatus(td ticketData, cat string) bool {
	if cat == "" {
		return true
	}
	return strings.EqualFold(td.Ticket.StatusCategory, cat)
}

func findingTypeLabel(ft hygiene.FindingType) string {
	switch ft {
	case hygiene.MergedNotDone:
		return "PR merged — not Done"
	case hygiene.MissingEpic:
		return "Missing epic"
	case hygiene.MissingSprint:
		return "Missing sprint"
	case hygiene.MissingDueDate:
		return "Missing due date"
	case hygiene.StaleInProgress:
		return "Stale In Progress"
	case hygiene.EpicMissingDates:
		return "Epic missing dates"
	default:
		return string(ft)
	}
}

func groupFindingTypes(findings []hygiene.Finding) []hygiene.FindingType {
	seen := map[hygiene.FindingType]bool{}
	var order []hygiene.FindingType
	for _, f := range findings {
		if !seen[f.Type] {
			seen[f.Type] = true
			order = append(order, f.Type)
		}
	}
	sort.Slice(order, func(i, j int) bool { return order[i] < order[j] })
	return order
}

func countFindingsByType(findings []hygiene.Finding) map[hygiene.FindingType]int {
	out := map[hygiene.FindingType]int{}
	for _, f := range findings {
		out[f.Type]++
	}
	return out
}
