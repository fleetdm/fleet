#!/bin/bash

orbit_identifier=$(cat /opt/orbit/identifier | tr -d '[:space:]')
fleet_url="https://fleet.yourdomain.com"

echo "Triggering refetch..."

if curl -sf -X POST "$fleet_url/api/v1/fleet/device/$orbit_identifier/refetch" > /dev/null; then
    echo "Refetch triggered successfully!"
else
    echo "Refetch failed!"
    exit 0
fi
