// Package config owns aitinerary's application configuration (Jira, GitHub,
// and behaviour knobs) as described in PLAN.md §5.
//
// Two hard rules, both lessons carried over from jj-tui and appr-ai-sal:
//
//   - The Jira token is NEVER stored in this config. It is resolved at runtime
//     from an environment variable (jira_token_env, default JIRA_TOKEN) or a
//     shell command (jira_token_cmd). There is deliberately no token field on
//     the struct, so it cannot be serialized by accident.
//   - AI provider profiles live separately in ai.json (package aiconfig), using
//     appr-ai-sal's api_key_env / api_key_cmd indirection.
//
// The package is split across small files (schema.go, load.go, secrets.go)
// rather than one god file.
package config

// Workspace is one logical bucket of tickets. Day-to-day work lives in project
// APP; IRE items live in project HELP filtered by label "Apps". Both are
// first-class and configurable — nothing is hardcoded to these values.
type Workspace struct {
	// Name is the display label (e.g. "Applications", "IRE").
	Name string `json:"name"`
	// Project is the Jira project key (e.g. "APP", "HELP").
	Project string `json:"project"`
	// Labels, when non-empty, restrict the workspace to tickets carrying ALL
	// of these labels (e.g. ["Apps"] for the IRE workspace).
	Labels []string `json:"labels,omitempty"`
	// JQLExtra is appended (AND-ed) to the generated JQL for advanced filters.
	JQLExtra string `json:"jql_extra,omitempty"`
}

// CustomFields caches Jira custom-field IDs resolved at runtime by field
// discovery (P1.2). Never hardcode IDs like customfield_10020 — they differ
// per instance. Keys are the logical names below.
type CustomFields struct {
	// Sprint is the Sprint field ID (e.g. "customfield_10020").
	Sprint string `json:"sprint,omitempty"`
	// StartDate is the Start date field ID used for timeline placement.
	StartDate string `json:"start_date,omitempty"`
	// EpicName is the Epic Name field ID (only required on some instances).
	EpicName string `json:"epic_name,omitempty"`
	// EpicLink is the legacy Epic Link field ID, used only as a fallback when
	// the modern `parent` field is rejected by the instance (P1.4).
	EpicLink string `json:"epic_link,omitempty"`
}

// Config is the on-disk application configuration (config.json). It is written
// 0600 with an atomic replace and is git-ignored when kept repo-local.
type Config struct {
	// --- Jira ---
	JiraURL      string `json:"jira_url"`
	JiraUser     string `json:"jira_user"`
	JiraTokenEnv string `json:"jira_token_env"`
	JiraTokenCmd string `json:"jira_token_cmd"`

	// --- Workspaces & boards ---
	Workspaces []Workspace `json:"workspaces"`
	// BoardIDs maps a project key to its agile board id. 0 means "auto-discover
	// and persist" (P1.5).
	BoardIDs map[string]int `json:"board_ids"`

	// --- Behaviour ---
	ExcludedStatuses    []string `json:"excluded_statuses"`
	DoneStatusName      string   `json:"done_status_name"`
	AutoDoneOnMerge     bool     `json:"auto_done_on_merge"`
	StaleInProgressDays int      `json:"stale_in_progress_days"`

	// --- GitHub ---
	// GitHubTokenSource selects how the gh layer authenticates ("gh_cli").
	GitHubTokenSource string `json:"github_token_source"`
	// GitHubRepos is an explicit repo list ("owner/name"); empty = discover.
	GitHubRepos []string `json:"github_repos"`

	// --- Theme (3-color) ---
	ThemePrimary   string `json:"theme_primary"`
	ThemeSecondary string `json:"theme_secondary"`
	ThemeMuted     string `json:"theme_muted"`

	// --- Discovered field IDs (cached; never hardcoded) ---
	CustomFields CustomFields `json:"custom_fields,omitempty"`
}

// Default returns a Config pre-populated with the PLAN.md §5 defaults. It is
// the base every load merges onto so a partial config.json still yields a
// complete, usable Config.
func Default() *Config {
	return &Config{
		JiraURL:      "https://stackadapt.atlassian.net",
		JiraUser:     "michael.madicen@stackadapt.com",
		JiraTokenEnv: "JIRA_TOKEN",
		JiraTokenCmd: "",
		Workspaces: []Workspace{
			{Name: "Applications", Project: "APP"},
			{Name: "IRE", Project: "HELP", Labels: []string{"Apps"}},
		},
		BoardIDs:            map[string]int{"APP": 0, "HELP": 0},
		ExcludedStatuses:    []string{"Won't Fix", "Will Not Do"},
		DoneStatusName:      "Done",
		AutoDoneOnMerge:     true,
		StaleInProgressDays: 10,
		GitHubTokenSource:   "gh_cli",
		GitHubRepos:         []string{},
		ThemePrimary:        "#7E00AF",
		ThemeSecondary:      "#50FA7B",
		ThemeMuted:          "#6272A4",
	}
}

// ProjectKeys returns the distinct Jira project keys across all workspaces, in
// declaration order. Used by the ticket-key matcher (P4.2) to build the key
// regex and by board discovery (P1.5).
func (c *Config) ProjectKeys() []string {
	seen := map[string]bool{}
	var out []string
	for _, w := range c.Workspaces {
		if w.Project == "" || seen[w.Project] {
			continue
		}
		seen[w.Project] = true
		out = append(out, w.Project)
	}
	return out
}
