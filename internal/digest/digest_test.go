package digest

import (
	"strings"
	"testing"
	"time"

	"github.com/madicen/aitinerary/internal/demo"
)

func TestWeekFilename(t *testing.T) {
	got := WeekFilename(time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC))
	if got != "2026-W25.md" {
		t.Fatalf("WeekFilename = %q, want 2026-W25.md", got)
	}
}

func TestGenerateMarkdownDemo(t *testing.T) {
	got := GenerateMarkdown(nil, true, time.Now())
	want := demo.WeeklyDigest()
	if got != want {
		t.Fatalf("demo digest mismatch:\n--- want ---\n%s\n--- got ---\n%s", want, got)
	}
}

func TestGenerateMarkdownFromTickets(t *testing.T) {
	got := GenerateMarkdown(demo.Tickets(), false, time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC))
	if !strings.Contains(got, "APP-101") {
		t.Fatalf("expected APP-101 in digest:\n%s", got)
	}
	if !strings.Contains(got, "HELP-201") {
		t.Fatalf("expected HELP-201 in digest:\n%s", got)
	}
}

func TestWriteWeekFile(t *testing.T) {
	t.Setenv("AITINERARY_CONFIG_DIR", t.TempDir())
	path, err := WriteWeekFile("# test\n", time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(path, "2026-W02.md") {
		t.Fatalf("path = %q", path)
	}
}
