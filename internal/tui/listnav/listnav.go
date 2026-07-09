package listnav

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	zone "github.com/lrstanley/bubblezone"
	"github.com/madicen/aitinerary/internal/tui/longpress"
)

const wheelStep = 3

// Model holds shared list scroll offset and long-press arming state.
type Model struct {
	YOffset int

	LongPressItemIndex int
	LongPressPressID   int
	LongPressMouseX    int
	LongPressMouseY    int
}

// New returns a Model with long-press index reset to -1.
func New() Model {
	return Model{LongPressItemIndex: -1}
}

// WheelScroll adjusts YOffset for mouse-wheel events.
func (m *Model) WheelScroll(msg tea.MouseMsg) bool {
	isWheel := tea.MouseEvent(msg).IsWheel() ||
		msg.Button == tea.MouseButtonWheelUp ||
		msg.Button == tea.MouseButtonWheelDown
	if !isWheel {
		return false
	}
	isUp := msg.Button == tea.MouseButtonWheelUp || msg.Button == tea.MouseButtonWheelLeft
	if isUp {
		m.YOffset -= wheelStep
		if m.YOffset < 0 {
			m.YOffset = 0
		}
	} else {
		m.YOffset += wheelStep
	}
	return true
}

// LongPressConfig parameterises ArmLongPress for a specific tab.
type LongPressConfig struct {
	MenuOpen  bool
	ItemCount int
	RowZoneID func(i int) string
	Threshold time.Duration
	MakeTick  func(pressID int) tea.Msg
}

// ArmLongPress runs the shared press/motion/release arming state machine.
func (m *Model) ArmLongPress(zm *zone.Manager, msg tea.MouseMsg, cfg LongPressConfig) tea.Cmd {
	switch msg.Action {
	case tea.MouseActionMotion:
		if !cfg.MenuOpen && m.LongPressItemIndex >= 0 {
			origin := cfg.RowZoneID(m.LongPressItemIndex)
			if !longpress.StillArmed(zm, origin, m.LongPressMouseX, m.LongPressMouseY, msg) {
				m.LongPressItemIndex = -1
			}
		}

	case tea.MouseActionPress:
		if msg.Button != tea.MouseButtonLeft {
			return nil
		}
		if cfg.MenuOpen {
			return nil
		}
		for i := 0; i < cfg.ItemCount; i++ {
			z := zm.Get(cfg.RowZoneID(i))
			if z != nil && z.InBounds(msg) {
				m.LongPressPressID++
				m.LongPressItemIndex = i
				m.LongPressMouseX = msg.X
				m.LongPressMouseY = msg.Y
				pressID := m.LongPressPressID
				threshold := cfg.Threshold
				makeTick := cfg.MakeTick
				return tea.Tick(threshold, func(time.Time) tea.Msg {
					return makeTick(pressID)
				})
			}
		}

	case tea.MouseActionRelease:
		m.LongPressItemIndex = -1
	}
	return nil
}

// ScrollToSelected nudges YOffset so row selected is visible in listHeight rows.
func (m *Model) ScrollToSelected(selected, total, listHeight int) {
	if selected < 0 || selected >= total {
		return
	}
	if selected < m.YOffset {
		m.YOffset = selected
	} else if selected >= m.YOffset+listHeight {
		m.YOffset = selected - listHeight + 1
	}
}

// VisibleRange clamps YOffset and returns [start, end) visible bounds.
func (m *Model) VisibleRange(total, listHeight int) (start, end int) {
	maxOffset := 0
	if total > listHeight {
		maxOffset = total - listHeight
	}
	if m.YOffset > maxOffset {
		m.YOffset = maxOffset
	}
	if m.YOffset < 0 {
		m.YOffset = 0
	}
	start = m.YOffset
	end = start + listHeight
	if end > total {
		end = total
	}
	return start, end
}
