package settings

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/madicen/aitinerary/internal/ai"
	"github.com/madicen/aitinerary/internal/aiconfig"
	"github.com/madicen/aitinerary/internal/tui/styles"
	"github.com/madicen/aitinerary/internal/tui/util/dropdown"
)

const (
	profileModeList = iota
	profileModeEdit
)

const (
	profFieldName = iota
	profFieldProvider
	profFieldModel
	profFieldBaseURL
	profFieldAPIKeyEnv
	profFieldAPIKeyCmd
	profFieldTemperature
	profFieldCount
)

var providerLabels = []string{
	string(aiconfig.ProviderOllama),
	string(aiconfig.ProviderOpenAICompatible),
	string(aiconfig.ProviderGemini),
	string(aiconfig.ProviderAnthropic),
}

type aiProfilesValidateMsg struct {
	err    error
	models int
}

type aiProfilesPanel struct {
	cfg      *aiconfig.Config
	demo     bool
	mode     int
	selected int
	field    int

	editingName string
	isNew       bool
	inputs      [profFieldCount]textinput.Model
	providerDD  dropdown.Host

	status     string
	validating bool
	contentTop int
	width      int
}

func newAIProfilesPanel(cfg *aiconfig.Config, demo bool) *aiProfilesPanel {
	p := &aiProfilesPanel{cfg: cfg, demo: demo}
	p.initInputs()
	p.providerDD = *dropdown.New("Select provider…")
	p.providerDD.OnSelect = func(int) tea.Cmd { return p.handleProviderSelect() }
	p.clampSelected()
	return p
}

func profileIndexByName(profiles []aiconfig.Profile, name string) int {
	target := strings.ToLower(strings.TrimSpace(name))
	for i, p := range profiles {
		if strings.ToLower(strings.TrimSpace(p.Name)) == target {
			return i
		}
	}
	return -1
}

func (p *aiProfilesPanel) initInputs() {
	placeholders := [profFieldCount]string{
		"Profile name",
		"",
		"Model id",
		"Base URL",
		"API key env var",
		"API key command",
		"Temperature (optional)",
	}
	for i := range p.inputs {
		ti := textinput.New()
		ti.Placeholder = placeholders[i]
		ti.CharLimit = 256
		if i == profFieldTemperature {
			ti.CharLimit = 16
		}
		p.inputs[i] = ti
	}
}

func (p *aiProfilesPanel) clampSelected() {
	if p.cfg == nil || len(p.cfg.Profiles) == 0 {
		p.selected = 0
		return
	}
	if p.selected < 0 {
		p.selected = 0
	}
	if p.selected >= len(p.cfg.Profiles) {
		p.selected = len(p.cfg.Profiles) - 1
	}
}

func (p *aiProfilesPanel) persist() error {
	if p.demo || p.cfg == nil {
		return nil
	}
	return aiconfig.Save(p.cfg, aiconfig.DefaultPath())
}

func (p *aiProfilesPanel) setValidateResult(msg aiProfilesValidateMsg) {
	p.validating = false
	if msg.err != nil {
		p.status = "validate failed: " + msg.err.Error()
		return
	}
	p.status = fmt.Sprintf("validate ok — %d model(s) listed", msg.models)
}

func (p *aiProfilesPanel) updateKey(msg tea.KeyMsg) tea.Cmd {
	if p.providerDD.Open() {
		if cmd := p.providerDD.Forward(msg); cmd != nil {
			return cmd
		}
		if msg.String() == "esc" {
			p.providerDD.Clear()
			p.rebuildProviderDD()
		}
		return nil
	}

	switch p.mode {
	case profileModeEdit:
		return p.updateEditKey(msg)
	default:
		return p.updateListKey(msg)
	}
}

func (p *aiProfilesPanel) updateListKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "j", "down":
		p.selected++
		p.clampSelected()
	case "k", "up":
		p.selected--
		p.clampSelected()
	case "enter", "a":
		return p.activateSelected()
	case "n":
		return p.addProfileCmd()
	case "e":
		p.beginEdit(false)
		return textinput.Blink
	case "d":
		return p.deleteSelected()
	case "v":
		return p.validateCmd()
	}
	return nil
}

