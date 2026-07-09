package aiconfig

import (
	"strings"
	"time"
)

// EffectiveTimeoutSec returns a positive timeout in seconds.
func (c *Config) EffectiveTimeoutSec() int {
	if c == nil || c.TimeoutSec <= 0 {
		return 300
	}
	return c.TimeoutSec
}

// StreamIdleTimeout is the idle timeout for streaming inference.
func (c *Config) StreamIdleTimeout() time.Duration {
	return time.Duration(c.EffectiveTimeoutSec()) * time.Second
}

// StreamFirstByteTimeout waits for the first response byte when streaming.
func (c *Config) StreamFirstByteTimeout() time.Duration {
	idle := c.StreamIdleTimeout()
	fb := 60 * time.Second
	if fb > idle {
		fb = idle
	}
	return fb
}

// AIBaseURLResolved returns the HTTP base URL for the active provider.
func (c *Config) AIBaseURLResolved() string {
	if c == nil {
		return ""
	}
	switch c.Provider {
	case ProviderOllama:
		if strings.TrimSpace(c.BaseURL) == "" {
			return DefaultOllamaBaseURL
		}
	case ProviderGemini:
		if strings.TrimSpace(c.BaseURL) == "" {
			return "https://generativelanguage.googleapis.com"
		}
	case ProviderAnthropic:
		if strings.TrimSpace(c.BaseURL) == "" {
			return "https://api.anthropic.com"
		}
	}
	return strings.TrimRight(strings.TrimSpace(c.BaseURL), "/")
}

// AIModelOrDefault picks a model id.
func (c *Config) AIModelOrDefault() string {
	if c == nil {
		return DefaultOllamaModel
	}
	if strings.TrimSpace(c.Model) != "" {
		return strings.TrimSpace(c.Model)
	}
	if c.Provider == ProviderOllama {
		return DefaultOllamaModel
	}
	return ""
}

// InferenceRetryMaxAttempts returns how many times each Complete call may try.
func (c *Config) InferenceRetryMaxAttempts() int {
	if c == nil {
		return 5
	}
	n := c.RetryMaxAttempts
	if n <= 0 {
		return 5
	}
	if n > 30 {
		return 30
	}
	return n
}

func (c *Config) InferenceRetryBase() time.Duration {
	ms := 1500
	if c != nil && c.RetryBaseMS > 0 {
		ms = c.RetryBaseMS
	}
	return time.Duration(ms) * time.Millisecond
}

func (c *Config) InferenceRetryMaxBackoff() time.Duration {
	ms := 120000
	if c != nil && c.RetryMaxMS > 0 {
		ms = c.RetryMaxMS
	}
	return time.Duration(ms) * time.Millisecond
}
