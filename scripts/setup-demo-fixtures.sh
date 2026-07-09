#!/usr/bin/env bash
# Seed AITINERARY_CONFIG_DIR with demo config for VHS recordings and `make demo`.
set -euo pipefail

ROOT="${AITINERARY_CONFIG_DIR:-$(pwd)/tmp/demo}"
mkdir -p "$ROOT/digests" "$ROOT/prompts"

cat >"$ROOT/config.json" <<'EOF'
{
  "jira_url": "https://stackadapt.atlassian.net",
  "jira_user": "demo@example.com",
  "jira_token_env": "JIRA_TOKEN",
  "workspaces": [
    {"name": "Applications", "project": "APP"},
    {"name": "IRE", "project": "HELP", "labels": ["Apps"]}
  ],
  "board_ids": {"APP": 42, "HELP": 43},
  "excluded_statuses": ["Won't Fix", "Will Not Do"],
  "done_status_name": "Done",
  "auto_done_on_merge": true,
  "stale_in_progress_days": 10,
  "github_token_source": "gh_cli",
  "github_repos": ["stackadapt/applications"],
  "theme_primary": "#7E00AF",
  "theme_secondary": "#50FA7B",
  "theme_muted": "#6272A4"
}
EOF

cat >"$ROOT/ai.json" <<'EOF'
{
  "profiles": [
    {
      "name": "default",
      "provider": "ollama",
      "base_url": "http://127.0.0.1:11434/v1",
      "model": "qwen2.5-coder:7b"
    }
  ],
  "active_profile": "default"
}
EOF

# Skip first-run wizard during VHS recordings.
touch "$ROOT/setup_wizard_done"

echo "Seeded demo config in $ROOT"
