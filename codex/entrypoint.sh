#!/bin/sh
set -eu

umask 077
mkdir -p "$CODEX_HOME" "$CODEX_HOME/skills" "$CODEX_HOME/workspace" "$(dirname "$CODEX_SOCKET")"
cp -n /dist/defaults/* "$CODEX_HOME/"
for skill in /opt/codex/skills/*; do
  [ -d "$skill" ] || continue
  cp -Rn "$skill" "$CODEX_HOME/skills/"
done
rm -f "$CODEX_SOCKET"
cd "$CODEX_HOME"

exec codex-app-server \
  -c 'default_permissions="workspace-only"' \
  -c 'permissions.workspace-only.filesystem./dist/data/skills="read"' \
  --listen "unix://$CODEX_SOCKET"
