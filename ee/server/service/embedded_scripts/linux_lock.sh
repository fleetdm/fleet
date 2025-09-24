#!/bin/sh

# Disable automatic login for common display managers
disable_autologin() {
    # GDM (GNOME Display Manager)
    if [ -f /etc/gdm3/custom.conf ]; then
        sed -i '/^AutomaticLoginEnable/s/^/#/' /etc/gdm3/custom.conf
        sed -i '/^AutomaticLogin/s/^/#/' /etc/gdm3/custom.conf
    fi

    # LightDM
    if [ -f /etc/lightdm/lightdm.conf ]; then
        sed -i '/^autologin-user=/s/^/#/' /etc/lightdm/lightdm.conf
    fi

    # Add similar cases for other display managers if needed
}

# Disable automatic login first
disable_autologin

# Loop through all users in /etc/passwd
awk -F':' '{ if ($3 >= 1000 && $3 < 60000) print $1 }' /etc/passwd | while read user
do
    if [ "$user" != "root" ]; then
        echo "Logging out $user"
        pkill -KILL -u "$user" # Kill user processes. This will log out logged-in users.
        passwd -l "$user"      # Lock the user account
    fi
done

# Logout any non-passwd users
logged_in=$(users | tr ' ' '\n' | sort  | uniq)
for user in $logged_in; do
    [ "$user" = "root" ] && continue
    echo "Logging out $user"
    pkill -KILL -u "$user"
done

# Now that users are logged out, check if we have Ubuntu with GDM
# Ubuntu with GDM has issues with /etc/nologin causing black/unresponsive screens
# This issue does not occur on Fedora or other distros
# Although we've only seen it on Ubuntu 24.04, we are doing it for all Ubuntu versions to be safe.
NEEDS_REBOOT=0
if [ -f /etc/os-release ] && grep -qi ubuntu /etc/os-release; then
    if systemctl is-active --quiet gdm3 || systemctl is-active --quiet gdm; then
        echo "Ubuntu with GDM detected - will reboot to text mode to avoid black screen issue"
        
        # Store current target for restoration during unlock
        systemctl get-default > /etc/fleet.systemd-target.backup 2>/dev/null
        
        # Switch default to text mode for next boot
        systemctl set-default multi-user.target
        
        # Mark that we switched to text mode
        touch /etc/fleet.text-mode-lock
        
        # Set flag to reboot after creating lock files
        NEEDS_REBOOT=1
    fi
fi

# Create the pam_nologin files with our custom message
# Both /etc/nologin and /run/nologin to ensure the message is shown
echo "Locked by administrator" > /etc/nologin
echo "Locked by administrator" > /run/nologin

# Set a GDM banner for any system with GDM to make the lock visible
# GDM GUI doesn't always display PAM nologin messages, so we need this banner
if systemctl is-active --quiet gdm || systemctl is-active --quiet gdm3; then
    echo "Setting GDM banner for lock notification"
    mkdir -p /etc/dconf/db/gdm.d
    # Use 99- prefix to ensure high priority (overrides lower numbers)
    cat > /etc/dconf/db/gdm.d/99-fleet-lock-banner << 'EOF'
[org/gnome/login-screen]
banner-message-enable=true
banner-message-text='System is locked by administrator'
EOF
    dconf update 2>/dev/null || true
fi

# Disable systemd-user-sessions, a service that deletes /etc/nologin
# Although we re-create /etc/nologin in another systemd service,
# we are doing this to prevent a race condition (security hole) where /etc/nologin is deleted
# before we create it.
if [ -f /usr/lib/systemd/system/systemd-user-sessions.service ]; then
    systemctl mask systemd-user-sessions
    systemctl daemon-reload
fi

# Create a systemd service to recreate nologin files on boot
# This ensures our message persists even after systemd-user-sessions runs
cat > /etc/systemd/system/fleet-lock-message.service << 'EOF'
[Unit]
Description=Fleet Lock Message
After=systemd-user-sessions.service
Before=getty.target gdm.service gdm3.service lightdm.service display-manager.service

[Service]
Type=oneshot
ExecStart=/bin/sh -c 'echo "Locked by administrator" > /run/nologin; echo "Locked by administrator" > /etc/nologin'
RemainAfterExit=yes

[Install]
WantedBy=multi-user.target graphical.target
EOF

systemctl enable fleet-lock-message.service 2>/dev/null

echo "All non-root users have been logged out and their accounts locked."

# Reboot if we switched Ubuntu+GDM to text mode
if [ "$NEEDS_REBOOT" = "1" ]; then
    echo "System needs to reboot to complete lock process..."

    # Schedule reboot instead of immediate to ensure script reports success to Fleet
    # The script already uses systemctl extensively, so systemd-run should be available
    # This gives us precise 10-second delay for the script to report success
    echo "Scheduling system reboot in 10 seconds to complete lock process..."
    systemd-run --on-active=10s --timer-property=AccuracySec=100ms /sbin/reboot
fi
exit 0
