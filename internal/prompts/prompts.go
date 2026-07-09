// Package prompts owns aitinerary's AI prompt templates and the typed parsing
// of their outputs (PLAN.md §7 P7.2).
//
// Templates are embedded via go:embed and may be overridden per-user by
// dropping a file of the same name in <ConfigDir>/prompts/. The templates
// enforce the content contract from PLAN.md §2: brief, factual, no first-person
// filler, no invented facts. Output parsing goes through llmjson so a model
// that fences or comments its JSON still parses.
package prompts

import (
	"bytes"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"text/template"
)

//go:embed templates/*.tmpl
var embedded embed.FS

// Template names (also the embedded/override file stem).
const (
	StatusUpdate        = "status_update"
	EpicDraft           = "epic_draft"
	TicketSummaryDigest = "ticket_summary_digest"
)

// Registry renders templates, preferring a user override file over the embedded
// default.
type Registry struct {
	overrideDir string
	base        *template.Template
}

// New parses the embedded templates and records the override directory
// (typically <ConfigDir>/prompts). overrideDir may be "" to disable overrides.
func New(overrideDir string) (*Registry, error) {
	base, err := template.ParseFS(embedded, "templates/*.tmpl")
	if err != nil {
		return nil, fmt.Errorf("parse embedded prompts: %w", err)
	}
	return &Registry{overrideDir: overrideDir, base: base}, nil
}

// render fills the named template with data. If <overrideDir>/<name>.tmpl
// exists it is parsed fresh and used; otherwise the embedded template is used.
func (r *Registry) render(name string, data any) (string, error) {
	var tmpl *template.Template
	if r.overrideDir != "" {
		p := filepath.Join(r.overrideDir, name+".tmpl")
		if b, err := os.ReadFile(p); err == nil {
			t, perr := template.New(name).Parse(string(b))
			if perr != nil {
				return "", fmt.Errorf("parse override %s: %w", p, perr)
			}
			tmpl = t
		}
	}
	if tmpl == nil {
		tmpl = r.base.Lookup(name + ".tmpl")
	}
	if tmpl == nil {
		return "", fmt.Errorf("unknown prompt template %q", name)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("render %s: %w", name, err)
	}
	return buf.String(), nil
}

// RenderStatusUpdate renders the status_update prompt.
func (r *Registry) RenderStatusUpdate(in StatusUpdateInput) (string, error) {
	return r.render(StatusUpdate, in)
}

// RenderEpicDraft renders the epic_draft prompt.
func (r *Registry) RenderEpicDraft(in EpicDraftInput) (string, error) {
	return r.render(EpicDraft, in)
}

// RenderDigest renders the ticket_summary_digest prompt.
func (r *Registry) RenderDigest(in DigestInput) (string, error) {
	return r.render(TicketSummaryDigest, in)
}
