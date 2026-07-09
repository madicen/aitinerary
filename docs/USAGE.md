# Usage

aitinerary is a keyboard- and mouse-driven TUI. Every action has a key binding
and a clickable zone; shortcuts are listed on the **Help** tab (rendered from
`keys.Map`) and via `?` / the status bar.

## Global keys

| Key | Action |
|-----|--------|
| `1` / `b` | Board tab |
| `2` / `q` | Queue tab |
| `3` / `e` | Epics tab |
| `4` / `,` | Settings tab |
| `5` | Help tab |
| `?` | Help overlay (shortcuts) |
| `ctrl+k` | Command palette |
| `ctrl+r` | Refresh ticket data |
| `ctrl+z` | Undo last applied action |
| `esc` | Back / dismiss overlay |
| `ctrl+c` | Quit |

On first launch (or in `--demo` mode), a **setup wizard** checks Jira URL/email,
token env, board discovery, `gh auth`, and Ollama reachability. Re-run it from
Help with `w`.

## Board tab

Default view shows all tickets with hygiene badges. Filter chips (click or key):

- `0` — all tickets
- `n` — needs attention (grouped by finding type)
- `m` — done candidates (merged PR, not Done)
- Workspace chips — APP / IRE
- Status chips — To Do / In flight / Done

List navigation: `j`/`k` or mouse wheel. `/` opens search. `enter` opens the
detail pane with transitions, sprint, due date, parent epic, labels.

**Sync-prep:** `S` runs refresh → PR match → hygiene scan → stub AI status
comments into the review queue.

**Epic builder:** `space` toggles multi-select on the current row, `v` range-
selects from the anchor row, `E` opens the AI-drafted epic form (editable), then
queues Create Epic + Set Parent actions.

Detail edits: `s` sprint, `d` due date, `p` parent epic, `l` labels, `t`
transition, `o` open in browser, `y` copy key.

## Queue tab

Pending actions grouped by kind. Per row:

- `a` approve (apply to Jira)
- `e` edit comment/summary
- `x` reject
- `A` bulk approve (with confirm)

After bulk-approving sync-prep comment actions, a weekly digest markdown file is
written to `<ConfigDir>/digests/YYYY-WW.md` and copied to the clipboard.

## Epics tab

Lists epics with child counts and a timeline strip. `h`/`l` scroll the timeline;
click a bar to focus an epic.

## Settings tab

Subtabs: General, Jira, AI Profiles, Theme. AI profile CRUD and theme color
picker are reachable by keyboard and mouse.

## Demo mode

```sh
make demo          # seeds tmp/demo and launches --demo
./aitinerary --demo
```

Demo mode uses fixture tickets, PRs, and epics from `internal/demo` — no live
Jira, GitHub, or Ollama calls.
