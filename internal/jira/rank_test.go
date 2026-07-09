package jira

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestRankIssuesBody(t *testing.T) {
	c, st := newStubClient(func(req *http.Request, body []byte) (*http.Response, error) {
		if req.Method != http.MethodPut || req.URL.Path != "/rest/agile/1.0/issue/rank" {
			t.Fatalf("unexpected %s %s", req.Method, req.URL.Path)
		}
		var got map[string]any
		if err := json.Unmarshal(body, &got); err != nil {
			t.Fatal(err)
		}
		issues, _ := got["issues"].([]any)
		if len(issues) != 2 {
			t.Fatalf("issues = %v", got["issues"])
		}
		if got["rankBeforeIssue"] != "APP-99" {
			t.Fatalf("rankBeforeIssue = %v", got["rankBeforeIssue"])
		}
		return jsonResponse(204, "", nil), nil
	})
	if err := c.RankIssues(context.Background(), []string{"APP-1", "APP-2"}, "APP-99"); err != nil {
		t.Fatal(err)
	}
	if st.last().Method != http.MethodPut {
		t.Fatalf("method = %s", st.last().Method)
	}
}

func TestBoardTypeForProject(t *testing.T) {
	c, _ := newStubClient(func(_ *http.Request, _ []byte) (*http.Response, error) {
		return jsonResponse(200, `{"values":[{"id":1,"name":"Help","type":"kanban"}]}`, nil), nil
	})
	typ, err := c.BoardTypeForProject(context.Background(), "HELP")
	if err != nil {
		t.Fatal(err)
	}
	if typ != "kanban" {
		t.Fatalf("type = %q", typ)
	}
}
