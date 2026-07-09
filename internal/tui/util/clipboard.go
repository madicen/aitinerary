package util

import (
	"io"
	"os"

	"github.com/atotto/clipboard"
	osc52 "github.com/aymanbagabas/go-osc52/v2"
	tea "github.com/charmbracelet/bubbletea"
)

// ClipboardCopiedMsg is delivered after a copy-to-clipboard attempt.
type ClipboardCopiedMsg struct {
	Success  bool
	ViaOSC52 bool
	Err      error
}

var (
	nativeClipboardWrite = clipboard.WriteAll
	osc52Writer          io.Writer = os.Stdout
)

func copyToClipboard(text string) (viaOSC52 bool, err error) {
	if nerr := nativeClipboardWrite(text); nerr == nil {
		return false, nil
	}
	if _, werr := osc52.New(text).WriteTo(osc52Writer); werr != nil {
		return true, werr
	}
	return true, nil
}

// CopyPlainTextCmd copies text to the clipboard and returns ClipboardCopiedMsg.
func CopyPlainTextCmd(text string) tea.Cmd {
	return func() tea.Msg {
		via, err := copyToClipboard(text)
		if err != nil {
			return ClipboardCopiedMsg{Success: false, ViaOSC52: via, Err: err}
		}
		return ClipboardCopiedMsg{Success: true, ViaOSC52: via}
	}
}
