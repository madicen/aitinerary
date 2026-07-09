// Package aiconfig holds AI inference settings (provider, model, keys, timeouts).
//
// Profiles live in ai.json under the app config dir (see internal/appdirs).
// The active profile's fields are mirrored onto Config's top-level fields so
// callers can read cfg.Provider / cfg.Model directly.
package aiconfig

import (
	"fmt"
	"strings"
)

// Provider selects how aitinerary runs inference.
type Provider string

const (
	ProviderGemini           Provider = "gemini"
	ProviderOllama           Provider = "ollama"
	ProviderOpenAICompatible Provider = "openai_compatible"
	ProviderAnthropic        Provider = "anthropic"
)

const (
	DefaultProfileName   = "default"
	DefaultOllamaBaseURL = "http://127.0.0.1:11434/v1"
	DefaultOllamaModel   = "qwen2.5-coder:7b"
)

// Profile is one named (provider, model, baseURL, apiKey, timeout, retry) preset.
type Profile struct {
	Name     string   `json:"name"`
	Provider Provider `json:"provider,omitempty"`
	BaseURL  string   `json:"base_url,omitempty"`
	Model    string   `json:"model,omitempty"`
	APIKey   string   `json:"api_key,omitempty"`
	// APIKeyEnv names an environment variable to read the API key from at
	// resolution time. APIKeyCmd is a command whose stdout is the key.
	// Precedence: explicit api_key > api_key_env > api_key_cmd.
	APIKeyEnv string `json:"api_key_env,omitempty"`
	APIKeyCmd string `json:"api_key_cmd,omitempty"`
	// Temperature is an optional sampling temperature. Omitted from requests
	// when nil (provider default applies).
	Temperature       *float64 `json:"temperature,omitempty"`
	TimeoutSec        int      `json:"timeout_sec,omitempty"`
	RetryMaxAttempts  int      `json:"retry_max_attempts,omitempty"`
	RetryBaseMS       int      `json:"retry_base_ms,omitempty"`
	RetryMaxMS        int      `json:"retry_max_ms,omitempty"`
}

// Clone returns a deep copy safe to mutate.
func (p Profile) Clone() Profile {
	cp := p
	if p.Temperature != nil {
		v := *p.Temperature
		cp.Temperature = &v
	}
	return cp
}

// Summary returns a short "provider · model" label for UI rows.
func (p Profile) Summary() string {
	prov := string(p.Provider)
	if prov == "" {
		prov = string(ProviderOllama)
	}
	model := strings.TrimSpace(p.Model)
	if model == "" {
		model = "(default)"
	}
	return prov + " · " + model
}

// Config is the resolved AI settings after Load / merges.
type Config struct {
	Provider Provider `json:"provider,omitempty"`
	BaseURL  string   `json:"base_url,omitempty"`
	Model    string   `json:"model,omitempty"`
	APIKey   string   `json:"api_key,omitempty"`
	APIKeyEnv string `json:"api_key_env,omitempty"`
	APIKeyCmd string `json:"api_key_cmd,omitempty"`
	Temperature *float64 `json:"temperature,omitempty"`
	TimeoutSec       int `json:"timeout_sec,omitempty"`
	RetryMaxAttempts int `json:"retry_max_attempts,omitempty"`
	RetryBaseMS      int `json:"retry_base_ms,omitempty"`
	RetryMaxMS       int `json:"retry_max_ms,omitempty"`

	Profiles      []Profile `json:"profiles,omitempty"`
	ActiveProfile string    `json:"active_profile,omitempty"`

	oneShot map[overrideField]bool
}

type overrideField int

const (
	fldProvider overrideField = iota
	fldBaseURL
	fldModel
	fldAPIKey
	fldTemperature
	fldTimeout
	fldRetryMaxAttempts
	fldRetryBaseMS
	fldRetryMaxMS
)

func (c *Config) markOneShot(f overrideField) {
	if c == nil {
		return
	}
	if c.oneShot == nil {
		c.oneShot = make(map[overrideField]bool)
	}
	c.oneShot[f] = true
}

func (c *Config) isOneShot(f overrideField) bool {
	return c != nil && c.oneShot != nil && c.oneShot[f]
}

// DefaultConfig returns built-in defaults (local Ollama first).
func DefaultConfig() *Config {
	c := &Config{
		Provider:      ProviderOllama,
		BaseURL:       DefaultOllamaBaseURL,
		Model:         DefaultOllamaModel,
		TimeoutSec:    300,
		ActiveProfile: DefaultProfileName,
	}
	c.Profiles = []Profile{c.snapshotProfile(DefaultProfileName)}
	return c
}

// DefaultPath returns ai.json under the app config dir.
func DefaultPath() string { return configPath() }

// Clone returns a deep copy safe to pass into async work.
func (c *Config) Clone() *Config {
	if c == nil {
		return DefaultConfig()
	}
	cp := *c
	if c.Profiles != nil {
		cp.Profiles = make([]Profile, len(c.Profiles))
		for i, p := range c.Profiles {
			cp.Profiles[i] = p.Clone()
		}
	}
	if c.Temperature != nil {
		v := *c.Temperature
		cp.Temperature = &v
	}
	if c.oneShot != nil {
		cp.oneShot = make(map[overrideField]bool, len(c.oneShot))
		for k, v := range c.oneShot {
			cp.oneShot[k] = v
		}
	}
	return &cp
}

