package actions

import "strings"

// Policy decides which actions may be applied automatically versus queued for
// review. Per PLAN.md §7 P5.2 the ONLY auto-applied action is a
// Transition→Done that came from the merged-PR detector (Source auto) with a
// high-confidence single-PR match, and only when auto_done_on_merge is enabled.
// Everything AI-authored or ambiguous is queued.
type Policy struct {
	AutoDoneOnMerge bool
	DoneStatusName  string
}

// AutoApplyable reports whether a may be applied without review.
func (p Policy) AutoApplyable(a Action) bool {
	if !p.AutoDoneOnMerge {
		return false
	}
	if a.Kind != KindTransition {
		return false
	}
	if a.Source != SourceAuto {
		return false
	}
	if a.Confidence != ConfHigh {
		return false
	}
	done := p.DoneStatusName
	if done == "" {
		done = "Done"
	}
	return strings.EqualFold(a.ToStatus, done)
}

// Partition splits actions into those that may auto-apply and those that must
// be queued for review, preserving order within each bucket.
func (p Policy) Partition(actions []Action) (auto, queued []Action) {
	for _, a := range actions {
		if p.AutoApplyable(a) {
			auto = append(auto, a)
		} else {
			queued = append(queued, a)
		}
	}
	return auto, queued
}
