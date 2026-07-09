package jira

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// APIError is a parsed view of a non-2xx response from the Jira REST API. It
// mirrors the taxonomy appr-ai-sal uses for GitHub: a real status code, a
// top-level message, per-field errors, the path we called, the raw body, and
// any Retry-After hint so callers can honour Jira's 429 backoff instead of
// guessing.
type APIError struct {
	Status     int               // HTTP status (0 if the request never completed).
	Message    string            // Top-level "message" / first errorMessages entry.
	Errors     map[string]string // Per-field errors from the "errors" object.
	APIPath    string            // The path that was called, for diagnostics.
	RawBody    string            // Original response body (always populated when available).
	RetryAfter time.Duration     // Parsed Retry-After; zero when absent/unparseable.
}

// Error returns a human-friendly, single-or-multi-line description.
func (e *APIError) Error() string {
	if e == nil {
		return ""
	}
	var b strings.Builder
	switch {
	case e.Status > 0 && e.Message != "":
		fmt.Fprintf(&b, "Jira %d: %s", e.Status, e.Message)
	case e.Status > 0:
		fmt.Fprintf(&b, "Jira %d", e.Status)
	case e.Message != "":
		b.WriteString(e.Message)
	default:
		b.WriteString("Jira API error")
	}
	// Deterministic ordering of field errors for stable output/tests.
	if len(e.Errors) > 0 {
		keys := make([]string, 0, len(e.Errors))
		for k := range e.Errors {
			keys = append(keys, k)
		}
		sortStrings(keys)
		for _, k := range keys {
			fmt.Fprintf(&b, "\n  · %s: %s", k, e.Errors[k])
		}
	}
	if e.RetryAfter > 0 {
		fmt.Fprintf(&b, "\n→ retry after %s", e.RetryAfter)
	}
	return b.String()
}

// sortStrings is a tiny insertion sort to avoid importing sort in a hot path
// that only ever handles a handful of field-error keys.
func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j-1] > s[j]; j-- {
			s[j-1], s[j] = s[j], s[j-1]
		}
	}
}

// apiErrorFrom builds an *APIError from a completed HTTP response's status,
// body, path, and headers. Jira error bodies come in two shapes:
//
//	{"errorMessages":["..."],"errors":{"field":"reason"}}
//
// and occasionally a bare {"message":"..."}. We parse both.
func apiErrorFrom(status int, body []byte, path string, header http.Header) *APIError {
	ae := &APIError{Status: status, APIPath: path, RawBody: strings.TrimSpace(string(body))}
	var parsed struct {
		ErrorMessages []string          `json:"errorMessages"`
		Errors        map[string]string `json:"errors"`
		Message       string            `json:"message"`
	}
	if err := json.Unmarshal(body, &parsed); err == nil {
		switch {
		case len(parsed.ErrorMessages) > 0:
			ae.Message = parsed.ErrorMessages[0]
		case parsed.Message != "":
			ae.Message = parsed.Message
		}
		if len(parsed.Errors) > 0 {
			ae.Errors = parsed.Errors
		}
	}
	if ae.Message == "" {
		ae.Message = firstLine(ae.RawBody)
	}
	if header != nil {
		ae.RetryAfter = retryAfterFrom(header)
	}
	return ae
}

// retryAfterFrom reads the Retry-After header, which may be delta-seconds
// (e.g. "120") or an HTTP date. Returns zero when absent or unparseable.
func retryAfterFrom(h http.Header) time.Duration {
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

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return strings.TrimSpace(s[:i])
	}
	return strings.TrimSpace(s)
}

// StatusOf returns the HTTP status carried by err if it is an *APIError, else 0.
func StatusOf(err error) int {
	var ae *APIError
	if errors.As(err, &ae) && ae != nil {
		return ae.Status
	}
	return 0
}

// IsUnauthorized reports whether err is a 401/403 from Jira.
func IsUnauthorized(err error) bool {
	s := StatusOf(err)
	return s == http.StatusUnauthorized || s == http.StatusForbidden
}

// IsNotFound reports whether err is a 404 from Jira.
func IsNotFound(err error) bool { return StatusOf(err) == http.StatusNotFound }

// IsRateLimited reports whether err is a 429 from Jira; the *APIError's
// RetryAfter carries the suggested backoff.
func IsRateLimited(err error) bool { return StatusOf(err) == http.StatusTooManyRequests }

// IsBadRequest reports whether err is a 400 — used by SetParent to detect a
// `parent` field rejection and fall back to the Epic Link custom field.
func IsBadRequest(err error) bool { return StatusOf(err) == http.StatusBadRequest }

// MentionsField reports whether a 400 APIError names the given field (case
// insensitively) in either its per-field errors or message. SetParent uses
// this to decide whether a rejection was specifically about `parent`.
func MentionsField(err error, field string) bool {
	var ae *APIError
	if !errors.As(err, &ae) || ae == nil {
		return false
	}
	field = strings.ToLower(field)
	if _, ok := ae.Errors[field]; ok {
		return true
	}
	for k, v := range ae.Errors {
		if strings.Contains(strings.ToLower(k), field) || strings.Contains(strings.ToLower(v), field) {
			return true
		}
	}
	return strings.Contains(strings.ToLower(ae.Message), field)
}
