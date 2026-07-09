// Package data wires runtime services (Jira client, actions engine, journal)
// for the TUI root model.
package data

import (
	"path/filepath"

	"github.com/madicen/aitinerary/internal/actions"
	"github.com/madicen/aitinerary/internal/appdirs"
	"github.com/madicen/aitinerary/internal/config"
	"github.com/madicen/aitinerary/internal/jira"
)

// Services holds live integrations used by the TUI.
type Services struct {
	Jira    *jira.Client
	Engine  *actions.Engine
	Journal *actions.Journal
}

// NewServices builds integrations from config. In demo mode Jira is nil and the
// engine uses a nil executor (queue/apply are no-ops against live Jira).
func NewServices(cfg *config.Config, demo bool) (*Services, error) {
	if cfg == nil {
		cfg = config.Default()
	}
	var jc *jira.Client
	if !demo {
		token, err := cfg.ResolveToken()
		if err != nil {
			return nil, err
		}
		jc = jira.New(cfg.JiraURL, cfg.JiraUser, token)
		if cfg.CustomFields.Sprint != "" || cfg.CustomFields.StartDate != "" {
			jc.SetFieldIDs(jira.FieldIDs{
				Sprint:    cfg.CustomFields.Sprint,
				StartDate: cfg.CustomFields.StartDate,
				EpicName:  cfg.CustomFields.EpicName,
				EpicLink:  cfg.CustomFields.EpicLink,
			})
		}
	}
	journalPath := filepath.Join(appdirs.ConfigDir(), "journal.jsonl")
	journal := actions.NewJournal(journalPath)
	var exec actions.Executor
	if jc != nil {
		exec = actions.NewJiraExecutor(jc)
	}
	return &Services{
		Jira:    jc,
		Engine:  actions.NewEngine(exec, journal),
		Journal: journal,
	}, nil
}
