#!/bin/bash
# build-in-container.sh: run build-fleetd-proxy-msi.sh inside Fleet's fleetdm/wix image
# (Debian + Wine + WiX 3), using `fleetctl package --native-tooling`.
#
# Everything the build installs lives in a throwaway container that's removed afterwards
# (--rm). The host only needs Docker or Podman on an amd64 machine: no packages, no Docker
# socket passed in, no --privileged. The finished MSI lands in ./fleetd-proxy-out/.
#
# Same knobs as build-fleetd-proxy-msi.sh (FLEET_VER, ORBIT_CHANNEL, OSQUERYD_CHANNEL,
# EXTRA_PACKAGE_ARGS, OUT_NAME, SIGN_PFX, SIGN_PFX_PASS_FILE, TIMESTAMP_URL), plus:
#   OUT_DIR  host directory for the MSI     (default ./fleetd-proxy-out)
#   ENGINE   docker | podman                (default: docker if present, else podman)
#   IMAGE    build image                    (default docker.io/fleetdm/wix:latest, the image fleetctl itself uses)
# Proxy env vars (HTTPS_PROXY, HTTP_PROXY, NO_PROXY) are passed through if set.
set -euo pipefail
HERE="$(cd "$(dirname "$0")" && pwd)"
# Podman fails if XDG_RUNTIME_DIR points at a directory that doesn't exist (e.g. root via su/sudo).
if [ -n "${XDG_RUNTIME_DIR:-}" ] && [ ! -d "$XDG_RUNTIME_DIR" ]; then unset XDG_RUNTIME_DIR; fi
ENGINE="${ENGINE:-$(command -v docker >/dev/null && echo docker || echo podman)}"
IMAGE="${IMAGE:-docker.io/fleetdm/wix:latest}"   # fully qualified: Podman on RHEL rejects short names without a TTY
OUT_DIR="${OUT_DIR:-$PWD/fleetd-proxy-out}"; mkdir -p "$OUT_DIR"; OUT_DIR="$(cd "$OUT_DIR" && pwd)"

args=(run --rm --platform linux/amd64
  -v "$HERE/build-fleetd-proxy-msi.sh:/build.sh:ro"
  -v "$OUT_DIR:/out"
  -e NATIVE_TOOLING=1 -e WORK=/tmp/build -e OUT_DIR=/out
  -e HOST_UID="$(id -u)" -e HOST_GID="$(id -g)")
for v in FLEET_VER ORBIT_CHANNEL OSQUERYD_CHANNEL ARCH EXTRA_PACKAGE_ARGS OUT_NAME TIMESTAMP_URL \
         HTTPS_PROXY HTTP_PROXY NO_PROXY https_proxy http_proxy no_proxy; do
  [ -z "${!v:-}" ] || args+=(-e "$v=${!v}")
done
if [ -n "${SIGN_PFX:-}" ]; then
  args+=(-v "$(cd "$(dirname "$SIGN_PFX")" && pwd)/$(basename "$SIGN_PFX"):/secrets/cert.pfx:ro" -e SIGN_PFX=/secrets/cert.pfx)
fi
if [ -n "${SIGN_PFX_PASS_FILE:-}" ]; then
  args+=(-v "$(cd "$(dirname "$SIGN_PFX_PASS_FILE")" && pwd)/$(basename "$SIGN_PFX_PASS_FILE"):/secrets/pass.txt:ro" -e SIGN_PFX_PASS_FILE=/secrets/pass.txt)
fi
exec "$ENGINE" "${args[@]}" "$IMAGE" bash /build.sh
