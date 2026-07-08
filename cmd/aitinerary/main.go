// Command aitinerary is a TUI for Jira ticket hygiene: it detects tickets that
// should already be Done from merged GitHub PRs, flags hygiene problems, builds
// retroactive epics, and runs a sync-prep sweep before weekly syncs.
//
// This is the P0 scaffold entry point. The interactive TUI is wired in during
// phase P2; until then the binary supports --version and a --help banner so
// the build is verifiable end to end.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
)

// version is the release identifier, overridden at build time via
// -ldflags "-X main.version={{.Version}}" (see the Makefile / goreleaser).
// Defaults to "dev" for local `go run` / `go build` invocations.
var version = "dev"

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run parses flags and returns the process exit code. It is split out from
// main so tests can drive it with explicit args and captured output.
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

	// The interactive TUI is wired in phase P2. Until then, print a banner so
	// the scaffold binary is runnable and self-describing.
	fmt.Fprintf(stdout, "aitinerary %s\n", version)
	fmt.Fprintln(stdout, "TUI not yet wired (phase P2). Run with --version for the build id.")
	if demo {
		fmt.Fprintln(stdout, "(demo mode requested)")
	}
	return 0
}
