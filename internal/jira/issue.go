package jira

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

// Ticket is aitinerary's normalized view of a Jira issue. It carries exactly
// the fields the board, hygiene detectors, and actions engine need.
type Ticket struct {
	Key            string
	Summary        string
	Status         string
	StatusCategory string // "new" | "indeterminate" | "done" (Jira status category key)
	Type           string
	IsSubtask      bool
	Priority       string
	Labels         []string
	DueDate        string // ISO "2006-01-02", "" when unset
	StartDate      string // ISO date from the discovered start-date field, "" when unset
	ParentKey      string
	ParentSummary  string
	Sprint         *Sprint // the assigned active/future sprint, nil when none
	Assignee       string  // display name
	AssigneeEmail  string
	Updated        time.Time
	Description    string  // flattened plain text
	DescriptionADF *ADFDoc // raw ADF, preserved for potential round-trips
}

// IsDone reports whether the ticket sits in Jira's "done" status category.
func (t Ticket) IsDone() bool { return t.StatusCategory == "done" }

// SearchFields is the field list aitinerary requests for board tickets. The
// two custom fields (sprint, start date) are appended at call time from the
// discovered FieldIDs.
var SearchFields = []string{
	"key", "summary", "status", "priority", "issuetype",
	"labels", "duedate", "parent", "assignee", "updated", "description",
}

// requestFields returns SearchFields plus any discovered custom-field IDs.
func (c *Client) requestFields() []string {
	f := append([]string(nil), SearchFields...)
	if c.fields.Sprint != "" {
		f = append(f, c.fields.Sprint)
	}
	if c.fields.StartDate != "" {
		f = append(f, c.fields.StartDate)
	}
	return f
}

// rawIssue is the wire shape: a key plus an untyped field bag so custom fields
// (whose IDs vary per instance) can be extracted dynamically.
type rawIssue struct {
	Key    string                     `json:"key"`
	Fields map[string]json.RawMessage `json:"fields"`
}

// GetIssue fetches a single issue and parses it into a Ticket using the
// client's cached FieldIDs.
func (c *Client) GetIssue(ctx context.Context, key string) (*Ticket, error) {
	path := "/rest/api/3/issue/" + key + "?fields=" + strings.Join(c.requestFields(), ",")
	var raw rawIssue
	if err := c.doJSON(ctx, http.MethodGet, path, nil, &raw); err != nil {
		return nil, err
	}
	t := parseIssue(raw, c.fields)
	return &t, nil
}

// parseIssue converts a rawIssue into a Ticket. Absent fields leave zero
// values; malformed individual fields are skipped rather than failing the whole
// parse, so one odd ticket never breaks a board load.
func parseIssue(raw rawIssue, fids FieldIDs) Ticket {
	t := Ticket{Key: raw.Key}
	f := raw.Fields

	unmarshalField(f, "summary", &t.Summary)
	t.Labels = parseStringSlice(f["labels"])
	unmarshalDate(f, "duedate", &t.DueDate)

	if v, ok := f["status"]; ok {
		var s struct {
			Name           string `json:"name"`
			StatusCategory struct {
				Key string `json:"key"`
			} `json:"statusCategory"`
		}
		if json.Unmarshal(v, &s) == nil {
			t.Status = s.Name
			t.StatusCategory = s.StatusCategory.Key
		}
	}
	if v, ok := f["issuetype"]; ok {
		var it struct {
			Name    string `json:"name"`
			Subtask bool   `json:"subtask"`
		}
		if json.Unmarshal(v, &it) == nil {
			t.Type = it.Name
			t.IsSubtask = it.Subtask
		}
	}
	if v, ok := f["priority"]; ok {
		var p struct {
			Name string `json:"name"`
		}
		if json.Unmarshal(v, &p) == nil {
			t.Priority = p.Name
		}
	}
	if v, ok := f["parent"]; ok {
		var p struct {
			Key    string `json:"key"`
			Fields struct {
				Summary string `json:"summary"`
			} `json:"fields"`
		}
		if json.Unmarshal(v, &p) == nil {
			t.ParentKey = p.Key
			t.ParentSummary = p.Fields.Summary
		}
	}
	if v, ok := f["assignee"]; ok {
		var a struct {
			DisplayName  string `json:"displayName"`
			EmailAddress string `json:"emailAddress"`
		}
		if json.Unmarshal(v, &a) == nil {
			t.Assignee = a.DisplayName
			t.AssigneeEmail = a.EmailAddress
		}
	}
	if v, ok := f["updated"]; ok {
		var s string
		if json.Unmarshal(v, &s) == nil {
			t.Updated = parseJiraTime(s)
		}
	}
	if v, ok := f["description"]; ok && len(v) > 0 && string(v) != "null" {
		var doc ADFDoc
		if json.Unmarshal(v, &doc) == nil {
			t.DescriptionADF = &doc
			t.Description = ADFToText(&doc)
		}
	}

	// Custom fields — only when discovered.
	if fids.Sprint != "" {
		t.Sprint = parseAssignedSprint(f[fids.Sprint])
	}
	if fids.StartDate != "" {
		unmarshalDate(f, fids.StartDate, &t.StartDate)
	}
	return t
}

// parseAssignedSprint extracts the "current" sprint from the Sprint custom
// field, which Jira returns as an array of sprint objects. It prefers an
// active sprint, else the last (most recent) entry, else nil.
func parseAssignedSprint(raw json.RawMessage) *Sprint {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	var arr []Sprint
	if err := json.Unmarshal(raw, &arr); err != nil || len(arr) == 0 {
		return nil
	}
	for i := range arr {
		if strings.EqualFold(arr[i].State, "active") {
			return &arr[i]
		}
	}
	return &arr[len(arr)-1]
}

func unmarshalField(f map[string]json.RawMessage, key string, out *string) {
	if v, ok := f[key]; ok && len(v) > 0 && string(v) != "null" {
		_ = json.Unmarshal(v, out)
	}
}

// unmarshalDate reads a JSON string date, tolerating both bare ISO dates and
// full timestamps (it keeps only the yyyy-mm-dd prefix).
func unmarshalDate(f map[string]json.RawMessage, key string, out *string) {
	var s string
	unmarshalField(f, key, &s)
	if len(s) >= 10 {
		*out = s[:10]
	} else {
		*out = s
	}
}

func parseStringSlice(raw json.RawMessage) []string {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	var out []string
	_ = json.Unmarshal(raw, &out)
	return out
}

// parseJiraTime parses Jira's timestamp format (RFC3339 with a numeric zone
// offset like +0000), falling back to RFC3339.
func parseJiraTime(s string) time.Time {
	for _, layout := range []string{"2006-01-02T15:04:05.999-0700", time.RFC3339} {
		if t, err := time.Parse(layout, s); err == nil {
			return t
		}
	}
	return time.Time{}
}
