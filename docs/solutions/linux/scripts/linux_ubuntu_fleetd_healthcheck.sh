#!/usr/bin/env bash
# linux_ubuntu_fleetd_healthcheck.sh
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

# No `set -e`: a diagnostic script should keep going and report as much as it
# can, rather than abort on the first unexpected non-zero exit.
set -uo pipefail

# ── Output helpers ─────────────────────────────────────────────────────────────
# Colour only when writing to a terminal, so pasted/redirected output stays clean.
if [[ -t 1 ]]; then
  RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; NC='\033[0m'
else
  RED=''; GREEN=''; YELLOW=''; NC=''
fi
# Every line shown to the user also goes to summary.txt (without colour codes).
status() { printf "  ${2}%-6s${NC}  %s\n" "$1" "$3"; printf "  %-6s  %s\n" "$1" "$3" >> "${SUMMARY}"; }
log()  { printf '%s\n' "$*"; printf '%s\n' "$*" >> "${SUMMARY}"; }
ok()   { status "[OK]"   "${GREEN}"  "$*"; }
warn() { status "[WARN]" "${YELLOW}" "$*"; }
fail() { status "[FAIL]" "${RED}"    "$*"; }
info() { status "[INFO]" ""          "$*"; }
indent() { sed 's/^/    /' | tee -a "${SUMMARY}"; }

# /etc/default/orbit read the way systemd's EnvironmentFile reads it, never
# sourced: backslash continuations joined, KEY=value lines only, surrounding
# quotes removed. orbit_env prints one variable's value (the last one wins).
ENV_FILE=/etc/default/orbit
orbit_env_lines() {
  [[ -f "${ENV_FILE}" ]] || return 0
  awk '{ if (sub(/\\$/, "")) { buf = buf $0; next } print buf $0; buf = "" } END { if (buf != "") print buf }' "${ENV_FILE}" \
    | sed -n -E 's/^[[:space:]]*(export[[:space:]]+)?([A-Za-z_][A-Za-z0-9_]*)=[[:space:]]*(.*)$/\2=\3/p' \
    | sed -E "s/^([^=]*)=\"(.*)\"[[:space:]]*\$/\\1=\\2/; s/^([^=]*)='(.*)'[[:space:]]*\$/\\1=\\2/"
}
orbit_env() { orbit_env_lines | sed -n "s/^$1=//p" | tail -1; }
# orbit reads booleans with Go's strconv.ParseBool: 1, t, true in any case.
is_true() { [[ "${1,,}" =~ ^(1|t|true)$ ]]; }

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

# PID and executable only: command lines can be long, and a manually started
# orbit can carry --enroll-secret.
show_pids() {
  local pid
  for pid in $1; do
    printf '%s %s\n' "${pid}" "$(readlink "/proc/${pid}/exe" 2>/dev/null)"
  done | indent
}

# PIDs from pgrep whose executable (not just command line) starts with $2. An
# editor with a file under /opt/orbit open would otherwise count as running.
pids_with_exe() {
  local pid exe out=""
  for pid in $1; do
    exe=$(readlink "/proc/${pid}/exe" 2>/dev/null) || continue
    [[ "${exe}" == "$2"* ]] && out="${out} ${pid}"
  done
  echo "${out# }"
}

check_process() {
  local label="$1"
  local prefix="$2"
  local pids
  pids=$(pids_with_exe "$(pgrep -f "${prefix}" 2>/dev/null || true)" "${prefix}")
  if [[ -n "${pids}" ]]; then
    ok "${label} is running"
    show_pids "${pids}"
  else
    fail "${label} is NOT running (no process running from ${prefix})"
    OVERALL_EXIT=1
  fi
}

# orbit binary path is /opt/orbit/bin/orbit/orbit
# Source: linux_shared.go ExecStart=/opt/orbit/bin/orbit/orbit
# After an auto-update, orbit can run from bin/orbit/<platform>/<channel>/orbit,
# so match the folder rather than the bin/orbit/orbit launcher path.
check_process "orbit"        "/opt/orbit/bin/orbit/"

# Match orbit's own osqueryd only: another osquery install must not count as
# fleetd being healthy.
check_process "osqueryd (fleetd)" "/opt/orbit/bin/osqueryd/"

