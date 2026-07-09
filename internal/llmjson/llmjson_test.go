package llmjson

import (
	"strings"
	"testing"
)

// draftLike mirrors the epic_draft / status_update prompt output shapes closely
// enough to exercise the full salvage ladder (llmjson is a leaf).
type draftLike struct {
	Title     string   `json:"title"`
	Body      string   `json:"body"`
	Bullets   []string `json:"bullets"`
	StartDate string   `json:"start_date"`
	DueDate   string   `json:"due_date"`
}

func TestParseObjectLadder(t *testing.T) {
	tests := []struct {
		name      string
		raw       string
		wantTitle string
	}{
		{"bare object", `{"title":"ok","body":"b"}`, "ok"},
		{"markdown fence with json tag", "```json\n{\"title\":\"x\"}\n```", "x"},
		{"markdown fence no tag", "```\n{\"title\":\"y\"}\n```", "y"},
		{"fence immediately followed by brace", "```json{\"title\":\"z\"}```", "z"},
		{"leading and trailing prose", "Here is the epic:\n{\"title\":\"p\"}\nHope this helps!", "p"},
		{"line and block comments", "{\n// note\n\"title\":\"ok\",\n/* here */\n\"body\":\"b\"}", "ok"},
		{"trailing commas", `{"title":"tc","bullets":["a","b",],}`, "tc"},
		{"fence plus comments plus trailing comma", "```json\n{\n  // top\n  \"title\": \"combo\",\n  \"bullets\": [],\n}\n```", "combo"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Parse[draftLike](tc.raw)
			if err != nil {
				t.Fatalf("Parse error: %v\nraw:\n%s", err, tc.raw)
			}
			if got.Title != tc.wantTitle {
				t.Fatalf("title = %q, want %q", got.Title, tc.wantTitle)
			}
		})
	}
}

func TestParseTripleQuotedBody(t *testing.T) {
	raw := `{"title":"x","body":"""line1
line2"""}`
	got, err := Parse[draftLike](raw)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got.Body != "line1\nline2" {
		t.Fatalf("body = %q", got.Body)
	}
}

type commentLike struct {
	Comment string `json:"comment"`
}

func TestParseStatusCommentFencedAndCommented(t *testing.T) {
	raw := "Sure:\n```json\n{\n  // concise\n  \"comment\": \"Merged PR closes the work; moving to Done.\",\n}\n```"
	got, err := Parse[commentLike](raw)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if !strings.Contains(got.Comment, "moving to Done") {
		t.Fatalf("comment = %q", got.Comment)
	}
}

func TestParseArrayLadder(t *testing.T) {
	got, err := Parse[[]string]("Here you go:\n[\"a\",\"b\",]\nthanks")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(got) != 2 || got[0] != "a" {
		t.Fatalf("array = %+v", got)
	}
}

func TestParseNonJSONReturnsError(t *testing.T) {
	if _, err := Parse[draftLike]("sorry, I cannot comply, here is prose"); err == nil {
		t.Fatal("expected error on non-JSON input")
	}
}

func TestStripCodeFence(t *testing.T) {
	tests := []struct{ in, want string }{
		{"```json\n{\"a\":1}\n```", `{"a":1}`},
		{"```\n## Heading\n\nbody\n```", "## Heading\n\nbody"},
		{"## Not fenced\n\nbody", "## Not fenced\n\nbody"},
	}
	for _, tc := range tests {
		if got := StripCodeFence(tc.in); got != tc.want {
			t.Errorf("StripCodeFence(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestStripTrailingCommasPreservesStrings(t *testing.T) {
	in := `{"comment":"a, b, and c,","bullets":[],}`
	got := StripTrailingCommas(in)
	if !strings.Contains(got, `"a, b, and c,"`) {
		t.Fatalf("string content mutated: %q", got)
	}
	if strings.Contains(got, "[],}") {
		t.Fatalf("trailing comma before } not removed: %q", got)
	}
}
