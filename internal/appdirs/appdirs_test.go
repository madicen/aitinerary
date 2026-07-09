package appdirs

import (
	"path/filepath"
	"testing"
)

func TestConfigDirPrecedence(t *testing.T) {
	t.Run("AITINERARY_CONFIG_DIR wins verbatim", func(t *testing.T) {
		t.Setenv("AITINERARY_CONFIG_DIR", "/custom/cfg")
		t.Setenv("XDG_CONFIG_HOME", "/xdg")
		if got := ConfigDir(); got != "/custom/cfg" {
			t.Fatalf("ConfigDir() = %q, want /custom/cfg", got)
		}
	})

	t.Run("falls back to XDG_CONFIG_HOME", func(t *testing.T) {
		t.Setenv("AITINERARY_CONFIG_DIR", "")
		t.Setenv("XDG_CONFIG_HOME", "/xdg")
		want := filepath.Join("/xdg", appName)
		if got := ConfigDir(); got != want {
			t.Fatalf("ConfigDir() = %q, want %q", got, want)
		}
	})
}

func TestCacheSubdir(t *testing.T) {
	t.Run("override roots the cache", func(t *testing.T) {
		t.Setenv("AITINERARY_CACHE_DIR", "/demo/cache")
		want := filepath.Join("/demo/cache", "store")
		if got := CacheSubdir("store"); got != want {
			t.Fatalf("CacheSubdir = %q, want %q", got, want)
		}
	})

	t.Run("falls back to XDG_CACHE_HOME", func(t *testing.T) {
		t.Setenv("AITINERARY_CACHE_DIR", "")
		t.Setenv("XDG_CACHE_HOME", "/xdgcache")
		want := filepath.Join("/xdgcache", appName, "store")
		if got := CacheSubdir("store"); got != want {
			t.Fatalf("CacheSubdir = %q, want %q", got, want)
		}
	})
}

func TestConfigFile(t *testing.T) {
	t.Setenv("AITINERARY_CONFIG_DIR", "/c")
	if got := ConfigFile("config.json"); got != filepath.Join("/c", "config.json") {
		t.Fatalf("ConfigFile = %q", got)
	}
}
