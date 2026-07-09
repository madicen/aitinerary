package util

import (
	"regexp"
	"strings"

	"github.com/charmbracelet/x/ansi"
)

var bubbleZoneCSI = regexp.MustCompile("\x1b\\[\\d+z")

func enforceMaxLineWidth(s string, maxCols int) string {
	if maxCols < 8 {
		maxCols = 8
	}
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		if bubbleZoneCSI.MatchString(line) {
			continue
		}
		if ansi.StringWidth(line) > maxCols {
			lines[i] = ansi.Truncate(line, maxCols, "")
		}
	}
	return strings.Join(lines, "\n")
}

func hardWrapOverflowLines(s string, maxCols int) string {
	if maxCols < 8 {
		maxCols = 8
	}
	lines := strings.Split(s, "\n")
	var b strings.Builder
	for i, line := range lines {
		if i > 0 {
			b.WriteByte('\n')
		}
		if bubbleZoneCSI.MatchString(line) {
			b.WriteString(line)
			continue
		}
		if ansi.StringWidth(line) <= maxCols {
			b.WriteString(line)
			continue
		}
		b.WriteString(ansi.Hardwrap(line, maxCols, false))
	}
	return b.String()
}

// WrapForViewport hard-wraps text to viewport content width.
func WrapForViewport(s string, contentCols int) string {
	if contentCols < 8 {
		contentCols = 8
	}
	if s == "" {
		return s
	}
	wrapped := ansi.Wrap(s, contentCols, " /.,;:[]{}()=_`\"'+|&")
	wrapped = hardWrapOverflowLines(wrapped, contentCols)
	return enforceMaxLineWidth(wrapped, contentCols)
}
