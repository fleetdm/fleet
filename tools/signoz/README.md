# Running SigNoz locally with Fleet

[SigNoz](https://signoz.io/) is an open-source observability platform that provides traces, metrics, and logs in a single UI. This guide explains how to run SigNoz locally for Fleet development with optimized settings for reduced latency.

## Prerequisites

- Docker and Docker Compose
- A locally-built Fleet server (see [Testing and local development](../../docs/Contributing/getting-started/testing-and-local-development.md))

## Setup

1. Clone the SigNoz repository at a specific release:

```bash
git clone --branch v0.110.1 --depth 1 https://github.com/SigNoz/signoz.git
cd signoz/deploy
```

2. Modify the SigNoz UI port to avoid conflict with Fleet (which uses port 8080):

In `docker/docker-compose.yaml`, change the signoz service port mapping:

```yaml
services:
  signoz:
    ports:
      - "8085:8080"  # Changed from 8080:8080 to avoid conflict with Fleet
```

3. (Optional) For reduced latency during development, modify `docker/otel-collector-config.yaml`:

```yaml
processors:
  batch:
    send_batch_size: 10000
    send_batch_max_size: 11000
    timeout: 200ms  # reduced from 10s for dev
  # ...
  signozspanmetrics/delta:
    # ...
    metrics_flush_interval: 5s  # reduced from 60s for dev
```

4. Start SigNoz:

```bash
cd docker
docker compose up -d
```

Give it a minute for all services to initialize. The SigNoz UI will be available at http://localhost:8085.

## Configuring Fleet

Start the Fleet server with OpenTelemetry tracing and logging enabled:

```bash
export FLEET_LOGGING_TRACING_ENABLED=true
export FLEET_LOGGING_OTEL_LOGS_ENABLED=true
export OTEL_SERVICE_NAME=fleet
export OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317

./build/fleet serve
```

> **Note:** All log levels (including debug) are always sent to SigNoz regardless of the `--logging_debug` flag. That flag only controls stderr output.

### Multiple environments

When you point more than one Fleet deployment (for example `production`, `staging`, and `dev-local`) at the same SigNoz instance, keep the service name the same and tell them apart by environment instead.

Leave `OTEL_SERVICE_NAME` at its default of `fleet` on every deployment so they stay grouped under one service, then set the environment through `OTEL_RESOURCE_ATTRIBUTES`. Set both of these attributes to the same value:

- `deployment.environment.name` is the current [OpenTelemetry semantic convention](https://opentelemetry.io/docs/specs/semconv/resource/deployment-environment/).
- `deployment.environment` is the older, now-deprecated attribute that SigNoz still keys off (the dashboard `environment` selector reads this one).

```bash
export OTEL_SERVICE_NAME=fleet
export OTEL_RESOURCE_ATTRIBUTES=deployment.environment.name=dev-local,deployment.environment=dev-local
```

Use a distinct value on each deployment and keep both attributes in sync so the dashboard `environment` selector resolves to a single deployment.

### Low-latency configuration (optional)

For faster feedback during development, you can reduce the batch processing delays on the Fleet side:

```bash
# Batch span processor delay (default 5000ms)
export OTEL_BSP_SCHEDULE_DELAY=1000

# Log batch processor settings
export OTEL_BLRP_EXPORT_TIMEOUT=1000
export OTEL_BLRP_SCHEDULE_DELAY=500
export OTEL_BLRP_MAX_EXPORT_BATCH_SIZE=1

./build/fleet serve
```

## Using SigNoz

After starting Fleet with the above configuration, you should start seeing traces, logs, and metrics in SigNoz UI at http://localhost:8085.

## Pre-canned dashboards

JSON exports of Fleet-specific SigNoz dashboards live alongside this README. Import them from the SigNoz UI via **Dashboards → New dashboard → Import JSON** (top-right dropdown).

- `database_custom_dashboard.json` — MySQL query metrics (RPS, latency, slow queries) derived from `db.sql.*` instrumentation.
- `host_cache_dashboard.json` — Redis-backed host lookup cache (`LoadHostByNodeKey` / `LoadHostByOrbitNodeKey`). Shows hit rate over time, lookups/sec by result, errors/sec by op, and invalidations/sec by write-path reason. Requires `FLEET_REDIS_HOST_CACHE_ENABLED=true` (default on).
- `http_errors_dashboard.json` — Fleet HTTP errors (the "Errors" signal of the RED method / Google SRE Golden Signals). Shows 4XX client errors and 5XX server errors from `fleet.http.client_errors` / `fleet.http.server_errors`, broken down by `error.type`.

Each dashboard includes an `environment` selector (a dynamic dashboard variable on the `deployment.environment` resource attribute) so you can scope panels to a single deployment or view all. It defaults to ALL. Fleet always emits `deployment.environment` (default value `default`, overridable via `OTEL_RESOURCE_ATTRIBUTES`), so the selector populates on any instance Fleet reports to.
