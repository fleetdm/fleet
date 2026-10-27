#!/bin/bash
# remediate-php2shell-macos.sh
# Update Homebrew PHP for PHP2Shell (CVE-2026-109618, CVE-2026-109619, disclosed 2026-10-21).
#   1. Runs brew update, then brew upgrade for every installed php or php@X.Y formula, as the
#      Homebrew owner (Fleet runs scripts as root, and brew refuses to run as root).
#   2. Restarts PHP services that launchd manages for Homebrew (php-fpm and httpd), so no
#      worker keeps the old code.
#   3. Flags runtimes it can't update: php@8.1 and older (no fix), MAMP, containers.
# Fixed releases: 8.5.12, 8.4.27, 8.3.36, 8.2.35.
# Idempotent: safe to re-run. Built for Fleet "Run script" (runs as root).
# Exit: 0 = updated or nothing to do, 2 = something needs a look (read output), 1 = error.
# Advisories: https://github.com/php/php-src/security/advisories/GHSA-w8hw-w9fx-j79w
#             https://github.com/php/php-src/security/advisories/GHSA-6wmf-4rv5-p3rp

set -u
export PATH="/usr/bin:/bin:/usr/sbin:/sbin"
EXIT=0

log()  { echo "$*"; }
warn() { echo "WARN: $*"; EXIT=2; }

# Fixed version for a Homebrew formula version string ("8.3.35_1" -> 8.3.36), empty if no fix.
fixed_for() {
    case "$1" in
        8.5.*) echo 8.5.12 ;;
        8.4.*) echo 8.4.27 ;;
        8.3.*) echo 8.3.36 ;;
        8.2.*) echo 8.2.35 ;;
        *) echo "" ;;
    esac
}

# 0 when installed version ($1, revision suffix allowed) is at or above fixed ($2).
at_least() {
    local v="${1%%_*}"
    [ "$(printf '%s\n%s\n' "$2" "$v" | sort -V | head -1)" = "$2" ]
}

restart_php_services() {
    local f="$1" label
    for label in "homebrew.mxcl.$f" "homebrew.mxcl.httpd"; do
        if launchctl print "system/$label" >/dev/null 2>&1; then
            launchctl kickstart -k "system/$label" 2>/dev/null && log "RESTARTED: $label (system)"
        fi
    done
    # User-level brew services run in the owner's launchd domain.
    if run_brew services list 2>/dev/null | awk -v f="$f" '($1 == f || $1 == "httpd") && $2 ~ /started|scheduled/ { found = 1 } END { exit !found }'; then
        run_brew services restart "$f" >/dev/null 2>&1 && log "RESTARTED: brew services $f"
    fi
}

main() {
    local brew owner installed name version fixed new imgs changed=0
    for brew in /opt/homebrew/bin/brew /usr/local/bin/brew; do [ -x "$brew" ] && break; brew=""; done
    if [ -z "$brew" ]; then
        log "Homebrew is not installed; nothing to do."
    else
        # /usr/local is root-owned on Intel Macs; the Homebrew repository is owned by the brew user.
        owner=$(dirname "$(dirname "$brew")")
        [ -d "$owner/Homebrew" ] && owner="$owner/Homebrew"
        owner=$(stat -f %Su "$owner")
        run_brew() {
            # sudo resets the environment, so pass Homebrew settings through env explicitly.
            if [ "$(id -un)" = "$owner" ]; then (cd /tmp && env HOMEBREW_NO_INSTALLED_DEPENDENTS_CHECK=1 HOMEBREW_NO_AUTO_UPDATE=1 "$brew" "$@")
            else (cd /tmp && sudo -u "$owner" -H env HOMEBREW_NO_INSTALLED_DEPENDENTS_CHECK=1 HOMEBREW_NO_AUTO_UPDATE=1 "$brew" "$@"); fi
        }
        installed=$(run_brew list --formula --versions 2>/dev/null | awk '$1 ~ /^php(@[0-9.]+)?$/ { print $1, $NF }')
        if [ -z "$installed" ]; then
            log "No Homebrew PHP formula installed."
        else
            log "----- Homebrew PHP before -----"
            echo "$installed" | sed 's/^/FORMULA: /'
            run_brew update >/dev/null 2>&1 || warn "brew update failed (network or git issue); upgrading from the current tap"
            while read -r name version; do
                fixed=$(fixed_for "$version")
                if [ -z "$fixed" ]; then
                    warn "$name $version has no fix: Homebrew no longer updates this branch. Replace it with php@8.2 or newer."
                    continue
                fi
                if at_least "$version" "$fixed"; then
                    log "OK: $name $version is at or above $fixed"
                    continue
                fi
                if run_brew upgrade "$name" >/dev/null 2>&1; then
                    # brew upgrade also exits 0 when the tap has no newer formula yet.
                    new=$(run_brew list --versions "$name" 2>/dev/null | awk '{ print $NF }')
                    if at_least "$new" "$fixed"; then
                        changed=1
                        log "UPGRADED: $name -> $new"
                        restart_php_services "$name"
                    else
                        warn "$name is still $new after brew upgrade; Homebrew hasn't published $fixed yet, or brew update failed"
                    fi
                else
                    warn "brew upgrade $name failed; run it by hand as $owner"
                fi
            done <<< "$installed"
            log "----- Homebrew PHP after -----"
            run_brew list --formula --versions 2>/dev/null | awk '$1 ~ /^php(@[0-9.]+)?$/ { print "FORMULA: " $1, $NF }'
            [ "$changed" -eq 0 ] && log "Nothing upgraded."
        fi
    fi

    log "----- other PHP runtimes -----"
    if [ -d /Applications/MAMP/bin/php ]; then
        warn "MAMP bundles its own PHP: $(find /Applications/MAMP/bin/php -maxdepth 1 -mindepth 1 -type d -exec basename {} \; 2>/dev/null | tr '\n' ' '). Update MAMP to a release that ships 8.2.35, 8.3.36, 8.4.27, or 8.5.12."
    fi
    if command -v docker >/dev/null 2>&1 || [ -x /usr/local/bin/docker ]; then
        imgs=$(/usr/local/bin/docker ps --format '{{.Names}} {{.Image}}' 2>/dev/null | grep -iE 'php|wordpress|drupal|joomla|nextcloud|laravel|frankenphp')
        if [ -n "$imgs" ]; then
            warn "running containers use PHP images; pull or rebuild them on a fixed base and recreate:"
            echo "$imgs" | sed 's/^/  CONTAINER: /'
        fi
    fi
    pgrep -x php-fpm >/dev/null 2>&1 && log "php-fpm is running: $(pgrep -x php-fpm | head -1 | xargs -I{} sh -c 'ps -o comm= -p {}' 2>/dev/null)"

    if [ "$EXIT" -eq 0 ]; then
        log "DONE: the Fleet policy re-checks formula versions on its next run."
    else
        log "DONE with warnings (exit 2): read the lines above."
    fi
    exit "$EXIT"
}

main "$@"
