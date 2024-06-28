#!/bin/sh

if [ $(id -u) -ne 0 ]; then
    echo "Please run as root"
    exit 1
fi


function remove_fleet {
    launchctl stop com.fleetdm.orbit
    launchctl unload /Library/LaunchDaemons/com.fleetdm.orbit.plist

    pkill fleet-desktop || true
    rm -rf /Library/LaunchDaemons/com.fleetdm.orbit.plist /var/lib/orbit /usr/local/bin/orbit /var/log/orbit /opt/orbit/

    pkgutil --forget com.fleetdm.orbit.base.pkg || true
}

if [ $1 == "remove" ]; then
    # We are in the detached child process
    # Give the parent process time to report the success before removing
    sleep 15
    remove_fleet
else
    # We are in the parent shell, start the detached child and return success
    echo "Removing fleet, system will be unenrolled"
    echo "Executing detached child process"
    nohup sh $0 remove >/dev/null 2>/dev/null </dev/null &
fi
