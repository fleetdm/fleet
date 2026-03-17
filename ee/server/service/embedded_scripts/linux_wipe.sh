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
        echo "Error: /proc/mounts not found; aborting wipe to avoid unsafe network data deletion" >&2
        exit 1
    fi

    awk '$3 ~ /^('"$NETWORK_FS_TYPES"')$/ {print $2}' /proc/mounts \
        | awk '{print length, $0}' \
        | sort -nr \
        | cut -d" " -f2- \
        | while read -r mnt_esc; do
        # Unescape mountpoint in case it contains octal escapes like \040 for space.
        mnt=$(printf '%b' "$mnt_esc")
        # Never unmount critical mountpoints that may contain required userland.
        case "$mnt" in
            /|/usr|/bin|/sbin|/lib|/lib64|/usr/bin|/usr/sbin|/usr/lib|/usr/lib64)
                echo "Skipping critical network-mounted filesystem: $mnt"
                continue
                ;;
        esac
        echo "Unmounting network filesystem: $mnt"
        umount -f -l "$mnt" 2>/dev/null || echo "Warning: failed to unmount $mnt"
    done
}

# Returns 0 (true) if the given path resides on a network filesystem.
is_network_mount() {
    if [ ! -f /proc/mounts ]; then
        echo "Error: /proc/mounts not found; aborting wipe to avoid unsafe network data deletion" >&2
        exit 1
    fi
    _target="$1"
    # Resolve the target to a canonical path if possible, so that symlinks
    # (e.g. /home -> /mnt/nfs/home) do not hide network mounts.
    if command -v readlink >/dev/null 2>&1; then
        _resolved=$(readlink -f -- "$_target" 2>/dev/null || printf '%s\n' "$_target")
        _target="$_resolved"
    fi
    # Walk up to find the mount point that contains this path.
    _match=$(awk '$3 ~ /^('"$NETWORK_FS_TYPES"')$/ {print $2}' /proc/mounts | while read -r mnt_esc; do
        mnt=$(printf '%b' "$mnt_esc")
        case "$mnt" in
            /)
                case "$_target" in
                    /*) echo "$mnt"; break ;;
                esac
                ;;
            *)
                # Normalize mountpoint by removing any trailing slash (except for root,
                # which is already handled above) and perform literal prefix checks
                # so that glob metacharacters in $mnt do not affect matching.
                mnt_no_slash=${mnt%/}
                if [ "$_target" = "$mnt_no_slash" ] || [ "${_target#"$mnt_no_slash"/}" != "$_target" ]; then
                    echo "$mnt_no_slash"
                    break
                fi
                ;;
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
    # Fallback for non-GNU rm (e.g. BusyBox): use find -xdev to stay on the
    # same filesystem. Avoid rm -rf so we never recurse into nested mounts
    # whose mountpoint entries live on the local device.
    # If the path is not a directory or is a symlink, just unlink it directly.
    if [ ! -d "$_path" ] || [ -L "$_path" ]; then
        rm -f "$_path" 2>/dev/null
        return
    fi
    (
        cd "$_path" 2>/dev/null || exit 0
        find . -xdev -depth ! -name . ! -type d -exec rm -f {} \; 2>/dev/null
        find . -xdev -depth ! -name . -type d -exec rmdir {} \; 2>/dev/null
    )
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
    # We are in the detached child process
    wipe_all_files
else
    # We are in the parent shell, logout users and begin the detached
    # wipe child process
    logout_users
    echo "Wiping, system will be unreachable"
    (/usr/bin/nohup sh $0 wipe >/dev/null 2>/dev/null </dev/null) &
fi
