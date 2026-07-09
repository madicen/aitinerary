package aiconfig

import (
	"fmt"
	"net/url"
	"os"
	"strings"
)

func validateBaseURL(raw string) error {
	s := strings.TrimSpace(raw)
	u, err := url.Parse(s)
	if err != nil {
		return fmt.Errorf("base URL %q is not a valid URL: %w", raw, err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("base URL %q must start with http:// or https://", raw)
	}
	if u.Host == "" {
		return fmt.Errorf("base URL %q is missing a host", raw)
	}
	return nil
}

func (p Profile) hasAPIKeySource() bool {
	if strings.TrimSpace(p.APIKey) != "" {
		return true
	}
	if env := strings.TrimSpace(p.APIKeyEnv); env != "" {
		if strings.TrimSpace(os.Getenv(env)) != "" {
			return true
		}
	}
	return strings.TrimSpace(p.APIKeyCmd) != ""
}

// ValidateForProvider checks structural requirements for the profile's provider.
func (p Profile) ValidateForProvider() error {
	label := strings.TrimSpace(p.Name)
	if label == "" {
		label = "(unnamed)"
	}
	prov := p.Provider
	if strings.TrimSpace(string(prov)) == "" {
		prov = ProviderOllama
	}
	switch prov {
	case ProviderOpenAICompatible:
		if strings.TrimSpace(p.BaseURL) == "" {
			return fmt.Errorf("profile %q: openai_compatible requires a base URL", label)
		}
		if err := validateBaseURL(p.BaseURL); err != nil {
			return fmt.Errorf("profile %q: %w", label, err)
		}
	case ProviderGemini:
		if !p.hasAPIKeySource() {
			return fmt.Errorf("profile %q: gemini requires an API key", label)
		}
		if strings.TrimSpace(p.BaseURL) != "" {
			if err := validateBaseURL(p.BaseURL); err != nil {
				return fmt.Errorf("profile %q: %w", label, err)
			}
		}
	case ProviderAnthropic:
		if !p.hasAPIKeySource() {
			return fmt.Errorf("profile %q: anthropic requires an API key", label)
		}
		if strings.TrimSpace(p.BaseURL) != "" {
			if err := validateBaseURL(p.BaseURL); err != nil {
				return fmt.Errorf("profile %q: %w", label, err)
			}
		}
	case ProviderOllama:
		if strings.TrimSpace(p.BaseURL) != "" {
			if err := validateBaseURL(p.BaseURL); err != nil {
				return fmt.Errorf("profile %q: %w", label, err)
			}
		}
	}
	return nil
}

// ValidateForProvider validates the effective settings a run will use.
func (c *Config) ValidateForProvider() error {
	if c == nil {
		return nil
	}
	eff := Profile{
		Name:        c.ActiveProfile,
		Provider:    c.Provider,
		BaseURL:     c.BaseURL,
		Model:       c.Model,
		APIKey:      c.APIKey,
		APIKeyEnv:   c.APIKeyEnv,
		APIKeyCmd:   c.APIKeyCmd,
		Temperature: cloneFloatPtr(c.Temperature),
		TimeoutSec:  c.TimeoutSec,
	}
	if strings.TrimSpace(eff.Name) == "" {
		eff.Name = DefaultProfileName
	}
	return eff.ValidateForProvider()
}
