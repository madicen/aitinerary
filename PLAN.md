# aitinerary — Build Plan

A BubbleTea TUI for Jira ticket hygiene. This plan is written for an AI agent to execute
incrementally, one work item per changeset. Reference item IDs (e.g. `P3.2`) in commit messages.

## 1. Product summary

Michael creates tickets fine (jj-tui handles that). What rots is everything *after*:
tickets not marked Done when the PR merges, missing epics, missing sprints, and (soon
mandatory) missing due dates. Before weekly syncs he burns time getting the board to
reflect reality for his TL and manager.

aitinerary is the after-the-fact ticket manager:

- **Done detection** — scan merged GitHub PRs for ticket keys, surface tickets that should
  already be Done, auto-transition the unambiguous ones.
- **Hygiene detectors** — flag tickets missing epic, sprint, due date, or stale In Progress.
- **Retroactive epic builder** — multi-select tickets → AI drafts an epic → create it →
  set it as parent on all selected tickets, with start/due dates so it lands correctly on
  the Jira timeline view.
- **Sync-prep mode** — one command before the weekly sync: AI drafts a concise status
  comment per active ticket, proposes transitions, and produces a digest for TL/manager.
- **AI profiles** — local-first (Ollama), switchable per-action via long-press menu,
  same profile model as jj-tui/appr-ai-sal.

Workspaces: day-to-day work in project **APP** (Applications); IRE items in project
**HELP** (EngOps Support) filtered by label **Apps**. Both must be first-class; nothing
may be hardcoded to these — they are config defaults.

## 2. Operating principles

**Content**: everything written to Jira must be *useful, brief, accurate*. AI prompts must
enforce: ≤2 sentences for status updates, no filler ("I am pleased to report"), no invented
facts — only what the diff/PR/ticket data supports. Audience is a TL/manager scanning a board.

**Autonomy** (decided): *auto for safe ops*. When a merged PR maps to exactly one ticket
with an unambiguous key match, transition to Done automatically (with journal + undo).
Everything AI-written (comments, epic drafts, due dates) and every ambiguous match goes
to a review queue — approve/edit/reject, with bulk approve. Nothing AI-authored reaches
Jira unreviewed.

**Every action must be reachable by both mouse and keyboard.** No exceptions. Every
clickable element gets a bubblezone; every zone action has a key binding; help and status
bar render from the same binding definitions as the handlers.

**Local-first AI**: default profile is Ollama (`http://127.0.0.1:11434/v1`). Cloud
providers are opt-in profiles.

## 3. Reference repos — pattern map

Both live as siblings in this GitHub folder. Lift these patterns (copy code where licensing
is ours; it all is):

