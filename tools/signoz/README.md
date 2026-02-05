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

./build/fleet serve --dev
```

> **Note:** All log levels (including debug) are always sent to SigNoz regardless of the `--logging_debug` flag. That flag only controls stderr output.

### Low-latency configuration (optional)

For faster feedback during development, you can reduce the batch processing delays on the Fleet side:

```bash
# Tracing and logging
export FLEET_LOGGING_TRACING_ENABLED=true
export FLEET_LOGGING_OTEL_LOGS_ENABLED=true
export OTEL_SERVICE_NAME=fleet
export OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317

# Batch span processor delay (default 5000ms)
export OTEL_BSP_SCHEDULE_DELAY=1000

# Log batch processor settings
export OTEL_BLRP_EXPORT_TIMEOUT=1000
export OTEL_BLRP_SCHEDULE_DELAY=500
export OTEL_BLRP_MAX_EXPORT_BATCH_SIZE=1

./build/fleet serve --dev
```

## Using SigNoz

After starting Fleet with the above configuration, you should start seeing traces, logs, and metrics in SigNoz UI at http://localhost:8085.

