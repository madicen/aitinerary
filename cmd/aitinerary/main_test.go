package main

import (
	"bytes"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestRunVersion(t *testing.T) {
	version = "test-1.2.3"
	var out, errBuf bytes.Buffer
	code := run([]string{"--version"}, &out, &errBuf)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if got := strings.TrimSpace(out.String()); got != "test-1.2.3" {
		t.Fatalf("stdout = %q, want %q", got, "test-1.2.3")
	}
	if errBuf.Len() != 0 {
		t.Fatalf("stderr should be empty, got %q", errBuf.String())
	}
}

func TestRunDefaultLaunchesTUI(t *testing.T) {
	version = "dev"
	old := runProgram
	runProgram = func(m tea.Model, opts ...tea.ProgramOption) error { return nil }
	t.Cleanup(func() { runProgram = old })

	var out, errBuf bytes.Buffer
	code := run(nil, &out, &errBuf)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if errBuf.Len() != 0 {
		t.Fatalf("stderr should be empty, got %q", errBuf.String())
	}
}

func TestRunBadFlag(t *testing.T) {
	var out, errBuf bytes.Buffer
	code := run([]string{"--nope"}, &out, &errBuf)
	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
}