| Pattern | Source | Where |
|---|---|---|
| Jira REST v3 client (auth, search, create, transitions, ADF) | jj-tui | `internal/integrations/jira/service.go`, `internal/integrations/httpapi/` |
| Ticket abstraction interface | jj-tui | `internal/tickets/interface.go` |
| Tab registry + adapter (no concrete-tab imports in root Update/View) | jj-tui | `internal/tui/tab/`, `internal/tui/model/tab_adapters.go`, `init.go` |
| `state.NavigateMsg` single navigation callback | both | jj-tui `internal/tui/state/navigate.go`; appr-ai-sal `navigation.go` |
| Overlay stack + one `DismissMsg`, dispatch on popped concrete type | appr-ai-sal | `internal/tui/overlays/stack.go` |
| Overlay window chrome (drag/resize/minimize) | both | jj-tui `model/modal_layer.go`, `overlay_helpers.go`; appr-ai-sal `reviewWindowConfig()` |
| Mouse zone catalog + naming (`zone:<area>:<action>` consts, indexed helpers) | jj-tui | `internal/tui/mouse/zones.go` |
| Long-press context menu + double-click + list nav | jj-tui | `internal/tui/longpress/`, `mousedouble/`, `listnav/` |
| AI-profile long-press picker on generate buttons | jj-tui | `internal/tui/genmenu/genmenu.go` |
| Keys map = single source for handlers, help, status bar, palette | appr-ai-sal | `internal/tui/keys/keys.go` |
| `ctrl+k` fuzzy command palette | appr-ai-sal | `commands.Registry` |
| Multi-profile AI config with secret indirection (`api_key_env`, `api_key_cmd`) | appr-ai-sal | `internal/aiconfig/config.go` |
| AI provider layer (ollama/openai-compat/gemini/anthropic, retry, streaming, JSON mode) | appr-ai-sal | `internal/ai/` |
| LLM JSON salvage parser | appr-ai-sal | `internal/llmjson/` (copy wholesale) |
| GitHub transport tier (go-gh, `graphQLQuery[T]`, `APIError` taxonomy) | appr-ai-sal | `internal/gh/gh.go`, `client.go` |
| XDG dir resolution | appr-ai-sal | `internal/appdirs/` |
| Theme (3-color + light/dark/NO_COLOR) + color-picker settings panel | both | appr-ai-sal `internal/theme/`, `tabs/settings/theme_panel.go` |
| `dropdown.Host` wrapper (rebuild-on-options-change, Y-rebasing, OnSelect) | appr-ai-sal | `internal/tui/util/dropdown/host.go` |
| Golden tests (`ForceMonochrome`, `Normalize`, `AssertGolden`) + teatest flows | appr-ai-sal | `internal/tui/tuitest/` |
| Demo mode + VHS tapes + fixture scripts | both | `vhs/`, `scripts/setup-demo-fixtures.sh`, `internal/demo/` |
| File logging for TUI (`applog`) | appr-ai-sal | `internal/applog/` |

