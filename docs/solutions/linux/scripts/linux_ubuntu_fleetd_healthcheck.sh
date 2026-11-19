#!/usr/bin/env bash
# fleetd_healthcheck_ubuntu.sh
#
# Checks the health of all fleetd components on Ubuntu and collects logs into
# a timestamped archive for support/troubleshooting.
#
# Components checked:
#   - orbit.service      (systemd service)
#   - orbit              (process: /opt/orbit/bin/orbit/orbit)
#   - osqueryd           (process: spawned and managed by orbit)
#   - fleet-desktop      (process: optional, only present if packaged with --fleet-desktop)
#
# Sources:
#   - Service name/unit:  orbit/pkg/packaging/linux_shared.go (writeSystemdUnit)
#   - Binary path:        /opt/orbit/bin/orbit/orbit (symlinked to /usr/local/bin/orbit)
#   - Process name:       constant.DesktopAppExecName = "fleet-desktop"
#   - Log paths:          /var/log/orbit/, /var/log/osquery/ (created at install time)
#   - Env file:           /etc/default/orbit (written by writeEnvFile)
#
# Also collects a lookback window (default 72h, override with LOOKBACK_HOURS env
# var) of system events — reboots, package changes, systemd failures, journal
# errors — to help correlate a reported problem with what changed beforehand.
#
# Must be run as root.

set -euo pipefail

# ── Colour helpers ─────────────────────────────────────────────────────────────
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; NC='\033[0m'
ok()   { echo -e "  ${GREEN}[OK]${NC}    $*"; }
warn() { echo -e "  ${YELLOW}[WARN]${NC}  $*"; }
fail() { echo -e "  ${RED}[FAIL]${NC}  $*"; }
info() { echo -e "  [INFO]  $*"; }

if [[ $EUID -ne 0 ]]; then
  echo "This script must be run as root." >&2
  exit 1
fi

LOOKBACK_HOURS="${LOOKBACK_HOURS:-72}"

TIMESTAMP=$(date +%Y%m%d_%H%M%S)
HOSTNAME_SAFE=$(hostname | tr '.' '_')
ARCHIVE_NAME="fleetd_healthcheck_${HOSTNAME_SAFE}_${TIMESTAMP}"
WORK_DIR=$(mktemp -d "/tmp/${ARCHIVE_NAME}.XXXXXX")
SUMMARY="${WORK_DIR}/summary.txt"
OVERALL_EXIT=0

log() { echo "$*" | tee -a "${SUMMARY}"; }

# ── Header ─────────────────────────────────────────────────────────────────────
log "============================================================"
log " Fleet fleetd Health Check"
log " Host:      $(hostname)"
log " Date:      $(date)"
log " Kernel:    $(uname -r)"
log " OS:        $(. /etc/os-release 2>/dev/null && echo "$PRETTY_NAME" || echo "unknown")"
log "============================================================"
log ""

# ══════════════════════════════════════════════════════════════════════════════
# 1. SYSTEMD SERVICE
# ══════════════════════════════════════════════════════════════════════════════
log "── 1. systemd service (orbit.service) ──────────────────────"

SERVICE="orbit.service"
if systemctl is-active --quiet "${SERVICE}" 2>/dev/null; then
  ok "${SERVICE} is active (running)"
else
  fail "${SERVICE} is NOT active"
  OVERALL_EXIT=1
fi

if systemctl is-enabled --quiet "${SERVICE}" 2>/dev/null; then
  ok "${SERVICE} is enabled"
else
  warn "${SERVICE} is not enabled — will not start on boot"
fi

SYSTEMD_STATUS=$(systemctl status "${SERVICE}" --no-pager 2>&1 || true)
echo "${SYSTEMD_STATUS}" >> "${SUMMARY}"

# ══════════════════════════════════════════════════════════════════════════════
# 2. PROCESS CHECKS
# ══════════════════════════════════════════════════════════════════════════════
log ""
log "── 2. Processes ────────────────────────────────────────────"

check_process() {
  local label="$1"
  local pattern="$2"
  local result
  result=$(pgrep -af "${pattern}" 2>/dev/null || true)
  if [[ -n "${result}" ]]; then
    ok "${label} is running"
    echo "    ${result}" | tee -a "${SUMMARY}"
  else
    fail "${label} is NOT running (pattern: ${pattern})"
    OVERALL_EXIT=1
  fi
}

# orbit binary path is /opt/orbit/bin/orbit/orbit
# Source: linux_shared.go ExecStart=/opt/orbit/bin/orbit/orbit
check_process "orbit"        "/opt/orbit/bin/orbit/orbit"

# osqueryd is spawned by orbit; match the binary name
check_process "osqueryd"     "osqueryd"

# fleet-desktop: only present if package was built with --fleet-desktop
# Source: constant.DesktopAppExecName = "fleet-desktop"
if pgrep -af "fleet-desktop" >/dev/null 2>&1; then
  ok "fleet-desktop is running"
  pgrep -af "fleet-desktop" | tee -a "${SUMMARY}" | sed 's/^/    /'
