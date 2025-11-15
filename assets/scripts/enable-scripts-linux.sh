#!/bin/bash
# Please don't delete. This script is used in the guide here: https://fleetdm.com/guides/scripts

if [ "$EUID" -ne 0 ]; then
  echo "This script requires administrator privileges. Please run with sudo."
  exit 1
fi
# Enable scripts in Orbit environment variables
if grep -q "^ORBIT_ENABLE_SCRIPTS=" /etc/default/orbit; then
  sed -i 's/^ORBIT_ENABLE_SCRIPTS=.*/ORBIT_ENABLE_SCRIPTS=true/' /etc/default/orbit
else
  echo "ORBIT_ENABLE_SCRIPTS=true" >> /etc/default/orbit
fi
# Reload and restart Orbit
systemctl daemon-reload
systemctl restart orbit
