package longpress

import (
	tea "github.com/charmbracelet/bubbletea"
	zone "github.com/lrstanley/bubblezone"
)

// MotionSlack is the max Chebyshev distance the cursor may drift while armed.
const MotionSlack = 2

// StillArmed reports whether a long-press should remain armed given motion msg.
func StillArmed(zm *zone.Manager, originZone string, anchorX, anchorY int, msg tea.MouseMsg) bool {
	if zm != nil && originZone != "" {
		if z := zm.Get(originZone); z != nil && z.InBounds(msg) {
			return true
		}
	}
	dx := msg.X - anchorX
	if dx < 0 {
		dx = -dx
	}
	dy := msg.Y - anchorY
	if dy < 0 {
		dy = -dy
	}
	return dx <= MotionSlack && dy <= MotionSlack
}
