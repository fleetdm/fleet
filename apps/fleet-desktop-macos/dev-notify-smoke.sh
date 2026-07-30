#!/bin/bash
# Assert the notify exit-code contract.
#
# Only covers cases that exit before AppKit starts, so it needs no display, no
# logged-in user and no Fleet configuration — which makes it safe to run in CI. The
# paths that actually put a window on screen have to be checked by hand; see
# dev-notify.sh.

set -uo pipefail

cd "$(dirname "$0")"

BIN="build/Fleet Desktop.app/Contents/MacOS/FleetDesktop"
if [ ! -x "$BIN" ]; then
    ./build.sh
fi

failures=0

# expect <want> <description> [env=value ...] -- <args...>
expect() {
    local want="$1" description="$2"
    shift 2

    local env=()
    while [ "$#" -gt 0 ] && [ "$1" != "--" ]; do
        env+=("$1")
        shift
    done
    shift || true

    # The +"${...}" form keeps an empty array from tripping `set -u` on bash 3.2,
    # which is what macOS ships as /bin/bash.
    local output
    output=$(env ${env[@]+"${env[@]}"} "$BIN" "$@" 2>&1)
    local got=$?

    if [ "$got" = "$want" ]; then
        printf 'ok   %-34s exit=%s\n' "$description" "$got"
    else
        printf 'FAIL %-34s want=%s got=%s\n     %s\n' "$description" "$want" "$got" "$(echo "$output" | head -1)"
        failures=$((failures + 1))
    fi
}

expect 0  "help"                    -- help
expect 0  "--help"                  -- --help
expect 2  "notify without an id"    -- notify
expect 2  "non-numeric id"          -- notify --patch-notification-id abc
expect 2  "empty id"                -- notify --patch-notification-id ""
expect 2  "unknown option"          -- notify --patch-notification-id 42 --nope
expect 2  "unknown subcommand"      -- frobnicate
expect 2  "missing flag value"      -- notify --patch-notification-id
expect 2  "child without pipe"      -- notify --patch-notification-id 42 --detached-child

# A token file that doesn't exist, and one whose contents aren't a valid token. Both
# report the single configuration code. ORBIT_ROOT_DIR works on release builds too.
mkdir -p /tmp/fleet-notify-smoke
printf 'not/a/valid token\n' > /tmp/fleet-notify-smoke/identifier
expect 20 "missing token file"       ORBIT_ROOT_DIR=/tmp/fleet-notify-smoke-nonexistent -- notify --patch-notification-id 42
expect 20 "malformed token"          ORBIT_ROOT_DIR=/tmp/fleet-notify-smoke -- notify --patch-notification-id 42
rm -rf /tmp/fleet-notify-smoke

echo
if [ "$failures" -eq 0 ]; then
    echo "all checks passed"
else
    echo "$failures check(s) failed"
    exit 1
fi
