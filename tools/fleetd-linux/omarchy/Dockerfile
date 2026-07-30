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

# Arch does not support partial upgrades: refreshing the package databases with
# -Sy and then installing a package that pulls newer dependencies can leave the
# system inconsistent. Upgrade with -Syu and install fleetd in the same layer so
# both steps succeed or fail together.
#
# --disable-sandbox is required because this image is built for linux/amd64, so
# on an arm64 host it runs emulated and pacman's download sandbox cannot install
# its seccomp filter ("error restricting syscalls via seccomp: 22").
RUN pacman --disable-sandbox -Syu --noconfirm --needed ca-certificates \
	&& pacman -U --noconfirm /fleet-osquery_amd64.pkg.tar.zst

ENTRYPOINT ["/run-fleetd.sh"]