Known anti-patterns to avoid (documented in jj-tui's IMPROVEMENT_PLAN.md): god files
(`service.go` 3k+ LOC, `config.go` 1.4k LOC, monolithic root `model.go`); ad-hoc concrete
tab wiring; copy-pasted component glue (wrap once, like `dropdown.Host`). Also: **never
commit config containing tokens** — jj-tui committed a live `.jj-tui.json`; aitinerary
git-ignores local config from the first commit and prefers `api_key_env`/`api_key_cmd`.

## 4. Architecture

Module `github.com/madicen/aitinerary`, Go 1.25. Deps mirror the reference repos:
bubbletea v1.3.x, bubbles, lipgloss, lrstanley/bubblezone, madicen/bubble-overlay,
madicen/bubble-dropdown, madicen/bubble-color-picker, cli/go-gh/v2, glamour,
sahilm/fuzzy, atotto/clipboard, x/exp/teatest (test).

```
cmd/aitinerary/main.go        flags, config load, theme, tea.NewProgram(WithAltScreen, WithMouseCellMotion), FlushMouse on exit
internal/
  appdirs/                    XDG resolution ($AITINERARY_CONFIG_DIR → XDG → ~/.config/aitinerary)
  applog/                     file logging
  config/                     app config (jira, github, behavior) — small files, not one god file
  aiconfig/                   AI profiles (schema from appr-ai-sal)
  ai/                         provider layer (port from appr-ai-sal: ollama, openai_compatible, gemini, anthropic; retry; JSON mode)
  llmjson/                    salvage parser (copied)
  jira/                       REST client: client.go (transport), search.go, issue.go, transitions.go, sprints.go, epics.go, fields.go, adf.go
  gh/                         thin PR layer over gh CLI / go-gh: merged-PR scan, PR lookup by key
  match/                      ticket-key ↔ PR matching engine
  hygiene/                    detectors + rules → findings
  actions/                    action queue: pending/apply/journal/undo
  prompts/                    embedded AI prompt templates (go:embed), user overrides in <ConfigDir>/prompts/
  tui/
    tui.go                    thin facade (Model, New, FlushMouse)
    model/                    root model, routing, navigation, modal layer
    state/                    ViewMode, NavigateMsg
    keys/                     keys.Map — single source of truth
    mouse/                    zones.go catalog
    commands/                 ctrl+k palette registry
    overlays/                 stack helpers, DismissMsg
    tabs/board/               main ticket list (APP + HELP unified)
    tabs/queue/               review queue
    tabs/epics/               epic builder + timeline
    tabs/settings/            general / jira / ai-profiles / theme subtabs
    tabs/help/                shortcuts (rendered from keys.Map)
    util/dropdown/            dropdown.Host wrapper
    tuitest/                  golden test helpers
  demo/                       canned data for VHS/demo mode
vhs/  screenshots/  scripts/  docs/
```

**Model tree**: root Model owns `mode state.ViewMode`, `tabs map[ViewMode]Tab` (appr-ai-sal
`Tab` interface: `Init/Update/View/Resize/SetContentOrigin`), `overlayStack overlay.OverlayStack`
+ `overlay.FocusTrap`, `keys keys.Map`, `palette *commands.Registry`, zone manager, spinner,
status bar. Routing order (proven in appr-ai-sal): overlay-forwarded msgs → typed async msgs →
key/mouse (status bar → active tab → overlay trap → global keys) → fallthrough to active tab.
Sub-models never change views; they emit `state.NavigateMsg`.

**Data flow**: `jira.Client` + `gh.Client` feed a `store` cache (tickets, sprints, epics,
merged PRs) refreshed async with per-source freshness timestamps. `hygiene.Scan(store)`
produces `[]Finding`. Findings map to `actions.Action` (transition, set-parent, set-sprint,
set-duedate, add-comment, create-epic). Safe actions auto-apply; the rest wait in the queue.
Every applied action is journaled (`<ConfigDir>/journal.jsonl`) with enough data to undo
(previous status, previous field values).

## 5. Config

`<ConfigDir>/config.json` (0600, atomic write, git-ignored if repo-local override used):

```json
{
  "jira_url": "https://stackadapt.atlassian.net",
  "jira_user": "michael.madicen@stackadapt.com",
  "jira_token_env": "JIRA_TOKEN",
  "jira_token_cmd": "",
  "workspaces": [
    {"name": "Applications", "project": "APP", "jql_extra": ""},
    {"name": "IRE", "project": "HELP", "labels": ["Apps"], "jql_extra": ""}
  ],
  "board_ids": {"APP": 0, "HELP": 0},
  "excluded_statuses": ["Won't Fix", "Will Not Do"],
  "done_status_name": "Done",
  "auto_done_on_merge": true,
  "stale_in_progress_days": 10,
  "github_token_source": "gh_cli",
  "github_repos": [],
  "theme_primary": "#7E00AF", "theme_secondary": "#50FA7B", "theme_muted": "#6272A4"
}
```

- Token precedence: `JIRA_TOKEN` env → `jira_token_cmd` stdout → error with setup hint.
  Never store the token itself in config (lesson from jj-tui).
- `github_repos` empty = discover from current directory's `gh repo view` plus a configured list.
- `board_ids` 0 = auto-discover via agile API and persist.
- AI profiles live separately in `<ConfigDir>/ai.json` using appr-ai-sal's exact schema
  (`profiles[]`, `active_profile`, `api_key_env`/`api_key_cmd`, `stage_models`).

## 6. Jira API surface (net-new beyond jj-tui)

jj-tui covers: basic-auth transport, `GET /rest/api/3/search/jql`, `GET/POST issue`,
transitions, ADF read/write. aitinerary adds:

| Need | Endpoint | Notes |
|---|---|---|
| Field discovery | `GET /rest/api/3/field` | Resolve custom field IDs at runtime for Sprint, Start date, Epic Name. Cache in config. **Never hardcode `customfield_10020` etc.** |
| Boards | `GET /rest/agile/1.0/board?projectKeyOrId={key}` | Auto-discover board per project |
| Sprints | `GET /rest/agile/1.0/board/{id}/sprint?state=active,future` | For sprint pickers |
| Assign to sprint | `POST /rest/agile/1.0/sprint/{sprintId}/issue` body `{"issues":["APP-1"]}` | Preferred over field edit |
| Set epic parent | `PUT /rest/api/3/issue/{key}` body `{"fields":{"parent":{"key":"APP-100"}}}` | Modern API uses `parent` for both team- and company-managed. Fall back to Epic Link custom field if the instance rejects `parent` (detect via 400 + field discovery). |
| Due date | `PUT .../issue/{key}` `{"fields":{"duedate":"2026-08-01"}}` | ISO date |
| Start date (timeline) | `{"fields":{"<discovered start date field>":"2026-07-15"}}` | Timeline view positions epics by start+due; both must be set by the epic builder |
| Create epic | `POST /rest/api/3/issue` with issuetype Epic (+ Epic Name field only if discovery says it's required) | |
| Comments | `POST /rest/api/3/issue/{key}/comment` (ADF body) | For AI status updates |
| Labels | `PUT` `{"update":{"labels":[{"add":"Apps"}]}}` | |
| Rank (board order) | `PUT /rest/agile/1.0/issue/rank` `{"issues":[...],"rankBeforeIssue":"..."}` | Sync-prep board ordering |
| Search fields | extend jj-tui's field list with `labels,duedate,parent,assignee,updated,<sprint field>,<start date field>` | |

Verification requirement: the agent building each endpoint must write a hermetic test with
a stub RoundTripper (appr-ai-sal pattern) asserting method, path, and body shape.

## 7. Phases

Each item is one prompt-sized changeset. Acceptance criteria (AC) must pass before "done".
Global AC for every item: `make build`, `make test`, `golangci-lint run` all green.

### P0 — Scaffold
- **P0.1** `go mod init github.com/madicen/aitinerary`; Makefile (build/run/test/test-race/cover/fmt/vet/lint/clean); `.gitignore` (binary, coverage, `.aitinerary*.json`, tmp); MIT LICENSE; README stub. AC: `make build` produces a binary that prints version.
- **P0.2** Port `appdirs` and `applog` from appr-ai-sal (rename env prefix `AITINERARY_`). AC: unit tests for dir precedence.
- **P0.3** `internal/config`: schema from §5, load/merge/save (0600, atomic), token resolution (env→cmd), `Redacted()` for display. Split into small files (schema.go, load.go, secrets.go). AC: round-trip + precedence tests; token never serialized.
- **P0.4** CI workflow (build, vet, test -race, golangci-lint, govulncheck; ubuntu+macos), `.golangci.yml`, `.goreleaser.yml` from appr-ai-sal templates. AC: CI green on push.

### P1 — Jira client
- **P1.1** `internal/jira/client.go`: transport with basic auth (email+token), JSON helpers, `APIError` taxonomy with status + Retry-After (port appr-ai-sal's `apiErrorFrom` shape), injectable `http.RoundTripper`. Startup permission check (`GET /rest/api/3/mypermissions?permissions=BROWSE_PROJECTS`, from jj-tui). AC: stub-transport tests for auth header, error mapping, retry-after parse.
- **P1.2** `fields.go`: field discovery per §6, resolving sprint/start-date/epic-name IDs; cached in config with manual refresh. AC: fixture JSON test resolving all three; graceful absence.
- **P1.3** `issue.go` + `adf.go`: Ticket model (key, summary, status, type, priority, labels, duedate, parent, sprint, assignee, updated, description); ADF↔plaintext (port jj-tui, extend to bullet lists — needed for concise epic bodies). `search.go`: JQL builder for workspaces (`project = APP` / `project = HELP AND labels = Apps`, `assignee = currentUser()`, excluded statuses, `ORDER BY updated DESC`), pagination beyond 50. AC: JQL golden tests per workspace; ADF round-trip tests.
- **P1.4** `transitions.go` (list/execute, port jj-tui) + write ops: set duedate, set start date, set labels, set parent (with Epic Link fallback per §6). AC: stub tests assert exact request bodies.
- **P1.5** `sprints.go` + `epics.go`: board discovery, sprint list, assign-to-sprint, create epic (with start/due dates), epic list per project. AC: stub tests; create-epic omits Epic Name field when discovery says unneeded.

### P2 — TUI shell
- **P2.1** `tui/state` (ViewMode: Board, Queue, Epics, Settings, Help; NavigateMsg), `tui/keys` (keys.Map, Sections() for help; global: `1-5`/`b q e , ?` tabs, `ctrl+k` palette, `ctrl+r` refresh, `ctrl+z` undo, `esc` back, `ctrl+c` quit), `tui/mouse/zones.go` catalog seeded with tab-bar zones. AC: keys tests; no string-matched keys anywhere.
- **P2.2** Root model: Tab interface + adapter, tab registry, routing order per §4, window size propagation, status bar (mode, workspace, freshness, active AI profile — all clickable), spinner. AC: golden test of empty shell; teatest tab-switch flow (keyboard and mouse).
- **P2.3** Overlay stack integration: `overlays` package with `DismissMsg{Result any}`, FocusTrap gating, synthetic WindowSizeMsg on push, window chrome config helper. Error + confirm modals. AC: teatest: modal traps keys, esc pops, click-outside per config.
- **P2.4** Theme: port appr-ai-sal `theme` (3 base colors + severity/status slots), applied pre-first-frame; `demo` mode flag isolating config to temp dir. AC: NO_COLOR golden test.
- **P2.5** Settings tab skeleton with subtabs (General, Jira, AI Profiles, Theme); Theme panel with bubble-color-picker swatches; dropdowns via ported `dropdown.Host`. AC: golden per subtab; mouse click on swatch opens picker (teatest).
- **P2.6** `ctrl+k` command palette (port commands.Registry + fuzzy). Every action registered from keys.Map. AC: palette lists ≥ all global actions; Run executes same handler as key.

### P3 — Board tab
- **P3.1** `store`: async fetch of workspaces' tickets, merge, freshness per workspace, refresh cmd; demo-mode canned data. AC: store tests with fake jira client.
- **P3.2** Board list UI: unified list with workspace chip (APP/IRE), status, key, summary, hygiene badges (icons for missing epic/sprint/due date, stale); filter chips (workspace, status category, needs-attention) clickable + keyed; search `/`. j/k + wheel + click-select + double-click-open-in-browser (jj-tui listnav/mousedouble). AC: golden with fixture tickets incl. badge rendering; teatest select flows.
- **P3.3** Detail pane (enter/click): full fields, description via glamour, transitions list. Edit affordances: `s` sprint dropdown, `d` due-date input, `p` parent-epic dropdown (epics of that project), `l` labels editor, `t` transition menu — each also a clickable zone. Writes go through `actions` (P5) once it exists; direct-apply until then. AC: teatest per edit flow; stub asserts write body.
- **P3.4** Long-press context menu on rows (jj-tui longpress): transition, assign sprint, set epic, open in browser, copy key. AC: teatest long-press → menu → action.

### P4 — GitHub merge detection
- **P4.1** `internal/gh`: go-gh transport (port appr-ai-sal client.go), `CheckAuth`, list merged PRs for configured repos since N days (GraphQL search `is:pr is:merged author:@me merged:>=...` + head ref/title/body/mergedAt). AC: stub-transport tests; self-skip without gh.
- **P4.2** `internal/match`: extract ticket keys (`(APP|HELP)-\d+` built from configured projects) from branch name, PR title, PR body; confidence: exact-single-key-in-branch = high; title/body-only or multi-key = ambiguous. Match against store tickets not Done. AC: table tests covering false positives (e.g. APP-1 vs APP-11, keys in URLs).
- **P4.3** Board integration: "PR merged" badge on matched tickets; done-candidates filter chip; detail pane shows matched PRs with URL. AC: golden with matched fixture.

### P5 — Actions engine (queue, auto-apply, undo)
- **P5.1** `internal/actions`: Action types (Transition, SetParent, SetSprint, SetDueDate, SetStartDate, AddComment, CreateEpic, SetLabels, Rank), `Source` (auto-detector, ai, user), `Confidence`. Journal (jsonl, records prior values). Apply executor via jira client. Undo for field/transition actions (reverse transition looked up live; warn if unavailable). AC: journal round-trip; undo restores stub state.
- **P5.2** Auto-apply policy: only `Transition→Done` with high-confidence single-PR match and `auto_done_on_merge=true`; everything else queued. Applied-auto items appear in a "recently auto-applied" strip with one-key undo. AC: policy table tests.
- **P5.3** Queue tab: pending actions grouped by kind; per-item approve `a` / edit `e` / reject `x`; bulk approve `A` (with confirm modal); every control zoned. Status bar badge shows queue count from any tab. AC: teatest approve/bulk/undo flows; golden.

### P6 — Hygiene detectors
- **P6.1** `internal/hygiene`: detectors producing Findings: MergedNotDone (from match), MissingEpic, MissingSprint (status ≠ Done, no active/future sprint), MissingDueDate, StaleInProgress (> config days without update), EpicMissingDates (epics lacking start/due — breaks timeline). Each Finding proposes an Action (or an AI task). Scan runs after every refresh. AC: table tests per detector with edge cases (subtasks skip epic check, Done tickets skipped).
- **P6.2** Board "Needs attention" view (default filter on launch): grouped by finding type, count summary in status bar. One-key fix per row where non-AI (e.g. `s` on MissingSprint opens sprint dropdown pre-filtered to active sprint). AC: golden; teatest quick-fix.

### P7 — AI layer
- **P7.1** Port `internal/ai` provider layer + `llmjson` + `aiconfig` from appr-ai-sal (rename env/config prefixes; keep `api_key_env`/`api_key_cmd`, retry, JSON mode, streaming; add `temperature` optional field to profile and request body — gap noted in appr-ai-sal). Default profile: ollama/`qwen2.5-coder:7b`. AC: config round-trip; request-body tests incl. temperature omission when unset.
- **P7.2** `internal/prompts` (go:embed + `<ConfigDir>/prompts/` overrides): `status_update` (input: ticket + matched PRs + diffstat; output JSON `{"comment": "..."}`, ≤2 sentences, factual, no first-person fluff), `epic_draft` (input: selected tickets' keys/summaries/descriptions; output JSON `{"title","body","start_date","due_date"}` — dates inferred from ticket dates/sprints, body ≤4 bullets), `ticket_summary_digest` (input: week's activity; output markdown digest ≤15 lines). Port jj-tui's leading-ticket-key preservation rule where relevant. AC: prompt snapshot tests; llmjson parses lenient fixture outputs (literal newlines, fences).
- **P7.3** Settings → AI Profiles subtab: list/add/edit/delete/activate profiles (dropdowns for provider, inputs for model/base_url/key-env/key-cmd/temperature), redacted display, validate button hitting `/models` or equivalent. AC: teatest CRUD; secrets never rendered.
- **P7.4** genmenu long-press profile picker (port jj-tui): every "Generate" button supports hold-500ms → pick profile → one-shot generate with override, without changing active profile. Request IDs discard stale responses. AC: teatest long-press flow; stale-response test.

### P8 — Epic builder + timeline
- **P8.1** Board multi-select (`space` toggle, `v` range, click with shift-equivalent zone handling) → `E` "Create epic from selection": overlay form pre-filled by AI (`epic_draft` via P7), editable title/body/start/due/project, then CreateEpic + SetParent on all selected as queued actions (epic create itself is user-confirmed in the form — the form is the review). AC: teatest full flow against stub jira; parent fallback path covered.
- **P8.2** Epics tab: list epics per project with child counts, progress (done/total), start/due; edit dates inline (feeds timeline placement); assign-existing-tickets picker (list unparented tickets, multi-select, bulk SetParent). AC: golden; teatest assign flow.
- **P8.3** Timeline strip: horizontal month-scale lane view of epics (start→due bars, today marker), wheel/`h l` scroll, click epic → detail. Read-only render of what Jira's timeline will show; drag-to-move dates is a stretch goal (single follow-up item P8.4, optional). AC: golden at fixed width; bars clamp correctly for undated epics (shown in a "no dates" gutter).

### P9 — Sync-prep mode
- **P9.1** `S` from Board: sweep = refresh → match → hygiene scan → for each active-or-recently-merged ticket, AI drafts status comment (`status_update`) → all as queued actions; plus proposed transitions and rank ordering (In Progress first, then blocked, then to-do by due date). Runs with progress overlay; cancellable. AC: teatest sweep on demo data produces expected queue composition.
- **P9.2** Review + apply loop specialized for sync-prep: side-by-side ticket vs draft comment, edit-in-place, approve advances to next. AC: teatest.
- **P9.3** Digest export: after apply, generate `ticket_summary_digest` markdown (per-project sections; done this week / in flight / blocked / next), copy to clipboard (osc52) and write `<ConfigDir>/digests/YYYY-WW.md`. AC: golden digest from demo fixtures.
- **P9.4** IRE quick-capture: `n` opens minimal create form defaulting to HELP + label Apps (type, title, AI-draft body optional) — keeps IRE flow inside aitinerary. AC: stub asserts create body includes label.

### P10 — Polish & release
- **P10.1** Help tab rendered from keys.Map Sections + first-run setup wizard (jira url/email/token-env check, board discovery, gh auth check, ollama ping). AC: teatest wizard happy path.
- **P10.2** Demo fixtures + VHS tapes (board, queue, epic-builder, sync-prep, settings) + screenshots in README. AC: `make screenshots` succeeds locally.
- **P10.3** docs/: USAGE.md (per-tab keys/workflows), DECISIONS.md (ADRs: auto-apply policy, parent-vs-epic-link fallback, key-matching confidence), TESTING.md. Release v0.1.0 via goreleaser. AC: docs build; tag pushed.

## 8. Agent working conventions

Adopted from jj-tui's IMPROVEMENT_PLAN.md protocol:

1. One work item per changeset; commit message starts with the item ID (`P4.2: ticket-key matcher`).
2. Definition of done: `make build` + `make test` (+ `-race` locally) + `golangci-lint run` green, and the item's AC demonstrably met (goldens updated deliberately via `-update`, never blindly).
3. Tests move with code; new packages get tests in the same changeset.
4. Hermetic by default: no network, no real `gh`/`jira`/ollama in tests — stub transports and `ai.SetBaseProviderForTest`-style seams. Binaries-required tests self-skip.
5. Keep files under ~500 LOC; split before they become god files.
6. Every new user-facing action: key binding in keys.Map + zone in mouse catalog + palette registration + help text — in the same changeset, or it doesn't merge.
7. Never log or serialize secrets; `Redacted()` everywhere config is displayed.
8. When Jira instance behavior is uncertain (custom field IDs, parent-field support), discover at runtime and record the fallback in DECISIONS.md — do not hardcode instance specifics.

## 9. Risks & open questions

- **Instance specifics**: HELP project's board type (Kanban vs Scrum) affects sprint detectors — P6.1 must degrade MissingSprint to skip for Kanban boards. Board discovery (P1.5) should record board type.
- **Reverse transitions** for undo may not exist in every workflow; undo must degrade to "reopen manually" guidance with the journal entry preserved.
- **`parent` field rejection** on company-managed projects with old config → Epic Link fallback path (P1.4) must be tested against the real instance early; do P1 items against live Jira once with a throwaway ticket before building P3+ on top.
- **Local model quality**: qwen2.5-coder:7b may produce weak epic drafts; profiles + genmenu make it one hold-click to retry with a stronger model. Prompt AC should be validated with the eval-harness pattern from appr-ai-sal if drafts disappoint (optional P11).
- **Rate limits**: Jira 429s handled by Retry-After in the client (P1.1); sync-prep sweeps batch reads and serialize writes.
