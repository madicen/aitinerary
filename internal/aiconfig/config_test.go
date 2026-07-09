package aiconfig

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func withConfigDir(t *testing.T, fn func(dir string)) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("AITINERARY_CONFIG_DIR", dir)
	for _, k := range []string{
		"AITINERARY_AI_PROVIDER",
		"AITINERARY_AI_BASE_URL",
		"AITINERARY_AI_MODEL",
		"AITINERARY_AI_API_KEY",
		"AITINERARY_AI_TEMPERATURE",
		"AITINERARY_AI_TIMEOUT_SEC",
		"AITINERARY_AI_RETRY_MAX_ATTEMPTS",
		"AITINERARY_AI_RETRY_BASE_MS",
		"AITINERARY_AI_RETRY_MAX_MS",
	} {
		t.Setenv(k, "")
	}
	fn(dir)
}

func TestDefaultConfigIsOllamaLocal(t *testing.T) {
	c := DefaultConfig()
	if c.Provider != ProviderOllama {
		t.Fatalf("provider: got %q, want ollama", c.Provider)
	}
	if c.BaseURL != DefaultOllamaBaseURL {
		t.Fatalf("base_url: got %q, want %q", c.BaseURL, DefaultOllamaBaseURL)
	}
	if c.Model != DefaultOllamaModel {
		t.Fatalf("model: got %q, want %q", c.Model, DefaultOllamaModel)
	}
	if len(c.Profiles) != 1 || c.ActiveProfile != DefaultProfileName {
		t.Fatalf("expected one default profile, got %+v", c.Profiles)
	}
}

func TestLoadLegacyFlatShapeMigratesToProfile(t *testing.T) {
	withConfigDir(t, func(dir string) {
		legacy := []byte(`{
  "provider": "openai_compatible",
  "base_url": "http://localhost:1234",
  "model": "qwen-coder",
  "api_key": "sk-test",
  "timeout_sec": 120
}`)
		if err := os.WriteFile(filepath.Join(dir, "ai.json"), legacy, 0o600); err != nil {
			t.Fatalf("write legacy: %v", err)
		}
		c, err := Load()
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		if len(c.Profiles) != 1 || c.Profiles[0].Provider != ProviderOpenAICompatible {
			t.Fatalf("migration failed: %+v", c.Profiles)
		}
		if c.Provider != ProviderOpenAICompatible || c.Model != "qwen-coder" {
			t.Fatalf("top-level not mirrored: %+v", *c)
		}
	})
}

func TestSaveAndLoadRoundTripPreservesMultipleProfiles(t *testing.T) {
	withConfigDir(t, func(dir string) {
		c := DefaultConfig()
		temp := 0.2
		c.Profiles[0].Temperature = &temp
		if err := c.AddProfile(Profile{
			Name:     "cloud",
			Provider: ProviderOpenAICompatible,
			BaseURL:  "https://openrouter.ai/api/v1",
			Model:    "anthropic/claude-3.5-sonnet",
			APIKey:   "sk-or-foo",
		}); err != nil {
			t.Fatalf("AddProfile: %v", err)
		}
		if err := c.SetActive("cloud"); err != nil {
			t.Fatalf("SetActive: %v", err)
		}
		if err := Save(c, ""); err != nil {
			t.Fatalf("Save: %v", err)
		}

		raw, err := os.ReadFile(filepath.Join(dir, "ai.json"))
		if err != nil {
			t.Fatalf("read saved file: %v", err)
		}
		var onDisk map[string]any
		if err := json.Unmarshal(raw, &onDisk); err != nil {
			t.Fatalf("unmarshal saved file: %v", err)
		}
		if _, has := onDisk["provider"]; has {
			t.Fatalf("saved file should not contain top-level provider")
		}

		c2, err := Load()
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		if len(c2.Profiles) != 2 || c2.ActiveProfile != "cloud" {
			t.Fatalf("round-trip profiles: %+v active=%q", c2.Profiles, c2.ActiveProfile)
		}
		def, _ := c2.findProfileIndex(DefaultProfileName)
		if c2.Profiles[def].Temperature == nil || *c2.Profiles[def].Temperature != 0.2 {
			t.Fatalf("temperature not preserved on default profile")
		}
	})
}

func TestEnvOverrideTemperature(t *testing.T) {
	withConfigDir(t, func(_ string) {
		t.Setenv("AITINERARY_AI_TEMPERATURE", "0.7")
		c, err := Load()
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		if c.Temperature == nil || *c.Temperature != 0.7 {
			t.Fatalf("temperature override: got %v", c.Temperature)
		}
	})
}

func TestSetActiveSwitchesAndMirrors(t *testing.T) {
	c := DefaultConfig()
	if err := c.AddProfile(Profile{Name: "fast", Provider: ProviderOllama, Model: "phi3"}); err != nil {
		t.Fatalf("AddProfile: %v", err)
	}
	if err := c.SetActive("fast"); err != nil {
		t.Fatalf("SetActive: %v", err)
	}
	if c.Model != "phi3" {
		t.Fatalf("top-level not mirrored: model=%q", c.Model)
	}
}

func TestDeleteProfileBlockedOnLast(t *testing.T) {
	c := DefaultConfig()
	if err := c.DeleteProfile(DefaultProfileName); err == nil {
		t.Fatal("expected error deleting last profile")
	}
}

func TestCloneIsDeep(t *testing.T) {
	c := DefaultConfig()
	temp := 0.5
	_ = c.AddProfile(Profile{Name: "x", Provider: ProviderOllama, Model: "first", Temperature: &temp})
	cp := c.Clone()
	cp.Profiles[1].Model = "mutated"
	if c.Profiles[1].Model == "mutated" {
		t.Fatal("Clone is not a deep copy of Profiles slice")
	}
	v := 0.9
	cp.Profiles[1].Temperature = &v
	if c.Profiles[1].Temperature != nil && *c.Profiles[1].Temperature == 0.9 {
		t.Fatal("Clone should not share Temperature pointer")
	}
}
