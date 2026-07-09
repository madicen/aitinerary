// Package gh is aitinerary's GitHub integration layer. Auth, host, and tokens
// resolve through the user's gh CLI configuration; REST and GraphQL traffic runs
// in-process via go-gh.
package gh

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/madicen/aitinerary/internal/match"
)

// PR is a merged pull request used for ticket-key matching.
type PR struct {
	Number     int
	Title      string
	URL        string
	Repository string // "owner/name"
	Body       string
	HeadRef    string // branch name
	MergedAt   time.Time
}

// ToMatch converts to the minimal PR view used by internal/match.
func (p PR) ToMatch() match.PR {
	return match.PR{
		Number:  p.Number,
		Repo:    p.Repository,
		Title:   p.Title,
		Body:    p.Body,
		HeadRef: p.HeadRef,
		URL:     p.URL,
	}
}

// CheckAuth returns nil if gh is installed, recent enough, and the user is logged in.
func CheckAuth() error {
	if _, err := exec.LookPath("gh"); err != nil {
		return fmt.Errorf("gh not found on PATH; install from https://cli.github.com")
	}
	if err := checkGHVersion(); err != nil {
		return err
	}
	return checkAuthViaAPI(context.Background())
}

var (
	viewerLoginMu    sync.RWMutex
	viewerLoginCache string
)

func cacheViewerLogin(login string) {
	login = strings.TrimSpace(login)
	if login == "" {
		return
	}
	viewerLoginMu.Lock()
	viewerLoginCache = login
	viewerLoginMu.Unlock()
}

func cachedViewerLogin() string {
	viewerLoginMu.RLock()
	defer viewerLoginMu.RUnlock()
	return viewerLoginCache
}

// ViewerLogin returns the GitHub login for the authenticated gh user.
func ViewerLogin(ctx context.Context) (string, error) {
	if v := cachedViewerLogin(); v != "" {
		return v, nil
	}
	var resp struct {
		Login string `json:"login"`
	}
	if err := ghAPIGet(ctx, "user", &resp); err != nil {
		return "", err
	}
	login := strings.TrimSpace(resp.Login)
	cacheViewerLogin(login)
	return login, nil
}

func checkAuthViaAPI(ctx context.Context) error {
	if _, err := ViewerLogin(ctx); err != nil {
		return fmt.Errorf("gh auth: not logged in or token invalid — run `gh auth login`: %w", err)
	}
	return nil
}

// MergedSearchQuery builds the GitHub search query for merged PRs authored by
// the viewer since the given date. repos, when non-empty, restrict the search
// to those owner/name pairs.
func MergedSearchQuery(repos []string, since time.Time) string {
	parts := []string{
		"is:pr",
		"is:merged",
		"author:@me",
		fmt.Sprintf("merged:>=%s", since.Format("2006-01-02")),
		"archived:false",
	}
	for _, r := range repos {
		if r = strings.TrimSpace(r); r != "" {
			parts = append(parts, "repo:"+r)
		}
	}
	return strings.Join(parts, " ")
}

const graphqlMergedPRQuery = `query($q: String!) {
  search(query: $q, type: ISSUE, first: 100) {
    pageInfo { hasNextPage }
    nodes {
      ... on PullRequest {
        number
        title
        url
        body
        headRefName
        mergedAt
        repository { nameWithOwner }
      }
    }
  }
}`

type graphqlMergedPRData struct {
	Search struct {
		PageInfo pageInfo           `json:"pageInfo"`
		Nodes    []graphqlMergedNode `json:"nodes"`
	} `json:"search"`
}

type pageInfo struct {
	HasNextPage bool `json:"hasNextPage"`
}

type graphqlMergedNode struct {
	Number     int    `json:"number"`
	Title      string `json:"title"`
	URL        string `json:"url"`
	Body       string `json:"body"`
	HeadRefName string `json:"headRefName"`
	MergedAt   string `json:"mergedAt"`
	Repository struct {
		NameWithOwner string `json:"nameWithOwner"`
	} `json:"repository"`
}

// ListMergedPRs returns merged PRs authored by the viewer since sinceDays ago.
// When repos is non-empty, the search is restricted to those repositories.
func ListMergedPRs(ctx context.Context, repos []string, sinceDays int) ([]PR, error) {
	if sinceDays < 1 {
		sinceDays = 30
	}
	since := time.Now().AddDate(0, 0, -sinceDays)
	q := MergedSearchQuery(repos, since)
	data, err := graphQLQuery[graphqlMergedPRData](ctx, graphqlMergedPRQuery, map[string]any{"q": q})
	if err != nil {
		return nil, fmt.Errorf("list merged PRs: %w", err)
	}
	out := make([]PR, 0, len(data.Search.Nodes))
	for _, n := range data.Search.Nodes {
		mergedAt, _ := time.Parse(time.RFC3339, n.MergedAt)
		out = append(out, PR{
			Number:     n.Number,
			Title:      strings.TrimSpace(n.Title),
			URL:        strings.TrimSpace(n.URL),
			Repository: strings.TrimSpace(n.Repository.NameWithOwner),
			Body:       n.Body,
			HeadRef:    strings.TrimSpace(n.HeadRefName),
			MergedAt:   mergedAt,
		})
	}
	return out, nil
}

// GHAvailable reports whether the gh binary is on PATH. Tests use this to
// decide whether to run integration-style checks.
func GHAvailable() bool {
	_, err := exec.LookPath("gh")
	return err == nil
}
