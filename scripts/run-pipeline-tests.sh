#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PLUGIN_NAME="workflow-plugin-encrypted-spaces"
PLUGIN_DIR="$ROOT/.wfctl/test-plugins"
WFCTL="${WFCTL:-}"
WORKFLOW_REPO="${WORKFLOW_REPO:-$ROOT/../workflow}"

if [[ -z "$WFCTL" ]]; then
  if [[ -x "$WORKFLOW_REPO/bin/wfctl" ]]; then
    WFCTL="$WORKFLOW_REPO/bin/wfctl"
  else
    (cd "$WORKFLOW_REPO" && GOWORK=off go build -o bin/wfctl ./cmd/wfctl)
    WFCTL="$WORKFLOW_REPO/bin/wfctl"
  fi
fi

rm -rf "$PLUGIN_DIR/$PLUGIN_NAME"
mkdir -p "$PLUGIN_DIR/$PLUGIN_NAME"

(cd "$ROOT" && GOWORK=off go build \
  -ldflags "-X github.com/GoCodeAlone/workflow-plugin-encrypted-spaces/internal.Version=${VERSION:-0.0.0}" \
  -o "$PLUGIN_DIR/$PLUGIN_NAME/$PLUGIN_NAME" ./cmd/workflow-plugin-encrypted-spaces)
cp "$ROOT/plugin.json" "$PLUGIN_DIR/$PLUGIN_NAME/plugin.json"

"$WFCTL" test "$ROOT/tests/pipeline"
