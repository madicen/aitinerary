package data

import (
	"testing"

	"github.com/madicen/aitinerary/internal/actions"
	"github.com/madicen/aitinerary/internal/config"
	"github.com/madicen/aitinerary/internal/demo"
	"github.com/madicen/aitinerary/internal/store"
)

func TestRefreshAllCmdDemoMode(t *testing.T) {
	cfg := config.Default()
	st := store.New(cfg, nil, true)
	cmd := RefreshAllCmd(cfg, true, st, nil, actions.NewEngine(nil, nil), nil)
	msg, ok := cmd().(RefreshAllMsg)
	if !ok {
		t.Fatalf("expected RefreshAllMsg, got %T", cmd())
	}
	if !msg.KanbanProjects["HELP"] {
		t.Fatalf("demo refresh should mark HELP kanban: %+v", msg.KanbanProjects)
	}
	if len(msg.MergedPRs) == 0 {
		t.Fatal("demo refresh should return merged PRs")
	}
	wantPRs := demo.MergedPRs()
	if len(msg.MergedPRs) != len(wantPRs) {
		t.Fatalf("merged PR count = %d, want %d", len(msg.MergedPRs), len(wantPRs))
	}
	if msg.MergedPRs[0].Number != wantPRs[0].Number {
		t.Fatalf("first merged PR = #%d, want #%d", msg.MergedPRs[0].Number, wantPRs[0].Number)
	}
}
