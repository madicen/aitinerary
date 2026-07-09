// Package digest writes weekly sync-prep digest markdown to disk (P9.3).
package digest

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/madicen/aitinerary/internal/appdirs"
	"github.com/madicen/aitinerary/internal/demo"
	"github.com/madicen/aitinerary/internal/jira"
	"github.com/madicen/aitinerary/internal/tui/util"
)

// ExportedMsg reports a completed digest export.
type ExportedMsg struct {
	Path    string
	Content string
	Err     error
}

// WeekFilename returns YYYY-WW.md for the given time (ISO week).
func WeekFilename(t time.Time) string {
	y, w := t.ISOWeek()
	return fmt.Sprintf("%04d-W%02d.md", y, w)
}

// GenerateMarkdown builds a digest from tickets. Demo mode uses canned output.
func GenerateMarkdown(tickets []jira.Ticket, demoMode bool, now time.Time) string {
	if demoMode || len(tickets) == 0 {
		return demo.WeeklyDigest()
	}
	var b stringsBuilder
	b.WriteString(fmt.Sprintf("# Weekly digest (%s)\n\n", now.Format("2006-01-02")))
	for _, pk := range projectKeys(tickets) {
		b.WriteString("## " + pk + "\n")
		var active []string
		var done []string
		for _, t := range tickets {
			if projectOf(t.Key) != pk {
				continue
			}
			if t.IsDone() {
				done = append(done, fmt.Sprintf("- %s %s", t.Key, t.Summary))
			} else {
				active = append(active, fmt.Sprintf("- %s %s (%s)", t.Key, t.Summary, t.Status))
			}
		}
		if len(done) > 0 {
			b.WriteString("### Done this week\n")
			for _, line := range done {
				b.WriteString(line + "\n")
			}
			b.WriteString("\n")
		}
		if len(active) > 0 {
			b.WriteString("### In flight\n")
			for _, line := range active {
				b.WriteString(line + "\n")
			}
			b.WriteString("\n")
		}
	}
	return b.String()
}

type stringsBuilder struct{ s string }

func (b *stringsBuilder) WriteString(s string) { b.s += s }
func (b *stringsBuilder) String() string       { return b.s }

func projectKeys(tickets []jira.Ticket) []string {
	seen := map[string]bool{}
	var out []string
	for _, t := range tickets {
		pk := projectOf(t.Key)
		if pk == "" || seen[pk] {
			continue
		}
		seen[pk] = true
		out = append(out, pk)
	}
	return out
}

func projectOf(key string) string {
	for i := 0; i < len(key); i++ {
		if key[i] == '-' {
			return key[:i]
		}
	}
	return key
}

// WriteWeekFile writes content to ConfigDir/digests/YYYY-WW.md.
func WriteWeekFile(content string, now time.Time) (string, error) {
	dir := filepath.Join(appdirs.ConfigDir(), "digests")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(dir, WeekFilename(now))
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return "", err
	}
	return path, nil
}

// ExportCmd writes the digest file and copies markdown to the clipboard.
func ExportCmd(tickets []jira.Ticket, demoMode bool, now time.Time) tea.Cmd {
	return func() tea.Msg {
		content := GenerateMarkdown(tickets, demoMode, now)
		path, err := WriteWeekFile(content, now)
		if err != nil {
			return ExportedMsg{Content: content, Err: err}
		}
		_, clipErr := copyText(content)
		return ExportedMsg{Path: path, Content: content, Err: clipErr}
	}
}

func copyText(text string) (viaOSC52 bool, err error) {
	cmd := util.CopyPlainTextCmd(text)
	msg := cmd()
	if cm, ok := msg.(util.ClipboardCopiedMsg); ok {
		return cm.ViaOSC52, cm.Err
	}
	return false, nil
}
