package jira

import (
	"context"
)

// CreateIssueSpec is a minimal issue create payload.
type CreateIssueSpec struct {
	Project     string
	IssueType   string
	Summary     string
	Description string
	Labels      []string
}

// CreateIssue creates a standard issue and returns its key.
func (c *Client) CreateIssue(ctx context.Context, spec CreateIssueSpec) (string, error) {
	fields := map[string]any{
		"project":   map[string]string{"key": spec.Project},
		"summary":   spec.Summary,
		"issuetype": map[string]string{"name": spec.IssueType},
	}
	if spec.Description != "" {
		fields["description"] = TextToADF(spec.Description)
	}
	if len(spec.Labels) > 0 {
		fields["labels"] = spec.Labels
	}
	body := map[string]any{"fields": fields}
	var out struct {
		Key string `json:"key"`
	}
	if err := c.doJSON(ctx, "POST", "/rest/api/3/issue", body, &out); err != nil {
		return "", err
	}
	return out.Key, nil
}
