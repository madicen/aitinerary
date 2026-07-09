# Architecture decisions (ADRs)

Lightweight records of non-obvious product and integration choices.

## ADR-001: Auto-apply policy

**Status:** Accepted

**Context:** Merged PRs can map to Jira tickets with varying confidence. AI-
written content must not reach Jira unreviewed.

**Decision:** Auto-apply only `Transition → Done` when:

1. `auto_done_on_merge` is true,
2. the matcher reports **high** confidence (exactly one key in the branch),
3. the ticket is not already Done.

Everything else — ambiguous PR matches, AI comments, epic drafts, field edits —
enters the review queue. Auto-applied transitions appear in a recent strip with
one-key undo and are journaled.

**Consequences:** Safe hygiene is automatic; TL-facing text always passes human
review. Sync-prep bulk apply still requires explicit approval in the Queue tab.

---

## ADR-002: Parent field vs Epic Link fallback

**Status:** Accepted

**Context:** Modern Jira uses `parent` for epics on team-managed projects.
Company-managed instances may reject `parent` and require the legacy Epic Link
custom field.

**Decision:** At runtime, field discovery resolves Sprint, Start date, Epic Name,
and Epic Link IDs — never hardcode `customfield_*`. Set parent via `PUT
/rest/api/3/issue/{key}` with `{"fields":{"parent":{"key":"…"}}}`. On HTTP 400
with a parent-field error, retry using the discovered Epic Link field.

**Consequences:** One code path works across APP (team-managed) and HELP
(company-managed) once discovery runs against the live instance. Fallback
behavior must be verified early on production Jira.

---

## ADR-003: Ticket-key matching confidence

**Status:** Accepted

**Context:** PR branches, titles, and bodies may mention zero, one, or many
ticket keys. False positives (e.g. `APP-1` inside `APP-11`) must be avoided.

**Decision:** Build the key regex from configured project keys (`APP`, `HELP`,
…). Confidence rules:

| Signal | Confidence |
|--------|------------|
| Exactly one `(PROJ-\d+)` in branch name | **High** — eligible for auto-Done |
| Key only in title/body, or multiple keys | **Ambiguous** — queue for review |
| Key appears as substring of a longer key | No match (word-boundary aware) |

**Consequences:** Branch naming discipline (`feature/APP-123-slug`) unlocks
hands-off Done transitions; messy PR text stays in the queue.
