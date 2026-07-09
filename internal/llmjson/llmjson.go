// Package llmjson is a leaf, dependency-free JSON-salvage library for parsing
// the noisy JSON that language models emit. aitinerary uses it to parse the
// status_update / epic_draft prompt outputs, which arrive fenced, commented, or
// with trailing commas depending on the model.
//
// Copied wholesale from appr-ai-sal's internal/llmjson (PLAN.md §3).
//
// The single entry point, Parse[T], runs a full sanitize ladder and then
// unmarshals into T. It performs salvage + unmarshal only: no domain-specific
// normalization lives here — that stays in the caller's layer, applied to the
// typed struct Parse returns.
//
// The ladder covers, in escalating order:
//
//	fence      strip a leading/trailing ```json … ``` markdown wrapper
//	extract    pull the first balanced top-level {…} object or […] array out
//	           of surrounding prose
//	comments   remove // line and /* */ block comments (JSON5-style)
//	triple     rewrite Python-style """…""" string values into JSON strings
//	commas     drop trailing commas before } or ]
//
// Each transformation is string-aware (it never edits inside a JSON string
// literal), and the candidate variants are attempted in order until one
// unmarshals successfully into T.
package llmjson

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// Parse runs the full sanitize ladder over raw and unmarshals the first
// candidate that parses into T. It returns the zero value of T and an error
// when no candidate yields valid JSON for T.
func Parse[T any](raw string) (T, error) {
	var zero T
	var lastErr error
	for _, cand := range candidates(raw) {
		var v T
		if err := json.Unmarshal([]byte(cand), &v); err != nil {
			lastErr = err
			continue
		}
		return v, nil
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("llmjson: no JSON value found")
	}
	return zero, lastErr
}

// candidates returns the ordered, de-duplicated list of strings to attempt
// unmarshalling.
func candidates(raw string) []string {
	seen := map[string]struct{}{}
	var out []string
	add := func(s string) {
		s = strings.TrimSpace(s)
		if s == "" {
			return
		}
		if _, ok := seen[s]; ok {
			return
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}

	for _, whole := range cleanVariants(raw) {
		add(whole)
		if obj := ExtractObject(whole); obj != "" {
			for _, c := range combos(obj) {
				add(c)
			}
		}
		if arr := ExtractArray(whole); arr != "" {
			for _, c := range combos(arr) {
				add(c)
			}
		}
	}
	return out
}

func cleanVariants(s string) []string {
	s = strings.TrimSpace(s)
	bases := []string{s}
	if f := StripCodeFence(s); f != s {
		bases = append(bases, f)
	}
	var out []string
	for _, b := range bases {
		out = append(out, combos(b)...)
	}
	return out
}

func combos(s string) []string {
	base := []string{
		s,
		StripComments(s),
		StripTripleQuoted(s),
		StripTripleQuoted(StripComments(s)),
		StripComments(StripTripleQuoted(s)),
	}
	out := make([]string, 0, len(base)*2)
	out = append(out, base...)
	for _, v := range base {
		if tc := StripTrailingCommas(v); tc != v {
			out = append(out, tc)
		}
	}
	return out
}

// StripCodeFence removes a single outer ```…``` wrapper when the entire
// (trimmed) input is fenced.
func StripCodeFence(s string) string {
	t := strings.TrimSpace(s)
	if !strings.HasPrefix(t, "```") {
		return t
	}
	nl := strings.IndexByte(t, '\n')
	if nl < 0 {
		return t
	}
	inner := strings.TrimRight(t[nl+1:], "\n")
	if !strings.HasSuffix(inner, "```") {
		return t
	}
	inner = strings.TrimSuffix(inner, "```")
	return strings.TrimSpace(inner)
}

// StripComments removes // line comments and /* */ block comments that appear
// outside JSON string literals.
func StripComments(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	inStr := false
	esc := false
	for i := 0; i < len(s); {
		c := s[i]
		if inStr {
			if esc {
				b.WriteByte(c)
				esc = false
				i++
				continue
			}
			if c == '\\' {
				b.WriteByte(c)
				esc = true
				i++
				continue
			}
			if c == '"' {
				inStr = false
			}
			b.WriteByte(c)
			i++
			continue
		}
		if c == '"' {
			inStr = true
			b.WriteByte(c)
			i++
			continue
		}
		if c == '/' && i+1 < len(s) && s[i+1] == '/' {
			i += 2
			for i < len(s) && s[i] != '\n' && s[i] != '\r' {
				i++
			}
			if i < len(s) {
				b.WriteByte('\n')
				i++
			}
			continue
		}
		if c == '/' && i+1 < len(s) && s[i+1] == '*' {
			i += 2
			for i+1 < len(s) && !(s[i] == '*' && s[i+1] == '/') {
				i++
			}
			if i+1 < len(s) {
				i += 2
			}
			b.WriteByte(' ')
			continue
		}
		b.WriteByte(c)
		i++
	}
	return b.String()
}

// tripleQuotedField matches a few object keys whose values are sometimes
// emitted as Python-style """…""" instead of a JSON string.
var tripleQuotedField = regexp.MustCompile(
	`(?is)("(?:suggestion|comment|prompt|title|summary|path|body)"\s*:\s*)\s*"""(.*?)"""`,
)

// StripTripleQuoted rewrites invalid triple-quoted string values into valid
// JSON strings.
func StripTripleQuoted(s string) string {
	if !strings.Contains(s, `"""`) {
		return s
	}
	return tripleQuotedField.ReplaceAllStringFunc(s, func(match string) string {
		sub := tripleQuotedField.FindStringSubmatch(match)
		if len(sub) < 3 {
			return match
		}
		quoted, err := json.Marshal(sub[2])
		if err != nil {
			return match
		}
		return sub[1] + string(quoted)
	})
}

// StripTrailingCommas removes a comma immediately followed (ignoring
// whitespace) by a closing } or ] outside a JSON string literal.
func StripTrailingCommas(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	inStr := false
	esc := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if inStr {
			b.WriteByte(c)
			if esc {
				esc = false
				continue
			}
			if c == '\\' {
				esc = true
				continue
			}
			if c == '"' {
				inStr = false
			}
			continue
		}
		if c == '"' {
			inStr = true
			b.WriteByte(c)
			continue
		}
		if c == ',' {
			j := i + 1
			for j < len(s) && (s[j] == ' ' || s[j] == '\t' || s[j] == '\n' || s[j] == '\r') {
				j++
			}
			if j < len(s) && (s[j] == '}' || s[j] == ']') {
				continue
			}
		}
		b.WriteByte(c)
	}
	return b.String()
}

// ExtractObject returns the first balanced top-level {…} block in s, skipping
// braces inside JSON string literals. Returns "" when none exists.
func ExtractObject(s string) string { return extractBalanced(s, '{', '}') }

// ExtractArray returns the first balanced top-level […] block in s, skipping
// brackets inside JSON string literals. Returns "" when none exists.
func ExtractArray(s string) string { return extractBalanced(s, '[', ']') }

func extractBalanced(s string, open, close byte) string {
	start := strings.IndexByte(s, open)
	if start < 0 {
		return ""
	}
	depth := 0
	inStr := false
	esc := false
	for i := start; i < len(s); i++ {
		c := s[i]
		if inStr {
			if esc {
				esc = false
				continue
			}
			if c == '\\' {
				esc = true
				continue
			}
			if c == '"' {
				inStr = false
			}
			continue
		}
		switch c {
		case '"':
			inStr = true
		case open:
			depth++
		case close:
			depth--
			if depth == 0 {
				return s[start : i+1]
			}
		}
	}
	return ""
}
