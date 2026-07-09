// Package ai is the leaf provider layer for aitinerary inference.
//
// It owns HTTP provider transport (Ollama, OpenAI-compatible, Gemini,
// Anthropic), retry/backoff, usage capture, and inference-call logging.
package ai

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/madicen/aitinerary/internal/aiconfig"
)

// Request is one inference request.
type Request struct {
	System, User string
	WantJSON     bool
	JSONSchema   json.RawMessage
	Stage        string
	Stream       bool
	// Temperature overrides the profile temperature when set. Omitted from the
	// provider request body when both this and the profile temperature are unset.
	Temperature *float64
}

// Usage captures token counts and cost when the provider reports them.
type Usage struct {
	InputTokens, OutputTokens int
	CostUSD                   float64
}

// Result is one completed inference.
type Result struct {
	Text  string
	Usage Usage
	Model string
}

// Capabilities describes what a provider backend can do.
type Capabilities struct {
	NativeJSON, Streaming bool
}

// Provider is one inference backend.
type Provider interface {
	Complete(ctx context.Context, req Request) (Result, error)
	Capabilities() Capabilities
	Name() string
}

// CompleteFunc is the shared inference-function type for callers.
type CompleteFunc func(ctx context.Context, cfg *aiconfig.Config, system, user string) (string, error)

var testBaseProviderHook func(cfg *aiconfig.Config) (Provider, error)

// SetBaseProviderForTest installs a base-provider factory for tests.
func SetBaseProviderForTest(fn func(cfg *aiconfig.Config) (Provider, error)) func() {
	prev := testBaseProviderHook
	testBaseProviderHook = fn
	return func() { testBaseProviderHook = prev }
}

func baseProviderFor(cfg *aiconfig.Config) (Provider, error) {
	if cfg == nil {
		cfg = aiconfig.DefaultConfig()
	}
	if testBaseProviderHook != nil {
		return testBaseProviderHook(cfg)
	}
	switch cfg.Provider {
	case aiconfig.ProviderOllama, aiconfig.ProviderOpenAICompatible:
		return &openAIProvider{cfg: cfg}, nil
	case aiconfig.ProviderGemini:
		return &geminiProvider{cfg: cfg}, nil
	case aiconfig.ProviderAnthropic:
		return &anthropicProvider{cfg: cfg}, nil
	default:
		return nil, fmt.Errorf("unsupported AI provider %q", cfg.Provider)
	}
}

// ProviderFor returns the Provider for cfg wrapped with retry/backoff and logging.
func ProviderFor(cfg *aiconfig.Config) (Provider, error) {
	if cfg == nil {
		cfg = aiconfig.DefaultConfig()
	}
	base, err := baseProviderFor(cfg)
	if err != nil {
		return nil, err
	}
	return &retryProvider{cfg: cfg, inner: base}, nil
}

// CapabilitiesFor reports the capabilities of cfg's provider.
func CapabilitiesFor(cfg *aiconfig.Config) Capabilities {
	base, err := baseProviderFor(cfg)
	if err != nil {
		return Capabilities{}
	}
	return base.Capabilities()
}