func (p *aiProfilesPanel) updateEditKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "esc":
		p.mode = profileModeList
		p.status = ""
		return nil
	case "tab", "down":
		p.advanceField(1)
		return textinput.Blink
	case "shift+tab", "up":
		p.advanceField(-1)
		return textinput.Blink
	case "enter":
		if p.field == profFieldProvider {
			p.openProviderDD()
			return p.providerDD.Forward(tea.KeyMsg{Type: tea.KeyEnter})
		}
		if p.field < profFieldCount-1 {
			p.advanceField(1)
			return textinput.Blink
		}
		return p.commitEdit()
	}
	if p.field == profFieldProvider {
		if msg.String() == " " {
			p.openProviderDD()
			return p.providerDD.Forward(tea.KeyMsg{Type: tea.KeySpace})
		}
		return nil
	}
	ti := &p.inputs[p.field]
	var cmd tea.Cmd
	*ti, cmd = ti.Update(msg)
	return cmd
}

func (p *aiProfilesPanel) advanceField(delta int) {
	p.blurAll()
	p.field = (p.field + delta + profFieldCount) % profFieldCount
	p.focusField(p.field)
}

func (p *aiProfilesPanel) blurAll() {
	for i := range p.inputs {
		p.inputs[i].Blur()
	}
	p.providerDD.SetFocused(false)
}

func (p *aiProfilesPanel) focusField(i int) {
	p.blurAll()
	p.field = i
	if i == profFieldProvider {
		p.rebuildProviderDD()
		p.providerDD.SetFocused(true)
		return
	}
	p.inputs[i].Focus()
}

func (p *aiProfilesPanel) openProviderDD() {
	p.rebuildProviderDD()
	press := tea.MouseMsg{Button: tea.MouseButtonLeft, Action: tea.MouseActionPress}
	p.providerDD.Forward(press)
}

func (p *aiProfilesPanel) rebuildProviderDD() {
	prov := strings.TrimSpace(p.inputs[profFieldProvider].Value())
	if prov == "" {
		prov = string(aiconfig.ProviderOllama)
	}
	idx := 0
	for i, label := range providerLabels {
		if strings.EqualFold(label, prov) {
			idx = i
			break
		}
	}
	p.providerDD.Rebuild(providerLabels, idx)
	p.providerDD.ContentTop = p.contentTop
}

func (p *aiProfilesPanel) beginEdit(isNew bool) {
	p.mode = profileModeEdit
	p.isNew = isNew
	p.field = 0
	p.status = ""

	var prof aiconfig.Profile
	if isNew {
		prof = aiconfig.Profile{
			Name:     p.uniqueProfileName(),
			Provider: aiconfig.ProviderOllama,
			Model:    aiconfig.DefaultOllamaModel,
			BaseURL:  aiconfig.DefaultOllamaBaseURL,
		}
		p.editingName = ""
	} else {
		p.clampSelected()
		prof = p.cfg.Profiles[p.selected].Clone()
		p.editingName = prof.Name
	}

	p.fillForm(prof)
	p.focusField(0)
}

func (p *aiProfilesPanel) fillForm(prof aiconfig.Profile) {
	p.inputs[profFieldName].SetValue(prof.Name)
	prov := string(prof.Provider)
	if prov == "" {
		prov = string(aiconfig.ProviderOllama)
	}
	p.inputs[profFieldProvider].SetValue(prov)
	p.inputs[profFieldModel].SetValue(prof.Model)
	p.inputs[profFieldBaseURL].SetValue(prof.BaseURL)
	p.inputs[profFieldAPIKeyEnv].SetValue(prof.APIKeyEnv)
	p.inputs[profFieldAPIKeyCmd].SetValue(prof.APIKeyCmd)
	if prof.Temperature != nil {
		p.inputs[profFieldTemperature].SetValue(strconv.FormatFloat(*prof.Temperature, 'f', -1, 64))
	} else {
		p.inputs[profFieldTemperature].SetValue("")
	}
	p.rebuildProviderDD()
}

