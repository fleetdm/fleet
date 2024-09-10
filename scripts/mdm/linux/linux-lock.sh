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

# Disable automatic login
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

# Create the pam_nologin file
echo "Locked by administrator" > /etc/nologin

# Disable systemd-user-sessions, a service that deletes /etc/nologin
if [ -f /usr/lib/systemd/system/systemd-user-sessions.service ]; then
    systemctl mask systemd-user-sessions
    systemctl daemon-reload
fi

echo "All non-root users have been logged out and their accounts locked."
