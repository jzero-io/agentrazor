#!/bin/sh
set -eu

umask 077
mkdir -p "$CODEX_HOME" "$CODEX_HOME/skills" "$(dirname "$CODEX_SOCKET")"
cp -n /dist/defaults/* "$CODEX_HOME/"
for skill in /dist/skills/*; do
  [ -d "$skill" ] || continue
  cp -Rn "$skill" "$CODEX_HOME/skills/"
done
rm -f "$CODEX_SOCKET"

exec codex-app-server --listen "unix://$CODEX_SOCKET"
