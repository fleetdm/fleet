#!/bin/bash
#
# cleanup-quay-sha-tags.sh — One-time script to remove commit-SHA-tagged
# Docker images from Quay.io for the fleetdm/fleet repository.
#
# Usage:
#   ./cleanup-quay-sha-tags.sh [--dry-run] [--delay SECONDS]
#
# Environment variables:
#   QUAY_REGISTRY_PASSWORD   Quay.io bearer token (required)
#
# This script is intended to be run manually by an engineer. It is NOT wired
# into any CI workflow. Requires jq and curl to be installed.

set -euo pipefail

DRY_RUN=false
DELAY=1
REPO="fleetdm/fleet"
SHA_PATTERN="^[0-9a-f]{7,12}$"
PAGE_SIZE=100

while [[ $# -gt 0 ]]; do
  case "$1" in
    --dry-run)  DRY_RUN=true; shift ;;
    --delay)    DELAY="$2"; shift 2 ;;
    *)          echo "Unknown option: $1" >&2; exit 1 ;;
  esac
done

if [[ -z "${QUAY_REGISTRY_PASSWORD:-}" ]]; then
  echo "Error: QUAY_REGISTRY_PASSWORD must be set." >&2
  exit 1
fi

# Collect all tags via pagination
echo "Fetching tags from Quay.io (this may take a while)..."
ALL_TAGS=()
PAGE=1
HAS_MORE=true

while [[ "$HAS_MORE" == "true" ]]; do
  RESPONSE=$(curl -s "https://quay.io/api/v1/repository/${REPO}/tag/?page=${PAGE}&limit=${PAGE_SIZE}" \
    -H "Authorization: Bearer $QUAY_REGISTRY_PASSWORD")

  PAGE_TAGS=$(echo "$RESPONSE" | jq -r '.tags[].name // empty')
  COUNT=0
  while IFS= read -r tag; do
    if [[ -n "$tag" ]]; then
      ALL_TAGS+=("$tag")
      COUNT=$((COUNT + 1))
    fi
  done <<< "$PAGE_TAGS"

  HAS_MORE=$(echo "$RESPONSE" | jq -r '.has_additional')
  PAGE=$((PAGE + 1))
  echo "  Fetched ${#ALL_TAGS[@]} tags so far..."

  if [[ $COUNT -eq 0 ]]; then
    break
  fi
done

echo "Total tags found: ${#ALL_TAGS[@]}"

# Filter to SHA-only tags
SHA_TAGS=()
for tag in "${ALL_TAGS[@]}"; do
  if [[ "$tag" =~ $SHA_PATTERN ]]; then
    SHA_TAGS+=("$tag")
  fi
done

echo "Tags matching commit SHA pattern: ${#SHA_TAGS[@]}"

if [[ ${#SHA_TAGS[@]} -eq 0 ]]; then
  echo "No SHA tags to delete. Done."
  exit 0
fi

if [[ "$DRY_RUN" == "true" ]]; then
  echo ""
  echo "=== Tags that will be KEPT (do not match SHA pattern) ==="
  KEPT=0
  for tag in "${ALL_TAGS[@]}"; do
    if ! [[ "$tag" =~ $SHA_PATTERN ]]; then
      echo "  $tag"
      KEPT=$((KEPT + 1))
    fi
  done
  echo "=== ${KEPT} tags kept ==="
  echo ""
  echo "=== DRY RUN — ${#SHA_TAGS[@]} tags would be deleted ==="
  exit 0
fi

# Delete tags
DELETED=0
FAILED=0
TOTAL=${#SHA_TAGS[@]}

echo ""
echo "Deleting ${TOTAL} SHA tags from Quay.io..."

for tag in "${SHA_TAGS[@]}"; do
  HTTP_STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE \
    "https://quay.io/api/v1/repository/${REPO}/tag/${tag}" \
    -H "Authorization: Bearer $QUAY_REGISTRY_PASSWORD")

  if [[ "$HTTP_STATUS" == "200" || "$HTTP_STATUS" == "204" ]]; then
    DELETED=$((DELETED + 1))
  elif [[ "$HTTP_STATUS" == "404" ]]; then
    : # already gone, not a failure
  else
    echo "  FAILED (HTTP $HTTP_STATUS): $tag"
    FAILED=$((FAILED + 1))
  fi

  # Progress
  PROCESSED=$((DELETED + FAILED))
  if (( PROCESSED % 50 == 0 )); then
    echo "  Progress: ${PROCESSED}/${TOTAL} processed (${DELETED} deleted, ${FAILED} failed)"
  fi

  sleep "$DELAY"
done

echo ""
echo "Done. ${DELETED} deleted, ${FAILED} failed (out of ${TOTAL})"
