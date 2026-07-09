package jira

import (
	"context"
	"net/http"
	"strings"
)

// Field is one entry from GET /rest/api/3/field.
type Field struct {
	ID     string `json:"id"`
	Key    string `json:"key"`
	Name   string `json:"name"`
	Custom bool   `json:"custom"`
	Schema struct {
		Type   string `json:"type"`
		Custom string `json:"custom"` // e.g. com.pyxis.greenhopper.jira:gh-sprint
	} `json:"schema"`
}

// FieldIDs holds the custom-field IDs aitinerary resolves at runtime. Any of
// these may be empty when the instance does not expose that field (e.g. Epic
// Name is not required on team-managed projects) — callers must degrade
// gracefully rather than hardcode IDs.
type FieldIDs struct {
	Sprint    string
	StartDate string
	EpicName  string
	EpicLink  string
}

// Well-known Greenhopper custom schema identifiers. These are stable across
// Jira Cloud instances even though the numeric customfield_ ID is not, so we
// match on them first and fall back to display-name matching.
const (
	schemaSprint    = "com.pyxis.greenhopper.jira:gh-sprint"
	schemaEpicName  = "com.pyxis.greenhopper.jira:gh-epic-label"
	schemaEpicLink  = "com.pyxis.greenhopper.jira:gh-epic-link"
	schemaStartDate = "com.atlassian.jpo:jpo-custom-field-baseline-start"
)

// GetFields returns all fields visible to the authenticated user.
func (c *Client) GetFields(ctx context.Context) ([]Field, error) {
	var fields []Field
	if err := c.doJSON(ctx, http.MethodGet, "/rest/api/3/field", nil, &fields); err != nil {
		return nil, err
	}
	return fields, nil
}

// DiscoverFieldIDs resolves the Sprint, Start date, Epic Name, and Epic Link
// field IDs from the instance's field catalog. Missing fields yield empty
// strings (graceful absence). Matching prefers the stable Greenhopper/JPO
// schema identifier and falls back to a case-insensitive display-name match.
func (c *Client) DiscoverFieldIDs(ctx context.Context) (FieldIDs, error) {
	fields, err := c.GetFields(ctx)
	if err != nil {
		return FieldIDs{}, err
	}
	return ResolveFieldIDs(fields), nil
}

// ResolveFieldIDs is the pure matching core of DiscoverFieldIDs, split out so
// it can be unit-tested against a fixture field list without a client.
func ResolveFieldIDs(fields []Field) FieldIDs {
	var ids FieldIDs
	for _, f := range fields {
		switch {
		case ids.Sprint == "" && (f.Schema.Custom == schemaSprint || eqFold(f.Name, "Sprint")):
			ids.Sprint = f.ID
		case ids.StartDate == "" && (f.Schema.Custom == schemaStartDate || eqFold(f.Name, "Start date")):
			ids.StartDate = f.ID
		case ids.EpicName == "" && (f.Schema.Custom == schemaEpicName || eqFold(f.Name, "Epic Name")):
			ids.EpicName = f.ID
		case ids.EpicLink == "" && (f.Schema.Custom == schemaEpicLink || eqFold(f.Name, "Epic Link")):
			ids.EpicLink = f.ID
		}
	}
	return ids
}

func eqFold(a, b string) bool { return strings.EqualFold(strings.TrimSpace(a), b) }
