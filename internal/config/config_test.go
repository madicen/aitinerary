package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultsAreComplete(t *testing.T) {
	c := Default()
	if err := c.Validate(); err != nil {
		t.Fatalf("Default() should validate, got %v", err)
	}
	if len(c.Workspaces) != 2 || c.Workspaces[0].Project != "APP" || c.Workspaces[1].Project != "HELP" {
		t.Fatalf("unexpected default workspaces: %+v", c.Workspaces)
	}
	if got := c.ProjectKeys(); len(got) != 2 || got[0] != "APP" || got[1] != "HELP" {
		t.Fatalf("ProjectKeys = %v", got)
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	in := Default()
	in.JiraURL = "https://example.atlassian.net"
	in.BoardIDs["APP"] = 42
	in.CustomFields.Sprint = "customfield_10020"

	if err := Save(in, path); err != nil {
		t.Fatalf("Save: %v", err)
	}
	// 0600 perms.
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if perm := fi.Mode().Perm(); perm != 0o600 {
		t.Fatalf("perm = %o, want 600", perm)
	}

	out, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if out.JiraURL != in.JiraURL {
		t.Fatalf("jira_url round-trip: got %q", out.JiraURL)
	}
	if out.BoardIDs["APP"] != 42 {
		t.Fatalf("board id round-trip: got %d", out.BoardIDs["APP"])
	}
	if out.CustomFields.Sprint != "customfield_10020" {
		t.Fatalf("custom field round-trip: got %q", out.CustomFields.Sprint)
	}
}

func TestLoadMissingFileReturnsDefaults(t *testing.T) {
	c, err := Load(filepath.Join(t.TempDir(), "nope.json"))
	if err != nil {
		t.Fatalf("missing file must not error: %v", err)
	}
	if c.DoneStatusName != "Done" {
		t.Fatalf("expected defaults, got %+v", c)
	}
}

func TestPartialFileMergesOverDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte(`{"jira_url":"https://x.atlassian.net","stale_in_progress_days":3}`), 0o600); err != nil {
		t.Fatal(err)
	}
	c, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if c.JiraURL != "https://x.atlassian.net" {
		t.Fatalf("override not applied: %q", c.JiraURL)
	}
	if c.StaleInProgressDays != 3 {
		t.Fatalf("stale days: %d", c.StaleInProgressDays)
	}
	// Untouched keys keep their defaults.
	if c.DoneStatusName != "Done" || len(c.Workspaces) != 2 {
		t.Fatalf("defaults not preserved: %+v", c)
	}
}

func TestTokenNeverSerialized(t *testing.T) {
	c := Default()
	b, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	for _, forbidden := range []string{"jira_token\"", "\"token\"", "password"} {
		if strings.Contains(s, forbidden) {
			t.Fatalf("serialized config contains forbidden field %q: %s", forbidden, s)
		}
	}
}

func TestLoadRejectsSecretField(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte(`{"jira_token":"sk-secret"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("Load should reject a config carrying a jira_token field")
	}
}

func TestResolveTokenPrecedence(t *testing.T) {
	t.Run("env wins", func(t *testing.T) {
		t.Setenv("MY_JIRA_TOKEN", "env-token")
		c := Default()
		c.JiraTokenEnv = "MY_JIRA_TOKEN"
		c.JiraTokenCmd = "echo cmd-token"
		got, err := c.ResolveToken()
		if err != nil {
			t.Fatal(err)
		}
		if got != "env-token" {
			t.Fatalf("got %q, want env-token", got)
		}
	})

	t.Run("falls back to cmd", func(t *testing.T) {
		c := Default()
		c.JiraTokenEnv = "DEFINITELY_UNSET_TOKEN_VAR"
		c.JiraTokenCmd = "echo cmd-token-42"
		got, err := c.ResolveToken()
		if err != nil {
			t.Fatal(err)
		}
		if got != "cmd-token-42" {
			t.Fatalf("got %q, want cmd-token-42", got)
		}
	})

	t.Run("errors with hint when nothing set", func(t *testing.T) {
		c := Default()
		c.JiraTokenEnv = "DEFINITELY_UNSET_TOKEN_VAR"
		c.JiraTokenCmd = ""
		_, err := c.ResolveToken()
		if err == nil {
			t.Fatal("expected ErrNoToken")
		}
		if !strings.Contains(err.Error(), "DEFINITELY_UNSET_TOKEN_VAR") {
			t.Fatalf("hint should name the env var, got %q", err.Error())
		}
	})
}

func TestRedactedIsDeepCopy(t *testing.T) {
	c := Default()
	r := c.Redacted()
	r.BoardIDs["APP"] = 999
	r.Workspaces[0].Project = "MUTATED"
	if c.BoardIDs["APP"] == 999 {
		t.Fatal("Redacted() must not share the BoardIDs map")
	}
	if c.Workspaces[0].Project == "MUTATED" {
		t.Fatal("Redacted() must not share the Workspaces slice")
	}
}
