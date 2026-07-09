package ai

import "github.com/madicen/aitinerary/internal/aiconfig"

// effectiveTemperature returns the request override or profile value when set.
func effectiveTemperature(cfg *aiconfig.Config, req Request) (float64, bool) {
	if req.Temperature != nil {
		return *req.Temperature, true
	}
	if cfg != nil && cfg.Temperature != nil {
		return *cfg.Temperature, true
	}
	return 0, false
}

func applyTemperature(body map[string]any, cfg *aiconfig.Config, req Request) {
	if t, ok := effectiveTemperature(cfg, req); ok {
		body["temperature"] = t
	}
}

func applyTemperatureToGenCfg(genCfg map[string]any, cfg *aiconfig.Config, req Request) {
	if t, ok := effectiveTemperature(cfg, req); ok {
		genCfg["temperature"] = t
	}
}
