#!/bin/bash
# Build (if needed) and show a toast, then print the exit code.
#
# Execs the binary directly rather than going through `open`, because `open`
# returns its own status rather than the app's — which is the whole point of the
# exit code here.
#
# Usage:
#   ./dev-notify.sh                        placeholder page, id 42
#   ./dev-notify.sh 7                      placeholder page, id 7
#   ./dev-notify.sh 7 --timeout 10         auto-dismiss after 10s
#   ./dev-notify.sh 7 --url https://…      dev builds only; overrides --placeholder

# Note: no `set -e`. Observing a non-zero exit code is the point.
set -uo pipefail

cd "$(dirname "$0")"

BIN="build/Fleet Desktop.app/Contents/MacOS/FleetDesktop"
if [ ! -x "$BIN" ]; then
    FLEET_DESKTOP_DEV=1 ./build.sh
fi

# Optional leading id, so both `dev-notify.sh 7 --timeout 10` and
# `dev-notify.sh --timeout 10` work. Anything non-numeric is a flag, not an id.
ID=42
if [[ "${1:-}" =~ ^[0-9]+$ ]]; then
    ID="$1"
    shift
fi

# --placeholder unless the caller passed their own source override.
SOURCE=(--placeholder)
for arg in "$@"; do
    case "$arg" in
        --url|--html) SOURCE=() ;;
    esac
done

# The +"${...}" form keeps an empty array from tripping `set -u` on bash 3.2, which
# is what macOS ships as /bin/bash.
"$BIN" notify --patch-notification-id "$ID" ${SOURCE[@]+"${SOURCE[@]}"} "$@"
echo "exit=$?"
