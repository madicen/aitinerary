package epics

import (
	"testing"

	"github.com/madicen/aitinerary/internal/config"
	"github.com/madicen/aitinerary/internal/store"
	"github.com/madicen/aitinerary/internal/tui/keys"
	"github.com/madicen/aitinerary/internal/tui/tuitest"
)

func TestGoldenEpicsTimeline(t *testing.T) {
	tuitest.ForceMonochrome(t)
	cfg := config.Default()
	st := store.New(cfg, nil, true)
	st.RefreshCmd()()
	m := New(Opts{Store: st, Keys: keys.Default()})
	m.Resize(100, 30)
	m.SetContentOrigin(2)
	m.SetNow(st.Tickets()[0].Updated) // demo anchor
	got := m.View()
	tuitest.AssertGolden(t, "epics_timeline", got)
}
