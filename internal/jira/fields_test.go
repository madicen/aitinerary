package jira

import (
	"context"
	"net/http"
	"testing"
)

const fieldsFixture = `[
  {"id":"summary","key":"summary","name":"Summary","custom":false,"schema":{"type":"string"}},
  {"id":"customfield_10020","key":"customfield_10020","name":"Sprint","custom":true,"schema":{"type":"array","custom":"com.pyxis.greenhopper.jira:gh-sprint"}},
  {"id":"customfield_10015","key":"customfield_10015","name":"Start date","custom":true,"schema":{"type":"date","custom":"com.atlassian.jpo:jpo-custom-field-baseline-start"}},
  {"id":"customfield_10011","key":"customfield_10011","name":"Epic Name","custom":true,"schema":{"type":"string","custom":"com.pyxis.greenhopper.jira:gh-epic-label"}},
  {"id":"customfield_10014","key":"customfield_10014","name":"Epic Link","custom":true,"schema":{"type":"any","custom":"com.pyxis.greenhopper.jira:gh-epic-link"}}
]`

func TestDiscoverFieldIDs(t *testing.T) {
	c, st := newStubClient(func(_ *http.Request, _ []byte) (*http.Response, error) {
		return jsonResponse(200, fieldsFixture, nil), nil
	})
	ids, err := c.DiscoverFieldIDs(context.Background())
	if err != nil {
		t.Fatalf("DiscoverFieldIDs: %v", err)
	}
	if ids.Sprint != "customfield_10020" {
		t.Errorf("Sprint = %q", ids.Sprint)
	}
	if ids.StartDate != "customfield_10015" {
		t.Errorf("StartDate = %q", ids.StartDate)
	}
	if ids.EpicName != "customfield_10011" {
		t.Errorf("EpicName = %q", ids.EpicName)
	}
	if ids.EpicLink != "customfield_10014" {
		t.Errorf("EpicLink = %q", ids.EpicLink)
	}
	if st.last().Path != "/rest/api/3/field" {
		t.Fatalf("path = %q", st.last().Path)
	}
}

func TestResolveFieldIDsGracefulAbsence(t *testing.T) {
	// A minimal instance exposing only Summary — everything else stays empty.
	ids := ResolveFieldIDs([]Field{{ID: "summary", Name: "Summary"}})
	if ids.Sprint != "" || ids.StartDate != "" || ids.EpicName != "" || ids.EpicLink != "" {
		t.Fatalf("expected all empty, got %+v", ids)
	}
}

func TestResolveFieldIDsByNameFallback(t *testing.T) {
	// No schema custom identifiers — must fall back to display-name matching.
	ids := ResolveFieldIDs([]Field{
		{ID: "cf1", Name: "Sprint", Custom: true},
		{ID: "cf2", Name: "Start date", Custom: true},
	})
	if ids.Sprint != "cf1" || ids.StartDate != "cf2" {
		t.Fatalf("name fallback failed: %+v", ids)
	}
}