func (p *aiProfilesPanel) formProfile() (aiconfig.Profile, error) {
	name := strings.TrimSpace(p.inputs[profFieldName].Value())
	if name == "" {
		return aiconfig.Profile{}, fmt.Errorf("profile name is required")
	}
	prov, err := aiconfig.ParseProvider(p.inputs[profFieldProvider].Value())
	if err != nil {
		return aiconfig.Profile{}, err
	}
	var temp *float64
	if s := strings.TrimSpace(p.inputs[profFieldTemperature].Value()); s != "" {
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return aiconfig.Profile{}, fmt.Errorf("temperature must be a number")
		}
		temp = &f
	}

	prof := aiconfig.Profile{
		Name:        name,
		Provider:    prov,
		Model:       strings.TrimSpace(p.inputs[profFieldModel].Value()),
		BaseURL:     strings.TrimSpace(p.inputs[profFieldBaseURL].Value()),
		APIKeyEnv:   strings.TrimSpace(p.inputs[profFieldAPIKeyEnv].Value()),
		APIKeyCmd:   strings.TrimSpace(p.inputs[profFieldAPIKeyCmd].Value()),
		Temperature: temp,
	}
	if !p.isNew {
		if idx := profileIndexByName(p.cfg.Profiles, p.editingName); idx >= 0 {
			existing := p.cfg.Profiles[idx]
			prof.APIKey = existing.APIKey
			prof.TimeoutSec = existing.TimeoutSec
			prof.RetryMaxAttempts = existing.RetryMaxAttempts
			prof.RetryBaseMS = existing.RetryBaseMS
			prof.RetryMaxMS = existing.RetryMaxMS
		}
	}
	return prof, nil
}

func (p *aiProfilesPanel) commitEdit() tea.Cmd {
	prof, err := p.formProfile()
	if err != nil {
		p.status = err.Error()
		return nil
	}
	if err := prof.ValidateForProvider(); err != nil {
		p.status = err.Error()
		return nil
	}
	if p.isNew {
		if err := p.cfg.AddProfile(prof); err != nil {
			p.status = err.Error()
			return nil
		}
	} else {
		if err := p.cfg.UpdateProfile(p.editingName, prof); err != nil {
			p.status = err.Error()
			return nil
		}
	}
	if err := p.persist(); err != nil {
		p.status = "save failed: " + err.Error()
		return nil
	}
	if idx := profileIndexByName(p.cfg.Profiles, prof.Name); idx >= 0 {
		p.selected = idx
	}
	p.mode = profileModeList
	p.status = ""
	return nil
}

func (p *aiProfilesPanel) uniqueProfileName() string {
	base := "profile"
	for n := 2; ; n++ {
		candidate := fmt.Sprintf("%s-%d", base, n)
		if profileIndexByName(p.cfg.Profiles, candidate) < 0 {
			return candidate
		}
	}
}

func (p *aiProfilesPanel) activateSelected() tea.Cmd {
	p.clampSelected()
	name := p.cfg.Profiles[p.selected].Name
	if err := p.cfg.SetActive(name); err != nil {
		p.status = err.Error()
		return nil
	}
	if err := p.persist(); err != nil {
		p.status = "save failed: " + err.Error()
		return nil
	}
	p.status = ""
	return nil
}

func (p *aiProfilesPanel) addProfileCmd() tea.Cmd {
	prof := aiconfig.Profile{
		Name:     p.uniqueProfileName(),
		Provider: aiconfig.ProviderOllama,
		Model:    aiconfig.DefaultOllamaModel,
		BaseURL:  aiconfig.DefaultOllamaBaseURL,
	}
	if err := p.cfg.AddProfile(prof); err != nil {
		p.status = err.Error()
		return nil
	}
	if err := p.persist(); err != nil {
		p.status = "save failed: " + err.Error()
		return nil
	}
	if idx := profileIndexByName(p.cfg.Profiles, prof.Name); idx >= 0 {
		p.selected = idx
	}
	p.status = ""
	return nil
}

func (p *aiProfilesPanel) deleteSelected() tea.Cmd {
	p.clampSelected()
	name := p.cfg.Profiles[p.selected].Name
	if err := p.cfg.DeleteProfile(name); err != nil {
		p.status = err.Error()
		return nil
	}
	if err := p.persist(); err != nil {
		p.status = "save failed: " + err.Error()
		return nil
	}
	p.clampSelected()
	p.status = ""
	return nil
}

func (p *aiProfilesPanel) validateCmd() tea.Cmd {
	p.clampSelected()
	prof := p.cfg.Profiles[p.selected].Clone()
	cfg := profileConfig(p.cfg, prof)
	p.validating = true
	p.status = "validating…"
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()
		models, err := ai.ListModels(ctx, cfg)
		return aiProfilesValidateMsg{err: err, models: len(models)}
	}
}

