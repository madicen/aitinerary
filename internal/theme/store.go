package theme

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/madicen/aitinerary/internal/appdirs"
)

// DefaultPath is ~/.config/aitinerary/theme.json.
func DefaultPath() string {
	return filepath.Join(appdirs.ConfigDir(), "theme.json")
}

// Load reads theme.json when present and merges onto defaults.
func Load() (*Theme, error) {
	path := DefaultPath()
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Default(), nil
		}
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var t Theme
	if err := json.Unmarshal(b, &t); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	out := Default()
	if validHex(t.Primary) {
		out.Primary = normalizeHex(t.Primary)
	}
	if validHex(t.Secondary) {
		out.Secondary = normalizeHex(t.Secondary)
	}
	if validHex(t.Muted) {
		out.Muted = normalizeHex(t.Muted)
	}
	out.Mode = t.Mode
	return out, nil
}

// Save writes t to path (DefaultPath when empty). Only non-default values persist.
func Save(t *Theme, path string) error {
	if t == nil {
		return fmt.Errorf("nil theme")
	}
	if path == "" {
		path = DefaultPath()
	}
	def := Default()
	persist := Theme{}
	if validHex(t.Primary) && normalizeHex(t.Primary) != def.Primary {
		persist.Primary = normalizeHex(t.Primary)
	}
	if validHex(t.Secondary) && normalizeHex(t.Secondary) != def.Secondary {
		persist.Secondary = normalizeHex(t.Secondary)
	}
	if validHex(t.Muted) && normalizeHex(t.Muted) != def.Muted {
		persist.Muted = normalizeHex(t.Muted)
	}
	if m := strings.ToLower(strings.TrimSpace(t.Mode)); m != "" && ParseMode(m) != ModeDark {
		persist.Mode = ParseMode(m).String()
	}
	b, err := json.MarshalIndent(persist, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal theme: %w", err)
	}
	b = append(b, '\n')
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", dir, err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
