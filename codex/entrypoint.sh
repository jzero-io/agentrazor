#!/bin/sh
set -eu

umask 077
mkdir -p "$CODEX_HOME" "$CODEX_HOME/skills" "$CODEX_HOME/workspace" "$(dirname "$CODEX_SOCKET")"

for default_skill in /etc/codex/skills/*; do
  [ -d "$default_skill" ] || continue
  skill_name=$(basename "$default_skill")
  [ -e "$CODEX_HOME/skills/$skill_name" ] || cp -a "$default_skill" "$CODEX_HOME/skills/$skill_name"
done

cp -n /dist/defaults/* "$CODEX_HOME/"
rm -f "$CODEX_SOCKET"
cd "$CODEX_HOME"

exec codex-app-server --listen "unix://$CODEX_SOCKET"
