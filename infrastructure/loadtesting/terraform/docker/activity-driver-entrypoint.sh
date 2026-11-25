#!/bin/sh
# Starts the load-test activity driver in ECS. The task passes FLEET_URL and FLEET_DRIVER_TOKEN, and optionally:
#   DRIVER_PROFILE        the private profile YAML (a secret), written to a 0600 file and passed as -profile
#   DRIVER_GITOPS_BUNDLE  s3://bucket/key.tar.gz of the private GitOps repo, unpacked for gitops.apply, whose
#                         apply command defaults to the bundle's own apply.sh
# Everything else is the driver's own flags, from the task's command.
set -eu
: "${FLEET_URL:?}" "${FLEET_DRIVER_TOKEN:?}"
umask 077

if [ -n "${DRIVER_PROFILE:-}" ]; then
  printf '%s\n' "$DRIVER_PROFILE" > "$HOME/profile.yaml"
  unset DRIVER_PROFILE
  set -- -profile "$HOME/profile.yaml" "$@"
fi

if [ -n "${DRIVER_GITOPS_BUNDLE:-}" ]; then
  mkdir -p "$HOME/gitops"
  aws s3 cp --only-show-errors "$DRIVER_GITOPS_BUNDLE" - | tar -xz -C "$HOME/gitops"
  export DRIVER_GITOPS_REPO="$HOME/gitops"
  export DRIVER_GITOPS_COMMAND="${DRIVER_GITOPS_COMMAND:-sh apply.sh}"
  export DRIVER_GITOPS_BUSY_FILE="$HOME/gitops.busy"
  TOKEN="$FLEET_DRIVER_TOKEN" fleetctl config set --address "$FLEET_URL" >/dev/null
fi

exec driver -fleet_url "$FLEET_URL" -scenarios /scenarios "$@"
