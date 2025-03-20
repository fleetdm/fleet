#!/bin/sh

if [ $(id -u) -ne 0 -a -z "$GITHUB_ACTIONS" ]; then
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

    # Check MDM status on a macOS device
    mdm_status=$(profiles status -type enrollment)

    # Check for MDM enrollment status and cleanup enrollment profile
    if echo "$mdm_status" | grep -q "MDM enrollment: Yes"; then
        echo "This Mac is MDM enrolled. Removing enrollment profile."
        profiles remove -identifier com.fleetdm.fleet.mdm.apple
    elif echo "$mdm_status" | grep -q "MDM enrollment: No"; then
        echo "This Mac is not MDM enrolled."
    else
        echo "MDM status is unknown."
    fi

}

if [ "$1" = "remove" ]; then
    # We are in the detached child process
    # Give the parent process time to report the success before removing
    echo "inside remove process" >>/tmp/fleet_remove_log.txt
    sleep 15
    if [ -z "$GITHUB_ACTIONS" ]; then
        # We are root
        remove_fleet >>/tmp/fleet_remove_log.txt 2>&1
    else
        # Inside a github action, sudo is passwordless
        sudo remove_fleet >>/tmp/fleet_remove_log.txt 2>&1
    fi
else
    # We are in the parent shell, start the detached child and return success
    echo "Removing fleet, system will be unenrolled in 15 seconds..."
    echo "Executing detached child process"
    if [ -z "$GITHUB_ACTIONS" ]; then
        # We are root
        bash -c "bash $0 remove >/dev/null 2>/dev/null </dev/null &"
    else
        # We are in a github action
        sudo bash -c "bash $0 remove >/dev/null 2>/dev/null </dev/null &"
    fi
fi
