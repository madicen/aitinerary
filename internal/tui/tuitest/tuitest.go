// Package tuitest holds shared helpers for hermetic TUI render tests.
package tuitest

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	zone "github.com/lrstanley/bubblezone"

	"github.com/madicen/aitinerary/internal/theme"
)

func init() {
	if flag.Lookup("update") == nil {
		flag.Bool("update", false, "rewrite golden files instead of comparing against them")
	}
}

// ShouldUpdate reports whether the -update flag was passed.
func ShouldUpdate() bool {
	f := flag.Lookup("update")
	if f == nil {
		return false
	}
	g, ok := f.Value.(flag.Getter)
	if !ok {
		return false
	}
	b, _ := g.Get().(bool)
	return b
}

// ForceMonochrome makes rendering deterministic for golden / teatest flows.
func ForceMonochrome(t testing.TB) {
	t.Helper()
	prev := lipgloss.ColorProfile()
	t.Cleanup(func() { lipgloss.SetColorProfile(prev) })
	t.Setenv("NO_COLOR", "1")
	zone.NewGlobal()
	theme.ApplyAppearance(theme.Appearance{Mode: theme.ModeDark, NoColor: true})
}

// ZoneCenter returns the center cell of a registered zone. bubblezone updates
// zone bounds asynchronously after Scan; this polls briefly so tests stay stable.
func ZoneCenter(zoneID string) (x, y int, ok bool) {
	deadline := time.Now().Add(200 * time.Millisecond)
	for time.Now().Before(deadline) {
		if z := zone.Get(zoneID); z != nil && !z.IsZero() {
			x = z.StartX + (z.EndX-z.StartX)/2
			y = z.StartY + (z.EndY-z.StartY)/2
			return x, y, true
		}
		time.Sleep(2 * time.Millisecond)
	}
	return 0, 0, false
}

// Normalize canonicalizes rendered TUI output for golden comparison.
func Normalize(s string) string {
	s = zone.Scan(s)
	s = ansi.Strip(s)
	lines := strings.Split(s, "\n")
	for i := range lines {
		lines[i] = strings.TrimRight(lines[i], " \t")
	}
	return strings.TrimRight(strings.Join(lines, "\n"), "\n") + "\n"
}

// AssertGolden compares got against testdata/<name>.golden after Normalize.
func AssertGolden(t testing.TB, name, got string) {
	t.Helper()
	got = Normalize(got)
	path := filepath.Join("testdata", name+".golden")
	if ShouldUpdate() {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatalf("tuitest: mkdir testdata: %v", err)
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatalf("tuitest: write golden %s: %v", path, err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("tuitest: read golden %s: %v (run with -update to create)", path, err)
	}
	if got != string(want) {
		t.Fatalf("golden mismatch for %s.\n--- want ---\n%s\n--- got ---\n%s", name, string(want), got)
	}
}
