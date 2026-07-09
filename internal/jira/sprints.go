package jira

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// Board is a Jira agile board. Type is "scrum" or "kanban" — the hygiene layer
// skips the MissingSprint detector for kanban boards (PLAN.md §9).
type Board struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

// Sprint is an agile sprint. State is "active", "future", or "closed".
type Sprint struct {
	ID            int    `json:"id"`
	Name          string `json:"name"`
	State         string `json:"state"`
	StartDate     string `json:"startDate,omitempty"`
	EndDate       string `json:"endDate,omitempty"`
	OriginBoardID int    `json:"originBoardId,omitempty"`
}

// GetBoards lists agile boards for a project key.
func (c *Client) GetBoards(ctx context.Context, projectKeyOrID string) ([]Board, error) {
	var resp struct {
		Values []Board `json:"values"`
	}
	path := "/rest/agile/1.0/board?projectKeyOrId=" + url.QueryEscape(projectKeyOrID)
	if err := c.doJSON(ctx, http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Values, nil
}

// DiscoverBoard returns the first board for a project, the value board_ids=0
// resolves to. It errors when the project has no board (e.g. a pure
// company-managed project without an agile board).
func (c *Client) DiscoverBoard(ctx context.Context, projectKey string) (Board, error) {
	boards, err := c.GetBoards(ctx, projectKey)
	if err != nil {
		return Board{}, err
	}
	if len(boards) == 0 {
		return Board{}, fmt.Errorf("no agile board found for project %s", projectKey)
	}
	return boards[0], nil
}

// GetSprints lists sprints for a board, filtered to the given states (e.g.
// "active", "future"). With no states it returns all.
func (c *Client) GetSprints(ctx context.Context, boardID int, states ...string) ([]Sprint, error) {
	path := fmt.Sprintf("/rest/agile/1.0/board/%d/sprint", boardID)
	if len(states) > 0 {
		path += "?state=" + url.QueryEscape(strings.Join(states, ","))
	}
	var resp struct {
		Values []Sprint `json:"values"`
	}
	if err := c.doJSON(ctx, http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Values, nil
}

// AssignToSprint moves issues into a sprint. This agile endpoint is preferred
// over editing the Sprint custom field directly (PLAN.md §6).
func (c *Client) AssignToSprint(ctx context.Context, sprintID int, keys []string) error {
	if len(keys) == 0 {
		return nil
	}
	body := map[string]any{"issues": keys}
	path := fmt.Sprintf("/rest/agile/1.0/sprint/%d/issue", sprintID)
	return c.doJSON(ctx, http.MethodPost, path, body, nil)
}
