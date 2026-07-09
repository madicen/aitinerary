// Package actions is aitinerary's action queue and executor (PLAN.md §7 P5).
//
// Every change aitinerary makes to Jira is modelled as an Action. Safe actions
// (a high-confidence Transition→Done from a single merged PR) may auto-apply;
// everything AI-authored or ambiguous waits in a review queue. Applying an
// action journals enough prior state to undo it, and undo reverses field and
// transition changes (looking the reverse transition up live, degrading to
// manual guidance when the workflow has no path back).
//
// The executor is an interface so the engine is tested with a fake — no network.
package actions

import (
	"fmt"
	"strings"
	"time"
)

// Kind enumerates the mutations aitinerary can perform.
type Kind string

const (
	KindTransition  Kind = "transition"
	KindSetParent   Kind = "set_parent"
	KindSetSprint   Kind = "set_sprint"
	KindSetDueDate  Kind = "set_due_date"
	KindSetStartDate Kind = "set_start_date"
	KindAddComment  Kind = "add_comment"
	KindCreateEpic  Kind = "create_epic"
	KindCreateTicket Kind = "create_ticket"
	KindSetLabels   Kind = "set_labels"
	KindRank        Kind = "rank"
)

// Source records who proposed an action; it drives the auto-apply policy and
// the review-queue grouping.
type Source string

const (
	SourceAuto Source = "auto" // the merged-PR detector.
	SourceAI   Source = "ai"   // an AI-authored comment / epic draft.
	SourceUser Source = "user" // a direct user edit.
)

// Confidence is the matcher's confidence carried onto the action.
type Confidence string

const (
	ConfHigh      Confidence = "high"
	ConfAmbiguous Confidence = "ambiguous"
)

// EpicPayload carries the fields for a KindCreateEpic action.
type EpicPayload struct {
	Project         string
	Summary         string
	BodyIntro       string
	BodyBullets     []string
	StartDate       string
	DueDate         string
	RequireEpicName bool
	// SetParentOn lists issue keys to reparent onto the newly-created epic once
	// it exists (the epic builder flow, P8.1).
	SetParentOn []string
}

// TicketPayload carries fields for KindCreateTicket (IRE quick-capture).
type TicketPayload struct {
	Project     string
	IssueType   string
	Summary     string
	Description string
	Labels      []string
}

// Action is one proposed or applied mutation.
type Action struct {
	ID         string     `json:"id"`
	Kind       Kind       `json:"kind"`
	TicketKey  string     `json:"ticket_key,omitempty"` // target; empty for CreateEpic.
	Source     Source     `json:"source"`
	Confidence Confidence `json:"confidence,omitempty"`
	Summary    string     `json:"summary,omitempty"` // human description for the queue.

	// Payload (per Kind):
	ToStatus     string       `json:"to_status,omitempty"`      // Transition
	ParentKey    string       `json:"parent_key,omitempty"`     // SetParent
	SprintID     int          `json:"sprint_id,omitempty"`      // SetSprint
	DueDate      string       `json:"due_date,omitempty"`       // SetDueDate
	StartDate    string       `json:"start_date,omitempty"`     // SetStartDate
	Comment      string       `json:"comment,omitempty"`        // AddComment (plain text)
	Labels       []string     `json:"labels,omitempty"`         // SetLabels add
	RemoveLabels []string     `json:"remove_labels,omitempty"`  // SetLabels remove
	RankBefore   string       `json:"rank_before,omitempty"`    // Rank
	Epic         *EpicPayload   `json:"epic,omitempty"`           // CreateEpic
	Ticket       *TicketPayload `json:"ticket,omitempty"`         // CreateTicket

	// PRRef is a "repo#number" back-reference for auto Transition→Done actions.
	PRRef string `json:"pr_ref,omitempty"`
}

// Undoable reports whether an applied action of this kind can be reversed by
// the engine. Comments, epic creation, and rank are not cleanly reversible
// with aitinerary's API surface.
func (a Action) Undoable() bool {
	switch a.Kind {
	case KindTransition, KindSetParent, KindSetSprint, KindSetDueDate, KindSetStartDate, KindSetLabels:
		return true
	default:
		return false
	}
}

// Describe returns a one-line human description, used by the queue and journal.
func (a Action) Describe() string {
	if a.Summary != "" {
		return a.Summary
	}
	switch a.Kind {
	case KindTransition:
		return fmt.Sprintf("%s → %s", a.TicketKey, a.ToStatus)
	case KindSetParent:
		return fmt.Sprintf("%s: set epic %s", a.TicketKey, a.ParentKey)
	case KindSetSprint:
		return fmt.Sprintf("%s: assign to sprint %d", a.TicketKey, a.SprintID)
	case KindSetDueDate:
		return fmt.Sprintf("%s: due %s", a.TicketKey, a.DueDate)
	case KindSetStartDate:
		return fmt.Sprintf("%s: start %s", a.TicketKey, a.StartDate)
	case KindAddComment:
		return fmt.Sprintf("%s: comment", a.TicketKey)
	case KindCreateEpic:
		if a.Epic != nil {
			return fmt.Sprintf("create epic %q in %s", a.Epic.Summary, a.Epic.Project)
		}
		return "create epic"
	case KindCreateTicket:
		if a.Ticket != nil {
			return fmt.Sprintf("create %s in %s", a.Ticket.Summary, a.Ticket.Project)
		}
		return "create ticket"
	case KindSetLabels:
		return fmt.Sprintf("%s: labels +%s -%s", a.TicketKey, strings.Join(a.Labels, ","), strings.Join(a.RemoveLabels, ","))
	case KindRank:
		return fmt.Sprintf("%s: rank before %s", a.TicketKey, a.RankBefore)
	default:
		return string(a.Kind)
	}
}

// NewID returns a monotonic-ish action id from t and a short kind tag. Kept
// simple (time-based) so the engine has no external id dependency.
func NewID(kind Kind, t time.Time) string {
	return fmt.Sprintf("%s-%d", kind, t.UnixNano())
}
