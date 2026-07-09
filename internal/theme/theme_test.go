package theme

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

func TestParseModeDefaultsToDark(t *testing.T) {
	for in, want := range map[string]Mode{
		"": ModeDark, "light": ModeLight, "auto": ModeAuto, "nonsense": ModeDark,
	} {
		if got := ParseMode(in); got != want {
			t.Errorf("ParseMode(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestDetectAppearanceEnvOverrides(t *testing.T) {
	t.Run("AITINERARY_THEME=light", func(t *testing.T) {
		t.Setenv("AITINERARY_THEME", "light")
		_ = os.Unsetenv("NO_COLOR")
		if a := DetectAppearance(ModeDark); a.Mode != ModeLight {
			t.Errorf("got %+v, want light", a)
		}
	})
	t.Run("NO_COLOR wins", func(t *testing.T) {
		t.Setenv("AITINERARY_THEME", "light")
		t.Setenv("NO_COLOR", "")
		if a := DetectAppearance(ModeDark); !a.NoColor {
			t.Errorf("NO_COLOR should force monochrome; got %+v", a)
		}
	})
}

func withRendererState(t *testing.T) {
	t.Helper()
	prevProfile := lipgloss.ColorProfile()
	prevDark := lipgloss.HasDarkBackground()
	prevAppearance := ActiveAppearance()
	t.Cleanup(func() {
		lipgloss.SetColorProfile(prevProfile)
		lipgloss.SetHasDarkBackground(prevDark)
		ApplyAppearance(prevAppearance)
	})
}

func TestApplyAppearanceNoColorForcesAsciiProfile(t *testing.T) {
	withRendererState(t)
	lipgloss.SetColorProfile(termenv.TrueColor)
	ApplyAppearance(Appearance{Mode: ModeDark, NoColor: true})
	if lipgloss.ColorProfile() != termenv.Ascii {
		t.Errorf("NoColor should force Ascii profile; got %v", lipgloss.ColorProfile())
	}
	out := lipgloss.NewStyle().Foreground(Adaptive(RoleError)).Render("boom")
	if strings.Contains(out, "\x1b[") {
		t.Errorf("NO_COLOR render still contains ANSI: %q", out)
	}
}

func TestThemeStoreRoundTrip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("AITINERARY_CONFIG_DIR", dir)
	saved := Default()
	saved.Mode = "light"
	saved.Primary = "#112233"
	if err := Save(saved, ""); err != nil {
		t.Fatalf("Save: %v", err)
	}
	b, err := os.ReadFile(filepath.Join(dir, "theme.json"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !strings.Contains(string(b), `"mode": "light"`) {
		t.Errorf("missing mode in %s", b)
	}
	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.AppearanceMode() != ModeLight {
		t.Errorf("mode = %v, want light", loaded.AppearanceMode())
	}
	if loaded.Primary != "#112233" {
		t.Errorf("primary = %q", loaded.Primary)
	}
}

func TestAdaptiveUsesCustomPrimary(t *testing.T) {
	Apply(&Theme{Primary: "#aabbcc", Secondary: "#ddeeff", Muted: "#001122"})
	a := Adaptive(RoleAccent)
	if a.Dark != "#aabbcc" {
		t.Errorf("accent dark = %q, want #aabbcc", a.Dark)
	}
	Apply(Default())
}
