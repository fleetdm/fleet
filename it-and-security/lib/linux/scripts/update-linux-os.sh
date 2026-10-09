#!/bin/bash
# Installs pending OS package updates (apt or dnf) so the host reaches its distribution's
# current point release. It never reboots, never upgrades to a new release (e.g. Ubuntu
# 24.04 -> 26.04), and never removes packages.

set -euo pipefail

TAG="[update-linux-os]"
UNIT="fleet-os-update"
STATE_DIR="/var/lib/fleet"
STATUS_FILE="$STATE_DIR/os-update.status"
WAIT_SECONDS=240          # keep under Fleet's script timeout (default 300s)
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
  started_at="$(systemctl show -p ActiveEnterTimestamp --value "$UNIT.service" 2>/dev/null)"
  log "An OS update started at $started_at is still running in the background (see: journalctl -u $UNIT). Exiting before Fleet's script timeout; the policy re-checks on its next run."
  exit 0
fi

for supply in /sys/class/power_supply/*; do
  [ -r "$supply/type" ] && [ "$(cat "$supply/type")" = "Battery" ] || continue
  [ -r "$supply/scope" ] && [ "$(cat "$supply/scope")" = "Device" ] && continue
  [ -r "$supply/status" ] && [ -r "$supply/capacity" ] || continue
  if [ "$(cat "$supply/status")" = "Discharging" ] && [ "$(cat "$supply/capacity")" -lt "$MIN_BATTERY_PERCENT" ]; then
    log "On battery at $(cat "$supply/capacity")%; deferring until charging or above $MIN_BATTERY_PERCENT%."
    exit 0
  fi
done

case " ${ID:-} ${ID_LIKE:-} " in
  *" debian "*|*" ubuntu "*)
    UPGRADE_CMD='export DEBIAN_FRONTEND=noninteractive NEEDRESTART_MODE=a UCF_FORCE_CONFFOLD=1
      if [ -n "$(ls -A /var/lib/dpkg/updates 2>/dev/null)" ] && ! pgrep -x dpkg >/dev/null; then
        dpkg --force-confdef --force-confold --configure -a
      fi
      apt-get update -o DPkg::Lock::Timeout=300 || true
      apt-get -y -o DPkg::Lock::Timeout=300 -o Dpkg::Use-Pty=0 \
        -o Dpkg::Options::=--force-confdef -o Dpkg::Options::=--force-confold \
        upgrade --with-new-pkgs'
    ;;
  *" fedora "*|*" rhel "*|*" centos "*)
    UPGRADE_CMD='dnf -y upgrade'
    ;;
  *)
    log "No unattended update path for ${PRETTY_NAME:-$ID}. Update it as described in the policy resolution, then refetch." >&2
    exit 1
    ;;
esac

mkdir -p "$STATE_DIR"
if [ -f "$STATUS_FILE" ]; then
  prev_status="$(cat "$STATUS_FILE" 2>/dev/null || echo "")"
  if [ -n "$prev_status" ] && [ "$prev_status" != "0" ]; then
    log "Previous background upgrade exited $prev_status (see: journalctl -u $UNIT); retrying."
  fi
fi
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
  log "OS update started by this run is still running after ${WAIT_SECONDS}s and continues in the background (see: journalctl -u $UNIT). Exiting before Fleet's script timeout; the policy re-checks on its next run."
  exit 0
fi

status="$(cat "$STATUS_FILE" 2>/dev/null || echo unknown)"
echo "--- package manager output (tail) ---"
journalctl _SYSTEMD_UNIT="$UNIT.service" --since "$started" --no-pager -o cat 2>/dev/null | tail -n 40 || true
echo "--- end ---"
if [ "$status" != "0" ]; then
  log "Package upgrade failed (exit $status)." >&2
  exit 1
fi

. /etc/os-release
after="${PRETTY_NAME:-$NAME ${VERSION:-}}"
log "After: $after"

reboot_needed=""
[ -e /var/run/reboot-required ] && reboot_needed=1
if command -v dnf >/dev/null 2>&1; then
  if dnf needs-restarting --help >/dev/null 2>&1; then
    rc=0; dnf needs-restarting -r >/dev/null 2>&1 || rc=$?
    [ "$rc" -eq 1 ] && reboot_needed=1
  elif rpm -q --quiet kernel-core 2>/dev/null; then
    newest_kernel="$(rpm -q kernel-core --qf '%{VERSION}-%{RELEASE}.%{ARCH}\n' | sort -V | tail -n 1)"
    [ "$newest_kernel" != "$(uname -r)" ] && reboot_needed=1
  fi
fi
if [ -n "$reboot_needed" ]; then
  log "Updates installed; a restart is required to finish (kernel or core libraries)."
  uid="$(ps -o uid= -C fleet-desktop 2>/dev/null | head -n 1 | tr -d ' ' || true)"
  if [ -n "$uid" ] && command -v notify-send >/dev/null 2>&1; then
    sudo -u "#$uid" -H env DBUS_SESSION_BUS_ADDRESS="unix:path=/run/user/$uid/bus" \
      notify-send --app-name=Fleet "Operating system updated" \
      "Fleet installed OS updates. Restart when convenient to finish applying them." 2>/dev/null || true
  fi
fi

if [ "${ID:-}" = "fedora" ] && [ "$after" = "$before" ]; then
  log "Note: package updates don't move a host to a newer Fedora release. If the policy keeps failing, run: sudo dnf system-upgrade download --releasever=<latest> && sudo dnf system-upgrade reboot"
fi
