#!/bin/bash
# remediate-php2shell-linux.sh
# Update PHP for PHP2Shell (CVE-2026-109618, CVE-2026-109619, disclosed 2026-10-21).
#   1. Upgrades every installed PHP package through the distro package manager
#      (apt, dnf, yum, zypper, apk). Vendor backports keep the old upstream version
#      number, so the script reports full package versions and changelog CVE mentions,
#      never php -v.
#   2. Restarts PHP-FPM pools, Apache when mod_php is loaded, and LiteSpeed PHP workers,
#      because a running worker keeps the old code until it is restarted.
#   3. Flags PHP runtimes it can't update: containers, FrankenPHP, self-built PHP.
# Idempotent: safe to re-run. Built for Fleet "Run script" (runs as root). Needs bash.
# Exit: 0 = updated or nothing to do, 2 = something needs a look (read output), 1 = error.
# Advisories: https://github.com/php/php-src/security/advisories/GHSA-w8hw-w9fx-j79w
#             https://github.com/php/php-src/security/advisories/GHSA-6wmf-4rv5-p3rp

set -u
export PATH="/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"
export DEBIAN_FRONTEND=noninteractive NEEDRESTART_MODE=a
EXIT=0

log()  { echo "$*"; }
warn() { echo "WARN: $*"; EXIT=2; }
die()  { echo "ERROR: $*" >&2; exit 1; }

[ "$(id -u)" -eq 0 ] || die "must run as root"

# Installed PHP packages, one "name version" per line. Covers distro packages, Sury,
# Remi (php84-php-*), LiteSpeed (lsphp*), cPanel (ea-php*), and CloudLinux (alt-php*).
php_packages() {
    if command -v dpkg-query >/dev/null 2>&1; then
        dpkg-query -W -f='${binary:Package} ${Version} ${Status}\n' 'php*' 'libapache2-mod-php*' 'lsphp*' 2>/dev/null \
            | awk '$NF == "installed" { print $1, $2 }'
    elif command -v rpm >/dev/null 2>&1; then
        rpm -qa --qf '%{NAME} %{VERSION}-%{RELEASE}\n' 'php*' 'apache2-mod_php*' 'mod_php*' 'lsphp*' 'ea-php*' 'alt-php*' 2>/dev/null | sort
    elif command -v apk >/dev/null 2>&1; then
        apk list -I 2>/dev/null | awk '$1 ~ /^php/ { print $1 }'
    fi
}

upgrade_php() {
    local names
    if command -v apt-get >/dev/null 2>&1; then
        names=$(php_packages | awk '{ print $1 }' | tr '\n' ' ')
        apt-get update -qq >/dev/null 2>&1 || warn "apt-get update failed (repository or network issue)"
        # shellcheck disable=SC2086
        apt-get install --only-upgrade -y -qq -o Dpkg::Options::=--force-confdef -o Dpkg::Options::=--force-confold $names >/dev/null 2>&1 \
            || warn "apt-get upgrade of PHP packages failed"
    elif command -v dnf >/dev/null 2>&1; then
        dnf -y -q upgrade 'php*' 'lsphp*' 'ea-php*' 'alt-php*' >/dev/null 2>&1 || warn "dnf upgrade of PHP packages failed"
    elif command -v yum >/dev/null 2>&1; then
        yum -y -q update 'php*' 'lsphp*' 'ea-php*' 'alt-php*' >/dev/null 2>&1 || warn "yum update of PHP packages failed"
    elif command -v zypper >/dev/null 2>&1; then
        zypper -n -q refresh >/dev/null 2>&1 || true
        zypper -n -q update 'php*' >/dev/null 2>&1 || warn "zypper update of PHP packages failed"
    elif command -v apk >/dev/null 2>&1; then
        names=$(php_packages | sed 's/-[0-9].*//' | tr '\n' ' ')
        apk update -q >/dev/null 2>&1 || true
        # shellcheck disable=SC2086
        apk upgrade -q $names >/dev/null 2>&1 || warn "apk upgrade of PHP packages failed"
    else
        warn "no supported package manager found; update PHP manually"
    fi
}

