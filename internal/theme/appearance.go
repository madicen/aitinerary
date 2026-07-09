package theme

import (
	"os"
	"strings"
	"sync"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// Mode selects how light/dark presets resolve.
type Mode int

const (
	ModeDark Mode = iota
	ModeLight
	ModeAuto
)

func (m Mode) String() string {
	switch m {
	case ModeLight:
		return "light"
	case ModeAuto:
		return "auto"
	default:
		return "dark"
	}
}

// ParseMode maps a persisted / env string to a Mode.
func ParseMode(s string) Mode {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "light":
		return ModeLight
	case "auto", "adaptive", "system":
		return ModeAuto
	default:
		return ModeDark
	}
}

// Appearance is the resolved rendering decision.
type Appearance struct {
	Mode    Mode
	NoColor bool
}

var (
	appearanceMu     sync.RWMutex
	activeAppearance = Appearance{Mode: ModeDark}
)

func noColorEnv() bool {
	_, ok := os.LookupEnv("NO_COLOR")
	return ok
}

// DetectAppearance resolves appearance from env layered over saved mode.
// Precedence: NO_COLOR → AITINERARY_THEME → saved mode → dark default.
func DetectAppearance(saved Mode) Appearance {
	a := Appearance{Mode: saved}
	switch strings.ToLower(strings.TrimSpace(os.Getenv("AITINERARY_THEME"))) {
	case "dark":
		a.Mode = ModeDark
	case "light":
		a.Mode = ModeLight
	case "auto", "adaptive", "system":
		a.Mode = ModeAuto
	case "none", "mono", "no-color", "nocolor":
		a.NoColor = true
	}
	if noColorEnv() {
		a.NoColor = true
	}
	return a
}

// ApplyAppearance enforces appearance on the global lipgloss renderer.
func ApplyAppearance(a Appearance) {
	appearanceMu.Lock()
	activeAppearance = a
	appearanceMu.Unlock()

	if a.NoColor {
		lipgloss.SetColorProfile(termenv.Ascii)
		return
	}
	switch a.Mode {
	case ModeDark:
		lipgloss.SetHasDarkBackground(true)
	case ModeLight:
		lipgloss.SetHasDarkBackground(false)
	case ModeAuto:
		// Defer to lipgloss terminal detection.
	}
}

// SetupRendering detects appearance from env + theme mode and applies it.
func SetupRendering(t *Theme) Appearance {
	mode := ModeDark
	if t != nil {
		mode = t.AppearanceMode()
	}
	a := DetectAppearance(mode)
	ApplyAppearance(a)
	return a
}

// ActiveAppearance returns the most recently applied appearance.
func ActiveAppearance() Appearance {
	appearanceMu.RLock()
	defer appearanceMu.RUnlock()
	return activeAppearance
}

// NoColor reports whether colour output is disabled.
func NoColor() bool {
	return ActiveAppearance().NoColor
}
