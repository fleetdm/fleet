FROM archlinux:latest

# This is NOT a real Omarchy install: Omarchy publishes no container image, and
# Omarchy is a desktop environment on top of Arch that osquery only tells apart
# by /etc/os-release. Omarchy ships that file via its omarchy-settings package,
# so writing it onto an Arch base is enough to exercise the "omarchy" platform
# string end to end. Anything that depends on Omarchy's actual packages or
# desktop session needs a real host instead.
#
# Values match https://github.com/omacom-io/omarchy-pkgs
# pkgbuilds/omarchy-settings/PKGBUILD
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
