package jira

import (
	"context"
	"fmt"
	"net/http"
)

// editIssue PUTs a fields/update payload to an issue. Jira returns 204.
func (c *Client) editIssue(ctx context.Context, key string, payload map[string]any) error {
	return c.doJSON(ctx, http.MethodPut, "/rest/api/3/issue/"+key, payload, nil)
}

// SetDueDate sets the issue due date (ISO "2006-01-02"). An empty date clears
// it (Jira accepts null).
func (c *Client) SetDueDate(ctx context.Context, key, isoDate string) error {
	var val any
	if isoDate != "" {
		val = isoDate
	}
	return c.editIssue(ctx, key, map[string]any{"fields": map[string]any{"duedate": val}})
}

// SetStartDate sets the timeline start date via the discovered start-date field.
// It errors when the field was never discovered so callers surface a clear
// message rather than silently no-op.
func (c *Client) SetStartDate(ctx context.Context, key, isoDate string) error {
	if c.fields.StartDate == "" {
		return fmt.Errorf("start-date field not discovered; run field discovery first")
	}
	var val any
	if isoDate != "" {
		val = isoDate
	}
	return c.editIssue(ctx, key, map[string]any{"fields": map[string]any{c.fields.StartDate: val}})
}

// SetLabels adds and/or removes labels using the update verb form, which is
// non-destructive to labels the caller didn't mention.
func (c *Client) SetLabels(ctx context.Context, key string, add, remove []string) error {
	var ops []map[string]string
	for _, l := range add {
		ops = append(ops, map[string]string{"add": l})
	}
	for _, l := range remove {
		ops = append(ops, map[string]string{"remove": l})
	}
	if len(ops) == 0 {
		return nil
	}
	return c.editIssue(ctx, key, map[string]any{"update": map[string]any{"labels": ops}})
}

// ParentMethod reports which mechanism SetParent used, for journaling / ADR.
type ParentMethod string

const (
	ParentViaField    ParentMethod = "parent"    // modern parent field
	ParentViaEpicLink ParentMethod = "epic_link" // legacy Epic Link custom field
)

// SetParent sets an issue's parent (epic). It first tries the modern `parent`
// field, which works for both team- and company-managed projects on current
// Jira Cloud. If the instance rejects it with a 400 that names `parent` and an
// Epic Link field was discovered, it falls back to writing the Epic Link custom
// field (PLAN.md §6). It returns which method succeeded.
func (c *Client) SetParent(ctx context.Context, key, parentKey string) (ParentMethod, error) {
	err := c.editIssue(ctx, key, map[string]any{"fields": map[string]any{"parent": map[string]string{"key": parentKey}}})
	if err == nil {
		return ParentViaField, nil
	}
	if IsBadRequest(err) && MentionsField(err, "parent") && c.fields.EpicLink != "" {
		if ferr := c.editIssue(ctx, key, map[string]any{"fields": map[string]any{c.fields.EpicLink: parentKey}}); ferr != nil {
			return "", fmt.Errorf("parent field rejected and Epic Link fallback failed: %w", ferr)
		}
		return ParentViaEpicLink, nil
	}
	return "", err
}
