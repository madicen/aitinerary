package model

import tea "github.com/charmbracelet/bubbletea"

// Tab is the uniform contract every full-screen tab satisfies.
type Tab interface {
	Init() tea.Cmd
	Update(tea.Msg) (Tab, tea.Cmd)
	View() string
	Resize(w, h int)
	SetContentOrigin(top int)
}

// teaTab is the shape concrete tab sub-models expose (Update returns tea.Model).
type teaTab interface {
	Init() tea.Cmd
	Update(tea.Msg) (tea.Model, tea.Cmd)
	View() string
	Resize(w, h int)
	SetContentOrigin(top int)
}

// tabAdapter wraps a teaTab and presents it as Tab.
type tabAdapter struct{ inner teaTab }

// NewTabAdapter wraps a concrete tab sub-model for the root registry.
func NewTabAdapter(inner teaTab) *tabAdapter { return &tabAdapter{inner: inner} }

func (a *tabAdapter) Init() tea.Cmd { return a.inner.Init() }

func (a *tabAdapter) Update(msg tea.Msg) (Tab, tea.Cmd) {
	upd, cmd := a.inner.Update(msg)
	if t, ok := upd.(teaTab); ok {
		a.inner = t
	}
	return a, cmd
}

func (a *tabAdapter) View() string           { return a.inner.View() }
func (a *tabAdapter) Resize(w, h int)        { a.inner.Resize(w, h) }
func (a *tabAdapter) SetContentOrigin(t int) { a.inner.SetContentOrigin(t) }
