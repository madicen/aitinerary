package aiconfig

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/madicen/aitinerary/internal/applog"
)

// EffectiveAPIKey resolves secret indirection: api_key > api_key_env > api_key_cmd.
func (c *Config) EffectiveAPIKey() string {
	if c == nil {
		return ""
	}
	if strings.TrimSpace(c.APIKey) != "" {
		return c.APIKey
	}
	if env := strings.TrimSpace(c.APIKeyEnv); env != "" {
		if v := strings.TrimSpace(os.Getenv(env)); v != "" {
			return v
		}
	}
	if cmd := strings.TrimSpace(c.APIKeyCmd); cmd != "" {
		return resolveKeyCmd(cmd)
	}
	return ""
}

var (
	keyCmdMu    sync.Mutex
	keyCmdCache = map[string]string{}
)

func resolveKeyCmd(cmd string) string {
	keyCmdMu.Lock()
	if v, ok := keyCmdCache[cmd]; ok {
		keyCmdMu.Unlock()
		return v
	}
	keyCmdMu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "sh", "-c", cmd).Output()
	if err != nil {
		applog.Warn("api_key_cmd failed; treating API key as empty", "err", err.Error())
		return ""
	}
	key := strings.TrimSpace(string(out))
	if key == "" {
		applog.Warn("api_key_cmd produced empty output; treating API key as empty")
		return ""
	}
	keyCmdMu.Lock()
	keyCmdCache[cmd] = key
	keyCmdMu.Unlock()
	return key
}

func maskKey(s string) string {
	if strings.TrimSpace(s) == "" {
		return ""
	}
	return "****"
}

// Redacted returns a deep copy with API keys masked for logging/display.
func (c *Config) Redacted() *Config {
	if c == nil {
		return nil
	}
	cp := c.Clone()
	cp.APIKey = maskKey(cp.APIKey)
	for i := range cp.Profiles {
		cp.Profiles[i].APIKey = maskKey(cp.Profiles[i].APIKey)
	}
	cp.oneShot = nil
	return cp
}

// RedactedJSON returns indented JSON with API keys masked.
func (c *Config) RedactedJSON() string {
	r := c.Redacted()
	if r == nil {
		return "null"
	}
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return fmt.Sprintf("<config marshal error: %v>", err)
	}
	return string(b)
}

// BearerForHTTP returns the Authorization bearer token for OpenAI-compatible APIs.
func (c *Config) BearerForHTTP() string {
	key := strings.TrimSpace(c.EffectiveAPIKey())
	if key != "" {
		return key
	}
	if c != nil && (c.Provider == ProviderOllama || c.Provider == ProviderOpenAICompatible) {
		return "ollama"
	}
	return ""
}
