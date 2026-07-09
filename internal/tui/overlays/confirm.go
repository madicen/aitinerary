package overlays

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	zone "github.com/lrstanley/bubblezone"

	"github.com/madicen/aitinerary/internal/tui/mouse"
	"github.com/madicen/aitinerary/internal/tui/styles"
)

// ConfirmOverlay asks the user to confirm an action.
type ConfirmOverlay struct {
	Message string
}

// NewConfirmOverlay constructs a yes/no confirm modal.
func NewConfirmOverlay(message string) ConfirmOverlay {
	return ConfirmOverlay{Message: message}
}

func (m ConfirmOverlay) Init() tea.Cmd { return nil }

func (m ConfirmOverlay) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, confirmYesKeys):
			return m, dismiss(ConfirmAnswer{Confirm: true})
		case key.Matches(msg, confirmNoKeys):
			return m, dismiss(ConfirmAnswer{Confirm: false})
		case msg.String() == "ctrl+c":
			return m, dismiss(ConfirmAnswer{Confirm: false})
		}
	case tea.MouseMsg:
		if msg.Action != tea.MouseActionPress || msg.Button != tea.MouseButtonLeft {
			return m, nil
		}
		if z := zone.Get(mouse.ConfirmYes); z != nil && z.InBounds(msg) {
			return m, dismiss(ConfirmAnswer{Confirm: true})
		}
		if z := zone.Get(mouse.ConfirmNo); z != nil && z.InBounds(msg) {
			return m, dismiss(ConfirmAnswer{Confirm: false})
		}
	}
	return m, nil
}

func (m ConfirmOverlay) View() string {
	body := m.Message + "\n\n" +
		zone.Mark(mouse.ConfirmYes, styles.OkStyle.Render(" Yes ")) + "  " +
		zone.Mark(mouse.ConfirmNo, styles.ErrStyle.Render(" No "))
	return ModalFrameSized(56).Render(body)
}
