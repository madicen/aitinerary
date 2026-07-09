package jira

import (
	"context"
	"net/http"
)

// EpicSpec describes an epic to create. StartDate/DueDate are ISO dates and
// both should be set so the epic lands correctly on Jira's timeline view.
type EpicSpec struct {
	Project     string
	Summary     string
	Description *ADFDoc
	StartDate   string // ISO date; written to the discovered start-date field.
	DueDate     string // ISO date.
	// RequireEpicName mirrors what field discovery reports: some company-managed
	// instances require the Epic Name field on create; team-managed ones reject
	// it. When true and an Epic Name field was discovered, it is included.
	RequireEpicName bool
}

// CreateEpic creates an Epic issue and returns its key. Epic Name is included
// only when the instance requires it and the field was discovered — otherwise
// sending it errors on team-managed projects (PLAN.md §6/§7 P1.5).
func (c *Client) CreateEpic(ctx context.Context, spec EpicSpec) (string, error) {
	fields := map[string]any{
		"project":   map[string]string{"key": spec.Project},
		"issuetype": map[string]string{"name": "Epic"},
		"summary":   spec.Summary,
	}
	if spec.Description != nil {
		fields["description"] = spec.Description
	}
	if spec.DueDate != "" {
		fields["duedate"] = spec.DueDate
	}
	if spec.StartDate != "" && c.fields.StartDate != "" {
		fields[c.fields.StartDate] = spec.StartDate
	}
	if spec.RequireEpicName && c.fields.EpicName != "" {
		fields[c.fields.EpicName] = spec.Summary
	}

	var resp struct {
		Key string `json:"key"`
	}
	if err := c.doJSON(ctx, http.MethodPost, "/rest/api/3/issue", map[string]any{"fields": fields}, &resp); err != nil {
		return "", err
	}
	return resp.Key, nil
}

// GetEpics returns the epics in a project (issuetype = Epic), most-recently
// updated first, parsed as Tickets.
func (c *Client) GetEpics(ctx context.Context, projectKey string) ([]Ticket, error) {
	jql := BuildJQL(WorkspaceQuery{Project: projectKey, JQLExtra: "issuetype = Epic"})
	return c.SearchAll(ctx, jql)
}

// AddComment posts an ADF comment to an issue (used for AI status updates).
func (c *Client) AddComment(ctx context.Context, key string, body *ADFDoc) error {
	return c.doJSON(ctx, http.MethodPost, "/rest/api/3/issue/"+key+"/comment", map[string]any{"body": body}, nil)
}
