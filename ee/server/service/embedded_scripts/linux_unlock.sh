#!/bin/sh

# Unlock password for all non-root users
awk -F':' '{ if ($3 >= 1000 && $3 < 60000) print $1 }' /etc/passwd | while read user
do
    echo "$user"
    if [ "$user" != "root" ]; then
        echo "Unlocking password for $user"
        STDERR=$(passwd -u "$user" 2>&1 >/dev/null)
        if [ $? -eq 3 ]; then
          # possibly due to the user not having a password
          # use this convoluted case approach to avoid bashisms (POSIX portable)
          case "$STDERR" in
            *"unlocking the password would result in a passwordless account"* )
              # unlock and delete password to set it back to empty
              passwd -ud "$user"
            ;;
          esac
        fi
    fi
done

# Remove the pam_nologin files
[ -f /etc/nologin ] && rm /etc/nologin
[ -f /run/nologin ] && rm /run/nologin

# Remove our custom lock message service
if [ -f /etc/systemd/system/fleet-lock-message.service ]; then
    systemctl stop fleet-lock-message.service 2>/dev/null || true
    systemctl disable fleet-lock-message.service 2>/dev/null
    rm /etc/systemd/system/fleet-lock-message.service
    systemctl daemon-reload
fi

# Enable systemd-user-sessions, a service that deletes /etc/nologin
if [ -f /usr/lib/systemd/system/systemd-user-sessions.service ]; then
    systemctl unmask systemd-user-sessions
    systemctl daemon-reload
    /usr/lib/systemd/systemd-user-sessions start
fi

# Check if we switched to text mode during lock and restore GUI if needed
if [ -f /etc/fleet.text-mode-lock ]; then
    echo "Restoring graphical mode"
    
    # Restore the original systemd target
    if [ -f /etc/fleet.systemd-target.backup ]; then
        TARGET=$(cat /etc/fleet.systemd-target.backup)
        systemctl set-default "$TARGET" 2>/dev/null
        rm /etc/fleet.systemd-target.backup
    else
        # Default to graphical target if no backup found
        systemctl set-default graphical.target
    fi
    
    # Clean up marker file
    rm /etc/fleet.text-mode-lock
    
    # System needs reboot to properly restore GUI
fi

echo "All non-root users have been unlocked."

# Although rebooting is not strictly necessary for all cases, we've seen some UI issues that
# can be resolved by rebooting. For example, the password prompt is not fully visible in LightDM+Ubuntu24.04
reboot