func profileConfig(base *aiconfig.Config, p aiconfig.Profile) *aiconfig.Config {
	cfg := base.Clone()
	cfg.Provider = p.Provider
	cfg.BaseURL = p.BaseURL
	cfg.Model = p.Model
	cfg.APIKey = p.APIKey
	cfg.APIKeyEnv = p.APIKeyEnv
	cfg.APIKeyCmd = p.APIKeyCmd
	cfg.Temperature = p.Temperature
	if cfg.Provider == "" {
		cfg.Provider = aiconfig.ProviderOllama
	}
	return cfg
}

func (p *aiProfilesPanel) render(width int) string {
	p.width = width
	if p.mode == profileModeEdit {
		return p.renderEdit()
	}
	return p.renderList()
}

func (p *aiProfilesPanel) renderList() string {
	r := p.cfg.Redacted()
	var b strings.Builder
	b.WriteString(styles.DimStyle.Render("AI profiles · j/k select · enter/a activate · n add · e edit · d delete · v validate"))
	b.WriteString("\n\n")
	active := r.ActiveProfile
	if active == "" {
		active = aiconfig.DefaultProfileName
	}
	b.WriteString(fmt.Sprintf("Active: %s\n\n", styles.BoldStyle.Render(active)))
	for i, prof := range r.Profiles {
		marker := "  "
		if i == p.selected {
			marker = styles.OkStyle.Render("▶ ")
		} else if strings.EqualFold(prof.Name, active) {
			marker = styles.DimStyle.Render("● ")
		}
		line := fmt.Sprintf("%s%s — %s", marker, prof.Name, prof.Summary())
		if strings.TrimSpace(prof.APIKey) != "" {
			line += styles.DimStyle.Render(" · key ****")
		}
		b.WriteString(line)
		b.WriteString("\n")
	}
	b.WriteString("\n")
	if p.validating {
		b.WriteString(styles.DimStyle.Render("validating…"))
	} else if p.status != "" {
		b.WriteString(styles.DimStyle.Render(p.status))
	}
	b.WriteString("\n")
	b.WriteString(styles.DimStyle.Render("Config: " + aiconfig.DefaultPath()))
	return b.String()
}

func (p *aiProfilesPanel) renderEdit() string {
	var b strings.Builder
	b.WriteString(styles.DimStyle.Render("Edit profile · tab next field · enter save · esc cancel"))
	b.WriteString("\n\n")
	labels := []string{"Name", "Provider", "Model", "Base URL", "API key env", "API key cmd", "Temperature"}
	for i, label := range labels {
		focus := "  "
		if i == p.field {
			focus = styles.OkStyle.Render("▶ ")
		}
		b.WriteString(fmt.Sprintf("%s%-14s ", focus, label))
		if i == profFieldProvider {
			if p.providerDD.Built() {
				b.WriteString(p.providerDD.TriggerView())
			} else {
				b.WriteString(p.inputs[i].Value())
			}
		} else {
			b.WriteString(p.inputs[i].View())
		}
		b.WriteString("\n")
	}
	if !p.isNew {
		if idx := profileIndexByName(p.cfg.Profiles, p.editingName); idx >= 0 {
			if strings.TrimSpace(p.cfg.Profiles[idx].APIKey) != "" {
				b.WriteString("\n")
				b.WriteString(styles.DimStyle.Render("Stored api_key: **** (edit ai.json to change)"))
			}
		}
	}
	b.WriteString("\n")
	if p.status != "" {
		b.WriteString(styles.DimStyle.Render(p.status))
	}
	return b.String()
}

func (p *aiProfilesPanel) applyOverlays(panel string, width, height int) string {
	if p.mode != profileModeEdit || !p.providerDD.Open() {
		return panel
	}
	row := 4 + profFieldProvider // title + blank + hint + first fields
	return p.providerDD.Composite(panel, row, 16, width, height)
}

func (p *aiProfilesPanel) handleProviderSelect() tea.Cmd {
	if !p.providerDD.Built() {
		return nil
	}
	idx := p.providerDD.SelectedIndex()
	if idx < 0 || idx >= len(providerLabels) {
		return nil
	}
	p.inputs[profFieldProvider].SetValue(providerLabels[idx])
	p.providerDD.Clear()
	p.rebuildProviderDD()
	return nil
}
