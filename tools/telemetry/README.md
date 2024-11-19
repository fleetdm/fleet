### Telemetry tools

Running the services specified in the `docker-compose.yml` file will give you access to:

- The Jaeger UI with both the `/monitor` (latency, errors, req/sec) and `/search` (traces) tabs ready to use.
- A Prometheus server used by Jaeger with enhanced monitoring data provided by OpenTelemetry.

To get started:

1. Start the necessary services by running `docker compose up` in this directory.
2. Start the Fleet server with telemetry enabled and configured with this:

```
OTEL_SERVICE_NAME="fleet" \
OTEL_EXPORTER_OTLP_ENDPOINT="http://localhost:4317" \
./build/fleet serve \
  --logging_tracing_enabled=true \
  --logging_tracing_type=opentelemetry \
  --dev --logging_debug
```

Afterwards, you can navigate to http://localhost:16686/ to access the Jaeger UI.
