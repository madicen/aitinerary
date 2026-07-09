package aiconfig

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/madicen/aitinerary/internal/appdirs"
	"github.com/madicen/aitinerary/internal/applog"
)

func configPath() string {
	return appdirs.ConfigFile("ai.json")
}

// Load reads defaults, optional ai.json, then AITINERARY_* environment overrides.
func Load() (*Config, error) {
	c := DefaultConfig()
	path := DefaultPath()
	if b, err := os.ReadFile(path); err == nil {
		var fileCfg Config
		if err := json.Unmarshal(b, &fileCfg); err != nil {
			return nil, fmt.Errorf("parse %s: %w", path, err)
		}
		if len(fileCfg.Profiles) == 0 {
			c.Merge(&fileCfg)
			c.Profiles = []Profile{c.snapshotProfile(DefaultProfileName)}
			c.ActiveProfile = DefaultProfileName
		} else {
			c.Profiles = make([]Profile, len(fileCfg.Profiles))
			for i, p := range fileCfg.Profiles {
				c.Profiles[i] = p.Clone()
			}
			c.ActiveProfile = strings.TrimSpace(fileCfg.ActiveProfile)
			c.applyActiveProfile()
		}
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	mergeEnv(c)
	if err := c.normalize(); err != nil {
		return nil, err
	}
	return c, nil
}

func warnInvalidEnv(name, value, want string) {
	applog.Warn("ignoring invalid environment override",
		"env", name, "value", value, "want", want)
}

func mergeEnv(c *Config) {
	if v := strings.TrimSpace(os.Getenv("AITINERARY_AI_PROVIDER")); v != "" {
		if p, err := ParseProvider(v); err == nil {
			c.Provider = p
			c.markOneShot(fldProvider)
		} else {
			warnInvalidEnv("AITINERARY_AI_PROVIDER", v, "ollama, openai_compatible, gemini, anthropic")
		}
	}
	if v := os.Getenv("AITINERARY_AI_BASE_URL"); v != "" {
		c.BaseURL = strings.TrimSpace(v)
		c.markOneShot(fldBaseURL)
	}
	if v := strings.TrimSpace(os.Getenv("AITINERARY_AI_MODEL")); v != "" {
		c.Model = v
		c.markOneShot(fldModel)
	}
	if v := os.Getenv("AITINERARY_AI_API_KEY"); v != "" {
		c.APIKey = v
		c.markOneShot(fldAPIKey)
	}
	if v := strings.TrimSpace(os.Getenv("AITINERARY_AI_TEMPERATURE")); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			c.Temperature = &f
			c.markOneShot(fldTemperature)
		} else {
			warnInvalidEnv("AITINERARY_AI_TEMPERATURE", v, "a number")
		}
	}
	if v := strings.TrimSpace(os.Getenv("AITINERARY_AI_TIMEOUT_SEC")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			c.TimeoutSec = n
			c.markOneShot(fldTimeout)
		} else {
			warnInvalidEnv("AITINERARY_AI_TIMEOUT_SEC", v, "a positive integer (seconds)")
		}
	}
	if v := strings.TrimSpace(os.Getenv("AITINERARY_AI_RETRY_MAX_ATTEMPTS")); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			c.RetryMaxAttempts = n
			c.markOneShot(fldRetryMaxAttempts)
		} else {
			warnInvalidEnv("AITINERARY_AI_RETRY_MAX_ATTEMPTS", v, "an integer")
		}
	}
	if v := strings.TrimSpace(os.Getenv("AITINERARY_AI_RETRY_BASE_MS")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			c.RetryBaseMS = n
			c.markOneShot(fldRetryBaseMS)
		} else {
			warnInvalidEnv("AITINERARY_AI_RETRY_BASE_MS", v, "a positive integer (milliseconds)")
		}
	}
	if v := strings.TrimSpace(os.Getenv("AITINERARY_AI_RETRY_MAX_MS")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			c.RetryMaxMS = n
			c.markOneShot(fldRetryMaxMS)
		} else {
			warnInvalidEnv("AITINERARY_AI_RETRY_MAX_MS", v, "a positive integer (milliseconds)")
		}
	}
}

func (c *Config) normalize() error {
	if c.Provider == "" {
		c.Provider = ProviderOllama
	}
	p, err := ParseProvider(string(c.Provider))
	if err != nil {
		return err
	}
	c.Provider = p
	if c.TimeoutSec <= 0 {
		c.TimeoutSec = 300
	}
	if c.RetryMaxAttempts == 0 {
		c.RetryMaxAttempts = 5
	}
	if c.RetryMaxAttempts > 30 {
		c.RetryMaxAttempts = 30
	}
	if c.RetryBaseMS == 0 {
		c.RetryBaseMS = 1500
	}
	if c.RetryMaxMS == 0 {
		c.RetryMaxMS = 120000
	}
	if c.RetryBaseMS > c.RetryMaxMS {
		c.RetryBaseMS = c.RetryMaxMS
	}
	if len(c.Profiles) == 0 {
		c.ActiveProfile = DefaultProfileName
		c.Profiles = []Profile{c.snapshotProfile(DefaultProfileName)}
	} else {
		if _, ok := c.findProfileIndex(c.ActiveProfile); !ok {
			c.ActiveProfile = c.Profiles[0].Name
		}
		c.syncActiveProfileFromFlat()
	}
	if c.Provider == ProviderOllama {
		if strings.TrimSpace(c.BaseURL) == "" {
			c.BaseURL = DefaultOllamaBaseURL
		}
		if strings.TrimSpace(c.Model) == "" {
			c.Model = DefaultOllamaModel
		}
	}
	return nil
}

// MergeFlags applies non-empty CLI overrides. Pass timeoutSec < 0 to leave timeout unchanged.
func (c *Config) MergeFlags(provider, baseURL, model, apiKey string, temperature *float64, timeoutSec int) error {
	if strings.TrimSpace(provider) != "" {
		p, err := ParseProvider(provider)
		if err != nil {
			return err
		}
		c.Provider = p
		c.markOneShot(fldProvider)
	}
	if baseURL != "" {
		c.BaseURL = strings.TrimSpace(baseURL)
		c.markOneShot(fldBaseURL)
	}
	if model != "" {
		c.Model = strings.TrimSpace(model)
		c.markOneShot(fldModel)
	}
	if apiKey != "" {
		c.APIKey = apiKey
		c.markOneShot(fldAPIKey)
	}
	if temperature != nil {
		v := *temperature
		c.Temperature = &v
		c.markOneShot(fldTemperature)
	}
	if timeoutSec >= 0 {
		c.TimeoutSec = timeoutSec
		c.markOneShot(fldTimeout)
	}
	return c.normalize()
}

// Save writes c to path with restrictive permissions.
func Save(c *Config, path string) error {
	if c == nil {
		return fmt.Errorf("nil config")
	}
	if err := c.normalize(); err != nil {
		return err
	}
	if path == "" {
		path = DefaultPath()
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	persisted := struct {
		Profiles      []Profile `json:"profiles"`
		ActiveProfile string    `json:"active_profile,omitempty"`
	}{
		Profiles:      c.Profiles,
		ActiveProfile: c.ActiveProfile,
	}
	b, err := json.MarshalIndent(persisted, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
