package settings

import (
	"fmt"
	"strings"

	"github.com/madicen/aitinerary/internal/config"
	"github.com/madicen/aitinerary/internal/tui/styles"
)

func renderGeneral(cfg *config.Config, demo bool) string {
	var b strings.Builder
	b.WriteString(styles.DimStyle.Render("General application behaviour"))
	b.WriteString("\n\n")
	if demo {
		b.WriteString(styles.ErrStyle.Render("Demo mode") + " — isolated config directory\n\n")
	}
	b.WriteString(fmt.Sprintf("Auto-done on merge: %v\n", cfg.AutoDoneOnMerge))
	b.WriteString(fmt.Sprintf("Done status: %s\n", cfg.DoneStatusName))
	b.WriteString(fmt.Sprintf("Stale In Progress threshold: %d days\n", cfg.StaleInProgressDays))
	b.WriteString(fmt.Sprintf("GitHub token source: %s\n", cfg.GitHubTokenSource))
	if len(cfg.GitHubRepos) == 0 {
		b.WriteString("GitHub repos: (auto-discover)\n")
	} else {
		b.WriteString("GitHub repos: " + strings.Join(cfg.GitHubRepos, ", ") + "\n")
	}
	b.WriteString("\n")
	b.WriteString(styles.DimStyle.Render("[ ] subtabs · ctrl+s save theme · tab cycle swatches"))
	return b.String()
}
