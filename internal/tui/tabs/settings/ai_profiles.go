package settings

import (
	"fmt"
	"strings"

	"github.com/madicen/aitinerary/internal/aiconfig"
	"github.com/madicen/aitinerary/internal/tui/styles"
)

func renderAIProfiles(cfg *aiconfig.Config) string {
	r := cfg.Redacted()
	var b strings.Builder
	b.WriteString(styles.DimStyle.Render("AI profiles (keys redacted)"))
	b.WriteString("\n\n")
	active := r.ActiveProfile
	if active == "" {
		active = aiconfig.DefaultProfileName
	}
	b.WriteString(fmt.Sprintf("Active: %s\n\n", styles.BoldStyle.Render(active)))
	for i, p := range r.Profiles {
		marker := "  "
		if strings.EqualFold(p.Name, active) {
			marker = styles.OkStyle.Render("▶ ")
		}
		b.WriteString(fmt.Sprintf("%s%s — %s\n", marker, p.Name, p.Summary()))
		if i < len(r.Profiles)-1 {
			b.WriteString("\n")
		}
	}
	b.WriteString("\n")
	b.WriteString(styles.DimStyle.Render("Config: " + aiconfig.DefaultPath()))
	return b.String()
}
