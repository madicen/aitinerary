package jira

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestADFToTextParagraphsAndBullets(t *testing.T) {
	raw := `{"type":"doc","version":1,"content":[
      {"type":"paragraph","content":[{"type":"text","text":"Intro line."}]},
      {"type":"bulletList","content":[
        {"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"first"}]}]},
        {"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"second"}]}]}
      ]}
    ]}`
	var doc ADFDoc
	if err := json.Unmarshal([]byte(raw), &doc); err != nil {
		t.Fatal(err)
	}
	got := ADFToText(&doc)
	want := "Intro line.\n\n- first\n- second"
	if got != want {
		t.Fatalf("ADFToText =\n%q\nwant\n%q", got, want)
	}
}

func TestADFToTextNil(t *testing.T) {
	if ADFToText(nil) != "" {
		t.Fatal("nil doc should flatten to empty string")
	}
}

func TestTextToADFRoundTrip(t *testing.T) {
	doc := TextToADF("Line one.\n\nLine two.")
	if doc.Type != "doc" || doc.Version != 1 {
		t.Fatalf("bad doc header: %+v", doc)
	}
	if len(doc.Content) != 2 {
		t.Fatalf("want 2 paragraphs, got %d", len(doc.Content))
	}
	// Re-flatten should recover the paragraphs separated by a blank line.
	if got := ADFToText(doc); got != "Line one.\n\nLine two." {
		t.Fatalf("round trip = %q", got)
	}
	// It must marshal to valid JSON with text nodes.
	b, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"type":"text"`) {
		t.Fatalf("marshalled ADF missing text node: %s", b)
	}
}

func TestBulletsToADF(t *testing.T) {
	doc := BulletsToADF("Retro epic covering:", []string{"a", "", "b"})
	// intro paragraph + bulletList (empty bullet skipped)
	if len(doc.Content) != 2 {
		t.Fatalf("want intro + list, got %d blocks", len(doc.Content))
	}
	list := doc.Content[1]
	if list.Type != "bulletList" || len(list.Content) != 2 {
		t.Fatalf("bulletList should have 2 items, got %+v", list)
	}
	if got := ADFToText(doc); got != "Retro epic covering:\n\n- a\n- b" {
		t.Fatalf("flattened = %q", got)
	}
}

func TestBulletsToADFEmptyStillValid(t *testing.T) {
	doc := BulletsToADF("", nil)
	if len(doc.Content) != 1 || doc.Content[0].Type != "paragraph" {
		t.Fatalf("empty input must yield one empty paragraph, got %+v", doc.Content)
	}
}
