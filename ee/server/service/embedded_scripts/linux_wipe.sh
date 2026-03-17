#!/bin/sh

NETWORK_FS_TYPES="nfs|nfs4|cifs|smb|smbfs|fuse\.sshfs|afs|ncpfs|9p"

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

# Unmount all network filesystems to prevent remote data deletion.
unmount_network_filesystems() {
    if [ ! -f /proc/mounts ]; then
        echo "Warning: /proc/mounts not found, cannot detect network mounts"
        return
    fi

    awk '$3 ~ /^('"$NETWORK_FS_TYPES"')$/ {print $2}' /proc/mounts \
        | awk '{print length, $0}' \
        | sort -nr \
        | cut -d" " -f2- \
        | while read -r mnt; do
        echo "Unmounting network filesystem: $mnt"
        umount -f -l "$mnt" 2>/dev/null || echo "Warning: failed to unmount $mnt"
    done
}

# Returns 0 (true) if the given path resides on a network filesystem.
is_network_mount() {
    if [ ! -f /proc/mounts ]; then
        return 1
    fi
    _target="$1"
    # Walk up to find the mount point that contains this path.
    _match=$(awk '$3 ~ /^('"$NETWORK_FS_TYPES"')$/ {print $2}' /proc/mounts | while read -r mnt; do
        case "$_target" in
            "$mnt"|"$mnt"/*) echo "$mnt"; break ;;
        esac
    done)
    [ -n "$_match" ]
}

# rm -rf wrapper that prevents crossing filesystem boundaries.
# Uses GNU --one-file-system when available, falls back to find -xdev.
safe_rm() {
    _path="$1"
    if rm --one-file-system -rf "$_path" 2>/dev/null; then
        return
    fi
    find "$_path" -xdev -mindepth 1 -exec rm -rf {} + 2>/dev/null
    rmdir "$_path" 2>/dev/null
}

# Function to wipe non-essential data
wipe_non_essential_data() {
    non_essential_paths="/home/* /tmp /var/tmp /var/log /home/*/.cache /var/cache /home/*/.local/share/Trash"

    for path in $non_essential_paths
    do
        if [ -e "$path" ]; then
            if is_network_mount "$path"; then
                echo "Skipping $path (network filesystem)"
                continue
            fi
            echo "Wiping $path"
            safe_rm "$path"
        fi
    done
}

# Function to wipe system files - Warning: This will render the system inoperable
wipe_system_files() {
    essential_system_paths="/bin /sbin /usr /lib /opt /etc /var /srv"

    for path in $essential_system_paths
    do
        if is_network_mount "$path"; then
            echo "Skipping $path (network filesystem)"
            continue
        fi
        echo "Wiping $path"
        safe_rm "$path"
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
    unmount_network_filesystems
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
