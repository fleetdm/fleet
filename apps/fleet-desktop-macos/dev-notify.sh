#!/bin/bash
# Build (if needed) and show a toast, then print the exit code.
#
# Execs the binary directly rather than going through `open`, because `open`
# returns its own status rather than the app's — which is the whole point of the
# exit code here.
#
# Usage:
#   ./dev-notify.sh [--url] <https url>
#
# Pass a real device page URL, or any page implementing the fleetDesktop bridge (see
# ToastWindow.swift for the message contract).

# Note: no `set -e`. Observing a non-zero exit code is the point.
set -uo pipefail

cd "$(dirname "$0")" || exit 1

# Accept both `dev-notify.sh <url>` and `dev-notify.sh --url <url>`. The flag is what
# the binary takes, so typing it here is the natural thing to do.
if [ "${1:-}" = "--url" ]; then
    shift
fi

if [ "$#" -eq 0 ]; then
    echo "usage: $0 [--url] <https url>" >&2
    exit 2
fi

BIN="build/Fleet Desktop.app/Contents/MacOS/FleetDesktop"
if [ ! -x "$BIN" ]; then
    ./build.sh
fi

URL="$1"
shift

# Remaining arguments pass straight through to notify.
"$BIN" notify --url "$URL" "$@"
echo "exit=$?"
