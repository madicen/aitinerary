package jira

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestGetIssueParsesAllFields(t *testing.T) {
	body := `{
      "key":"APP-42",
      "fields":{
        "summary":"Wire the board list",
        "status":{"name":"In Progress","statusCategory":{"key":"indeterminate"}},
        "issuetype":{"name":"Story","subtask":false},
        "priority":{"name":"High"},
        "labels":["Apps","frontend"],
        "duedate":"2026-08-01",
        "parent":{"key":"APP-10","fields":{"summary":"Board epic"}},
        "assignee":{"displayName":"Michael Madicen","emailAddress":"m@x.com"},
        "updated":"2026-07-01T09:30:00.000-0400",
        "description":{"type":"doc","version":1,"content":[{"type":"paragraph","content":[{"type":"text","text":"hello"}]}]},
        "customfield_10020":[{"id":7,"name":"Sprint 7","state":"active"}],
        "customfield_10015":"2026-07-15"
      }
    }`
	c, st := newStubClient(func(_ *http.Request, _ []byte) (*http.Response, error) {
		return jsonResponse(200, body, nil), nil
	})
	c.SetFieldIDs(FieldIDs{Sprint: "customfield_10020", StartDate: "customfield_10015"})

	tk, err := c.GetIssue(context.Background(), "APP-42")
	if err != nil {
		t.Fatalf("GetIssue: %v", err)
	}
	if tk.Summary != "Wire the board list" || tk.Status != "In Progress" || tk.StatusCategory != "indeterminate" {
		t.Fatalf("status/summary wrong: %+v", tk)
	}
	if tk.Type != "Story" || tk.IsSubtask {
		t.Fatalf("type wrong: %+v", tk)
	}
	if len(tk.Labels) != 2 || tk.Labels[0] != "Apps" {
		t.Fatalf("labels wrong: %+v", tk.Labels)
	}
	if tk.DueDate != "2026-08-01" || tk.StartDate != "2026-07-15" {
		t.Fatalf("dates wrong: due=%q start=%q", tk.DueDate, tk.StartDate)
	}
	if tk.ParentKey != "APP-10" || tk.ParentSummary != "Board epic" {
		t.Fatalf("parent wrong: %+v", tk)
	}
	if tk.Assignee != "Michael Madicen" {
		t.Fatalf("assignee wrong: %q", tk.Assignee)
	}
	if tk.Sprint == nil || tk.Sprint.Name != "Sprint 7" || tk.Sprint.State != "active" {
		t.Fatalf("sprint wrong: %+v", tk.Sprint)
	}
	if tk.Description != "hello" {
		t.Fatalf("description flatten wrong: %q", tk.Description)
	}
	if tk.Updated.Year() != 2026 || tk.Updated.Month() != 7 {
		t.Fatalf("updated parse wrong: %v", tk.Updated)
	}
	if tk.IsDone() {
		t.Fatalf("In Progress ticket should not be Done")
	}
	// The request must ask for the custom fields too.
	if q := st.last().Query; !strings.Contains(q, "customfield_10020") || !strings.Contains(q, "customfield_10015") {
		t.Fatalf("fields query missing custom fields: %q", q)
	}
}

func TestParseIssueToleratesMissingFields(t *testing.T) {
	// An issue with an empty field bag must parse without panicking and yield
	// only its key.
	tk := parseIssue(rawIssue{Key: "APP-1", Fields: map[string]json.RawMessage{}}, FieldIDs{})
	if tk.Key != "APP-1" {
		t.Fatalf("key = %q", tk.Key)
	}
	if tk.Sprint != nil || tk.Description != "" || len(tk.Labels) != 0 {
		t.Fatalf("expected zero values, got %+v", tk)
	}
}

func TestParseAssignedSprintPrefersActive(t *testing.T) {
	raw := json.RawMessage(`[{"id":1,"name":"S1","state":"closed"},{"id":2,"name":"S2","state":"active"}]`)
	sp := parseAssignedSprint(raw)
	if sp == nil || sp.Name != "S2" {
		t.Fatalf("should prefer active sprint, got %+v", sp)
	}
	if parseAssignedSprint(json.RawMessage("null")) != nil {
		t.Fatal("null sprint field should be nil")
	}
}
