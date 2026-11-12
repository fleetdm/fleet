#!/usr/bin/env bash
set -euo pipefail

NAMESPACE="${NAMESPACE:-signoz}"
ENDPOINT="${OTLP_ENDPOINT:-otlp.signoz.dogfood.fleetdm.com:443}"
TOKEN="${OTEL_BEARER_TOKEN:-}"
DURATION="${DURATION:-15s}"
IMAGE="${TELEMETRYGEN_IMAGE:-ghcr.io/open-telemetry/opentelemetry-collector-contrib/telemetrygen:latest}"

if [[ -z "${TOKEN}" ]]; then
  echo "OTEL_BEARER_TOKEN is required." >&2
  exit 1
fi

kubectl -n "${NAMESPACE}" run signoz-telemetrygen \
  --rm -i --restart=Never \
  --image "${IMAGE}" \
  --command -- /telemetrygen traces \
  --otlp-endpoint "${ENDPOINT}" \
  --otlp-header "authorization=\"Bearer ${TOKEN}\"" \
  --duration "${DURATION}"
