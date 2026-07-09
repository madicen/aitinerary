package state

// ViewMode identifies a full-screen tab the root TUI can show.
type ViewMode int

const (
	ModeBoard ViewMode = iota
	ModeQueue
	ModeEpics
	ModeSettings
	ModeHelp
)

func (v ViewMode) String() string {
	switch v {
	case ModeBoard:
		return "board"
	case ModeQueue:
		return "queue"
	case ModeEpics:
		return "epics"
	case ModeSettings:
		return "settings"
	case ModeHelp:
		return "help"
	default:
		return "unknown"
	}
}
