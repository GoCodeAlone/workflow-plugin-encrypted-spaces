#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PLUGIN_NAME="workflow-plugin-encrypted-spaces"
PLUGIN_DIR="$ROOT/.wfctl/test-plugins"
WFCTL="${WFCTL:-}"
WORKFLOW_REPO="${WORKFLOW_REPO:-}"

find_workflow_repo() {
  if [[ -n "$WORKFLOW_REPO" ]]; then
    [[ -d "$WORKFLOW_REPO" ]] && printf '%s\n' "$WORKFLOW_REPO" && return 0
    return 1
  fi
  local candidate
  for candidate in "$ROOT/../workflow" "$ROOT/../../../workflow"; do
    if [[ -d "$candidate" ]]; then
      local main_sha worktree
      main_sha="$(git -C "$candidate" rev-parse --verify origin/main 2>/dev/null || true)"
      if [[ -n "$main_sha" ]]; then
        while IFS= read -r worktree; do
          if [[ -d "$worktree" ]] && [[ "$(git -C "$worktree" rev-parse HEAD 2>/dev/null || true)" == "$main_sha" ]]; then
            printf '%s\n' "$worktree"
            return 0
          fi
        done < <(git -C "$candidate" worktree list --porcelain 2>/dev/null | awk '/^worktree / {print substr($0, 10)}')
      fi
      printf '%s\n' "$candidate"
      return 0
    fi
  done
  return 1
}

if [[ -z "$WFCTL" ]]; then
  WORKFLOW_REPO="$(find_workflow_repo)" || {
    echo "workflow repo not found; set WFCTL or WORKFLOW_REPO" >&2
    exit 1
  }
  mkdir -p "$WORKFLOW_REPO/bin"
  (cd "$WORKFLOW_REPO" && GOWORK=off go build -o bin/wfctl ./cmd/wfctl)
  WFCTL="$WORKFLOW_REPO/bin/wfctl"
fi

rm -rf "$PLUGIN_DIR/$PLUGIN_NAME"
mkdir -p "$PLUGIN_DIR/$PLUGIN_NAME"

(cd "$ROOT" && GOWORK=off go build \
  -ldflags "-X github.com/GoCodeAlone/workflow-plugin-encrypted-spaces/internal.Version=${VERSION:-0.0.0}" \
  -o "$PLUGIN_DIR/$PLUGIN_NAME/$PLUGIN_NAME" ./cmd/workflow-plugin-encrypted-spaces)
cp "$ROOT/plugin.json" "$PLUGIN_DIR/$PLUGIN_NAME/plugin.json"

"$WFCTL" test "$ROOT/tests/pipeline"
