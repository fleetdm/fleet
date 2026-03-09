#!/usr/bin/env bash

# -e: Immediately exit if any command has a non-zero exit status.
# -x: Print all executed commands to the terminal.
# -u: Exit if an undefined variable is used.
# -o pipefail: Exit if any command in a pipeline fails.
set -exuo pipefail

FLEET_GITOPS_DIR="${FLEET_GITOPS_DIR:-.}"
FLEET_GLOBAL_FILE="${FLEET_GLOBAL_FILE:-$FLEET_GITOPS_DIR/default.yml}"
FLEETCTL="${FLEETCTL:-fleetctl}"
FLEET_DRY_RUN_ONLY="${FLEET_DRY_RUN_ONLY:-false}"
FLEET_DELETE_OTHER_FLEETS="${FLEET_DELETE_OTHER_FLEETS:-true}"

# Check for existence of the global config file (conventionally called default.yml)
# in case the script is used on repositories with only per-fleet .yml files.
if [ -f "$FLEET_GLOBAL_FILE" ]; then
	# Validate that global file contains org_settings
	grep -Exq "^org_settings:.*" "$FLEET_GLOBAL_FILE"
else
	FLEET_DELETE_OTHER_FLEETS=false
fi

# If you are using secrets to manage SSO metadata for Fleet SSO login or MDM SSO login, uncomment the below:

# FLEET_SSO_METADATA=$( sed '2,$s/^/      /' <<<  "${FLEET_MDM_SSO_METADATA}")
# FLEET_MDM_SSO_METADATA=$( sed '2,$s/^/        /' <<<  "${FLEET_MDM_SSO_METADATA}")

# Copy/pasting raw SSO metadata into GitHub secrets will result in malformed yaml. 
# Adds spaces to all but the first line of metadata keeps the  multiline string in bounds.

if compgen -G "$FLEET_GITOPS_DIR"/fleets/*.yml > /dev/null; then
  # Validate that every fleet has a unique name.
  # This is a limited check that assumes all per-fleet .yml files contain the phrase: `name: <fleet_name>`
  ! perl -nle 'print $1 if /^name:\s*(.+)$/' "$FLEET_GITOPS_DIR"/fleets/*.yml | sort | uniq -d | grep . -cq
fi

args=()
if [ -f "$FLEET_GLOBAL_FILE" ]; then
	args=(-f "$FLEET_GLOBAL_FILE")
fi

for per_fleet_config_file in "$FLEET_GITOPS_DIR"/fleets/*.yml; do
  if [ -f "$per_fleet_config_file" ]; then
    args+=(-f "$per_fleet_config_file")
  fi
done
if [ "$FLEET_DELETE_OTHER_FLEETS" = true ]; then
  args+=(--delete-other-fleets)
fi

# Dry run
$FLEETCTL gitops "${args[@]}" --dry-run
if [ "$FLEET_DRY_RUN_ONLY" = true ]; then
  exit 0
fi

# Real run
$FLEETCTL gitops "${args[@]}"