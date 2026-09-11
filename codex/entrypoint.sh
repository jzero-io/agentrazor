#!/bin/sh
set -eu

umask 077
mkdir -p "$CODEX_HOME" "$CODEX_HOME/skills" "$CODEX_HOME/workspace" "$(dirname "$CODEX_SOCKET")"
cp -n /dist/defaults/* "$CODEX_HOME/"
rm -f "$CODEX_SOCKET"
cd "$CODEX_HOME"

exec codex-app-server --listen "unix://$CODEX_SOCKET"
