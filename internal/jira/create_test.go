package jira

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

func TestCreateIssueIncludesLabels(t *testing.T) {
	c, st := newStubClient(func(req *http.Request, body []byte) (*http.Response, error) {
		if req.Method != http.MethodPost {
			t.Fatalf("method = %s", req.Method)
		}
		if !strings.Contains(string(body), `"Apps"`) {
			t.Fatalf("body missing Apps label: %s", body)
		}
		return jsonResponse(201, `{"key":"HELP-999"}`, nil), nil
	})
	key, err := c.CreateIssue(context.Background(), CreateIssueSpec{
		Project: "HELP", IssueType: "Task", Summary: "IRE item", Labels: []string{"Apps"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if key != "HELP-999" {
		t.Fatalf("key = %q", key)
	}
	_ = st
}
