FROM archlinux:latest

# Omarchy ships its own /etc/os-release via the omarchy-settings package, and
# there is no official Omarchy container image. Reproduce that file on an Arch
# base so the "omarchy" platform string is exercised end to end. Values match
# https://github.com/omacom-io/omarchy-pkgs pkgbuilds/omarchy-settings/PKGBUILD
RUN printf '%s\n' \
	'NAME="Omarchy"' \
	'PRETTY_NAME="Omarchy"' \
	'ID=omarchy' \
	'ID_LIKE=arch' \
	'BUILD_ID="4.0.0"' \
	'VERSION_ID="4.0.0"' \
	'HOME_URL="https://omarchy.org/"' \
	> /etc/os-release

COPY fleet-osquery_amd64.pkg.tar.zst /
COPY run-fleetd.sh /
RUN chmod +x /run-fleetd.sh

RUN pacman -Sy --noconfirm --needed ca-certificates

RUN pacman -U --noconfirm /fleet-osquery_amd64.pkg.tar.zst

ENTRYPOINT ["/run-fleetd.sh"]
