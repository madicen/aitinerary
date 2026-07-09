// Package commands is the command registry shared by keys and the ctrl+k palette.
package commands

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/sahilm/fuzzy"
)

// Context is the app-state snapshot a Command's Enabled predicate reads.
type Context struct {
	Mode string // board | queue | epics | settings | help
}

// Command is one registry entry.
type Command struct {
	ID       string
	Title    string
	Category string
	Binding  key.Binding
	Enabled  func(Context) bool
	Run      func() tea.Cmd
}

// HasBinding reports whether the command advertises a keyboard shortcut.
func (c Command) HasBinding() bool {
	return len(c.Binding.Keys()) > 0
}

// ShortcutLabel returns the display key for the shortcut, or "".
func (c Command) ShortcutLabel() string {
	if !c.HasBinding() {
		return ""
	}
	return c.Binding.Help().Key
}

// Registry is an ordered set of commands.
type Registry struct {
	cmds []Command
}

// New returns an empty registry.
func New() *Registry { return &Registry{} }

// Register appends a command preserving registration order.
func (r *Registry) Register(c Command) { r.cmds = append(r.cmds, c) }

// All returns every registered command.
func (r *Registry) All() []Command { return r.cmds }

// Enabled returns commands whose Enabled predicate accepts ctx.
func (r *Registry) Enabled(ctx Context) []Command {
	out := make([]Command, 0, len(r.cmds))
	for _, c := range r.cmds {
		if c.Enabled == nil || c.Enabled(ctx) {
			out = append(out, c)
		}
	}
	return out
}

// Find returns the command with the given id.
func (r *Registry) Find(id string) (Command, bool) {
	for _, c := range r.cmds {
		if c.ID == id {
			return c, true
		}
	}
	return Command{}, false
}

// Filter fuzzy-matches cmds against query over titles.
func Filter(cmds []Command, query string) []Command {
	query = strings.TrimSpace(query)
	if query == "" {
		return cmds
	}
	titles := make([]string, len(cmds))
	for i, c := range cmds {
		titles[i] = c.Title
	}
	matches := fuzzy.Find(query, titles)
	out := make([]Command, 0, len(matches))
	for _, mt := range matches {
		out = append(out, cmds[mt.Index])
	}
	return out
}
