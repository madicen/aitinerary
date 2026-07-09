package config

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// ErrNoToken is returned by ResolveToken when neither the env var nor the
// command produced a token. Callers surface the accompanying setup hint.
type ErrNoToken struct {
	EnvVar string
	Cmd    string
}

func (e *ErrNoToken) Error() string {
	switch {
	case e.EnvVar != "" && e.Cmd != "":
		return fmt.Sprintf("no Jira token: set $%s or make jira_token_cmd (%q) succeed", e.EnvVar, e.Cmd)
	case e.EnvVar != "":
		return fmt.Sprintf("no Jira token: export $%s with a Jira API token (https://id.atlassian.com/manage-profile/security/api-tokens)", e.EnvVar)
	default:
		return "no Jira token: configure jira_token_env (e.g. JIRA_TOKEN) or jira_token_cmd"
	}
}

// ResolveToken returns the Jira API token, never storing it in Config.
// Precedence per PLAN.md §5:
//
//  1. the environment variable named by jira_token_env (default JIRA_TOKEN)
//  2. the stdout of jira_token_cmd
//  3. *ErrNoToken with a setup hint
func (c *Config) ResolveToken() (string, error) {
	if env := strings.TrimSpace(c.JiraTokenEnv); env != "" {
		if v := strings.TrimSpace(os.Getenv(env)); v != "" {
			return v, nil
		}
	}
	if cmd := strings.TrimSpace(c.JiraTokenCmd); cmd != "" {
		if v := resolveTokenCmd(cmd); v != "" {
			return v, nil
		}
	}
	return "", &ErrNoToken{EnvVar: c.JiraTokenEnv, Cmd: c.JiraTokenCmd}
}

// HasTokenSource reports whether a token could plausibly be resolved without
// actually running the command (used by the setup wizard and status bar). It
// checks that the env var is currently set, or that a command is configured.
func (c *Config) HasTokenSource() bool {
	if env := strings.TrimSpace(c.JiraTokenEnv); env != "" {
		if strings.TrimSpace(os.Getenv(env)) != "" {
			return true
		}
	}
	return strings.TrimSpace(c.JiraTokenCmd) != ""
}

var (
	tokenCmdMu    sync.Mutex
	tokenCmdCache = map[string]string{}
)

// resolveTokenCmd runs cmd via the shell and returns its trimmed stdout.
// Failures yield "" (never fatal) so ResolveToken surfaces a clear ErrNoToken
// rather than a panic. Results are memoised for the process lifetime so a
// possibly-slow command (e.g. `op read`, `pass show`) runs at most once.
func resolveTokenCmd(cmd string) string {
	tokenCmdMu.Lock()
	if v, ok := tokenCmdCache[cmd]; ok {
		tokenCmdMu.Unlock()
		return v
	}
	tokenCmdMu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "sh", "-c", cmd).Output()
	if err != nil {
		return ""
	}
	v := strings.TrimSpace(string(out))
	if v == "" {
		return ""
	}
	tokenCmdMu.Lock()
	tokenCmdCache[cmd] = v
	tokenCmdMu.Unlock()
	return v
}

// Redacted returns a display-safe deep copy of c. Config holds no secret
// fields (the token is resolved at runtime, never stored), so this is a defence
// in depth: callers should render Redacted() rather than the live struct so
// that any future secret-ish field can be masked in exactly one place.
func (c *Config) Redacted() *Config {
	if c == nil {
		return nil
	}
	cp := *c
	// Deep-copy the mutable containers so the display copy can't mutate live
	// config, and vice-versa.
	cp.Workspaces = append([]Workspace(nil), c.Workspaces...)
	for i := range cp.Workspaces {
		cp.Workspaces[i].Labels = append([]string(nil), c.Workspaces[i].Labels...)
	}
	cp.BoardIDs = map[string]int{}
	for k, v := range c.BoardIDs {
		cp.BoardIDs[k] = v
	}
	cp.ExcludedStatuses = append([]string(nil), c.ExcludedStatuses...)
	cp.GitHubRepos = append([]string(nil), c.GitHubRepos...)
	return &cp
}
