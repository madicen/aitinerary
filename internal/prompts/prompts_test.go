package prompts

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newRegistry(t *testing.T) *Registry {
	t.Helper()
	r, err := New("")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return r
}

func TestRenderStatusUpdateEnforcesContract(t *testing.T) {
	r := newRegistry(t)
	got, err := r.RenderStatusUpdate(StatusUpdateInput{
		Key: "APP-1", Summary: "Wire the board", Status: "In Progress",
		PRs:      []PRRef{{Repo: "o/r", Number: 12, Title: "board list"}},
		Diffstat: "3 files, +120 -4",
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	// Content contract: brevity + no-fluff + factual + JSON-only.
	for _, must := range []string{"AT MOST 2 sentences", "first-person fluff", "JSON only", "APP-1", "o/r#12"} {
		if !strings.Contains(got, must) {
			t.Errorf("rendered status_update missing %q\n---\n%s", must, got)
		}
	}
}

func TestRenderEpicDraftListsTickets(t *testing.T) {
	r := newRegistry(t)
	got, err := r.RenderEpicDraft(EpicDraftInput{Tickets: []EpicTicket{
		{Key: "APP-1", Summary: "list", DueDate: "2026-08-01"},
		{Key: "APP-2", Summary: "detail"},
	}})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	for _, must := range []string{"AT MOST 4 short bullets", "APP-1 list (due 2026-08-01)", "APP-2 detail", "start_date"} {
		if !strings.Contains(got, must) {
			t.Errorf("rendered epic_draft missing %q\n---\n%s", must, got)
		}
	}
}

func TestParseStatusUpdateLenient(t *testing.T) {
	// Fenced + commented + trailing comma — the salvage ladder must recover it.
	raw := "```json\n{\n  // concise\n  \"comment\": \"Merged o/r#12 closes the work; moving to Done.\",\n}\n```"
	out, err := ParseStatusUpdate(raw)
	if err != nil {
		t.Fatalf("ParseStatusUpdate: %v", err)
	}
	if !strings.Contains(out.Comment, "moving to Done") {
		t.Fatalf("comment = %q", out.Comment)
	}
}

func TestParseStatusUpdateRejectsEmpty(t *testing.T) {
	if _, err := ParseStatusUpdate(`{"comment":"   "}`); err == nil {
		t.Fatal("empty comment should be rejected")
	}
}

func TestParseEpicDraftAndBullets(t *testing.T) {
	raw := `{"title":"Board revamp","body":"- ship list\n- ship detail","start_date":"2026-07-15","due_date":"2026-08-15"}`
	out, err := ParseEpicDraft(raw)
	if err != nil {
		t.Fatalf("ParseEpicDraft: %v", err)
	}
	if out.Title != "Board revamp" || out.StartDate != "2026-07-15" {
		t.Fatalf("parsed = %+v", out)
	}
	bullets := out.Bullets()
	if len(bullets) != 2 || bullets[0] != "ship list" || bullets[1] != "ship detail" {
		t.Fatalf("bullets = %+v", bullets)
	}
}

func TestOverrideTemplateWins(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "status_update.tmpl"), []byte("OVERRIDE {{.Key}}"), 0o600); err != nil {
		t.Fatal(err)
	}
	r, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	got, err := r.RenderStatusUpdate(StatusUpdateInput{Key: "APP-9"})
	if err != nil {
		t.Fatal(err)
	}
	if got != "OVERRIDE APP-9" {
		t.Fatalf("override not used, got %q", got)
	}
}
