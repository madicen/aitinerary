package mousedouble

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// DefaultDoubleClickWindow is the max delay between two left releases.
const DefaultDoubleClickWindow = 450 * time.Millisecond

// IsLeftRelease reports whether e is a left-button release.
func IsLeftRelease(e tea.MouseMsg) bool {
	return e.Button == tea.MouseButtonLeft && e.Action == tea.MouseActionRelease
}

// OnLeftPress increments the press generation for release dedupe.
func OnLeftPress(pressGen *uint64) {
	*pressGen++
}

// OverlapRelease tracks one zone-handling pass per physical left release.
type OverlapRelease struct {
	gen    uint64
	x, y   int
	active bool
}

// ShouldSkipOverlappingRelease returns true for duplicate zone hits on one release.
func (o *OverlapRelease) ShouldSkipOverlappingRelease(e tea.MouseMsg, pressGen uint64) bool {
	if !IsLeftRelease(e) {
		return false
	}
	if o.active && o.gen == pressGen && o.x == e.X && o.y == e.Y {
		return true
	}
	o.active = true
	o.gen = pressGen
	o.x = e.X
	o.y = e.Y
	return false
}

// DoubleClick tracks the previous left release per logical key.
type DoubleClick struct {
	lastKey  string
	lastTime time.Time
}

// ObserveLeftRelease returns true when this release is a double-click on key.
func (d *DoubleClick) ObserveLeftRelease(key string, e tea.MouseMsg, now time.Time, window time.Duration) bool {
	if !IsLeftRelease(e) {
		return false
	}
	if key == "" {
		return false
	}
	if d.lastKey == key && !d.lastTime.IsZero() && now.Sub(d.lastTime) <= window {
		d.lastKey = ""
		d.lastTime = time.Time{}
		return true
	}
	d.lastKey = key
	d.lastTime = now
	return false
}