// Merge overlays non-zero fields from o onto c (active profile only).
func (c *Config) Merge(o *Config) {
	if o == nil {
		return
	}
	if o.Provider != "" {
		c.Provider = o.Provider
	}
	if o.BaseURL != "" {
		c.BaseURL = o.BaseURL
	}
	if o.Model != "" {
		c.Model = o.Model
	}
	if o.APIKey != "" {
		c.APIKey = o.APIKey
	}
	if o.APIKeyEnv != "" {
		c.APIKeyEnv = o.APIKeyEnv
	}
	if o.APIKeyCmd != "" {
		c.APIKeyCmd = o.APIKeyCmd
	}
	if o.Temperature != nil {
		v := *o.Temperature
		c.Temperature = &v
	}
	if o.TimeoutSec != 0 {
		c.TimeoutSec = o.TimeoutSec
	}
	if o.RetryMaxAttempts != 0 {
		c.RetryMaxAttempts = o.RetryMaxAttempts
	}
	if o.RetryBaseMS != 0 {
		c.RetryBaseMS = o.RetryBaseMS
	}
	if o.RetryMaxMS != 0 {
		c.RetryMaxMS = o.RetryMaxMS
	}
}

// ParseProvider normalizes a user string to a Provider.
func ParseProvider(s string) (Provider, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, "-", "_")
	switch s {
	case "gemini":
		return ProviderGemini, nil
	case "ollama", "":
		return ProviderOllama, nil
	case "openai_compatible", "openai":
		return ProviderOpenAICompatible, nil
	case "anthropic", "anthropic_api":
		return ProviderAnthropic, nil
	default:
		return "", fmt.Errorf("unknown AI provider %q (want ollama, openai_compatible, gemini, anthropic)", s)
	}
}

func (c *Config) snapshotProfile(name string) Profile {
	return Profile{
		Name:             strings.TrimSpace(name),
		Provider:         c.Provider,
		BaseURL:          c.BaseURL,
		Model:            c.Model,
		APIKey:           c.APIKey,
		APIKeyEnv:        c.APIKeyEnv,
		APIKeyCmd:        c.APIKeyCmd,
		Temperature:      cloneFloatPtr(c.Temperature),
		TimeoutSec:       c.TimeoutSec,
		RetryMaxAttempts: c.RetryMaxAttempts,
		RetryBaseMS:      c.RetryBaseMS,
		RetryMaxMS:       c.RetryMaxMS,
	}
}

func cloneFloatPtr(p *float64) *float64 {
	if p == nil {
		return nil
	}
	v := *p
	return &v
}

func (c *Config) findProfileIndex(name string) (int, bool) {
	target := strings.ToLower(strings.TrimSpace(name))
	if target == "" {
		return -1, false
	}
	for i, p := range c.Profiles {
		if strings.ToLower(strings.TrimSpace(p.Name)) == target {
			return i, true
		}
	}
	return -1, false
}

// ApplyActiveProfile copies the active profile onto top-level fields.
func (c *Config) ApplyActiveProfile() { c.applyActiveProfile() }

func (c *Config) applyActiveProfile() {
	if len(c.Profiles) == 0 {
		return
	}
	idx, ok := c.findProfileIndex(c.ActiveProfile)
	if !ok {
		idx = 0
		c.ActiveProfile = c.Profiles[0].Name
	}
	p := c.Profiles[idx]
	c.Provider = p.Provider
	c.BaseURL = p.BaseURL
	c.Model = p.Model
	c.APIKey = p.APIKey
	c.APIKeyEnv = p.APIKeyEnv
	c.APIKeyCmd = p.APIKeyCmd
	c.Temperature = cloneFloatPtr(p.Temperature)
	c.TimeoutSec = p.TimeoutSec
	c.RetryMaxAttempts = p.RetryMaxAttempts
	c.RetryBaseMS = p.RetryBaseMS
	c.RetryMaxMS = p.RetryMaxMS
}

func (c *Config) syncActiveProfileFromFlat() {
	idx, ok := c.findProfileIndex(c.ActiveProfile)
	if !ok {
		return
	}
	existing := c.Profiles[idx]
	snap := c.snapshotProfile(existing.Name)
	if c.isOneShot(fldProvider) {
		snap.Provider = existing.Provider
	}
	if c.isOneShot(fldBaseURL) {
		snap.BaseURL = existing.BaseURL
	}
	if c.isOneShot(fldModel) {
		snap.Model = existing.Model
	}
	if c.isOneShot(fldAPIKey) {
		snap.APIKey = existing.APIKey
	}
	if c.isOneShot(fldTemperature) {
		snap.Temperature = cloneFloatPtr(existing.Temperature)
	}
	if c.isOneShot(fldTimeout) {
		snap.TimeoutSec = existing.TimeoutSec
	}
	if c.isOneShot(fldRetryMaxAttempts) {
		snap.RetryMaxAttempts = existing.RetryMaxAttempts
	}
	if c.isOneShot(fldRetryBaseMS) {
		snap.RetryBaseMS = existing.RetryBaseMS
	}
	if c.isOneShot(fldRetryMaxMS) {
		snap.RetryMaxMS = existing.RetryMaxMS
	}
	c.Profiles[idx] = snap
}
