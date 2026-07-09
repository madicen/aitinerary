package jira

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

// Transition is an available workflow transition for an issue.
type Transition struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	ToStatus string // resolved from the transition's target status name.
}

// ListTransitions returns the transitions currently available on an issue.
func (c *Client) ListTransitions(ctx context.Context, key string) ([]Transition, error) {
	var resp struct {
		Transitions []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
			To   struct {
				Name string `json:"name"`
			} `json:"to"`
		} `json:"transitions"`
	}
	if err := c.doJSON(ctx, http.MethodGet, "/rest/api/3/issue/"+key+"/transitions", nil, &resp); err != nil {
		return nil, err
	}
	out := make([]Transition, 0, len(resp.Transitions))
	for _, t := range resp.Transitions {
		out = append(out, Transition{ID: t.ID, Name: t.Name, ToStatus: t.To.Name})
	}
	return out, nil
}

// DoTransition executes a transition by ID. Jira returns 204 No Content on
// success (doJSON accepts any 2xx).
func (c *Client) DoTransition(ctx context.Context, key, transitionID string) error {
	body := map[string]any{"transition": map[string]string{"id": transitionID}}
	return c.doJSON(ctx, http.MethodPost, "/rest/api/3/issue/"+key+"/transitions", body, nil)
}

// FindTransitionTo returns the transition whose target status matches
// statusName (case-insensitive), preferring an exact target-status match over a
// transition merely named the same.
func FindTransitionTo(transitions []Transition, statusName string) (Transition, bool) {
	for _, t := range transitions {
		if strings.EqualFold(t.ToStatus, statusName) {
			return t, true
		}
	}
	for _, t := range transitions {
		if strings.EqualFold(t.Name, statusName) {
			return t, true
		}
	}
	return Transition{}, false
}

// TransitionTo moves an issue to the named status by looking up the live
// transition list first. It errors clearly when no transition reaches that
// status from the issue's current state.
func (c *Client) TransitionTo(ctx context.Context, key, statusName string) error {
	transitions, err := c.ListTransitions(ctx, key)
	if err != nil {
		return err
	}
	t, ok := FindTransitionTo(transitions, statusName)
	if !ok {
		return fmt.Errorf("no transition from %s reaches status %q (available: %s)", key, statusName, transitionNames(transitions))
	}
	return c.DoTransition(ctx, key, t.ID)
}

func transitionNames(ts []Transition) string {
	names := make([]string, 0, len(ts))
	for _, t := range ts {
		names = append(names, t.ToStatus)
	}
	return strings.Join(names, ", ")
}
