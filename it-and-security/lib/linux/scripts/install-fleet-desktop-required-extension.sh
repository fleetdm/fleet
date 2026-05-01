#!/bin/bash

# Script assumes one user is using the desktop environment (no multi-session).
# It was tested on Fedora 38, 39, Debian 12, openSUSE Leap 15/Tumbleweed, and
# openSUSE Leap 16.

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
	distro_version_id="$VERSION_ID"
else
	distro_id="unknown"
	distro_name="unknown"
	distro_version_id=""
fi

# Detect openSUSE Leap 16+. On Leap 16 the GNOME extension install path differs
# from other distros: extensions.gnome.org doesn't list a compatible build of
# this extension, and `sudo -i` exec's are denied, so we install the extension
# directly from the upstream tarball and use a `sudo -u ... env` invocation
# instead of `sudo -i`. Other distros keep the previously QA'd behavior.
is_opensuse_leap_16_plus=false
if [ "$distro_id" = "opensuse-leap" ]; then
	major_version="${distro_version_id%%.*}"
	if [ -n "$major_version" ] && [ "$major_version" -ge 16 ] 2>/dev/null; then
		is_opensuse_leap_16_plus=true
	fi
fi

# run_as_user runs a command as the GUI user with the session DBus address set.
# On openSUSE Leap 16+ we drop sudo's -i flag because, in that environment,
# `sudo -i` wraps the command in `bash --login -c '<escaped>'` and the exec of
# /bin/bash is denied (the same root cause that breaks fleet-desktop launch).
# Other distros keep the previous "-i + DBUS=val command" form so we don't
# change behavior on already-QA'd platforms.
run_as_user() {
	if [ "$is_opensuse_leap_16_plus" = true ]; then
		sudo -u "$username" -H \
			env DBUS_SESSION_BUS_ADDRESS="unix:path=/run/user/$uid/bus" "$@"
	else
		sudo -i -u "$username" -H \
			DBUS_SESSION_BUS_ADDRESS="unix:path=/run/user/$uid/bus" "$@"
	fi
}

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

# Check if the AppIndicator extension is already installed. We look for
# metadata.json (not just the directory) so that a half-baked stub left over
# from an earlier failed install — e.g. one where gnome-shell created the
# directory but bailed before writing metadata.json — is treated as
# "not installed" and gets re-installed properly. Without this, downstream
# `gnome-extensions enable` would fail with "Extension does not exist".
extension_path="/home/$username/.local/share/gnome-shell/extensions/$extension_name"
extension_metadata="$extension_path/metadata.json"

extension_installed=false
if [ -f "$extension_metadata" ]; then
	extension_installed=true
fi

# If no extension is installed, install the appropriate one
if [ "$extension_installed" = false ]; then
	# Show notification to user before the prompt.
	run_as_user gdbus call --session \
		--dest org.freedesktop.Notifications \
		--object-path /org/freedesktop/Notifications \
		--method org.freedesktop.Notifications.Notify \
		"Fleet Desktop" 0 \"\" "Fleet Desktop" "Install a GNOME extension to enable Fleet Desktop. This lets you see what your organization is doing on your computer." "[]" '{"urgency": <2>}' 0

	# Give some time to user to see notification.
	sleep 10

	if [ "$is_opensuse_leap_16_plus" = true ]; then
		# On openSUSE Leap 16+, skip the dbus InstallRemoteExtension call and go
		# straight to the upstream tarball: extensions.gnome.org doesn't list a
		# compatible build of this extension for Leap 16's GNOME, so the dbus
		# call returns "Remote peer disconnected" immediately and at best leaves
		# a half-baked stub directory behind. Skipping it avoids ~90s of
		# dead-end waiting and a useless install prompt the user can't complete.
		#
		# Clear any stub directory left from a previous run so our copy below
		# isn't laying files on top of a half-baked install.
		if [ -d "$extension_path" ] && [ ! -f "$extension_metadata" ]; then
			sudo rm -rf "$extension_path"
		fi

		extensions_dir="/home/$username/.local/share/gnome-shell/extensions"
		tarball_url="https://github.com/ubuntu/gnome-shell-extension-appindicator/archive/refs/heads/master.tar.gz"
		tmp_dir=$(mktemp -d /tmp/fleet-appindicator.XXXXXX)
		tarball="$tmp_dir/extension.tar.gz"

		# Download tarball — curl preferred, wget as fallback. Bail out cleanly
		# if neither can fetch it (e.g. no network) so we don't leave a
		# half-installed extension.
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
			# tar isn't part of a minimal openSUSE Leap 16 install, so pull it
			# in via zypper if it's missing. We're already running as root here
			# (this whole branch runs from the root-detached script invocation
			# at the top of this file).
			if ! command -v tar >/dev/null 2>&1; then
				zypper --non-interactive install --no-recommends tar \
					>/dev/null 2>&1 || true
			fi

			if command -v tar >/dev/null 2>&1; then
				sudo -u $username -H mkdir -p "$extensions_dir"
				# Extract into a staging dir under our root-owned tmp_dir, then
				# copy the contents into the user's UUID-named extension path
				# and hand ownership to the user. We do the copy as root rather
				# than `sudo -u $username` because mktemp's tmp_dir is mode 700
				# owned by root, which the user can't traverse.
				staging="$tmp_dir/staging"
				mkdir -p "$staging"
				if tar -xzf "$tarball" -C "$staging" --strip-components=1; then
					mkdir -p "$extension_path"
					cp -r "$staging/." "$extension_path/"
					chown -R "$username":"$username" "$extension_path"
					if [ -d "$extension_path/schemas" ] && command -v glib-compile-schemas >/dev/null 2>&1; then
						sudo -u $username -H glib-compile-schemas "$extension_path/schemas/"
					fi
				fi
			fi
		fi

		rm -rf "$tmp_dir"
	else
		# Other distributions: prompt the user via gnome-shell's
		# InstallRemoteExtension and wait indefinitely for them to accept.
		# This is the previously QA'd behavior on Fedora / Debian / Ubuntu /
		# openSUSE Tumbleweed.
		run_as_user gdbus call --session \
			--dest org.gnome.Shell.Extensions \
			--object-path /org/gnome/Shell/Extensions \
			--method org.gnome.Shell.Extensions.InstallRemoteExtension \
			"$extension_name"

		while [ ! -f "$extension_metadata" ]; do
			sleep 1
		done

		# Give gnome-shell a moment to finish writing extension files after
		# the directory shows up. Not needed on the Leap 16 path above, where
		# we already wrote everything synchronously via curl + tar.
		sleep 15
	fi
fi

# Enable the extension.
if [ -f "$extension_metadata" ]; then
	if [ "$is_opensuse_leap_16_plus" = true ]; then
		# gnome-shell on Leap 16 (Wayland) doesn't rescan
		# ~/.local/share/gnome-shell/extensions while running, so
		# `gnome-extensions enable` (which talks to the live gnome-shell)
		# reports the extension as missing. Pre-seed the dconf list directly;
		# gnome-shell will pick it up and enable it on the user's next login.
		current_extensions=$(run_as_user gsettings get org.gnome.shell enabled-extensions)
		case "$current_extensions" in
			*"'$extension_name'"*)
				;;
			"@as []"|"[]")
				run_as_user gsettings set org.gnome.shell enabled-extensions \
					"['$extension_name']"
				;;
			*)
				run_as_user gsettings set org.gnome.shell enabled-extensions \
					"${current_extensions%]}, '$extension_name']"
				;;
		esac
	else
		run_as_user gnome-extensions enable "$extension_name"
	fi
fi
