#!/bin/sh
set -eu

umask 077
mkdir -p "$CODEX_HOME" "$CODEX_HOME/skills" "$CODEX_HOME/workspace" "$(dirname "$CODEX_SOCKET")"
system_skills_home="$CODEX_HOME/skills/.system"
mkdir -p "$system_skills_home"

cp -n /dist/defaults/* "$CODEX_HOME/"
rm -f "$CODEX_SOCKET"
cd "$CODEX_HOME"

codex-app-server --listen "unix://$CODEX_SOCKET" &
app_server_pid=$!

forward_signal() {
  kill -TERM "$app_server_pid" 2>/dev/null || true
}
trap forward_signal INT TERM

attempt=0
while [ ! -S "$CODEX_SOCKET" ]; do
  if ! kill -0 "$app_server_pid" 2>/dev/null; then
    wait "$app_server_pid"
    exit $?
  fi
  attempt=$((attempt + 1))
  if [ "$attempt" -ge 100 ]; then
    echo "codex app-server socket was not ready in time" >&2
    kill -TERM "$app_server_pid" 2>/dev/null || true
    wait "$app_server_pid" || true
    exit 1
  fi
  sleep 0.1
done

for default_skill in /etc/codex/skills/*; do
  [ -d "$default_skill" ] || continue
  skill_name=$(basename "$default_skill")
  [ -e "$system_skills_home/$skill_name" ] || cp -a "$default_skill" "$system_skills_home/$skill_name"
done

status=0
wait "$app_server_pid" || status=$?
trap - INT TERM
exit "$status"
