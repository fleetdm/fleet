#!/bin/bash

# Script assumes one user is using the desktop environment (no multi-session).
# It was tested on Fedora 38, 39, Debian 12, and OpenSUSE Leap/Tumbleweed.

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

username=$(ps -o user= -p $fleet_desktop_pid | xargs)
uid=$(ps -o uid= -p $fleet_desktop_pid | xargs)

# Detect the Linux distribution
if [ -f /etc/os-release ]; then
	. /etc/os-release
	distro_id="$ID"
	distro_name="$NAME"
else
	distro_id="unknown"
	distro_name="unknown"
fi

# Determine extension name and installation method based on distribution
case "$distro_id" in
	"opensuse-leap"|"opensuse-tumbleweed"|"opensuse")
		extension_name="ubuntu-appindicators@ubuntu.com"
		install_method="zypper"
		;;
	"fedora"|"debian"|"ubuntu")
		extension_name="appindicatorsupport@rgcjonas.gmail.com"
		install_method="gnome-extensions"
		;;
	*)
		# Default to the original extension for unknown distributions
		extension_name="appindicatorsupport@rgcjonas.gmail.com"
		install_method="gnome-extensions"
		;;
esac

# Check if any AppIndicator extension is already installed
fedora_extension_path="/home/$username/.local/share/gnome-shell/extensions/appindicatorsupport@rgcjonas.gmail.com"
opensuse_extension_path="/home/$username/.local/share/gnome-shell/extensions/ubuntu-appindicators@ubuntu.com"

extension_installed=false
if [ -d "$fedora_extension_path" ] || [ -d "$opensuse_extension_path" ]; then
	extension_installed=true
fi

# If no extension is installed, install the appropriate one
if [ "$extension_installed" = false ]; then
	# Show notification to user before the prompt.
	sudo -i -u $username -H DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/$uid/bus \
		gdbus call --session \
		--dest org.freedesktop.Notifications \
		--object-path /org/freedesktop/Notifications \
		--method org.freedesktop.Notifications.Notify \
		"Fleet Desktop" 0 \"\" "Fleet Desktop" "Install a GNOME extension to enable Fleet Desktop. This lets you see what your organization is doing on your computer." "[]" '{"urgency": <2>}' 0

	# Give some time to user to see notification.
	sleep 10

	if [ "$install_method" = "zypper" ]; then
		# For OpenSUSE, install via package manager
		zypper install -y gnome-shell-extension-appindicator
		
		# Wait for package installation to complete
		sleep 5
	else
		# For Fedora/Debian, use GNOME Extensions
		sudo -i -u $username -H DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/$uid/bus \
			gdbus call --session \
			--dest org.gnome.Shell.Extensions \
			--object-path /org/gnome/Shell/Extensions \
			--method org.gnome.Shell.Extensions.InstallRemoteExtension \
			"$extension_name"

		# Wait until the extension is accepted by the user ("gdbus call" command above is asynchronous).
		while [ ! -d "/home/$username/.local/share/gnome-shell/extensions/$extension_name" ]; do
			sleep 1
		done

		# Sleep to give some time for files to be downloaded.
		sleep 15
	fi
fi

# Enable the appropriate extension(s)
if [ -d "$fedora_extension_path" ]; then
	sudo -i -u $username -H DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/$uid/bus \
		gnome-extensions enable "appindicatorsupport@rgcjonas.gmail.com"
fi

if [ -d "$opensuse_extension_path" ]; then
	sudo -i -u $username -H DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/$uid/bus \
		gnome-extensions enable "ubuntu-appindicators@ubuntu.com"
fi
