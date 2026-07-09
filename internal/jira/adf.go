package jira

import "strings"

// Atlassian Document Format (ADF) read/write.
//
// jj-tui only flattened top-level paragraph text. aitinerary extends this in
// both directions: reading understands bullet/ordered lists, headings, and
// blockquotes so ticket descriptions render sensibly; writing can emit either
// plain paragraphs (AI status comments) or an intro-plus-bullets document
// (concise epic bodies, ≤4 bullets — see PLAN.md §7 P7.2).

// ADFDoc is the root ADF document.
type ADFDoc struct {
	Type    string    `json:"type"`    // always "doc"
	Version int       `json:"version"` // always 1
	Content []ADFNode `json:"content"`
}

// ADFNode is a generic ADF node. Only the fields aitinerary reads or writes are
// modelled; unknown attrs/marks are ignored on read and never emitted on write.
type ADFNode struct {
	Type    string    `json:"type"`
	Text    string    `json:"text,omitempty"`
	Content []ADFNode `json:"content,omitempty"`
}

// ADFToText flattens an ADF document to plain text, rendering bullet/ordered
// lists as "- "/"1. " lines and blockquotes with a "> " prefix. Returns "" for
// a nil document.
func ADFToText(doc *ADFDoc) string {
	if doc == nil {
		return ""
	}
	var blocks []string
	for _, b := range doc.Content {
		blocks = append(blocks, strings.Join(renderBlock(b), "\n"))
	}
	return strings.TrimRight(strings.Join(blocks, "\n\n"), "\n")
}

// renderBlock returns the text lines for one block-level node.
func renderBlock(n ADFNode) []string {
	switch n.Type {
	case "paragraph", "heading", "codeBlock":
		return []string{inlineText(n)}
	case "bulletList":
		return renderList(n, func(int) string { return "- " })
	case "orderedList":
		i := 0
		return renderList(n, func(int) string { i++; return itoa(i) + ". " })
	case "blockquote":
		var out []string
		for _, c := range n.Content {
			for _, line := range renderBlock(c) {
				out = append(out, "> "+line)
			}
		}
		return out
	default:
		// Unknown container: recurse; leaf: emit its text.
		if len(n.Content) > 0 {
			var out []string
			for _, c := range n.Content {
				out = append(out, renderBlock(c)...)
			}
			return out
		}
		if n.Text != "" {
			return []string{n.Text}
		}
		return nil
	}
}

// renderList renders each listItem, prefixing its first line with marker(i).
func renderList(n ADFNode, marker func(i int) string) []string {
	var out []string
	for i, item := range n.Content {
		if item.Type != "listItem" {
			continue
		}
		var itemLines []string
		for _, c := range item.Content {
			itemLines = append(itemLines, renderBlock(c)...)
		}
		for j, line := range itemLines {
			if j == 0 {
				out = append(out, marker(i)+line)
			} else {
				out = append(out, "  "+line)
			}
		}
	}
	return out
}

// inlineText concatenates the text of a node's inline children, turning a
// hardBreak into a space.
func inlineText(n ADFNode) string {
	if n.Text != "" && len(n.Content) == 0 {
		return n.Text
	}
	var b strings.Builder
	for _, c := range n.Content {
		switch c.Type {
		case "text":
			b.WriteString(c.Text)
		case "hardBreak":
			b.WriteByte(' ')
		default:
			b.WriteString(inlineText(c))
		}
	}
	return b.String()
}

// TextToADF builds a paragraph-only ADF document from plain text. Blank lines
// (\n\n) separate paragraphs. Empty input yields a single empty paragraph so
// Jira always receives a well-formed doc.
func TextToADF(text string) *ADFDoc {
	paras := splitParagraphs(text)
	doc := &ADFDoc{Type: "doc", Version: 1}
	if len(paras) == 0 {
		doc.Content = []ADFNode{{Type: "paragraph"}}
		return doc
	}
	for _, p := range paras {
		doc.Content = append(doc.Content, paragraph(p))
	}
	return doc
}

// BulletsToADF builds an ADF document with an optional intro paragraph followed
// by a bullet list — the shape the epic-draft prompt emits (a one-line intro
// plus ≤4 bullets). Empty bullets are skipped; an empty intro is omitted.
func BulletsToADF(intro string, bullets []string) *ADFDoc {
	doc := &ADFDoc{Type: "doc", Version: 1}
	if strings.TrimSpace(intro) != "" {
		doc.Content = append(doc.Content, paragraph(intro))
	}
	var items []ADFNode
	for _, b := range bullets {
		if strings.TrimSpace(b) == "" {
			continue
		}
		items = append(items, ADFNode{
			Type:    "listItem",
			Content: []ADFNode{paragraph(b)},
		})
	}
	if len(items) > 0 {
		doc.Content = append(doc.Content, ADFNode{Type: "bulletList", Content: items})
	}
	if len(doc.Content) == 0 {
		doc.Content = []ADFNode{{Type: "paragraph"}}
	}
	return doc
}

func paragraph(text string) ADFNode {
	return ADFNode{Type: "paragraph", Content: []ADFNode{{Type: "text", Text: text}}}
}

func splitParagraphs(text string) []string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	raw := strings.Split(text, "\n\n")
	var out []string
	for _, p := range raw {
		if s := strings.TrimSpace(p); s != "" {
			out = append(out, s)
		}
	}
	return out
}

// itoa is a tiny non-allocating-ish int->string for small list indices,
// avoiding a strconv import in this leaf file.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
