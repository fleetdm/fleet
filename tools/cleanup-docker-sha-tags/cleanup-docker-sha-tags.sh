#!/bin/bash
#
# cleanup-docker-sha-tags.sh — One-time script to remove commit-SHA-tagged Docker images
# from Docker Hub and Quay.io for the fleetdm/fleet repository.
#
# Usage:
#   ./cleanup-docker-sha-tags.sh [--dry-run] [--delay SECONDS] [--skip-quay]
#
# Environment variables:
#   DOCKERHUB_USERNAME       Docker Hub username (required)
#   DOCKERHUB_ACCESS_TOKEN   Docker Hub access token (required)
#   QUAY_REGISTRY_PASSWORD   Quay.io bearer token (required unless --skip-quay)
#
# This script is intended to be run manually by an engineer.

set -euo pipefail

DRY_RUN=false
DELAY=1
SKIP_QUAY=false
REPO="fleetdm/fleet"
SHA_PATTERN="^[0-9a-f]{7,12}$"
PAGE_SIZE=100

while [[ $# -gt 0 ]]; do
  case "$1" in
    --dry-run)  DRY_RUN=true; shift ;;
    --delay)    DELAY="$2"; shift 2 ;;
    --skip-quay) SKIP_QUAY=true; shift ;;
    *)          echo "Unknown option: $1" >&2; exit 1 ;;
  esac
done

# Validate required env vars
if [[ -z "${DOCKERHUB_USERNAME:-}" || -z "${DOCKERHUB_ACCESS_TOKEN:-}" ]]; then
  echo "Error: DOCKERHUB_USERNAME and DOCKERHUB_ACCESS_TOKEN must be set." >&2
  exit 1
fi
if [[ "$SKIP_QUAY" == "false" && -z "${QUAY_REGISTRY_PASSWORD:-}" ]]; then
  echo "Error: QUAY_REGISTRY_PASSWORD must be set (or use --skip-quay)." >&2
  exit 1
fi

# Authenticate to Docker Hub
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
  echo "=== DRY RUN — the following tags would be deleted ==="
  for tag in "${SHA_TAGS[@]}"; do
    echo "  $tag"
  done
  echo "=== End dry run (${#SHA_TAGS[@]} tags) ==="
  exit 0
fi

# Delete tags
DELETED=0
FAILED=0
TOTAL=${#SHA_TAGS[@]}

echo ""
echo "Deleting ${TOTAL} SHA tags..."

for tag in "${SHA_TAGS[@]}"; do
  # Delete from Docker Hub
  ATTEMPT=0
  MAX_RETRIES=3
  while true; do
    HTTP_STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE \
      "https://hub.docker.com/v2/repositories/${REPO}/tags/${tag}/" \
      -H "Authorization: Bearer $DOCKERHUB_TOKEN")

    if [[ "$HTTP_STATUS" == "204" || "$HTTP_STATUS" == "404" ]]; then
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

  # Delete from Quay.io
  if [[ "$SKIP_QUAY" == "false" && ("$HTTP_STATUS" == "204" || "$HTTP_STATUS" == "404") ]]; then
    curl -s -o /dev/null -X DELETE \
      "https://quay.io/api/v1/repository/${REPO}/tag/${tag}" \
      -H "Authorization: Bearer $QUAY_REGISTRY_PASSWORD" || true
  fi

  if [[ "$HTTP_STATUS" == "204" || "$HTTP_STATUS" == "404" ]]; then
    DELETED=$((DELETED + 1))
  fi

  # Progress
  if (( DELETED % 50 == 0 )); then
    echo "  Progress: ${DELETED}/${TOTAL} deleted (${FAILED} failed)"
  fi

  sleep "$DELAY"
done

echo ""
echo "Done. Deleted: ${DELETED}/${TOTAL}, Failed: ${FAILED}"
