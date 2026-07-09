// Package jira is aitinerary's Jira Cloud REST v3 (+ agile 1.0) client.
//
// It builds on the transport/auth/ADF patterns proven in jj-tui's jira service
// and the APIError taxonomy from appr-ai-sal's GitHub layer, and adds the
// endpoints aitinerary needs that jj-tui lacked: runtime field discovery,
// boards/sprints, epic parenting, due/start dates, labels, and epic creation.
//
// The HTTP transport is injectable (WithTransport / WithHTTPClient) so every
// endpoint is exercised by hermetic stub-RoundTripper tests — no network.
package jira

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client talks to one Jira Cloud site using basic auth (email + API token).
type Client struct {
	baseURL string
	user    string
	token   string
	http    *http.Client
	ua      string
	fields  FieldIDs // discovered custom-field IDs used to parse/write issues.
}

// SetFieldIDs installs the custom-field IDs (from DiscoverFieldIDs or a config
// cache) the client uses to parse sprint/start-date fields and to write them.
func (c *Client) SetFieldIDs(f FieldIDs) { c.fields = f }

// FieldIDsCached returns the currently installed custom-field IDs.
func (c *Client) FieldIDsCached() FieldIDs { return c.fields }

// Option customises a Client at construction.
type Option func(*Client)

// WithHTTPClient sets the underlying *http.Client (timeouts, etc.).
func WithHTTPClient(h *http.Client) Option {
	return func(c *Client) {
		if h != nil {
			c.http = h
		}
	}
}

// WithTransport injects an http.RoundTripper — the seam hermetic tests use to
// stub Jira responses without a network.
func WithTransport(rt http.RoundTripper) Option {
	return func(c *Client) {
		if rt != nil {
			c.http = &http.Client{Transport: rt, Timeout: c.http.Timeout}
		}
	}
}

// WithUserAgent sets the User-Agent header sent on every request.
func WithUserAgent(ua string) Option {
	return func(c *Client) { c.ua = ua }
}

// New constructs a Client for baseURL authenticating as user with token. A
// trailing slash on baseURL is trimmed so paths concatenate cleanly.
func New(baseURL, user, token string, opts ...Option) *Client {
	c := &Client{
		baseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		user:    user,
		token:   token,
		http:    &http.Client{Timeout: 30 * time.Second},
		ua:      "aitinerary",
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

// BaseURL returns the site base URL (no trailing slash).
func (c *Client) BaseURL() string { return c.baseURL }

// BrowseURL returns the human browser URL for an issue key.
func (c *Client) BrowseURL(key string) string { return c.baseURL + "/browse/" + key }

// newRequest builds an authenticated request. body may be nil.
func (c *Client) newRequest(ctx context.Context, method, path string, body []byte) (*http.Request, error) {
	var r io.Reader
	if body != nil {
		r = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, r)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(c.user, c.token)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.ua != "" {
		req.Header.Set("User-Agent", c.ua)
	}
	return req, nil
}

// do executes a request and returns the raw response. The caller owns the body.
func (c *Client) do(ctx context.Context, method, path string, body []byte) (*http.Response, error) {
	req, err := c.newRequest(ctx, method, path, body)
	if err != nil {
		return nil, err
	}
	return c.http.Do(req)
}

// doJSON executes method path with an optional JSON request body and, on a 2xx
// response, decodes the response body into out (when out != nil). Non-2xx
// responses are mapped to *APIError. It is the single seam every endpoint funnels
// through, so auth, error mapping, and Retry-After parsing live in one place.
func (c *Client) doJSON(ctx context.Context, method, path string, reqBody, out any) error {
	var b []byte
	if reqBody != nil {
		var err error
		if b, err = json.Marshal(reqBody); err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}
	}
	resp, err := c.do(ctx, method, path, b)
	if err != nil {
		return fmt.Errorf("%s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return apiErrorFrom(resp.StatusCode, respBody, path, resp.Header)
	}
	if out != nil && len(bytes.TrimSpace(respBody)) > 0 {
		if err := json.Unmarshal(respBody, out); err != nil {
			return fmt.Errorf("decode %s response: %w", path, err)
		}
	}
	return nil
}

// CheckPermissions verifies the token can browse projects, the same startup
// gate jj-tui used. A 401 is reported as an auth failure; a present-but-false
// BROWSE_PROJECTS permission points the user at token regeneration.
func (c *Client) CheckPermissions(ctx context.Context) error {
	var result struct {
		Permissions map[string]struct {
			HavePermission bool `json:"havePermission"`
		} `json:"permissions"`
	}
	err := c.doJSON(ctx, http.MethodGet, "/rest/api/3/mypermissions?permissions=BROWSE_PROJECTS", nil, &result)
	if err != nil {
		if IsUnauthorized(err) {
			return fmt.Errorf("authentication failed — check your Jira email and API token: %w", err)
		}
		return fmt.Errorf("permission check failed: %w", err)
	}
	bp, ok := result.Permissions["BROWSE_PROJECTS"]
	if !ok {
		return fmt.Errorf("could not determine BROWSE_PROJECTS permission")
	}
	if !bp.HavePermission {
		return fmt.Errorf("API token lacks 'Browse Projects' permission — regenerate it at https://id.atlassian.com/manage-profile/security/api-tokens")
	}
	return nil
}
