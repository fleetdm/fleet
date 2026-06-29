#!/bin/bash
#
# cleanup-dockerhub-sha-tags.sh — One-time script to remove commit-SHA-tagged
# Docker images from Docker Hub for the fleetdm/fleet repository.
#
# Usage:
#   ./cleanup-dockerhub-sha-tags.sh [--dry-run] [--delay SECONDS]
#
# Environment variables:
#   DOCKERHUB_USERNAME       Docker Hub username (required)
#   DOCKERHUB_ACCESS_TOKEN   Docker Hub access token (required)
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

if [[ -z "${DOCKERHUB_USERNAME:-}" || -z "${DOCKERHUB_ACCESS_TOKEN:-}" ]]; then
  echo "Error: DOCKERHUB_USERNAME and DOCKERHUB_ACCESS_TOKEN must be set." >&2
  exit 1
fi

# Authenticate
echo "Authenticating to Docker Hub..."
DOCKERHUB_TOKEN=$(curl -s -X POST "https://hub.docker.com/v2/users/login/" \
  -H "Content-Type: application/json" \
  -d "{\"username\": \"$DOCKERHUB_USERNAME\", \"password\": \"$DOCKERHUB_ACCESS_TOKEN\"}" \
  | jq -r .token)

if [[ -z "$DOCKERHUB_TOKEN" || "$DOCKERHUB_TOKEN" == "null" ]]; then
  echo "Error: Failed to authenticate to Docker Hub." >&2
  exit 1
fi
echo "Authenticated."

# Collect all tags via pagination
echo "Fetching tags from Docker Hub (this may take a while)..."
ALL_TAGS=()
NEXT_URL="https://hub.docker.com/v2/repositories/${REPO}/tags/?page_size=${PAGE_SIZE}"

while [[ -n "$NEXT_URL" && "$NEXT_URL" != "null" ]]; do
  RESPONSE=$(curl -s "$NEXT_URL" -H "Authorization: Bearer $DOCKERHUB_TOKEN")
  PAGE_TAGS=$(echo "$RESPONSE" | jq -r '.results[].name')
  while IFS= read -r tag; do
    [[ -n "$tag" ]] && ALL_TAGS+=("$tag")
  done <<< "$PAGE_TAGS"
  NEXT_URL=$(echo "$RESPONSE" | jq -r '.next')
  echo "  Fetched ${#ALL_TAGS[@]} tags so far..."
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
echo "Deleting ${TOTAL} SHA tags from Docker Hub..."

for tag in "${SHA_TAGS[@]}"; do
  ATTEMPT=0
  MAX_RETRIES=3
  while true; do
    HTTP_STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE \
      "https://hub.docker.com/v2/repositories/${REPO}/tags/${tag}/" \
      -H "Authorization: Bearer $DOCKERHUB_TOKEN")

    if [[ "$HTTP_STATUS" == "204" ]]; then
      DELETED=$((DELETED + 1))
      break
    elif [[ "$HTTP_STATUS" == "404" ]]; then
      break
    elif [[ "$HTTP_STATUS" == "429" ]]; then
      ATTEMPT=$((ATTEMPT + 1))
      if [[ $ATTEMPT -ge $MAX_RETRIES ]]; then
        echo "  FAILED (rate limited, max retries): $tag"
        FAILED=$((FAILED + 1))
        break
      fi
      BACKOFF=$((DELAY * ATTEMPT * 5))
      echo "  Rate limited, waiting ${BACKOFF}s before retry..."
      sleep "$BACKOFF"
    else
      echo "  FAILED (HTTP $HTTP_STATUS): $tag"
      FAILED=$((FAILED + 1))
      break
    fi
  done

  # Progress
  PROCESSED=$((DELETED + FAILED))
  if (( PROCESSED % 50 == 0 )); then
    echo "  Progress: ${PROCESSED}/${TOTAL} processed (${DELETED} deleted, ${FAILED} failed)"
  fi

  sleep "$DELAY"
done

echo ""
echo "Done. ${DELETED} deleted, ${FAILED} failed (out of ${TOTAL})"
