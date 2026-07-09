# Testing

## Quick commands

```sh
make test        # go test ./...
make test-race   # race detector
make cover       # coverage report
make lint        # golangci-lint (install separately)
make build       # compile ./aitinerary
```

CI runs build, vet, test with `-race`, golangci-lint, and govulncheck on Ubuntu
and macOS.

## Hermetic tests

Unit and integration tests do **not** call live Jira, GitHub, or Ollama:

- Jira client tests use an injectable `http.RoundTripper` stub.
- GitHub tests stub go-gh transport.
- AI provider tests use `ai.SetBaseProviderForTest` or request-body fixtures.
- TUI flows use `--demo` fixture data via `store.New(..., demo: true)`.

Tests that require the `gh` binary on PATH self-skip when unavailable.

## Golden tests

Render tests live under `internal/tui/**/testdata/*.golden`. They run with
`NO_COLOR=1` and strip ANSI / bubblezone markers via `tuitest.Normalize`.

Update goldens deliberately when UI output changes intentionally:

```sh
go test ./internal/tui/... -update
```

Never commit blind `-update` runs — review the diff.

## teatest flows

End-to-end TUI flows use `github.com/charmbracelet/x/exp/teatest`:

- `internal/tui/model/teatest_flows_test.go` — tab switch (keyboard + mouse)
- `internal/tui/tabs/board/teatest_flow_test.go` — sync-prep sweep
- `internal/tui/tabs/board/teatest_epic_test.go` — epic builder queue
- `internal/tui/tabs/help/teatest_wizard_test.go` — setup wizard (demo)

Helpers: `internal/tui/tuitest` (`ForceMonochrome`, `AssertGolden`,
`ZoneCenter`).

## Demo mode

```sh
make demo                 # build + seed tmp/demo + launch
scripts/setup-demo-fixtures.sh   # seed only (uses AITINERARY_CONFIG_DIR)
./aitinerary --demo
```

Demo mode isolates config to a temp dir (or `tmp/demo` when seeded) and loads
canned tickets/PRs from `internal/demo`.

## VHS screenshots

Requires [VHS](https://github.com/charmbracelet/vhs) (`brew install vhs`):

```sh
make screenshots   # build, seed fixtures, render vhs/*.tape → screenshots/
make vhs-fixtures  # re-seed tmp/demo only
```

Tapes: `vhs/board.tape`, `vhs/queue.tape`, `vhs/settings.tape`.

## Digest export

Queue bulk-approve of sync-prep comments writes
`<ConfigDir>/digests/YYYY-WW.md`. Test with:

```sh
go test ./internal/digest/...
```

Use `AITINERARY_CONFIG_DIR=$(mktemp -d)` in tests for isolation.
