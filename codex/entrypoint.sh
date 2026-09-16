#!/bin/sh
set -eu

umask 077
mkdir -p "$CODEX_HOME" "$CODEX_HOME/skills" "$CODEX_HOME/workspace" "$(dirname "$CODEX_SOCKET")"
rm -rf "$CODEX_HOME/skills/request-guard" "$CODEX_HOME/skills/.system/request-guard"
if [ -f /etc/codex/AGENTS.md ] && [ ! -f "$CODEX_HOME/AGENTS.md" ]; then
  cp /etc/codex/AGENTS.md "$CODEX_HOME/AGENTS.md"
fi
if [ -d /dist/defaults ]; then
  cp -r --update=none /dist/defaults/. "$CODEX_HOME/"
  rm -rf /dist/defaults
fi
rm -f "$CODEX_SOCKET"
cd "$CODEX_HOME"

codex-app-server -c skills.bundled.enabled=false --listen "unix://$CODEX_SOCKET" &
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
status=0
wait "$app_server_pid" || status=$?
trap - INT TERM
exit "$status"
