package longpress

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func motion(x, y int) tea.MouseMsg {
	return tea.MouseMsg{Action: tea.MouseActionMotion, Button: tea.MouseButtonLeft, X: x, Y: y}
}

func TestStillArmed_NilManager_WithinSlack(t *testing.T) {
	if !StillArmed(nil, "", 10, 5, motion(10+MotionSlack, 5+MotionSlack)) {
		t.Fatal("motion at edge of MotionSlack should keep armed")
	}
}

func TestStillArmed_NilManager_BeyondSlack(t *testing.T) {
	if StillArmed(nil, "", 10, 5, motion(10+MotionSlack+1, 5)) {
		t.Fatal("motion past MotionSlack on x must cancel")
	}
}

func TestStillArmed_EmptyZoneID(t *testing.T) {
	if StillArmed(nil, "", 0, 0, motion(100, 100)) {
		t.Fatal("empty origin zone must not bypass slack check")
	}
}
