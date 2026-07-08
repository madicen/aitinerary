package main

import (
	"bytes"
	"strings"
	"testing"
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

func TestRunBannerDefault(t *testing.T) {
	version = "dev"
	var out, errBuf bytes.Buffer
	code := run(nil, &out, &errBuf)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if !strings.Contains(out.String(), "aitinerary dev") {
		t.Fatalf("banner missing version, got %q", out.String())
	}
}

func TestRunBadFlag(t *testing.T) {
	var out, errBuf bytes.Buffer
	code := run([]string{"--nope"}, &out, &errBuf)
	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
}