# Classify by executable path; skip a process that exited before the lookup.
OTHER_OSQUERYD=""
for pid in $(pgrep -x osqueryd 2>/dev/null || true); do
  exe=$(readlink "/proc/${pid}/exe" 2>/dev/null) || continue
  [[ -z "${exe}" || "${exe}" == /opt/orbit/bin/osqueryd/* ]] || OTHER_OSQUERYD="${OTHER_OSQUERYD} ${pid}"
done
if [[ -n "${OTHER_OSQUERYD// /}" ]]; then
  warn "osqueryd running from outside /opt/orbit (another osquery install?):"
  show_pids "${OTHER_OSQUERYD}"
fi

# fleet-desktop: only present if package was built with --fleet-desktop
# Source: constant.DesktopAppExecName = "fleet-desktop"
DESKTOP_PIDS=$(pids_with_exe "$(pgrep -f "fleet-desktop" 2>/dev/null || true)" "/opt/orbit/bin/desktop/")
if [[ -n "${DESKTOP_PIDS}" ]]; then
  ok "fleet-desktop is running"
  show_pids "${DESKTOP_PIDS}"
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
# The .deb/.rpm put the enroll secret in /etc/default/orbit (ORBIT_ENROLL_SECRET),
# not in a file. Once orbit has a node key, it doesn't need the secret.
SECRET_PATH=$(orbit_env ORBIT_ENROLL_SECRET_PATH)
ENROLLED=0; [[ -s /opt/orbit/secret-orbit-node-key.txt ]] && ENROLLED=1
if [[ -n "$(orbit_env ORBIT_ENROLL_SECRET)" ]]; then
  ok "enroll secret is configured (ORBIT_ENROLL_SECRET in /etc/default/orbit)"
elif [[ -n "${SECRET_PATH}" && -r "${SECRET_PATH}" && -s "${SECRET_PATH}" ]]; then
  ok "enroll secret is configured (${SECRET_PATH})"
elif [[ -z "${SECRET_PATH}" && -s /opt/orbit/secret.txt ]]; then
  ok "enroll secret is configured (/opt/orbit/secret.txt)"
elif [[ "${ENROLLED}" -eq 1 ]]; then
  info "enroll secret not found${SECRET_PATH:+ at ${SECRET_PATH}} (not needed: orbit is already enrolled)"
else
  fail "enroll secret not found${SECRET_PATH:+ at ${SECRET_PATH}} and orbit is not enrolled: check ORBIT_ENROLL_SECRET in /etc/default/orbit"
  OVERALL_EXIT=1
fi

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
  # Print parsed KEY=value pairs only (never raw lines, so a continued value
  # can't slip past), with secret-looking values redacted. Uppercase before
  # matching: IGNORECASE is gawk-only, and Ubuntu's default awk (mawk) ignores it.
  orbit_env_lines | awk '{
      name = toupper(substr($0, 1, index($0, "=") - 1))
      if (name ~ /SECRET|PASSWORD|TOKEN|KEY/ && name !~ /_PATH$/) print substr($0, 1, index($0, "=")) "<redacted>"
      else print
    }' | indent
  ORBIT_FLEET_URL=$(orbit_env ORBIT_FLEET_URL)
  FLEET_CERT=$(orbit_env ORBIT_FLEET_CERTIFICATE)
  FLEET_INSECURE=$(orbit_env ORBIT_INSECURE)
  if [[ -n "${ORBIT_FLEET_URL}" ]]; then
    # Probe the server the way orbit connects: https:// when no scheme is given,
    # the package's own CA file when set, and no verification when insecure.
    [[ "${ORBIT_FLEET_URL}" == http* ]] || ORBIT_FLEET_URL="https://${ORBIT_FLEET_URL}"
    CURL_TLS=()
    [[ -n "${FLEET_CERT}" && -f "${FLEET_CERT}" ]] && CURL_TLS+=(--cacert "${FLEET_CERT}")
    is_true "${FLEET_INSECURE}" && CURL_TLS+=(-k)
    HTTP_CODE=$(curl -sS -o /dev/null -w '%{http_code}' --max-time 15 "${CURL_TLS[@]+"${CURL_TLS[@]}"}" "${ORBIT_FLEET_URL%/}/healthz" 2>"${WORK_DIR}/curl_error.txt" || true)
    if [[ "${HTTP_CODE}" =~ ^[1-5][0-9][0-9]$ ]]; then
      ok "Fleet server reachable: ${ORBIT_FLEET_URL} (HTTP ${HTTP_CODE} from /healthz)"
    else
      fail "Cannot reach Fleet server ${ORBIT_FLEET_URL}: $(head -1 "${WORK_DIR}/curl_error.txt")"
      OVERALL_EXIT=1
    fi
  fi
else
  fail "/etc/default/orbit not found"
  OVERALL_EXIT=1
fi

# ══════════════════════════════════════════════════════════════════════════════
# 5. ORBIT VERSION
# ══════════════════════════════════════════════════════════════════════════════
log ""
log "── 5. Orbit version ────────────────────────────────────────"
# Call the real binary rather than the PATH symlink, so a missing symlink
# doesn't hide the version.
if [[ -x /opt/orbit/bin/orbit/orbit ]]; then
  /opt/orbit/bin/orbit/orbit version 2>/dev/null | indent
else
  warn "orbit binary not found at /opt/orbit/bin/orbit/orbit"
fi
# Package manager's view of the install (deb or rpm)
if command -v dpkg-query >/dev/null 2>&1; then
  dpkg-query -W -f='${Package} ${Version} ${Status}\n' 2>/dev/null | grep -i -E 'fleet|orbit' | indent
elif command -v rpm >/dev/null 2>&1; then
  rpm -qa 2>/dev/null | grep -i -E 'fleet|orbit' | indent
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
  # A log can rotate between the check and the copy, so report copy failures.
  if [[ -f "${src}" ]]; then
    mkdir -p "${dest_dir}"
    if cp "${src}" "${dest_dir}/" 2>/dev/null; then
      ok "Collected ${label}: ${src}"
    else
      warn "Failed to copy ${label}: ${src}"
    fi
  elif [[ -d "${src}" ]]; then
    mkdir -p "${dest_dir}"
    if cp -r "${src}/." "${dest_dir}/" 2>/dev/null; then
      ok "Collected ${label} directory: ${src}"
    else
      warn "Collected ${label} directory only partly (some files failed to copy): ${src}"
    fi
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
    # grep exits 1 for no matches, 2 when it can't read the file.
    grep -i "orbit\|osquery\|fleet" "${syslog_path}" \
      > "${WORK_DIR}/logs/syslog_orbit_grep.log" 2>/dev/null
    if [[ $? -le 1 ]]; then
      ok "Grepped syslog for orbit/osquery/fleet: ${syslog_path}"
    else
      warn "Could not read ${syslog_path}"
    fi
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
  if [[ -s "${WORK_DIR}/logs/system_events/reboots.log" ]]; then
    ok "Collected reboot/shutdown history"
  else
    warn "No reboot/shutdown history from \`last\`"
  fi
fi

# Package installs/upgrades/removals (dpkg.log lines are ISO-timestamped,
# so a lexicographic compare against SINCE_TS also orders them chronologically)
if [[ -f /var/log/dpkg.log ]]; then
  awk -v since="${SINCE_TS}" '$0 >= since' /var/log/dpkg.log \
    > "${WORK_DIR}/logs/system_events/dpkg_recent.log" || true
  # grep -c prints 0 *and* exits 1 on no match, so `|| echo 0` would yield "0\n0".
  RECENT_PKG_COUNT=$(grep -cE ' (install|upgrade|remove|purge) ' \
    "${WORK_DIR}/logs/system_events/dpkg_recent.log" 2>/dev/null || true)
  if [[ "${RECENT_PKG_COUNT:-0}" -gt 0 ]]; then
    warn "${RECENT_PKG_COUNT} package install/upgrade/remove event(s) in last ${LOOKBACK_HOURS}h"
  else
    ok "No package install/upgrade/remove events in last ${LOOKBACK_HOURS}h"
  fi
elif command -v rpm >/dev/null 2>&1; then
  # RHEL/Fedora: no dpkg.log; list packages by install time instead
  rpm -qa --last 2>/dev/null | head -50 > "${WORK_DIR}/logs/system_events/rpm_recent.log" || true
  ok "Collected recent rpm installs (rpm_recent.log)"
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
  # -q drops "-- No entries --", which would otherwise count as one error line
  journalctl -q -p 3 --since "${LOOKBACK_HOURS} hours ago" --no-pager \
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
# FINAL RESULT AND ARCHIVE
# ══════════════════════════════════════════════════════════════════════════════
# The result is written before archiving so summary.txt in the archive is
# complete, and nothing writes to WORK_DIR after it's removed.
ARCHIVE_PATH="/tmp/${ARCHIVE_NAME}.tar.gz"

log ""
log "============================================================"
if [[ ${OVERALL_EXIT} -eq 0 ]]; then
  log " Result: ALL CHECKS PASSED"
else
  log " Result: ONE OR MORE CHECKS FAILED — review summary above"
fi
log "============================================================"

if tar -czf "${ARCHIVE_PATH}" -C "$(dirname "${WORK_DIR}")" "$(basename "${WORK_DIR}")"; then
  rm -rf "${WORK_DIR}"
  # Logs can contain usernames, URLs, and emails: keep the archive private to
  # the admin who ran the script rather than world-readable in /tmp.
  chmod 600 "${ARCHIVE_PATH}"
  [[ -n "${SUDO_USER:-}" ]] && chown "${SUDO_USER}" "${ARCHIVE_PATH}" 2>/dev/null
  echo " Archive: ${ARCHIVE_PATH}"
  echo " Send this file to Fleet support."
else
  echo " Archive creation failed; diagnostic data kept at: ${WORK_DIR}" >&2
  OVERALL_EXIT=1
fi
echo "============================================================"

exit ${OVERALL_EXIT}
