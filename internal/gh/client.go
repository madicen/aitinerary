package gh

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	ghapi "github.com/cli/go-gh/v2/pkg/api"
	"github.com/madicen/aitinerary/internal/applog"
)

// restDoer is the subset of go-gh's *api.RESTClient this package depends on.
type restDoer interface {
	DoWithContext(ctx context.Context, method, path string, body io.Reader, response interface{}) error
	RequestWithContext(ctx context.Context, method, path string, body io.Reader) (*http.Response, error)
}

// graphQLDoer is the subset of go-gh's *api.GraphQLClient this package needs.
type graphQLDoer interface {
	DoWithContext(ctx context.Context, query string, variables map[string]interface{}, response interface{}) error
}

// ghClientTransport, when non-nil, overrides the HTTP transport used by the
// go-gh clients. Tests set it to a fake RoundTripper; production leaves it nil
// so go-gh resolves the same auth + host as the gh CLI.
var ghClientTransport http.RoundTripper

// clientOptions builds go-gh client options. When a test transport is installed
// we also supply a dummy token + host so go-gh skips auth/host resolution.
func clientOptions() ghapi.ClientOptions {
	if ghClientTransport != nil {
		return ghapi.ClientOptions{
			Transport: ghClientTransport,
			AuthToken: "test-token",
			Host:      "github.com",
		}
	}
	return ghapi.ClientOptions{}
}

var newRESTClient = func() (restDoer, error) {
	c, err := ghapi.NewRESTClient(clientOptions())
	if err != nil {
		return nil, err
	}
	return c, nil
}

var newGraphQLClient = func() (graphQLDoer, error) {
	c, err := ghapi.NewGraphQLClient(clientOptions())
	if err != nil {
		return nil, err
	}
	return c, nil
}

// graphQLQuery executes a GraphQL query and unmarshals the data object into T.
func graphQLQuery[T any](ctx context.Context, query string, vars map[string]any) (T, error) {
	var out T
	c, err := newGraphQLClient()
	if err != nil {
		return out, fmt.Errorf("gh graphql client: %w", err)
	}
	start := time.Now()
	derr := c.DoWithContext(ctx, query, vars, &out)
	applog.GHInvocation([]string{"api", "graphql"}, time.Since(start), derr)
	if derr != nil {
		var zero T
		return zero, derr
	}
	return out, nil
}

// ghAPIGet issues a REST GET and unmarshals the JSON response into out.
func ghAPIGet(ctx context.Context, apiPath string, out interface{}) error {
	c, err := newRESTClient()
	if err != nil {
		return fmt.Errorf("gh rest client: %w", err)
	}
	start := time.Now()
	derr := c.DoWithContext(ctx, http.MethodGet, apiPath, nil, out)
	applog.GHInvocation([]string{"api", apiPath}, time.Since(start), derr)
	if derr != nil {
		return apiErrorFrom(derr, apiPath)
	}
	return nil
}

// apiErrorFrom converts a go-gh transport error into *APIError.
func apiErrorFrom(err error, apiPath string) *APIError {
	if err == nil {
		return nil
	}
	ae := &APIError{APIPath: apiPath, RawBody: err.Error(), Message: err.Error()}
	var he *ghapi.HTTPError
	if errors.As(err, &he) {
		ae.Status = he.StatusCode
		if strings.TrimSpace(he.Message) != "" {
			ae.Message = he.Message
		}
		for _, it := range he.Errors {
			ae.Errors = append(ae.Errors, APIErrorItem{
				Resource: it.Resource,
				Code:     it.Code,
				Field:    it.Field,
				Message:  it.Message,
			})
		}
		if ra := retryAfterFromHeaders(he.Headers); ra > 0 {
			ae.RetryAfter = ra
		}
	}
	ae.HumanReason = inferHumanReason(ae)
	return ae
}

func retryAfterFromHeaders(h http.Header) time.Duration {
	if h == nil {
		return 0
	}
	v := strings.TrimSpace(h.Get("Retry-After"))
	if v == "" {
		return 0
	}
	if secs, err := strconv.Atoi(v); err == nil && secs >= 0 {
		return time.Duration(secs) * time.Second
	}
	if t, err := http.ParseTime(v); err == nil {
		if d := time.Until(t); d > 0 {
			return d
		}
	}
	return 0
}
