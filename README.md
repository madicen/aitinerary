# aitinerary

A BubbleTea TUI for Jira ticket hygiene — the after-the-fact ticket manager.

Creating tickets is easy; keeping the board honest afterwards is the chore.
aitinerary finds tickets that should already be Done (by scanning merged GitHub
PRs for ticket keys), flags hygiene problems (missing epic, sprint, due date,
stale In Progress), builds retroactive epics with an AI assist, and runs a
one-command sync-prep sweep before weekly syncs.

## Status

Early development. See [PLAN.md](./PLAN.md) for the phased build plan.

## Highlights

- **Done detection** — scan merged PRs for ticket keys; auto-transition the
  unambiguous ones, queue the rest for review.
- **Hygiene detectors** — flag tickets missing epic / sprint / due date, or
  stale In Progress.
- **Retroactive epic builder** — multi-select tickets, AI drafts an epic, set it
  as parent with start/due dates so it lands correctly on the Jira timeline.
- **Sync-prep mode** — AI drafts a concise status comment per active ticket,
  proposes transitions, and produces a digest for your TL/manager.
- **AI profiles** — local-first (Ollama) by default, switchable per-action.

Nothing AI-authored reaches Jira unreviewed; every applied action is journaled
and undoable.

## Configuration

Config lives under `$AITINERARY_CONFIG_DIR` → `$XDG_CONFIG_HOME/aitinerary` →
`~/.config/aitinerary`. Secrets are never stored in config — the Jira token is
resolved from `JIRA_TOKEN` (or a `jira_token_cmd`), and AI provider keys via
`api_key_env` / `api_key_cmd`.

## Build

```sh
make build     # -> ./aitinerary
make test      # unit tests
make lint      # golangci-lint (must be installed)
./aitinerary --version
```

Requires Go 1.25+.

## License

MIT — see [LICENSE](./LICENSE).
