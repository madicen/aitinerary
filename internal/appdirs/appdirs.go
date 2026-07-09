// Package appdirs is the single source of truth for aitinerary's on-disk
// config and cache locations.
//
// It collapses XDG-style resolution into one implementation so relocating the
// tree (via AITINERARY_CONFIG_DIR / AITINERARY_CACHE_DIR / the XDG_* vars)
// behaves identically everywhere — the demo mode and tests rely on the
// AITINERARY_CONFIG_DIR override for isolation.
//
// It imports only the standard library so any package — including other leaf
// packages such as config and aiconfig — can depend on it without risking an
// import cycle.
package appdirs

import (
	"os"
	"path/filepath"
)

// appName is the per-user subdirectory the app owns under the XDG roots.
const appName = "aitinerary"

// ConfigDir returns the app's configuration directory. Precedence:
//
//  1. AITINERARY_CONFIG_DIR (used verbatim, enabling test/demo isolation)
//  2. XDG_CONFIG_HOME/aitinerary
//  3. ~/.config/aitinerary
//  4. ./.aitinerary (last resort when the home dir is unknowable)
func ConfigDir() string {
	if v := os.Getenv("AITINERARY_CONFIG_DIR"); v != "" {
		return v
	}
	if v := os.Getenv("XDG_CONFIG_HOME"); v != "" {
		return filepath.Join(v, appName)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "." + appName
	}
	return filepath.Join(home, ".config", appName)
}

// CacheSubdir returns the app cache directory named sub. Precedence:
//
//  1. AITINERARY_CACHE_DIR/<sub>
//  2. XDG_CACHE_HOME/aitinerary/<sub>
//  3. ~/.cache/aitinerary/<sub>
//  4. ./.cache/aitinerary/<sub> (last resort)
func CacheSubdir(sub string) string {
	if v := os.Getenv("AITINERARY_CACHE_DIR"); v != "" {
		return filepath.Join(v, sub)
	}
	if v := os.Getenv("XDG_CACHE_HOME"); v != "" {
		return filepath.Join(v, appName, sub)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".cache", appName, sub)
	}
	return filepath.Join(home, ".cache", appName, sub)
}

// ConfigFile joins name onto ConfigDir (e.g. "config.json", "ai.json").
func ConfigFile(name string) string {
	return filepath.Join(ConfigDir(), name)
}