else
  warn "fleet-desktop is NOT running (expected if not packaged with --fleet-desktop)"
fi

# ══════════════════════════════════════════════════════════════════════════════
# 3. KEY FILES
# ══════════════════════════════════════════════════════════════════════════════
log ""
log "── 3. Key files and directories ────────────────────────────"

check_file() {
  local label="$1"
  local path="$2"
  if [[ -e "${path}" ]]; then
    ok "${label}: ${path}"
  else
    fail "${label} not found: ${path}"
    OVERALL_EXIT=1
  fi
}

check_file "orbit binary"        "/opt/orbit/bin/orbit/orbit"
check_file "orbit symlink"       "/usr/local/bin/orbit"
check_file "osquery pidfile"     "/opt/orbit/osquery.pid"
check_file "env file"            "/etc/default/orbit"
check_file "orbit node key"      "/opt/orbit/secret-orbit-node-key.txt"
check_file "enroll secret"       "/opt/orbit/secret.txt"

# Report orbit node key presence without printing value
if [[ -s "/opt/orbit/secret-orbit-node-key.txt" ]]; then
  ok "orbit node key is non-empty (enrolled)"
else
  fail "orbit node key is missing or empty (not enrolled)"
  OVERALL_EXIT=1
fi

# ══════════════════════════════════════════════════════════════════════════════
# 4. ENV FILE SUMMARY
# ══════════════════════════════════════════════════════════════════════════════
log ""
log "── 4. Orbit environment (/etc/default/orbit) ───────────────"
if [[ -f /etc/default/orbit ]]; then
  # Print contents but redact any secrets
  awk 'BEGIN { IGNORECASE=1 } !/secret|password|token|key/' /etc/default/orbit \
    | tee -a "${SUMMARY}" \
    | sed 's/^/    /'
else
  fail "/etc/default/orbit not found"
  OVERALL_EXIT=1
fi

# ══════════════════════════════════════════════════════════════════════════════
# 5. ORBIT VERSION
# ══════════════════════════════════════════════════════════════════════════════
log ""
log "── 5. Orbit version ────────────────────────────────────────"
if command -v orbit >/dev/null 2>&1; then
  ORBIT_VERSION=$(orbit version 2>/dev/null || echo "unknown")
  info "${ORBIT_VERSION}"
  echo "${ORBIT_VERSION}" >> "${SUMMARY}"
else
  warn "orbit not found on PATH (/usr/local/bin/orbit missing or not in PATH)"
fi

# ══════════════════════════════════════════════════════════════════════════════
# 6. LOG COLLECTION
# ══════════════════════════════════════════════════════════════════════════════
log ""
log "── 6. Log collection ───────────────────────────────────────"

collect_log() {
  local label="$1"
  local src="$2"
  local dest_dir="$3"
  if [[ -f "${src}" ]]; then
    mkdir -p "${dest_dir}"
    cp "${src}" "${dest_dir}/"
    ok "Collected ${label}: ${src}"
  elif [[ -d "${src}" ]]; then
    mkdir -p "${dest_dir}"
    cp -r "${src}/." "${dest_dir}/"
    ok "Collected ${label} directory: ${src}"
  else
    warn "${label} not found at ${src} (skipping)"
  fi
}

# orbit logs — /var/log/orbit/ created at install time by linux_shared.go
# (usually empty; orbit's stdout/stderr goes to syslog/journal by default — see below)
collect_log "orbit logs"   "/var/log/orbit"   "${WORK_DIR}/logs/orbit"

# osquery logs — /var/log/osquery/ created at install time by linux_shared.go
# (usually empty; osqueryd's stdout/stderr goes to syslog/journal by default — see below)
collect_log "osquery logs" "/var/log/osquery" "${WORK_DIR}/logs/osquery"

# osquery filesystem logger output — only populated when logger_path/logger_plugin
# is set to "filesystem" in agent options. This is where result/status logs
# (osqueryd.INFO*, osqueryd.results.log, osqueryd.snapshots.log, etc.) actually live.
collect_log "osquery filesystem logger" "/opt/orbit/osquery_log" "${WORK_DIR}/logs/osquery_log"

# systemd journal for orbit.service (last 500 lines)
mkdir -p "${WORK_DIR}/logs"
if command -v journalctl >/dev/null 2>&1; then
  journalctl -u orbit.service --no-pager -n 500 \
    > "${WORK_DIR}/logs/orbit_journal.log" 2>&1 && \
    ok "Collected systemd journal (last 500 lines)" || \
    warn "journalctl failed for orbit.service"
fi

# syslog fallback — orbit/osqueryd stderr goes to syslog on Debian/Ubuntu
for syslog_path in /var/log/syslog /var/log/messages; do
  if [[ -f "${syslog_path}" ]]; then
    grep -i "orbit\|osquery\|fleet" "${syslog_path}" \
      > "${WORK_DIR}/logs/syslog_orbit_grep.log" 2>/dev/null || true
    ok "Grepped syslog for orbit/osquery/fleet: ${syslog_path}"
    break
  fi
