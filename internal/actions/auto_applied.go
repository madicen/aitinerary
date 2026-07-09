package actions

import "sync"

// AutoApplied tracks recently auto-applied actions for the board strip.
type AutoApplied struct {
	mu      sync.RWMutex
	entries []JournalEntry
}

// NewAutoApplied returns an empty tracker.
func NewAutoApplied() *AutoApplied { return &AutoApplied{} }

// Record appends an auto-applied journal entry (most recent last).
func (a *AutoApplied) Record(entry JournalEntry) {
	if a == nil {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.entries = append(a.entries, entry)
	if len(a.entries) > 8 {
		a.entries = a.entries[len(a.entries)-8:]
	}
}

// Recent returns a copy of recent entries, newest last.
func (a *AutoApplied) Recent() []JournalEntry {
	if a == nil {
		return nil
	}
	a.mu.RLock()
	defer a.mu.RUnlock()
	out := make([]JournalEntry, len(a.entries))
	copy(out, a.entries)
	return out
}

// PopLast removes and returns the most recent entry.
func (a *AutoApplied) PopLast() (JournalEntry, bool) {
	if a == nil {
		return JournalEntry{}, false
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if len(a.entries) == 0 {
		return JournalEntry{}, false
	}
	e := a.entries[len(a.entries)-1]
	a.entries = a.entries[:len(a.entries)-1]
	return e, true
}

// Clear removes all entries.
func (a *AutoApplied) Clear() {
	if a == nil {
		return
	}
	a.mu.Lock()
	a.entries = nil
	a.mu.Unlock()
}
