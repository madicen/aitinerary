package jira

import (
	"context"
	"fmt"
)

// RankIssues moves issues before rankBeforeIssue on the agile board.
func (c *Client) RankIssues(ctx context.Context, issues []string, rankBeforeIssue string) error {
	if len(issues) == 0 {
		return nil
	}
	body := map[string]any{"issues": issues}
	if rankBeforeIssue != "" {
		body["rankBeforeIssue"] = rankBeforeIssue
	}
	return c.doJSON(ctx, "PUT", "/rest/agile/1.0/issue/rank", body, nil)
}

// RankAfter moves issues immediately after rankAfterIssue.
func (c *Client) RankAfter(ctx context.Context, issues []string, rankAfterIssue string) error {
	if len(issues) == 0 {
		return nil
	}
	body := map[string]any{"issues": issues}
	if rankAfterIssue != "" {
		body["rankAfterIssue"] = rankAfterIssue
	}
	return c.doJSON(ctx, "PUT", "/rest/agile/1.0/issue/rank", body, nil)
}

// BoardTypeForProject returns "kanban", "scrum", or "" when unknown.
func (c *Client) BoardTypeForProject(ctx context.Context, projectKey string) (string, error) {
	boards, err := c.GetBoards(ctx, projectKey)
	if err != nil {
		return "", err
	}
	if len(boards) == 0 {
		return "", fmt.Errorf("no board for project %s", projectKey)
	}
	return boards[0].Type, nil
}
