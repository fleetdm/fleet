#!/bin/bash
# build-linux-in-container.sh: run build-fleetd-proxy-linux.sh in a throwaway Fedora container
# (it needs dpkg-deb, rpm and rpmrebuild, which Fedora packages). Nothing is installed on the
# host; no Docker socket or --privileged. Packages land in ./fleetd-proxy-linux-out/.
#
# Same knobs as build-fleetd-proxy-linux.sh (FLEET_VER, FLEET_URL, ORBIT_CHANNEL, OSQUERYD_CHANNEL,
# ARCH, TYPES), plus OUT_DIR, ENGINE (docker|podman) and IMAGE (default docker.io/library/fedora:latest).
# Proxy env vars (HTTPS_PROXY, HTTP_PROXY, NO_PROXY) are passed through if set.
set -euo pipefail
HERE="$(cd "$(dirname "$0")" && pwd)"
# Podman fails if XDG_RUNTIME_DIR points at a directory that doesn't exist (e.g. root via su/sudo).
if [ -n "${XDG_RUNTIME_DIR:-}" ] && [ ! -d "$XDG_RUNTIME_DIR" ]; then unset XDG_RUNTIME_DIR; fi
ENGINE="${ENGINE:-$(command -v docker >/dev/null && echo docker || echo podman)}"
IMAGE="${IMAGE:-docker.io/library/fedora:latest}"   # fully qualified: Podman on RHEL rejects short names without a TTY
OUT_DIR="${OUT_DIR:-$PWD/fleetd-proxy-linux-out}"; mkdir -p "$OUT_DIR"; OUT_DIR="$(cd "$OUT_DIR" && pwd)"
args=(run --rm --platform linux/amd64
  -v "$HERE/build-fleetd-proxy-linux.sh:/build.sh:ro" -v "$OUT_DIR:/out"
  -e OUT_DIR=/out -e WORK=/tmp/build -e HOST_UID="$(id -u)" -e HOST_GID="$(id -g)")
for v in FLEET_VER FLEET_URL ORBIT_CHANNEL OSQUERYD_CHANNEL ARCH TYPES \
         HTTPS_PROXY HTTP_PROXY NO_PROXY https_proxy http_proxy no_proxy; do
  [ -z "${!v:-}" ] || args+=(-e "$v=${!v}")
done
exec "$ENGINE" "${args[@]}" "$IMAGE" bash /build.sh
