package jira

import (
	"context"
	"encoding/base64"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestBasicAuthHeader(t *testing.T) {
	c, st := newStubClient(func(_ *http.Request, _ []byte) (*http.Response, error) {
		return jsonResponse(200, `{"permissions":{"BROWSE_PROJECTS":{"havePermission":true}}}`, nil), nil
	})
	if err := c.CheckPermissions(context.Background()); err != nil {
		t.Fatalf("CheckPermissions: %v", err)
	}
	got := st.last().Auth
	want := "Basic " + base64.StdEncoding.EncodeToString([]byte("me@example.com:tok-123"))
	if got != want {
		t.Fatalf("auth header = %q, want %q", got, want)
	}
	if st.last().Path != "/rest/api/3/mypermissions" {
		t.Fatalf("path = %q", st.last().Path)
	}
	if !strings.Contains(st.last().Query, "BROWSE_PROJECTS") {
		t.Fatalf("query missing permission: %q", st.last().Query)
	}
}

func TestCheckPermissionsRejectsMissingPermission(t *testing.T) {
	c, _ := newStubClient(func(_ *http.Request, _ []byte) (*http.Response, error) {
		return jsonResponse(200, `{"permissions":{"BROWSE_PROJECTS":{"havePermission":false}}}`, nil), nil
	})
	err := c.CheckPermissions(context.Background())
	if err == nil || !strings.Contains(err.Error(), "Browse Projects") {
		t.Fatalf("expected browse-projects error, got %v", err)
	}
}

func TestErrorMappingUnauthorized(t *testing.T) {
	c, _ := newStubClient(func(_ *http.Request, _ []byte) (*http.Response, error) {
		return jsonResponse(401, `{"errorMessages":["Client must be authenticated"]}`, nil), nil
	})
	err := c.CheckPermissions(context.Background())
	if !IsUnauthorized(err) {
		t.Fatalf("expected unauthorized, got %v", err)
	}
	if !strings.Contains(err.Error(), "authentication failed") {
		t.Fatalf("expected friendly auth message, got %v", err)
	}
}

func TestErrorMappingFieldErrors(t *testing.T) {
	body := `{"errorMessages":[],"errors":{"duedate":"Invalid date format"}}`
	ae := apiErrorFrom(400, []byte(body), "/rest/api/3/issue/APP-1", http.Header{})
	if ae.Status != 400 {
		t.Fatalf("status = %d", ae.Status)
	}
	if !MentionsField(ae, "duedate") {
		t.Fatalf("MentionsField(duedate) should be true: %v", ae)
	}
	if !strings.Contains(ae.Error(), "Invalid date format") {
		t.Fatalf("error string missing field detail: %q", ae.Error())
	}
}

func TestRetryAfterParsing(t *testing.T) {
	t.Run("delta seconds", func(t *testing.T) {
		h := http.Header{}
		h.Set("Retry-After", "120")
		ae := apiErrorFrom(429, []byte(`{"errorMessages":["Too many"]}`), "/x", h)
		if ae.RetryAfter != 120*time.Second {
			t.Fatalf("RetryAfter = %v, want 120s", ae.RetryAfter)
		}
		if !IsRateLimited(ae) {
			t.Fatal("IsRateLimited should be true for 429")
		}
	})
	t.Run("absent header", func(t *testing.T) {
		ae := apiErrorFrom(429, []byte(`{}`), "/x", http.Header{})
		if ae.RetryAfter != 0 {
			t.Fatalf("RetryAfter = %v, want 0", ae.RetryAfter)
		}
	})
	t.Run("http date", func(t *testing.T) {
		h := http.Header{}
		future := time.Now().Add(90 * time.Second).UTC().Format(http.TimeFormat)
		h.Set("Retry-After", future)
		ae := apiErrorFrom(429, []byte(`{}`), "/x", h)
		if ae.RetryAfter <= 0 || ae.RetryAfter > 91*time.Second {
			t.Fatalf("RetryAfter = %v, want ~90s", ae.RetryAfter)
		}
	})
}

func TestBrowseURL(t *testing.T) {
	c := New("https://x.atlassian.net/", "u", "t")
	if got := c.BrowseURL("APP-1"); got != "https://x.atlassian.net/browse/APP-1" {
		t.Fatalf("BrowseURL = %q", got)
	}
}
