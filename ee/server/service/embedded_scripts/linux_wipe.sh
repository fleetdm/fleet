#!/bin/sh

# Function to log out all users and lock their passwords except root
logout_users() {
    for user in $(who | awk '{print $1}' | sort | uniq)
    do
        if [ "$user" != "root" ]; then
            echo "Logging out $user"
            pkill -KILL -u "$user"
            passwd -l "$user"
        fi
    done
}

# Function to wipe non-essential data
wipe_non_essential_data() {
    # Define non-essential paths
    non_essential_paths="/home/* /tmp /var/tmp /var/log /home/*/.cache /var/cache /home/*/.local/share/Trash"

    for path in $non_essential_paths
    do
        if [ -e "$path" ]; then
            echo "Wiping $path"
            rm -rf "$path"
        fi
    done
}

# Function to wipe system files - Warning: This will render the system inoperable
wipe_system_files() {
    # Define essential system paths
    essential_system_paths="/bin /sbin /usr /lib"

    for path in $essential_system_paths
    do
        echo "Wiping $path"
        rm -rf "$path"
    done
}

prepare_system_reset() {
    cp /usr/bin/sync /sync_bin
    # https://docs.kernel.org/admin-guide/sysrq.html
    echo "1" > /proc/sys/kernel/sysrq
}

system_reset() {
    # Give the system time to sync
    /sync_bin
    # Halt the system immediately
    echo "o" > /proc/sysrq-trigger
}

wipe_all_files() {
    sleep 10 # Give fleetd enough time to register the script as completed
    prepare_system_reset
    wipe_non_essential_data
    wipe_system_files
    system_reset
}

if [ "$1" = "wipe" ]; then
    # We are in the detatched child process
    wipe_all_files
else
    # We are in the parent shell, logout users and begin the detached
    # wipe child process
    logout_users
    echo "Wiping, system will be unreachable"
    (/usr/bin/nohup sh $0 wipe >/dev/null 2>/dev/null </dev/null) &
fi
