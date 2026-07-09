package actions

import "testing"

func TestAutoApplyablePolicy(t *testing.T) {
	p := Policy{AutoDoneOnMerge: true, DoneStatusName: "Done"}
	cases := []struct {
		name string
		a    Action
		want bool
	}{
		{"high-conf auto transition to Done", Action{Kind: KindTransition, Source: SourceAuto, Confidence: ConfHigh, ToStatus: "Done"}, true},
		{"ambiguous transition queued", Action{Kind: KindTransition, Source: SourceAuto, Confidence: ConfAmbiguous, ToStatus: "Done"}, false},
		{"user transition queued", Action{Kind: KindTransition, Source: SourceUser, Confidence: ConfHigh, ToStatus: "Done"}, false},
		{"transition to non-Done queued", Action{Kind: KindTransition, Source: SourceAuto, Confidence: ConfHigh, ToStatus: "In Review"}, false},
		{"ai comment queued", Action{Kind: KindAddComment, Source: SourceAI, Confidence: ConfHigh}, false},
		{"set parent queued", Action{Kind: KindSetParent, Source: SourceAuto, Confidence: ConfHigh}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := p.AutoApplyable(tc.a); got != tc.want {
				t.Fatalf("AutoApplyable(%s) = %v, want %v", tc.name, got, tc.want)
			}
		})
	}
}

func TestAutoApplyableDisabledFlag(t *testing.T) {
	p := Policy{AutoDoneOnMerge: false, DoneStatusName: "Done"}
	a := Action{Kind: KindTransition, Source: SourceAuto, Confidence: ConfHigh, ToStatus: "Done"}
	if p.AutoApplyable(a) {
		t.Fatal("nothing should auto-apply when auto_done_on_merge is off")
	}
}

func TestPartition(t *testing.T) {
	p := Policy{AutoDoneOnMerge: true, DoneStatusName: "Done"}
	actions := []Action{
		{Kind: KindTransition, Source: SourceAuto, Confidence: ConfHigh, ToStatus: "Done"}, // auto
		{Kind: KindAddComment, Source: SourceAI},                                           // queued
		{Kind: KindTransition, Source: SourceAuto, Confidence: ConfAmbiguous, ToStatus: "Done"}, // queued
	}
	auto, queued := p.Partition(actions)
	if len(auto) != 1 || len(queued) != 2 {
		t.Fatalf("partition = auto %d queued %d", len(auto), len(queued))
	}
}

func TestDefaultDoneStatusName(t *testing.T) {
	// Empty DoneStatusName defaults to "Done".
	p := Policy{AutoDoneOnMerge: true}
	a := Action{Kind: KindTransition, Source: SourceAuto, Confidence: ConfHigh, ToStatus: "Done"}
	if !p.AutoApplyable(a) {
		t.Fatal("empty DoneStatusName should default to Done")
	}
}
