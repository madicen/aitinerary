package prompts

import (
	"fmt"
	"strings"

	"github.com/madicen/aitinerary/internal/llmjson"
)

// PRRef is a merged PR referenced by a status update.
type PRRef struct {
	Repo   string
	Number int
	Title  string
}

// StatusUpdateInput is the input to the status_update prompt.
type StatusUpdateInput struct {
	Key         string
	Summary     string
	Status      string
	Description string
	PRs         []PRRef
	Diffstat    string
}

// StatusUpdateOutput is the parsed status_update result.
type StatusUpdateOutput struct {
	Comment string `json:"comment"`
}

// EpicTicket is one ticket summarised for the epic_draft prompt.
type EpicTicket struct {
	Key     string
	Summary string
	DueDate string
}

// EpicDraftInput is the input to the epic_draft prompt.
type EpicDraftInput struct {
	Tickets []EpicTicket
}

// EpicDraftOutput is the parsed epic_draft result. Body is a bullet block
// (each line "- ..."); Bullets splits it for the epic builder form.
type EpicDraftOutput struct {
	Title     string `json:"title"`
	Body      string `json:"body"`
	StartDate string `json:"start_date"`
	DueDate   string `json:"due_date"`
}

// DigestInput is the input to the ticket_summary_digest prompt.
type DigestInput struct {
	Activity string
}

// ParseStatusUpdate salvages and validates a status_update model response.
func ParseStatusUpdate(raw string) (StatusUpdateOutput, error) {
	out, err := llmjson.Parse[StatusUpdateOutput](raw)
	if err != nil {
		return StatusUpdateOutput{}, err
	}
	out.Comment = strings.TrimSpace(out.Comment)
	if out.Comment == "" {
		return StatusUpdateOutput{}, fmt.Errorf("status_update produced an empty comment")
	}
	return out, nil
}

// ParseEpicDraft salvages and validates an epic_draft model response.
func ParseEpicDraft(raw string) (EpicDraftOutput, error) {
	out, err := llmjson.Parse[EpicDraftOutput](raw)
	if err != nil {
		return EpicDraftOutput{}, err
	}
	out.Title = strings.TrimSpace(out.Title)
	out.Body = strings.TrimSpace(out.Body)
	if out.Title == "" {
		return EpicDraftOutput{}, fmt.Errorf("epic_draft produced an empty title")
	}
	return out, nil
}

// Bullets splits an epic body into individual bullet strings, stripping a
// leading "- " / "* " marker from each non-empty line. Used to pre-fill the
// epic builder form and to build the ADF bullet list.
func (o EpicDraftOutput) Bullets() []string {
	var out []string
	for _, line := range strings.Split(o.Body, "\n") {
		line = strings.TrimSpace(line)
		line = strings.TrimPrefix(line, "- ")
		line = strings.TrimPrefix(line, "* ")
		if s := strings.TrimSpace(line); s != "" {
			out = append(out, s)
		}
	}
	return out
}
