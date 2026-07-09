package ai

import (
	"context"
	"testing"

	"github.com/madicen/aitinerary/internal/aiconfig"
)

const anthropicReplyBody = `{"content":[{"type":"text","text":"ok"}],"usage":{"input_tokens":1,"output_tokens":1}}`

func floatPtr(v float64) *float64 { return &v }

func TestOpenAITemperatureOmissionAndInclusion(t *testing.T) {
	t.Run("omitWhenUnset", func(t *testing.T) {
		srv, got := captureBody(t, openAIReply)
		cfg := &aiconfig.Config{Provider: aiconfig.ProviderOllama, BaseURL: srv.URL, Model: "qwen"}
		if _, err := (&openAIProvider{cfg: cfg}).Complete(context.Background(), Request{System: "s", User: "u"}); err != nil {
			t.Fatalf("Complete: %v", err)
		}
		if _, ok := (*got)["temperature"]; ok {
			t.Fatalf("plain request must omit temperature, got %#v", (*got)["temperature"])
		}
	})

	t.Run("profileTemperature", func(t *testing.T) {
		srv, got := captureBody(t, openAIReply)
		cfg := &aiconfig.Config{Provider: aiconfig.ProviderOllama, BaseURL: srv.URL, Model: "qwen", Temperature: floatPtr(0.3)}
		if _, err := (&openAIProvider{cfg: cfg}).Complete(context.Background(), Request{System: "s", User: "u"}); err != nil {
			t.Fatalf("Complete: %v", err)
		}
		if (*got)["temperature"] != 0.3 {
			t.Fatalf("temperature = %v, want 0.3", (*got)["temperature"])
		}
	})

	t.Run("requestOverridesProfile", func(t *testing.T) {
		srv, got := captureBody(t, openAIReply)
		cfg := &aiconfig.Config{Provider: aiconfig.ProviderOllama, BaseURL: srv.URL, Model: "qwen", Temperature: floatPtr(0.3)}
		req := Request{System: "s", User: "u", Temperature: floatPtr(0.9)}
		if _, err := (&openAIProvider{cfg: cfg}).Complete(context.Background(), req); err != nil {
			t.Fatalf("Complete: %v", err)
		}
		if (*got)["temperature"] != 0.9 {
			t.Fatalf("temperature = %v, want 0.9", (*got)["temperature"])
		}
	})
}

func TestGeminiTemperatureOmissionAndInclusion(t *testing.T) {
	t.Run("omitWhenUnset", func(t *testing.T) {
		srv, got := captureBody(t, geminiReply)
		cfg := &aiconfig.Config{Provider: aiconfig.ProviderGemini, BaseURL: srv.URL, Model: "gemini-2.0-flash", APIKey: "k"}
		if _, err := (&geminiProvider{cfg: cfg}).Complete(context.Background(), Request{System: "s", User: "u"}); err != nil {
			t.Fatalf("Complete: %v", err)
		}
		if _, ok := (*got)["generationConfig"]; ok {
			t.Fatalf("plain request must omit generationConfig, got %#v", (*got)["generationConfig"])
		}
	})

	t.Run("profileTemperature", func(t *testing.T) {
		srv, got := captureBody(t, geminiReply)
		cfg := &aiconfig.Config{Provider: aiconfig.ProviderGemini, BaseURL: srv.URL, Model: "gemini-2.0-flash", APIKey: "k", Temperature: floatPtr(0.4)}
		if _, err := (&geminiProvider{cfg: cfg}).Complete(context.Background(), Request{System: "s", User: "u"}); err != nil {
			t.Fatalf("Complete: %v", err)
		}
		gc, ok := (*got)["generationConfig"].(map[string]any)
		if !ok || gc["temperature"] != 0.4 {
			t.Fatalf("generationConfig.temperature = %#v, want 0.4", (*got)["generationConfig"])
		}
	})
}

func TestAnthropicTemperatureOmissionAndInclusion(t *testing.T) {
	t.Run("omitWhenUnset", func(t *testing.T) {
		srv, got := captureBody(t, anthropicReplyBody)
		cfg := &aiconfig.Config{Provider: aiconfig.ProviderAnthropic, BaseURL: srv.URL, Model: "claude-x", APIKey: "k"}
		if _, err := (&anthropicProvider{cfg: cfg}).Complete(context.Background(), Request{System: "s", User: "u"}); err != nil {
			t.Fatalf("Complete: %v", err)
		}
		if _, ok := (*got)["temperature"]; ok {
			t.Fatalf("plain request must omit temperature, got %#v", (*got)["temperature"])
		}
	})

	t.Run("requestTemperature", func(t *testing.T) {
		srv, got := captureBody(t, anthropicReplyBody)
		cfg := &aiconfig.Config{Provider: aiconfig.ProviderAnthropic, BaseURL: srv.URL, Model: "claude-x", APIKey: "k"}
		req := Request{System: "s", User: "u", Temperature: floatPtr(0.15)}
		if _, err := (&anthropicProvider{cfg: cfg}).Complete(context.Background(), req); err != nil {
			t.Fatalf("Complete: %v", err)
		}
		if (*got)["temperature"] != 0.15 {
			t.Fatalf("temperature = %v, want 0.15", (*got)["temperature"])
		}
	})
}
