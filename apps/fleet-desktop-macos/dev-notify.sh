#!/bin/bash
# Build (if needed) and show a toast, then print the exit code.
#
# Execs the binary directly rather than going through `open`, because `open`
# returns its own status rather than the app's — which is the whole point of the
# exit code here.
#
# Usage:
#   ./dev-notify.sh https://your-deployment.vercel.app
#
# Deploy placeholder-page/ somewhere (see its README) to get a URL to point at, or
# pass a real device page URL.

# Note: no `set -e`. Observing a non-zero exit code is the point.
set -uo pipefail

cd "$(dirname "$0")"

if [ "$#" -eq 0 ]; then
    echo "usage: $0 <https url>" >&2
    exit 2
fi

BIN="build/Fleet Desktop.app/Contents/MacOS/FleetDesktop"
if [ ! -x "$BIN" ]; then
    ./build.sh
fi

"$BIN" notify --url "$1"
echo "exit=$?"
