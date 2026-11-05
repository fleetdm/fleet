#!/bin/bash
# mitigate-php2shell-linux.sh
# PHP's interim mitigation for PHP2Shell (CVE-2026-109618, CVE-2026-109619) on hosts that
# can't update PHP yet. Writes 99-php2shell-mitigation.ini into every PHP configuration
# directory it finds, with:
#   session.upload_progress.enabled = 0   closes the confirmed upload-progress paths (CVE-2026-109619)
#   fatal_error_backtraces = 0            PHP 8.5 only; older releases ignore it
# then restarts PHP-FPM, and Apache when mod_php is loaded, because php.ini is read at startup.
# Partial by design: it doesn't cover the out-of-memory path (CVE-2026-109618). PHP's
# memory_limit=-1 guidance needs a cgroup limit and php_admin_value, so it is left to you.
# Updating PHP is the only complete fix.
# Idempotent: safe to re-run. Built for Fleet "Run script" (runs as root). Needs bash.
# Rollback: delete the 99-php2shell-mitigation.ini files and restart the same services.
# Exit: 0 = applied or nothing to do, 2 = applied with warnings (read output), 1 = error.
# Advisory: https://github.com/php/php-src/security/advisories/GHSA-6wmf-4rv5-p3rp

set -u
export PATH="/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"
DROPIN="99-php2shell-mitigation.ini"
EXIT=0

log()  { echo "$*"; }
warn() { echo "WARN: $*"; EXIT=2; }
die()  { echo "ERROR: $*" >&2; exit 1; }

[ "$(id -u)" -eq 0 ] || die "must run as root"

read -r -d '' CONTENT <<'INI' || true
; PHP2Shell interim mitigation (CVE-2026-109618, CVE-2026-109619). Not a substitute for updating PHP.
; https://github.com/php/php-src/security/advisories/GHSA-6wmf-4rv5-p3rp
session.upload_progress.enabled = 0
fatal_error_backtraces = 0
INI

# Debian and Ubuntu (/etc/php/X.Y/<sapi>/conf.d), RHEL, Fedora, and Amazon (/etc/php.d),
# SUSE (/etc/php8/conf.d), Alpine (/etc/php83/conf.d), Remi, cPanel, CloudLinux, LiteSpeed,
# and the default scan directory of a source build.
conf_dirs() {
    local d
    for d in /etc/php/*/*/conf.d /etc/php.d /etc/php[0-9]/conf.d /etc/php[0-9][0-9]/conf.d \
             /etc/opt/remi/php*/php.d /opt/cpanel/ea-php*/root/etc/php.d /opt/alt/php*/etc/php.d \
             /usr/local/lsws/lsphp*/etc/php.d /usr/local/etc/php/conf.d; do
        [ -d "$d" ] && echo "$d"
    done
}

write_dropin() {
    local f="$1/$DROPIN"
    if [ -f "$f" ] && [ "$(cat "$f")" = "$CONTENT" ]; then
        log "UNCHANGED: $f"
        return 1
    fi
    printf '%s\n' "$CONTENT" > "$f" || { warn "cannot write $f"; return 1; }
    chmod 0644 "$f"
    log "WROTE: $f"
    return 0
}

apache_uses_mod_php() {
    local pid
    for pid in $(pgrep -x 'apache2|httpd|httpd-prefork' 2>/dev/null); do
        grep -qs 'libphp' "/proc/$pid/maps" && return 0
    done
    return 1
}

restart_workers() {
    local unit
    for unit in $(systemctl list-units --type=service --state=active --plain --no-legend 2>/dev/null \
                  | awk '{ print $1 }' | grep -E '^(ea-|alt-)?php[0-9.]*(-php)?-fpm\.service$'); do
        if systemctl restart "$unit" >/dev/null 2>&1; then log "RESTARTED: $unit"; else warn "failed to restart $unit"; fi
    done
    if apache_uses_mod_php; then
        for unit in apache2 httpd; do
            systemctl is-active --quiet "$unit" 2>/dev/null || continue
            if systemctl restart "$unit" >/dev/null 2>&1; then log "RESTARTED: $unit (mod_php)"; else warn "failed to restart $unit"; fi
        done
    fi
    if pgrep -x lsphp >/dev/null 2>&1; then
        pkill -x lsphp && log "RESTARTED: LiteSpeed lsphp workers (the web server respawns them)"
    fi
    pgrep -x php-cgi >/dev/null 2>&1 && warn "php-cgi processes are running; restart the FastCGI spawner so they read the new setting"
    pgrep -x frankenphp >/dev/null 2>&1 && warn "FrankenPHP is running; it reads its own php.ini, so apply the setting there and restart it"
    return 0
}

main() {
    local dirs d changed=0
    dirs=$(conf_dirs)
    if [ -z "$dirs" ]; then
        log "No PHP configuration directory found; nothing to do."
        exit 0
    fi
    log "----- writing $DROPIN -----"
    for d in $dirs; do
        write_dropin "$d" && changed=1
    done
    log "----- restarting PHP workers -----"
    if [ "$changed" -eq 1 ]; then
        restart_workers
    else
        log "Drop-ins already in place; workers not restarted."
    fi
    if command -v php >/dev/null 2>&1; then
        log "CLI check: $(php -r 'echo "session.upload_progress.enabled=", ini_get("session.upload_progress.enabled") === "0" || ini_get("session.upload_progress.enabled") === "" ? "0" : "1", "\n";' 2>/dev/null) (web SAPIs read the same drop-in after the restart)"
    fi
    if [ "$EXIT" -eq 0 ]; then
        log "DONE: interim mitigation applied. Update PHP as soon as a fixed package is available."
    else
        log "DONE with warnings (exit 2): read the lines above."
    fi
    exit "$EXIT"
}

main "$@"
