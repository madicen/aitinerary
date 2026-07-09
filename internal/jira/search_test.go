package jira

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestBuildJQLAppWorkspace(t *testing.T) {
	got := BuildJQL(WorkspaceQuery{
		Project:             "APP",
		AssigneeCurrentUser: true,
		ExcludedStatuses:    []string{"Won't Fix", "Will Not Do"},
	})
	want := `project = "APP" AND assignee = currentUser() AND status NOT IN ("Won't Fix", "Will Not Do") ORDER BY updated DESC`
	if got != want {
		t.Fatalf("APP JQL =\n%q\nwant\n%q", got, want)
	}
}

func TestBuildJQLHelpWorkspaceWithLabels(t *testing.T) {
	got := BuildJQL(WorkspaceQuery{
		Project:             "HELP",
		Labels:              []string{"Apps"},
		AssigneeCurrentUser: true,
	})
	want := `project = "HELP" AND labels = "Apps" AND assignee = currentUser() ORDER BY updated DESC`
	if got != want {
		t.Fatalf("HELP JQL =\n%q\nwant\n%q", got, want)
	}
}

func TestBuildJQLExtraClause(t *testing.T) {
	got := BuildJQL(WorkspaceQuery{Project: "APP", JQLExtra: "priority = High"})
	if !strings.Contains(got, `(priority = High)`) {
		t.Fatalf("extra clause not wrapped: %q", got)
	}
}

func TestSearchPaginationFollowsToken(t *testing.T) {
	page := 0
	c, st := newStubClient(func(req *http.Request, _ []byte) (*http.Response, error) {
		page++
		if req.URL.Query().Get("nextPageToken") == "" {
			return jsonResponse(200, `{"issues":[{"key":"APP-1","fields":{"summary":"one"}}],"nextPageToken":"tok2"}`, nil), nil
		}
		return jsonResponse(200, `{"issues":[{"key":"APP-2","fields":{"summary":"two"}}],"nextPageToken":""}`, nil), nil
	})
	tickets, err := c.SearchAll(context.Background(), "project = APP ORDER BY updated DESC")
	if err != nil {
		t.Fatalf("SearchAll: %v", err)
	}
	if len(tickets) != 2 || tickets[0].Key != "APP-1" || tickets[1].Key != "APP-2" {
		t.Fatalf("unexpected tickets: %+v", tickets)
	}
	if page != 2 {
		t.Fatalf("expected 2 pages, got %d", page)
	}
	// The first page request carries the jql and the enhanced endpoint path.
	first := st.requests[0]
	if first.Path != "/rest/api/3/search/jql" {
		t.Fatalf("path = %q", first.Path)
	}
	q, _ := url.ParseQuery(first.Query)
	if !strings.Contains(q.Get("jql"), "project = APP") {
		t.Fatalf("jql param missing: %q", first.Query)
	}
	if q.Get("fields") == "" {
		t.Fatal("fields param should be set")
	}
}
