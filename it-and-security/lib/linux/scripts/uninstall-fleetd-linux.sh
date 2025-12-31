#!/bin/bash
# Please don't delete. This script is used in tests (tools/tuf/test/migration/migration_test.sh), workflors (.github/workflows/), and in the guide here: https://fleetdm.com/guides/how-to-uninstall-fleetd

if [ $(id -u) -ne 0 ]; then
    echo "Please run as root"
    exit 1
fi

function remove_fleet {
    set -x
    systemctl stop orbit.service || true
    systemctl disable orbit.service || true
    rm -rf /var/lib/orbit /opt/orbit /var/log/orbit /usr/local/bin/orbit /etc/default/orbit /usr/lib/systemd/system/orbit.service

    # Remove any package references
    if command -v dpkg > /dev/null; then
        dpkg --purge fleet-osquery || true
    elif command -v rpm > /dev/null; then
        rpm -e fleet-osquery || true
    elif command -v pacman > /dev/null; then
        pacman -Rns --noconfirm fleet-osquery || true
    fi

    # Kill any running Fleet processes
    pkill -f fleet-desktop || true

    # Reload systemd configuration
    systemctl daemon-reload

    echo "Fleetd has been successfully removed from the system."
}

if [ "$1" = "remove" ]; then
    # We are in the detached child process
    # Give the parent process time to report the success before removing
    echo "inside remove process" >>/tmp/fleet_remove_log.txt
    sleep 15

    # We are root
    remove_fleet >>/tmp/fleet_remove_log.txt 2>&1
else
    # We are in the parent shell, start the detached child and return success
    echo "Removing fleetd, system will be unenrolled in 15 seconds..."
    echo "Executing detached child process"

    # We are root
    # Use systemd-run to spawn the removal process in a separate transient unit.
    # This ensures the process escapes the orbit.service cgroup, so when
    # orbit.service is stopped, the removal script continues running.
    if command -v systemd-run > /dev/null; then
        systemd-run --quiet bash "$0" remove
    else
        # Fallback for non-systemd systems (rare for modern Linux)
        bash -c "bash $0 remove >/dev/null 2>/dev/null </dev/null &"
    fi
fi
