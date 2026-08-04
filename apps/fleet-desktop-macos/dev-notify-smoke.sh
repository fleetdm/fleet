#!/bin/bash
# Assert the notify exit-code contract.
#
# Only covers cases that exit before AppKit starts, so it needs no display and no
# logged-in user — which makes it safe to run in CI. The paths that actually put a
# window on screen have to be checked by hand; see dev-notify.sh.

set -uo pipefail

cd "$(dirname "$0")" || exit 1

BIN="build/Fleet Desktop.app/Contents/MacOS/FleetDesktop"
if [ ! -x "$BIN" ]; then
    ./build.sh
fi

failures=0

expect() {
    local want="$1" description="$2"
    shift 2

    local output
    output=$("$BIN" "$@" 2>&1)
    local got=$?

    if [ "$got" = "$want" ]; then
        printf 'ok   %-32s exit=%s\n' "$description" "$got"
    else
        printf 'FAIL %-32s want=%s got=%s\n     %s\n' \
            "$description" "$want" "$got" "$(echo "$output" | head -1)"
        failures=$((failures + 1))
    fi
}

expect 2 "notify without a url"    notify
expect 2 "url missing its value"   notify --url
expect 2 "http url"                notify --url http://example.com
expect 2 "host-less url"           notify --url "https://"
expect 2 "not a url"               notify --url "not a url"
expect 2 "unknown option"          notify --url https://example.com --nope
expect 2 "unknown subcommand"      frobnicate
expect 2 "child without pipe"      notify --url https://example.com --detached-child
expect 2 "fd is not a number"      notify --url https://example.com --detached-child --handshake-fd abc
expect 2 "negative fd"             notify --url https://example.com --detached-child --handshake-fd -1
expect 0 "help"                    help
expect 0 "--help"                  --help

echo
if [ "$failures" -eq 0 ]; then
    echo "all checks passed"
else
    echo "$failures check(s) failed"
    exit 1
fi
