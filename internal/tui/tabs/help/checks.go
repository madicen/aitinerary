package help

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/madicen/aitinerary/internal/aiconfig"
	"github.com/madicen/aitinerary/internal/ai"
	"github.com/madicen/aitinerary/internal/config"
	"github.com/madicen/aitinerary/internal/gh"
)

// CheckResult is one setup wizard step outcome.
type CheckResult struct {
	Title   string
	Status  string // "ok", "warn", "fail"
	Detail  string
	Hint    string
}

// Checker runs first-run setup checks. Fields are injectable for tests.
type Checker struct {
	Config   *config.Config
	AIConfig *aiconfig.Config
	Demo     bool

	CheckGH     func() error
	CheckOllama func(context.Context, *aiconfig.Config) error
}

// RunAll executes every wizard check and returns ordered results.
func (c *Checker) RunAll() []CheckResult {
	if c == nil {
		return nil
	}
	if c.Demo {
		return demoCheckResults()
	}
	cfg := c.Config
	if cfg == nil {
		cfg = config.Default()
	}
	aiCfg := c.AIConfig
	if aiCfg == nil {
		aiCfg = aiconfig.DefaultConfig()
	}

	var out []CheckResult

	// Jira URL
	if strings.TrimSpace(cfg.JiraURL) == "" {
		out = append(out, CheckResult{
			Title: "Jira URL", Status: "fail",
			Detail: "not configured",
			Hint:   "Set jira_url in config.json or AITINERARY_JIRA_URL",
		})
	} else {
		out = append(out, CheckResult{
			Title: "Jira URL", Status: "ok",
			Detail: cfg.JiraURL,
		})
	}

	// Jira user
	if strings.TrimSpace(cfg.JiraUser) == "" {
		out = append(out, CheckResult{
			Title: "Jira email", Status: "fail",
			Detail: "not configured",
			Hint:   "Set jira_user in config.json",
		})
	} else {
		out = append(out, CheckResult{
			Title: "Jira email", Status: "ok",
			Detail: cfg.JiraUser,
		})
	}

	// Token env
	envName := strings.TrimSpace(cfg.JiraTokenEnv)
	if envName == "" {
		envName = "JIRA_TOKEN"
	}
	if cfg.HasTokenSource() {
		out = append(out, CheckResult{
			Title: "Jira token", Status: "ok",
			Detail: fmt.Sprintf("$%s is set (or jira_token_cmd configured)", envName),
		})
	} else {
		out = append(out, CheckResult{
			Title: "Jira token", Status: "fail",
			Detail: fmt.Sprintf("$%s is empty", envName),
			Hint:   "export JIRA_TOKEN with an Atlassian API token",
		})
	}

	// Board discovery
	var boardHints []string
	allBoards := true
	for _, pk := range cfg.ProjectKeys() {
		id := cfg.BoardIDs[pk]
		if id == 0 {
			allBoards = false
			boardHints = append(boardHints, fmt.Sprintf("%s: auto-discover on first refresh (ctrl+r)", pk))
		} else {
			boardHints = append(boardHints, fmt.Sprintf("%s: board %d", pk, id))
		}
	}
	boardStatus := "ok"
	if !allBoards {
		boardStatus = "warn"
	}
	out = append(out, CheckResult{
		Title: "Board IDs", Status: boardStatus,
		Detail: strings.Join(boardHints, "; "),
		Hint:   "Board ids of 0 are discovered via GET /rest/agile/1.0/board and persisted",
	})

	// gh auth
	checkGH := c.CheckGH
	if checkGH == nil {
		checkGH = gh.CheckAuth
	}
	if err := checkGH(); err != nil {
		out = append(out, CheckResult{
			Title: "GitHub (gh)", Status: "warn",
			Detail: err.Error(),
			Hint:   "Install gh and run `gh auth login` for merged-PR detection",
		})
	} else {
		out = append(out, CheckResult{
			Title: "GitHub (gh)", Status: "ok",
			Detail: "authenticated",
		})
	}

	// Ollama ping
	checkOllama := c.CheckOllama
	if checkOllama == nil {
		checkOllama = func(ctx context.Context, cfg *aiconfig.Config) error {
			_, err := ai.ListModels(ctx, cfg)
			return err
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := checkOllama(ctx, aiCfg); err != nil {
		out = append(out, CheckResult{
			Title: "Ollama", Status: "warn",
			Detail: err.Error(),
			Hint:   "Start Ollama locally or switch AI profile in Settings → AI Profiles",
		})
	} else {
		out = append(out, CheckResult{
			Title: "Ollama", Status: "ok",
			Detail: fmt.Sprintf("%s reachable", aiCfg.AIBaseURLResolved()),
		})
	}

	return out
}

func demoCheckResults() []CheckResult {
	return []CheckResult{
		{Title: "Jira URL", Status: "ok", Detail: "https://stackadapt.atlassian.net (demo)"},
		{Title: "Jira email", Status: "ok", Detail: "demo@example.com"},
		{Title: "Jira token", Status: "ok", Detail: "demo mode — no live token needed"},
		{Title: "Board IDs", Status: "ok", Detail: "APP: 42; HELP: 43 (fixture)"},
		{Title: "GitHub (gh)", Status: "ok", Detail: "demo mode — fixture PRs loaded"},
		{Title: "Ollama", Status: "ok", Detail: "demo mode — stub AI responses"},
	}
}
