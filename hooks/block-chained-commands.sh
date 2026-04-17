#!/bin/bash
# Thin launcher that resolves the platform-specific chain-command-blocker
# binary and execs it. The actual logic lives in cmd/chain-command-blocker.

set -u

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case "$ARCH" in
  x86_64)  ARCH="amd64" ;;
  aarch64) ARCH="arm64" ;;
esac

PLUGIN_ROOT="${CLAUDE_PLUGIN_ROOT:-$(cd "$(dirname "$0")/.." && pwd)}"
BIN="${PLUGIN_ROOT}/bin/chain-command-blocker-${OS}_${ARCH}"

if [ ! -x "$BIN" ]; then
  echo "chain-command-blocker: binary not found for ${OS}_${ARCH}: $BIN" >&2
  exit 0
fi

exec "$BIN" "$@"
