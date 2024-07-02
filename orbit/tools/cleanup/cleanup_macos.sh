#!/bin/sh

if [ $(id -u) -ne 0 ]; then
    echo "Please run as root"
    exit 1
fi

function remove_fleet {
    set -x

    rm -rf /Library/LaunchDaemons/com.fleetdm.orbit.plist /var/lib/orbit /usr/local/bin/orbit /var/log/orbit /opt/orbit/

    pkgutil --forget com.fleetdm.orbit.base.pkg || true

    launchctl stop com.fleetdm.orbit
    launchctl unload /Library/LaunchDaemons/com.fleetdm.orbit.plist

    pkill fleet-desktop || true
}

if [ "$1" = "remove" ]; then
    # We are in the detached child process
    # Give the parent process time to report the success before removing
    echo "inside remove process" >>/tmp/fleet_remove_log.txt
    sleep 15
    remove_fleet >>/tmp/fleet_remove_log.txt 2>&1
else
    # We are in the parent shell, start the detached child and return success
    echo "Removing fleet, system will be unenrolled in 15 seconds..."
    echo "Executing detached child process"
    bash -c "bash $0 remove >/dev/null 2>/dev/null </dev/null &"
fi
