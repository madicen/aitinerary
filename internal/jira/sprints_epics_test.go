package jira

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestDiscoverBoardAndSprints(t *testing.T) {
	c, st := newStubClient(func(req *http.Request, _ []byte) (*http.Response, error) {
		switch {
		case strings.HasPrefix(req.URL.Path, "/rest/agile/1.0/board") && !strings.Contains(req.URL.Path, "sprint"):
			return jsonResponse(200, `{"values":[{"id":12,"name":"APP board","type":"scrum"}]}`, nil), nil
		default:
			return jsonResponse(200, `{"values":[{"id":7,"name":"Sprint 7","state":"active"}]}`, nil), nil
		}
	})

	b, err := c.DiscoverBoard(context.Background(), "APP")
	if err != nil {
		t.Fatalf("DiscoverBoard: %v", err)
	}
	if b.ID != 12 || b.Type != "scrum" {
		t.Fatalf("board = %+v", b)
	}
	if q := st.requests[0].Query; !strings.Contains(q, "projectKeyOrId=APP") {
		t.Fatalf("board query = %q", q)
	}

	sprints, err := c.GetSprints(context.Background(), 12, "active", "future")
	if err != nil {
		t.Fatalf("GetSprints: %v", err)
	}
	if len(sprints) != 1 || sprints[0].ID != 7 {
		t.Fatalf("sprints = %+v", sprints)
	}
	if p := st.last().Path; p != "/rest/agile/1.0/board/12/sprint" {
		t.Fatalf("sprint path = %q", p)
	}
	if q := st.last().Query; !strings.Contains(q, "state=active%2Cfuture") {
		t.Fatalf("sprint state query = %q", q)
	}
}

func TestAssignToSprintBody(t *testing.T) {
	c, st := newStubClient(func(_ *http.Request, _ []byte) (*http.Response, error) {
		return jsonResponse(204, "", nil), nil
	})
	if err := c.AssignToSprint(context.Background(), 7, []string{"APP-1", "APP-2"}); err != nil {
		t.Fatal(err)
	}
	req := st.last()
	if req.Method != http.MethodPost || req.Path != "/rest/agile/1.0/sprint/7/issue" {
		t.Fatalf("method/path = %s %s", req.Method, req.Path)
	}
	var m map[string]any
	_ = json.Unmarshal(req.Body, &m)
	issues := m["issues"].([]any)
	if len(issues) != 2 || issues[0] != "APP-1" {
		t.Fatalf("issues body = %+v", issues)
	}
}

func TestCreateEpicOmitsEpicNameWhenNotRequired(t *testing.T) {
	c, st := newStubClient(func(_ *http.Request, _ []byte) (*http.Response, error) {
		return jsonResponse(201, `{"key":"APP-200"}`, nil), nil
	})
	c.SetFieldIDs(FieldIDs{StartDate: "customfield_10015", EpicName: "customfield_10011"})

	key, err := c.CreateEpic(context.Background(), EpicSpec{
		Project:         "APP",
		Summary:         "Retro epic",
		Description:     BulletsToADF("Covers:", []string{"x", "y"}),
		StartDate:       "2026-07-15",
		DueDate:         "2026-08-15",
		RequireEpicName: false,
	})
	if err != nil {
		t.Fatal(err)
	}
	if key != "APP-200" {
		t.Fatalf("key = %q", key)
	}
	var m map[string]any
	_ = json.Unmarshal(st.last().Body, &m)
	fields := m["fields"].(map[string]any)
	if fields["duedate"] != "2026-08-15" {
		t.Fatalf("duedate = %+v", fields["duedate"])
	}
	if fields["customfield_10015"] != "2026-07-15" {
		t.Fatalf("start date field missing: %+v", fields)
	}
	if _, present := fields["customfield_10011"]; present {
		t.Fatalf("Epic Name should be omitted when not required: %+v", fields)
	}
}

func TestCreateEpicIncludesEpicNameWhenRequired(t *testing.T) {
	c, st := newStubClient(func(_ *http.Request, _ []byte) (*http.Response, error) {
		return jsonResponse(201, `{"key":"APP-201"}`, nil), nil
	})
	c.SetFieldIDs(FieldIDs{EpicName: "customfield_10011"})
	if _, err := c.CreateEpic(context.Background(), EpicSpec{Project: "APP", Summary: "E", RequireEpicName: true}); err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	_ = json.Unmarshal(st.last().Body, &m)
	fields := m["fields"].(map[string]any)
	if fields["customfield_10011"] != "E" {
		t.Fatalf("Epic Name should be present when required: %+v", fields)
	}
}

func TestAddCommentBody(t *testing.T) {
	c, st := newStubClient(func(_ *http.Request, _ []byte) (*http.Response, error) {
		return jsonResponse(201, `{"id":"1"}`, nil), nil
	})
	if err := c.AddComment(context.Background(), "APP-1", TextToADF("done")); err != nil {
		t.Fatal(err)
	}
	req := st.last()
	if req.Path != "/rest/api/3/issue/APP-1/comment" {
		t.Fatalf("path = %q", req.Path)
	}
	if !strings.Contains(string(req.Body), `"body"`) || !strings.Contains(string(req.Body), `"doc"`) {
		t.Fatalf("comment body should carry ADF doc: %s", req.Body)
	}
}
