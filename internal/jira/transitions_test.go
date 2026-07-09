package jira

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestListAndDoTransition(t *testing.T) {
	c, st := newStubClient(func(req *http.Request, _ []byte) (*http.Response, error) {
		if req.Method == http.MethodGet {
			return jsonResponse(200, `{"transitions":[{"id":"31","name":"Done","to":{"name":"Done"}},{"id":"21","name":"Start","to":{"name":"In Progress"}}]}`, nil), nil
		}
		return jsonResponse(204, "", nil), nil
	})

	if err := c.TransitionTo(context.Background(), "APP-1", "Done"); err != nil {
		t.Fatalf("TransitionTo: %v", err)
	}
	// Last call is the POST executing transition 31.
	req := st.last()
	if req.Method != http.MethodPost || req.Path != "/rest/api/3/issue/APP-1/transitions" {
		t.Fatalf("method/path = %s %s", req.Method, req.Path)
	}
	var m map[string]any
	_ = json.Unmarshal(req.Body, &m)
	if m["transition"].(map[string]any)["id"] != "31" {
		t.Fatalf("transition body = %+v", m)
	}
}

func TestFindTransitionToPrefersTargetStatus(t *testing.T) {
	ts := []Transition{
		{ID: "1", Name: "Done", ToStatus: "Closed"},
		{ID: "2", Name: "Finish", ToStatus: "Done"},
	}
	got, ok := FindTransitionTo(ts, "Done")
	if !ok || got.ID != "2" {
		t.Fatalf("should match target status Done (id 2), got %+v ok=%v", got, ok)
	}
}

func TestTransitionToNoPathErrors(t *testing.T) {
	c, _ := newStubClient(func(_ *http.Request, _ []byte) (*http.Response, error) {
		return jsonResponse(200, `{"transitions":[{"id":"21","name":"Start","to":{"name":"In Progress"}}]}`, nil), nil
	})
	if err := c.TransitionTo(context.Background(), "APP-1", "Done"); err == nil {
		t.Fatal("expected error when no transition reaches Done")
	}
}
