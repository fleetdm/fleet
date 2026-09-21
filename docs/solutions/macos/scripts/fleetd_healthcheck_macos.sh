#!/usr/bin/env bash
# fleetd_healthcheck_macos.sh
#
# Checks the health of all fleetd components on macOS and collects logs into
# a timestamped archive for support/troubleshooting. macOS counterpart to
# fleetd_healthcheck_ubuntu.sh / fleetd_healthcheck_windows.ps1.
#
# Components checked:
#   - com.fleetdm.orbit   (LaunchDaemon, runs orbit as root)
#   - orbit               (process: /opt/orbit/bin/orbit/orbit)
#   - osqueryd            (process: spawned and managed by orbit, runs from an
#                          app bundle under /opt/orbit/bin/osqueryd/macos-app/)
#   - fleet-desktop       (process: optional, only present if packaged with
#                          --fleet-desktop; started by orbit, not its own LaunchAgent)
#
# Sources:
#   - LaunchDaemon:        /Library/LaunchDaemons/com.fleetdm.orbit.plist
#                          (Label com.fleetdm.orbit, ProgramArguments
#                          /opt/orbit/bin/orbit/orbit)
#   - Install root:        /opt/orbit (orbit/CHANGELOG.md — installer paths)
#   - Orbit symlink:       /usr/local/bin/orbit
#   - Enroll secret:       /opt/orbit/secret.txt (ORBIT_ENROLL_SECRET_PATH).
#                          On macOS this is removed after first successful
#                          enrollment and the secret is kept in the login
#                          keychain instead — so an absent file is expected,
#                          not a failure. https://fleetdm.com/guides/fleetd-authentication
#   - Orbit node key:      /opt/orbit/secret-orbit-node-key.txt
#   - Log paths:           https://fleetdm.com/guides/fleet-troubleshooting-for-it-admins
#                            - orbit stdout/stderr: /var/log/orbit/orbit.stdout.log,
#                              /var/log/orbit/orbit.stderr.log (set in the LaunchDaemon plist)
#                            - osquery filesystem logger: /opt/orbit/osquery_log
#                            - fleet-desktop: $HOME/Library/Logs/Fleet/fleet-desktop.log (per user)
#   - Profile dumps:       /opt/orbit/profiles (pkill -USR1 orbit)
#
# Also collects a lookback window (default 72h, override with LOOKBACK_HOURS env
# var) of system events — reboots, installer activity, launchd job failures,
# unified log errors — to help correlate a reported problem with what changed
# beforehand.
#
# Must be run as root (sudo).

set -euo pipefail

# ── Colour helpers ─────────────────────────────────────────────────────────────
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; NC='\033[0m'
ok()   { echo -e "  ${GREEN}[OK]${NC}    $*"; }
warn() { echo -e "  ${YELLOW}[WARN]${NC}  $*"; }
fail() { echo -e "  ${RED}[FAIL]${NC}  $*"; }
info() { echo -e "  [INFO]  $*"; }

if [[ $EUID -ne 0 ]]; then
  echo "This script must be run as root (sudo)." >&2
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

ORBIT_ROOT="/opt/orbit"
LAUNCH_DAEMON_LABEL="com.fleetdm.orbit"
LAUNCH_DAEMON_PLIST="/Library/LaunchDaemons/${LAUNCH_DAEMON_LABEL}.plist"

# ── Header ─────────────────────────────────────────────────────────────────────
log "============================================================"
log " Fleet fleetd Health Check"
log " Host:      $(hostname)"
log " Date:      $(date)"
log " macOS:     $(sw_vers -productName) $(sw_vers -productVersion) ($(sw_vers -buildVersion))"
log " Kernel:    $(uname -r)"
log "============================================================"
log ""

# ══════════════════════════════════════════════════════════════════════════════
# 1. LAUNCHDAEMON
# ══════════════════════════════════════════════════════════════════════════════
log "── 1. LaunchDaemon (${LAUNCH_DAEMON_LABEL}) ─────────────────"

if [[ -f "${LAUNCH_DAEMON_PLIST}" ]]; then
  ok "LaunchDaemon plist found: ${LAUNCH_DAEMON_PLIST}"
else
  fail "LaunchDaemon plist not found: ${LAUNCH_DAEMON_PLIST}"
  OVERALL_EXIT=1
fi

LAUNCHCTL_PRINT=$(launchctl print "system/${LAUNCH_DAEMON_LABEL}" 2>&1 || true)
if echo "${LAUNCHCTL_PRINT}" | grep -q "state = running"; then
  ok "${LAUNCH_DAEMON_LABEL} is running"
elif [[ -n "${LAUNCHCTL_PRINT}" ]] && ! echo "${LAUNCHCTL_PRINT}" | grep -qi "could not find"; then
  fail "${LAUNCH_DAEMON_LABEL} is loaded but not running"
  OVERALL_EXIT=1
