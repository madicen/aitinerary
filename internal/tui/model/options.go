package model

import (
	"github.com/madicen/aitinerary/internal/aiconfig"
	"github.com/madicen/aitinerary/internal/config"
)

// Options configures the root TUI model.
type Options struct {
	Config   *config.Config
	Demo     bool
	AIConfig *aiconfig.Config
}