done

# ══════════════════════════════════════════════════════════════════════════════
# 7. SYSTEM EVENTS (lookback window)
# ══════════════════════════════════════════════════════════════════════════════
# Captures what changed on the box before the user noticed a problem — reboots,
# package installs/upgrades/removals, systemd failures, and journal errors.
# Users rarely remember every change; having this saves a round trip of
# clarifying questions when triaging a report.
log ""
log "── 7. System events, last ${LOOKBACK_HOURS}h ───────────────"

mkdir -p "${WORK_DIR}/logs/system_events"
SINCE_TS=$(date -d "-${LOOKBACK_HOURS} hours" '+%Y-%m-%d %H:%M:%S')

# Reboots/shutdowns
if command -v last >/dev/null 2>&1; then
  last -x reboot shutdown -F 2>/dev/null | head -20 \
    > "${WORK_DIR}/logs/system_events/reboots.log" || true
  ok "Collected reboot/shutdown history"
fi

# Package installs/upgrades/removals (dpkg.log lines are ISO-timestamped,
# so a lexicographic compare against SINCE_TS also orders them chronologically)
if [[ -f /var/log/dpkg.log ]]; then
  awk -v since="${SINCE_TS}" '$0 >= since' /var/log/dpkg.log \
    > "${WORK_DIR}/logs/system_events/dpkg_recent.log" || true
  RECENT_PKG_COUNT=$(grep -cE ' (install|upgrade|remove|purge) ' \
    "${WORK_DIR}/logs/system_events/dpkg_recent.log" 2>/dev/null || echo 0)
  if [[ "${RECENT_PKG_COUNT}" -gt 0 ]]; then
    warn "${RECENT_PKG_COUNT} package install/upgrade/remove event(s) in last ${LOOKBACK_HOURS}h"
  else
    ok "No package install/upgrade/remove events in last ${LOOKBACK_HOURS}h"
  fi
fi

# Currently-failed systemd units (unrelated failures often explain "it stopped working")
FAILED_UNITS=$(systemctl --failed --no-legend 2>/dev/null || true)
echo "${FAILED_UNITS}" > "${WORK_DIR}/logs/system_events/systemd_failed_units.log"
if [[ -n "${FAILED_UNITS}" ]]; then
  warn "systemd reports failed units:"
  echo "${FAILED_UNITS}" | tee -a "${SUMMARY}" | sed 's/^/    /'
else
  ok "No failed systemd units"
fi

# Journal errors (priority err and above) in the window
if command -v journalctl >/dev/null 2>&1; then
  journalctl -p 3 --since "${LOOKBACK_HOURS} hours ago" --no-pager \
    > "${WORK_DIR}/logs/system_events/journal_errors.log" 2>&1 || true
  JOURNAL_ERR_COUNT=$(wc -l < "${WORK_DIR}/logs/system_events/journal_errors.log" 2>/dev/null || echo 0)
  if [[ "${JOURNAL_ERR_COUNT}" -gt 0 ]]; then
    warn "${JOURNAL_ERR_COUNT} journal error-level line(s) in last ${LOOKBACK_HOURS}h (see journal_errors.log)"
  else
    ok "No journal errors in last ${LOOKBACK_HOURS}h"
  fi

  # OOM killer events — a frequent, easily-missed cause of "it just stopped"
  OOM_HITS=$(journalctl --since "${LOOKBACK_HOURS} hours ago" --no-pager 2>/dev/null \
    | grep -i "out of memory\|oom-killer" || true)
  if [[ -n "${OOM_HITS}" ]]; then
    warn "OOM killer activity detected in last ${LOOKBACK_HOURS}h"
    echo "${OOM_HITS}" > "${WORK_DIR}/logs/system_events/oom_events.log"
  fi
fi

# ══════════════════════════════════════════════════════════════════════════════
# 8. PACKAGE THE ARCHIVE
# ══════════════════════════════════════════════════════════════════════════════
log ""
log "── 8. Packaging archive ────────────────────────────────────"

ARCHIVE_PATH="/tmp/${ARCHIVE_NAME}.tar.gz"
tar -czf "${ARCHIVE_PATH}" -C "$(dirname "${WORK_DIR}")" "$(basename "${WORK_DIR}")"
rm -rf "${WORK_DIR}"

info "Archive created: ${ARCHIVE_PATH}"
log ""

# ══════════════════════════════════════════════════════════════════════════════
# FINAL RESULT
# ══════════════════════════════════════════════════════════════════════════════
log "============================================================"
if [[ ${OVERALL_EXIT} -eq 0 ]]; then
  log " Result: ALL CHECKS PASSED"
else
  log " Result: ONE OR MORE CHECKS FAILED — review summary above"
fi
log " Archive: ${ARCHIVE_PATH}"
log "============================================================"

exit ${OVERALL_EXIT}
