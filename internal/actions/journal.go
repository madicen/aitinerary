package actions

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// PriorState captures the values an action overwrote, enough to undo it.
type PriorState struct {
	Status    string   `json:"status,omitempty"`     // status name before a transition.
	DueDate   string   `json:"due_date,omitempty"`   // ISO date; "" means it was unset.
	StartDate string   `json:"start_date,omitempty"` // "" means it was unset.
	ParentKey string   `json:"parent_key,omitempty"` // "" means it was unparented.
	Labels    []string `json:"labels,omitempty"`     // full prior label set.
	SprintID  int      `json:"sprint_id,omitempty"`  // prior sprint id (0 = none).
}

// JournalEntry is one applied action plus the state it replaced, appended to
// journal.jsonl so it can be inspected and undone.
type JournalEntry struct {
	Time   time.Time  `json:"time"`
	Action Action     `json:"action"`
	Prior  PriorState `json:"prior"`
	Result string     `json:"result,omitempty"` // e.g. created epic key.
	Undone bool       `json:"undone,omitempty"`
}

// Journal is an append-only JSONL log of applied actions.
type Journal struct {
	path string
	mu   sync.Mutex
}

// NewJournal returns a Journal backed by path (created lazily on first append).
func NewJournal(path string) *Journal { return &Journal{path: path} }

// Append writes one entry as a JSON line, creating the file/dir if needed.
func (j *Journal) Append(e JournalEntry) error {
	j.mu.Lock()
	defer j.mu.Unlock()
	if err := os.MkdirAll(filepath.Dir(j.path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(j.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	b, err := json.Marshal(e)
	if err != nil {
		return err
	}
	if _, err := f.Write(append(b, '\n')); err != nil {
		return err
	}
	return nil
}

// Load reads every entry in order. A missing file yields an empty slice.
func (j *Journal) Load() ([]JournalEntry, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	f, err := os.Open(j.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()
	var out []JournalEntry
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1<<20)
	for sc.Scan() {
		line := sc.Bytes()
		if len(line) == 0 {
			continue
		}
		var e JournalEntry
		if err := json.Unmarshal(line, &e); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, sc.Err()
}
