#!/bin/bash
set -euo pipefail

fleet_url="https://fleet.yourdomain.com"
identifier_file="/opt/orbit/identifier"

if [[ ! -r "$identifier_file" ]]; then
    echo "Missing or unreadable orbit identifier: $identifier_file" >&2
    exit 1
fi

orbit_identifier="$(tr -d '[:space:]' < "$identifier_file")"
if [[ -z "$orbit_identifier" ]]; then
    echo "Orbit identifier is empty." >&2
    exit 1
fi

echo "Triggering refetch..."

if curl -sf --connect-timeout 5 --max-time 20 -X POST \
  "$fleet_url/api/v1/fleet/device/$orbit_identifier/refetch" > /dev/null; then
    echo "Refetch triggered successfully!"
else
    rc=$?
    echo "Refetch failed!" >&2
    exit "$rc"
fi
