// Command aitinerary is a TUI for Jira ticket hygiene.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"github.com/madicen/aitinerary/internal/aiconfig"
	"github.com/madicen/aitinerary/internal/applog"
	"github.com/madicen/aitinerary/internal/config"
	"github.com/madicen/aitinerary/internal/theme"
	"github.com/madicen/aitinerary/internal/tui"
)

var version = "dev"

// runProgram is the tea program runner. Tests replace it with a no-op.
var runProgram = func(m tea.Model, opts ...tea.ProgramOption) error {
	_, err := tea.NewProgram(m, opts...).Run()
	return err
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("aitinerary", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var (
		showVersion bool
		demo        bool
	)
	fs.BoolVar(&showVersion, "version", false, "print version and exit")
	fs.BoolVar(&demo, "demo", false, "run against isolated demo data (no live Jira/GitHub)")
	fs.Usage = func() {
		fmt.Fprintf(stderr, "aitinerary %s — Jira ticket hygiene TUI\n\n", version)
		fmt.Fprintf(stderr, "Usage:\n  aitinerary [flags]\n\nFlags:\n")
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		return 2
	}

	if showVersion {
		fmt.Fprintln(stdout, version)
		return 0
	}

	if err := runTUI(demo); err != nil {
		fmt.Fprintf(stderr, "aitinerary: %v\n", err)
		return 1
	}
	return 0
}

func runTUI(demoMode bool) error {
	if err := applog.Init(version); err != nil {
		fmt.Fprintf(os.Stderr, "aitinerary: logging disabled: %v\n", err)
	}

	if demoMode {
		if err := configureDemoEnv(); err != nil {
			return fmt.Errorf("demo setup: %w", err)
		}
	}

	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}

	aiCfg, err := aiconfig.Load()
	if err != nil {
		return fmt.Errorf("ai config: %w", err)
	}

	th := theme.FromConfig(cfg)
	if saved, err := theme.Load(); err == nil && saved != nil {
		th = saved
	}
	theme.Apply(th)
	theme.SetupRendering(th)

	defer tui.FlushMouse()

	m := tui.New(tui.Options{
		Config:   cfg,
		Demo:     demoMode,
		AIConfig: aiCfg,
	})
	return runProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
}

func configureDemoEnv() error {
	lipgloss.SetColorProfile(termenv.TrueColor)

	demoRoot := strings.TrimSpace(os.Getenv("AITINERARY_DEMO_DIR"))
	if demoRoot == "" {
		var err error
		demoRoot, err = os.MkdirTemp("", "aitinerary-demo-")
		if err != nil {
			return fmt.Errorf("mkdir demo dir: %w", err)
		}
	}
	cfgDir := filepath.Join(demoRoot, "config")
	cacheDir := filepath.Join(demoRoot, "cache")
	for _, d := range []string{cfgDir, cacheDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return fmt.Errorf("mkdir %s: %w", d, err)
		}
	}
	if err := os.Setenv("AITINERARY_CONFIG_DIR", cfgDir); err != nil {
		return err
	}
	if err := os.Setenv("AITINERARY_CACHE_DIR", cacheDir); err != nil {
		return err
	}
	return nil
}