# Debian and Ubuntu changelogs name the CVEs. RPM changelogs usually do too, except on some
# RHEL 8 and 9 module streams, where only the package version proves the fix.
changelog_mentions_fix() {
    local f p
    for f in /usr/share/doc/php*-common/changelog.Debian.gz /usr/share/doc/php*-common/changelog.gz; do
        [ -r "$f" ] || continue
        zgrep -q -m1 -E 'CVE-2026-10961[89]' "$f" 2>/dev/null && log "CHANGELOG: $f names CVE-2026-109618/109619"
    done
    if command -v rpm >/dev/null 2>&1; then
        for p in $(rpm -qa --qf '%{NAME}\n' 'php' 'php*-common' 2>/dev/null); do
            rpm -q --changelog "$p" 2>/dev/null | grep -q -m1 -E 'CVE-2026-10961[89]' && log "CHANGELOG: $p names CVE-2026-109618/109619"
        done
    fi
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
    # php-fpm.service (RHEL, SUSE, Alpine), phpX.Y-fpm.service (Debian, Ubuntu), php84-php-fpm.service (Remi),
    # ea-php84-php-fpm.service (cPanel), alt-php84-fpm.service (CloudLinux)
    for unit in $(systemctl list-units --type=service --state=active --plain --no-legend 2>/dev/null \
                  | awk '{ print $1 }' | grep -E '^(ea-|alt-)?php[0-9.]*(-php)?-fpm\.service$'); do
        if systemctl restart "$unit" >/dev/null 2>&1; then log "RESTARTED: $unit"; else warn "failed to restart $unit"; fi
    done
    if apache_uses_mod_php; then
        for unit in apache2 httpd; do
            systemctl is-active --quiet "$unit" 2>/dev/null || continue
            # mod_php needs a full restart: reload and graceful keep the old libphp mapped.
            if systemctl restart "$unit" >/dev/null 2>&1; then log "RESTARTED: $unit (mod_php)"; else warn "failed to restart $unit"; fi
        done
    fi
    if pgrep -x lsphp >/dev/null 2>&1; then
        pkill -x lsphp && log "RESTARTED: LiteSpeed lsphp workers (the web server respawns them)"
    fi
    pgrep -x php-cgi >/dev/null 2>&1 && warn "php-cgi processes are running; restart the FastCGI spawner (mod_fcgid, spawn-fcgi, or the web server) so they load the new build"
    return 0
}

# A worker whose binary or libphp was replaced on disk still runs the old code.
stale_workers() {
    local pid exe found=1
    for pid in $(pgrep -x 'php-fpm[0-9.]*|php[0-9][0-9]-php-fpm|apache2|httpd|httpd-prefork|php-cgi|lsphp' 2>/dev/null); do
        exe=$(readlink "/proc/$pid/exe" 2>/dev/null) || continue
        case "$exe" in
            *' (deleted)') log "STALE: pid $pid ($(cat "/proc/$pid/comm" 2>/dev/null)) runs a replaced binary: $exe"; found=0 ;;
        esac
        if grep -qsE 'libphp[^ ]*\.so[^ ]* \(deleted\)$' "/proc/$pid/maps"; then
            log "STALE: pid $pid ($(cat "/proc/$pid/comm" 2>/dev/null)) maps a replaced libphp"; found=0
        fi
    done
    return $found
}

other_runtimes() {
    local b imgs
    if command -v frankenphp >/dev/null 2>&1 || pgrep -x frankenphp >/dev/null 2>&1; then
        warn "FrankenPHP found ($(frankenphp version 2>/dev/null | head -1)); package managers don't update it. Install FrankenPHP 1.13.2 or newer and restart it."
    fi
    if command -v docker >/dev/null 2>&1; then
        imgs=$(docker ps --format '{{.Names}} {{.Image}}' 2>/dev/null | grep -iE 'php|wordpress|drupal|joomla|nextcloud|laravel|symfony|magento|mediawiki|frankenphp')
        if [ -n "$imgs" ]; then
            warn "running containers use PHP images; pull or rebuild them on a fixed base (php:8.4.27 and up) and recreate:"
            echo "$imgs" | sed 's/^/  CONTAINER: /'
        fi
    fi
    for b in /usr/local/bin/php /usr/local/sbin/php-fpm /opt/*/bin/php; do
        [ -x "$b" ] || continue
        if ! dpkg -S "$b" >/dev/null 2>&1 && ! rpm -qf "$b" >/dev/null 2>&1; then
            warn "unpackaged PHP at $b ($("$b" -v 2>/dev/null | head -1)); rebuild it from a fixed release"
        fi
    done
}

main() {
    local before after
    log "----- PHP packages before -----"
    before=$(php_packages)
    if [ -z "$before" ]; then
        log "No PHP packages installed through the package manager."
    else
        echo "$before" | sed 's/^/PACKAGE: /'
        log "----- updating -----"
        upgrade_php
        after=$(php_packages)
        log "----- PHP packages after -----"
        echo "$after" | sed 's/^/PACKAGE: /'
        changelog_mentions_fix
        log "----- PHP workers -----"
        if [ "$before" != "$after" ]; then
            restart_workers
        elif stale_workers; then
            log "No package changed, but a worker still runs a replaced binary."
            restart_workers
        else
            log "No package changed; nothing to restart. If the policy still fails, this PHP branch has no vendor fix yet or the mirror hasn't synced."
        fi
        stale_workers && warn "a worker still runs old code after the restart; restart its service by hand"
    fi
    log "----- other PHP runtimes -----"
    other_runtimes
    if [ "$EXIT" -eq 0 ]; then
        log "DONE: the Fleet policy re-checks package versions on its next run."
    else
        log "DONE with warnings (exit 2): read the lines above."
    fi
    exit "$EXIT"
}

main "$@"
