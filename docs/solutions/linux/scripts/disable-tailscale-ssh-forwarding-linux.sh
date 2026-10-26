#!/bin/bash
# disable-tailscale-ssh-forwarding-linux.sh
# Interim mitigation for the October 2026 Tailscale SSH forwarding reports ("Tailworm").
# Sets TS_SSH_DISABLE_FORWARDING=1 for tailscaled with a systemd drop-in and restarts it.
# Tailscale SSH sessions keep working; local, remote, Unix socket, and agent forwarding
# are refused. The restart drops Tailscale connections on this host for a few seconds.
# Idempotent. Built for Fleet "Run script" (runs as root).
# Rollback: delete the drop-in, then systemctl daemon-reload && systemctl restart tailscaled.
# Exit: 0 = applied or nothing to do, 2 = needs a look (read output), 1 = error.

set -u
export PATH="/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"
UNIT="tailscaled.service"
DROPIN_DIR="/etc/systemd/system/${UNIT}.d"
DROPIN="${DROPIN_DIR}/50-disable-ssh-forwarding.conf"
CONTENT='[Service]
Environment=TS_SSH_DISABLE_FORWARDING=1'

log()  { echo "$*"; }
die()  { echo "ERROR: $*" >&2; exit 1; }

[ "$(id -u)" -eq 0 ] || die "must run as root"

# envknob reads the value with strconv.ParseBool
running_has_knob() {
    local pid="$1"
    [ -r "/proc/$pid/environ" ] || return 1
    tr '\0' '\n' < "/proc/$pid/environ" | grep -qiE '^TS_SSH_DISABLE_FORWARDING=(1|t|true)$'
}

main_pid() {
    systemctl show -p MainPID --value "$UNIT" 2>/dev/null
}

if ! command -v systemctl >/dev/null 2>&1 || ! systemctl cat "$UNIT" >/dev/null 2>&1; then
    if pgrep -x tailscaled >/dev/null 2>&1; then
        log "tailscaled is running but not as the systemd unit ${UNIT} (container, OpenWrt, or a custom service)."
        log "Set TS_SSH_DISABLE_FORWARDING=1 in the environment that starts tailscaled, then restart it."
        exit 2
    fi
    log "tailscaled isn't installed as a systemd service on this host. Nothing to do."
    exit 0
fi

pid="$(main_pid)"
if [ -f "$DROPIN" ] && [ "$(cat "$DROPIN")" = "$CONTENT" ] && [ -n "$pid" ] && [ "$pid" != "0" ] && running_has_knob "$pid"; then
    log "Already applied: tailscaled (pid $pid) is running with TS_SSH_DISABLE_FORWARDING=1."
    exit 0
fi

mkdir -p "$DROPIN_DIR" || die "cannot create $DROPIN_DIR"
printf '%s\n' "$CONTENT" > "$DROPIN" || die "cannot write $DROPIN"
chmod 0644 "$DROPIN"
log "WROTE: $DROPIN"
systemctl daemon-reload || die "systemctl daemon-reload failed"

if ! systemctl is-active --quiet "$UNIT"; then
    if pgrep -x tailscaled >/dev/null 2>&1; then
        log "tailscaled is running outside ${UNIT}, so this setting doesn't reach it."
        log "Set TS_SSH_DISABLE_FORWARDING=1 in the environment that starts it, then restart it."
        exit 2
    fi
    log "tailscaled isn't running. The setting applies the next time it starts."
    exit 0
fi

log "Restarting tailscaled..."
systemctl restart "$UNIT" || die "systemctl restart $UNIT failed"
for _ in 1 2 3 4 5 6 7 8 9 10; do
    pid="$(main_pid)"
    [ -n "$pid" ] && [ "$pid" != "0" ] && break
    sleep 1
done

if [ -n "$pid" ] && [ "$pid" != "0" ] && running_has_knob "$pid"; then
    log "DONE: tailscaled (pid $pid) is running with TS_SSH_DISABLE_FORWARDING=1."
    exit 0
fi
log "tailscaled restarted, but its environment doesn't show TS_SSH_DISABLE_FORWARDING=1."
log "Check /etc/default/tailscaled and other drop-ins for a conflicting value: systemctl cat $UNIT"
exit 2
