#!/usr/bin/env bash
#
# Verify that every Apple App Store app declared in the GitOps YAML is
# available on the VPP account, before fleetctl gitops runs.
#
# Fleet rejects an App Store app that the VPP account cannot supply during a
# real apply. When a fleet is being added to the VPP list, fleetctl first
# re-applies every VPP fleet without App Store apps and then adds them back,
# so an apply that fails on one unavailable app leaves the fleets it has not
# reached yet without their App Store apps. Running this check first turns
# that into a failed pull request instead.
#
# Uses the same environment as the other steps: FLEET_URL and FLEET_API_TOKEN.
# Play Store apps are identified by bundle identifier, not by a numeric App
# Store ID, and are not checked here.
set -euo pipefail

FLEET_GITOPS_DIR="${FLEET_GITOPS_DIR:-.}"
FLEET_URL="$(printf '%s' "$FLEET_URL" | sed 's:/*$::')"

api() {
  curl --fail --silent --show-error --header "Authorization: Bearer $FLEET_API_TOKEN" "$FLEET_URL/api/latest/fleet$1"
}

# Numeric app_store_id values declared under fleets/ (or teams/ in older repos).
declared="$(grep -rhoE 'app_store_id:[[:space:]]*"?[0-9]+"?' "$FLEET_GITOPS_DIR"/fleets "$FLEET_GITOPS_DIR"/teams 2>/dev/null | grep -oE '[0-9]+' | sort -u || true)"
if [[ -z "$declared" ]]; then
  echo "No Apple App Store apps declared, nothing to verify."
  exit 0
fi

# Fleets, called teams on older servers.
if fleets_json="$(api "/fleets?per_page=500" 2>/dev/null)"; then
  fleet_ids="$(jq -r '.fleets[]?.id' <<< "$fleets_json")"
  scope="fleet_id"
else
  fleets_json="$(api "/teams?per_page=500")"
  fleet_ids="$(jq -r '.teams[]?.id' <<< "$fleets_json")"
  scope="team_id"
fi

# Everything the VPP account can still add to a fleet, plus everything already
# added to one. Fleets without a VPP token answer with an error and are skipped.
available=""
for id in $fleet_ids; do
  if apps="$(api "/software/app_store_apps?$scope=$id" 2>/dev/null)"; then
    available+="$(jq -r '.app_store_apps[]?.app_store_id' <<< "$apps")"$'\n'
  fi
  page=0
  while titles="$(api "/software/titles?$scope=$id&available_for_install=true&per_page=500&page=$page" 2>/dev/null)"; do
    available+="$(jq -r '.software_titles[]? | select(.app_store_app != null) | .app_store_app.app_store_id' <<< "$titles")"$'\n'
    if [[ "$(jq -r '.meta.has_next_results // false' <<< "$titles")" != "true" ]]; then
      break
    fi
    page=$((page + 1))
  done
done
available="$(printf '%s' "$available" | sed '/^$/d' | sort -u)"

missing="$(comm -23 <(printf '%s\n' "$declared") <(printf '%s\n' "$available") | sed '/^$/d')"
if [[ -z "$missing" ]]; then
  echo "All $(wc -l <<< "$declared" | tr -d ' ') declared App Store apps are available on the VPP account."
  exit 0
fi

for id in $missing; do
  # Best effort readable name from Apple's public catalog; the verdict does not depend on it.
  name="$(curl --silent --max-time 8 "https://itunes.apple.com/lookup?id=$id" 2>/dev/null | jq -r '.results[0].trackName // empty' 2>/dev/null || true)"
  files="$(grep -rlE "app_store_id:[[:space:]]*\"?$id\"?" "$FLEET_GITOPS_DIR"/fleets "$FLEET_GITOPS_DIR"/teams 2>/dev/null | tr '\n' ' ' || true)"
  echo "::error::${name:-App Store app} ($id, https://apps.apple.com/app/id$id) is not available on the VPP account. Declared in: ${files:-unknown}. Buy licenses for exactly this app in Apple Business Manager and wait until it appears under Software > Add software > App Store in Fleet, or remove the entry."
done
echo "Error: $(wc -l <<< "$missing" | tr -d ' ') declared App Store app(s) are not available on the VPP account. Stopping before fleetctl gitops runs."
exit 1
