package gh

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	ghapi "github.com/cli/go-gh/v2/pkg/api"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// stubGHResponder installs a fake HTTP transport for the duration of the test.
func stubGHResponder(t *testing.T, respond func(*http.Request) (int, string)) {
	t.Helper()
	prev := ghClientTransport
	ghClientTransport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		status, body := respond(r)
		return &http.Response{
			StatusCode: status,
			Status:     http.StatusText(status),
			Header:     http.Header{"Content-Type": {"application/json"}},
			Body:       io.NopCloser(strings.NewReader(body)),
			Request:    r,
		}, nil
	})
	t.Cleanup(func() { ghClientTransport = prev })
}

func stubGraphQL(t *testing.T, payload string) {
	t.Helper()
	stubGHResponder(t, func(*http.Request) (int, string) { return http.StatusOK, payload })
}

func TestGraphQLQueryHelper(t *testing.T) {
	type viewerData struct {
		Viewer struct {
			Login string `json:"login"`
		} `json:"viewer"`
	}

	payload := `{"data":{"viewer":{"login":"octocat"}}}`
	stubGHResponder(t, func(*http.Request) (int, string) { return http.StatusOK, payload })

	got, err := graphQLQuery[viewerData](context.Background(), "query{viewer{login}}", nil)
	if err != nil {
		t.Fatalf("graphQLQuery: %v", err)
	}
	if got.Viewer.Login != "octocat" {
		t.Fatalf("login = %q, want octocat", got.Viewer.Login)
	}

	payload = `{"errors":[{"message":"bad query"}]}`
	if _, err := graphQLQuery[viewerData](context.Background(), "query{viewer{login}}", nil); err == nil {
		t.Fatal("expected error from graphql errors payload")
	}
}

func TestApiErrorFromHTTPError(t *testing.T) {
	he := &ghapi.HTTPError{
		StatusCode: 422,
		Message:    "Validation Failed",
		Errors: []ghapi.HTTPErrorItem{{
			Resource: "PullRequestReviewComment",
			Code:     "custom",
			Field:    "pull_request_review_thread.line",
			Message:  "could not be resolved",
		}},
		Headers: http.Header{"Retry-After": {"30"}},
	}
	ae := apiErrorFrom(he, "repos/o/r/pulls/1/comments")
	if ae == nil {
		t.Fatal("apiErrorFrom returned nil")
	}
	if ae.Status != 422 {
		t.Errorf("Status = %d, want 422", ae.Status)
	}
	if ae.RetryAfter != 30*time.Second {
		t.Errorf("RetryAfter = %v, want 30s", ae.RetryAfter)
	}
}

func TestApiErrorFromNonHTTPError(t *testing.T) {
	ae := apiErrorFrom(errors.New("connection refused"), "repos/o/r/pulls/1")
	if ae == nil {
		t.Fatal("apiErrorFrom returned nil for a plain error")
	}
	if ae.Status != 0 {
		t.Errorf("Status = %d, want 0", ae.Status)
	}
}

func TestCheckAuthViaAPIRequiresViewer(t *testing.T) {
	viewerLoginMu.Lock()
	prev := viewerLoginCache
	viewerLoginCache = ""
	viewerLoginMu.Unlock()
	t.Cleanup(func() {
		viewerLoginMu.Lock()
		viewerLoginCache = prev
		viewerLoginMu.Unlock()
	})

	stubGHResponder(t, func(r *http.Request) (int, string) {
		if r.Method != http.MethodGet || !strings.HasSuffix(r.URL.Path, "/user") {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		return http.StatusUnauthorized, `{"message":"Bad credentials"}`
	})
	if err := checkAuthViaAPI(context.Background()); err == nil {
		t.Fatal("expected auth error")
	}
}

func TestMergedSearchQuery(t *testing.T) {
	since := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
	got := MergedSearchQuery([]string{"stackadapt/foo"}, since)
	want := "is:pr is:merged author:@me merged:>=2026-01-15 archived:false repo:stackadapt/foo"
	if got != want {
		t.Fatalf("query = %q, want %q", got, want)
	}
}

func TestListMergedPRsGraphQL(t *testing.T) {
	payload := `{
  "data": {
    "search": {
      "pageInfo": {"hasNextPage": false},
      "nodes": [{
        "number": 42,
        "title": "APP-101 fix widget",
        "url": "https://github.com/o/r/pull/42",
        "body": "Closes APP-101",
        "headRefName": "feature/APP-101-widget",
        "mergedAt": "2026-06-01T12:00:00Z",
        "repository": {"nameWithOwner": "o/r"}
      }]
    }
  }
}`
	stubGraphQL(t, payload)
	prs, err := ListMergedPRs(context.Background(), nil, 30)
	if err != nil {
		t.Fatalf("ListMergedPRs: %v", err)
	}
	if len(prs) != 1 {
		t.Fatalf("len = %d, want 1", len(prs))
	}
	pr := prs[0]
	if pr.Number != 42 || pr.HeadRef != "feature/APP-101-widget" {
		t.Fatalf("unexpected PR: %+v", pr)
	}
	if pr.Repository != "o/r" {
		t.Fatalf("Repository = %q", pr.Repository)
	}
	m := pr.ToMatch()
	if m.HeadRef != "feature/APP-101-widget" || m.Repo != "o/r" {
		t.Fatalf("ToMatch: %+v", m)
	}
}

func TestCheckAuthIntegration(t *testing.T) {
	if !GHAvailable() {
		t.Skip("gh not available on PATH")
	}
	if err := CheckAuth(); err != nil {
		t.Skipf("gh auth not configured: %v", err)
	}
}

func TestListMergedPRsIntegration(t *testing.T) {
	if !GHAvailable() {
		t.Skip("gh not available on PATH")
	}
	if err := CheckAuth(); err != nil {
		t.Skipf("gh auth not configured: %v", err)
	}
	prs, err := ListMergedPRs(context.Background(), nil, 7)
	if err != nil {
		t.Fatalf("ListMergedPRs: %v", err)
	}
	t.Logf("found %d merged PRs in last 7 days", len(prs))
}