else
  fail "${LAUNCH_DAEMON_LABEL} is NOT loaded in launchd"
  OVERALL_EXIT=1
fi
echo "${LAUNCHCTL_PRINT}" >> "${SUMMARY}"

if [[ -f "${LAUNCH_DAEMON_PLIST}" ]]; then
  RUN_AT_LOAD=$(/usr/libexec/PlistBuddy -c "Print :RunAtLoad" "${LAUNCH_DAEMON_PLIST}" 2>/dev/null || echo "unknown")
  KEEP_ALIVE=$(/usr/libexec/PlistBuddy -c "Print :KeepAlive" "${LAUNCH_DAEMON_PLIST}" 2>/dev/null || echo "unknown")
  if [[ "${RUN_AT_LOAD}" == "true" ]]; then
    ok "RunAtLoad is true — will start on boot"
  else
    warn "RunAtLoad is '${RUN_AT_LOAD}' — may not start on boot"
  fi
  info "KeepAlive: ${KEEP_ALIVE}"
fi

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

# orbit binary path is /opt/orbit/bin/orbit/orbit, runs as root
check_process "orbit"    "${ORBIT_ROOT}/bin/orbit/orbit"

# osqueryd runs from an app bundle orbit manages under bin/osqueryd/macos-app/
check_process "osqueryd" "osqueryd"

# fleet-desktop: only present if package was built with --fleet-desktop;
# started by orbit itself rather than its own LaunchAgent
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

check_file "orbit binary"     "${ORBIT_ROOT}/bin/orbit/orbit"
check_file "orbit symlink"    "/usr/local/bin/orbit"
check_file "osquery pidfile"  "${ORBIT_ROOT}/osquery.pid"

# Enroll secret is intentionally removed from disk after first enrollment on
# macOS (moved to the login keychain), so its absence is normal — only flag
# it if orbit has never enrolled (no node key either, see below).
if [[ -f "${ORBIT_ROOT}/secret.txt" ]]; then
  info "enroll secret still on disk: ${ORBIT_ROOT}/secret.txt (expected before first enrollment)"
else
  info "enroll secret not on disk (expected after enrollment — stored in keychain)"
fi

# Report orbit node key presence without printing value
if [[ -s "${ORBIT_ROOT}/secret-orbit-node-key.txt" ]]; then
  ok "orbit node key is non-empty (enrolled)"
else
  fail "orbit node key is missing or empty (not enrolled): ${ORBIT_ROOT}/secret-orbit-node-key.txt"
  OVERALL_EXIT=1
fi

# ══════════════════════════════════════════════════════════════════════════════
# 4. LAUNCHDAEMON CONFIGURATION SUMMARY (REDACTED)
# ══════════════════════════════════════════════════════════════════════════════
log ""
log "── 4. LaunchDaemon configuration (redacted) ────────────────"
if [[ -f "${LAUNCH_DAEMON_PLIST}" ]]; then
  # Convert to XML text and redact anything that looks like a secret/token/key
  plutil -convert xml1 -o - "${LAUNCH_DAEMON_PLIST}" 2>/dev/null \
    | grep -v -i "secret\|password\|token" \
    | tee -a "${SUMMARY}" \
    | sed 's/^/    /' >/dev/null || true
  ok "Collected redacted LaunchDaemon plist contents"
else
  fail "${LAUNCH_DAEMON_PLIST} not found"
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

# orbit stdout/stderr — paths set via StandardOutPath/StandardErrorPath in the
# LaunchDaemon plist
collect_log "orbit stdout log" "/var/log/orbit/orbit.stdout.log" "${WORK_DIR}/logs/orbit"
collect_log "orbit stderr log" "/var/log/orbit/orbit.stderr.log" "${WORK_DIR}/logs/orbit"

# osquery filesystem logger output — only populated when logger_path/logger_plugin
# is set to "filesystem" in agent options.
collect_log "osquery filesystem logger" "${ORBIT_ROOT}/osquery_log" "${WORK_DIR}/logs/osquery_log"

