#!/bin/bash

# Script assumes one user is using the desktop environment (no multi-session).
# It was tested on Fedora 38, 39 and Debian 12.

set -x

run_uid=$(id -u)

# Start detached script and exit as root (to send result back to Fleet).
if [ $run_uid == 0 ] && [ $# -eq 0 ]; then
	/bin/bash -c "/bin/bash $0 1 >/var/log/orbit/appindicator_script.log 2>/var/log/orbit/appindicator_script.log </dev/null &"
	echo "A detached script to install extension has been started (logs can be found in /var/log/orbit/appindicator_script.log)."
	exit 0
fi

# Wait for user to be logged in to the GUI (by checking fleet-desktop process).
fleet_desktop_pid=$(pgrep fleet-desktop)
while [ -z $fleet_desktop_pid ]; do
	fleet_desktop_pid=$(pgrep fleet-desktop)
	sleep 10
done

extension_name="appindicatorsupport@rgcjonas.gmail.com"
username=$(ps -o user= -p $fleet_desktop_pid | xargs)
uid=$(ps -o uid= -p $fleet_desktop_pid | xargs)

# If the extension is not installed, then prompt the user.
if [ ! -d "/home/$username/.local/share/gnome-shell/extensions/$extension_name" ]; then
	# Show notification to user before the prompt.
	sudo -i -u $username -H DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/$uid/bus \
		gdbus call --session \
		--dest org.freedesktop.Notifications \
		--object-path /org/freedesktop/Notifications \
		--method org.freedesktop.Notifications.Notify \
		"Fleet Desktop" 0 \"\" "Fleet Desktop" "Install a GNOME extension to enable Fleet Desktop. This lets you see what your organization is doing on your computer." "[]" '{"urgency": <2>}' 0

	# Give some time to user to see notification.
	sleep 10

	# Prompt user for installation of extension.
	sudo -i -u $username -H DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/$uid/bus \
		gdbus call --session \
		--dest org.gnome.Shell.Extensions \
		--object-path /org/gnome/Shell/Extensions \
		--method org.gnome.Shell.Extensions.InstallRemoteExtension \
		"$extension_name"

	# Wait until the extension is accepted by the user ("gbus call" command above is asynchronous).
	while [ ! -d "/home/$username/.local/share/gnome-shell/extensions/$extension_name" ]; do
	  sleep 1
	done

	# Sleep to give some time for files to be downloaded.
	sleep 15
fi

# Enable the extension in case it was disabled in the past.
sudo -i -u $username -H DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/$uid/bus \
	gnome-extensions enable "$extension_name"
