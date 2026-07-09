package help

import (
	"os"
	"path/filepath"

	"github.com/madicen/aitinerary/internal/appdirs"
)

const setupMarker = "setup_wizard_done"

// MarkerPath returns the path to the first-run completion marker.
func MarkerPath() string {
	return filepath.Join(appdirs.ConfigDir(), setupMarker)
}

// WizardComplete reports whether the setup wizard has been dismissed before.
func WizardComplete() bool {
	_, err := os.Stat(MarkerPath())
	return err == nil
}

// MarkWizardComplete writes the setup wizard completion marker.
func MarkWizardComplete() error {
	path := MarkerPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte("1\n"), 0o600)
}

// ShouldShowWizard reports whether the first-run wizard should appear.
func ShouldShowWizard(demo bool) bool {
	if demo {
		return false
	}
	return !WizardComplete()
}