# fleet-desktop runs per-user; sweep all user home directories for its log file
DESKTOP_LOGS_FOUND=0
for user_home in /Users/*; do
  [[ -d "${user_home}" ]] || continue
  desktop_log="${user_home}/Library/Logs/Fleet/fleet-desktop.log"
  if [[ -f "${desktop_log}" ]]; then
    user_name=$(basename "${user_home}")
    dest_dir="${WORK_DIR}/logs/fleet-desktop/${user_name}"
    mkdir -p "${dest_dir}"
    cp "${desktop_log}" "${dest_dir}/"
    ok "Collected fleet-desktop log for user ${user_name}: ${desktop_log}"
    DESKTOP_LOGS_FOUND=1
  fi
done
if [[ "${DESKTOP_LOGS_FOUND}" -eq 0 ]]; then
  warn "No fleet-desktop.log found under any user's Library/Logs/Fleet (expected if fleet-desktop is not installed)"
fi

# Unified log entries mentioning orbit/osquery/fleet in the lookback window.
# `log show` over a wide window can take a while and produce a large file, so
# this is bounded to the same LOOKBACK_HOURS as the system-events section.
mkdir -p "${WORK_DIR}/logs/unified_log"
if log show --predicate 'eventMessage contains "orbit" or eventMessage contains "osquery" or eventMessage contains "fleet"' \
    --style syslog --last "${LOOKBACK_HOURS}h" \
    > "${WORK_DIR}/logs/unified_log/orbit_osquery_fleet.log" 2>&1; then
  ok "Collected unified log entries mentioning orbit/osquery/fleet (last ${LOOKBACK_HOURS}h)"
else
  warn "Failed to query unified log (log show) — see unified_log/orbit_osquery_fleet.log for details"
fi

# ══════════════════════════════════════════════════════════════════════════════
# 7. SYSTEM EVENTS (lookback window)
# ══════════════════════════════════════════════════════════════════════════════
# Captures what changed on the box before the user noticed a problem — reboots,
# installer activity, launchd job failures, and unified log errors. Users
# rarely remember every change; having this saves a round trip of clarifying
# questions when triaging a report.
log ""
log "── 7. System events, last ${LOOKBACK_HOURS}h ───────────────"

mkdir -p "${WORK_DIR}/logs/system_events"

# Reboots/shutdowns
if command -v last >/dev/null 2>&1; then
  last -x reboot shutdown 2>/dev/null | head -20 \
    > "${WORK_DIR}/logs/system_events/reboots.log" || true
  ok "Collected reboot/shutdown history"
fi

# Installer activity (pkg installs/removals go through installd)
if log show --predicate 'process == "installd"' --style syslog --last "${LOOKBACK_HOURS}h" \
    > "${WORK_DIR}/logs/system_events/installer_activity.log" 2>&1; then
  RECENT_PKG_COUNT=$(grep -cE 'PackageKit|Install|Removed' "${WORK_DIR}/logs/system_events/installer_activity.log" 2>/dev/null || echo 0)
  if [[ "${RECENT_PKG_COUNT}" -gt 0 ]]; then
    warn "${RECENT_PKG_COUNT} installer-related log line(s) in last ${LOOKBACK_HOURS}h (see installer_activity.log)"
  else
    ok "No installer activity in last ${LOOKBACK_HOURS}h"
  fi
else
  warn "Failed to query installer activity via log show"
fi

# launchd jobs with a non-zero/unknown last exit status (unrelated failures
# often explain "it stopped working")
FAILED_JOBS=$(launchctl list | awk 'NR>1 && $2 != "0" && $2 != "-" {print}' || true)
echo "${FAILED_JOBS}" > "${WORK_DIR}/logs/system_events/launchd_failed_jobs.log"
if [[ -n "${FAILED_JOBS}" ]]; then
  warn "launchd reports jobs with non-zero last exit status:"
  echo "${FAILED_JOBS}" | tee -a "${SUMMARY}" | sed 's/^/    /'
else
  ok "No launchd jobs with a non-zero last exit status"
fi

# Unified log errors/faults in the window
if log show --predicate 'messageType == error OR messageType == fault' --style syslog --last "${LOOKBACK_HOURS}h" \
    > "${WORK_DIR}/logs/system_events/log_errors.log" 2>&1; then
  ERR_COUNT=$(wc -l < "${WORK_DIR}/logs/system_events/log_errors.log" 2>/dev/null || echo 0)
  if [[ "${ERR_COUNT}" -gt 0 ]]; then
    warn "${ERR_COUNT} error/fault-level log line(s) in last ${LOOKBACK_HOURS}h (see log_errors.log)"
  else
    ok "No error/fault-level log entries in last ${LOOKBACK_HOURS}h"
  fi

  # Memory-pressure / jetsam kills — macOS' equivalent of the Linux OOM killer
  MEMORY_HITS=$(grep -i "jetsam\|memorystatus" "${WORK_DIR}/logs/system_events/log_errors.log" 2>/dev/null || true)
  if [[ -n "${MEMORY_HITS}" ]]; then
    warn "Memory-pressure/jetsam activity detected in last ${LOOKBACK_HOURS}h"
    echo "${MEMORY_HITS}" > "${WORK_DIR}/logs/system_events/memory_pressure_events.log"
  fi
else
  warn "Failed to query unified log for errors/faults"
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
