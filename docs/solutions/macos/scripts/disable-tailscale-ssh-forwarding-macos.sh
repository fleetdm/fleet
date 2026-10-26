#!/bin/bash
# disable-tailscale-ssh-forwarding-macos.sh
# Interim mitigation for the October 2026 Tailscale SSH forwarding reports ("Tailworm").
# Only the open-source tailscaled (Homebrew or "tailscaled install-system-daemon") runs a
# Tailscale SSH server on macOS; the Tailscale app from the App Store or tailscale.com doesn't.
# Sets TS_SSH_DISABLE_FORWARDING=1 in /etc/tailscale/tailscaled-env.txt and restarts tailscaled,
# which drops Tailscale connections on this Mac for a few seconds.
# Idempotent. Built for Fleet "Run script" (runs as root).
# Rollback: remove the line from the file, then launchctl kickstart -k system/<label>.
# Exit: 0 = applied or nothing to do, 2 = needs a look (read output), 1 = error.

set -u
export PATH="/usr/bin:/bin:/usr/sbin:/sbin"
export LC_ALL=C
ENV_FILE="/etc/tailscale/tailscaled-env.txt"
LINE="TS_SSH_DISABLE_FORWARDING=1"

log()  { echo "$*"; }
die()  { echo "ERROR: $*" >&2; exit 1; }

[ "$(id -u)" -eq 0 ] || die "must run as root"

label=""
for l in com.tailscale.tailscaled homebrew.mxcl.tailscale; do
    if launchctl print "system/$l" >/dev/null 2>&1; then
        label="$l"
        break
    fi
done

if [ -z "$label" ]; then
    if pgrep -x tailscaled >/dev/null 2>&1; then
        log "tailscaled is running, but not as com.tailscale.tailscaled or homebrew.mxcl.tailscale."
        log "Add $LINE to $ENV_FILE, then restart tailscaled however it was started."
        exit 2
    fi
    log "Open-source tailscaled isn't installed as a LaunchDaemon. The Tailscale app doesn't run a Tailscale SSH server. Nothing to do."
    exit 0
fi

tailscaled_pid() {
    # launchd is the parent of the daemon; SSH session children share its name
    pgrep -x -P 1 tailscaled | head -n 1
}

started_after_env_file() {
    local pid="$1" started changed
    started="$(date -j -f "%a %b %e %T %Y" "$(ps -o lstart= -p "$pid" | sed 's/  */ /g; s/ $//')" +%s 2>/dev/null)" || return 1
    changed="$(stat -f %m "$ENV_FILE")" || return 1
    [ "$started" -ge "$changed" ]
}

pid="$(tailscaled_pid)"
if [ -f "$ENV_FILE" ] && grep -qx "$LINE" "$ENV_FILE" && [ -n "$pid" ] && started_after_env_file "$pid"; then
    log "Already applied: tailscaled (pid $pid, $label) started after $ENV_FILE set $LINE."
    exit 0
fi

mkdir -p "$(dirname "$ENV_FILE")" || die "cannot create $(dirname "$ENV_FILE")"
tmp="$(mktemp "${ENV_FILE}.XXXXXX")" || die "cannot create a temp file next to $ENV_FILE"
if [ -f "$ENV_FILE" ]; then
    grep -v '^[[:space:]]*TS_SSH_DISABLE_FORWARDING[[:space:]]*=' "$ENV_FILE" > "$tmp"
fi
echo "$LINE" >> "$tmp"
if ! { chmod 0644 "$tmp" && chown root:wheel "$tmp" && mv -f "$tmp" "$ENV_FILE"; }; then
    rm -f "$tmp"
    die "cannot write $ENV_FILE"
fi
log "WROTE: $ENV_FILE"

log "Restarting tailscaled ($label)..."
launchctl kickstart -k "system/$label" || die "launchctl kickstart -k system/$label failed"
for _ in 1 2 3 4 5 6 7 8 9 10; do
    sleep 1
    pid="$(tailscaled_pid)"
    [ -n "$pid" ] && started_after_env_file "$pid" && break
done

if [ -n "$pid" ] && started_after_env_file "$pid"; then
    log "DONE: tailscaled (pid $pid) restarted with $LINE."
    exit 0
fi
log "tailscaled didn't come back after the restart, or started before $ENV_FILE changed. Check: launchctl print system/$label"
exit 2
