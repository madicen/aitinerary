// Package util holds cross-cutting helpers shared by the root TUI, tabs, and
// overlays. Nothing here imports back into tui or its tabs.
package util

import (
	"io"
	"os"
	"runtime"

	"github.com/charmbracelet/x/ansi"
)

var mouseTrackingOffSeq = ansi.ResetModeMouseX10 +
	ansi.ResetModeMouseNormal +
	ansi.ResetModeMouseHighlight +
	ansi.ResetModeMouseButtonEvent +
	ansi.ResetModeMouseAnyEvent +
	ansi.ResetModeMouseExtUtf8 +
	ansi.ResetModeMouseExtSgr +
	ansi.ResetModeMouseExtUrxvt +
	ansi.ResetModeMouseExtSgrPixel +
	"\x1b[0m"

func writeMouseOffAndSync(w io.Writer) {
	if w == nil {
		return
	}
	_, _ = io.WriteString(w, mouseTrackingOffSeq)
	if f, ok := w.(*os.File); ok {
		_ = f.Sync()
	}
}

// FlushMouse disables xterm mouse reporting on stdout, stderr, and /dev/tty
// (Unix). Called on quit so the terminal is not left in mouse-tracking mode.
func FlushMouse() {
	writeMouseOffAndSync(os.Stdout)
	writeMouseOffAndSync(os.Stderr)
	if runtime.GOOS == "windows" {
		return
	}
	f, err := os.OpenFile("/dev/tty", os.O_WRONLY, 0)
	if err != nil {
		return
	}
	_, _ = io.WriteString(f, mouseTrackingOffSeq)
	_ = f.Sync()
	_ = f.Close()
}

// Clamp constrains v to [lo, hi].
func Clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
