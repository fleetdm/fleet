#!/usr/bin/env bash

sudo launchctl stop com.fleetdm.orbit
sudo launchctl unload /Library/LaunchDaemons/com.fleetdm.orbit.plist

sudo rm -rf /Library/LaunchDaemons/com.fleetdm.orbit.plist /var/lib/orbit/ /usr/local/bin/orbit /var/log/orbit
