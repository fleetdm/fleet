#!/bin/bash
# remove-remote-access-apps.sh
# Quits and removes dedicated remote-access apps from a Mac, and unloads their launchd jobs.
# Run by the "macOS - Remote access apps not installed" policy; keep the lists below in sync with that policy's query.
#
# Zoom (us.zoom.xos, zoom.us.app) has a screen sharing feature and is explicitly allowed. It is not in any list
# below, and the guard in main() skips it even if a list is edited wrongly. Slack, Google Meet, and Microsoft
# Teams aren't touched. macOS built-in Screen Sharing and Remote Management aren't touched.
#
# What it does for each match: quits the app, boots out and deletes launchd jobs the vendor registered (system
# and the console user's), deletes the .app bundle, and forgets the vendor's package receipts. It leaves vendor
# support files (preferences, caches, Application Support) in place: they're inert without the app.
# Idempotent: safe to re-run. Built for Fleet "Run script" (runs as root).
# Exit: 0 = removed or nothing to do, 2 = something needs a look (read output), 1 = error.

set -u
export PATH="/usr/bin:/bin:/usr/sbin:/sbin"
export LC_ALL=C

STATUS=0
log()  { echo "$*"; }
warn() { echo "WARN: $*"; STATUS=2; }

[ "$(id -u)" -eq 0 ] || { echo "ERROR: must run as root" >&2; exit 1; }

# CFBundleIdentifier values. Verified against ee/maintained-apps (FMA) or Homebrew casks; see the PR description.
BUNDLE_IDS='
com.teamviewer.TeamViewer
com.teamviewer.TeamViewerQS
com.teamviewer.TeamViewerHost
com.philandro.anydesk
com.splashtop.stb.macosx
com.splashtop.Splashtop-Streamer
com.carriez.rustdesk
com.realvnc.vncserver_service
com.realvnc.rvncconnect
com.p5sys.jump.mac.viewer.web
com.p5sys.jump.connect
com.google.chromeremotedesktop.me2me-host-uninstaller
tv.parsec.www
'

# .app folder names, for apps whose bundle identifier isn't verified.
APP_NAMES='
TeamViewer QuickSupport.app
LogMeIn Client.app
NoMachine.app
Supremo.app
'

# launchd labels (case patterns) these vendors register as LaunchDaemons / LaunchAgents.
label_matches() {
    case "$1" in
        com.teamviewer.*|com.philandro.anydesk.*|com.splashtop.*|com.realvnc.*|com.p5sys.jump.connect*|\
        application.com.p5sys.jump.connect.*|com.nomachine.*|com.carriez.*|org.chromium.chromoting)
            return 0 ;;
    esac
    return 1
}

# Package receipts to forget after the app is gone.
PKG_RECEIPTS='
com.teamviewer.teamviewer.*
com.teamviewer.AuthorizationPlugin
com.teamviewer.AuthorizationResources
com.teamviewer.remoteaudiodriver
com.splashtop.*
com.realvnc.*
com.nomachine.*
com.p5sys.jump.connect
tv.parsec.www
com.google.pkg.ChromeRemoteDesktopHost
com.google.pkg.ChromeRemoteDesktopHostService
com.google.pkg.ChromeRemoteDesktopHostUninstaller
'

is_zoom() {
    case "$1" in
        us.zoom.*|zoom.us.app|zoom.us|Zoom*) return 0 ;;
    esac
    return 1
}

in_list() { # in_list <newline-separated list> <value>
    printf '%s\n' "$1" | grep -qxF -- "$2"
}

is_target_app() { # is_target_app <path to .app>
    local app="$1" name bid
    name="$(basename "$app")"
    bid="$(defaults read "$app/Contents/Info" CFBundleIdentifier 2>/dev/null || true)"
    if is_zoom "$bid" || is_zoom "$name"; then
        return 1
    fi
    in_list "$BUNDLE_IDS" "$bid" && return 0
    in_list "$APP_NAMES" "$name" && return 0
    case "$name" in
        ScreenConnect*|connectwisecontrol*) return 0 ;;
    esac
    return 1
}

kill_processes_in() { # kill_processes_in <path>: stop every process running from inside <path>
    local path="$1" pids
    pids="$(ps -axo pid=,command= | awk -v p="$path/" 'index($0, p) > 0 { print $1 }')"
    [ -n "$pids" ] || return 0
    log "Stopping processes from $path"
    # shellcheck disable=SC2086
    kill $pids 2>/dev/null
    sleep 2
    pids="$(ps -axo pid=,command= | awk -v p="$path/" 'index($0, p) > 0 { print $1 }')"
    # shellcheck disable=SC2086
    [ -z "$pids" ] || kill -9 $pids 2>/dev/null
    return 0
}

console_uid() {
    local user
    user="$(stat -f '%Su' /dev/console 2>/dev/null)"
    case "$user" in
        ""|root|loginwindow) return 1 ;;
    esac
    id -u "$user" 2>/dev/null
}

remove_launchd_jobs() {
    local uid plist label
    uid="$(console_uid || true)"
    for plist in /Library/LaunchDaemons/*.plist /Library/LaunchAgents/*.plist; do
        [ -f "$plist" ] || continue
        label="$(basename "$plist" .plist)"
        label_matches "$label" || continue
        log "Removing launchd job $label ($plist)"
        launchctl bootout "system/$label" >/dev/null 2>&1
        [ -z "$uid" ] || launchctl bootout "gui/$uid/$label" >/dev/null 2>&1
        rm -f -- "$plist" || warn "could not delete $plist"
    done
}

forget_receipts() {
    local pattern receipt
    while IFS= read -r pattern; do
        [ -n "$pattern" ] || continue
        # shellcheck disable=SC2086
        for receipt in $(pkgutil --pkgs 2>/dev/null | grep -E "^$(printf '%s' "$pattern" | sed -e 's/[.]/\\./g' -e 's/\*/.*/g')\$"); do
            log "Forgetting package receipt $receipt"
            pkgutil --forget "$receipt" >/dev/null 2>&1
        done
    done <<EOF
$PKG_RECEIPTS
EOF
}

main() {
    local root app removed=0
    # Roots only: /System and the Fleet agent's own folders are never searched.
    for root in /Applications /Applications/Utilities /Users/*/Applications; do
        [ -d "$root" ] || continue
        for app in "$root"/*.app; do
            [ -d "$app" ] && [ ! -L "$app" ] || continue
            is_target_app "$app" || continue
            log "Found $app"
            kill_processes_in "$app"
            if rm -rf -- "$app"; then
                log "Removed $app"
                removed=$((removed + 1))
            else
                warn "could not remove $app"
            fi
        done
    done

    remove_launchd_jobs
    [ "$removed" -eq 0 ] || forget_receipts

    if [ "$removed" -eq 0 ]; then
        log "No remote-access apps found."
    else
        log "Removed $removed remote-access app(s)."
    fi
    exit "$STATUS"
}

main
