// Package theme owns the user-configurable 3-colour palette (primary /
// secondary / muted) and light/dark/NO_COLOR appearance for the TUI.
package theme

import (
	"sync"

	"github.com/madicen/aitinerary/internal/config"
)

// Theme is the on-disk / in-memory colour configuration.
type Theme struct {
	Primary   string `json:"primary,omitempty"`
	Secondary string `json:"secondary,omitempty"`
	Muted     string `json:"muted,omitempty"`
	Mode      string `json:"mode,omitempty"`
}

// Default returns the PLAN.md §5 default colours.
func Default() *Theme {
	c := config.Default()
	return &Theme{
		Primary:   c.ThemePrimary,
		Secondary: c.ThemeSecondary,
		Muted:     c.ThemeMuted,
	}
}

// FromConfig builds a theme from application config colours and mode.
func FromConfig(c *config.Config) *Theme {
	if c == nil {
		return Default()
	}
	t := Default()
	if c.ThemePrimary != "" {
		t.Primary = c.ThemePrimary
	}
	if c.ThemeSecondary != "" {
		t.Secondary = c.ThemeSecondary
	}
	if c.ThemeMuted != "" {
		t.Muted = c.ThemeMuted
	}
	return t
}

// Clone returns a deep copy.
func (t *Theme) Clone() *Theme {
	if t == nil {
		return Default()
	}
	out := *t
	return &out
}

// AppearanceMode returns the parsed appearance mode, defaulting to dark.
func (t *Theme) AppearanceMode() Mode {
	if t == nil {
		return ModeDark
	}
	return ParseMode(t.Mode)
}

// Color resolves one of the three user slots (primary|secondary|muted).
func (t *Theme) Color(slot string) string {
	if t == nil {
		t = Default()
	}
	switch slot {
	case "primary":
		if t.Primary != "" {
			return normalizeHex(t.Primary)
		}
	case "secondary":
		if t.Secondary != "" {
			return normalizeHex(t.Secondary)
		}
	case "muted":
		if t.Muted != "" {
			return normalizeHex(t.Muted)
		}
	}
	d := Default()
	switch slot {
	case "primary":
		return d.Primary
	case "secondary":
		return d.Secondary
	case "muted":
		return d.Muted
	}
	return ""
}

// Set stores a validated hex against primary|secondary|muted.
func (t *Theme) Set(slot, hex string) {
	if !validHex(hex) {
		return
	}
	h := normalizeHex(hex)
	switch slot {
	case "primary":
		t.Primary = h
	case "secondary":
		t.Secondary = h
	case "muted":
		t.Muted = h
	}
}

// Reset removes an override so defaults apply.
func (t *Theme) Reset(slot string) {
	switch slot {
	case "primary":
		t.Primary = ""
	case "secondary":
		t.Secondary = ""
	case "muted":
		t.Muted = ""
	}
}

var (
	currentMu sync.RWMutex
	current   = Default()
)

// Current returns a defensive copy of the active theme.
func Current() *Theme {
	currentMu.RLock()
	defer currentMu.RUnlock()
	return current.Clone()
}

// Apply replaces the active theme.
func Apply(t *Theme) {
	currentMu.Lock()
	defer currentMu.Unlock()
	if t == nil {
		current = Default()
		return
	}
	current = t.Clone()
}

func validHex(s string) bool {
	if len(s) != 4 && len(s) != 7 {
		return false
	}
	if s[0] != '#' {
		return false
	}
	for i := 1; i < len(s); i++ {
		c := s[i]
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

func normalizeHex(s string) string {
	if len(s) == 4 {
		s = "#" + string([]byte{s[1], s[1], s[2], s[2], s[3], s[3]})
	}
	out := []byte(s)
	for i := 1; i < len(out); i++ {
		c := out[i]
		if c >= 'A' && c <= 'F' {
			out[i] = c + ('a' - 'A')
		}
	}
	return string(out)
}
