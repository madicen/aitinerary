package jira

import (
	"bytes"
	"io"
	"net/http"
)

// capturedRequest records everything a hermetic test asserts about an outgoing
// call: method, path, query, and decoded body.
type capturedRequest struct {
	Method string
	Path   string
	Query  string
	Header http.Header
	Body   []byte
	Auth   string // Authorization header value.
}

// stubTransport is an http.RoundTripper that records each request and replies
// from a queue of canned responses (or a handler function). It performs no
// network I/O, satisfying the hermetic-tests convention.
type stubTransport struct {
	// handler, when set, produces the response for each request.
	handler func(req *http.Request, body []byte) (*http.Response, error)
	// requests accumulates every call for post-hoc assertions.
	requests []capturedRequest
}

func (s *stubTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	var body []byte
	if req.Body != nil {
		body, _ = io.ReadAll(req.Body)
		_ = req.Body.Close()
	}
	s.requests = append(s.requests, capturedRequest{
		Method: req.Method,
		Path:   req.URL.Path,
		Query:  req.URL.RawQuery,
		Header: req.Header.Clone(),
		Body:   body,
		Auth:   req.Header.Get("Authorization"),
	})
	return s.handler(req, body)
}

func (s *stubTransport) last() capturedRequest {
	if len(s.requests) == 0 {
		return capturedRequest{}
	}
	return s.requests[len(s.requests)-1]
}

// jsonResponse builds a canned *http.Response with a JSON body and optional
// headers.
func jsonResponse(status int, body string, header http.Header) *http.Response {
	if header == nil {
		header = http.Header{}
	}
	header.Set("Content-Type", "application/json")
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(bytes.NewBufferString(body)),
		Header:     header,
	}
}

// newStubClient wires a Client to a stubTransport whose handler always returns
// resp for the given status/body. For multi-call or path-sensitive tests, set
// st.handler directly.
func newStubClient(handler func(req *http.Request, body []byte) (*http.Response, error)) (*Client, *stubTransport) {
	st := &stubTransport{handler: handler}
	c := New("https://example.atlassian.net/", "me@example.com", "tok-123", WithTransport(st))
	return c, st
}
