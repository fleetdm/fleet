#!/usr/bin/env bash
# Dev wrapper for `wails3 dev` with reliable Ctrl+C teardown.
#
# wails3 dev launches the Vite dev server as a *detached* child (its own
# process group), so a plain Ctrl+C — delivered only to the foreground group —
# leaves Vite orphaned on the port ("address already in use" next run).
#
# We run wails3 as a *background job under job control* (`set -m`, so it gets
# its own process group) and `wait` on it. That keeps this shell free to run
# its trap the instant a signal arrives (running wails3 in the foreground
# would block the trap until wails3 exits — which it doesn't, hence the hang).
# On signal we tear down explicitly: wails3 + the app (its process group),
# then Vite (by port, since it's in a separate group).
set -uo pipefail
set -m

PORT="${1:-9245}"

cleanup() {
  trap '' INT TERM   # ignore repeat signals while we tear down
  # Stop wails3 dev + the app (its process group). wails3 dev ignores SIGINT,
  # so TERM it, then escalate to KILL if it lingers.
  if [ -n "${WAILS:-}" ]; then
    kill -TERM -"${WAILS}" 2>/dev/null || kill -TERM "${WAILS}" 2>/dev/null || true
    sleep 0.3
    if kill -0 -"${WAILS}" 2>/dev/null || kill -0 "${WAILS}" 2>/dev/null; then
      kill -KILL -"${WAILS}" 2>/dev/null || kill -KILL "${WAILS}" 2>/dev/null || true
    fi
  fi
  # Vite is detached into its own group by wails3 — free it by port.
  local pids
  pids="$(lsof -ti "tcp:${PORT}" 2>/dev/null || true)"
  if [ -n "${pids}" ]; then
    # shellcheck disable=SC2086  # intentional word-split: kill may take several PIDs
    kill -KILL ${pids} 2>/dev/null || true
  fi
}
trap cleanup INT TERM EXIT

wails3 dev -config ./build/config.yml -port "${PORT}" &
WAILS=$!
wait "${WAILS}"
