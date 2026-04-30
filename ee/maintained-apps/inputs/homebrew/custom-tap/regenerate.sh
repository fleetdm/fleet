#!/usr/bin/env bash
#
# regenerate.sh
#
# Rebuild every api/<token>.json from its Casks/<token>.rb source using
# Homebrew. Run this whenever you edit or add a cask under Casks/, then
# commit both the updated .rb and the regenerated .json alongside the
# Fleet-maintained-app output manifest (produced separately by
# `go run cmd/maintained-apps/main.go --slug=<slug>`).
#
# Why a throwaway local tap? `brew info --cask --json=v2` can only parse
# casks reachable through a tap; it won't parse a loose .rb file path.
# So the script drops the Casks/*.rb into a private tap, runs brew info,
# extracts the single cask object, and tears the tap down on exit.
#
# Fields that vary between developer machines (install state, tap
# identity, build timestamps) are stripped so the committed JSON is
# deterministic.
#
# Requirements: macOS with Homebrew and jq installed.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

command -v brew >/dev/null 2>&1 || {
  echo "error: brew is required; install from https://brew.sh" >&2
  exit 1
}
command -v jq >/dev/null 2>&1 || {
  echo "error: jq is required; install with 'brew install jq'" >&2
  exit 1
}

if [ ! -d Casks ]; then
  echo "error: expected a Casks/ directory next to this script" >&2
  exit 1
fi

shopt -s nullglob
rb_files=(Casks/*.rb)
if [ ${#rb_files[@]} -eq 0 ]; then
  echo "error: no .rb files found under Casks/" >&2
  exit 1
fi

TAP_USER="fleetdm"
TAP_NAME="fma-custom-tap"
TAP_DIR="$(brew --repository)/Library/Taps/${TAP_USER}/homebrew-${TAP_NAME}"

cleanup() {
  rm -rf "$TAP_DIR"
}
trap cleanup EXIT

rm -rf "$TAP_DIR"
mkdir -p "$TAP_DIR/Casks"
cp Casks/*.rb "$TAP_DIR/Casks/"

(
  cd "$TAP_DIR"
  git init -q
  git add -A
  git -c user.email=fma@local -c user.name=fma commit -q -m "local regenerate"
) >/dev/null

mkdir -p api

# Fields stripped from brew's output because they vary by developer
# machine and are not read by Fleet's FMA ingester:
#   installed, installed_time, outdated — install state on this host
#   tap, tap_git_head, full_token       — tap identity (throwaway tap)
#   generated_date                      — build timestamp
STRIP='del(.installed, .installed_time, .outdated, .tap, .tap_git_head, .full_token, .generated_date)'

for rb in "${rb_files[@]}"; do
  token="$(basename "$rb" .rb)"
  out="api/${token}.json"
  echo "Regenerating ${out} from ${rb}..."
  brew info --cask --json=v2 "${TAP_USER}/${TAP_NAME}/${token}" \
    | jq ".casks[0] | ${STRIP}" \
    > "$out"
done

echo
echo "Done. If api/*.json changed, also regenerate the FMA output manifests:"
echo "  go run cmd/maintained-apps/main.go --slug=<app>/darwin"
echo "and commit everything together."
