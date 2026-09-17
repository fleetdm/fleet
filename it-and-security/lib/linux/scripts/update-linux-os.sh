#!/bin/bash
# Installs all available OS package updates so the host reaches its distribution's
# current point release. Pair it with an "Operating system up to date (Linux)" policy
# as its run_script remediation, or run it on demand.
#
# Package upgrades only: never reboots, never crosses a release boundary
# (Ubuntu 24.04 -> 26.04, Fedora 43 -> 44), never removes packages. The package
# manager runs in a transient systemd unit so Fleet's script timeout can't kill it
# mid-transaction; if it is still running when this script must return, it exits 0
# and the next policy run picks up where it left off.

set -euo pipefail

TAG="[update-linux-os]"
UNIT="fleet-os-update"
STATE_DIR="/var/lib/fleet"
STATUS_FILE="$STATE_DIR/os-update.status"
WAIT_SECONDS=240          # keep under agent_options.script_execution_timeout (default 300s)
MIN_BATTERY_PERCENT=30

log() { echo "$TAG $*"; }

if [ "$(id -u)" -ne 0 ]; then
  log "This script must run as root." >&2
  exit 1
fi
if ! command -v systemd-run >/dev/null 2>&1; then
  log "systemd-run not found; refusing to run a package upgrade that Fleet's timeout could interrupt." >&2
  exit 1
fi

. /etc/os-release
before="${PRETTY_NAME:-$NAME ${VERSION:-}}"

if systemctl is-active --quiet "$UNIT.service" 2>/dev/null; then
  log "A previous OS update is still running (see: journalctl -u $UNIT); will re-check on the next policy run."
  exit 0
fi

# An upgrade interrupted by a dead battery can leave dpkg/rpm half-applied.
for supply in /sys/class/power_supply/*; do
  [ -r "$supply/status" ] && [ -r "$supply/capacity" ] || continue
  if [ "$(cat "$supply/status")" = "Discharging" ] && [ "$(cat "$supply/capacity")" -lt "$MIN_BATTERY_PERCENT" ]; then
    log "On battery at $(cat "$supply/capacity")%; deferring until charging or above $MIN_BATTERY_PERCENT%."
    exit 0
  fi
done

case " ${ID:-} ${ID_LIKE:-} " in
  *" debian "*|*" ubuntu "*)
    # `upgrade --with-new-pkgs` pulls new dependencies (e.g. a new kernel through its
    # meta-package) but never removes anything, unlike full-upgrade.
    UPGRADE_CMD='export DEBIAN_FRONTEND=noninteractive NEEDRESTART_MODE=a UCF_FORCE_CONFFOLD=1
      apt-get update -o DPkg::Lock::Timeout=300 &&
      apt-get -y -o DPkg::Lock::Timeout=300 \
        -o Dpkg::Options::=--force-confdef -o Dpkg::Options::=--force-confold \
        upgrade --with-new-pkgs'
    ;;
  *" fedora "*|*" rhel "*|*" centos "*)
    # No --refresh: hourly policy runs would otherwise re-download repo metadata every time.
    UPGRADE_CMD='dnf -y upgrade'
    ;;
  *)
    log "No unattended update path for ${PRETTY_NAME:-$ID}. Update it as described in the policy resolution, then refetch." >&2
    exit 1
    ;;
esac

mkdir -p "$STATE_DIR"
rm -f "$STATUS_FILE"
systemctl reset-failed "$UNIT.service" 2>/dev/null || true
started="$(date '+%Y-%m-%d %H:%M:%S')"
log "Before: $before"
log "Starting package upgrade in transient unit $UNIT..."
systemd-run --unit="$UNIT" --collect --quiet --description="Fleet OS update" \
  /bin/bash -c "$UPGRADE_CMD; echo \$? > $STATUS_FILE"

waited=0
while systemctl is-active --quiet "$UNIT.service" && [ "$waited" -lt "$WAIT_SECONDS" ]; do
  sleep 5
  waited=$((waited + 5))
done

if systemctl is-active --quiet "$UNIT.service"; then
  log "Upgrade still running after ${WAIT_SECONDS}s; it continues in the background and the policy re-checks on its next run."
  exit 0
fi

status="$(cat "$STATUS_FILE" 2>/dev/null || echo unknown)"
echo "--- package manager output (tail) ---"
journalctl -u "$UNIT.service" --since "$started" --no-pager -o cat 2>/dev/null | tail -n 40 || true
echo "--- end ---"
if [ "$status" != "0" ]; then
  log "Package upgrade failed (exit $status)." >&2
  exit 1
fi

. /etc/os-release
log "After: ${PRETTY_NAME:-$NAME ${VERSION:-}}"

reboot_needed=""
[ -e /var/run/reboot-required ] && reboot_needed=1
# needs-restarting exits 1 when a reboot is required; skip if the plugin isn't installed.
if command -v dnf >/dev/null 2>&1 && dnf needs-restarting --help >/dev/null 2>&1; then
  rc=0; dnf needs-restarting -r >/dev/null 2>&1 || rc=$?
  [ "$rc" -eq 1 ] && reboot_needed=1
fi
if [ -n "$reboot_needed" ]; then
  log "Updates installed; a restart is required to finish (kernel or core libraries)."
  username="$(ps -o user= -C fleet-desktop 2>/dev/null | head -n 1)"
  if [ -n "$username" ] && command -v notify-send >/dev/null 2>&1; then
    uid="$(id -u "$username")"
    sudo -u "$username" -H env DBUS_SESSION_BUS_ADDRESS="unix:path=/run/user/$uid/bus" \
      notify-send --app-name=Fleet "Operating system updated" \
      "Fleet installed OS updates. Restart when convenient to finish applying them." 2>/dev/null || true
  fi
fi

if [ "${ID:-}" = "fedora" ]; then
  log "Note: package updates don't move a host to a newer Fedora release. If the policy keeps failing, run: sudo dnf system-upgrade download --releasever=<latest> && sudo dnf system-upgrade reboot"
fi
