#!/usr/bin/env bash

sudo launchctl stop com.fleetdm.orbit
sudo launchctl unload /Library/LaunchDaemons/com.fleetdm.orbit.plist

sudo pkill fleet-desktop || true
sudo rm -rf /Library/LaunchDaemons/com.fleetdm.orbit.plist /var/lib/orbit /usr/local/bin/orbit /var/log/orbit /opt/orbit/

sudo pkgutil --forget com.fleetdm.orbit.base.pkg || true
