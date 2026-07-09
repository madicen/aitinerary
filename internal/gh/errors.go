package gh

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// APIError is a parsed view of a non-2xx GitHub API response from go-gh.
type APIError struct {
	Status      int
	Message     string
	Errors      []APIErrorItem
	DocURL      string
	APIPath     string
	RawBody     string
	HumanReason string
	RetryAfter  time.Duration
}

// APIErrorItem is one entry from a 4xx response's "errors" array.
type APIErrorItem struct {
	Resource string `json:"resource"`
	Code     string `json:"code"`
	Field    string `json:"field"`
	Message  string `json:"message"`
}

// Error returns a human-friendly description.
func (e *APIError) Error() string {
	if e == nil {
		return ""
	}
	var b strings.Builder
	switch {
	case e.Status > 0 && e.Message != "":
		fmt.Fprintf(&b, "GitHub %d: %s", e.Status, e.Message)
	case e.Status > 0:
		fmt.Fprintf(&b, "GitHub %d", e.Status)
	case e.Message != "":
		b.WriteString(e.Message)
	default:
		b.WriteString("GitHub API error")
	}
	for _, item := range e.Errors {
		switch {
		case item.Field != "" && item.Message != "":
			fmt.Fprintf(&b, "\n  · %s: %s", item.Field, item.Message)
		case item.Message != "":
			fmt.Fprintf(&b, "\n  · %s", item.Message)
		}
	}
	if e.HumanReason != "" {
		b.WriteString("\n→ " + e.HumanReason)
	}
	return b.String()
}

// IsUnauthorized reports whether err is an auth/permission failure.
func IsUnauthorized(err error) bool {
	var ae *APIError
	if errors.As(err, &ae) && ae != nil {
		return ae.Status == 401 || ae.Status == 403
	}
	return false
}

func inferHumanReason(e *APIError) string {
	if e == nil {
		return ""
	}
	if e.Status == 401 || e.Status == 403 {
		return "GitHub rejected the request as unauthorized — re-run `gh auth login` or check repo permissions."
	}
	if e.Status == 404 {
		return "GitHub couldn't find that resource. The PR may have been closed or your gh user lacks access."
	}
	return ""
}
