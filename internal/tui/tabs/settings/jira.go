package settings

import (
	"fmt"
	"strings"

	"github.com/madicen/aitinerary/internal/config"
	"github.com/madicen/aitinerary/internal/tui/styles"
)

func renderJira(cfg *config.Config) string {
	r := cfg.Redacted()
	var b strings.Builder
	b.WriteString(styles.DimStyle.Render("Jira connection (secrets redacted)"))
	b.WriteString("\n\n")
	b.WriteString(fmt.Sprintf("URL: %s\n", r.JiraURL))
	b.WriteString(fmt.Sprintf("User: %s\n", r.JiraUser))
	b.WriteString(fmt.Sprintf("Token env: %s\n", orDefault(r.JiraTokenEnv, "JIRA_TOKEN")))
	if r.JiraTokenCmd != "" {
		b.WriteString(fmt.Sprintf("Token cmd: %s\n", r.JiraTokenCmd))
	}
	b.WriteString("\nWorkspaces:\n")
	for _, ws := range r.Workspaces {
		labels := ""
		if len(ws.Labels) > 0 {
			labels = " labels=" + strings.Join(ws.Labels, ",")
		}
		b.WriteString(fmt.Sprintf("  • %s (%s%s)\n", ws.Name, ws.Project, labels))
	}
	b.WriteString("\n")
	b.WriteString(styles.DimStyle.Render("Config: " + config.DefaultPath()))
	return b.String()
}

func orDefault(s, def string) string {
	if strings.TrimSpace(s) == "" {
		return def
	}
	return s
}
