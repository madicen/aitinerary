package match

import (
	"reflect"
	"testing"

	"github.com/madicen/aitinerary/internal/jira"
)

func TestExtractKeysBoundaries(t *testing.T) {
	re := KeyRegex([]string{"APP", "HELP"})
	cases := []struct {
		name string
		text string
		want []string
	}{
		{"single", "fix APP-1 done", []string{"APP-1"}},
		{"app-1 not prefix of app-11", "APP-11 only", []string{"APP-11"}},
		{"both distinct keys", "APP-1 and APP-11", []string{"APP-1", "APP-11"}},
		{"adjacent keys", "APP-1 APP-2", []string{"APP-1", "APP-2"}},
		{"lowercase branch", "app-42-fix-thing", []string{"APP-42"}},
		{"embedded in longer token ignored", "MAPP-1 xAPP-9", nil},
		{"other project ignored", "OTHER-5 here", nil},
		{"key inside url still extracted", "see https://x/browse/HELP-7)", []string{"HELP-7"}},
		{"trailing letter not a key", "APP-1a", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ExtractKeys(re, tc.text)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("ExtractKeys(%q) = %v, want %v", tc.text, got, tc.want)
			}
		})
	}
}

func liveSet(keys ...string) map[string]bool {
	m := map[string]bool{}
	for _, k := range keys {
		m[k] = true
	}
	return m
}

func TestMatchPRConfidence(t *testing.T) {
	re := KeyRegex([]string{"APP", "HELP"})

	t.Run("single branch key is high", func(t *testing.T) {
		pr := PR{HeadRef: "APP-5-add-widget", Title: "Add widget", Body: "closes APP-5"}
		got := MatchPR(re, pr, liveSet("APP-5"))
		if len(got) != 1 || got[0].Confidence != High || got[0].Source != SourceBranch {
			t.Fatalf("want single high branch match, got %+v", got)
		}
	})

	t.Run("body-only (e.g. URL mention) is ambiguous", func(t *testing.T) {
		pr := PR{HeadRef: "misc-cleanup", Body: "related https://x/browse/APP-9"}
		got := MatchPR(re, pr, liveSet("APP-9"))
		if len(got) != 1 || got[0].Confidence != Ambiguous || got[0].Source != SourceBody {
			t.Fatalf("want single ambiguous body match, got %+v", got)
		}
	})

	t.Run("multiple branch keys are ambiguous", func(t *testing.T) {
		pr := PR{HeadRef: "APP-1-APP-2-combo"}
		got := MatchPR(re, pr, liveSet("APP-1", "APP-2"))
		if len(got) != 2 {
			t.Fatalf("want 2 matches, got %+v", got)
		}
		for _, m := range got {
			if m.Confidence != Ambiguous {
				t.Fatalf("multi-key branch should be ambiguous: %+v", m)
			}
		}
	})

	t.Run("keys not in live set are dropped", func(t *testing.T) {
		pr := PR{HeadRef: "APP-5-thing"}
		got := MatchPR(re, pr, liveSet("APP-99"))
		if len(got) != 0 {
			t.Fatalf("want no matches for unknown/done ticket, got %+v", got)
		}
	})

	t.Run("branch source beats body for same key", func(t *testing.T) {
		pr := PR{HeadRef: "APP-5-x", Body: "APP-5 mentioned"}
		got := MatchPR(re, pr, liveSet("APP-5"))
		if len(got) != 1 || got[0].Source != SourceBranch {
			t.Fatalf("branch should win: %+v", got)
		}
	})
}

func TestMatchPRsExcludesDoneTickets(t *testing.T) {
	prs := []PR{{HeadRef: "APP-1-x"}, {HeadRef: "APP-2-y"}}
	tickets := []jira.Ticket{
		{Key: "APP-1", StatusCategory: "indeterminate"}, // live
		{Key: "APP-2", StatusCategory: "done"},          // already done → excluded
	}
	got := MatchPRs([]string{"APP"}, prs, tickets)
	if len(got) != 1 || got[0].Key != "APP-1" {
		t.Fatalf("done ticket should be excluded, got %+v", got)
	}
	if got[0].Confidence != High {
		t.Fatalf("single branch key should be high, got %v", got[0].Confidence)
	}
}
