package actions

import (
	"strings"
	"sync"
)

// Queue holds actions awaiting user review before apply.
type Queue struct {
	mu      sync.RWMutex
	pending []Action
}

// NewQueue returns an empty review queue.
func NewQueue() *Queue { return &Queue{} }

// Pending returns a copy of queued actions.
func (q *Queue) Pending() []Action {
	if q == nil {
		return nil
	}
	q.mu.RLock()
	defer q.mu.RUnlock()
	out := make([]Action, len(q.pending))
	copy(out, q.pending)
	return out
}

// Count reports how many actions are waiting for review.
func (q *Queue) Count() int {
	if q == nil {
		return 0
	}
	q.mu.RLock()
	defer q.mu.RUnlock()
	return len(q.pending)
}

// Enqueue appends an action to the review queue.
func (q *Queue) Enqueue(a Action) { q.Add(a) }

// Add appends actions to the review queue.
func (q *Queue) Add(actions ...Action) {
	if q == nil {
		return
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	q.pending = append(q.pending, actions...)
}

// Remove drops the action with the given id. Returns true when found.
func (q *Queue) Remove(id string) bool {
	if q == nil {
		return false
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	for i, a := range q.pending {
		if a.ID == id {
			q.pending = append(q.pending[:i], q.pending[i+1:]...)
			return true
		}
	}
	return false
}

// RemoveAll clears every pending action.
func (q *Queue) RemoveAll() {
	if q == nil {
		return
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	q.pending = nil
}

// QueueSection groups pending actions of one kind for display.
type QueueSection struct {
	Kind    Kind
	Actions []Action
}

// GroupedByKind returns pending actions grouped by Kind, preserving kind
// order of first appearance.
func (q *Queue) GroupedByKind() []QueueSection {
	if q == nil {
		return nil
	}
	pending := q.Pending()
	if len(pending) == 0 {
		return nil
	}
	order := make([]Kind, 0)
	byKind := map[Kind][]Action{}
	for _, a := range pending {
		if _, ok := byKind[a.Kind]; !ok {
			order = append(order, a.Kind)
		}
		byKind[a.Kind] = append(byKind[a.Kind], a)
	}
	out := make([]QueueSection, 0, len(order))
	for _, k := range order {
		out = append(out, QueueSection{Kind: k, Actions: byKind[k]})
	}
	return out
}

// KindLabel returns a human group title for queue list headers.
func KindLabel(k Kind) string {
	switch k {
	case KindTransition:
		return "Transitions"
	case KindSetParent:
		return "Set parent"
	case KindSetSprint:
		return "Set sprint"
	case KindSetDueDate:
		return "Set due date"
	case KindSetStartDate:
		return "Set start date"
	case KindAddComment:
		return "Comments"
	case KindCreateEpic:
		return "Create epic"
	case KindSetLabels:
		return "Set labels"
	case KindRank:
		return "Rank"
	default:
		return strings.ReplaceAll(string(k), "_", " ")
	}
}
