package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/madicen/aitinerary/internal/appdirs"
)

// FileName is the config file inside the app config dir.
const FileName = "config.json"

// DefaultPath returns the resolved config.json path under appdirs.ConfigDir().
func DefaultPath() string { return appdirs.ConfigFile(FileName) }

// Load reads config.json (from path, or DefaultPath when empty), merging it
// over Default() so a partial file still produces a complete Config. A missing
// file is not an error — the defaults are returned so first-run works. After
// the file merge, a small set of environment overrides is applied.
func Load(path string) (*Config, error) {
	if path == "" {
		path = DefaultPath()
	}
	c := Default()

	b, err := os.ReadFile(path)
	switch {
	case err == nil:
		if uerr := mergeJSON(c, b); uerr != nil {
			return nil, fmt.Errorf("parse %s: %w", path, uerr)
		}
	case errors.Is(err, os.ErrNotExist):
		// First run: defaults only.
	default:
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	mergeEnv(c)
	c.normalize()
	return c, nil
}

// mergeJSON decodes b onto c. Because c already holds the defaults, absent JSON
// keys leave the default in place; present keys (including explicit zero
// values) overwrite. Slices/maps that are present in JSON fully replace the
// default rather than appending.
func mergeJSON(c *Config, b []byte) error {
	// Reject an accidental token field before it can be silently ignored — a
	// loud error is better than a config that looks like it stored a secret.
	var probe map[string]json.RawMessage
	if err := json.Unmarshal(b, &probe); err != nil {
		return err
	}
	for _, k := range []string{"jira_token", "token", "password"} {
		if _, bad := probe[k]; bad {
			return fmt.Errorf("config contains forbidden secret field %q — use jira_token_env or jira_token_cmd instead", k)
		}
	}
	return json.Unmarshal(b, c)
}

// mergeEnv applies AITINERARY_* environment overrides on top of the file. Only
// a handful of operational knobs are overridable this way; secrets never are.
func mergeEnv(c *Config) {
	if v := strings.TrimSpace(os.Getenv("AITINERARY_JIRA_URL")); v != "" {
		c.JiraURL = v
	}
	if v := strings.TrimSpace(os.Getenv("AITINERARY_JIRA_USER")); v != "" {
		c.JiraUser = v
	}
	if v := strings.TrimSpace(os.Getenv("AITINERARY_JIRA_TOKEN_ENV")); v != "" {
		c.JiraTokenEnv = v
	}
	if v := strings.TrimSpace(os.Getenv("AITINERARY_STALE_DAYS")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			c.StaleInProgressDays = n
		}
	}
}

// normalize tidies whitespace and enforces sane fallbacks. It never fails —
// validation that can fail lives in Validate.
func (c *Config) normalize() {
	c.JiraURL = strings.TrimRight(strings.TrimSpace(c.JiraURL), "/")
	c.JiraUser = strings.TrimSpace(c.JiraUser)
	c.JiraTokenEnv = strings.TrimSpace(c.JiraTokenEnv)
	if c.JiraTokenEnv == "" && c.JiraTokenCmd == "" {
		c.JiraTokenEnv = "JIRA_TOKEN"
	}
	if strings.TrimSpace(c.DoneStatusName) == "" {
		c.DoneStatusName = "Done"
	}
	if c.StaleInProgressDays <= 0 {
		c.StaleInProgressDays = 10
	}
	if c.BoardIDs == nil {
		c.BoardIDs = map[string]int{}
	}
	if c.GitHubTokenSource == "" {
		c.GitHubTokenSource = "gh_cli"
	}
}

// Validate reports configuration problems that should block startup. It is
// separate from normalize so callers can display all issues at once.
func (c *Config) Validate() error {
	var errs []string
	if c.JiraURL == "" {
		errs = append(errs, "jira_url is empty")
	}
	if c.JiraUser == "" {
		errs = append(errs, "jira_user is empty")
	}
	if len(c.Workspaces) == 0 {
		errs = append(errs, "no workspaces configured")
	}
	for i, w := range c.Workspaces {
		if strings.TrimSpace(w.Project) == "" {
			errs = append(errs, fmt.Sprintf("workspace[%d] (%q) has no project key", i, w.Name))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("invalid config: %s", strings.Join(errs, "; "))
	}
	return nil
}

// Save writes c to path (or DefaultPath when empty) 0600 via a temp file and
// atomic rename. The Jira token is not a field on Config, so it can never be
// serialized here.
func Save(c *Config, path string) error {
	if c == nil {
		return errors.New("nil config")
	}
	if path == "" {
		path = DefaultPath()
	}
	c.normalize()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
