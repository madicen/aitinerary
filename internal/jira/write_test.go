package jira

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func decodeBody(t *testing.T, b []byte) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("body not JSON: %v (%s)", err, b)
	}
	return m
}

func TestSetDueDateBody(t *testing.T) {
	c, st := newStubClient(func(_ *http.Request, _ []byte) (*http.Response, error) {
		return jsonResponse(204, "", nil), nil
	})
	if err := c.SetDueDate(context.Background(), "APP-1", "2026-08-01"); err != nil {
		t.Fatal(err)
	}
	req := st.last()
	if req.Method != http.MethodPut || req.Path != "/rest/api/3/issue/APP-1" {
		t.Fatalf("method/path = %s %s", req.Method, req.Path)
	}
	m := decodeBody(t, req.Body)
	fields := m["fields"].(map[string]any)
	if fields["duedate"] != "2026-08-01" {
		t.Fatalf("duedate body = %+v", fields)
	}
}

func TestSetStartDateRequiresDiscoveredField(t *testing.T) {
	c, _ := newStubClient(func(_ *http.Request, _ []byte) (*http.Response, error) {
		return jsonResponse(204, "", nil), nil
	})
	if err := c.SetStartDate(context.Background(), "APP-1", "2026-07-15"); err == nil {
		t.Fatal("expected error when start-date field not discovered")
	}
	c.SetFieldIDs(FieldIDs{StartDate: "customfield_10015"})
	if err := c.SetStartDate(context.Background(), "APP-1", "2026-07-15"); err != nil {
		t.Fatal(err)
	}
}

func TestSetLabelsUpdateVerb(t *testing.T) {
	c, st := newStubClient(func(_ *http.Request, _ []byte) (*http.Response, error) {
		return jsonResponse(204, "", nil), nil
	})
	if err := c.SetLabels(context.Background(), "APP-1", []string{"Apps"}, []string{"old"}); err != nil {
		t.Fatal(err)
	}
	m := decodeBody(t, st.last().Body)
	update := m["update"].(map[string]any)
	ops := update["labels"].([]any)
	if len(ops) != 2 {
		t.Fatalf("want 2 label ops, got %+v", ops)
	}
	if ops[0].(map[string]any)["add"] != "Apps" || ops[1].(map[string]any)["remove"] != "old" {
		t.Fatalf("label ops wrong: %+v", ops)
	}
}

func TestSetParentUsesModernField(t *testing.T) {
	c, st := newStubClient(func(_ *http.Request, _ []byte) (*http.Response, error) {
		return jsonResponse(204, "", nil), nil
	})
	method, err := c.SetParent(context.Background(), "APP-1", "APP-100")
	if err != nil {
		t.Fatal(err)
	}
	if method != ParentViaField {
		t.Fatalf("method = %q, want parent", method)
	}
	m := decodeBody(t, st.last().Body)
	parent := m["fields"].(map[string]any)["parent"].(map[string]any)
	if parent["key"] != "APP-100" {
		t.Fatalf("parent body = %+v", parent)
	}
}

func TestSetParentFallsBackToEpicLink(t *testing.T) {
	calls := 0
	c, st := newStubClient(func(_ *http.Request, body []byte) (*http.Response, error) {
		calls++
		m := map[string]any{}
		_ = json.Unmarshal(body, &m)
		fields, _ := m["fields"].(map[string]any)
		if _, usesParent := fields["parent"]; usesParent {
			// First attempt with the modern field is rejected naming `parent`.
			return jsonResponse(400, `{"errors":{"parent":"Field 'parent' cannot be set"}}`, nil), nil
		}
		return jsonResponse(204, "", nil), nil
	})
	c.SetFieldIDs(FieldIDs{EpicLink: "customfield_10014"})

	method, err := c.SetParent(context.Background(), "APP-1", "APP-100")
	if err != nil {
		t.Fatalf("fallback should succeed: %v", err)
	}
	if method != ParentViaEpicLink {
		t.Fatalf("method = %q, want epic_link", method)
	}
	if calls != 2 {
		t.Fatalf("expected 2 attempts, got %d", calls)
	}
	// Final request wrote the Epic Link custom field.
	m := decodeBody(t, st.last().Body)
	if m["fields"].(map[string]any)["customfield_10014"] != "APP-100" {
		t.Fatalf("epic link body = %+v", m)
	}
}
