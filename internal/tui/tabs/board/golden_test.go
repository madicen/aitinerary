package board

import (
	"testing"

	zone "github.com/lrstanley/bubblezone"

	"github.com/madicen/aitinerary/internal/jira"
	"github.com/madicen/aitinerary/internal/tui/tuitest"
)

func TestGoldenNeedsAttentionList(t *testing.T) {
	tuitest.ForceMonochrome(t)
	m := demoBoardModel(100, 28)
	m.zone = zone.New()
	m.detailOpen = false
	tuitest.AssertGolden(t, "needs_attention_list", m.View())
}

func TestGoldenDetailWithMatchedPR(t *testing.T) {
	tuitest.ForceMonochrome(t)
	m := demoBoardModel(100, 36)
	m.zone = zone.New()
	for i, r := range m.rows {
		if r.Kind == RowTicket && r.Data.Ticket.Key == "APP-101" {
			m.selected = i
			break
		}
	}
	m.transitionsFor = []jira.Transition{{ID: "1", Name: "Done", ToStatus: "Done"}}
	tuitest.AssertGolden(t, "detail_matched_pr", m.View())
}

func TestGoldenDoneCandidatesFilter(t *testing.T) {
	tuitest.ForceMonochrome(t)
	m := demoBoardModel(100, 28)
	m.zone = zone.New()
	m.filters.Preset = FilterDoneCandidates
	m.rebuildRows()
	m.detailOpen = false
	tuitest.AssertGolden(t, "done_candidates", m.View())
}
