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
		extension_name="appindicatorsupport@rgcjonas.gmail.com"
		install_method="gnome-extensions"
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

# Check if the AppIndicator extension is already installed
extension_path="/home/$username/.local/share/gnome-shell/extensions/$extension_name"

extension_installed=false
if [ -d "$extension_path" ]; then
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

	# Use GNOME Extensions for all distributions (including OpenSUSE)
	sudo -i -u $username -H DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/$uid/bus \
		gdbus call --session \
		--dest org.gnome.Shell.Extensions \
		--object-path /org/gnome/Shell/Extensions \
		--method org.gnome.Shell.Extensions.InstallRemoteExtension \
		"$extension_name"

	# Wait until the extension is accepted by the user ("gdbus call" command above is asynchronous).
	# Cap the wait so we don't hang forever if InstallRemoteExtension can't deliver — for
	# example on openSUSE Leap 16, where extensions.gnome.org doesn't list a compatible
	# build of this extension and the install never completes.
	timeout=90
	while [ ! -d "$extension_path" ] && [ "$timeout" -gt 0 ]; do
		sleep 1
		timeout=$((timeout - 1))
	done

	# If the official install path didn't deliver, fall back to fetching the extension
	# tarball from upstream. This is required on openSUSE Leap 16, which neither packages
	# the extension via zypper nor serves a compatible build through extensions.gnome.org.
	# We use curl + tar (both present on a base Leap/Fedora/Debian install) rather than
	# git, which isn't installed by default on Leap 16.
	if [ ! -d "$extension_path" ]; then
		extensions_dir="/home/$username/.local/share/gnome-shell/extensions"
		tarball_url="https://github.com/ubuntu/gnome-shell-extension-appindicator/archive/refs/heads/master.tar.gz"
		tmp_dir=$(mktemp -d /tmp/fleet-appindicator.XXXXXX)
		tarball="$tmp_dir/extension.tar.gz"

		# Download tarball — curl preferred, wget as fallback. Bail out cleanly if neither
		# can fetch it (e.g. no network) so we don't leave a half-installed extension.
		fetched=false
		if command -v curl >/dev/null 2>&1; then
			if curl -fsSL --max-time 60 -o "$tarball" "$tarball_url"; then
				fetched=true
			fi
		elif command -v wget >/dev/null 2>&1; then
			if wget -q --timeout=60 -O "$tarball" "$tarball_url"; then
				fetched=true
			fi
		fi

		if [ "$fetched" = true ] && [ -s "$tarball" ]; then
			sudo -u $username -H mkdir -p "$extensions_dir"
			# Extract to a staging dir, then move the inner directory into place under
			# the user's UUID-named extension path. Run as the user so file ownership is right.
			staging="$tmp_dir/staging"
			mkdir -p "$staging"
			if tar -xzf "$tarball" -C "$staging" --strip-components=1; then
				sudo -u $username -H cp -r "$staging" "$extension_path"
				if [ -d "$extension_path/schemas" ] && command -v glib-compile-schemas >/dev/null 2>&1; then
					sudo -u $username -H glib-compile-schemas "$extension_path/schemas/"
				fi
			fi
		fi

		rm -rf "$tmp_dir"
	fi

	# Sleep to give some time for files to be downloaded.
	sleep 15
fi

# Enable the extension
if [ -d "$extension_path" ]; then
	sudo -i -u $username -H DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/$uid/bus \
		gnome-extensions enable "$extension_name"
fi
