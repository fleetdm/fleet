#!/bin/sh
#
# version-from-branch.sh — derive a semver-ish version string from a branch name.
#
# Usage:
#   version-from-branch.sh [BRANCH]
#
# If BRANCH is omitted the current git branch is used.
#
# The script tries two patterns in order:
#   1. rc-minor-fleet-vX.Y.Z or rc-patch-fleet-vX.Y.Z → X.Y.Z-rc.YYMMDDhhmm
#   2. X.Y.Z-anything  or vX.Y.Z-anything             → X.Y.Z+YYMMDDhhmm
#
# When a pattern matches the version is printed to stdout and the script
# exits 0.  When neither pattern matches nothing is printed and the script
# exits 1 so callers can apply their own fallback.

set -e

BRANCH="${1:-$(git rev-parse --abbrev-ref HEAD)}"
TIMESTAMP="$(date -u +'%y%m%d%H%M')"

# 1. rc-minor-fleet-vX.Y.Z or rc-patch-fleet-vX.Y.Z → X.Y.Z-rc.YYMMDDhhmm
VERSION=$(echo "$BRANCH" | sed -E -n "s/^rc-(minor|patch)-fleet-v([0-9]+\.[0-9]+\.[0-9]+).*/\2-rc.${TIMESTAMP}/p")

if [ -z "$VERSION" ]; then
  # 2. X.Y.Z-anything or vX.Y.Z-anything → X.Y.Z+YYMMDDhhmm
  VERSION=$(echo "$BRANCH" | sed -E -n "s/^v?([0-9]+\.[0-9]+\.[0-9]+)[-+].*/\1+${TIMESTAMP}/p")
fi

if [ -n "$VERSION" ]; then
  echo "$VERSION"
  exit 0
fi

exit 1
