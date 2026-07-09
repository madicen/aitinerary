package actions

import "testing"

func TestQueueAddRemoveCount(t *testing.T) {
	q := NewQueue()
	if q.Count() != 0 {
		t.Fatalf("empty queue count = %d", q.Count())
	}
	a := Action{ID: "a1", Kind: KindTransition, TicketKey: "APP-1", ToStatus: "Done"}
	q.Add(a)
	if q.Count() != 1 {
		t.Fatal("expected 1 pending")
	}
	if !q.Remove("a1") {
		t.Fatal("remove should find a1")
	}
	if q.Count() != 0 {
		t.Fatal("expected empty after remove")
	}
}

func TestQueueGroupedByKind(t *testing.T) {
	q := NewQueue()
	q.Add(
		Action{ID: "c1", Kind: KindAddComment, TicketKey: "APP-1"},
		Action{ID: "t1", Kind: KindTransition, TicketKey: "APP-2", ToStatus: "Done"},
		Action{ID: "c2", Kind: KindAddComment, TicketKey: "APP-3"},
	)
	sections := q.GroupedByKind()
	if len(sections) != 2 {
		t.Fatalf("expected 2 sections, got %d", len(sections))
	}
	if sections[0].Kind != KindAddComment {
		t.Fatalf("first section should be add_comment (first appearance), got %s", sections[0].Kind)
	}
	if sections[1].Kind != KindTransition {
		t.Fatalf("second section should be transition, got %s", sections[1].Kind)
	}
	if len(sections[1].Actions) != 1 {
		t.Fatalf("transition section: %+v", sections[1])
	}
}
