package jira

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// WorkspaceQuery describes one workspace's ticket selection. The board maps a
// config.Workspace onto this; the jira package stays independent of config.
type WorkspaceQuery struct {
	Project             string   // Jira project key (required).
	Labels              []string // ALL of these labels must be present.
	AssigneeCurrentUser bool     // add `assignee = currentUser()`.
	ExcludedStatuses    []string // statuses to exclude (e.g. Won't Fix).
	JQLExtra            string   // advanced clause, AND-ed verbatim.
}

// BuildJQL renders a WorkspaceQuery to a JQL string ending in
// `ORDER BY updated DESC`. Values are quoted defensively.
func BuildJQL(q WorkspaceQuery) string {
	var conds []string
	if p := strings.TrimSpace(q.Project); p != "" {
		conds = append(conds, fmt.Sprintf("project = %s", quoteJQL(p)))
	}
	for _, l := range q.Labels {
		if l = strings.TrimSpace(l); l != "" {
			conds = append(conds, fmt.Sprintf("labels = %s", quoteJQL(l)))
		}
	}
	if q.AssigneeCurrentUser {
		conds = append(conds, "assignee = currentUser()")
	}
	if len(q.ExcludedStatuses) > 0 {
		quoted := make([]string, 0, len(q.ExcludedStatuses))
		for _, s := range q.ExcludedStatuses {
			if s = strings.TrimSpace(s); s != "" {
				quoted = append(quoted, quoteJQL(s))
			}
		}
		if len(quoted) > 0 {
			conds = append(conds, fmt.Sprintf("status NOT IN (%s)", strings.Join(quoted, ", ")))
		}
	}
	if extra := strings.TrimSpace(q.JQLExtra); extra != "" {
		conds = append(conds, "("+extra+")")
	}
	return strings.Join(conds, " AND ") + " ORDER BY updated DESC"
}

// quoteJQL wraps a value in double quotes, escaping embedded quotes/backslashes.
func quoteJQL(v string) string {
	v = strings.ReplaceAll(v, `\`, `\\`)
	v = strings.ReplaceAll(v, `"`, `\"`)
	return `"` + v + `"`
}

// searchResponse is the enhanced /search/jql payload. It is token-paginated and
// deliberately carries no total.
type searchResponse struct {
	Issues        []rawIssue `json:"issues"`
	NextPageToken string     `json:"nextPageToken"`
	IsLast        *bool      `json:"isLast"`
}

// SearchPage runs one page of a JQL search via GET /rest/api/3/search/jql,
// returning the parsed tickets and the token for the next page ("" when done).
func (c *Client) SearchPage(ctx context.Context, jql string, maxResults int, pageToken string) ([]Ticket, string, error) {
	if maxResults <= 0 {
		maxResults = 50
	}
	q := url.Values{}
	q.Set("jql", jql)
	q.Set("maxResults", itoa(maxResults))
	q.Set("fields", strings.Join(c.requestFields(), ","))
	if pageToken != "" {
		q.Set("nextPageToken", pageToken)
	}
	var resp searchResponse
	if err := c.doJSON(ctx, http.MethodGet, "/rest/api/3/search/jql?"+q.Encode(), nil, &resp); err != nil {
		return nil, "", err
	}
	out := make([]Ticket, 0, len(resp.Issues))
	for _, raw := range resp.Issues {
		out = append(out, parseIssue(raw, c.fields))
	}
	return out, resp.NextPageToken, nil
}

// SearchAll runs a JQL search to exhaustion, following nextPageToken. A safety
// cap of 100 pages guards against a server that never signals the last page.
func (c *Client) SearchAll(ctx context.Context, jql string) ([]Ticket, error) {
	var all []Ticket
	token := ""
	for page := 0; page < 100; page++ {
		tickets, next, err := c.SearchPage(ctx, jql, 100, token)
		if err != nil {
			return nil, err
		}
		all = append(all, tickets...)
		if next == "" {
			break
		}
		token = next
	}
	return all, nil
}

// SearchWorkspace builds the JQL for q and returns all matching tickets.
func (c *Client) SearchWorkspace(ctx context.Context, q WorkspaceQuery) ([]Ticket, error) {
	return c.SearchAll(ctx, BuildJQL(q))
}
