package actions

import (
	"context"
	"fmt"
	"time"

	"github.com/madicen/aitinerary/internal/jira"
)

// Engine applies actions through an Executor and records them in a Journal so
// they can be undone.
type Engine struct {
	exec    Executor
	journal *Journal
	now     func() time.Time
}

// NewEngine builds an Engine. journal may be nil (applies are then not
// persisted, useful in tests that only assert executor effects).
func NewEngine(exec Executor, journal *Journal) *Engine {
	return &Engine{exec: exec, journal: journal, now: time.Now}
}

// ErrUndoUnavailable is returned by Undo when the change cannot be reversed
// automatically (e.g. the workflow has no transition back, or the field had no
// prior value). The journal entry is preserved so the user can act manually.
type ErrUndoUnavailable struct {
	Entry  JournalEntry
	Reason string
	Err    error
}

func (e *ErrUndoUnavailable) Error() string {
	msg := fmt.Sprintf("cannot auto-undo %s: %s", e.Entry.Action.Describe(), e.Reason)
	if e.Err != nil {
		msg += " (" + e.Err.Error() + ")"
	}
	return msg
}

func (e *ErrUndoUnavailable) Unwrap() error { return e.Err }

// Apply executes a and, on success, journals it with the prior state it
// replaced. The returned entry is also the one appended to the journal.
func (eng *Engine) Apply(ctx context.Context, a Action) (JournalEntry, error) {
	prior, err := eng.capturePrior(ctx, a)
	if err != nil {
		return JournalEntry{}, fmt.Errorf("capture prior state for %s: %w", a.TicketKey, err)
	}

	result, err := eng.execute(ctx, a)
	if err != nil {
		return JournalEntry{}, err
	}

	entry := JournalEntry{Time: eng.now(), Action: a, Prior: prior, Result: result}
	if eng.journal != nil {
		if jerr := eng.journal.Append(entry); jerr != nil {
			// The change already landed in Jira; a journal write failure must
			// not look like the action failed, but the caller should know undo
			// won't be possible.
			return entry, fmt.Errorf("action applied but journal write failed: %w", jerr)
		}
	}
	return entry, nil
}

// execute performs the mutation and returns any produced result (e.g. a created
// epic key).
func (eng *Engine) execute(ctx context.Context, a Action) (string, error) {
	switch a.Kind {
	case KindTransition:
		return "", eng.exec.Transition(ctx, a.TicketKey, a.ToStatus)
	case KindSetParent:
		return "", eng.exec.SetParent(ctx, a.TicketKey, a.ParentKey)
	case KindSetSprint:
		return "", eng.exec.AssignSprint(ctx, a.SprintID, []string{a.TicketKey})
	case KindSetDueDate:
		return "", eng.exec.SetDueDate(ctx, a.TicketKey, a.DueDate)
	case KindSetStartDate:
		return "", eng.exec.SetStartDate(ctx, a.TicketKey, a.StartDate)
	case KindSetLabels:
		return "", eng.exec.SetLabels(ctx, a.TicketKey, a.Labels, a.RemoveLabels)
	case KindAddComment:
		return "", eng.exec.AddComment(ctx, a.TicketKey, a.Comment)
	case KindCreateEpic:
		if a.Epic == nil {
			return "", fmt.Errorf("create_epic action missing epic payload")
		}
		spec := jira.EpicSpec{
			Project:         a.Epic.Project,
			Summary:         a.Epic.Summary,
			Description:     jira.BulletsToADF(a.Epic.BodyIntro, a.Epic.BodyBullets),
			StartDate:       a.Epic.StartDate,
			DueDate:         a.Epic.DueDate,
			RequireEpicName: a.Epic.RequireEpicName,
		}
		return eng.exec.CreateEpic(ctx, spec)
	default:
		return "", fmt.Errorf("unsupported action kind %q", a.Kind)
	}
}

// capturePrior reads the current ticket state for undoable, ticket-scoped
// actions. Non-undoable or ticket-less actions capture nothing.
func (eng *Engine) capturePrior(ctx context.Context, a Action) (PriorState, error) {
	if a.TicketKey == "" || !a.Undoable() {
		return PriorState{}, nil
	}
	t, err := eng.exec.GetTicket(ctx, a.TicketKey)
	if err != nil {
		return PriorState{}, err
	}
	ps := PriorState{
		Status:    t.Status,
		DueDate:   t.DueDate,
		StartDate: t.StartDate,
		ParentKey: t.ParentKey,
		Labels:    append([]string(nil), t.Labels...),
	}
	if t.Sprint != nil {
		ps.SprintID = t.Sprint.ID
	}
	return ps, nil
}

// Undo reverses an applied action using its journaled prior state. It returns
// an *ErrUndoUnavailable when the reversal cannot be performed automatically.
func (eng *Engine) Undo(ctx context.Context, entry JournalEntry) error {
	a := entry.Action
	if !a.Undoable() {
		return &ErrUndoUnavailable{Entry: entry, Reason: "this action kind cannot be reversed"}
	}
	switch a.Kind {
	case KindTransition:
		if entry.Prior.Status == "" {
			return &ErrUndoUnavailable{Entry: entry, Reason: "no prior status recorded"}
		}
		if err := eng.exec.Transition(ctx, a.TicketKey, entry.Prior.Status); err != nil {
			return &ErrUndoUnavailable{Entry: entry, Reason: "no workflow transition back to " + entry.Prior.Status, Err: err}
		}
		return nil
	case KindSetDueDate:
		return eng.exec.SetDueDate(ctx, a.TicketKey, entry.Prior.DueDate)
	case KindSetStartDate:
		return eng.exec.SetStartDate(ctx, a.TicketKey, entry.Prior.StartDate)
	case KindSetParent:
		if entry.Prior.ParentKey == "" {
			return &ErrUndoUnavailable{Entry: entry, Reason: "ticket had no prior epic; remove the parent manually"}
		}
		return eng.exec.SetParent(ctx, a.TicketKey, entry.Prior.ParentKey)
	case KindSetLabels:
		// Reverse: remove the labels we added, re-add the ones we removed.
		return eng.exec.SetLabels(ctx, a.TicketKey, a.RemoveLabels, a.Labels)
	case KindSetSprint:
		if entry.Prior.SprintID == 0 {
			return &ErrUndoUnavailable{Entry: entry, Reason: "ticket had no prior sprint; remove it from the sprint manually"}
		}
		return eng.exec.AssignSprint(ctx, entry.Prior.SprintID, []string{a.TicketKey})
	default:
		return &ErrUndoUnavailable{Entry: entry, Reason: "this action kind cannot be reversed"}
	}
}
